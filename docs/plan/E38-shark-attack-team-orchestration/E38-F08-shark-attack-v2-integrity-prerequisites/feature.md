---
feature_key: E38-F08-shark-attack-v2-integrity-prerequisites
epic_key: E38
title: Shark Attack v2 Integrity Prerequisites
description: Capture Phase 0 prerequisites from dev-artifacts/shark-attack-v2-plan/implementation-plan.md: non-mutating plan resolution, tech-debt related-document parity, research prompt/validator consistency, one tech-debt implementation gate, and the nested-worktree regression. Link rather than duplicate narrower E36-F04 and resolved TD-045/TD-048 coverage. The plan's open semantic decisions remain unapproved.
---

# Shark Attack v2 Integrity Prerequisites

**Feature Key**: E38-F08-shark-attack-v2-integrity-prerequisites

## Triage breadcrumb

The Shark Attack v2 plan identifies several existing Shark behaviors that would
undermine later coordination and release evidence: advisory planning can persist
simulated transitions, tech-debt entities lack related-document command parity,
research prompts can contradict the universal validator, and the tech-debt
workflow can dispatch duplicate implementation/full-gate work.

Capture these Phase 0 prerequisites as one integrity tranche before the v2
coordination protocol is qualified. Preserve keyed `shark next` behavior and the
already-repaired nested-worktree exclusion, adding only its focused regression.

Source: `dev-artifacts/shark-attack-v2-plan/implementation-plan.md`, especially
SA-001, SA-002, SA-004, SA-005, SA-006, Tranche A, and Phase 0.

Existing coverage to link rather than duplicate:

- E36-F04 owns portfolio-aware read-only advice.
- TD-045 resolved plan/next resolver duplication.
- TD-048 resolved narrower plan/next test gaps.

The proposed read-only `shark plan` semantic change remains an open decision.
This triage record does not approve it or any implementation design.
