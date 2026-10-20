import copy
import json
import subprocess
import sys
from datetime import UTC, datetime, timedelta
from pathlib import Path

import pyarrow as pa
import pyarrow.parquet as pq
import pytest

from payment_data_contracts import load_contract
from payment_data_contracts.cli import documentation
from payment_data_contracts.validation import lint_contract, validate_dataset

CONTRACT = load_contract()
START = datetime(2026, 10, 1, tzinfo=UTC)


def history(sequence=("created", "started", "approved", "requested", "succeeded")):
    snapshot = {
        "external_order_id": "order-1",
        "total": {"minor_units": 1000, "currency": "USD"},
        "items": [
            {
                "product_id": "sku-1",
                "category": "home",
                "condition": "new",
                "quantity": 2,
                "unit_price": {"minor_units": 500, "currency": "USD"},
            }
        ],
        "merchant_category": "home",
        "merchant_account_created_at": START - timedelta(days=90),
        "customer_account_created_at": START - timedelta(days=30),
        "shipping": {
            "country": "US",
            "region": "NY",
            "postal_code": "10001",
            "address_fingerprint": "synthetic-address",
        },
    }
    events = []
    attempt = 0
    for ordinal, kind in enumerate(sequence, 1):
        timestamp = START + timedelta(minutes=ordinal)
        if kind == "started":
            attempt += 1
        attempt_id = f"attempt-{attempt}"
        payloads = {
            "created": (
                "payment_created",
                {
                    "merchant": {"merchant_id": "merchant-1", "external_seller_id": "seller-1"},
                    "customer": {"customer_id": "customer-1", "external_buyer_id": "buyer-1"},
                    "snapshot": copy.deepcopy(snapshot),
                },
            ),
            "started": (
                "attempt_started",
                {
                    "attempt_id": attempt_id,
                    "signals": {
                        "attempted_at": timestamp,
                        "payment_method": {"type": "card", "fingerprint": "synthetic-method"},
                    },
                },
            ),
            "requested": (
                "processor_operation_requested",
                {"attempt_id": attempt_id, "operation_key": f"op-{attempt}"},
            ),
            "unknown": ("processor_outcome_unknown", {"attempt_id": attempt_id}),
            "succeeded": (
                "payment_succeeded",
                {"attempt_id": attempt_id, "processor_reference": "synthetic-processor"},
            ),
            "expired": ("payment_expired", {"_present": True}),
        }
        for outcome in ("approved", "rejected"):
            payloads[outcome] = (
                "risk_decision_recorded",
                {
                    "attempt_id": attempt_id,
                    "assessment": {
                        "outcome": "RISK_OUTCOME_APPROVE" if outcome == "approved" else "RISK_OUTCOME_DECLINE",
                        "reason_codes": ["ALLOW" if outcome == "approved" else "DECLINE"],
                        "policy_version": "policy-1",
                    },
                },
            )
        for source in ("risk", "processor"):
            payloads[f"{source}_declined"] = (
                "attempt_declined",
                {"attempt_id": attempt_id, "source": f"DECLINE_SOURCE_{source.upper()}", "reason": "synthetic-decline"},
            )
        name, value = payloads[kind]
        events.append(
            {
                "event_id": f"event-{ordinal}",
                "payment_id": "payment-1",
                "aggregate_version": ordinal,
                "integration_id": "synthetic-integration",
                "occurred_at": timestamp,
                "recorded_at": timestamp,
                "schema_version": 2,
                "correlation_id": "correlation-1",
                "causation_id": f"event-{ordinal - 1}" if ordinal > 1 else "payment-1",
                name: value,
            }
        )
    return events


def label(**overrides):
    return {
        "integration_id": "synthetic-integration",
        "payment_id": "payment-1",
        "attempt_id": "attempt-1",
        "is_fraud": True,
        "occurred_at": START + timedelta(minutes=5),
        "label_available_at": START + timedelta(days=7),
        "source": "LABEL_SOURCE_CHARGEBACK",
        **overrides,
    }


def write_dataset(root: Path, events, labels=()):
    for name, records in (("payment_events", events), ("fraud_labels", labels)):
        directory = root / name
        directory.mkdir(parents=True)
        versions = CONTRACT.versions(name) if name == "payment_events" else (1,)
        for version in versions:
            rows = [row for row in records if name != "payment_events" or row["schema_version"] == version]
            if rows:
                pq.write_table(
                    pa.Table.from_pylist(rows, schema=CONTRACT.arrow_schema(name, version)),
                    directory / f"v{version}.parquet",
                )
    return root


def failures(report):
    return {
        check.get("qualityId") or check.get("id")
        for check in report.checks
        if check["result"] not in ("passed", "info")
    }


