# Verification acceptance

Verify the planned gateway and data workloads with new, reproducible runs. Source files, configuration and passing checks establish only their tested scope; each roadmap capability needs its own acceptance evidence.

## Gateway acceptance

- Exercise checkout, risk decisions, simulated processing and persisted outcomes across the service boundaries.
- Verify idempotent retries, changed-request conflicts, tenant isolation, fraud unavailability, unknown processor outcomes and recovery.
- Replay recorded payment histories without repeating processor operations or webhook delivery.
- Demonstrate signed webhook delivery and receiver deduplication when that workload is implemented.

## Data and ML acceptance

- Record dataset manifests, source/contract versions, pipeline runs and immutable input snapshots.
- Validate event-time features, warmup, delayed-label cutoffs and offline/online parity.
- Evaluate models with chronological holdouts; distinguish synthetic performance from real-world fraud performance.
- Measure deployment, load and failure recovery using a stated workload, resources, duration and latency/error criteria.

For every result, record the revision, environment, commands, expected and observed outputs, and relevant failures. CI records checks and image builds; executed reports belong under `docs/verification/` with their workload. Follow the [roadmap](../architecture/roadmap.md) for acceptance scope and the [CI workflow](../../.github/workflows/ci.yml) for current checks.
