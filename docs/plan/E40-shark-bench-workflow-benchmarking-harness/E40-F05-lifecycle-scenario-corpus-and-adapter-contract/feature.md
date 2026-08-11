---
feature_key: E40-F05-lifecycle-scenario-corpus-and-adapter-contract
epic_key: E40
title: Lifecycle scenario corpus and adapter contract
description: Extend the completed v1 corpus into versioned feature, bug, change-card, and tech-debt lifecycle scenarios on a controlled Python fixture. Define the stage-applicability matrix, scenario package schema, fixture and adapter identity, evaluator references, and the language-neutral adapter boundary. Consumes I-01. Produces I-04.
---

# Lifecycle scenario corpus and adapter contract

**Feature Key**: E40-F05 · **Size**: M · **Execution order**: 5

## Outcome

Benchmark maintainers can load reproducible feature, bug, change-card, and
tech-debt scenarios against one controlled Python fixture without embedding
Python-specific commands in Shark's workflow contracts.

## Scope

- Define one versioned scenario-package schema with a stable scenario ID,
  scenario version, entity family, stage-applicability matrix, fixture SHA,
  adapter version, input references, replay references, and evaluator references.
- Add one admitted seed for each of the four lifecycle families.
- Add a controlled Python task-manager fixture with a language-specific
  execution adapter. Preserve the existing Go fixture as a compatibility
  adapter and low-cost harness regression surface.
- Define the agent-visible initial input for each entity family and the
  evaluator-only reference and execution-oracle references without exposing
  their contents at dispatch time.
- Reject a scenario that lacks a runnable base fixture, an applicable-stage
  declaration, or a machine-checkable final predicate.

## Acceptance boundary

- The loader accepts all four admitted scenario families and rejects malformed
  packages with the failing field named.
- Re-loading the same package yields the same scenario identity, fixture
  identity, stage matrix, and referenced inputs.
- A generic lifecycle component can select commands through the adapter without
  knowing the fixture language or toolchain.
- Feature scenarios declare the D01-D05 prelude applicable. Bug, change-card,
  and tech-debt scenarios declare those stages explicitly non-applicable.

## Contracts

- **Consumes I-01**: reuse the v1 manifest, admission, and held-back-oracle
  principles; do not require the v1 Go schema to become the global format.
- **Produces I-04 — Lifecycle scenario package**: the canonical scenario input
  consumed by E40-F06, E40-F07, and E40-F08. The authoritative shape lives in
  [architecture.md](../architecture.md#lifecycle-scenario-package-contract).

## Out of scope

- Driving Shark entities through their workflows.
- Replaying stakeholder or research interactions.
- Evaluating generated artifacts or publishing a baseline.

## Workflow handoff

The feature workflow must research and specify the package schema, fixture,
admission rules, and adapter interface before it generates implementation tasks.
It may refine these boundaries, but it must update I-04 and every named consumer
in the epic interaction map when the shape changes.

*Last updated: 2026-08-11*
