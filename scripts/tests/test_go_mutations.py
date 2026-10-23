import json
import os
import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path

SCRIPTS = Path(__file__).resolve().parents[1]


class GoMutationScopeTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        (self.root / "scripts").mkdir()
        shutil.copy(SCRIPTS / "check-go-mutations.sh", self.root / "scripts")
        self.module = self.root / "src/example"
        self.module.mkdir(parents=True)
        (self.module / "go.mod").write_text("module example\n\ngo 1.26.0\n")
        (self.module / "go.sum").touch()
        self.write("internal/domain/rule.go", "package domain\n")
        self.git("init", "-q")
        self.git("add", ".")
        self.git(
            "-c",
            "user.name=CI",
            "-c",
            "user.email=ci@example.com",
            "commit",
            "-qm",
            "base",
        )
        self.base = self.git("rev-parse", "HEAD").strip()
        tools = self.root / "bin"
        tools.mkdir()
        for name, body in {
            "go": "exit 0\n",
            "gremlins": 'printf "%s\\n" "$@" > "$MUTATION_ARGS"\n',
        }.items():
            path = tools / name
            path.write_text("#!/usr/bin/env bash\n" + body)
            path.chmod(0o755)
        self.env = {
            **os.environ,
            "PATH": f"{tools}:{os.environ['PATH']}",
            "CI_TEST_REPORTS": str(self.root / "reports"),
            "MUTATION_BASE": self.base,
            "MUTATION_ARGS": str(self.root / "args"),
            "GITHUB_STEP_SUMMARY": str(self.root / "summary"),
        }

    def git(self, *args):
        return subprocess.check_output(["git", *args], cwd=self.root, text=True)

    def write(self, name, content):
        path = self.module / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(content)

    def run_mutations(self, mode="diff"):
        return subprocess.check_output(
            ["bash", "scripts/check-go-mutations.sh", "src/example", mode],
            cwd=self.root,
            env=self.env,
            text=True,
        )

    def test_unchanged_module_does_not_fall_back_to_full_mutation(self):
        self.assertIn("Skipped", self.run_mutations())
        self.assertFalse((self.root / "args").exists())

    def test_tests_generated_transport_and_bootstrap_changes_are_skipped(self):
        for name in (
            "internal/domain/rule_test.go",
            "internal/adapters/postgres/sqlcgen/queries.go",
            "internal/transport/http/server.go",
            "cmd/example/main.go",
        ):
            self.write(name, "package example\n")
        self.git("add", ".")
        self.assertIn("Skipped", self.run_mutations())
        self.assertFalse((self.root / "args").exists())

    def test_ledger_validator_remains_in_changed_code_scope(self):
        self.write("cmd/ledger-check/main.go", "package main\n")
        self.git("add", ".")
        self.run_mutations()
        args = (self.root / "args").read_text().splitlines()
        self.assertEqual(args[:3], ["unleash", "--diff", self.base])

    def test_full_run_includes_unchanged_logic_without_diff_flag(self):
        self.run_mutations("full")
        args = (self.root / "args").read_text().splitlines()
        self.assertNotIn("--diff", args)

    def test_matrix_discovers_modules_without_a_service_list(self):
        shutil.copy(SCRIPTS / "go-ci-matrix.py", self.root / "scripts")
        self.write("nested/go.mod", "module nested\n")
        self.git("add", ".")
        matrix = json.loads(
            subprocess.check_output(
                ["python3", "scripts/go-ci-matrix.py"], cwd=self.root, text=True
            )
        )
        self.assertEqual(
            {entry["project"] for entry in matrix["include"]},
            {"src/example", "src/example/nested"},
        )
