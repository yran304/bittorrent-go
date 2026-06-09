# BitTorrent in Go

This repository contains my in-progress implementation of a BitTorrent client in Go.

It is being built as part of the [CodeCrafters "Build Your Own BitTorrent" challenge](https://app.codecrafters.io/courses/bittorrent/overview), but this repo is mainly where I track my own work and progress publicly.

## Status

This project is not complete yet.

Current progress:
- Recursive bencode decoding is implemented
- String, integer, list, and dictionary decoding are implemented
- The `info` command can parse a `.torrent` file and extract the tracker URL and file length
- Local Go tests were added for decoder validation, including dictionary cases
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

## Run tests

Local tests are available so the decoder can be validated without relying on Codecrafters' hosted tests.

```sh
go test ./app
```

To run only the decoder test suite:

```sh
go test ./app -run TestDecodeBencode
```

## Notes

This is an active learning project, so the codebase will continue to change as more protocol features are implemented.
