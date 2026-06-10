package main

import (
	"crypto/sha1"
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
		// TODO: Uncomment the code below to pass the first stage
		//
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
		data, err := os.ReadFile(target)
		if err != nil {
			fmt.Println(err)
			return
		}

		torrent, infoRaw, err := decodeTorrentFile(string(data))
		if err != nil {
			fmt.Println(err)
			return
		}

		announce, ok := torrent["announce"].(string)
		if !ok {
			fmt.Println("invalid torrent file: missing announce URL")
			return
		}
		fmt.Println("Tracker URL:", announce)

		info, ok := torrent["info"].(map[string]interface{})
		if !ok {
			fmt.Println("invalid torrent file: missing info dictionary")
			return
		}

		length, ok := info["length"].(int)
		if !ok {
			fmt.Println("invalid torrent file: missing length in info dictionary")
			return
		}
		fmt.Println("Length:", length)

		if infoRaw == "" {
			fmt.Println("invalid torrent file: missing raw info dictionary")
			return
		}
		infoHash := sha1.Sum([]byte(infoRaw))
		fmt.Printf("Info Hash: %x\n", infoHash)

		pieceLength, ok := info["piece length"].(int)
		if !ok {
			fmt.Println("invalid torrent file: missing piece length in info dictionary")
			return
		}
		fmt.Println("Piece Length:", pieceLength)

		pieces, ok := info["pieces"].(string)
		if !ok {
			fmt.Println("invalid torrent file: missing pieces in info dictionary")
			return
		}
		piecesBytes := []byte(pieces)
		if len(piecesBytes)%20 != 0 {
			fmt.Println("invalid torrent file: pieces length is not a multiple of 20")
			return
		}
		fmt.Println("Piece Hashes:")
		for i := 0; i < len(piecesBytes); i += 20 {
			fmt.Printf("%x\n", piecesBytes[i:i+20])
		}

	default:
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}
