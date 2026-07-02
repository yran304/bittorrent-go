package main

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"net"
	"os"
)

const (
	maxPeerWorkers  = 5
	maxPieceRetries = 3
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

	workerLimit := maxPeerWorkers
	if workerLimit > len(peerAddrs) {
		workerLimit = len(peerAddrs)
	}
	if workerLimit > len(meta.PieceHashes) {
		workerLimit = len(meta.PieceHashes)
	}

	fileData := make([]byte, meta.Length)
	jobs := make(chan int, len(meta.PieceHashes))
	results := make(chan pieceResult, len(meta.PieceHashes))
	workerDone := make(chan error, workerLimit)
	nextPeerIndex := 0
	activeWorkers := 0

	startNextWorker := func() bool {
		if nextPeerIndex >= len(peerAddrs) {
			return false
		}

		peerAddr := peerAddrs[nextPeerIndex]
		nextPeerIndex++
		activeWorkers++
		go func(peerAddr string) {
			var workerErr error
			defer func() {
				workerDone <- workerErr
			}()

			conn, bitfield, err := preparePeerConnection(meta, peerAddr)
			if err != nil {
				workerErr = err
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
				if err != nil {
					workerErr = err
					return
				}
			}
		}(peerAddr)

		return true
	}

	for activeWorkers < workerLimit && startNextWorker() {
	}

	for pieceIndex := 0; pieceIndex < len(meta.PieceHashes); pieceIndex++ {
		jobs <- pieceIndex
	}

	completedPieces := 0
	pieceRetries := make([]int, len(meta.PieceHashes))
	pieceCompleted := make([]bool, len(meta.PieceHashes))
	var lastErr error
	defer close(jobs)

	for completedPieces < len(meta.PieceHashes) {
		select {
		case result := <-results:
			if pieceCompleted[result.index] {
				continue
			}

			if result.err != nil {
				lastErr = result.err
				pieceRetries[result.index]++
				if pieceRetries[result.index] > maxPieceRetries {
					return nil, fmt.Errorf("piece %d failed after %d retries: %w", result.index, maxPieceRetries, result.err)
				}

				jobs <- result.index
				continue
			}

			begin := result.index * meta.PieceLength
			copy(fileData[begin:], result.data)
			pieceCompleted[result.index] = true
			completedPieces++

		case err := <-workerDone:
			activeWorkers--
			if err != nil {
				lastErr = err
			}
			for activeWorkers < workerLimit && completedPieces < len(meta.PieceHashes) && startNextWorker() {
			}
			if activeWorkers == 0 && completedPieces < len(meta.PieceHashes) {
				if lastErr != nil {
					return nil, fmt.Errorf("all peer workers stopped before completing the download: %w", lastErr)
				}
				return nil, fmt.Errorf("all peer workers stopped before completing the download")
			}
		}
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
