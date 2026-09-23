"""Select changed production functions and invoke the pinned runner's public CLI.

Run in a temporary source snapshot: no private mutmut configuration, stale cache,
coverage-based filtering, or mutation of the developer's working tree.
"""

from __future__ import annotations

import argparse
import ast
import fnmatch
import json
import os
import re
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

SERVICE = Path("src/fraud-service")
# Declarative Pydantic fields are validated by request/contract tests; mutmut
# cannot mutate class-level field definitions. Never treat them as evaluated.
DECLARATIVE_EXCLUSIONS = {
    "app/schema.py": "Pydantic class fields: exercised by gRPC request validation tests; no mutable functions",
}


def functions(source: str) -> dict[str, ast.AST]:
    tree = ast.parse(source)
    result = {}
    for node in tree.body:
        if isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef)):
            result[node.name] = node
        elif isinstance(node, ast.ClassDef):
            for method in node.body:
                if isinstance(method, (ast.FunctionDef, ast.AsyncFunctionDef)):
                    result[f"{node.name}.{method.name}"] = method
    return result


def module_statements(source: str) -> str:
    tree = ast.parse(source)
    for node in tree.body:
        if isinstance(node, ast.ClassDef):
            node.body = [n for n in node.body if not isinstance(n, (ast.FunctionDef, ast.AsyncFunctionDef))]
    tree.body = [n for n in tree.body if not isinstance(n, (ast.FunctionDef, ast.AsyncFunctionDef))]
    return ast.dump(tree)


def select_functions(before: str, after: str) -> dict:
    old, new = functions(before), functions(after)
    # Changed imports/constants/class attributes can affect every function.
    module_changed = module_statements(before) != module_statements(after)
    selected = [
        name
        for name, node in new.items()
        if module_changed
        or name not in old
        or ast.dump(node) != ast.dump(old[name])
        or node.lineno != old[name].lineno
    ]
    return {
        "functions": selected,
        "deleted_functions": sorted(old.keys() - new.keys()),
        "module_changed": module_changed,
    }


def git(repository: Path, *arguments: str) -> str:
    return subprocess.check_output(["git", *arguments], cwd=repository, text=True)


def selection_manifest(base: str, repository: Path) -> dict:
    revision = git(repository, "rev-parse", "--verify", f"{base}^{{commit}}").strip()
    tracked = git(repository, "diff", "--name-only", "--no-renames", revision, "--", str(SERVICE / "app"))
    untracked = git(repository, "ls-files", "--others", "--exclude-standard", "--", str(SERVICE / "app"))
    manifest = {"base_revision": revision, "selected": [], "deleted": [], "exclusions": [], "unsupported": []}
    for filename in sorted(set((tracked + untracked).splitlines())):
        if not filename.endswith(".py"):
            continue
        relative = str(Path(filename).relative_to(SERVICE))
        previous = subprocess.run(
            ["git", "show", f"{revision}:{filename}"], cwd=repository, capture_output=True, text=True
        )
        before = previous.stdout if previous.returncode == 0 else ""
        path = repository / filename
        after = path.read_text() if path.is_file() else ""
        selection = select_functions(before, after)
        for name in selection["deleted_functions"]:
            manifest["deleted"].append(f"{relative}:{name}")
        for name in selection["functions"]:
            node = functions(after)[name]
            manifest["selected"].append(
                {
                    "file": relative,
                    "function": name,
                    "line": node.lineno,
                    "end_line": node.end_lineno,
                    "module_changed": selection["module_changed"],
                }
            )
        if selection["module_changed"] and not selection["functions"] and after.strip():
            if relative in DECLARATIVE_EXCLUSIONS:
                manifest["exclusions"].append({"file": relative, "reason": DECLARATIVE_EXCLUSIONS[relative]})
            else:
                manifest["unsupported"].append(
                    {"file": relative, "reason": "Changed module statements without mutable functions"}
                )
    return manifest


def mutant_pattern(target: dict) -> str:
    module = target["file"][:-3].replace("/", ".").removesuffix(".__init__")
    parts = target["function"].split(".")
    mangled = "x_" + parts[0] if len(parts) == 1 else "xǁ" + "ǁ".join(parts)
    return f"{module}.{mangled}__mutmut_*"


