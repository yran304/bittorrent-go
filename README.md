# BitTorrent in Go

This repository contains my in-progress implementation of a BitTorrent client in Go.

It is being built as part of the [CodeCrafters "Build Your Own BitTorrent" challenge](https://app.codecrafters.io/courses/bittorrent/overview), but this repo is mainly where I track my own work and progress publicly.

## Status

This project is not complete yet.

Current progress:
- Basic bencode decoding is underway
- String, integer, and list decoding are implemented
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

## Notes

This is an active learning project, so the codebase will continue to change as more protocol features are implemented.
