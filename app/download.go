package main

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"net"
	"os"
)

const (
	maxPeerWorkers = 5
)

type pieceResult struct {
	index int
	data  []byte
	err   error
}

func preparePeerConnection(meta torrentMeta, peerAddr string) (net.Conn, []byte, error) {
	conn, _, err := connectAndHandshake(meta.InfoHash, peerAddr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to peer: %w", err)
	}

	bitfield, err := waitForBitfield(conn)
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("failed to receive bitfield message from peer: %w", err)
	}

	err = sendInterested(conn)
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("failed to send interested message to peer: %w", err)
	}

	err = waitForUnchoke(conn)
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("failed to receive unchoke message from peer: %w", err)
	}

	return conn, bitfield, nil // bitfield tells which piece is available
}

func downloadPieceFromPeer(conn net.Conn, bitfield []byte, meta torrentMeta, pieceIndex int) ([]byte, error) {
	if pieceIndex < 0 || pieceIndex >= len(meta.PieceHashes) {
		return nil, fmt.Errorf("piece index out of bounds")
	}

	if !hasPiece(bitfield, pieceIndex) {
		return nil, fmt.Errorf("peer does not have piece %d", pieceIndex)
	}

	pieceLength, err := getPieceLength(meta, pieceIndex)
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

func downloadFile(meta torrentMeta) ([]byte, error) {
	peerAddrs, err := fetchTrackerPeers(meta)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve peer list: %w", err)
	}
	if len(peerAddrs) == 0 {
		return nil, fmt.Errorf("did not find any peers")
	}

	workerCount := maxPeerWorkers
	if workerCount > len(peerAddrs) {
		workerCount = len(peerAddrs)
	}
	if workerCount > len(meta.PieceHashes) {
		workerCount = len(meta.PieceHashes)
	}

	fileData := make([]byte, meta.Length)
	jobs := make(chan int, len(meta.PieceHashes))
	results := make(chan pieceResult, len(meta.PieceHashes))

	for i := 0; i < workerCount; i++ {
		peerAddr := peerAddrs[i]

		go func(peerAddr string) {
			conn, bitfield, err := preparePeerConnection(meta, peerAddr)
			if err != nil {
				return
			}
			defer conn.Close()

			for pieceIndex := range jobs {
				pieceData, err := downloadPieceFromPeer(conn, bitfield, meta, pieceIndex)
				results <- pieceResult{
					index: pieceIndex,
					data:  pieceData,
					err:   err,
				}
			}
		}(peerAddr)
	}

	for pieceIndex := 0; pieceIndex < len(meta.PieceHashes); pieceIndex++ {
		jobs <- pieceIndex
	}
	close(jobs)

	for i := 0; i < len(meta.PieceHashes); i++ {
		result := <-results
		if result.err != nil {
			return nil, result.err
		}

		begin := result.index * meta.PieceLength
		copy(fileData[begin:], result.data)
	}

	return fileData, nil
}

func downloadFileToPath(torrentPath, outputPath string) error {
	meta, err := readTorrentMeta(torrentPath)
	if err != nil {
		return fmt.Errorf("failed to read torrent file: %w", err)
	}

	fileData, err := downloadFile(meta)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}

	err = os.WriteFile(outputPath, fileData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write file data to path: %w", err)
	}

	return nil
}
