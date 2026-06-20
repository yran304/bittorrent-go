package main

import (
	"crypto/rand"
	"fmt"
	"io"
	"net"
)

const bittorrentProtocol = "BitTorrent protocol"

func buildHandshake(infoHash [20]byte, localPeerID [20]byte) []byte {
	message := make([]byte, 0, 68)

	message = append(message, byte(len(bittorrentProtocol)))
	message = append(message, []byte(bittorrentProtocol)...)
	message = append(message, make([]byte, 8)...)
	message = append(message, infoHash[:]...)
	message = append(message, localPeerID[:]...)

	return message
}

func generatePeerID() ([20]byte, error) {
	var localPeerID [20]byte
	_, err := rand.Read(localPeerID[:])
	if err != nil {
		return localPeerID, fmt.Errorf("failed to generate peer ID: %w", err)
	}
	return localPeerID, nil
}

func parseHandshakeResponse(resp []byte, expectedInfoHash [20]byte) ([20]byte, error) {
	var remotePeerID [20]byte

	if len(resp) != 68 {
		return remotePeerID, fmt.Errorf("invalid handshake length: got %d, want 68", len(resp))
	}

	if resp[0] != byte(len(bittorrentProtocol)) {
		return remotePeerID, fmt.Errorf("invalid protocol length: got %d, want %d", resp[0], len(bittorrentProtocol))
	}

	if string(resp[1:20]) != bittorrentProtocol {
		return remotePeerID, fmt.Errorf("protocol mismatch")
	}

	if string(resp[28:48]) != string(expectedInfoHash[:]) {
		return remotePeerID, fmt.Errorf("info hash mismatch")
	}

	copy(remotePeerID[:], resp[48:68])
	return remotePeerID, nil
}

func performHandshake(infoHash [20]byte, peerAddr string) ([20]byte, error) {
	var remotePeerID [20]byte

	localPeerID, err := generatePeerID()
	if err != nil {
		return remotePeerID, fmt.Errorf("failed to generate local peer ID: %w", err)
	}

	handshakeMsg := buildHandshake(infoHash, localPeerID)

	conn, err := net.Dial("tcp", peerAddr)
	if err != nil {
		return remotePeerID, fmt.Errorf("failed to connect to peer: %w", err)
	}
	defer conn.Close()

	n, err := conn.Write(handshakeMsg)
	if err != nil {
		return remotePeerID, fmt.Errorf("failed to send handshake to peer: %w", err)
	}
	if n != len(handshakeMsg) {
		return remotePeerID, fmt.Errorf("failed to send full handshake")
	}

	resp := make([]byte, 68)
	_, err = io.ReadFull(conn, resp)
	if err != nil {
		return remotePeerID, fmt.Errorf("failed to read handshake response: %w", err)
	}

	remotePeerID, err = parseHandshakeResponse(resp, infoHash)
	if err != nil {
		return remotePeerID, fmt.Errorf("failed to parse handshake response: %w", err)
	}

	return remotePeerID, nil
}
