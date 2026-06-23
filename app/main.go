package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	// bencode "github.com/jackpal/bencode-go" // Available if you need it!
)

// Ensures gofmt doesn't remove the "os" encoding/json import (feel free to remove this!)
var _ = json.Marshal

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Fprintln(os.Stderr, "Logs from your program will appear here!")

	command := os.Args[1]

	switch command {
	case "decode":
		bencodedValue := os.Args[2]

		decoded, err := decodeBencode(bencodedValue)
		if err != nil {
			fmt.Println(err)
			return
		}

		jsonOutput, _ := json.Marshal(decoded)
		fmt.Println(string(jsonOutput))
	case "info":
		target := os.Args[2]
		meta, err := readTorrentMeta(target)
		if err != nil {
			fmt.Println(err)
			return
		}

		printTorrentInfo(meta)
	case "peers":
		target := os.Args[2]
		meta, err := readTorrentMeta(target)
		if err != nil {
			fmt.Println(err)
			return
		}

		peerAddrs, err := fetchTrackerPeers(meta)
		if err != nil {
			fmt.Println(err)
			return
		}

		for _, addr := range peerAddrs {
			fmt.Println(addr)
		}
	case "handshake":
		target := os.Args[2]
		meta, err := readTorrentMeta(target)
		if err != nil {
			fmt.Println(err)
			return
		}

		peerAddr := os.Args[3]
		remotePeerID, err := performHandshake(meta.InfoHash, peerAddr)
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Printf("Peer ID: %x\n", remotePeerID)
	case "download_piece":
		target := os.Args[4]
		meta, err := readTorrentMeta(target)
		if err != nil {
			fmt.Println(err)
			return
		}

		peerAddrs, err := fetchTrackerPeers(meta)
		if err != nil {
			fmt.Println(err)
			return
		}
		var peerAddr string
		if len(peerAddrs) == 0 {
			fmt.Println("did not find any peers")
			return
		}
		peerAddr = peerAddrs[0]

		conn, _, err := connectAndHandshake(meta.InfoHash, peerAddr)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer conn.Close()

		pieceIndex, err := strconv.Atoi(os.Args[5])
		if err != nil {
			fmt.Println("invalid piece index:", err)
			return
		}

		pieceLength, err := getPieceLength(meta, pieceIndex) // the last piece could be shorter than meta.PieceLength
		if err != nil {
			fmt.Println(err)
			return
		}

		_, err = waitForBitfield(conn)
		if err != nil {
			fmt.Println(err)
			return
		}

		err = sendInterested(conn)
		if err != nil {
			fmt.Println(err)
			return
		}

		err = waitForUnchoke(conn)
		if err != nil {
			fmt.Println(err)
			return
		}

		err = requestBlocks(conn, pieceIndex, pieceLength)
		if err != nil {
			fmt.Println(err)
			return
		}

		pieceData, err := receiveBlocks(conn, pieceIndex, pieceLength)
		if err != nil {
			fmt.Println(err)
			return
		}

		receivedPieceHash := sha1.Sum(pieceData)
		expectedHash := meta.PieceHashes[pieceIndex]
		if !bytes.Equal(receivedPieceHash[:], expectedHash) {
			fmt.Println("piece hash mismatch")
			return
		}

		outputPath := os.Args[3]
		if os.Args[2] != "-o" {
			fmt.Println("expected -o flag")
			return
		}
		err = os.WriteFile(outputPath, pieceData, 0644)
		if err != nil {
			fmt.Println("failed to write piece to file:", err)
			return
		}

	default:
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}
