from collections import Counter
from dataclasses import asdict, dataclass, field
from datetime import UTC, datetime
from pathlib import Path
from tempfile import TemporaryDirectory
from uuid import uuid4

import duckdb
import pyarrow as pa
import pyarrow.parquet as pq
from datacontract.data_contract import DataContract

from payment_data_contracts.contract import Contract, load_contract


@dataclass
class ValidationReport:
    contract: dict
    run_id: str = field(default_factory=lambda: str(uuid4()))
    generated_at: str = field(default_factory=lambda: datetime.now(UTC).isoformat())
    checks: list[dict] = field(default_factory=list)

    @property
    def passed(self) -> bool:
        return bool(self.checks) and all(check["result"] in ("passed", "info") for check in self.checks)

    def add(self, identifier: str, result: str, reason: str, **details) -> None:
        self.checks.append({"id": identifier, "result": result, "reason": reason, **details})

    def to_dict(self) -> dict:
        return {"report_version": 1, "status": "passed" if self.passed else "failed", **asdict(self)}


def declared_rules(contract: Contract) -> list[str]:
    def visit(element):
        for rule in element.quality or []:
            if not rule.id:
                raise ValueError("every quality rule must have a stable id")
            yield rule.id
        for prop in element.properties or []:
            yield from visit(prop)
        if getattr(element, "items", None) is not None:
            yield from visit(element.items)

    ids = [identifier for dataset in contract.model.schema_ or [] for identifier in visit(dataset)]
    if len(ids) != len(set(ids)):
        raise ValueError("quality rule ids must be unique across the contract")
    return ids


def lint_contract(contract: Contract) -> ValidationReport:
    report = ValidationReport(contract.provenance)
    try:
        run = DataContract(data_contract_str=contract.source).lint()
        report.checks.extend(check.model_dump(mode="json", exclude_none=True) for check in run.checks)
        declared_rules(contract)
        names = [dataset.name for dataset in contract.model.schema_ or []]
        if not names or len(names) != len(set(names)):
            raise ValueError("dataset names must be nonempty and unique")
        server = next((server for server in contract.model.servers or [] if server.server == "validation"), None)
        if server is None or server.type != "duckdb":
            raise ValueError("a DuckDB validation server is required")
        for dataset in contract.model.schema_ or []:
            if not dataset.name.isidentifier():
                raise ValueError(f"dataset name is not an identifier: {dataset.name}")
            for version in contract.versions(dataset.name):
                contract.arrow_schema(dataset.name, version)
    except Exception as exc:
        report.add("contract.support", "error", str(exc))
    return report


def _inspect_file(path: Path, name: str, contract: Contract) -> None:
    with pq.ParquetFile(path) as parquet:
        versions = set(contract.versions(name))
        if "schema_version" in parquet.schema_arrow.names:
            observed = set()
            for batch in parquet.iter_batches(columns=["schema_version"], batch_size=65536):
                observed.update(batch.column(0).to_pylist())
                if len(observed) > 1 or not observed.issubset(versions):
                    raise ValueError(f"file must contain one supported schema_version: {observed}")
            if observed:
                versions = observed
        if not any(
            parquet.schema_arrow.equals(contract.arrow_schema(name, version), check_metadata=False)
            for version in versions
        ):
            raise ValueError(f"physical Arrow schema differs from declared {name} versions {sorted(versions)}")


def validate_dataset(root: Path, contract: Contract | None = None) -> ValidationReport:
    contract = load_contract() if contract is None else contract
    report = lint_contract(contract)
    if not report.passed:
        return report
    sources: dict[str, list[Path]] = {}
    for dataset in contract.model.schema_ or []:
        name = dataset.name
        directory = root / name
        if not directory.is_dir():
            report.add(f"{name}.directory", "error", f"missing dataset directory: {directory}")
            continue
        entries = sorted(path for path in directory.rglob("*") if path.is_file())
        unexpected = [str(path) for path in entries if path.suffix != ".parquet"]
        if unexpected:
            report.add(
                f"{name}.source_files", "error", "dataset directory contains non-Parquet files", files=unexpected
            )
            continue
        sources[name] = [path.resolve() for path in entries]
        for path in sources[name]:
            try:
                _inspect_file(path, name, contract)
                report.add(f"{name}.physical_schema", "passed", "exact physical schema matches", file=str(path))
            except Exception as exc:
                report.add(f"{name}.physical_schema", "error", str(exc), file=str(path))
    if not report.passed:
        return report
    try:
        with TemporaryDirectory(prefix="payment-contract-") as temporary:
            staging = Path(temporary)
            database = staging / "validation.duckdb"
            with duckdb.connect(str(database)) as connection:
                for name, paths in sources.items():
                    empty = staging / f"{name}.parquet"
                    pq.write_table(pa.Table.from_pylist([], schema=contract.arrow_schema(name)), empty)
                    connection.read_parquet(
                        [str(empty), *(str(path) for path in paths)], union_by_name=True, hive_partitioning=False
                    ).create_view(name)
            model = contract.model.model_copy(deep=True)
            server = next(server for server in model.servers if server.server == "validation")
            server.database = str(database)
            run = DataContract(
                data_contract=model,
                server="validation",
                config={"DATACONTRACT_DUCKDB_DATABASE": str(database), "DATACONTRACT_DUCKDB_SCHEMA": "main"},
            ).test()
            report.checks.extend(check.model_dump(mode="json", exclude_none=True) for check in run.checks)
            executed = Counter(
                check.qualityId
                for check in run.checks
                if check.qualityId and check.result in ("passed", "failed", "error")
            )
            for identifier in declared_rules(contract):
                if executed[identifier] != 1:
                    report.add(
                        identifier, "error", f"expected one executed quality check, observed {executed[identifier]}"
                    )
    except Exception as exc:
        report.add("validation.execution", "error", str(exc))
    return report
