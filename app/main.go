package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"unicode"
	// bencode "github.com/jackpal/bencode-go" // Available if you need it!
)

// Ensures gofmt doesn't remove the "os" encoding/json import (feel free to remove this!)
var _ = json.Marshal

// Example:
// - 5:hello -> hello
// - 10:hello12345 -> hello12345
func decodeBencode(bencodedString string) (interface{}, error) {
	decoded, consumed, err := decodeValue(bencodedString)
	if err != nil {
		return nil, err
	}

	if consumed != len(bencodedString) {
		return nil, fmt.Errorf("invalid bencode: trailing data")
	}

	return decoded, nil
}

func decodeValue(bencodedString string) (interface{}, int, error) {
	if len(bencodedString) == 0 {
		return nil, 0, fmt.Errorf("invalid bencode: empty input")
	}

	if unicode.IsDigit(rune(bencodedString[0])) { // to decode a string
		var firstColonIndex = -1

		for i := 0; i < len(bencodedString); i++ {
			if bencodedString[i] == ':' {
				firstColonIndex = i
				break
			}
		}

		if firstColonIndex == -1 {
			return nil, 0, fmt.Errorf("invalid bencode string")
		}

		lengthStr := bencodedString[:firstColonIndex]
		length, err := strconv.Atoi(lengthStr)
		if err != nil {
			return nil, 0, err
		}

		start := firstColonIndex + 1
		end := start + length
		if end > len(bencodedString) {
			return nil, 0, fmt.Errorf("invalid bencode string length")
		}

		return bencodedString[start:end], end, nil
	} else if bencodedString[0] == 'i' { // to decode an integer
		var eIndex = -1

		for i := 1; i < len(bencodedString); i++ {
			if bencodedString[i] == 'e' {
				eIndex = i
				break
			}
		}

		if eIndex == -1 {
			return nil, 0, fmt.Errorf("invalid bencode integer")
		}

		intStr := bencodedString[1:eIndex]
		value, err := strconv.Atoi(intStr)
		if err != nil {
			return nil, 0, err
		}

		return value, eIndex + 1, nil
	} else if bencodedString[0] == 'l' { // to decode a list
		var values []interface{}
		currentIndex := 1

		for currentIndex < len(bencodedString) && bencodedString[currentIndex] != 'e' {
			value, consumed, err := decodeValue(bencodedString[currentIndex:])
			if err != nil {
				return nil, 0, err
			}

			values = append(values, value)
			currentIndex += consumed
		}

		if currentIndex >= len(bencodedString) || bencodedString[currentIndex] != 'e' {
			return nil, 0, fmt.Errorf("invalid bencode list")
		}

		return values, currentIndex + 1, nil
	} else if bencodedString[0] == 'd' {
		dict := make(map[string]interface{})
		previousKey := ""
		currentIndex := 1

		for currentIndex < len(bencodedString) && bencodedString[currentIndex] != 'e' {
			keyValue, keyConsumed, err := decodeValue(bencodedString[currentIndex:])
			if err != nil {
				return nil, 0, err
			}

			key, ok := keyValue.(string)
			if !ok {
				return nil, 0, fmt.Errorf("invalid bencode dictionary key")
			}

			if previousKey != "" && key <= previousKey {
				return nil, 0, fmt.Errorf("invalid bencode dictionary: keys not in order")
			}

			currentIndex += keyConsumed

			value, valueConsumed, err := decodeValue(bencodedString[currentIndex:])
			if err != nil {
				return nil, 0, err
			}

			dict[key] = value
			currentIndex += valueConsumed
			previousKey = key
		}

		if currentIndex >= len(bencodedString) || bencodedString[currentIndex] != 'e' {
			return nil, 0, fmt.Errorf("invalid bencode dictionary")
		}

		return dict, currentIndex + 1, nil
	}

	return nil, 0, fmt.Errorf("bencode type not supported")
}

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Fprintln(os.Stderr, "Logs from your program will appear here!")

	command := os.Args[1]

	if command == "decode" {
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
	} else {
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}
