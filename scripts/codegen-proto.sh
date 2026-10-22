#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

GO_MODULE_PREFIX=github.com/phuchoang2603/payment-gateway-fraud-detection

mkdir -p src/fraud-service/gen
mapfile -d '' protos < <(find contracts -type f -name '*.proto' -print0 | sort -z)
protoc \
  -I contracts \
  --python_out=src/fraud-service/gen --pyi_out=src/fraud-service/gen \
  --plugin="protoc-gen-grpc_python=$1" \
  --grpc_python_out=src/fraud-service/gen \
  "${protos[@]}"

mkdir -p shared/proto
protoc \
  -I contracts \
  --go_out=shared --go_opt="module=$GO_MODULE_PREFIX/shared" \
  --go-grpc_out=shared --go-grpc_opt="module=$GO_MODULE_PREFIX/shared" \
  "${protos[@]}"
