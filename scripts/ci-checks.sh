#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

check_python() {
  local project="$1" source="$2" tests="${3:-}"
  local -a paths=("$source")
  [[ -z "$tests" ]] || paths+=("$tests")
  (
    cd "$project"
    uv run --locked ruff check "${paths[@]}"
    uv run --locked ruff format --check "${paths[@]}"
    if [[ -n "$tests" ]]; then
      uv run --locked pytest -q "$tests"
    fi
  )
}

check_data_product() {
  local product="$1" name contract reference
  name="$(basename "$product")"
  for contract in "$product"/*.odcs.yaml; do
    reference="docs/data-ml/data-products/$name/$(basename "$contract" .odcs.yaml).md"
    uvx --from "datacontract-cli==$DATACONTRACT_CLI" datacontract lint "$contract" --all-errors
    uvx --from "datacontract-cli==$DATACONTRACT_CLI" datacontract export markdown "$contract" --output "$TMPDIR_CHECKS/reference.md"
    diff -u "$reference" "$TMPDIR_CHECKS/reference.md"
  done
  (
    cd "$product"
    uvx --from "dataproduct-cli==$DATAPRODUCT_CLI" --with "datacontract-cli==$DATACONTRACT_CLI" \
      dataproduct lint product.odps.yaml --all-errors --local-references
  )
}

check_chart() {
  local chart="$1" values
  helm lint "$chart"
  helm template check "$chart" > /dev/null
  for values in "$chart"/values-*.yaml; do
    [[ -f "$values" ]] || continue
    helm lint "$chart" -f "$values"
    helm template check "$chart" -f "$values" > /dev/null
  done
}

DATACONTRACT_CLI=1.2.7
DATAPRODUCT_CLI=0.3.2

case "${1:-}" in
  go)
    report_dir="${GATEWAY_TEST_REPORTS:-$(mktemp -d)}"
    mkdir -p "$report_dir"
    while IFS= read -r -d '' manifest; do
      (
        cd "$(dirname "$manifest")"
        GOWORK=off golangci-lint run ./...
        module_name="$(basename "$PWD")"
        GOWORK=off go test -p 1 -race -count=1 -coverprofile "$report_dir/$module_name.out" ./...
        go tool cover -func "$report_dir/$module_name.out" | tee "$report_dir/$module_name.txt"
        go tool cover -html "$report_dir/$module_name.out" -o "$report_dir/$module_name.html"
      )
    done < <(find src -type f -name go.mod -not -path '*/vendor/*' -print0 | sort -z)
    ;;
  go-mutation)
    : "${MUTATION_BASE:?Set MUTATION_BASE to the reviewed base commit}"
    : "${GATEWAY_TEST_REPORTS:?Set GATEWAY_TEST_REPORTS to the CI artifact directory}"
    mkdir -p "$GATEWAY_TEST_REPORTS"
    cd src/payment-gateway/payments/cmd/ledger-check
    mutation_dir="$(mktemp -d)"
    trap 'rm -rf "$mutation_dir"' EXIT
    cp ../../go.mod "$mutation_dir/go.mod"
    cp ../../go.sum "$mutation_dir/go.sum"
    go mod edit -modfile "$mutation_dir/go.mod" \
      -replace "github.com/phuchoang2603/payment-gateway-fraud-detection/shared=$(cd ../../../shared && pwd)"
    GOWORK=off GOFLAGS="-p=1 -modfile=$mutation_dir/go.mod" \
      GIT_CONFIG_COUNT=1 GIT_CONFIG_KEY_0=diff.relative GIT_CONFIG_VALUE_0=true \
      gremlins unleash --diff "$MUTATION_BASE" --workers 2 --test-cpu 1 \
      --output "$GATEWAY_TEST_REPORTS/ledger-mutation.json" | tee "$GATEWAY_TEST_REPORTS/ledger-mutation.txt"
    ;;
  contracts)
    TMPDIR_CHECKS="$(mktemp -d)"
    trap 'rm -rf "$TMPDIR_CHECKS"' EXIT
    while IFS= read -r -d '' product; do
      check_data_product "$(dirname "$product")"
    done < <(find contracts/data-products -mindepth 2 -maxdepth 2 -name product.odps.yaml -print0 | sort -z)
    ;;
  python)
    check_python src/fraud-service app tests
    check_python src/simulator simulator tests
    ;;
  helm)
    while IFS= read -r -d '' manifest; do
      check_chart "$(dirname "$manifest")"
    done < <(find infra -type f -name Chart.yaml -print0)
    ;;
  *)
    printf 'Usage: %s {contracts|python|go|go-mutation|helm}\n' "$0" >&2
    exit 2
    ;;
esac
