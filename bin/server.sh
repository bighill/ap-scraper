#!/usr/bin/env bash
set -euo pipefail

# Run the server: GET http://localhost:8080/articles (see internal/config for addr).
go run ./cmd/server
