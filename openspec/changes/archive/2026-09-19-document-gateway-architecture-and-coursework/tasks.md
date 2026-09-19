## 1. Coursework inventory and references

- [x] 1.1 Review both rubrics and identify required capabilities and proof for the coursework coverage plans; group related requirements where useful without requiring source snapshots, row preservation, points accounting, or provenance metadata.
- [x] 1.2 Record reference links for the example repository's mini-coursework documents, the marketplace gateway note, PG-01 and baseline acceptance #67; verify links and distinguish reference examples from this repository's implementation evidence.

## 2. Architecture documentation

- [x] 2.1 Write the current-state inventory in `docs/architecture/payment-gateway.md` with source links for inference, caller-provided terminal features, CI, deployment and telemetry; verify claims against files and explicitly separate code/configuration presence from live operational proof.
- [x] 2.2 Document target Go/Python service responsibilities, owned data, marketplace/shared-platform boundaries, integration-scoped buyer/seller identity and authorized merchant provisioning; verify all accepted boundaries from PG-01 are represented and remaining tool choices are marked deferred.
- [x] 2.3 Document event-sourced payment authority, idempotency/concurrency, projections, financial journal distinction, processor uncertainty/recovery and side-effect-free replay; verify that stored risk results are replayed and the marketplace failed-session retry mismatch is assigned to PG-04 rather than silently resolved.
- [x] 2.4 Add embedded target deployment, checkout/recovery and data/ML diagrams with numbered labeled arrows and explicit external/ownership boundaries; verify rendering and that major deployment nodes are independently deployable workloads/infrastructure rather than SDKs or domain classes.
- [x] 2.5 Document historical/live simulation, per-attempt training rows, delayed labels, information-availability constraints and the mini-to-final asset handoff; verify alignment with generator/feature tickets and the baseline acceptance #67 -> training acceptance #53 dependency.

## 3. Rubric coverage and evidence plan

- [x] 3.1 Create `docs/coursework/mini-coursework.md` organized by capability, with ticket links, expected artifacts, required proof and status; verify capability coverage including unoptimized benchmarks, DP1-DP3 governance, schema requirements and two baseline novelty proofs.
- [x] 3.2 Create `docs/coursework/final-ml.md` using the same columns; verify all required capabilities are mapped, including both streaming writers, distributed training, incremental data versions, each CI/CD workload, greater-than-90% coverage and greater-than-80% mutation score.
- [x] 3.3 Finish the coursework guide with documentation requirements, planned/existing-unverified/verified evidence conventions and versioned baseline handoff/regression expectations; verify no future artifact is presented as existing evidence and shared work is cross-referenced consistently between rubrics.

## 4. Navigation and acceptance

- [x] 4.1 Update README business summary, actual repository structure, table of contents and documentation links; verify it accurately separates the existing fraud foundation from the planned gateway and retains access to historical diagrams/demo material.
- [x] 4.2 Review all deliverables against issue #29 and the accepted project architecture; verify local links, diagram rendering, coursework capability coverage and ticket mappings, and record the checks in the coursework guide without claiming runtime completion.
- [x] 4.3 Run `openspec validate document-gateway-architecture-and-coursework --strict` and inspect the documentation diff; verify no application, deployment, marketplace or existing capability-spec files were changed as part of PG-01.
