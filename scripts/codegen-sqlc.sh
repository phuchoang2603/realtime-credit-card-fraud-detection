#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

while IFS= read -r -d '' config; do
  (cd "$(dirname "$config")" && sqlc generate)
done < <(find src -maxdepth 2 -name sqlc.yaml -print0)
