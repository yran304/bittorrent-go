package main

import (
	"fmt"
	"strconv"
	"unicode"
)

// Example:
// - 5:hello -> hello
// - 10:hello12345 -> hello12345
func decodeBencode(bencodedString string) (interface{}, error) {
	decoded, consumed, _, err := decodeValue(bencodedString)
	if err != nil {
		return nil, err
	}

	if consumed != len(bencodedString) {
		return nil, fmt.Errorf("invalid bencode: trailing data")
	}

	return decoded, nil
}

func decodeTorrentFile(bencodedString string) (map[string]interface{}, string, error) {
	decoded, consumed, infoRaw, err := decodeValue(bencodedString)
	if err != nil {
		return nil, "", err
	}

	if consumed != len(bencodedString) {
		return nil, "", fmt.Errorf("invalid bencode: trailing data")
	}

	torrent, ok := decoded.(map[string]interface{})
	if !ok {
		return nil, "", fmt.Errorf("invalid torrent file")
	}

	return torrent, infoRaw, nil
}

func decodeValue(bencodedString string) (interface{}, int, string, error) {
	if len(bencodedString) == 0 {
		return nil, 0, "", fmt.Errorf("invalid bencode: empty input")
	}

	if unicode.IsDigit(rune(bencodedString[0])) {
		firstColonIndex := -1

		for i := 0; i < len(bencodedString); i++ {
			if bencodedString[i] == ':' {
				firstColonIndex = i
				break
			}
		}

		if firstColonIndex == -1 {
			return nil, 0, "", fmt.Errorf("invalid bencode string")
		}

		lengthStr := bencodedString[:firstColonIndex]
		length, err := strconv.Atoi(lengthStr)
		if err != nil {
			return nil, 0, "", err
		}

		start := firstColonIndex + 1
		end := start + length
		if end > len(bencodedString) {
			return nil, 0, "", fmt.Errorf("invalid bencode string length")
		}

		return bencodedString[start:end], end, "", nil
	}

	if bencodedString[0] == 'i' {
		eIndex := -1

		for i := 1; i < len(bencodedString); i++ {
			if bencodedString[i] == 'e' {
				eIndex = i
				break
			}
		}

		if eIndex == -1 {
			return nil, 0, "", fmt.Errorf("invalid bencode integer")
		}

		intStr := bencodedString[1:eIndex]
		value, err := strconv.Atoi(intStr)
		if err != nil {
			return nil, 0, "", err
		}

		return value, eIndex + 1, "", nil
	}

	if bencodedString[0] == 'l' {
		var values []interface{}
		currentIndex := 1

		for currentIndex < len(bencodedString) && bencodedString[currentIndex] != 'e' {
			value, consumed, _, err := decodeValue(bencodedString[currentIndex:])
			if err != nil {
				return nil, 0, "", err
			}

			values = append(values, value)
			currentIndex += consumed
		}

		if currentIndex >= len(bencodedString) || bencodedString[currentIndex] != 'e' {
			return nil, 0, "", fmt.Errorf("invalid bencode list")
		}

		return values, currentIndex + 1, "", nil
	}

	if bencodedString[0] == 'd' {
		dict := make(map[string]interface{})
		previousKey := ""
		infoRaw := ""
		currentIndex := 1

		for currentIndex < len(bencodedString) && bencodedString[currentIndex] != 'e' {
			keyValue, keyConsumed, _, err := decodeValue(bencodedString[currentIndex:])
			if err != nil {
				return nil, 0, "", err
			}

			key, ok := keyValue.(string)
			if !ok {
				return nil, 0, "", fmt.Errorf("invalid bencode dictionary key")
			}

			if previousKey != "" && key <= previousKey {
				return nil, 0, "", fmt.Errorf("invalid bencode dictionary: keys not in order")
			}

			currentIndex += keyConsumed
			valueStart := currentIndex

			value, valueConsumed, nestedInfoRaw, err := decodeValue(bencodedString[currentIndex:])
			if err != nil {
				return nil, 0, "", err
			}

			dict[key] = value
			if key == "info" {
				infoRaw = bencodedString[valueStart : valueStart+valueConsumed]
			} else if infoRaw == "" && nestedInfoRaw != "" {
				infoRaw = nestedInfoRaw
			}

			currentIndex += valueConsumed
			previousKey = key
		}

		if currentIndex >= len(bencodedString) || bencodedString[currentIndex] != 'e' {
			return nil, 0, "", fmt.Errorf("invalid bencode dictionary")
		}

		return dict, currentIndex + 1, infoRaw, nil
	}

	return nil, 0, "", fmt.Errorf("bencode type not supported")
}
