#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
IMAGE_NAME="ap-scraper:prod"
CONTAINER_NAME="ap-scraper-prod"

mkdir -p "${ROOT_DIR}/data"

echo "Build new image..."

docker build -t "${IMAGE_NAME}" "${ROOT_DIR}"

if docker ps -a --format '{{.Names}}' | rg -x "${CONTAINER_NAME}" >/dev/null; then
  echo "Kill old image..."
  docker stop "${CONTAINER_NAME}" >/dev/null
  docker rm "${CONTAINER_NAME}" >/dev/null
fi

echo "Run comainer"

docker run -d \
  --name "${CONTAINER_NAME}" \
  --restart unless-stopped \
  -p 9191:9191 \
  -v "${ROOT_DIR}/web:/app/web" \
  -v "${ROOT_DIR}/data:/app/data" \
  "${IMAGE_NAME}"

echo "Container ${CONTAINER_NAME} is running on http://localhost:9191"
