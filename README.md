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
- Local Go tests cover decoder behavior and the expected `info` command output
- More BitTorrent features still need to be added

## Goal

The long-term goal is to build a working BitTorrent client that can:
- parse `.torrent` files
- talk to trackers
- connect to peers
- download file data

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

## Notes

This is an active learning project, so the codebase will continue to change as more protocol features are implemented.
