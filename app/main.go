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

		if infoRaw == "" {
			fmt.Println("invalid torrent file: missing raw info dictionary")
			return
		}

		infoHash := sha1.Sum([]byte(infoRaw))

		fmt.Println("Tracker URL:", announce)
		fmt.Println("Length:", length)
		fmt.Printf("Info Hash: %x\n", infoHash)

	default:
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}
