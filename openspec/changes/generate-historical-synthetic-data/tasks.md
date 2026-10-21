## 1. Data product folder

- [x] 1.1 Create `contracts/data-products/historical-payments/` and split the current ODCS into `payment-events.odcs.yaml` and `fraud-labels.odcs.yaml` with new IDs, keeping field meanings and quality rules; move the ODPS descriptor to `product.odps.yaml` (version 2.0.0) with `payment-events` and `fraud-labels` output ports, the labels port listing the events contract in `inputContracts`; delete `contracts/data/` and `contracts/products/`; verify both lint commands pass
- [x] 1.2 Move CI contract checks into `scripts/ci-checks.sh` as one function that discovers every product folder, lints descriptor and contracts and diffs generated references; verify an invalid contract and a stale reference each fail
- [x] 1.3 Replace `docs/data-ml/contracts/historical-payments-contract.md` with generated `docs/data-ml/data-products/historical-payments/{payment-events,fraud-labels}.md`; update `data-products.md` paths and validation command; verify links resolve

## 2. Generator

- [x] 2.1 Create `src/simulator` with `pyproject.toml` (numpy, pyarrow, duckdb, pydantic, PyYAML; dev: pytest, ruff; notebook: matplotlib, jupyter) and `uv.lock`; remove legacy `simulated-data-*` entries from `.gitignore`; verify `uv sync --locked`
- [x] 2.2 Implement the pydantic config and `configs/historical.yaml` (90-day default duration), rejecting unknown scenarios, inconsistent dates and credential fields; verify with a rejection test
- [x] 2.3 Add `contracts/synthetic/identity-vectors.json` unless the gateway change has; implement seeded profiles (customers, Zipf merchants, devices, IP networks, addresses, payment methods) on per-stage and per-day streams; verify same-seed identity, single-day rebuild and identity vectors
- [x] 2.4 Implement legitimate purchases with `sim-rules-v1` decisions and simulated processor outcomes (declines, unknown-then-resolved, retries, expiries)
- [x] 2.5 Implement the five fraud scenarios with time-bounded windows; verify card-testing and scenario-disabled behavior with small configs
- [x] 2.6 Implement chargeback and simulation-truth labels with configured delays; verify declined fraud has truth but no chargeback and events carry no scenario names

## 3. Output and validation

- [x] 3.1 Declare Arrow schemas for both tables; verify column names and nullability match the ODCS files
- [x] 3.2 Flush raw day-chunk Parquet to staging with schema-evolution cutover and duplicate export; verify cutover columns and duplicate rate
- [x] 3.3 Write final daily partitions with one DuckDB `COPY … ORDER BY` per partition (zstd level 3, `row_group_rows`, `target_file_mb`, memory limit and spill directory), deleting raw staging per day; verify sort order, single schema per partition, row-group bounds, duplicates beside originals and that a time-filtered read equals a filtered full scan
- [x] 3.4 Run `datacontract test` for both contracts over DuckDB views of the final files (`hive_partitioning = false`) and publish only on pass, writing reports and the manifest last; verify a corrupted history blocks publication
- [x] 3.5 Implement the `historical` CLI for local and `s3://` output, refusing existing manifests, run locally from devenv with credentials from the environment (no image, chart or workflow); verify local reruns are identical and a second run to the same version fails
- [x] 3.6 Add simulator lint, format and tests to the CI Python job; verify CI passes without credentials

## 4. Regenerate and record

- [x] 4.1 Generate a new 90-day dataset version to `s3://synthetic-source/historical/` and verify the manifest and passing validation reports exist and no day's file is under about 1 MB (otherwise raise volume)
- [x] 4.2 Build and execute `notebooks/historical-sanity.ipynb` (distributions, skew, approximate distinct counts, cutover nulls, duplicate rate, fraud rate by scenario, files, row groups and compressed bytes per column); tune and bump the version if needed; commit executed output
- [x] 4.3 Add the dataset version, commands and manifest statistics to the historical section of `data-products.md`; update the roadmap #43 row and storage wording in `data-pipeline.md`
- [x] 4.4 Delete superseded `historical/*` prefixes after confirming no consumer reads them; verify only the accepted version remains
- [x] 4.5 Run `openspec validate generate-historical-synthetic-data --strict`
