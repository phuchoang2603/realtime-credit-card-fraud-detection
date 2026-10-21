## Context

See proposal.md for motivation. Main has backend protobuf contracts but no historical generator runtime. This foundation must merge independently before the generator PR and be usable by later batch pipelines.

## Goals / Non-Goals

The ODCS agreement is authoritative for source dataset schemas, quality rules, ownership and field meanings. Use existing tooling for lint, documentation and execution. This foundation does not implement a Python contract runtime, wire dataset validation into a producer, deploy DataHub, define future curated datasets or replace backend protobuf.

## Decisions

- Keep the agreement under contracts/data. Applications follow the declared schema and rules; no shared runtime package is required.
- Pin the ODCS document to v3.1.0 and use Data Contract CLI 1.2.7 directly through uvx. CI lints the agreement and compares the native Markdown export with its committed schema reference under docs/data-ml. Keep product purpose and consumption guidance on a dedicated page alongside the generated reference; the architecture page links to it.
- Keep payment lifecycle and label checks in ODCS SQL quality rules. Canonicalize identical event deliveries inside the quality SQL and detect conflicts over the original rows.
- The validation server describes local DuckDB tables or views prepared by a producing or ingesting job. The existing CLI executes the rules and writes its native report. Connection settings, physical file inspection and publication gating belong with the job, not a new contract application.
- Treat schemaVersions and sinceSchemaVersion as descriptive physical layout metadata. They do not imply an automatic Arrow mapper or engine enforcement.
- Ingest this same agreement into DataHub when catalog integration is implemented; publish execution evidence separately.

## Risks / Trade-offs

- Agreement lint does not validate data. Dataset validation is integrated and tested with the historical generator and each pipeline.
- Current SQL rules use DuckDB syntax. Spark and Flink support must be verified with the actual job and its execution engine; ODCS does not make arbitrary SQL portable.
- Great Expectations exports do not automatically execute these ODCS SQL quality rules. Avoid separately maintained rule suites; select an integration only when a pipeline needs it.
- Native Markdown export documents schemas but does not render every ODCS section, including dataset SQL rules. The full agreement remains authoritative and directly readable.
- Pinning the CLI fixes its interface while its dependencies are resolved by uvx; this PR introduces no independent application dependency lock.

## Migration Plan

Merge this agreement foundation first. Update the historical generator to follow it and invoke existing validation tooling at its publication boundary. Remove superseded local rules rather than retaining parallel validation implementations. Rebase unrelated tooling/infra separately. Integrate DataHub and compute-engine execution with their concrete pipelines.
