package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

const (
	defaultPeerID = "00112233445566778899"
	defaultPort   = 6881
)

type trackerRequest struct {
	InfoHash   [20]byte
	PeerID     string
	Port       int
	Uploaded   int
	Downloaded int
	Left       int
	Compact    int
}

func newTrackerRequest(meta torrentMeta) trackerRequest {
	return trackerRequest{
		InfoHash:   meta.InfoHash,
		PeerID:     defaultPeerID,
		Port:       defaultPort,
		Uploaded:   0,
		Downloaded: 0,
		Left:       meta.Length,
		Compact:    1, // for current stage, we will always request compact peer lists
	}
}

func buildTrackerURL(meta torrentMeta) (string, error) {
	baseURL, err := url.Parse(meta.TrackerURL)
	if err != nil {
		return "", err
	}

	req := newTrackerRequest(meta)
	query := baseURL.Query()

	// info_hash must be the raw 20-byte SHA-1 value, not the 40-char hex string.
	query.Set("info_hash", string(req.InfoHash[:]))
	query.Set("peer_id", req.PeerID)
	query.Set("port", strconv.Itoa(req.Port))
	query.Set("uploaded", strconv.Itoa(req.Uploaded))
	query.Set("downloaded", strconv.Itoa(req.Downloaded))
	query.Set("left", strconv.Itoa(req.Left))
	query.Set("compact", strconv.Itoa(req.Compact))

	baseURL.RawQuery = query.Encode()
	return baseURL.String(), nil
}

func fetchTrackerPeers(meta torrentMeta) ([]string, error) {
	trackerURL, err := buildTrackerURL(meta)
	if err != nil {
		return nil, fmt.Errorf("failed to build tracker URL: %v", err)
	}

	trackerResp, err := http.Get(trackerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tracker peers: %v", err)
	}
	defer trackerResp.Body.Close()

	if trackerResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tracker request failed with status: %s", trackerResp.Status)
	}

	body, err := io.ReadAll(trackerResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read tracker response: %v", err)
	}

	peerAddrs, err := parseTrackerPeers(body)
	if err != nil {
		return nil, err
	}

	return peerAddrs, nil
}

func parseTrackerPeers(body []byte) ([]string, error) {
	decodedResp, err := decodeBencode(string(body))
	if err != nil {
		return nil, fmt.Errorf("failed to decode tracker response: %v", err)
	}

	trackerData, ok := decodedResp.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid tracker response: expected a dictionary")
	}
	fmt.Printf("%#v\n", trackerData)
	peers, ok := trackerData["peers"].(string)
	if !ok {
		return nil, fmt.Errorf("tracker response missing peers")
	}

	return parseCompactPeers([]byte(peers))
}

func parseCompactPeers(peersBytes []byte) ([]string, error) {
	if len(peersBytes)%6 != 0 {
		return nil, fmt.Errorf("invalid peers data: length is not a multiple of 6")
	}

	var peerAddrs []string
	for i := 0; i < len(peersBytes); i += 6 {
		ip := fmt.Sprintf("%d.%d.%d.%d", peersBytes[i], peersBytes[i+1], peersBytes[i+2], peersBytes[i+3])
		port := int(peersBytes[i+4])<<8 | int(peersBytes[i+5]) // bitwise OR to combine the two bytes into a single port number

		peerAddrs = append(peerAddrs, fmt.Sprintf("%s:%d", ip, port))
	}

	return peerAddrs, nil
}
