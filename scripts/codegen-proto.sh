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

declare -A go_protos=(
  [edge]="payments/v1/types.proto payments/v1/payments.proto vault/v1/vault.proto"
  [payments]="payments/v1/types.proto payments/v1/events.proto payments/v1/payments.proto fraud/v1/fraud.proto processor/v1/processor.proto"
  [notifier]="payments/v1/types.proto payments/v1/events.proto"
  [vault-simulator]="vault/v1/vault.proto"
  [processor-simulator]="processor/v1/processor.proto vault/v1/vault.proto"
)

go_package_name() {
  local dir
  dir="$(dirname "$1")"
  printf '%s%s' "${dir%%/*}" "${dir##*/}"
}

for module in "${!go_protos[@]}"; do
  out="src/$module/gen"
  rm -rf "$out"
  mkdir -p "$out"
  read -r -a files <<<"${go_protos[$module]}"
  mapping=()
  for file in "${files[@]}"; do
    mapping+=("M$file=$GO_MODULE_PREFIX/$module/gen/$(dirname "$file");$(go_package_name "$file")")
  done
  options="$(IFS=,; printf '%s' "${mapping[*]}")"
  protoc \
    -I contracts \
    --go_out="$out" --go_opt="paths=import,module=$GO_MODULE_PREFIX/$module/gen,$options" \
    --go-grpc_out="$out" --go-grpc_opt="paths=import,module=$GO_MODULE_PREFIX/$module/gen,$options" \
    "${files[@]}"
done
