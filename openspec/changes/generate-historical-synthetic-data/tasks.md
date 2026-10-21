## 1. Project and contracts

- [x] 1.1 Create `src/simulator` with `pyproject.toml` (numpy, pyarrow, DuckDB, Data Contract CLI, pydantic, PyYAML; dev: ruff; notebook group: matplotlib, scikit-learn, jupyter) and `uv.lock`; verify `uv sync --locked` succeeds
- [x] 1.2 Keep backend protobuf generation separate from the simulator; remove simulator bindings and verify historical generation needs no protobuf or gRPC imports
- [x] 1.3 Add `contracts/synthetic/identity-vectors.json` with the UUIDv5 namespace, merchant and customer ID cases, and synthetic-key fingerprint cases (unless `build-minimum-payment-gateway` already added them); verify each file parses
- [x] 1.4 Remove legacy `simulated-data-*` entries from `.gitignore` and verify `git status` is clean afterward

## 2. Core generation

- [x] 2.1 Implement the pydantic configuration model and `configs/historical-v1.yaml`, rejecting unknown scenarios, inconsistent dates and credential fields before any output is written
- [x] 2.2 Remove the local lifecycle validator and protobuf-JSON histories; execute the shared ODCS agreement with the native engine before publication and verify valid data and rule failures
- [x] 2.3 Implement seeded profile and familiarity generation (customers, merchants with Zipf popularity, devices, IP networks, addresses, payment methods) on per-stage and per-day random streams, with IDs and fingerprints derived as in the identity vectors; verify two runs with one seed produce identical profiles, a single day's schedule matches the full run, and the identity vector cases pass
- [x] 2.4 Implement legitimate purchase generation with the `sim-rules-v1` decision policy and simulated processor outcomes, including declines, unknown outcomes that resolve, retries and expiries; verify generated histories pass the ODCS checks
- [x] 2.5 Implement the eight fraud scenarios with per-scenario parameters and time-bounded compromise windows, including card-testing bursts and scenario-disabled behavior
- [x] 2.6 Implement chargeback and simulation-truth labels with configured delays and friendly-fraud noise; verify that declined fraud has truth but no chargeback and that events contain no scenario names

## 3. Output

- [x] 3.1 Implement explicit Arrow schemas and plain records (structs, UTC timestamps, categorical strings, int64); verify every payload and label preserves lifecycle semantics and stored values
- [x] 3.2 Implement partitioned Parquet writing, schema-evolution cutover, duplicate export and last-written manifest with configured and observed statistics, history end, profile digest and synthetic key version; verify cutover columns, duplicate rate, manifest counts and that rebuilt profiles match the digest on a small dataset
- [x] 3.3 Implement the `historical` CLI with local and `s3://` output via `S3FileSystem` against `minio.home.phuchoang.sbs`, using process-environment S3 credentials and refusing existing manifests; verify local reruns are identical and a second run to the same version fails
- [x] 3.7 Publish the native ODCS validation report and manifest contract provenance; verify all rules execute, invalid histories and labels leave output untouched, and empty schedules and deterministic Parquet remain supported
- [x] 3.6 Compact and externally sort staged partitions with independent row-group and file targets, Zstandard level 3, statistics and sorting metadata; verify row preservation, duplicate rates, schema cutover, file rotation, deterministic output and projected/filtered result equality
- [ ] 3.4 Add simulator lint and format checks to the CI Python job without a simulator test suite or pytest dependency; verify CI passes without credentials
- [ ] 3.5 Publish a digest-pinned simulator image and add a prod-only `infra/charts/historical-generator` child with its own namespace, source ExternalSecret referencing the shared ClusterSecretStore, and WorkflowTemplate that runs one bounded `historical` CLI job with a manifest guard and restricted service account; render the chart and prod root, and prove it reuses the source identity and creates no second Argo controller

## 4. Evidence

- [x] 4.1 Create the RustFS `synthetic-source` bucket and a dedicated `historical/*` source user, track its policy in `infra/policies/synthetic-historical-writer.json`, store its keys in Doppler project `realtime-credit-card-fraud-detection` config `prd`, and verify scoped read/write, denied out-of-prefix writes and denied deletes; keep administrator credentials out of the repo
- [x] 4.2 Consolidate the existing source identity into Doppler `prd`, verify the migrated keys and existing production secrets, confirm the existing ESO token and RustFS access work, and remove the extra data config and read-only policy. Render one shared ClusterSecretStore and both source ExternalSecrets using the same keys
- [ ] 4.3 Add the prod data namespace to the shared Argo controller watch list in talos-proxmox and run the digest-pinned historical WorkflowTemplate once there; verify S3 reachability, workflow run ID, a last-written `historical/v1/manifest.json` and expected partitions. Keep local CLI output available but do not use it as the evidence run
- [ ] 4.4 Build and execute `notebooks/historical-sanity.ipynb` only in a persistent, authenticated prod Kubeflow Notebook with a pinned image and the shared source Secret (distributions, skew, approximate distinct counts, cutover nulls, duplicate rate, fraud rate by scenario, learnability check); tune and bump the dataset version if the check fails; commit executed output, image digest and dataset/manifest hash. #53 owns the later feature-based training notebook
- [ ] 4.5 Add `docs/verification/historical-data.md` with configuration, Workflow/Notebook commands, Secret rotation and incomplete-run handling, dataset version, manifest statistics and explained captures; link it from the sidebar and the roadmap #43 row; update storage wording in data-and-ML flows; verify links resolve
- [x] 4.6 Run `openspec validate generate-historical-synthetic-data --strict` and verify it passes
