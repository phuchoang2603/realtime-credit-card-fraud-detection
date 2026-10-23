#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

: "${MUTATION_BASE:?Set MUTATION_BASE to the reviewed base commit}"
: "${CI_TEST_REPORTS:?Set CI_TEST_REPORTS to the CI artifact directory}"
report_dir="$(realpath -m "$CI_TEST_REPORTS")"
mkdir -p "$report_dir"
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
  --output "$report_dir/ledger-mutation.json" | tee "$report_dir/ledger-mutation.txt"
