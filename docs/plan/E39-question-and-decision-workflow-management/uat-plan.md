---
epic: E39
title: Question and Decision Workflow Management UAT Plan
date: 2026-07-30
---

# E39 UAT Plan

| UAT | Requirements | Acceptance description |
|---|---|---|
| UAT-01 — Create, discover, link | F-001, F-005, F-006, NF-002 | Create blocking `Q###`, link it to one feature and one task, then retrieve/list/search and query by recipient/blocked entity. Key, state, owner, links, and history are visible without parsing notes. |
| UAT-02 — Serial response and lease | F-002–004, NF-003 | Configure three responders. Only first dispatches and one claim succeeds. Valid bounded response plus release exposes exactly next; failure, expiry, or release never completes a responder. |
| UAT-03 — Direct scoped gate | F-005, F-006, F-009, NF-003 | Keyed dispatch and supported advance of linked work stop with compact summary; unlinked peer and nonqualifying Question remain eligible; neither record is mutated. |
| UAT-04 — Consequential provenance | F-007, F-008, NF-002 | Feature-changing answer resolves only after authoritative feature-spec pointer; kind, pointer, and history remain queryable. |
| UAT-05 — Safe bounded context | NF-001, F-002 | Prompt, transcript, credential, or oversized input is rejected before mutation and directs to a note/pointer; full response is restricted. |
| UAT-06 — Follow-up without mutation | F-008, F-009, NF-002 | Link created/existing Shark work for an implementation finding; Question records outcome/history and linked item gets no Question-owned claim/transition. |

## Cross-feature and cross-epic scenarios

| Interaction | Acceptance |
|---|---|
| I-01 | Persisted `Q###` uses generic registry/key/claim/history path and receives one valid responder dispatch. |
| I-02 | Validated open state and first-pending responder are the only state presented to gate/query consumers; resolution removes predicate without linked status mutation. |
| I-03 | Focused views expose safe state; blocked owners receive compact handoff only. |
| X-06 | E38-F09 later consumes public Question lifecycle/query/dispatch rather than a host queue. Proposed until decomposition assigns owner/coverage. |

## Non-functional evidence

- Service authorization checks requester, current responder, and resolution owner; blocked-work callers cannot retrieve full response content.
- Transaction/repository tests prove response, provenance, history, and links are consistent; race tests prove one active claimant.
- Migration tests cover fresh install and upgrade. Compatibility tests cover existing keys, generic `blocks`, ordinary workflows, and unlinked dispatch.
- Focused queries are indexed; no recursive graph or global scan occurs on every dispatch. Acceptance needs automated coverage, migration/CLI/API contracts, and integrated production-path demonstration.
