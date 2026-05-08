#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Run from repo root so relative paths (./data, ./web) resolve correctly.
cd "${ROOT_DIR}"
air \
  --build.cmd "go build -o ./tmp/main ./cmd/server" \
  --build.bin "./tmp/main"
