package main

import (
	"fmt"
	"io"
	"net/url"
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
			name:  "dictionary keys out of order",
			input: "d4:spam4:eggs3:cow3:mooe",
			want: map[string]interface{}{
				"spam": "eggs",
				"cow":  "moo",
			},
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
	got, consumed, _, err := decodeValue("5:helloi52e", decodeOptions{})
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
	meta, err := readTorrentMeta("../sample.torrent")
	if err != nil {
		t.Fatalf("failed to read torrent metadata: %v", err)
	}

	if meta.TrackerURL != "http://bittorrent-test-tracker.codecrafters.io/announce" {
		t.Fatalf("tracker URL = %q, want %q", meta.TrackerURL, "http://bittorrent-test-tracker.codecrafters.io/announce")
	}

	if meta.Length != 92063 {
		t.Fatalf("length = %d, want 92063", meta.Length)
	}

	if got := fmt.Sprintf("%x", meta.InfoHash); got != "d69f91e6b2ae4c542468d1073a71d4ea13879a7f" {
		t.Fatalf("info hash = %s, want %s", got, "d69f91e6b2ae4c542468d1073a71d4ea13879a7f")
	}

	if meta.PieceLength != 32768 {
		t.Fatalf("piece length = %d, want 32768", meta.PieceLength)
	}

	if len(meta.PieceHashes) != 3 {
		t.Fatalf("piece hash count = %d, want 3", len(meta.PieceHashes))
	}

	wantHashes := []string{
		"e876f67a2a8886e8f36b136726c30fa29703022d",
		"6e2275e604a0766656736e81ff10b55204ad8d35",
		"f00d937a0213df1982bc8d097227ad9e909acc17",
	}

	for i, want := range wantHashes {
		if got := fmt.Sprintf("%x", meta.PieceHashes[i]); got != want {
			t.Fatalf("piece hash %d = %s, want %s", i, got, want)
		}
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

func TestDecodeTorrentFileRejectsOutOfOrderKeys(t *testing.T) {
	_, _, err := decodeTorrentFile("d4:spam4:eggs3:cow3:mooe")
	if err == nil {
		t.Fatal("expected decodeTorrentFile to reject out-of-order dictionary keys")
	}
}

func TestBuildTrackerURL(t *testing.T) {
	meta, err := readTorrentMeta("../sample.torrent")
	if err != nil {
		t.Fatalf("failed to read torrent metadata: %v", err)
	}

	trackerURL, err := buildTrackerURL(meta)
	if err != nil {
		t.Fatalf("failed to build tracker URL: %v", err)
	}

	parsedURL, err := url.Parse(trackerURL)
	if err != nil {
		t.Fatalf("failed to parse tracker URL: %v", err)
	}

	query := parsedURL.Query()

	if got := parsedURL.Scheme + "://" + parsedURL.Host + parsedURL.Path; got != meta.TrackerURL {
		t.Fatalf("base tracker URL = %q, want %q", got, meta.TrackerURL)
	}

	if got := query.Get("peer_id"); got != defaultPeerID {
		t.Fatalf("peer_id = %q, want %q", got, defaultPeerID)
	}

	if got := query.Get("port"); got != "6881" {
		t.Fatalf("port = %q, want %q", got, "6881")
	}

	if got := query.Get("uploaded"); got != "0" {
		t.Fatalf("uploaded = %q, want %q", got, "0")
	}

	if got := query.Get("downloaded"); got != "0" {
		t.Fatalf("downloaded = %q, want %q", got, "0")
	}

	if got := query.Get("left"); got != "92063" {
		t.Fatalf("left = %q, want %q", got, "92063")
	}

	if got := query.Get("compact"); got != "1" {
		t.Fatalf("compact = %q, want %q", got, "1")
	}

	if got := query.Get("info_hash"); got != string(meta.InfoHash[:]) {
		t.Fatalf("info_hash bytes do not match the raw 20-byte hash")
	}
}

func TestParseCompactPeers(t *testing.T) {
	peersBytes := []byte{
		165, 232, 41, 73, 201, 100,
		165, 232, 38, 164, 201, 37,
		165, 232, 35, 114, 201, 20,
	}

	got, err := parseCompactPeers(peersBytes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{
		"165.232.41.73:51556",
		"165.232.38.164:51493",
		"165.232.35.114:51476",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseCompactPeers() = %#v, want %#v", got, want)
	}
}

func TestParseCompactPeersRejectsInvalidLength(t *testing.T) {
	_, err := parseCompactPeers([]byte{1, 2, 3, 4, 5})
	if err == nil {
		t.Fatal("expected parseCompactPeers to reject non-6-byte-aligned data")
	}
}

func TestParseTrackerPeers(t *testing.T) {
	body := []byte("d8:intervali1800e5:peers18:\xa5\xe8)I\xc9d\xa5\xe8&\xa4\xc9%\xa5\xe8#r\xc9\x14e")

	got, err := parseTrackerPeers(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{
		"165.232.41.73:51556",
		"165.232.38.164:51493",
		"165.232.35.114:51476",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseTrackerPeers() = %#v, want %#v", got, want)
	}
}
