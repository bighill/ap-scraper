#!/usr/bin/env bash
set -euo pipefail

# Example: run all Go tests.
go -C server test ./...
