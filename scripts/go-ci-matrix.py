#!/usr/bin/env python3
import json
import subprocess
from pathlib import PurePosixPath

manifests = (
    subprocess.check_output(["git", "ls-files", "-z", "--", "src/**/go.mod"])
    .decode()
    .split("\0")
)
projects = [
    {
        "project": str(PurePosixPath(path).parent),
        "id": str(PurePosixPath(path).parent).replace("/", "-"),
    }
    for path in manifests
    if path
]
if not projects:
    raise SystemExit("No tracked Go modules found")
print(json.dumps({"include": projects}, separators=(",", ":")))
