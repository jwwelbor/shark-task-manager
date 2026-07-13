# Migration Readiness

## Purpose

If the codebase needs to be migrated, modernized, or rewritten, produce the actionable
roadmap that makes that effort safe and ordered.

## Component order

In `migration/component-order.md`, define a dependency-ordered migration sequence:

- Which components can be migrated independently
- The dependency chain that determines order
- A breakdown of the recommended sequence with rationale

This sequence describes the *migration work* — the order in which components should be
tackled during the actual migration effort. It is a domain output, not a description of how
this analysis area was performed.

## Test specifications

In `migration/test-specifications.md`, define the test coverage needed before and during
migration:

- An assessment of current test coverage
- Required test types per component — unit, integration, end-to-end
- Recommended test frameworks
- Test data requirements

## Validation criteria

In `migration/validation-criteria.md`, define how to verify each migration step succeeded:

- Functional acceptance criteria per component
- Performance benchmarks to maintain
- Data integrity checks
- Rollback procedures
