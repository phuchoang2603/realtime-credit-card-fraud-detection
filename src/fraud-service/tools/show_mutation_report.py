"""Display recorded mutation evidence without rerunning or certifying a new source tree."""

import argparse
import json
from pathlib import Path

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("report", type=Path)
args = parser.parse_args()
report = json.loads(args.report.read_text())
print(f"Report: {args.report}")
print(f"Base: {report['base_revision']}")
print(f"Selected changed functions: {len(report['selected'])}")
for name, count in report["raw_counts"].items():
    print(f"{name}: {count}")
print(f"Score: {report['score']:.4%}" if report["score"] is not None else "Score: N/A")
print(f"Strict acceptance (>80%, no unresolved outcomes): {report['status']}")
print("Equivalent exclusions and unsupported changes are recorded in the JSON report.")
