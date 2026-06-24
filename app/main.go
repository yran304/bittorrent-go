package main

import (
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
		if os.Args[2] != "-o" {
			fmt.Println("expected -o flag")
			return
		}

		outputPath := os.Args[3]
		torrentPath := os.Args[4]
		pieceIndex, err := strconv.Atoi(os.Args[5])
		if err != nil {
			fmt.Println("invalid piece index:", err)
			return
		}

		err = downloadPieceToFile(torrentPath, pieceIndex, outputPath)
		if err != nil {
			fmt.Println(err)
			return
		}
	case "download":
		if os.Args[2] != "-o" {
			fmt.Println("expected -o flag")
			return
		}

		outputPath := os.Args[3]
		torrentPath := os.Args[4]
		err := downloadFileToPath(torrentPath, outputPath)
		if err != nil {
			fmt.Println(err)
			return
		}

	default:
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}
