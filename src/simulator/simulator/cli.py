import argparse
import json
import os
from pathlib import Path
from urllib.parse import urlsplit

import pyarrow.fs as fs

from simulator.config import load_config
from simulator.storage import generate


def destination(output: str) -> tuple[fs.FileSystem, str]:
    if not output.startswith("s3://"):
        return fs.LocalFileSystem(), str(Path(output).expanduser().resolve())
    parsed = urlsplit(output)
    if not parsed.netloc or parsed.username or parsed.password or parsed.query or parsed.fragment:
        raise ValueError("S3 output must be a bucket and optional key prefix, without credentials or query")
    access_key = os.environ.get("AWS_ACCESS_KEY_ID")
    secret_key = os.environ.get("AWS_SECRET_ACCESS_KEY")
    if not access_key or not secret_key:
        raise ValueError("S3 output requires AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY in the environment")
    filesystem = fs.S3FileSystem(
        access_key=access_key,
        secret_key=secret_key,
        region=os.environ.get("AWS_DEFAULT_REGION", "us-east-1"),
        scheme="https",
        endpoint_override="minio.home.phuchoang.sbs",
        force_virtual_addressing=False,
        allow_bucket_creation=False,
        allow_bucket_deletion=False,
    )
    return filesystem, f"{parsed.netloc}{parsed.path}".rstrip("/")


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Generate immutable synthetic payment histories")
    parser.add_argument("--config", type=Path, required=True)
    parser.add_argument("--output", help="Override the config's s3:// bucket or local output path")
    args = parser.parse_args(argv)
    try:
        config = load_config(args.config)
        output = args.output or config.output
        filesystem, root = destination(output)
        manifest = generate(config, output, filesystem, root)
    except (ValueError, FileExistsError, OSError) as error:
        parser.exit(2, f"historical generation failed: {error}\n")
    print(json.dumps({"dataset_version": manifest["dataset_version"], "config_sha256": manifest["config_sha256"]}))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
