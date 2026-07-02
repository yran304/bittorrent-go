package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"
)

type peerMessage struct {
	ID      byte
	Payload []byte
}

func readPeerMessage(conn net.Conn) (peerMessage, error) {
	var msg peerMessage

	if err := setPeerDeadline(conn); err != nil {
		return msg, fmt.Errorf("failed to set peer read deadline: %w", err)
	}
	defer conn.SetDeadline(time.Time{})

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

func hasPiece(bitfield []byte, pieceIndex int) bool { // since in current challenge, all peers have all blocks, this func is not useful at the moment
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

	if err := setPeerDeadline(conn); err != nil {
		return fmt.Errorf("failed to set peer write deadline: %w", err)
	}
	defer conn.SetDeadline(time.Time{})

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

func requestBlock(conn net.Conn, pieceIndex, begin, length int) error {
	msg := buildRequestMsg(pieceIndex, begin, length)

	if err := setPeerDeadline(conn); err != nil {
		return fmt.Errorf("failed to set peer write deadline: %w", err)
	}
	defer conn.SetDeadline(time.Time{})

	n, err := conn.Write(msg)
	if err != nil {
		return fmt.Errorf("failed to request block from peer: %w", err)
	}
	if n != len(msg) {
		return fmt.Errorf("failed to send full request message")
	}

	return nil
}

func setPeerDeadline(conn net.Conn) error {
	return conn.SetDeadline(time.Now().Add(peerIOTimeout))
}
