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

case "${1:-}" in
  python)
    check_python src/fraud-service app tests
    check_python src/simulator simulator
    ;;
  helm)
    while IFS= read -r -d '' manifest; do
      check_chart "$(dirname "$manifest")"
    done < <(find infra -type f -name Chart.yaml -print0)
    ;;
  *)
    printf 'Usage: %s {python|helm}\n' "$0" >&2
    exit 2
    ;;
esac
