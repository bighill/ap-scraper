#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Run from module root so relative paths (./data, ../web) resolve correctly.
cd "${ROOT_DIR}/server"
air \
  --build.cmd "go build -o ../tmp/main ." \
  --build.bin "../tmp/main"
