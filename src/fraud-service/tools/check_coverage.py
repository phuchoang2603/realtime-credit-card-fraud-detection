"""Gate raw statement counts; rounded display percentages are not acceptance."""

import argparse
import json
from pathlib import Path


def passes(totals: dict[str, int]) -> bool:
    covered, total = totals["covered_lines"], totals["num_statements"]
    return 0 <= covered <= total and total > 0 and covered * 10 > total * 9


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("report", type=Path)
    args = parser.parse_args()
    totals = json.loads(args.report.read_text())["totals"]
    accepted = passes(totals)
    print(f"Isolated statement coverage: {totals['covered_lines']}/{totals['num_statements']}; >90%: {accepted}")
    return 0 if accepted else 1


if __name__ == "__main__":
    raise SystemExit(main())