def test_contract_drives_nested_arrow_versions_and_rejects_unsupported_types():
    assert lint_contract(CONTRACT).passed
    old = CONTRACT.arrow_schema("payment_events", 1).field("attempt_started").type.field("signals").type
    current = CONTRACT.arrow_schema("payment_events", 2).field("attempt_started").type.field("signals").type
    assert "user_agent_family" not in [field.name for field in old]
    assert current.field("user_agent_family").type == pa.string()
    assert CONTRACT.arrow_schema("payment_events").field("aggregate_version").type == pa.int64()
    with pytest.raises(ValueError, match="unsupported"):
        CONTRACT.arrow_schema("payment_events", 3)
    changed = copy.deepcopy(CONTRACT)
    changed.dataset("fraud_labels").properties[0].physicalType = "BINARY"
    assert not lint_contract(changed).passed


@pytest.mark.parametrize(
    ("before", "after"),
    [
        ("id: payment_delivery_consistency", "id: invalid.rule"),
        ("required: true", "requried: true"),
    ],
)
def test_raw_contract_is_linted_before_model_fields_can_be_ignored(tmp_path, before, after):
    path = tmp_path / "contract.odcs.yaml"
    path.write_text(CONTRACT.source.replace(before, after, 1))
    assert not lint_contract(load_contract(path)).passed


def test_duplicate_yaml_keys_are_rejected(tmp_path):
    path = tmp_path / "contract.odcs.yaml"
    path.write_text(CONTRACT.source.replace("version: 1.0.0", "version: 1.0.0\nversion: 2.0.0", 1))
    with pytest.raises(ValueError, match="duplicate YAML"):
        load_contract(path)


@pytest.mark.parametrize(
    "sequence",
    [
        ("created", "expired"),
        ("created", "started", "approved", "requested", "succeeded"),
        ("created", "started", "approved", "requested", "unknown", "succeeded"),
        ("created", "started", "approved", "requested", "unknown", "processor_declined", "expired"),
        ("created", "started", "rejected", "risk_declined", "started", "approved", "requested", "succeeded"),
    ],
)
def test_valid_lifecycles_and_identical_delivery_duplicates(tmp_path, sequence):
    events = history(sequence)
    events.append(copy.deepcopy(events[0]))
    report = validate_dataset(write_dataset(tmp_path, events))
    assert report.passed, report.to_dict()


@pytest.mark.parametrize(
    "sequence",
    [
        ("started",),
        ("created", "created", "expired"),
        ("created", "started", "started", "expired"),
        ("created", "approved", "expired"),
        ("created", "started", "rejected", "risk_declined", "requested", "succeeded"),
        ("created", "started", "approved", "requested", "unknown", "expired"),
        ("created", "started", "approved", "requested", "unknown", "started", "expired"),
        ("created", "started", "approved", "requested", "succeeded", "expired"),
        ("created", "started", "approved", "risk_declined", "expired"),
        ("created", "started", "approved", "requested"),
    ],
)
def test_illegal_lifecycles_fail_the_declared_rule(tmp_path, sequence):
    report = validate_dataset(write_dataset(tmp_path, history(sequence)))
    assert "payment_lifecycle" in failures(report), report.to_dict()


def test_cross_version_history_is_checked_as_one_history(tmp_path):
    events = history()
    for event in events[:2]:
        event["schema_version"] = 1
    report = validate_dataset(write_dataset(tmp_path, events, [label()]))
    assert report.passed, report.to_dict()
    assert report.contract == CONTRACT.provenance
    assert {check.get("qualityId") for check in report.checks if check.get("qualityId")} == {
        rule.id for dataset in CONTRACT.model.schema_ for rule in dataset.quality
    }


@pytest.mark.parametrize(
    ("change", "expected"),
    [
        ("conflicting_delivery", "payment_delivery_consistency"),
        ("version_gap", "payment_lifecycle"),
        ("wrong_attempt", "payment_lifecycle"),
        ("wrong_causation", "payment_lifecycle"),
        ("bad_total", "payment_snapshot_amounts"),
        ("empty_reasons", "payment_risk_assessment"),
    ],
)
def test_domain_corruption_has_a_traceable_quality_rule(tmp_path, change, expected):
    events = history()
    if change == "conflicting_delivery":
        events.append(copy.deepcopy(events[0]))
        events[-1]["correlation_id"] = "conflicting"
    elif change == "version_gap":
        events[-1]["aggregate_version"] += 1
    elif change == "wrong_attempt":
        events[-1]["payment_succeeded"]["attempt_id"] = "other-attempt"
    elif change == "wrong_causation":
        events[-1]["causation_id"] = "other-event"
    elif change == "bad_total":
        events[0]["payment_created"]["snapshot"]["total"]["minor_units"] += 1
    elif change == "empty_reasons":
        events[2]["risk_decision_recorded"]["assessment"]["reason_codes"] = []
    report = validate_dataset(write_dataset(tmp_path, events))
    assert expected in failures(report), report.to_dict()


