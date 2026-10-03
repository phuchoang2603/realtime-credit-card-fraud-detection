## 1. Project and contracts

- [ ] 1.1 Create `src/simulator` with `pyproject.toml` (numpy, pyarrow, pydantic, PyYAML, protobuf; dev: pytest, ruff; notebook group: duckdb, matplotlib, scikit-learn, jupyter) and `uv.lock`; verify `uv sync --locked` succeeds
- [ ] 1.2 Extend `codegen:proto` to also generate Python bindings into `src/simulator/`, add treefmt exclusions, commit bindings; verify `python -c "import payments.v1.events_pb2"` works inside the project
- [ ] 1.3 Add golden histories under `contracts/payments/v1/testdata/` covering every legal transition and each prohibited transition in the lifecycle requirement, and `contracts/synthetic/identity-vectors.json` with the UUIDv5 namespace, merchant and customer ID cases, and synthetic-key fingerprint cases (unless `build-minimum-payment-gateway` already added them); verify each file parses
- [ ] 1.4 Remove legacy `simulated-data-*` entries from `.gitignore` and verify `git status` is clean afterward

## 2. Core generation

- [ ] 2.1 Implement the pydantic configuration model and `configs/historical-v1.yaml`, rejecting unknown scenarios, inconsistent dates and credential fields; verify with a config-rejection test
- [ ] 2.2 Implement the lifecycle history validator; verify it accepts all valid and rejects all invalid golden histories
- [ ] 2.3 Implement seeded profile and familiarity generation (customers, merchants with Zipf popularity, devices, IP networks, addresses, payment methods) on per-stage and per-day random streams, with IDs and fingerprints derived as in the identity vectors; verify two runs with one seed produce identical profiles, a single day's schedule matches the full run, and the identity vector cases pass
- [ ] 2.4 Implement legitimate purchase generation with the `sim-rules-v1` decision policy and simulated processor outcomes, including declines, unknown outcomes that resolve, retries and expiries; verify generated histories pass the validator
- [ ] 2.5 Implement the eight fraud scenarios with per-scenario parameters and time-bounded compromise windows; verify the card-testing and scenario-disabled behaviors with small-config tests
- [ ] 2.6 Implement chargeback and simulation-truth labels with configured delays and friendly-fraud noise; verify that declined fraud has truth but no chargeback and that events contain no scenario names

## 3. Output

- [ ] 3.1 Implement descriptor-driven protobuf-to-Arrow conversion (structs, UTC timestamps, enum names, int64); verify a round trip on each payload type
- [ ] 3.2 Implement partitioned Parquet writing, schema-evolution cutover, duplicate export and last-written manifest with configured and observed statistics, history end, profile digest and synthetic key version; verify cutover columns, duplicate rate, manifest counts and that rebuilt profiles match the digest on a small dataset
- [ ] 3.3 Implement the `historical` CLI with local and `s3://` output via `S3FileSystem` against `minio.home.phuchoang.sbs`, refusing existing manifests; verify local reruns are identical and a second run to the same version fails
- [ ] 3.4 Add simulator lint, format and tests to the CI Python job; verify CI passes without credentials

## 4. Evidence

- [ ] 4.1 Create the `synthetic-source` bucket and scoped key on TrueNAS MinIO, generate `historical/v1`, and verify `manifest.json` exists with expected partition counts
- [ ] 4.2 Build and execute `notebooks/historical-sanity.ipynb` against `historical/v1` (distributions, skew, approximate distinct counts, cutover nulls, duplicate rate, fraud rate by scenario, learnability check); tune and bump the dataset version if the check fails; commit executed output
- [ ] 4.3 Add `docs/verification/historical-data.md` with config, commands, dataset version, manifest statistics and explained captures; link it from the sidebar and the roadmap #43 row; update storage wording in data-and-ML flows; verify links resolve
- [ ] 4.4 Run `openspec validate generate-historical-synthetic-data --strict` and verify it passes
