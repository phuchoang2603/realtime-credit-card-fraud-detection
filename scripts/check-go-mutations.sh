#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."
project="${1:?Set the Go module directory}"
mode="${2:-diff}"
: "${CI_TEST_REPORTS:?Set CI_TEST_REPORTS to the CI artifact directory}"
report_dir="$(realpath -m "$CI_TEST_REPORTS")/${project#src/}"
mkdir -p "$report_dir"
cd "$project"

exclude="(^|/)(proto|sqlcgen|testutil|testsupport|testvectors|db|config|metrics|transport|app)/|(^|/)cmd/$(basename "$PWD")/|(_test|\.pb|_grpc\.pb)\.go$"
args=()
case "$mode" in
  diff)
    : "${MUTATION_BASE:?Set MUTATION_BASE to the reviewed base commit}"
    args+=(--diff "$MUTATION_BASE")
    candidates="$(git -c diff.relative=true diff --merge-base --name-only --diff-filter=ACMR "$MUTATION_BASE" -- '*.go')"
    ;;
  full)
    candidates="$(git ls-files -- '*.go')"
    ;;
  *)
    printf 'Usage: %s <module> {diff|full}\n' "$0" >&2
    exit 2
    ;;
esac

eligible=false
while IFS= read -r file; do
  if [[ "$file" == *.go && -f "$file" && ! "$file" =~ $exclude ]]; then
    eligible=true
    break
  fi
done <<< "$candidates"
if [[ "$eligible" == false ]]; then
  printf 'Skipped %s: no eligible Go source in %s scope.\n' "$project" "$mode" | tee "$report_dir/mutation.txt"
  if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
    cat "$report_dir/mutation.txt" >> "$GITHUB_STEP_SUMMARY"
  fi
  exit 0
fi

mutation_dir="$(mktemp -d)"
trap 'rm -rf "$mutation_dir"' EXIT
cp go.mod "$mutation_dir/go.mod"
cp go.sum "$mutation_dir/go.sum"
if [[ -d ../shared && "$(basename "$PWD")" != shared ]]; then
  go mod edit -modfile "$mutation_dir/go.mod" \
    -replace "github.com/phuchoang2603/payment-gateway-fraud-detection/shared=$(cd ../shared && pwd)"
fi
status=0
GOWORK=off GOFLAGS="-p=1 -modfile=$mutation_dir/go.mod" \
  GIT_CONFIG_COUNT=1 GIT_CONFIG_KEY_0=diff.relative GIT_CONFIG_VALUE_0=true \
  timeout --signal=TERM --kill-after=30s "${MUTATION_BUDGET:-15m}" \
  gremlins unleash "${args[@]}" --exclude-files "$exclude" --workers 2 --test-cpu 1 \
  --output "$report_dir/mutation.json" | tee "$report_dir/mutation.txt" || status=$?
if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
  {
    printf '\n### Mutation report: %s (%s)\n\n```text\n' "$project" "$mode"
    tail -n 18 "$report_dir/mutation.txt"
    printf '\n```\n'
  } >> "$GITHUB_STEP_SUMMARY"
fi
exit "$status"
