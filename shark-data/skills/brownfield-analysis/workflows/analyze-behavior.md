# Behavior Analysis

## Purpose

Document what the system *does* — its business logic, workflows, decision rules, and how it
handles errors. This is where code structure becomes business understanding. Approach it from
the perspective of someone who needs to understand the *business* the code implements, not
just the code structure.

## Business logic

In `behavior/business-logic.md`, identify and document each business domain the system
handles:

- Domain name and description
- Key business rules
- Core operations within the domain
- Business constraints and invariants
- Cross-domain interactions

## Workflows

In `behavior/workflows.md`, document end-to-end process flows — user-initiated (e.g. "user
creates a statement"), system-initiated (e.g. "nightly batch recalculation"), and integration
(e.g. "data feed ingestion"). For each:

- Trigger — what starts it
- Steps in sequence — which classes/services participate
- Data transformations at each step
- Branching logic — where the flow can diverge
- Terminal states — how it ends, on success and on failure
- A Mermaid sequence diagram

## Decision logic

In `behavior/decision-logic.md`, document non-trivial business rules and decision trees:

- State machines, if any
- Business rule engines or rule sets
- Conditional logic that determines system behavior
- Configuration-driven behavior
- Feature flags or toggles

For complex branching, use Mermaid flowcharts to make the logic visible.

## Error handling

In `behavior/error-handling.md`, document:

- Exception hierarchy — the class tree of custom exceptions
- Error codes and their meanings
- Error response formats — what users/callers see
- Recovery patterns — retries, fallbacks, dead-letter queues
- Validation patterns — where and how input is validated
- Logging patterns — what gets logged, at what level
