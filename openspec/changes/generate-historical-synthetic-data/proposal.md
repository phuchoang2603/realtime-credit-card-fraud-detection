## Why

DP1–DP3, the analytical schemas and the offline rubric rows (#43) need reproducible ecommerce payment histories with known fraud, realistic data problems and delayed labels, stored outside bronze. The handbook simulator this project used before produces card-present terminal transactions that the new `payments.v1` contracts cannot represent. The new contracts carry devices, IP geography, shipping addresses and payment methods, so the generator can also simulate the ecommerce fraud patterns the gateway is meant to detect.

## What Changes

- Add a Python `src/simulator` project with a seeded `historical` command. It builds customer, merchant, device and payment-method profiles, generates legitimate purchases and eight fraud scenarios, and emits lifecycle-valid payment events and delayed labels using generator-owned Arrow schemas.
- Simulate the offline data problems the rubric names, through configuration: merchant/category skew, high-cardinality identifiers, schema evolution (a field absent before a cutover date), and duplicate exports of the same event.
- Validate every generated payment history against the lifecycle transitions before writing. Add shared golden valid/invalid histories under `contracts/payments/v1/testdata/` that Payments will reuse.
- Make datasets continuable by the live simulator: profiles and daily scenario schedules can be rebuilt from the seed, IDs and fingerprints match the gateway through shared identity vectors, and the manifest records the history end and a profile digest.
- Write immutable, versioned Parquet datasets with a manifest to the `synthetic-source` bucket on RustFS at `minio.home.phuchoang.sbs`, partitioned by event date. Compact and sort staged partitions using bounded external sorting, independent row-group and file sizing, and Zstandard compression. Local filesystem output is supported for verification.
- Run the finite, containerized historical evidence job in a prod data namespace on the existing Argo Workflows controller; retain CLI/local-output support. Do not install a second controller or run the generator interactively in a notebook.
- Execute the sanity notebook only in a persistent prod Kubeflow Notebook using the same source identity as the generator. Record distributions, schema cutover, duplicates, fraud rates and learnability as #43 evidence; #53 owns the later feature-based training notebook.
- Run simulator lint and format checks in the existing Python CI job; the simulator has no automated test suite.

## Capabilities

### New Capabilities

- `historical-data-generation`: Seeded, configurable generation of ecommerce payment histories, fraud scenarios, offline data problems, delayed labels, and immutable versioned output to object storage.

### Modified Capabilities

- `payment-contracts`: Adds shared golden lifecycle histories that every producer's validator must accept or reject.
- `kubeflow-workspace`: Adds a source ExternalSecret to the Profile and a finite generator workflow in a separate prod data namespace, sharing one source identity and ClusterSecretStore.

## Impact

- New `src/simulator/` (own `pyproject.toml`/`uv.lock`), `contracts/payments/v1/testdata/`, and `src/simulator/configs/historical-v1.yaml`.
- The simulator owns explicit Arrow schemas and has no protobuf or gRPC dependency; backend code generation remains separate.
- CI Python job runs both Python projects; `.gitignore` drops the legacy `simulated-data-*` entries.
- External prerequisite: the `synthetic-source` bucket and `historical/*`-scoped `synthetic-historical-writer` identity exist on RustFS. Its key pair is stored in Doppler `prd` for the generator, Notebook and downstream source consumers; local-output generation needs no key. The shared `secret-store` chart provides one ClusterSecretStore for both consumer namespaces. No credentials or RustFS administrator access belong in the repository.
- Add an immutable simulator image, a data-namespace WorkflowTemplate and source ExternalSecrets for the generator and existing prod workspace chart. The platform must watch the data namespace and the consumer must own its GitOps namespace resources. Docs: new `docs/verification/historical-data.md` with workflow/notebook commands and captures, sidebar link, roadmap #43 row status, and data-and-ML flows storage wording.
- Depends on `define-payment-and-decision-contracts` and prod Profile, Notebook and shared Argo readiness for evidence. This knowingly makes #43 notebook acceptance wait for the experimental prod Kubeflow rollout.
