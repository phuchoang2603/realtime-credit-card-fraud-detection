import json
from collections import Counter, defaultdict
from contextlib import ExitStack
from datetime import timedelta
from hashlib import sha256
from pathlib import Path
from tempfile import TemporaryDirectory

import duckdb
import pyarrow as pa
import pyarrow.fs as fs
import pyarrow.parquet as pq
from datacontract.data_contract import DataContract

from simulator.arrow import LABEL_SCHEMA, Record, event_schema
from simulator.config import HistoricalConfig, StorageLayout
from simulator.generation import build_history
from simulator.profiles import build_profiles, schedule_for_day, stage_rng
from simulator.statistics import DatasetStatistics

BATCH_SIZE = 4096
COMPRESSION = "zstd"
COMPRESSION_LEVEL = 3
EVENT_SORT_COLUMNS = ("occurred_at", "payment_id", "aggregate_version", "event_id")
LABEL_SORT_COLUMNS = ("label_available_at", "payment_id", "attempt_id", "source")


class _PartitionWriter:
    def __init__(self, root: Path, partition: str, schemas: dict[int, pa.Schema]):
        self.root = root
        self.partition = partition
        self.schemas = schemas
        self.counts: Counter = Counter()
        self.parts: Counter = Counter()
        self.pending: dict[tuple[str, int], list[dict]] = defaultdict(list)
        self.pending_count = 0

    def append(self, record: Record, day: str, version: int = 1) -> None:
        self.pending[day, version].append(record)
        self.pending_count += 1
        if self.pending_count >= BATCH_SIZE:
            self.flush()

    def write_table(self, day: str, table: pa.Table) -> None:
        directory = self.root / f"{self.partition}={day}"
        directory.mkdir(parents=True, exist_ok=True)
        pq.write_table(
            table,
            directory / f"part-{self.parts[day]:05d}.parquet",
            compression=COMPRESSION,
            compression_level=COMPRESSION_LEVEL,
        )
        self.parts[day] += 1
        self.counts[day] += table.num_rows

    def flush(self) -> None:
        for (day, version), rows in sorted(self.pending.items()):
            self.write_table(day, pa.Table.from_pylist(rows, schema=self.schemas[version]))
        self.pending.clear()
        self.pending_count = 0


def _write_duplicates(writer: _PartitionWriter, rate: float, seed: int) -> int:
    if not rate:
        return 0
    random = stage_rng(seed, 5)
    originals = sorted(writer.root.rglob("*.parquet"))
    unique_events = 0
    duplicate_events = 0
    for path in originals:
        day = path.parent.name.split("=", 1)[1]
        for batch in pq.ParquetFile(path).iter_batches(batch_size=BATCH_SIZE):
            unique_events += batch.num_rows
            target = round(unique_events * rate / (1 - rate))
            remaining = target - duplicate_events
            table = pa.Table.from_batches([batch])
            while remaining:
                size = min(remaining, BATCH_SIZE)
                indices = random.integers(0, batch.num_rows, size=size)
                writer.write_table(day, table.take(pa.array(indices)))
                remaining -= size
                duplicate_events += size
    return duplicate_events


