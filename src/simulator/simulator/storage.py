import json
from collections import Counter, defaultdict
from dataclasses import dataclass
from datetime import timedelta
from hashlib import sha256
from pathlib import Path
from tempfile import TemporaryDirectory

import duckdb
import numpy as np
import pyarrow as pa
import pyarrow.fs as fs
import pyarrow.parquet as pq
import yaml
from datacontract.data_contract import DataContract

from simulator.arrow import LABEL_SCHEMA, event_schema
from simulator.config import HistoricalConfig, StorageLayout
from simulator.generation import Record, build_history
from simulator.profiles import DUPLICATE_STAGE, HISTORY_STAGE, build_profiles, schedule_for_day, stage_rng
from simulator.statistics import DatasetStatistics

PRODUCT_DIR = Path(__file__).resolve().parents[3] / "contracts" / "data-products" / "historical-payments"
COMPRESSION = "zstd"
COMPRESSION_LEVEL = 3


class ContractViolation(ValueError):
    pass


@dataclass(frozen=True)
class Table:
    name: str
    partition: str
    time_column: str
    sort_columns: tuple[str, ...]
    contract: str


EVENTS = Table(
    "payment_events",
    "occurred_date",
    "occurred_at",
    ("occurred_at", "payment_id", "aggregate_version", "event_id"),
    "payment-events",
)
LABELS = Table(
    "fraud_labels",
    "label_available_date",
    "label_available_at",
    ("label_available_at", "payment_id", "attempt_id", "source"),
    "fraud-labels",
)


class _Staging:
    def __init__(self, root: Path):
        self.root = root
        self.rows: Counter = Counter()

    def write(self, table: Table, records: list[Record], name: str) -> None:
        by_day: dict[str, list[Record]] = defaultdict(list)
        for record in records:
            by_day[record[table.time_column].date().isoformat()].append(record)
        for day, rows in sorted(by_day.items()):
            schema = event_schema(rows[0]["schema_version"]) if table is EVENTS else LABEL_SCHEMA
            directory = self.root / table.name / f"{table.partition}={day}"
            directory.mkdir(parents=True, exist_ok=True)
            pq.write_table(
                pa.Table.from_pylist(rows, schema=schema), directory / f"{name}.parquet", compression="zstd"
            )
            self.rows[table.name, day] += len(rows)


def _duplicates(events: tuple[Record, ...], rate: float, random: np.random.Generator) -> list[Record]:
    if not rate:
        return []
    copies = random.poisson(rate / (1 - rate), len(events))
    return [event for event, count in zip(events, copies) for _ in range(count)]


def _compact(table: Table, staging: Path, export: Path, layout: StorageLayout, spill: Path) -> dict:
    files = row_groups = size = 0
    with duckdb.connect(
        config={
            "memory_limit": f"{layout.sort_memory_limit_mb}MiB",
            "temp_directory": str(spill),
            "threads": 1,
        }
    ) as connection:
        for directory in sorted((staging / table.name).iterdir()):
            sources = sorted(str(path) for path in directory.glob("*.parquet"))
            output = export / table.name / directory.name
            output.parent.mkdir(parents=True, exist_ok=True)
            connection.execute(
                f"COPY (SELECT * FROM read_parquet($sources, hive_partitioning = false) "
                f"ORDER BY {', '.join(table.sort_columns)}) TO '{output}' "
                f"(FORMAT parquet, COMPRESSION {COMPRESSION}, COMPRESSION_LEVEL {COMPRESSION_LEVEL}, "
                f"ROW_GROUP_SIZE {layout.row_group_rows}, FILE_SIZE_BYTES {layout.target_file_mb * 1024 * 1024}, "
                "FILENAME_PATTERN 'part-{i}')",
                {"sources": sources},
            )
            for path in directory.glob("*.parquet"):
                path.unlink()
            directory.rmdir()
            for path in output.glob("*.parquet"):
                files += 1
                row_groups += pq.ParquetFile(path).metadata.num_row_groups
                size += path.stat().st_size
    return {"files": files, "row_groups": row_groups, "bytes": size, "sort_columns": list(table.sort_columns)}


def _validate(export: Path, temporary: Path) -> dict:
    database = temporary / "validation.duckdb"
    with duckdb.connect(str(database)) as connection:
        for table, schema in ((EVENTS, event_schema(2)), (LABELS, LABEL_SCHEMA)):
            empty = temporary / f"{table.name}-schema.parquet"
            pq.write_table(schema.empty_table(), empty)
            paths = [str(empty), *sorted(str(path) for path in (export / table.name).rglob("*.parquet"))]
            connection.read_parquet(paths, union_by_name=True, hive_partitioning=False).create_view(table.name)
    reports = export / "validation"
    reports.mkdir()
    results = {}
    failures = []
    for table in (EVENTS, LABELS):
        run = DataContract(
            data_contract_file=str(PRODUCT_DIR / f"{table.contract}.odcs.yaml"),
            server="validation",
            config={"DATACONTRACT_DUCKDB_DATABASE": str(database), "DATACONTRACT_DUCKDB_SCHEMA": "main"},
        ).test()
        (reports / f"{table.contract}.json").write_text(run.model_dump_json(indent=2))
        results[table.contract] = run.result
        failures += [
            f"{table.contract}/{check.name}: {check.reason or check.result}"
            for check in run.checks
            if check.result not in ("passed", "info")
        ]
        if not run.has_passed() and not failures:
            failures.append(f"{table.contract}: {run.result}")
    if failures:
        raise ContractViolation("contract validation failed: " + "; ".join(failures))
    return results


