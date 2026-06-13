package main

import (
	"encoding/json"
	"fmt"
	"os"
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
	default:
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}
