package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

type peerMessage struct {
	ID      byte
	Payload []byte
}

func readPeerMessage(conn net.Conn) (peerMessage, error) {
	var msg peerMessage

	for { // for loop is to handle the keep alive message since we need to skip them and wait for next message
		lengthBuf := make([]byte, 4)
		_, err := io.ReadFull(conn, lengthBuf)
		if err != nil {
			return msg, fmt.Errorf("failed to read message length: %w", err)
		}

		length := int(binary.BigEndian.Uint32(lengthBuf))
		if length == 0 { // keep alive message, ignore it and wait for next message
			continue
		}

		body := make([]byte, length)
		_, err = io.ReadFull(conn, body)
		if err != nil {
			return msg, fmt.Errorf("failed to read message body: %w", err)
		}
		msg.ID = body[0]

		if length == 1 {
			msg.Payload = nil
			return msg, nil
		}

		msg.Payload = body[1:]
		return msg, nil
	}
}

func waitForBitfield(conn net.Conn) ([]byte, error) {
	bitfieldMsg, err := readPeerMessage(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to read message: %w", err)
	}

	if bitfieldMsg.ID != byte(5) {
		return nil, fmt.Errorf("expecting bitfield (message id: 5) and got %d instead.", bitfieldMsg.ID)
	}
	return bitfieldMsg.Payload, nil
}

func hasPiece(bitfield []byte, pieceIndex int) bool {
	byteIndex := pieceIndex / 8
	bitIndex := 7 - (pieceIndex % 8)

	if byteIndex >= len(bitfield) {
		return false
	}

	return (bitfield[byteIndex] & (1 << bitIndex)) != 0
}

func buildInterestedMessage() []byte {
	msg := make([]byte, 5)

	binary.BigEndian.PutUint32(msg[0:4], 1)
	msg[4] = 2

	return msg
}

func sendInterested(conn net.Conn) error {
	msg := buildInterestedMessage()

	n, err := conn.Write(msg)
	if err != nil {
		return fmt.Errorf("failed to send interested message: %w", err)
	}
	if n != len(msg) {
		return fmt.Errorf("failed to send full interested message")
	}

	return nil
}

func waitForUnchoke(conn net.Conn) error {
	unchokeMsg, err := readPeerMessage(conn)
	if err != nil {
		return fmt.Errorf("failed to read message: %w", err)
	}

	if unchokeMsg.ID != byte(1) {
		return fmt.Errorf("expecting unchoke (message id: 1) and got %d instead.", unchokeMsg.ID)
	}
	return nil
}

func buildRequestMsg(index, begin, length int) []byte {
	msg := make([]byte, 17)
	binary.BigEndian.PutUint32(msg[0:4], 13)
	msg[4] = byte(6)
	binary.BigEndian.PutUint32(msg[5:9], uint32(index))
	binary.BigEndian.PutUint32(msg[9:13], uint32(begin))
	binary.BigEndian.PutUint32(msg[13:17], uint32(length))

	return msg
}

func requestBlocks(conn net.Conn, pieceIndex, pieceLength int) error {
	blockSize := 16 * 1024

	for begin := 0; begin < pieceLength; begin += blockSize {
		blockLength := blockSize
		remaining := pieceLength - begin
		if remaining < blockSize {
			blockLength = remaining
		}

		msg := buildRequestMsg(pieceIndex, begin, blockLength)
		n, err := conn.Write(msg)
		if err != nil {
			return fmt.Errorf("failed to request block from peer: %w", err)
		}
		if n != len(msg) {
			return fmt.Errorf("failed to send full request message")
		}
	}
	return nil
}

func receiveBlocks(conn net.Conn, pieceIndex, pieceLength int) ([]byte, error) {
	blockSize := 16 * 1024
	pieceData := make([]byte, pieceLength)

	totalBlocks := (pieceLength + blockSize - 1) / blockSize

	for i := 0; i < totalBlocks; i++ {
		msg, err := readPeerMessage(conn)
		if err != nil {
			return nil, fmt.Errorf("failed to read piece message: %w", err)
		}

		if msg.ID != 7 {
			return nil, fmt.Errorf("expected piece message (id 7), got %d", msg.ID)
		}

		if len(msg.Payload) < 8 {
			return nil, fmt.Errorf("invalid piece payload: too short")
		}

		receivedIndex := int(binary.BigEndian.Uint32(msg.Payload[0:4]))
		begin := int(binary.BigEndian.Uint32(msg.Payload[4:8]))
		block := msg.Payload[8:]

		if receivedIndex != pieceIndex {
			return nil, fmt.Errorf("received piece index %d, expected %d", receivedIndex, pieceIndex)
		}

		if begin < 0 || begin >= pieceLength {
			return nil, fmt.Errorf("invalid block begin offset %d", begin)
		}

		if begin+len(block) > pieceLength {
			return nil, fmt.Errorf("block exceeds piece bounds")
		}

		copy(pieceData[begin:], block)
	}

	return pieceData, nil
}
