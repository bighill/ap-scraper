#!/usr/bin/env bash
set -euo pipefail

# Build the server binary (HTTP API + scheduler).
go build -o ./apnews-server ./cmd/server
