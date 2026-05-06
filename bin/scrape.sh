#!/usr/bin/env bash
set -euo pipefail

# Example: run a scrape ingest (refresh cache by default).
go run ./cmd/apnews scrape
