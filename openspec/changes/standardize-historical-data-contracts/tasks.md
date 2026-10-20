## 1. Contract and schemas

- [x] 1.1 Define the source ODCS contract, including physical schema evolution and SQL quality rules, and verify official schema lint passes.
- [x] 1.2 Package the contract and derive Arrow schemas without duplicate definitions; verify nested types, version rejection and installed-wheel resources.

## 2. Validation and evidence

- [x] 2.1 Execute shared native contract checks over staged Parquet and emit structured success/failure reports; verify identical duplicates, conflicts, lifecycle errors and label linkage.
- [x] 2.2 Verify exact physical schemas before normalization and fail on unsupported or missing checks; cover wrong widths, missing fields, empty inputs and unexecuted rules.

## 3. Documentation and delivery

- [x] 3.1 Generate the contract reference from ODCS, add minimal contributor commands and verify documentation drift is rejected.
- [x] 3.2 Wire locked contract checks into CI, verify focused tests, lint, packaging and OpenSpec validation, and remove unused draft implementations or compatibility paths.
- [ ] 3.3 Commit the foundation sequentially and open an independent PR against main, recording the generator rewrite and future DataHub integration as follow-up scope.
