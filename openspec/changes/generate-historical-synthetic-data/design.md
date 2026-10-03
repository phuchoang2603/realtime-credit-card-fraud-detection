## Context

`define-payment-and-decision-contracts` provides `payments.v1` types and events, `labels.v1`, and Python codegen. There is no data infrastructure in this repository or talos-proxmox. Object storage is an existing MinIO on TrueNAS at `minio.home.phuchoang.sbs`, outside the Talos clusters and outside Argo CD. The previous generator followed the [fraud detection handbook](https://fraud-detection-handbook.github.io/fraud-detection-handbook/Chapter_3_GettingStarted/SimulatedDataset.html): customer and terminal profiles, terminals within radius `r`, and three fraud scenarios.

## Goals / Non-Goals

**Goals:**

- One command that reproduces a dataset from config and seed, with rubric-visible data problems.
- Scenarios that force DP3 and streaming features to matter.
- A layout DP1 can ingest incrementally by partition.

**Non-Goals:**

- Live generation, transport faults and Kafka (`simulate-live-traffic-with-transport-faults`, which drives the gateway and continues datasets from this generator).
- A container image, Kubernetes job or orchestrator. DP1 decides how generation is scheduled.
- Bronze tables, analytical schemas (#45) or feature definitions.
- Configurable drift for the final coursework (#44 extension). The config leaves room for it.

## Decisions

### Handbook model mapped to ecommerce

| Handbook | Generator |
|---|---|
| Customer location, amount mean/std, transactions per day | Customer home region, account age, purchase frequency, category preferences, per-category amount distribution |
| Terminals within radius `r` | Familiarity sets: 1–3 devices, home IP country and networks, 1–2 shipping addresses, preferred merchants |
| Terminal | Merchant (seller) with category and Zipf-distributed popularity |
| Scenario 1: amount > 220 | `obvious_high_value`: attempts far above category price; kept as the single-attempt sanity scenario |
| Scenario 2: compromised terminal, 28 days | `compromised_device_farm`: a device/IP network used across many stolen cards for a configured window |
| Scenario 3: leaked card, 1/3 of transactions ×5 for 14 days | `stolen_card_new_device`: a share of a compromised customer's attempts from a new device and IP at elevated amounts |
| 7-day feedback delay | Explicit chargeback delay distribution and `label_available_at` |

New scenarios: `account_takeover` (old account, new device, foreign IP, new address, high-value categories), `reshipping_mule` (one address fingerprint shared by unrelated customers, forwarder postal codes), `card_testing` (bursts of low amounts, many method fingerprints, shared device, high processor declines), `retry_attack` (repeated attempts on one payment with new methods after declines) and `friendly_fraud` (legitimate attempts later charged back). A day-by-day loop selects compromised entities per day, as the handbook does, so scenario activity is time-bounded and gives later drift work a natural shape.

Detectable-by-history features the scenarios target (computed later by DP3/streaming, not by the generator): device or IP distinct customers over 24 hours, new-device and new-IP flags, IP country ≠ shipping country, account age, distinct buyers per address, declines per device over 1 hour, attempts per payment, amount ÷ 30-day customer average, merchant decline rate over 7 days, and time since the previous purchase.

### Generator-owned decision policy

Histories need `RiskDecisionRecorded` events. The generator implements its own small policy, recorded as `sim-rules-v1`, rather than importing the fraud service (a cross-service import) or calling it (historical generation must not call live APIs). The version string makes clear that these decisions simulate past gateway behavior and are not produced by `rules-v1`. Processor outcomes are simulated in-process with configured issuer-decline, insufficient-funds and unknown-outcome rates. Unknown outcomes resolve within a configured delay.

### Protobuf first, then Arrow

Each record is built as a generated protobuf message, so the contract is enforced where records are created. Records are converted to Arrow through a descriptor-driven function: nested messages become structs, timestamps become `timestamp[us, UTC]`, enums become strings, and `int64` stays `int64` (protobuf JSON would stringify it). Alternative considered: protobuf JSON strings in a payload column. Rejected because Parquet-level nulls and nested columns are what the schema-evolution evidence has to show.

### One wide events dataset plus labels

```
s3://synthetic-source/historical/<dataset_version>/
  payment_events/occurred_date=YYYY-MM-DD/part-NNNNN.parquet
  fraud_labels/label_available_date=YYYY-MM-DD/part-NNNNN.parquet
  manifest.json
```

`payment_events` has typed envelope columns plus one struct column per payload type, exactly one of which is non-null per row. This mirrors the single `PaymentEvent` topic that strict streaming will publish later, so DP1 has one shape for batch and live input. Labels are partitioned by availability date so point-in-time reads can prune them. A separate `synthetic-source` bucket keeps source data outside bronze and allows write-only generator credentials.

### Data problems as explicit write-time transforms

- **Skew:** Zipf exponents for merchant popularity and category mix.
- **High cardinality:** configured distinct counts for fingerprints. Fraud scenarios add many one-off fingerprints.
- **Schema evolution:** config declares `schema_evolution.cutover_date` and field paths (default `AttemptSignals.user_agent_family`). Partitions before the cutover are written with a schema lacking that column and envelope `schema_version` 1; later partitions use version 2.
- **Duplicates:** a seeded sample of events is written again in a later part file with an identical `event_id` and content, simulating export retries. The histories themselves stay valid.

The manifest records configured and observed values for each, which is the numeric side of the evidence.

### Continuity with live simulation

The live simulator continues a dataset through the gateway, so three things must match across the handoff:

- **Random streams:** derived as `SeedSequence(seed, stage)` for profiles and `SeedSequence(seed, stage, day)` for each day's scenario selection and purchases. Profiles and any day's schedule can then be rebuilt without replaying purchases. Scenario windows that started earlier are recovered by re-evaluating selection over the maximum window length before that day.
- **Identities:** merchant and customer IDs are UUIDv5 values of the configured synthetic `integration_id` and the external seller or buyer ID, matching Accounts. Fingerprints are HMAC-SHA256 values under the non-secret synthetic key, matching the edge for synthetic integrations. Both are checked against `contracts/synthetic/identity-vectors.json`.
- **Time:** the manifest records the history end. The live simulator refuses a gap larger than its configured maximum, so evidence runs generate a continuation dataset ending on the run date.

The manifest also stores a SHA-256 digest of the canonical profile records. The live simulator compares it after rebuilding and refuses to run on a mismatch, which catches generator changes made after the dataset was written. Alternative considered: writing profiles as a dataset that the simulator reads. Rejected because profiles carry raw synthetic addresses and compromise truth that bronze consumers should not ingest.

### Validation before write

A Python history validator implements the lifecycle rules and runs over every payment before any file is written. Golden histories in `contracts/payments/v1/testdata/` (one protobuf-JSON history per file, with a `valid` or `invalid: <rule>` header) are the conformance suite. The simulator tests use them now, and the Go Payments aggregate reuses them later. Generation is in-memory per day-chunk, and validation failure aborts the run before the manifest exists.

### Storage access

`pyarrow.fs.S3FileSystem` with `endpoint_override=https://minio.home.phuchoang.sbs`, path-style addressing and credentials from `AWS_ACCESS_KEY_ID`/`AWS_SECRET_ACCESS_KEY`, loaded from the environment or a gitignored `.env`. `--output` accepts `s3://…` or a local path. The operator creates the bucket and access key on TrueNAS. No Terraform or Kubernetes resource is added for an externally managed NAS service.

### Libraries

numpy `Generator` (PCG64) with streams derived per stage and per day from the seed; pyarrow for Parquet and S3; pydantic for config; PyYAML for the config file. The notebook dependency group adds DuckDB (fast `approx_count_distinct` on Parquet), matplotlib, scikit-learn and Jupyter. Polars was considered, but pyarrow plus DuckDB covers writing and analysis without a third dataframe library.

### Sanity notebook

`src/simulator/notebooks/historical-sanity.ipynb` reads a dataset version and shows event and label counts, fraud rate per scenario, amount and category distributions, merchant skew, approximate distinct fingerprints, nulls around the cutover, and the observed duplicate rate. As a learnability check, it fits a small gradient-boosting model once on single-attempt fields only and once with a few history features computed in the notebook. The expectation is that history features clearly help and the single-attempt model is not near-perfect. This is exploratory evidence, not the #53 rubric notebook. Executed output is committed so captures match the recorded dataset version.

## Risks / Trade-offs

- [Scenarios too easy or impossible] → The learnability check in the notebook, then tune parameters and bump `dataset_version`.
- [Large default volumes are slow in Python] → Vectorize per day with numpy and write per day-chunk. Default config is sized for minutes, not hours; larger volumes are a config change.
- [Duplicate policy between the Python validator and the Go aggregate] → Golden histories are the shared oracle for both.
- [NAS availability] → Local output keeps development and CI independent. Evidence runs record the endpoint and dataset version.
- [Generator decisions differ from rules-v1] → Intentional and labelled by policy version. Training uses labels, not these decisions.

## Migration Plan

Additive. Create the bucket and key, generate `historical/v1`, run the notebook and commit the evidence. Rollback deletes the dataset prefix; no consumer exists yet.

## Open Questions

- Default volume (customers, days) for the evidence run can be tuned after the first timing run without changing the design.
