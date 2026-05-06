#!/usr/bin/env bash
set -euo pipefail

# Example: scrape using local cached HTML.
go run ./cmd/apnews scrape --use-cache
