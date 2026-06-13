package main

import (
	"crypto/sha1"
	"fmt"
	"os"
)

type torrentMeta struct {
	TrackerURL  string
	Length      int
	InfoHash    [20]byte
	PieceLength int
	PieceHashes [][]byte
}

func readTorrentMeta(path string) (torrentMeta, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return torrentMeta{}, err
	}

	torrent, infoRaw, err := decodeTorrentFile(string(data))
	if err != nil {
		return torrentMeta{}, err
	}

	announce, ok := torrent["announce"].(string)
	if !ok {
		return torrentMeta{}, fmt.Errorf("invalid torrent file: missing announce URL")
	}

	info, ok := torrent["info"].(map[string]interface{})
	if !ok {
		return torrentMeta{}, fmt.Errorf("invalid torrent file: missing info dictionary")
	}

	length, ok := info["length"].(int)
	if !ok {
		return torrentMeta{}, fmt.Errorf("invalid torrent file: missing length in info dictionary")
	}

	if infoRaw == "" {
		return torrentMeta{}, fmt.Errorf("invalid torrent file: missing raw info dictionary")
	}

	pieceLength, ok := info["piece length"].(int)
	if !ok {
		return torrentMeta{}, fmt.Errorf("invalid torrent file: missing piece length in info dictionary")
	}

	pieces, ok := info["pieces"].(string)
	if !ok {
		return torrentMeta{}, fmt.Errorf("invalid torrent file: missing pieces in info dictionary")
	}

	pieceHashes, err := splitPieceHashes([]byte(pieces))
	if err != nil {
		return torrentMeta{}, err
	}

	return torrentMeta{
		TrackerURL:  announce,
		Length:      length,
		InfoHash:    sha1.Sum([]byte(infoRaw)),
		PieceLength: pieceLength,
		PieceHashes: pieceHashes,
	}, nil
}

func splitPieceHashes(pieces []byte) ([][]byte, error) {
	if len(pieces)%20 != 0 {
		return nil, fmt.Errorf("invalid torrent file: pieces length is not a multiple of 20")
	}

	hashes := make([][]byte, 0, len(pieces)/20)
	for i := 0; i < len(pieces); i += 20 {
		hashes = append(hashes, pieces[i:i+20])
	}

	return hashes, nil
}

func printTorrentInfo(meta torrentMeta) {
	fmt.Println("Tracker URL:", meta.TrackerURL)
	fmt.Println("Length:", meta.Length)
	fmt.Printf("Info Hash: %x\n", meta.InfoHash)
	fmt.Println("Piece Length:", meta.PieceLength)
	fmt.Println("Piece Hashes:")
	for _, pieceHash := range meta.PieceHashes {
		fmt.Printf("%x\n", pieceHash)
	}
}
