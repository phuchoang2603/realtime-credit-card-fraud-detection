# Coursework guide

The mini-coursework builds the reproducible ecommerce data platform. Final ML extends that accepted implementation with online/offline feature publication, training, serving and drift monitoring. These pages describe planned capability coverage and required evidence; they do not certify runtime completion.

- [Mini-coursework coverage](mini-coursework.md)
- [Final ML coverage](final-ml.md)
- [Gateway architecture and flows](../architecture/payment-gateway.md)
- [PG-01 documentation task](https://github.com/phuchoang2603/realtime-credit-card-fraud-detection/issues/29)
- [Mini baseline acceptance #67](https://github.com/phuchoang2603/realtime-credit-card-fraud-detection/issues/67)
- [Project roadmap](https://github.com/users/phuchoang2603/projects/3)

## Evidence conventions

| Status | Meaning |
| --- | --- |
| Planned | Required implementation or proof does not yet exist in this repository. Expected artifacts are descriptions, not claimed evidence links. |
| Existing-but-unverified | Source/configuration or historical material exists, but current execution evidence for the complete capability has not been accepted. |
| Verified-with-proof | A scoped claim has linked implementation and reproducible outputs, version/run context and annotated evidence. Verification applies only to that claim. |

The coverage plans currently contain no verified runtime capabilities. The existing fraud API, tests, container, manifests and telemetry are a foundation; the [source inventory](../architecture/payment-gateway.md#current-foundation) explains their limits. A configured deployment is not proof of a live service, and a historical screenshot is not proof of current behavior.

For each implemented capability, record the implementation PR/commit, artifact or dataset version, environment/resources, commands/configuration, expected result, observed output, and failure cases. Embed screenshots beside explanations of the stages, metrics or invariants they demonstrate. Link source code and machine-readable results where available. Keep test conditions and unoptimized measurements so comparisons remain reproducible. Future evidence filenames should be introduced when the artifacts exist, not as broken placeholder links.

Capabilities may group related rubric requirements. Preserving source rows, original points, CSV snapshots or retrieval provenance is not required. This affects rubric bookkeeping, not the data/model lineage and versioned baseline evidence needed to reproduce the system. GitHub issues #29/#67 still use row/point-oriented wording; these capability plans follow the revised documentation scope while retaining required behavior and numerical thresholds.

## Mini-to-final handoff

| Accepted mini asset | Final extension | Regression expectation |
| --- | --- | --- |
| Seeded historical and streaming generators, scenario contracts | Configurable drift and delayed labels | Same seeds/config produce the accepted histories; truth remains separate from decision-time features. |
| DP1 ingestion, DP2 curation, schemas and contracts | Versioned training inputs and expanded lineage | Preserve validation, quarantine, idempotency, schema evolution and source-to-table relationships. |
| DP3 historical features and temporal semantics | Point-in-time training and incremental materialization | Recheck late-data and label-availability boundaries against fixed expected outputs. |
| Streaming event-time windows and deduplication | Independently deployed offline and online writers | Replay baseline windows unchanged; validate sink parity, TTL/freshness and late-write ordering. |
| Unoptimized/optimized batch, streaming and storage benchmarks | Performance regression comparisons | Retain original inputs, resource settings, metrics and correctness checks; explain changed conditions. |
| Quality monitoring and deterministic batch/stream replay novelty proofs | Online-store parity and payment reconciliation | Keep both mini demonstrations; later novelty extends rather than substitutes for them. |

[Baseline acceptance #67](https://github.com/phuchoang2603/realtime-credit-card-fraud-detection/issues/67) must preserve a source commit/release; data, event, schema and feature versions; generator seeds/configuration; dataset manifests; pipeline run IDs; point-in-time cutoffs; expected outputs; and an evidence index. Acceptance reproduces source → bronze → curated tables → offline features plus streaming windows under burst, late and duplicate conditions.

[Notebook/training acceptance #53](https://github.com/phuchoang2603/realtime-credit-card-fraud-detection/issues/53) depends on that accepted baseline. Exploratory work may overlap, but final acceptance must reference the baseline and rerun its regressions. Shared capabilities link back to their mini evidence rather than creating disconnected replacement pipelines.

## Documentation checklist

- README explains the business domain, folder roles and navigation to detail under `docs/`.
- High-level diagrams label planned/current scope, deployable units, external boundaries, numbered flows and data carried by arrows. Libraries/classes are not deployment nodes.
- Schema documentation includes all zones, naming, SCD2, feature timestamps and dim/fact relationships.
- Implementation files/modules and functions/classes explain their responsibilities. Final ML documents five actual key classes and demonstrated patterns.
- Optimization documents preserve unoptimized baselines, changes, comparable measurements, correctness and annotated execution views.
- Evidence includes successful and failed/retried runs, limitations and justified capability-equivalent tool choices.

## References

The public coursework workbook contains the [mini-coursework rubric](https://docs.google.com/spreadsheets/d/1QXWe4s9Mu2C6-aafwaJfciQS79LsRciDQjPu5DD-aL8/edit#gid=879988496) and [final ML rubric](https://docs.google.com/spreadsheets/d/1QXWe4s9Mu2C6-aafwaJfciQS79LsRciDQjPu5DD-aL8/edit#gid=1745721073). Both informed these plans. Course tool names are examples; preserve the underlying capability and proof requirements when choosing alternatives.

The [marketplace gateway note](https://github.com/phuchoang2603/refurbished-marketplace/blob/main/docs/ecommerce-fraud-gateway.md) defines commerce context; the [architecture guide](../architecture/payment-gateway.md) records the agreed gateway refinements and outstanding retry mismatch. The [platform ownership guide](https://github.com/phuchoang2603/talos-proxmox/blob/main/apps/README.md) identifies shared infrastructure responsibilities.

The reference repository's mini-coursework documents were reviewed for evidence presentation: pair claims with code, repeatable commands, annotated captures and explanations of limitations. Its recommendation entities, technology choices, deployment results and current omissions are not this project's requirements or completion evidence. In particular, its current catalog description is not a reason to omit our required DP1-DP3 job/table lineage.

- [README.md](https://github.com/itsmekhoathekid/RecSys-MLops/blob/main/docs/submission/rubic-%28mini-coursework%29/README.md)
- [data_generator.md](https://github.com/itsmekhoathekid/RecSys-MLops/blob/main/docs/submission/rubic-%28mini-coursework%29/data_generator.md)
- [data_governance.md](https://github.com/itsmekhoathekid/RecSys-MLops/blob/main/docs/submission/rubic-%28mini-coursework%29/data_governance.md)
- [data_pipeline_orchestration.md](https://github.com/itsmekhoathekid/RecSys-MLops/blob/main/docs/submission/rubic-%28mini-coursework%29/data_pipeline_orchestration.md)
- [data_storage.md](https://github.com/itsmekhoathekid/RecSys-MLops/blob/main/docs/submission/rubic-%28mini-coursework%29/data_storage.md)
- [docker.md](https://github.com/itsmekhoathekid/RecSys-MLops/blob/main/docs/submission/rubic-%28mini-coursework%29/docker.md)
- [novel_ideas.md](https://github.com/itsmekhoathekid/RecSys-MLops/blob/main/docs/submission/rubic-%28mini-coursework%29/novel_ideas.md)
- [processing_jobs.md](https://github.com/itsmekhoathekid/RecSys-MLops/blob/main/docs/submission/rubic-%28mini-coursework%29/processing_jobs.md)
- [schema_design.md](https://github.com/itsmekhoathekid/RecSys-MLops/blob/main/docs/submission/rubic-%28mini-coursework%29/schema_design.md)

## Documentation validation

Documentation checks completed on 2026-09-19:

| Check | Result |
| --- | --- |
| Both coursework rubrics and roadmap review | Required capabilities/proof mapped, including mini unoptimized baselines and DP1-DP3 governance; final test thresholds, distributed training, incremental versions, both writers and all ten required CI/CD workloads. |
| Local references | 57 relative file links/anchors resolved across README and the four new guides. |
| Roadmap references | 40 distinct issue references matched the fetched GitHub issue inventory; project architecture reviewed. |
| External reference access | Workbook, marketplace notes/code, platform ownership and project pages accessible; all nine linked reference mini-coursework Markdown documents fetched and reviewed. |
| Diagrams | All three embedded Mermaid diagrams rendered to SVG with Mermaid CLI 11.17.0; PNG views inspected for labels/flows, then the processor exchange simplified and live-event ingestion made explicit; final SVG render passed. |
| Planning validation | `devenv shell -- openspec validate document-gateway-architecture-and-coursework --strict` passed; behavioral specs are intentionally skipped. |
| Change scope and whitespace | README, new architecture/coursework guides and selected OpenSpec planning files only; existing runtime, deployment, marketplace and capability-spec files unchanged. Whitespace checks passed, including new files. |

To repeat diagram rendering, run `mmdc -i docs/architecture/payment-gateway.md -o /tmp/payment-gateway-rendered.md` with Mermaid CLI available. Rendered review files are temporary; the embedded Mermaid is the maintained source.

These checks verify documentation coverage only. No application tests, deployment, live platform verification or coursework runtime acceptance were performed. Implementation proof remains planned, and PG-01 was not closed remotely.
