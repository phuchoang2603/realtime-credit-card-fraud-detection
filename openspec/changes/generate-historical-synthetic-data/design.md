## Context

`define-payment-and-decision-contracts` provides `payments.v1` types and events, `labels.v1`, and Python codegen. The `synthetic-source` bucket is hosted by the existing RustFS S3 endpoint at `minio.home.phuchoang.sbs`, outside the Talos clusters and Argo CD. The previous generator followed the [fraud detection handbook](https://fraud-detection-handbook.github.io/fraud-detection-handbook/Chapter_3_GettingStarted/SimulatedDataset.html): customer and terminal profiles, terminals within radius `r`, and three fraud scenarios.

## Goals / Non-Goals

**Goals:**

- One command that reproduces a dataset from config and seed, with rubric-visible data problems.
- Scenarios that force DP3 and streaming features to matter.
- A layout DP1 can ingest incrementally by partition.

**Non-Goals:**

- Live generation, transport faults and Kafka (`simulate-live-traffic-with-transport-faults`, which drives the gateway and continues datasets from this generator).
- A second Argo controller or a KFP training pipeline for source generation; DP1 decides how downstream ingestion is scheduled.
- Bronze tables, analytical schemas (#45) or feature definitions.
- Configurable drift for the final coursework (#44 extension). The config leaves room for it.

## Decisions

### Handbook model mapped to ecommerce

| Handbook | Generator |
|---|---|
| Customer location, amount mean/std, transactions per day | Customer account age, purchase frequency, familiar merchants and per-merchant-category amount distribution |
| Terminals within radius `r` | Familiarity sets: 1–3 devices, home IP country and networks, 1–2 shipping addresses, preferred merchants |
| Terminal | Merchant (seller) with category and Zipf-distributed popularity |
| Scenario 1: amount > 220 | `obvious_high_value`: attempts far above category price; kept as the single-attempt sanity scenario |
| Scenario 2: compromised terminal, 28 days | `compromised_device_farm`: a device/IP network used across many stolen cards for a configured window |
| Scenario 3: leaked card, 1/3 of transactions ×5 for 14 days | `stolen_card_new_device`: a share of a compromised customer's attempts from a new device and IP at elevated amounts |
| 7-day feedback delay | Explicit chargeback delay distribution and `label_available_at` |

New scenarios: `account_takeover` (old account, new device, foreign IP, new address, high-value categories), `reshipping_mule` (one address fingerprint shared by unrelated customers, forwarder postal codes), `card_testing` (bursts of low amounts, many method fingerprints, shared device, high processor declines), `retry_attack` (repeated attempts on one payment with new methods after declines) and `friendly_fraud` (legitimate attempts later charged back). A day-by-day loop selects compromised entities per day, as the handbook does, so scenario activity is time-bounded and gives later drift work a natural shape. Card-testing bursts group purchases already selected by the configured rate rather than adding purchases. Legitimate purchases also include new contexts, foreign IPs and small amounts, controlled by `familiar_context_probability`, `foreign_ip_probability` and `small_purchase_probability`; forwarding postal codes also occur in legitimate shipping, so these fields alone do not identify fraud.

Detectable-by-history features the scenarios target (computed later by DP3/streaming, not by the generator): device or IP distinct customers over 24 hours, new-device and new-IP flags, IP country ≠ shipping country, account age, distinct buyers per address, declines per device over 1 hour, attempts per payment, amount ÷ 30-day customer average, merchant decline rate over 7 days, and time since the previous purchase.

### Generator-owned decision policy

Histories need `RiskDecisionRecorded` events. The generator implements its own small policy, recorded as `sim-rules-v1`, rather than importing the fraud service (a cross-service import) or calling it (historical generation must not call live APIs). The version string makes clear that these decisions simulate past gateway behavior and are not produced by `rules-v1`. Processor outcomes are simulated in-process with configured issuer-decline, insufficient-funds and unknown-outcome rates. Unknown outcomes resolve within a configured delay.

### Explicit Arrow schemas

The generator creates plain Python records and writes them through explicit, versioned Arrow schemas in `simulator/arrow.py`. Nested values are structs, timestamps are `timestamp[us, UTC]`, categorical values are strings, and integer amounts remain `int64`. The ODCS agreement in contracts/data/historical-payments.odcs.yaml owns schema and lifecycle quality rules; the existing Data Contract CLI engine verifies the written records against it. Protobuf and gRPC remain backend contracts; historical generation needs neither generated bindings nor descriptor conversion.

### One wide events dataset plus labels

```
s3://synthetic-source/historical/<dataset_version>/
  payment_events/occurred_date=YYYY-MM-DD/part-NNNNN.parquet
  fraud_labels/label_available_date=YYYY-MM-DD/part-NNNNN.parquet
  manifest.json
```

`payment_events` has typed envelope columns plus one struct column per payload type, exactly one of which is non-null per row. The fields follow the payment lifecycle semantics; a future live ingestion adapter maps backend events into the analytical schema. Labels are partitioned by availability date so point-in-time reads can prune them. A separate `synthetic-source` bucket keeps source data outside bronze. The generator, Notebook and downstream source consumers share one identity with bucket listing and object read/write access under `historical/*`.

### Data problems as explicit write-time transforms

- **Skew:** Zipf exponents for merchant popularity and category mix.
- **High cardinality:** configured distinct counts for fingerprints. Fraud scenarios add many one-off fingerprints.
- **Schema evolution:** config declares `schema_evolution.cutover_date` and field paths (default `attempt_started.signals.user_agent_family`). Each event's timestamp determines its schema version and partition, including retries that cross the cutover. Partitions before the cutover lack the declared column and use envelope `schema_version` 1; later partitions use version 2.
- **Duplicates:** a seeded sample of events is exported again with an identical `event_id` and content, simulating export retries. Sampling permits repeated copies so rates above 50% remain supported; the rate is duplicate rows divided by all exported event rows. Final compaction preserves these copies. The histories themselves stay valid.

The manifest records configured and observed values for each, which is the numeric side of the evidence.

### Sorted and compacted Parquet

Generation flushes at most 4,096 pending records to temporary Parquet files. After duplicate sampling, DuckDB sorts one date partition at a time with a configurable 128 MiB engine memory budget and disk spill. It streams Arrow batches into final Parquet files; the generator does not collect a whole partition in memory. The engine budget is not a cap on process RSS: Arrow batches, profiles, statistics and sort buffers also consume memory.

Events are sorted by `occurred_at`, `payment_id`, `aggregate_version`, and `event_id`; labels by `label_available_at`, `payment_id`, `attempt_id`, and `source`. Each date retains its physical schema, including the cutover. Row-group statistics and truthful sorting metadata support downstream pruning; readers still choose columns and apply filters.

The `storage` configuration independently controls `row_group_rows` (default 65,536), `target_file_size_mb` (default 128 MiB), and `sort_memory_limit_mb` (default 128 MiB). Files rotate after a completed row group reaches the compressed-byte target, so the target is approximate and a file can exceed it by one group plus its footer. Small partitions remain small rather than being padded or combined across dates. Final output uses Zstandard level 3, dictionary encoding with writer fallback, and statistics. The manifest records the codec, sorting columns, file and row-group counts, and stored bytes. Temporary disk holds staging, sorted output and spill files; completed input partitions are removed after successful compaction.

### Continuity with live simulation

The live simulator continues a dataset through the gateway, so three things must match across the handoff:

- **Random streams:** derived as `SeedSequence(seed, stage)` for profiles and `SeedSequence(seed, stage, day)` for purchases and scenario activations. Each scenario reconstructs customer activations over its own lookback window, including warmup before the history start. An activation remains effective for `window_days`; its start day identifies reused fraud contexts. Activation probability is `1 - (1 - rate) ** (1 / window_days)`, so the marginal probability of an active customer matches the configured rate before overlapping scenarios are resolved in declared priority order. Friendly fraud remains per-success label noise. Profiles and any day's schedule can be rebuilt without replaying purchases. Category skew belongs to merchants; customer category preferences are not modeled.
- **Identities:** merchant and customer IDs are UUIDv5 values of the configured synthetic `integration_id` and the external seller or buyer ID, matching Accounts. Fingerprints are HMAC-SHA256 values under the non-secret synthetic key, matching the edge for synthetic integrations. Both are checked against `contracts/synthetic/identity-vectors.json`.
- **Time:** the manifest records the history end. The live simulator refuses a gap larger than its configured maximum, so evidence runs generate a continuation dataset ending on the run date.

The manifest also stores a SHA-256 digest of the canonical profile records. The live simulator compares it after rebuilding and refuses to run on a mismatch, which catches generator changes made after the dataset was written. Alternative considered: writing profiles as a dataset that the simulator reads. Rejected because profiles carry raw synthetic addresses and compromise truth that bronze consumers should not ingest.

### Validation before write

Generation stages bounded Parquet batches, compacts them and accumulates manifest statistics on temporary disk. DuckDB views expose complete payment histories and labels across all final partitions, normalizing the older event layout with an empty current-layout input. Data Contract CLI 1.2.7 runs the ODCS schema and quality checks directly. Failure identifies the failed rules and leaves the destination untouched. A passing native report is copied as validation.json; the manifest records its path and the ODCS ID, version and SHA-256 and is written last. Parquet records are deterministic; the native report carries per-run identifiers and timestamps. The agreement is read from the repository, and the image preserves that repository layout. Temporary storage must accommodate the exported dataset; profiles, one day's schedule and exact distinct-identity sets remain in memory.

### Storage access

The historical command writes from a bounded prod Argo Workflow pod in a data namespace to the RustFS S3 API using `pyarrow.fs.S3FileSystem(scheme="https", endpoint_override="minio.home.phuchoang.sbs")` with path-style addressing. `--output` accepts `s3://synthetic-source` or a local path. The S3 client reads `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` from the process environment. Both workload charts map Doppler `HISTORICAL_RUSTFS_ACCESS_KEY` and `HISTORICAL_RUSTFS_SECRET_KEY` from `prd` into a namespace-local `historical-source` Secret. Local-output generation needs no credentials. No access keys, administrator credentials or dataset contents are committed.

The `synthetic-source` bucket and `synthetic-historical-writer` IAM user exist on RustFS. Its policy is tracked in `infra/policies/synthetic-historical-writer.json`; its key pair is stored in Doppler project `realtime-credit-card-fraud-detection`, config `prd`. RustFS rejected a prefix-conditioned `ListBucket` in live testing, so the policy allows listing this dedicated bucket, while object read/write/multipart permissions are restricted to `historical/*` and deletion is not granted. Read and write under the prefix succeeded; writes outside it and deletes were denied. The Notebook and downstream source consumers share this identity and therefore have the same source write permission. The generator must refuse a previously completed dataset version. Only one writer may use a dataset version at a time; incomplete versions require operator cleanup or a new version.

Do not add a Terraform root for a single externally hosted bucket. In `talos-proxmox`, the operator-applied `terraform/foundation` root creates shared infrastructure and Doppler service tokens; its documentation notes that provider-read secrets and generated credentials enter Terraform state. If provisioning later needs repeatable management across environments, add an application-owned `infra/iac/foundation` root with restricted remote state and an explicit key-rotation procedure, rather than managing this bucket through the cluster's GitOps or another repository's state. A Doppler Terraform provider is not required for runtime injection and would broaden the state exposure of the generator key.

### Libraries

numpy `Generator` (PCG64) with streams derived per stage and per day from the seed; pyarrow for schemas, Parquet and S3; DuckDB for bounded external sorting and validation views; pinned Data Contract CLI for native ODCS execution; pydantic for config; PyYAML for the config file. The notebook dependency group adds matplotlib, scikit-learn and Jupyter, and reuses DuckDB for analysis. Polars was considered, but pyarrow plus DuckDB covers writing and analysis without a third dataframe library.

### Sanity notebook

`src/simulator/notebooks/historical-sanity.ipynb` reads a dataset version and shows event and label counts, fraud rate per scenario, amount and category distributions, merchant skew, approximate distinct fingerprints, nulls around the cutover, and the observed duplicate rate. As a learnability check, it fits a small gradient-boosting model once on single-attempt fields only and once with a few history features computed in the notebook. The expectation is that history features clearly help and the single-attempt model is not near-perfect. This is exploratory evidence, not the #53 rubric notebook. Executed output is committed so captures match the recorded dataset version.

The digest-pinned simulator image packages `src/simulator` as a CLI for the Argo job, not as a Jupyter server. The notebook runs in a separate pinned Kubeflow Jupyter image with access to the same versioned project code and notebook dependencies; derive a custom Notebook image from the Kubeflow base only if the chosen standard image lacks them. Keep notebook source in Git and export executed evidence back to the repository; the Kubeflow PVC is a workspace, not the source of truth.

Execute the sanity notebook in a persistent `fraud-ml-prod` Kubeflow Notebook with a pinned image and the shared `synthetic-source/historical/*` identity. Record the dataset/manifest hash, image digest and executed results under `docs/verification/`. This #43 source-level check is not the #53 rubric notebook: that baseline still requires governed point-in-time features and delayed-label joins. There is no local-notebook evidence requirement, so #43 acceptance waits for authenticated prod Notebook access and namespace Secret readiness.

### Evidence workload and namespace

Package the `historical` CLI as a pinned simulator image and a WorkflowTemplate in the prod data namespace (`fraud-data-prod`). Extend the platform's sole prod Argo controller watch list to include it. The consumer GitOps root creates the data namespace and its resources separately from the Kubeflow Profile. Bound concurrency to one run per dataset version and keep immutable-manifest refusal and explicit cleanup for interrupted attempts; retries cannot silently overwrite accepted objects. The shared `secret-store` chart creates one `fraud-doppler` ClusterSecretStore for `prd`, serving `fraud-data-prod` and `fraud-ml-prod` through the existing token Secret in `fraud-ml-prod`. Both workload charts reference that store. A future live simulator can consume the external manifest with the shared source identity, without depending on the prod cluster's Kubernetes API. Namespace credentials, image digests and workflow run IDs are recorded without exposing secret values.

## Risks / Trade-offs

- [Scenarios too easy or impossible] → The learnability check in the notebook, then tune parameters and bump `dataset_version`.
- [Large volumes exhaust memory or temporary storage] → Buffer bounded record batches, reuse Arrow schemas and stage Parquet on disk. Size the workflow's temporary storage for the exported dataset; exact distinct counts and profiles still grow with identity cardinality.
- [Contract and writer divergence] → Native schema and quality checks gate publication against the shared ODCS agreement; Arrow schemas remain the writer's physical representation.
- [RustFS or prod Profile unavailable] → Local output keeps generator development and CI independent, but #43 workflow/notebook evidence waits until prod Argo, Notebook and scoped secrets work; record the endpoint, dataset version and failed prerequisites.
- [Generator decisions differ from rules-v1] → Intentional and labelled by policy version. Training uses labels, not these decisions.

## Migration Plan

Additive. Retain the existing RustFS source identity and consolidate its keys into Doppler `prd`. The Development environment and extra data config are removed. Reuse the existing Doppler token and shared ClusterSecretStore across both workload namespaces, grant the shared Argo controller watch access, reconcile the source ExternalSecrets and generator WorkflowTemplate, generate a new `historical/<dataset_version>` with the shared Argo controller, execute the prod Notebook and commit the evidence. Rollback of a partially written dataset is an operator action with elevated credentials; the source identity cannot delete objects.

## Open Questions

- Default volume (customers, days) for the evidence run can be tuned after the first timing run without changing the design.