def evaluate(outcomes: list[str], unsupported: bool = False) -> dict:
    killed = outcomes.count("killed")
    survived = outcomes.count("survived")
    unresolved = len(outcomes) - killed - survived
    score = killed / (killed + survived) if killed + survived else None
    status = (
        "FAIL"
        if unsupported or unresolved
        else ("N/A" if score is None else "PASS" if killed * 5 > (killed + survived) * 4 else "FAIL")
    )
    return {
        "status": status,
        "score": score,
        "raw_counts": {
            "generated": len(outcomes),
            "killed": killed,
            "survived": survived,
            "unresolved": unresolved,
            "equivalent_exclusions": 0,
        },
    }


def run_selected(manifest: dict, service: Path, children: int, log_path: Path, debug_dir: Path | None = None) -> dict:
    patterns = [mutant_pattern(target) for target in manifest["selected"]]
    with tempfile.TemporaryDirectory(prefix="fraud-mutation-") as directory:
        root = Path(directory)
        snapshot = root / SERVICE
        snapshot.mkdir(parents=True)
        for name in ("app", "tests", "tools", "models", "fraud"):
            shutil.copytree(
                service / name, snapshot / name, ignore=shutil.ignore_patterns("__pycache__", ".pytest_cache")
            )
        shutil.copytree(service.parents[1] / "contracts", root / "contracts")
        shutil.copy2(service / "pyproject.toml", snapshot / "pyproject.toml")
        environment = os.environ | {"TESTING_MODE": "true"}
        with log_path.open("w") as log:

            def run(*args):
                return subprocess.run(
                    [sys.executable, *args],
                    cwd=snapshot,
                    env=environment,
                    stdout=log,
                    stderr=subprocess.STDOUT,
                    timeout=900,
                    check=True,
                )

            run("-m", "pytest", "-q", "-m", "not property", "tests")
            run("-m", "mutmut", "run", "--max-children", str(children), *patterns)
            result = subprocess.check_output(
                [sys.executable, "-m", "mutmut", "results", "--all", "true"],
                cwd=snapshot,
                env=environment,
                text=True,
                timeout=30,
            )
            log.write(result)
        results = []
        for name, status in re.findall(r"^\s+(\S+): ([\w -]+)$", result, re.MULTILINE):
            if any(fnmatch.fnmatchcase(name, pattern) for pattern in patterns):
                results.append({"mutant": name, "outcome": status.strip()})
        if debug_dir is not None:
            shutil.copytree(root, debug_dir)
        if not results:
            raise RuntimeError("Runner produced no matching mutant results for selected functions")
        return evaluate([r["outcome"] for r in results], bool(manifest["unsupported"])) | {
            "mutants": results,
            "targets_without_mutants": [
                target
                for target in manifest["selected"]
                if not any(fnmatch.fnmatchcase(r["mutant"], mutant_pattern(target)) for r in results)
            ],
        }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--base", required=True)
    parser.add_argument("--debug-dir", type=Path, help="Retain a fresh runner snapshot for inspecting mutant diffs")
    parser.add_argument("--max-children", type=int, default=4)
    parser.add_argument("--report", type=Path, default=Path("mutation-report.json"))
    parser.add_argument("--select-only", action="store_true", help="Write selection manifest without claiming a score")
    args = parser.parse_args()
    args.report = args.report.resolve()
    args.report.parent.mkdir(parents=True, exist_ok=True)
    service = Path(__file__).resolve().parents[1]
    report = {"status": "ERROR", "score": None}
    try:
        manifest = selection_manifest(args.base, service.parents[1])
        report.update(manifest)
        if args.select_only:
            report["status"] = "SELECTED"
        elif not manifest["selected"]:
            report.update(evaluate([], bool(manifest["unsupported"])))
            report["reason"] = "No changed mutable functions; see deleted/excluded/unsupported manifest"
        else:
            print(
                f"Running {len(manifest['selected'])} changed functions; log: {args.report.with_suffix('.log')}",
                flush=True,
            )
            report.update(
                run_selected(manifest, service, args.max_children, args.report.with_suffix(".log"), args.debug_dir)
            )
    except (OSError, ValueError, RuntimeError, subprocess.SubprocessError) as exc:
        report.update(status="ERROR", error=str(exc))
    finally:
        args.report.write_text(json.dumps(report, indent=2) + "\n")
    print(f"Mutation: {report['status']}; score={report.get('score')}; report={args.report}")
    return 0 if report["status"] in {"PASS", "N/A", "SELECTED"} else 1


if __name__ == "__main__":
    raise SystemExit(main())
