#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."
mkdir -p src/fraud-service/gen

mapfile -d '' protos < <(find contracts -type f -name '*.proto' -print0 | sort -z)
protoc \
  -I contracts \
  --python_out=src/fraud-service/gen --pyi_out=src/fraud-service/gen \
  --plugin="protoc-gen-grpc_python=$1" \
  --grpc_python_out=src/fraud-service/gen \
  "${protos[@]}"
