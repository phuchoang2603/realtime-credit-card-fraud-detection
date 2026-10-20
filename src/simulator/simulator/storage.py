import json
from collections import Counter, defaultdict
from datetime import timedelta
from hashlib import sha256
from pathlib import Path
from tempfile import TemporaryDirectory

import pyarrow as pa
import pyarrow.fs as fs
import pyarrow.parquet as pq
from google.protobuf.message import Message

from labels.v1.labels_pb2 import FraudLabel
from payments.v1.events_pb2 import PaymentEvent
from simulator.arrow import arrow_schema, message_row
from simulator.config import HistoricalConfig
from simulator.generation import build_history
from simulator.profiles import build_profiles, schedule_for_day, stage_rng
from simulator.statistics import DatasetStatistics

BATCH_SIZE = 4096


def _event_schemas() -> dict[int, pa.Schema]:
    schema = arrow_schema(PaymentEvent.DESCRIPTOR)
    attempt_field = schema.field("attempt_started")
    signals_field = attempt_field.type.field("signals")
    signals = pa.struct([field for field in signals_field.type if field.name != "user_agent_family"])
    attempt = pa.struct(
        [field.with_type(signals) if field.name == "signals" else field for field in attempt_field.type]
    )
    old_schema = pa.schema(
        [field.with_type(attempt) if field.name == "attempt_started" else field for field in schema]
    )
    return {1: old_schema, 2: schema}


class _PartitionWriter:
    def __init__(self, root: Path, partition: str, schemas: dict[int, pa.Schema]):
        self.root = root
        self.partition = partition
        self.schemas = schemas
        self.counts: Counter = Counter()
        self.parts: Counter = Counter()
        self.pending: dict[tuple[str, int], list[dict]] = defaultdict(list)
        self.pending_count = 0

    def append(self, message: Message, day: str, version: int = 1) -> None:
        self.pending[day, version].append(message_row(message))
        self.pending_count += 1
        if self.pending_count >= BATCH_SIZE:
            self.flush()

    def write_table(self, day: str, table: pa.Table) -> None:
        directory = self.root / f"{self.partition}={day}"
        directory.mkdir(parents=True, exist_ok=True)
        pq.write_table(table, directory / f"part-{self.parts[day]:05d}.parquet")
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
) -> dict:
    configuration = config.model_dump(mode="json")
    canonical_config = json.dumps(configuration, sort_keys=True, separators=(",", ":"))
    return {
        "dataset_version": config.dataset_version,
        "seed": config.seed,
        "configuration": configuration,
        "config_sha256": sha256(canonical_config.encode()).hexdigest(),
        "generator_revision": _generator_revision(),
        "contract_versions": {"payments": "payments.v1", "labels": "labels.v1"},
        "profile_sha256": profile_digest,
        "synthetic_key_version": config.synthetic_key_version,
        "history_end": stats.history_end.isoformat() if stats.history_end is not None else None,
        "output": output,
        "partitions": partitions,
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
    profiles = build_profiles(config)
    stats = DatasetStatistics()
    with TemporaryDirectory(prefix="historical-") as temporary:
        staging = Path(temporary)
        events = _PartitionWriter(staging / "payment_events", "occurred_date", _event_schemas())
        labels = _PartitionWriter(
            staging / "fraud_labels", "label_available_date", {1: arrow_schema(FraudLabel.DESCRIPTOR)}
        )
        for offset in range(config.duration_days):
            day = config.start_date + timedelta(days=offset)
            random = stage_rng(config.seed, 4, offset)
            for ordinal, purchase in enumerate(schedule_for_day(config, day, profiles)):
                history = build_history(config, profiles, purchase, day, ordinal, random)
                stats.observe(history)
                for event in history.events:
                    events.append(event, event.occurred_at.ToDatetime().date().isoformat(), event.schema_version)
                for label in history.labels:
                    labels.append(label, label.label_available_at.ToDatetime().date().isoformat())
        events.flush()
        labels.flush()
        duplicates = _write_duplicates(events, config.data_problems.duplicate_rate, config.seed)
        partitions = {
            "payment_events": dict(sorted(events.counts.items())),
            "fraud_labels": dict(sorted(labels.counts.items())),
        }
        manifest = _manifest(config, output, profiles.digest(), partitions, stats, duplicates)
        _require_empty(filesystem, prefix)
        filesystem.create_dir(prefix, recursive=True)
        fs.copy_files(
            str(staging),
            prefix,
            source_filesystem=fs.LocalFileSystem(),
            destination_filesystem=filesystem,
            use_threads=False,
        )
        payload = json.dumps(manifest, sort_keys=True, separators=(",", ":"), ensure_ascii=False).encode()
        with filesystem.open_output_stream(f"{prefix}/manifest.json") as output_stream:
            output_stream.write(payload)
    return manifest
