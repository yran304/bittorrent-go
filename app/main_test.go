package main

import (
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"reflect"
	"testing"
)

func TestDecodeBencode(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    interface{}
		wantErr bool
	}{
		{
			name:  "string",
			input: "5:hello",
			want:  "hello",
		},
		{
			name:  "integer",
			input: "i52e",
			want:  52,
		},
		{
			name:  "flat list",
			input: "l5:helloi52ee",
			want:  []interface{}{"hello", 52},
		},
		{
			name:  "nested list",
			input: "ll5:helloi52eee",
			want:  []interface{}{[]interface{}{"hello", 52}},
		},
		{
			name:  "mixed nested list",
			input: "l4:spaml1:a1:bee",
			want:  []interface{}{"spam", []interface{}{"a", "b"}},
		},
		{
			name:  "dictionary",
			input: "d3:cow3:moo4:spam4:eggse",
			want: map[string]interface{}{
				"cow":  "moo",
				"spam": "eggs",
			},
		},
		{
			name:  "nested dictionary",
			input: "d4:listl3:one3:twoe4:nestd3:fooi42eee",
			want: map[string]interface{}{
				"list": []interface{}{"one", "two"},
				"nest": map[string]interface{}{
					"foo": 42,
				},
			},
		},
		{
			name:    "trailing data",
			input:   "5:hellojunk",
			wantErr: true,
		},
		{
			name:    "unterminated list",
			input:   "l5:helloi52e",
			wantErr: true,
		},
		{
			name:    "dictionary keys out of order",
			input:   "d4:spam4:eggs3:cow3:mooe",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeBencode(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q, got nil", tt.input)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tt.input, err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("decodeBencode(%q) = %#v, want %#v", tt.input, got, tt.want)
			}
		})
	}
}

func TestDecodeValueConsumedBytes(t *testing.T) {
	got, consumed, _, err := decodeValue("5:helloi52e")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != "hello" {
		t.Fatalf("got value %#v, want %q", got, "hello")
	}

	if consumed != 7 {
		t.Fatalf("consumed = %d, want 7", consumed)
	}
}

func TestInfoCommandDataExtraction(t *testing.T) {
	data, err := os.ReadFile("../sample.torrent")
	if err != nil {
		t.Fatalf("failed to read sample torrent: %v", err)
	}

	torrent, infoRaw, err := decodeTorrentFile(string(data))
	if err != nil {
		t.Fatalf("failed to decode sample torrent: %v", err)
	}

	if infoRaw == "" {
		t.Fatal("expected raw info dictionary to be captured")
	}

	infoHash := sha1.Sum([]byte(infoRaw))
	announce, ok := torrent["announce"].(string)
	if !ok {
		t.Fatalf("announce has type %T, want string", torrent["announce"])
	}

	if announce != "http://bittorrent-test-tracker.codecrafters.io/announce" {
		t.Fatalf("announce = %q, want %q", announce, "http://bittorrent-test-tracker.codecrafters.io/announce")
	}

	info, ok := torrent["info"].(map[string]interface{})
	if !ok {
		t.Fatalf("info has type %T, want map[string]interface{}", torrent["info"])
	}

	length, ok := info["length"].(int)
	if !ok {
		t.Fatalf("length has type %T, want int", info["length"])
	}

	if length != 92063 {
		t.Fatalf("length = %d, want 92063", length)
	}

	if got := fmt.Sprintf("%x", infoHash); got != "d69f91e6b2ae4c542468d1073a71d4ea13879a7f" {
		t.Fatalf("info hash = %s, want %s", got, "d69f91e6b2ae4c542468d1073a71d4ea13879a7f")
	}

	pieceLength, ok := info["piece length"].(int)
	if !ok {
		t.Fatalf("piece length has type %T, want int", info["piece length"])
	}

	if pieceLength != 32768 {
		t.Fatalf("piece length = %d, want 32768", pieceLength)
	}

	pieces, ok := info["pieces"].(string)
	if !ok {
		t.Fatalf("pieces has type %T, want string", info["pieces"])
	}

	if len([]byte(pieces)) != 60 {
		t.Fatalf("pieces byte length = %d, want 60", len([]byte(pieces)))
	}
}

func TestInfoCommandOutput(t *testing.T) {
	originalArgs := os.Args
	originalStdout := os.Stdout
	defer func() {
		os.Args = originalArgs
		os.Stdout = originalStdout
	}()

	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdout pipe: %v", err)
	}

	os.Args = []string{"your_program.sh", "info", "../sample.torrent"}
	os.Stdout = writePipe

	main()

	if err := writePipe.Close(); err != nil {
		t.Fatalf("failed to close stdout writer: %v", err)
	}

	output, err := io.ReadAll(readPipe)
	if err != nil {
		t.Fatalf("failed to read stdout: %v", err)
	}

	expected := "" +
		"Tracker URL: http://bittorrent-test-tracker.codecrafters.io/announce\n" +
		"Length: 92063\n" +
		"Info Hash: d69f91e6b2ae4c542468d1073a71d4ea13879a7f\n" +
		"Piece Length: 32768\n" +
		"Piece Hashes:\n" +
		"e876f67a2a8886e8f36b136726c30fa29703022d\n" +
		"6e2275e604a0766656736e81ff10b55204ad8d35\n" +
		"f00d937a0213df1982bc8d097227ad9e909acc17\n"

	if string(output) != expected {
		t.Fatalf("info command output = %q, want %q", string(output), expected)
	}
}
