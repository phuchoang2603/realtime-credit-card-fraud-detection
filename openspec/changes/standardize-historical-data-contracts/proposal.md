## Why

Historical payment datasets need one maintained agreement for schema, meaning and quality before the generator and lake pipelines depend on them. Separate documentation, pipeline checks and catalog assertions would drift and repeat validation logic.

## What Changes

- Add an ODCS v3.1.0 agreement for versioned source payment events and separate delayed labels under contracts/data.
- Declare payment lifecycle and label quality rules as SQL in the agreement.
- Use pinned Data Contract CLI commands directly for contract lint and native Markdown export; CI checks agreement validity and documentation drift.
- Let producing and ingesting applications enforce the agreement through existing engines, with their own data connections and publication boundaries.
- Keep backend wire contracts intact. Dataset integration, Spark/Flink execution and DataHub deployment are follow-up work.
- Add no custom Python application, library, schema mapper, validation wrapper or engine-specific rule suite.

## Capabilities

### New Capabilities

- `dataset-contracts`: Authoritative ODCS dataset schemas and quality rules, native agreement lint and generated schema documentation.

### Modified Capabilities

None.

## Impact

Adds one ODCS agreement, an independent CI job using an existing CLI, and generated schema documentation linked from the data and ML wiki page. The foundation merges before the historical generator; producers and consumers follow the agreement without depending on a new application. No running service, storage platform or DataHub deployment changes.
