#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DATA_DIR="${ROOT_DIR}/server/data"

# Sanity check $DATA_DIR
if [ ! -d "${DATA_DIR}" ]; then
  echo "Data directory not found: ${DATA_DIR}"
  exit 1
fi

# Collect TARGETS to delete
TARGETS=()
while IFS= read -r f; do
  TARGETS+=("$f")
done < <(find "${DATA_DIR}" -maxdepth 1 -type f -name '*.db*' | sort)

# Sanity check $TARGETS
if [ "${#TARGETS[@]}" -eq 0 ]; then
  echo "No database files found. Nothing to purge."
  exit 0
fi

echo "The following files will be removed:"
printf '  %s\n' "${TARGETS[@]}"
echo ""
read -r -p "Are you sure? Type 'yes' to continue: " answer

if [ "${answer}" != "yes" ]; then
  echo "Aborted."
  exit 0
fi

for f in "${TARGETS[@]}"; do
  rm -f "$f"
done
echo "Data purged."
