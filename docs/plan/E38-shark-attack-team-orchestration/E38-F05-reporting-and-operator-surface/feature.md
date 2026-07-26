---
feature_key: E38-F05-reporting-and-operator-surface
epic_key: E38
title: Deterministic Council Artifacts and Operator Handoffs
description: Establish deterministic, validated council artifacts and thin Shark admin tooling for operator handoffs, without introducing a team-run orchestration or reporting runtime.
---

# Deterministic Council Artifacts and Operator Handoffs

## Purpose

E38-F05 owns the Shark Attack v2 deterministic council-artifact contract. It
makes the operator handoff, decision, escalation, resolution, and
batch-observation artifacts machine-validatable and safely generated, so a
human operator can rely on stable identifiers, scope, role routing, evidence,
timestamps, and revision history.

The original lightweight guidance is retained as the operator-facing outcome,
but prose alone is not the feature's implementation boundary. The approved
v2 triage assigns the following capabilities to F05:

- typed council-artifact data with artifact type, immutable identifier, role
  references, scope, evidence, timestamps, and `supersedes`;
- project-confined reading, listing, and atomic create-without-overwrite under
  `docs/council`;
- service-level validation and generation, including revision rules,
  effective-roster role checks, evidence confinement, and execution-wave
  ownership validation;
- thin `shark admin council` commands to create and validate artifacts and
  validate execution waves; and
- schemas, templates, documentation, fixtures, and contract tests that make
  the format usable both from the authored Shark Attack skill and its embedded
  Shark-data copy.

## Boundaries

F05 is data tooling and validation, not a team-run orchestration, reporting,
ledger, preview, start, resume, or summary runtime. Existing Shark lifecycle,
claim, note, history, telemetry, and Rider dispatch surfaces remain their own
owners. F05 must preserve parent-only Shark lifecycle mutation and should not
make production loaders traverse project `docs/council` merely to load the
embedded skill bundle.

The v2 plan proposes a tagged entity-or-collection scope, immutable
`supersedes` revisions, repository-relative evidence restricted to the
project root, and a default Technical Director closeout role. Those are
implementation decisions to validate during research and specification; this
feature brief does not silently approve the plan's open namespace, schema,
closeout-role, or migration choices.

## Relationships and sequencing

F05 follows the completed F04/F06/F07 foundations and the v2 integrity
prerequisites in F08. Its deterministic artifact contract is consumed by the
provider-neutral coordination and lifecycle qualification work in F09 and
F11. The interaction map is the authoritative cross-feature interface record.
