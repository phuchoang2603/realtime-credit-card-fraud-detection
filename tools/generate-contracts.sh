#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
temporary=$(mktemp -d)
trap 'rm -rf "$temporary"' EXIT
GOBIN="$temporary" go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.11
GOBIN="$temporary" go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.6.1
python_output=src/fraud-service
go_output=src/edge/gen
if [[ "${1:-}" == --check ]]; then
  python_output="$temporary/python"
  go_output="$temporary/go"
elif [[ $# != 0 ]]; then
  echo 'Usage: tools/generate-contracts.sh [--check]' >&2
  exit 2
fi
mkdir -p "$python_output" "$go_output"
uv run --locked --project src/fraud-service python -m grpc_tools.protoc \
  -I contracts \
  --python_out="$python_output" --pyi_out="$python_output" \
  --grpc_python_out="$python_output" \
  --plugin="protoc-gen-go=$temporary/protoc-gen-go" \
  --plugin="protoc-gen-go-grpc=$temporary/protoc-gen-go-grpc" \
  --go_out="$go_output" --go_opt=paths=source_relative \
  --go-grpc_out="$go_output" --go-grpc_opt=paths=source_relative \
  contracts/fraud/v1/fraud.proto
if [[ "${1:-}" == --check ]]; then
  for file in fraud/v1/fraud_pb2.py fraud/v1/fraud_pb2.pyi fraud/v1/fraud_pb2_grpc.py; do
    diff -u "src/fraud-service/$file" "$python_output/$file"
  done
  for file in fraud/v1/fraud.pb.go fraud/v1/fraud_grpc.pb.go; do
    diff -u "src/edge/gen/$file" "$go_output/$file"
  done
fi