@pytest.mark.parametrize(
    ("overrides", "expected"),
    [
        ({"attempt_id": "missing"}, "label_attempt_linkage"),
        ({"integration_id": "different-integration"}, "label_attempt_linkage"),
        ({"label_available_at": START}, "label_availability"),
    ],
)
def test_invalid_labels_fail_the_declared_rule(tmp_path, overrides, expected):
    report = validate_dataset(write_dataset(tmp_path, history(), [label(**overrides)]))
    assert expected in failures(report), report.to_dict()


def test_chargebacks_require_success_but_simulation_truth_can_label_declines(tmp_path):
    events = history(("created", "started", "rejected", "risk_declined", "expired"))
    write_dataset(tmp_path, events, [label(source="LABEL_SOURCE_SIMULATION_TRUTH", scenario="card_testing")])
    assert validate_dataset(tmp_path).passed
    pq.write_table(
        pa.Table.from_pylist([label()], schema=CONTRACT.arrow_schema("fraud_labels")),
        tmp_path / "fraud_labels/v1.parquet",
    )
    assert "label_attempt_linkage" in failures(validate_dataset(tmp_path))


def test_empty_datasets_are_valid_but_missing_datasets_are_not(tmp_path):
    write_dataset(tmp_path, [])
    assert validate_dataset(tmp_path).passed
    (tmp_path / "fraud_labels").rmdir()
    assert "fraud_labels.directory" in failures(validate_dataset(tmp_path))


def test_foreign_files_do_not_silently_become_empty_datasets(tmp_path):
    write_dataset(tmp_path, [])
    (tmp_path / "payment_events/events.csv").write_text("event_id\nevent-1\n")
    assert "payment_events.source_files" in failures(validate_dataset(tmp_path))


@pytest.mark.parametrize("change", ["integer_width", "extra_column", "wrong_version"])
def test_physical_schema_errors_are_not_hidden_by_normalization(tmp_path, change):
    write_dataset(tmp_path, history())
    path = tmp_path / "payment_events/v2.parquet"
    table = pq.read_table(path)
    if change == "integer_width":
        schema = table.schema.set(2, pa.field("aggregate_version", pa.int32(), nullable=False))
        table = table.cast(schema)
    elif change == "extra_column":
        table = table.append_column("raw_card_number", pa.array(["unexpected"] * table.num_rows))
    else:
        table = table.set_column(
            6, table.schema.field("schema_version"), pa.array([1] * table.num_rows, type=pa.uint32())
        )
    pq.write_table(table, path)
    assert "payment_events.physical_schema" in failures(validate_dataset(tmp_path))


def test_unexecuted_rule_cannot_pass(tmp_path):
    write_dataset(tmp_path, history())
    changed = copy.deepcopy(CONTRACT)
    rule = changed.dataset("payment_events").quality[0]
    rule.type = "custom"
    rule.query = None
    report = validate_dataset(tmp_path, changed)
    assert not report.passed


@pytest.mark.parametrize("outcome", ["missing", "info"])
def test_runner_rejects_missing_native_rule_results(tmp_path, monkeypatch, outcome):
    from datacontract.model.run import Check, Run

    write_dataset(tmp_path, [])
    run = Run.create_run()
    run.checks = (
        [
            Check(type="quality", qualityId=rule.id, result="info", name="not executed")
            for dataset in CONTRACT.model.schema_
            for rule in dataset.quality
        ]
        if outcome == "info"
        else [Check(type="quality", result="passed", name="incomplete execution")]
    )
    monkeypatch.setattr("payment_data_contracts.validation.DataContract.test", lambda self: run)
    report = validate_dataset(tmp_path)
    assert "payment_lifecycle" in failures(report)
    assert "label_attempt_linkage" in failures(report)


def test_cli_writes_failure_evidence_and_detects_documentation_drift(tmp_path):
    result = subprocess.run(
        [
            sys.executable,
            "-m",
            "payment_data_contracts.cli",
            "validate",
            str(tmp_path),
            "--report",
            str(tmp_path / "report.json"),
        ],
        capture_output=True,
        text=True,
    )
    assert result.returncode == 1, result.stderr
    report = json.loads((tmp_path / "report.json").read_text())
    assert report["status"] == "failed"
    assert report["contract"] == CONTRACT.provenance
    docs = tmp_path / "contract.md"
    docs.write_text(documentation(CONTRACT) + "stale\n")
    result = subprocess.run(
        [sys.executable, "-m", "payment_data_contracts.cli", "docs", str(docs), "--check"],
        capture_output=True,
        text=True,
    )
    assert result.returncode == 1
    assert "stale" in result.stdout
