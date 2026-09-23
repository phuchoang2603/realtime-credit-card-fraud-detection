"""Capture actual verification commands in an isolated Xvfb/xterm display.

Run inside devenv with Xvfb, xterm and ImageMagick available on PATH. Captions
are added below the unmodified terminal capture. This does not render saved
text into a pretend terminal. Each test command is really executed in xterm.
"""

from __future__ import annotations

import argparse
import os
import shlex
import subprocess
import sys
import tempfile
import time
from pathlib import Path


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--mutation-report", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--only", nargs="+", choices=["coverage", "api", "boundaries", "property", "mutation"])
    args = parser.parse_args()
    args.output.mkdir(parents=True, exist_ok=True)
    service = Path(__file__).resolve().parents[1]
    python = shlex.quote(sys.executable)
    revision = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=service, text=True).strip()
    cases = {
        "coverage": (
            f'{python} -m pytest -q -m "not property" --disable-warnings --cov=app --cov-report=term-missing --cov-report=json:/tmp/fraud-capture-coverage.json tests && {python} tools/check_coverage.py /tmp/fraud-capture-coverage.json',
            "Isolated unit + API coverage; property tests excluded. Raw ratio must exceed 90%.",
        ),
        "api": (
            f"{python} -m pytest -v --disable-warnings tests/test_grpc.py",
            "Actual API execution with fixture-managed fake/failing model dependencies.",
        ),
        "boundaries": (
            f"{python} -m pytest -v --disable-warnings tests/test_rules.py",
            "Named partitions exercise independent amount/anomaly boundaries and blocklist classes.",
        ),
        "property": (
            f"{python} -m pytest -q -s --disable-warnings tests/test_prediction_property.py",
            "Real model; generated repeatability and input preservation. Artifact digest/profile are retained in the report.",
        ),
        "mutation": (
            f"{python} tools/show_mutation_report.py {shlex.quote(str(args.mutation_report.resolve()))}",
            "Live inspection of the recorded full mutation run; this command does not rerun mutants.",
        ),
    }
    with tempfile.TemporaryDirectory(prefix="fraud-evidence-") as temporary:
        temp = Path(temporary)
        read_fd, write_fd = os.pipe()
        server = subprocess.Popen(
            ["Xvfb", "-displayfd", str(write_fd), "-screen", "0", "2400x1000x24", "-nolisten", "tcp"],
            pass_fds=(write_fd,),
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
        )
        os.close(write_fd)
        try:
            with os.fdopen(read_fd) as display_pipe:
                display = display_pipe.readline().strip()
            if not display:
                raise RuntimeError("Xvfb failed to create an isolated display")
            environment = os.environ | {
                "DISPLAY": f":{display}",
                "TESTING_MODE": "true",
                "HYPOTHESIS_PROFILE": "ci",
                "PYTHONUNBUFFERED": "1",
            }
            for name in args.only or cases:
                command, caption = cases[name]
                marker = temp / f"{name}.done"
                script = temp / f"{name}.sh"
                script.write_text(
                    "cd "
                    + shlex.quote(str(service))
                    + "\n"
                    + "printf '%s\\n' "
                    + shlex.quote("Verification: " + name + " | HEAD " + revision[:12] + " + working tree")
                    + "\n"
                    + "printf '%s\\n' "
                    + shlex.quote("$ " + command)
                    + "\n"
                    + command
                    + '\nresult=$?\nprintf "%s" "$result" > '
                    + shlex.quote(str(marker))
                    + "\n"
                )
                terminal = subprocess.Popen(
                    [
                        "xterm",
                        "-hold",
                        "-fa",
                        "DejaVu Sans Mono",
                        "-fs",
                        "11",
                        "-geometry",
                        "155x51+0+0",
                        "-bg",
                        "#101820",
                        "-fg",
                        "#ecf2f8",
                        "-e",
                        "bash",
                        str(script),
                    ],
                    env=environment,
                    stdout=subprocess.DEVNULL,
                    stderr=subprocess.DEVNULL,
                )
                try:
                    deadline = time.monotonic() + 120
                    while not marker.exists():
                        if terminal.poll() is not None or time.monotonic() > deadline:
                            raise RuntimeError(f"{name}: terminal exited or command exceeded capture budget")
                        time.sleep(0.2)
                    if marker.read_text() != "0":
                        raise RuntimeError(f"{name}: verification command failed")
                    time.sleep(0.3)  # Let the X server finish painting the final output.
                    raw = args.output / f"{name}-terminal.png"
                    subprocess.run(
                        ["magick", "import", "-display", f":{display}", "-window", "root", "-silent", str(raw)],
                        check=True,
                    )
                    subprocess.run(
                        [
                            "magick",
                            str(raw),
                            "-gravity",
                            "South",
                            "-background",
                            "#edf4fb",
                            "-fill",
                            "#17263b",
                            "-splice",
                            "0x70",
                            "-font",
                            "DejaVu-Sans",
                            "-pointsize",
                            "20",
                            "-annotate",
                            "+0+22",
                            caption,
                            str(args.output / f"{name}-annotated.png"),
                        ],
                        check=True,
                    )
                    print(f"Captured {name}: {args.output / name}", flush=True)
                finally:
                    terminal.terminate()
                    terminal.wait(timeout=5)
        finally:
            server.terminate()
            server.wait(timeout=5)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
