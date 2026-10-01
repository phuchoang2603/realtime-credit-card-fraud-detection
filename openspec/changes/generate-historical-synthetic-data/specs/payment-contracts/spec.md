## ADDED Requirements

### Requirement: Golden lifecycle histories

The contracts SHALL include versioned golden payment histories encoded as protobuf JSON, each marked valid or invalid with the violated rule. They SHALL cover every legal transition and at least each prohibited transition named by the lifecycle requirement. Every producer's history validator SHALL accept all valid and reject all invalid golden histories.

#### Scenario: Producer validator conformance

- **WHEN** a producer's validator runs against the golden histories
- **THEN** it accepts each valid history and rejects each invalid history for its recorded rule
