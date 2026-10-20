## Why

Historical payment datasets need one maintained agreement for schema, meaning and quality before the generator and lake pipelines depend on them. Separate Arrow definitions, documentation and catalog assertions would drift and repeat validation logic.

## What Changes

- Add a pinned ODCS contract for versioned source payment events and separate delayed labels.
- Derive Arrow schemas from the contract and expose one reusable validation entry point with structured rule results and contract provenance.
- Declare payment lifecycle and label quality rules as SQL in ODCS; fail on unsupported checks instead of silently skipping them.
- Generate contract documentation and check it, the contract and meaningful valid/invalid datasets in CI.
- Keep backend wire contracts intact. DataHub deployment and rewriting the historical generator are follow-up work.
- Introduce no legacy schema aliases or protobuf-JSON compatibility path in the new dataset package.

## Capabilities

### New Capabilities

- `dataset-contracts`: Authoritative ODCS datasets, Arrow mapping, shared quality execution and generated documentation.

### Modified Capabilities

None.

## Impact

Adds an independently packaged Python contract runtime under `contracts/data/`, contract-focused CI and generated documentation linked from the contributor guide. The foundation merges before the historical generator; downstream consumers will use this package rather than copy its schemas or rules. No running service, storage platform or DataHub deployment changes.