def _compact_partitions(
    source: _PartitionWriter,
    destination: Path,
    layout: StorageLayout,
    sort_columns: tuple[str, ...],
    temporary: Path,
) -> dict:
    file_count = 0
    row_group_count = 0
    size_bytes = 0
    with duckdb.connect(
        config={
            "memory_limit": f"{layout.sort_memory_limit_mb}MiB",
            "temp_directory": str(temporary),
            "threads": 1,
            "preserve_insertion_order": False,
        }
    ) as connection:
        for day in sorted(source.counts):
            directory = source.root / f"{source.partition}={day}"
            paths = sorted(directory.glob("*.parquet"))
            schema = pq.ParquetFile(paths[0]).schema_arrow
            output_directory = destination / directory.name
            output_directory.mkdir(parents=True)
            relation = connection.read_parquet([str(path) for path in paths], hive_partitioning=False)
            reader = relation.order(", ".join(sort_columns)).to_arrow_reader(layout.row_group_rows)
            sorting = pq.SortingColumn.from_ordering(schema, [(column, "ascending") for column in sort_columns])
            part = 0
            writer = None
            with reader, ExitStack() as streams:
                for batch in reader:
                    if writer is None:
                        path = output_directory / f"part-{part:05d}.parquet"
                        output = streams.enter_context(pa.OSFile(str(path), "wb"))
                        writer = streams.enter_context(
                            pq.ParquetWriter(
                                output,
                                schema,
                                compression=COMPRESSION,
                                compression_level=COMPRESSION_LEVEL,
                                use_dictionary=True,
                                write_statistics=True,
                                sorting_columns=sorting,
                            )
                        )
                        part += 1
                    table = pa.Table.from_batches([batch]).cast(schema)
                    writer.write_table(table, row_group_size=layout.row_group_rows)
                    row_group_count += 1
                    if output.tell() >= layout.target_file_size_mb * 1024 * 1024:
                        streams.close()
                        writer = None
            file_count += part
            size_bytes += sum(path.stat().st_size for path in output_directory.glob("*.parquet"))
            for path in paths:
                path.unlink()
    return {
        "file_count": file_count,
        "row_group_count": row_group_count,
        "size_bytes": size_bytes,
        "sort_columns": list(sort_columns),
    }


def _generator_revision() -> str:
    source = Path(__file__).parent
    digest = sha256()
    for path in sorted(source.glob("*.py")):
        digest.update(path.name.encode())
        digest.update(path.read_bytes())
    return digest.hexdigest()


def _require_empty(filesystem: fs.FileSystem, prefix: str) -> None:
    if filesystem.get_file_info(f"{prefix}/manifest.json").type != fs.FileType.NotFound:
        raise FileExistsError(f"dataset version already complete: {prefix}/manifest.json")
    if filesystem.get_file_info(fs.FileSelector(prefix, allow_not_found=True)):
        raise FileExistsError(f"dataset version is incomplete; operator cleanup required: {prefix}")


def _manifest(
    config: HistoricalConfig,
    output: str,
    profile_digest: str,
    partitions: dict,
    stats: DatasetStatistics,
    duplicates: int,
    storage: dict,
    contract: dict,
) -> dict:
    configuration = config.model_dump(mode="json")
    canonical_config = json.dumps(configuration, sort_keys=True, separators=(",", ":"))
    return {
        "dataset_version": config.dataset_version,
        "seed": config.seed,
        "configuration": configuration,
        "config_sha256": sha256(canonical_config.encode()).hexdigest(),
        "generator_revision": _generator_revision(),
        "contract": contract,
        "validation_report": "validation.json",
        "schema_versions": {"payment_events": [1, 2], "fraud_labels": [1]},
        "profile_sha256": profile_digest,
        "synthetic_key_version": config.synthetic_key_version,
        "history_end": stats.history_end.isoformat() if stats.history_end is not None else None,
        "output": output,
        "partitions": partitions,
        "storage": {
            "format": "parquet",
            "compression": COMPRESSION,
            "compression_level": COMPRESSION_LEVEL,
            "dictionary_encoding": True,
            "write_statistics": True,
            **storage,
        },
        "configured": {
            "duplicate_rate": config.data_problems.duplicate_rate,
            "schema_cutover": config.data_problems.schema_evolution.cutover_date.isoformat(),
            "merchant_zipf_exponent": config.behavior.merchant_zipf_exponent,
            "category_zipf_exponent": config.behavior.category_zipf_exponent,
            "cardinality": config.counts.model_dump(),
            "scenario_rates": {name: settings.rate for name, settings in config.scenarios},
        },
        "observed": stats.observed(duplicates),
    }


