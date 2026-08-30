---
type: uat-plan
epic: E34
last_updated: 2026-08-30
---

# E34 acceptance plan

## Goal

Verify that E34 makes quality-policy handoffs durable, safe, and reusable
without changing parent-loop authority or making E40 benchmark work an E34
delivery dependency.

## Prerequisites

- A Shark project with the E34 canonical bundle installed.
- Fixtures for SIMPLE, STANDARD, and COMPLEX feature routes.
- A controlled parent-loop session that can return valid, malformed, and replayed gate results.
- A project data root with regular, redundant, legacy, changed, and orphaned override fixtures.

## Acceptance scenarios

| ID | Success criteria | What to verify |
|----|------------------|----------------|
| UAT-01 Persist a failed gate before routing | Epic criteria 2; REQ-F-009 through REQ-F-012 | A configured gate result with findings, evidence, a sweep, and a valid kickback is validated, bound to the parent session, and persisted before the configured route runs. A malformed envelope, conflicting replay, or incomplete persistence cannot advance the entity. |
| UAT-02 Preserve parent authority | Epic criteria 2; REQ-NF-003 | The worker result contains no trusted entity, source-status, lease, or transition authority. The parent supplies those values, validates the configured opaque outcome, and records the result before any lifecycle action. |
| UAT-03 Close a defect class | Epic criteria 3; REQ-F-013 through REQ-F-016 | A class sweep enumerates its complete declared scope, records every instance and disposition, reports zero open instances before closure, and verifies a structural guard. A later finding is classified from fingerprint and prior scope, not review-round count. |
| UAT-04 Propagate a material decision | Epic criteria 4; REQ-F-017 through REQ-F-020 | A changed Question, ADR, design, state, or debt decision yields an impact set with affected artifacts, consumer caller paths, acceptance criteria, and regression coverage. Each item is amended or has a linked follow-up. |
| UAT-05 Apply the tier matrix | Epic criteria 5; REQ-F-021 and REQ-F-022 | SIMPLE, STANDARD, and COMPLEX fixtures render only their required artifacts and gates. Each required gate records command, working directory, exit status, runner-native counts, expected and unexpected skips, and a bounded evidence pointer. |
| UAT-06 Close the accumulated epic diff | Epic criteria 6; REQ-F-023 through REQ-F-025 | `integration_review` examines the resolved accumulated diff, open I-##/X-## edges, findings, sweeps, guards, decisions, standards, and predicted debt. Its pass result cannot replace a failed required feature verdict. |
| UAT-07 Inspect overrides safely | Epic criteria 7; REQ-F-026 through REQ-F-028 | Override status and upgrade dry-run deterministically classify each eligible path as `current`, `upstream_changed`, `identical_redundant`, `orphaned`, or `baseline_unknown`; they neither modify nor print override content. |
| UAT-08 Account for adoption work | Epic criteria 8; REQ-F-029 through REQ-F-031 | The E34 plan assigns every E04 proposal and WWGM override a disposition, links or resolves CC-007 and CC-008 without duplicates, and records E40 scenarios only as later validation. |
| UAT-09 Preserve reusable policy handoffs | Epic criterion 1; REQ-F-001 through REQ-F-008 | The registered I-## map resolves every producer, consumer, shape, payload, style, and shared verification pointer. Rendered prompts and layered skill consumers preserve the documented harness and evidence boundaries. A material unresolved decision creates or reuses a linked Question whose resolution points to the narrowest authoritative record. |

## Cross-feature and cross-epic scenarios

| ID | What to verify |
|----|----------------|
| I-01 | A staged demo handoff preserves independent assessor verdict, owner decision, open conditions, gate mode, activation owner, closure key, live counterpart status, review basis, and demonstrability disposition. |
| I-02 through I-04 | GateResult carries the required bounded evidence; defect sweeps and decision-impact sets remain traceable into final review. |
| I-05 | Final integration review produces a canonical adoption manifest with changed paths, bundle digest, compatibility notes, and validation evidence for override adoption. |
| X-14 | E40 can identify canonical and reconciled project-style configurations from I-05 and baseline digests when its harness is ready. Missing E40 coverage leaves X-14 proposed and does not block E34 acceptance. |

## Performance, reliability, and security checks

- Verify deterministic ordering and stable JSON fields for gate results and override status.
- Verify exact replay succeeds without duplicate notes, kickbacks, or transitions; verify conflicting replay fails closed.
- Verify partial persistence resumes only missing idempotent operations.
- Verify bounded fields reject oversized content, transcripts, rendered prompts, credentials, and forbidden markers without echoing them.
- Verify override handling rejects absolute, escaping, symlinked, and non-regular paths; reports only relative paths and digests.
- Verify projects without overrides retain compatible upgrade behavior and zero-valued summary counts.

## Result

Accept E34 only when UAT-01 through UAT-09 pass, every I-## contract resolves to
the architecture, and X-14 remains accurately proposed until E40 adds its
coverage pointer. Route any material discrepancy through the existing Question
or council workflow; do not create a new Question for a settled design choice.
