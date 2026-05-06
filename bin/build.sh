#!/usr/bin/env bash
set -euo pipefail

# Example: build the apnews CLI binary.
go build -o ./apnews ./cmd/apnews