def generate(config: HistoricalConfig, output: str, filesystem: fs.FileSystem, root: str) -> dict:
    prefix = f"{root.rstrip('/')}/historical/{config.dataset_version}"
    _require_empty(filesystem, prefix)
    contract_source = (
        Path(__file__).resolve().parents[3] / "contracts/data/historical-payments.odcs.yaml"
    ).read_text()
    profiles = build_profiles(config)
    stats = DatasetStatistics()
    with TemporaryDirectory(prefix="historical-") as temporary:
        staging = Path(temporary) / "raw"
        export = Path(temporary) / "export"
        export.mkdir()
        events = _PartitionWriter(
            staging / "payment_events", "occurred_date", {version: event_schema(version) for version in (1, 2)}
        )
        labels = _PartitionWriter(staging / "fraud_labels", "label_available_date", {1: LABEL_SCHEMA})
        for offset in range(config.duration_days):
            day = config.start_date + timedelta(days=offset)
            random = stage_rng(config.seed, 4, offset)
            for ordinal, purchase in enumerate(schedule_for_day(config, day, profiles)):
                history = build_history(config, profiles, purchase, day, ordinal, random)
                stats.observe(history)
                for event in history.events:
                    events.append(event, event["occurred_at"].date().isoformat(), event["schema_version"])
                for label in history.labels:
                    labels.append(label, label["label_available_at"].date().isoformat())
        events.flush()
        labels.flush()
        duplicates = _write_duplicates(events, config.data_problems.duplicate_rate, config.seed)
        partitions = {
            "payment_events": dict(sorted(events.counts.items())),
            "fraud_labels": dict(sorted(labels.counts.items())),
        }
        storage = {
            "payment_events": _compact_partitions(
                events, export / "payment_events", config.storage, EVENT_SORT_COLUMNS, Path(temporary) / "sort"
            ),
            "fraud_labels": _compact_partitions(
                labels, export / "fraud_labels", config.storage, LABEL_SORT_COLUMNS, Path(temporary) / "sort"
            ),
        }
        database = Path(temporary) / "validation.duckdb"
        for name in ("payment_events", "fraud_labels"):
            (export / name).mkdir(exist_ok=True)
        with duckdb.connect(str(database)) as connection:
            for name, schema in (("payment_events", event_schema(2)), ("fraud_labels", LABEL_SCHEMA)):
                empty = Path(temporary) / f"{name}.parquet"
                pq.write_table(pa.Table.from_pylist([], schema=schema), empty)
                paths = sorted((export / name).rglob("*.parquet"))
                connection.read_parquet(
                    [str(empty), *(str(path) for path in paths)], union_by_name=True, hive_partitioning=False
                ).create_view(name)
        contract = DataContract(
            data_contract_str=contract_source,
            server="validation",
            config={"DATACONTRACT_DUCKDB_DATABASE": str(database), "DATACONTRACT_DUCKDB_SCHEMA": "main"},
        )
        run = contract.test()
        if not run.has_passed():
            failures = "; ".join(
                f"{check.qualityId or check.name}: {check.reason or check.result}"
                for check in run.checks
                if check.result not in ("passed", "info")
            )
            raise ValueError(f"ODCS validation failed: {failures}")
        (export / "validation.json").write_text(run.model_dump_json(indent=2))
        definition = contract.get_data_contract()
        manifest = _manifest(
            config,
            output,
            profiles.digest(),
            partitions,
            stats,
            duplicates,
            storage,
            {
                "id": definition.id,
                "version": definition.version,
                "sha256": sha256(contract_source.encode()).hexdigest(),
            },
        )
        _require_empty(filesystem, prefix)
        filesystem.create_dir(prefix, recursive=True)
        fs.copy_files(
            str(export),
            prefix,
            source_filesystem=fs.LocalFileSystem(),
            destination_filesystem=filesystem,
            use_threads=False,
        )
        payload = json.dumps(manifest, sort_keys=True, separators=(",", ":"), ensure_ascii=False).encode()
        with filesystem.open_output_stream(f"{prefix}/manifest.json") as output_stream:
            output_stream.write(payload)
    return manifest
