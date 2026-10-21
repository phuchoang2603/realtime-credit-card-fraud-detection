## 1. Agreement and native tooling

- [x] 1.1 Keep the ODCS source agreement and its SQL quality rules, and replace wrapper-specific server assumptions with a job-owned data source.
- [x] 1.2 Remove the custom Python package, Arrow mapper, CLI, validation wrapper, tests, lockfile and packaging links.
- [x] 1.3 Call the pinned native CLI directly from CI for agreement lint and generated schema-reference drift.

## 2. Execution boundaries and delivery

- [x] 2.1 Document downstream responsibilities, the native validation command, SQL dialect limits and deferred Spark/Flink/DataHub integration without a parallel rule suite.
- [x] 2.2 Verify native lint, documentation generation and failure detection; exercise native data validation on representative valid and invalid data, validate OpenSpec, and commit and update the independent foundation PR.