def _descriptor() -> dict:
    product = yaml.safe_load((PRODUCT_DIR / "product.odps.yaml").read_text())
    contracts = {}
    for path in sorted(PRODUCT_DIR.glob("*.odcs.yaml")):
        content = path.read_bytes()
        contract = yaml.safe_load(content)
        contracts[path.name.removesuffix(".odcs.yaml")] = {
            "id": contract["id"],
            "version": contract["version"],
            "sha256": sha256(content).hexdigest(),
        }
    return {"id": product["id"], "version": product["version"], "contracts": contracts}


def _generator_revision() -> str:
    digest = sha256()
    for path in sorted(Path(__file__).parent.glob("*.py")):
        digest.update(path.name.encode())
        digest.update(path.read_bytes())
    return digest.hexdigest()


def _require_absent(filesystem: fs.FileSystem, prefix: str) -> None:
    if filesystem.get_file_info(f"{prefix}/manifest.json").type != fs.FileType.NotFound:
        raise FileExistsError(f"dataset version already published: {prefix}/manifest.json")
    if filesystem.get_file_info(fs.FileSelector(prefix, allow_not_found=True)):
        raise FileExistsError(f"incomplete dataset version needs operator cleanup: {prefix}")


def _upload(filesystem: fs.FileSystem, key: str, payload: bytes) -> None:
    if isinstance(filesystem, fs.LocalFileSystem):
        filesystem.create_dir(key.rsplit("/", 1)[0], recursive=True)
    with filesystem.open_output_stream(key) as stream:
        stream.write(payload)


def generate(config: HistoricalConfig, filesystem: fs.FileSystem, root: str) -> dict:
    prefix = f"{root.rstrip('/')}/historical/{config.dataset_version}"
    _require_absent(filesystem, prefix)
    profiles = build_profiles(config)
    stats = DatasetStatistics()
    duplicate_rate = config.data_problems.duplicate_rate
    with TemporaryDirectory(prefix="historical-") as directory:
        temporary = Path(directory)
        staging = _Staging(temporary / "staging")
        for offset in range(config.duration_days):
            day = config.start_date + timedelta(days=offset)
            schedule = schedule_for_day(config, day, profiles)
            for chunk, start in enumerate(range(0, len(schedule), config.chunk_purchases)):
                random = stage_rng(config.seed, HISTORY_STAGE, offset, chunk)
                duplicate_random = stage_rng(config.seed, DUPLICATE_STAGE, offset, chunk)
                events: list[Record] = []
                labels: list[Record] = []
                for index in range(start, min(start + config.chunk_purchases, len(schedule))):
                    history = build_history(config, profiles, schedule, index, day, index, random)
                    copies = _duplicates(history.events, duplicate_rate, duplicate_random)
                    stats.observe(history, len(copies))
                    events += [*history.events, *copies]
                    labels += history.labels
                name = f"chunk-{offset:04d}-{chunk:04d}"
                staging.write(EVENTS, events, name)
                staging.write(LABELS, labels, name)

        export = temporary / "export"
        storage = {
            table.name: _compact(table, staging.root, export, config.storage, temporary / "spill")
            for table in (EVENTS, LABELS)
        }
        validation = _validate(export, temporary)
        configuration = config.model_dump(mode="json")
        manifest = {
            "dataset_version": config.dataset_version,
            "seed": config.seed,
            "configuration": configuration,
            "config_sha256": sha256(json.dumps(configuration, sort_keys=True).encode()).hexdigest(),
            "generator_revision": _generator_revision(),
            "data_product": _descriptor(),
            "schema_versions": {EVENTS.name: [1, 2], LABELS.name: [1]},
            "profile_sha256": profiles.digest(),
            "synthetic_key_version": config.synthetic_key_version,
            "history_end": stats.history_end.isoformat() if stats.history_end else None,
            "partitions": {
                table.name: {day: rows for (name, day), rows in sorted(staging.rows.items()) if name == table.name}
                for table in (EVENTS, LABELS)
            },
            "storage": {
                "format": "parquet",
                "compression": COMPRESSION,
                "compression_level": COMPRESSION_LEVEL,
                "row_group_rows": config.storage.row_group_rows,
                "target_file_mb": config.storage.target_file_mb,
                **storage,
            },
            "validation": validation,
            "configured": {
                "duplicate_rate": duplicate_rate,
                "schema_cutover_date": config.data_problems.schema_cutover_date.isoformat(),
                "merchant_zipf_exponent": config.behavior.merchant_zipf_exponent,
                "category_zipf_exponent": config.behavior.category_zipf_exponent,
                "cardinality": config.counts.model_dump(),
                "scenario_rates": {name: settings.rate for name, settings in config.scenarios},
            },
            "observed": stats.observed(),
        }
        _require_absent(filesystem, prefix)
        for path in sorted(item for item in export.rglob("*") if item.is_file()):
            _upload(filesystem, f"{prefix}/{path.relative_to(export).as_posix()}", path.read_bytes())
        _upload(filesystem, f"{prefix}/manifest.json", json.dumps(manifest, indent=2, sort_keys=True).encode())
    return manifest
