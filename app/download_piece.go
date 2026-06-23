package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"net"
	"os"
)

const (
	blockSize          = 16 * 1024
	pipelineWindowSize = 5
)

func downloadPieceToFile(torrentPath string, pieceIndex int, outputPath string) error {
	meta, err := readTorrentMeta(torrentPath)
	if err != nil {
		return fmt.Errorf("failed to read torrent file: %w", err)
	}

	pieceData, err := downloadPiece(meta, pieceIndex)
	if err != nil {
		return fmt.Errorf("failed to download piece: %w", err)
	}
	err = os.WriteFile(outputPath, pieceData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write piece data to file: %w", err)
	}

	return nil
}

func downloadPiece(meta torrentMeta, pieceIndex int) ([]byte, error) {
	if pieceIndex < 0 || pieceIndex >= len(meta.PieceHashes) {
		return nil, fmt.Errorf("piece index out of bounds")
	}

	conn, err := preparePeerForDownload(meta)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	pieceLength, err := getPieceLength(meta, pieceIndex) // the last piece could be shorter than meta.PieceLength
	if err != nil {
		return nil, err
	}
	pieceData, err := downloadPieceBlocks(conn, pieceIndex, pieceLength)
	if err != nil {
		return nil, err
	}

	receivedPieceHash := sha1.Sum(pieceData)
	expectedHash := meta.PieceHashes[pieceIndex]
	if !bytes.Equal(receivedPieceHash[:], expectedHash) {
		return nil, fmt.Errorf("piece hash mismatch")
	}

	return pieceData, nil
}

func preparePeerForDownload(meta torrentMeta) (net.Conn, error) {
	peerAddrs, err := fetchTrackerPeers(meta)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch peer list: %w", err)
	}
	var peerAddr string
	if len(peerAddrs) == 0 {
		return nil, fmt.Errorf("did not find any peers")
	}
	peerAddr = peerAddrs[0]

	conn, _, err := connectAndHandshake(meta.InfoHash, peerAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to peer: %w", err)
	}

	_, err = waitForBitfield(conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to receive bitfield message from peer: %w", err)
	}

	err = sendInterested(conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to send interested message to peer: %w", err)
	}

	err = waitForUnchoke(conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to receive unchoke message from peer: %w", err)
	}

	return conn, nil
}

func blockLengthForBegin(pieceLength, begin int) int {
	remaining := pieceLength - begin
	if remaining < blockSize {
		return remaining
	}

	return blockSize
}

func downloadPieceBlocks(conn net.Conn, pieceIndex, pieceLength int) ([]byte, error) {
	pieceData := make([]byte, pieceLength)
	totalBlocks := (pieceLength + blockSize - 1) / blockSize
	receivedBlocks := 0
	pendingRequests := 0
	nextBegin := 0
	received := make([]bool, totalBlocks)

	for receivedBlocks < totalBlocks {
		for pendingRequests < pipelineWindowSize && nextBegin < pieceLength {
			blockLength := blockLengthForBegin(pieceLength, nextBegin)
			if err := requestBlock(conn, pieceIndex, nextBegin, blockLength); err != nil {
				return nil, err
			}

			pendingRequests++
			nextBegin += blockLength
		}

		msg, err := readPeerMessage(conn)
		if err != nil {
			return nil, fmt.Errorf("failed to read piece message: %w", err)
		}

		if msg.ID != 7 {
			return nil, fmt.Errorf("expected piece message (id 7), got %d", msg.ID)
		}

		if len(msg.Payload) < 8 {
			return nil, fmt.Errorf("invalid piece payload: too short")
		}

		receivedIndex := int(binary.BigEndian.Uint32(msg.Payload[0:4]))
		begin := int(binary.BigEndian.Uint32(msg.Payload[4:8]))
		block := msg.Payload[8:]
		pendingRequests--

		if receivedIndex != pieceIndex {
			return nil, fmt.Errorf("received piece index %d, expected %d", receivedIndex, pieceIndex)
		}

		if begin < 0 || begin >= pieceLength {
			return nil, fmt.Errorf("invalid block begin offset %d", begin)
		}

		if begin+len(block) > pieceLength {
			return nil, fmt.Errorf("block exceeds piece bounds")
		}

		blockIndex := begin / blockSize
		if blockIndex >= totalBlocks {
			return nil, fmt.Errorf("invalid block index %d", blockIndex)
		}

		expectedBlockLength := blockLengthForBegin(pieceLength, begin)
		if len(block) != expectedBlockLength {
			return nil, fmt.Errorf("received block length %d, expected %d", len(block), expectedBlockLength)
		}

		copy(pieceData[begin:], block)

		if !received[blockIndex] {
			received[blockIndex] = true
			receivedBlocks++
		}
	}

	return pieceData, nil
}
