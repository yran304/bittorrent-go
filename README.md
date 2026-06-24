# BitTorrent in Go

This repository contains my in-progress implementation of a BitTorrent client in Go.

It is being built as part of the [CodeCrafters "Build Your Own BitTorrent" challenge](https://app.codecrafters.io/courses/bittorrent/overview), but this repo is mainly where I track my own work and progress publicly.

## Status

This project is not complete yet.

Current progress:
- Recursive bencode decoding is implemented
- String, integer, list, and dictionary decoding are implemented
- Torrent file parsing captures the raw bencoded `info` dictionary for hashing
- The `info` command can print:
  tracker URL, file length, info hash, piece length, and piece hashes
- Tracker requests are implemented, including compact peer list parsing
- The `peers` command can fetch and print peer addresses from a tracker
- Peer handshakes are implemented over TCP
- The `handshake` command can connect to a peer and print the remote peer ID
- Single-piece downloads are implemented
- The `download_piece` command can download one piece, verify its SHA-1 hash, and write it to disk
- Full-file downloads are implemented
- The `download` command can download all pieces, verify each piece hash, assemble the file, and write it to disk
- Multi-peer downloading is implemented with worker-style piece scheduling
- Piece downloads currently use a small pipelined request window
- Local Go tests cover decoder behavior, torrent metadata extraction, tracker parsing, and handshake helpers
- More BitTorrent features still need to be added

## Goal

The long-term goal is to build a working BitTorrent client that can:
- parse `.torrent` files
- talk to trackers
- connect to peers
- download file data
- download complete files across multiple pieces
- continue improving peer coordination and protocol coverage

## Run locally

If you have Go installed, you can run the program with:

```sh
./your_program.sh
```

You can also manually test the decode command:

```sh
go run ./app decode "l5:helloi52ee"
go run ./app decode "d3:cow3:moo4:spam4:eggse"
```

To inspect a torrent file with the `info` command:

```sh
go run ./app info sample.torrent
```

Current sample output:

```text
Tracker URL: http://bittorrent-test-tracker.codecrafters.io/announce
Length: 92063
Info Hash: d69f91e6b2ae4c542468d1073a71d4ea13879a7f
Piece Length: 32768
Piece Hashes:
e876f67a2a8886e8f36b136726c30fa29703022d
6e2275e604a0766656736e81ff10b55204ad8d35
f00d937a0213df1982bc8d097227ad9e909acc17
```

To fetch peers from the tracker:

```sh
go run ./app peers sample.torrent
```

To perform a peer handshake:

```sh
go run ./app handshake sample.torrent 165.232.41.73:51556
```

Example output:

```text
Peer ID: 0102030405060708090a0b0c0d0e0f1011121314
```

To download a single piece to disk:

```sh
go run ./app download_piece -o /tmp/test-piece sample.torrent 0
```

This writes the downloaded piece bytes to the path passed after `-o`.

To download the full file to disk:

```sh
go run ./app download -o /tmp/test.txt sample.torrent
```

This downloads all pieces, verifies each piece hash, reassembles the file in memory, and writes the completed file to the path passed after `-o`.

## Run tests

Local tests are available so the decoder can be validated without relying on Codecrafters' hosted tests.

```sh
go test ./app
```

To run only the decoder test suite:

```sh
go test ./app -run TestDecodeBencode
```

To run only the `info` command checks:

```sh
go test ./app -run TestInfoCommand
```

To run tracker and handshake related tests:

```sh
go test ./app -run 'TestBuildTrackerURL|TestParseCompactPeers|TestParseTrackerPeers|TestBuildHandshake|TestParseHandshakeResponse'
```

## Notes

This is an active learning project, so the codebase will continue to change as more protocol features are implemented.

Current limitations:
- `download_piece` is still useful as a simpler single-piece debugging path
- Multi-peer downloading is in place, but retry/requeue behavior still needs hardening for failure cases
- More protocol features, including magnet link support, still need to be added
