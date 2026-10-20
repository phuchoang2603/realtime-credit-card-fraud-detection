## Context

See proposal.md for motivation. Main has backend protobuf contracts but no historical generator runtime. This foundation must merge independently before the generator PR and remain consumable by later batch pipelines.

## Goals / Non-Goals

The ODCS document is authoritative for source dataset schemas, quality rules and generated documentation. The implementation supports local staged Parquet, including empty datasets and mixed event schema versions. It does not deploy DataHub, rewrite the generator, define future curated datasets or replace backend protobuf.

## Decisions

- Package ODCS and its Arrow adapter together under contracts/data so installed consumers load the exact same document as CI.
- Pin ODCS v3.1.0, its validated model and Data Contract CLI. Declare nested types using standard properties/items and explicit physical types; use an explicit sinceSchemaVersion custom property only for physical event evolution.
- Express domain checks as ODCS SQL quality rules executed by Data Contract CLI on DuckDB. Canonicalize identical event deliveries inside the quality SQL; keep conflict detection over the original rows. This removes the need for a second Python payment state machine or per-stage rule implementations.
- Inspect each Parquet file with derived Arrow schemas before normalization. Build DuckDB views over the files; normalize old event structs only after exact physical inspection. The native runner evaluates schema and quality against the combined logical datasets.
- Emit contract hash and per-rule results in a JSON report, including failures. Treat warnings, skipped checks and missing declared rules as unsuccessful validation.
- Generate a Markdown reference from the loaded ODCS model for the existing wiki, with a CI drift check. Keep the commands and execution boundaries in CONTRIBUTING.md.

## Risks / Trade-offs

- DuckDB SQL is not universally portable to Spark. Keep the declared rules and identifiers authoritative; add a compatible execution adapter only when Spark is implemented.
- Data Contract CLI's schema engine does not replace exact Parquet inspection. Physical checks run before union-by-name normalization.
- Native contract execution brings additional dependencies. Keep them in a validation extra so schema-only consumers need only the ODCS model, YAML and Arrow.
- ODCS import creates catalog metadata, not observed lineage or executed quality results. Future DataHub integration must ingest the same document and publish run evidence separately.

## Migration Plan

Merge this foundation first. Rewrite the historical generator to depend on it and delete its local Arrow schemas and lifecycle validator, replacing its protobuf-JSON fixture path with contract-native evidence. Rebase unrelated tooling/infra separately after the generator rewrite. No compatibility aliases are introduced here.
