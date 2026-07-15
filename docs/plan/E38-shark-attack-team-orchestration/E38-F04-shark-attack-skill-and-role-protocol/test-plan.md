# Test Plan: E38-F04 - Shark Attack Skill and Role Protocol

> Direction reset note (2026-07-13): this plan was written against the former
> runtime decomposition. Use it as a content-quality checklist only; the active
> acceptance gate is `../../uat-plan.md`, and no scheduler/ledger/aggregate
> implementation is required.

**Created:** 2026-07-13
**Feature PRD:** `docs/plan/E38-shark-attack-team-orchestration/E38-F04-shark-attack-skill-and-role-protocol/feature.md`
**Task Spec:** `docs/plan/E38-shark-attack-team-orchestration/E38-F04-shark-attack-skill-and-role-protocol/spec.md` (no separate F04 task spec exists; this is the authoritative implementation specification for planning)
**Status:** APPROVED WITH TOOLING EXCEPTION

## Spec Drift Analysis

### Drift Findings

No semantic drift was found between the feature description and `spec.md`.
The feature description names the distributable skill, roster, council memory,
communication, escalation, and role-aware pull; the specification expands each
into REQ-F-001 through REQ-F-012 and AC-001 through AC-010 without adding an
unrelated runtime or database scope.

The requested task-spec comparison is limited because F04 has no child task
specifications yet. `spec.md` is used as both the feature PRD and task-level
source for this pre-development plan. Before implementation, child tasks must
reference the ACs below and the observability requirements in this document.

### Traceability Matrix

| Feature requirement | Acceptance criterion / test coverage | Covered? | Notes |
|---|---|---|---|
| REQ-F-001 chair, IDs, responsibilities, communication, escalation, model preference, workflow precedence | AC-001, AC-002, TC-101, TC-102 | Yes | Invalid authority-bearing responsibility is rejected. |
| REQ-F-002 roster contract | AC-001, AC-002, TC-101, TC-102 | Yes | Required fields, uniqueness, mappings, optional model tier. |
| REQ-F-003 council layout and refreshed-worker continuity | AC-008, TC-103, TC-109 | Yes | Includes privacy/gitignore and required resume pointers. |
| REQ-F-004 bounded inbox lifecycle | AC-003, TC-001, TC-104 | Yes | Message shape and durable copy are asserted. |
| REQ-F-005 bounded artifacts and secret exclusion | AC-004, TC-002, TC-105 | Yes | All four artifact types and conflict behavior included. |
| REQ-F-006 escalation fallback | AC-005, TC-106 | Yes | Missing policy routes to council-review and pause/review. |
| REQ-F-007 workflow-authoritative role pull | AC-006, TC-107, TC-003 | Yes | Shared X-03 contract. |
| REQ-F-008 worker/root ownership boundary | AC-007, TC-108 | Yes | Root lease/status mutation is forbidden. |
| REQ-F-009 embedded bundle and replace-only override | AC-009, TC-103, TC-004 | Yes | Shared X-05 contract. |
| REQ-F-010 missing product context/capability fallback | AC-010, TC-110 | Yes | Ordinary `/run` behavior is a regression assertion. |
| REQ-F-011 stable roster/persona mapping | AC-001, TC-101 | Yes | Existing persona files are resolved, not duplicated. |
| REQ-F-012 resume context | AC-003, AC-008, TC-104, TC-109 | Yes | Decisions, handoffs, escalations, inbox state. |
| REQ-NF-001 validation diagnostics | AC-001, AC-002, TC-101, TC-102 | Yes | Field/path diagnostics are exact assertions. |
| REQ-NF-002 security boundaries | AC-002, AC-004, AC-008, AC-009, TC-002, TC-004, TC-102, TC-103 | Yes | Secret, path, symlink, and authority attack classes. |
| REQ-NF-003 idempotent reads/acks/writes | AC-003, AC-004, TC-001, TC-104, TC-002, TC-105 | Yes | Identical replay succeeds; conflicting ID fails. |
| REQ-NF-004 no second runtime/claim store | AC-006, AC-007, AC-009, TC-003, TC-004, TC-107, TC-108 | Yes | Static/content and caller-path checks. |
| REQ-NF-005 structured downstream metadata | I-04, TC-001, TC-002, TC-104 | Yes | Shared contract asserts shape, paths, and bounded content. |

## Acceptance Criteria Review

All ten acceptance criteria are testable and trace to the requirements above.
The terms “understandable”, “actionable”, and “appropriate” are made concrete
in the test cases: required field names, exact root/child keys, allowed route
`council-review`, pause/review recommendation, and diagnostic field/path
locations. No AC remains an open-ended security assertion; the attack classes
are enumerated in TC-002, TC-004, TC-102, TC-103, TC-107, and TC-108.

No feature requirement is missing from the matrix. Cross-feature I-04 and both
consumed cross-epic rows X-03 and X-05 have shared test pointers below.

## ISTQB Technique Application (per AC)

| AC | Technique(s) applied | Test cases generated | Rationale |
|---|---|---|---|
| AC-001 | Equivalence Partitioning + Contract Surface Enumeration | TC-101 | Valid complete roster and each required/optional role contract. |
| AC-002 | Equivalence Partitioning + Attack-class Enumeration + BVA | TC-102 | Missing, duplicate, empty, unknown, unsafe, and authority-bearing values. |
| AC-003 | State Transition + Contract Surface Enumeration | TC-001, TC-104 | Message unread → acknowledged/removed while durable result remains. |
| AC-004 | Contract Surface Enumeration + Attack-class Enumeration | TC-002, TC-105 | Four artifact types, required fields, secret classes, replay/conflict. |
| AC-005 | Decision Table + State Transition | TC-008 | Policy present/absent and question resolved/unresolved combinations. |
| AC-006 | Decision Table + Contract Surface Enumeration | TC-107, TC-003 | Workflow eligibility, ordering, legacy/model inputs, claim path. |
| AC-007 | Attack-class Enumeration + Contract Surface Enumeration | TC-108 | Worker authority bypass classes and production handoff shape. |
| AC-008 | Equivalence Partitioning + Attack-class Enumeration | TC-103, TC-104 | Committed/private/ignored content and refresh state classes. |
| AC-009 | State Transition + Contract Surface Enumeration | TC-103, TC-004 | Init/upgrade/override precedence and unrelated skill preservation. |
| AC-010 | Decision Table + Equivalence Partitioning | TC-110 | Product gate/team capability combinations and fallback choices. |

## ISO 25010 Coverage Matrix

| AC | Functional suitability | Performance efficiency | Compatibility | Usability | Reliability | Security | Maintainability | Portability |
|---|---|---|---|---|---|---|---|---|
| AC-001 | ✅ TC-101 | N/A: schema validation is not latency-sensitive | ✅ TC-101 | ✅ TC-101 diagnostics | N/A | ✅ TC-101 | ✅ TC-101 fixture-driven validator | ✅ TC-101 relative paths |
| AC-002 | ✅ TC-102 | N/A | N/A | ✅ TC-102 field/path errors | N/A | ✅ TC-102 | ✅ TC-102 table-driven invalid classes | ✅ TC-102 path rules |
| AC-003 | ✅ TC-001, TC-104 | N/A | ✅ TC-104 refreshed-worker files | ✅ TC-001 message lifecycle | ✅ TC-104 replay | ✅ TC-001 bounded body | ✅ TC-104 typed artifacts | ✅ TC-104 filesystem contract |
| AC-004 | ✅ TC-002, TC-105 | N/A | ✅ TC-105 artifact format | ✅ TC-002 actionable errors | ✅ TC-105 idempotency | ✅ TC-002 secret rejection | ✅ TC-002 schema fixtures | ✅ TC-002 atomic file behavior |
| AC-005 | ✅ TC-008 | N/A | N/A | ✅ TC-008 visible next action | ✅ TC-008 unresolved state | ✅ TC-008 no fixed human destination | ✅ TC-008 decision table | N/A |
| AC-006 | ✅ TC-107, TC-003 | ✅ TC-003 ordering/claim boundedness | ✅ TC-003 E19 contract | ✅ TC-107 role explanation | ✅ TC-003 duplicate-claim prevention | ✅ TC-107 authority separation | ✅ TC-107 production caller test | N/A |
| AC-007 | ✅ TC-108 | N/A | ✅ TC-108 scheduler boundary | ✅ TC-108 evidence semantics | ✅ TC-108 root remains live | ✅ TC-108 root mutation denial | ✅ TC-108 ownership contract | N/A |
| AC-008 | ✅ TC-103, TC-104 | N/A | ✅ TC-103 gitignore/override | ✅ TC-104 resume pointers | ✅ TC-104 refresh continuity | ✅ TC-103 private content boundary | ✅ TC-104 documented layout | ✅ TC-103 relative paths |
| AC-009 | ✅ TC-103, TC-004 | N/A | ✅ TC-004 embedded/override compatibility | ✅ TC-004 setup guidance | ✅ TC-103 upgrade preservation | ✅ TC-004 replace-only boundary | ✅ TC-004 bundle fixture | ✅ TC-004 embedded filesystem |
| AC-010 | ✅ TC-110 | N/A | ✅ TC-110 ordinary `/run` regression | ✅ TC-110 actionable fallback | ✅ TC-110 no silent mode change | ✅ TC-110 no guessed decisions | ✅ TC-110 decision table | ✅ TC-110 capability variants |

There are no unresolved AC × characteristic coverage gaps. The full feature
will still require the parent UAT's operational telemetry checks where the
content-only protocol hands off to F02/F03/F05 runtime components.

## Observability Design (per behavior)

F04 is primarily an embedded content and file-protocol feature. It must not
invent a second telemetry runtime. For file-only behaviors, structured audit
metadata and bounded logs are the runtime evidence; sensitive body content and
prompts must never be logged.

| Behavior | Metric | Log | Trace span | Alert threshold | Test assertion |
|---|---|---|---|---|---|
| Roster/bundle validation | `sharkdata.validation.count{skill="shark-attack",result}` if existing validation metrics are available; otherwise N/A | `sharkdata.validation` with skill, result, issue_count, and path | Existing validation span if caller provides one | N/A | TC-101/102 validates result, diagnostics, and absence of secrets |
| Inbox acknowledgement guidance | N/A — F04 adds no runtime | N/A — no new runtime log | N/A | N/A | TC-001/104 verifies the durable-copy and redaction instructions |
| Artifact replay/conflict guidance | N/A — F04 adds no runtime | N/A — no new runtime log | N/A | N/A | TC-002/105 verifies the documented idempotency and conflict rule |
| Escalation routing guidance | N/A — F04 adds no runtime | N/A — no new runtime log | N/A | N/A | TC-106 verifies `council-review`, unresolved status, and next-action guidance |
| Role pull handoff | Existing sprint/claim metrics; F04 adds no duplicate metric | Existing sprint/claim log fields: agent type, entity key, result; no roster tier | Existing sprint/claim spans | Existing claim alerts | TC-107/003 asserts production `GetNextTask(ctx, agentType)` and claim boundary |
| Worker evidence/root ownership | Existing team-run outcome metrics | `team.worker_evidence_returned` with child_key, semantic outcome, root_key; no status mutation | Existing scheduler span | Existing workflow/claim alerts | TC-108 checks evidence output and zero root mutation calls |
| Bundle override resolution | Existing shark-data upgrade/validation result if available | `sharkdata.override_resolved` with relative path and result | Existing resolver span | N/A | TC-103/004 asserts only target skill is replaced and unrelated skills survive |

**Implementation hook:** F04 adds no observability runtime. Its content tests
must keep the protocol explicit about redaction and the owning entrypoints'
existing observability remains authoritative. F04 must not add credentials,
rendered prompts, unrestricted worker output, a new claim store, or a second
workflow engine.

## Cross-feature contract tests (I-04)

The same tests are referenced by E38-F02, E38-F03, and E38-F05. They must live
at the exact pointers declared in `spec.md`; no consumer-specific twin tests.

| I-## | Producer | Consumers | Shape source | Contract test pointer | TC |
|---|---|---|---|---|---|
| I-04 | E38-F04 | E38-F02, E38-F03, E38-F05 | E38 `architecture.md` §4.5 Council communication contract | `tests/contracts/e38_f04_interactions_test.go#TC-001` canonical shared contract pointer; TC-002 adds durable artifact lifecycle coverage | TC-001, TC-002, TC-104, TC-105 |

### TC-001: I-04 shared inbox message shape and acknowledgement

**Feature Requirement:** REQ-F-004 / I-04.
**Acceptance Criterion:** AC-003.
**Technique:** State Transition + Contract Surface Enumeration.
**ISO 25010:** Functional Suitability, Compatibility, Reliability, Security.

**Caller-Path Contract:**

- **Entrypoint:** The embedded `shark-attack` `message-schema.md` and
  `communicate.md` protocol documents, as resolved through the production
  shark-data bundle.
- **Lowest allowed mock seam:** None; read the embedded content directly.
- **Forbidden mocks:** Do not replace embedded resolution or assert an F04
  parser/writer, because F04 deliberately provides neither.
- **Counter-factual:** A protocol that omits `sender_role`, `root_key`, or
  `created_at`, or permits removing the only durable copy during
  acknowledgement, fails the shared content-contract assertions.

**Preconditions/Input:** `msg-001` under `inbox/developer/` has sender role
`architect`, recipient role `developer`, root `E38`, child `E38-F04`, subject,
requested action, urgency `normal`, evidence link `docs/council/decisions/d-001.md`,
and RFC3339 `created_at`.

**Expected:** The protocol names every required field; acknowledgement/removal
is allowed only after the durable handoff/decision exists. Repeated
acknowledgement is documented as a no-op. No prompt transcript or secret is
permitted.

**Edges/negative:** Missing optional child key is valid; empty evidence is
invalid; malformed key, missing created time, and body containing a rendered
prompt are rejected and leave the original message intact.

### TC-002: I-04 durable artifact shape, secret exclusion, and lifecycle

**Feature Requirement:** REQ-F-005 / I-04.
**Acceptance Criterion:** AC-004.
**Technique:** Contract Surface Enumeration + Attack-class Enumeration.
**ISO 25010:** Functional Suitability, Compatibility, Reliability, Security.

**Caller-Path Contract:**

- **Entrypoint:** The embedded `shark-attack` `message-schema.md` protocol
  document, as resolved through the production shark-data bundle.
- **Lowest allowed mock seam:** None; read the embedded content directly.
- **Forbidden mocks:** Do not introduce or mock a schema validator, secret
  scanner, path validator, or artifact writer; those would recreate the
  rejected F04 runtime surface.
- **Counter-factual:** A protocol that permits a bearer token, prompt
  transcript, missing `next_action`, or silent conflicting-ID overwrite fails
  the content-contract assertions.

**Input/Expected:** The protocol defines each artifact type with root `E38`,
child `E38-F04`, roles, evidence, status, timestamps, and next action.
Identical replay is documented as idempotent; changed content with the same ID
is documented as a conflict. Artifact paths stay below `docs/council/`.

**Edges/negative:** The optional child is documented as valid when absent;
empty roles/evidence/next action, invalid Shark key, `../../outside`, absolute
path, symlink escape, `Authorization: Bearer ...`, API-key-like value,
rendered prompt marker, and unrestricted worker stdout are explicitly
prohibited before record creation.

## Cross-epic integration tests (X-##)

| X-## | Contract / shape source | Coverage pointer | TC | Required assertion |
|---|---|---|---|---|
| X-03 | E38 architecture §4.1 and §4.6; E19 sprint pull/claim contract | `tests/contracts/e38_f04_interactions_test.go#TC-003` | TC-003 | Workflow-resolved role and deterministic priority/dependency order drive the owning claim path; legacy assignment/model tier cannot override. |
| X-05 | E38 architecture §2 ADR-007 and §5 Phase 4; E32 embedded bundle contract | `tests/contracts/e38_f04_interactions_test.go#TC-004` | TC-004 | Embedded skill resolves, target override replaces only that skill, and unrelated embedded skills remain available. |

The feature spec's four shared pointers are intentionally canonical: TC-001
and TC-002 prove I-04, while TC-003 and TC-004 prove X-03 and X-05. No twin
consumer tests are permitted.

## Acceptance Test Cases

### TC-101: Valid chair-led roster validates and maps personas

**Requirements/AC:** REQ-F-001, REQ-F-002, REQ-F-011 / AC-001.
**Technique:** Equivalence Partitioning + Contract Surface Enumeration.
**ISO 25010:** Functional Suitability, Compatibility, Usability, Security,
Maintainability, Portability.

**Caller-Path Contract:**

- **Entrypoint:** `shark admin validate-data --project-root <temp-project>` (or the exact public validator used by that command), against a materialized `shark-attack` roster fixture.
- **Lowest allowed mock seam:** None for validation/content fixtures; use `t.TempDir` and the embedded bundle.
- **Forbidden mocks:** Do not mock manifest loading, embedded resolution, persona lookup, or path diagnostics.
- **Counter-factual:** A validator that treats `model_tier: opus` as authority or maps an unknown persona silently passes a malformed roster and fails the assertions.

**Input/Expected:** Team `shark-attack`, chair `tech-director`, memory root
`docs/council`, inbox root `docs/council/inbox`, communication retention flags,
seven unique IDs (`tech-director`, `product-manager`, `architect`,
`business-analyst`, `scrum-master`, `developer`, `qa`), non-empty role and
responsibilities, existing persona mappings, and optional `model_tier`.
Validation succeeds and returns explicit role/persona mappings; workflow
metadata remains authoritative and the roster contains no claim/status fields.

**Edges/negative:** A specialist member with no persona and a member without
model tier remain valid; empty member list, wrong team, chair not in members,
and model tier used as an assignment override are invalid.

### TC-102: Invalid roster and unsafe-content diagnostics

**Requirements/AC:** REQ-F-002, REQ-NF-001, REQ-NF-002 / AC-002.
**Technique:** Equivalence Partitioning + BVA + Attack-class Enumeration.
**ISO 25010:** Functional Suitability, Usability, Security, Maintainability,
Portability.

**Caller-Path Contract:**

- **Entrypoint:** The production `shark admin validate-data --project-root <temp-project>` entrypoint, one invalid fixture per subtest.
- **Lowest allowed mock seam:** None; embedded/materialized fixture and real validator.
- **Forbidden mocks:** Do not call an internal schema helper directly or mock the error formatter.
- **Counter-factual:** A validator that validates only YAML syntax, rather than semantic fields and safe paths, accepts at least one invalid partition.

**Input/Expected:** Independently test missing chair, duplicate `developer`
ID, empty responsibility, `../outside`, absolute `/tmp/council`, symlink
escape, unknown persona, `status_mutation` responsibility, and invalid root or
child key. Each fails with the field/path and reason; no files are written.

**Edges/negative:** Empty string, whitespace-only value, null optional field,
minimum valid one-member roster, and maximum supported member count must each
have a defined result; no generic “invalid roster” without location is allowed.

### TC-103: Council layout, private memory, and replace-only bundle override

**Requirements/AC:** REQ-F-003, REQ-F-009 / AC-008, AC-009; X-05.
**Technique:** State Transition + Equivalence Partitioning + Attack-class Enumeration.
**ISO 25010:** Functional Suitability, Compatibility, Usability, Reliability,
Security, Portability.

**Caller-Path Contract:**

- **Entrypoint:** `shark admin install-shark-data --project-root <temp-project>` followed by `shark admin validate-data`; exercise `Upgrade`/override resolution only through the public command or documented production API.
- **Lowest allowed mock seam:** OS filesystem in a temporary project; no bundle resolver mocks.
- **Forbidden mocks:** Do not mock the embedded FS, override precedence, gitignore generation, or unrelated skill lookup.
- **Counter-factual:** An upgrade that overwrites private council content or an override that shadows all skills fails byte-preservation and unrelated-skill assertions.

**Expected:** `docs/council/README.md`, `decisions/`, `handoffs/`,
`escalations/`, and inbox marker/layout exist; private content can be ignored
or locally overridden; `overrides/skills/shark-attack/...` replaces only the
target skill and `quality`/other embedded skills remain resolvable.

**Edges/negative:** Existing private file remains byte-identical on upgrade;
missing marker is restored; override outside allowed subtree, symlink escape,
and absolute override path are rejected.

### TC-104: Refreshed-worker message acknowledgement preserves durable state

**Requirements/AC:** REQ-F-004, REQ-F-012 / AC-003, AC-008; I-04.
**Technique:** State Transition + BVA.
**ISO 25010:** Functional Suitability, Compatibility, Reliability, Security.

**Caller-Path Contract:**

- **Entrypoint:** Public resume workflow `shark-attack` `resume` procedure with project root and member ID, using the actual documented file paths.
- **Lowest allowed mock seam:** Filesystem only.
- **Forbidden mocks:** Do not inject a pre-parsed inbox or bypass resume loading.
- **Counter-factual:** A resume implementation dependent on prior chat or one that discards durable context after acknowledgement fails after a fresh process.

**Input/Expected:** Decision `d-001`, handoff `h-001`, unresolved escalation
`e-001`, and `msg-001` for `developer`; start a fresh worker process, read
resume state, acknowledge `msg-001`, and repeat. All durable records remain
discoverable, message is gone/marked acknowledged exactly once, and pointers
are bounded paths and metadata.

**Edges/negative:** No child key, empty inbox, already acknowledged message,
stale referenced artifact, and duplicate resume must produce defined empty,
idempotent, or actionable results; no transcript is required.

### TC-105: Artifact replay and conflicting-ID protection

**Requirements/AC:** REQ-F-005, REQ-NF-003 / AC-004.
**Technique:** State Transition + Attack-class Enumeration.
**ISO 25010:** Functional Suitability, Reliability, Security, Maintainability.

**Caller-Path Contract:**

- **Entrypoint:** Public typed artifact-write operation through the communication workflow, with the exact production artifact argument shape.
- **Lowest allowed mock seam:** Atomic filesystem writer.
- **Forbidden mocks:** Do not mock the existing artifact, ID comparison, or retry path.
- **Counter-factual:** A non-idempotent writer duplicates records, while a last-write-wins writer hides conflicting reuse; both fail.

**Expected:** First write creates one artifact; byte-equivalent replay returns
success and one artifact; changed content with the same ID returns an
actionable conflict and preserves the first artifact. Retry after an injected
transient write failure does not erase completed sibling artifacts.

**Negative:** Partial file, invalid status transition, missing next action,
and secret-bearing content never become visible as a durable artifact.

### TC-106: Missing escalation policy routes to council review and pause

**Requirements/AC:** REQ-F-006 / AC-005; UAT-11.
**Technique:** Decision Table + State Transition.
**ISO 25010:** Functional Suitability, Usability, Reliability, Security,
Maintainability.

**Caller-Path Contract:**

- **Entrypoint:** Public `shark-attack` escalation workflow with root/child scope and structured question/evidence input.
- **Lowest allowed mock seam:** Filesystem read/write; use the configured workflow adapter only at its documented boundary.
- **Forbidden mocks:** Do not mock policy-file existence, route selection, or pause recommendation.
- **Counter-factual:** An implementation that guesses a human destination or continues after an unresolved material question fails.

**Decision table:** policy present + resolved → resolution; policy present +
unresolved → configured route/status; policy absent + unresolved → escalation
status `unresolved`, route `council-review`, recommendation `pause/review`;
policy absent + non-material → no fabricated escalation. Every row is tested.

**Expected:** Escalation contains trigger, evidence, roles, root/child keys,
requested decision, route, status, and next action. No fixed human destination
is invented.

### TC-107: Role-aware self-pull excludes legacy/model authority

**Requirements/AC:** REQ-F-007, REQ-F-008 / AC-006, AC-007; X-03.
**Technique:** Decision Table + Contract Surface Enumeration.
**ISO 25010:** Functional Suitability, Compatibility, Usability, Reliability,
Security, Maintainability.

**Caller-Path Contract:**

- **Entrypoint:** `shark sprint next --agent=developer` → production `SprintService.GetNextTask(ctx, "developer")`, then the owning claim path with its production `ClaimInput` shape.
- **Lowest allowed mock seam:** Repository/claim service interfaces below the service; real CLI/service wiring must execute.
- **Forbidden mocks:** Do not call a helper with `agent_type` omitted, mock `GetNextTask`, use roster `model_tier` as agent type, or mock the claim service above `ClaimInput`.
- **Counter-factual:** A selector using the legacy `agent` field or model tier returns an ineligible item; a detached claim call permits duplicate ownership.

**Input/Expected:** Priority-ordered sprint items: architecture assigned by
workflow to `architect`, implementation to `developer`, QA to `qa`, with one
dependency and misleading legacy assignments/model tiers. Developer receives
only eligible work in deterministic order and claim uses the owning path;
returned metadata includes canonical prompt pointer/role, not free-form role
prose. Roster membership grants no status or claim authority by itself.

**Edges/negative:** no eligible item, blocked dependency, equal priority tie,
live claim, and repeated pull from the same session produce exclusion or
ownership diagnostics without stealing or duplicate active claims.

### TC-108: Child worker returns evidence without root mutation

**Requirements/AC:** REQ-F-008 / AC-007; UAT-10.
**Technique:** Attack-class Enumeration + Contract Surface Enumeration.
**ISO 25010:** Functional Suitability, Compatibility, Reliability, Security,
Maintainability.

**Caller-Path Contract:**

- **Entrypoint:** Production child-worker handoff/evidence-return boundary used by E38-F02, with root lease context and child outcome argument shape.
- **Lowest allowed mock seam:** Owning child claim/evidence adapter; root transition service remains real and instrumented.
- **Forbidden mocks:** Do not mock root lease ownership checks, status-transition calls, or worker output sanitization.
- **Counter-factual:** A worker that calls root `status set`, root `release`, force-claims, or returns a prompt transcript is rejected by call audit and output assertions.

**Expected:** Worker may read state, claim/heartbeat/release its authorized
child, write scoped artifact, and return semantic outcome/evidence pointer;
root lease and root workflow history are unchanged. Coordinator-only root
transition remains the sole valid transition source.

**Attack classes:** root status mutation, root lease release, force claim,
cross-child write, arbitrary path, secret/prompt output, and invalid evidence
pointer. Each is denied with an actionable error and no root mutation.

### TC-109: Resume guidance loads all required durable context

**Requirements/AC:** REQ-F-003, REQ-F-012 / AC-003, AC-008; UAT-09.
**Technique:** State Transition + Contract Surface Enumeration.
**ISO 25010:** Functional Suitability, Compatibility, Usability, Reliability,
Security, Portability.

**Caller-Path Contract:**

- **Entrypoint:** Production `shark-attack` resume workflow for member `qa` after a fresh worker start.
- **Lowest allowed mock seam:** Filesystem fixture; no in-memory context injection.
- **Forbidden mocks:** Do not supply decisions/handoffs/escalations/inbox via a conversation fixture or bypass path discovery.
- **Counter-factual:** A resume flow that reads only the inbox or only the latest decision misses unresolved context and fails the complete pointer set.

**Input/Expected:** Populate one decision, handoff, unresolved escalation, and
actionable inbox item scoped to `E38`/`E38-F04`. Fresh QA worker sees all four
bounded pointers, scope, roles, statuses, and next actions; acknowledgement
does not delete the durable records.

**Negative:** unrelated root and malformed/secret-bearing files are excluded
or reported, not loaded into worker context.

### TC-003: X-03 role-pull contract preserves workflow order and claim ownership

**Requirements/AC:** REQ-F-007 / AC-006; X-03; UAT-01, UAT-02.
**Technique:** Contract Surface Enumeration + Decision Table.
**ISO 25010:** Functional Suitability, Performance Efficiency, Compatibility,
Reliability, Security.

**Caller-Path Contract:**

- **Entrypoint:** Shared contract test at `tests/contracts/e38_f04_interactions_test.go#TC-003`, driving the production sprint/claim handoff with `agent=developer`.
- **Lowest allowed mock seam:** E19 service/repository boundary; no mock above the public sprint command/service caller.
- **Forbidden mocks:** Do not test a convenient roster parser or fake atomic claim-next implementation; F04 consumes E19's owner contract.
- **Counter-factual:** If legacy assignment, roster model tier, or non-deterministic ordering controls eligibility, the shared fixture returns the wrong item or duplicate claim.

**Expected:** Workflow role filtering, priority/dependency order, canonical
prompt metadata, and owning claim/session identity are all asserted. Repeated
pulls never yield two active claims for one child. Atomic claim-next remains
E19/F02-owned if it is not yet available.

### TC-004: X-05 embedded bundle and replace-only override contract

**Requirements/AC:** REQ-F-009 / AC-009; X-05; UAT-08/UAT-09.
**Technique:** Contract Surface Enumeration + State Transition.
**ISO 25010:** Functional Suitability, Compatibility, Reliability, Security,
Maintainability, Portability.

**Caller-Path Contract:**

- **Entrypoint:** Shared contract test at `tests/contracts/e38_f04_interactions_test.go#TC-004`, through `shark admin install-shark-data`/`validate-data` and resolver behavior.
- **Lowest allowed mock seam:** Embedded FS and temp-project filesystem only.
- **Forbidden mocks:** Do not mock manifest identity, embedded skill lookup, override precedence, or unrelated skill availability.
- **Counter-factual:** A disk-only implementation, wrong manifest identity, or broad override causes the skill to disappear or unrelated skills to be unavailable.

**Expected:** Canonical `shark-attack` identity is embedded and installable;
an override replaces only its content; `quality` and existing personas remain
available; private council memory is project-local and not bundled as secret
content.

### TC-110: Missing product context and unavailable capability fallback

**Requirements/AC:** REQ-F-010 / AC-010; UAT-07/UAT-08.
**Technique:** Decision Table + Equivalence Partitioning.
**ISO 25010:** Functional Suitability, Compatibility, Usability, Reliability,
Security, Portability.

**Caller-Path Contract:**

- **Entrypoint:** Production `shark-attack` setup/dispatch guidance invoked with project-root context and capability/product-gate inputs, followed by ordinary `/run` regression invocation.
- **Lowest allowed mock seam:** Capability/product-context providers at their documented interfaces; keep fallback selection and `/run` routing real.
- **Forbidden mocks:** Do not mock the fallback decision, invent a product answer, or replace ordinary `/run` with an attack-team implementation.
- **Counter-factual:** A flow that silently changes `/run`, guesses a product decision, or claims parallel capability when unavailable fails the mode and command assertions.

**Decision table:** product context present + capability available → proceed;
missing product gates → bootstrap/escalation recommendation; capability
unavailable + sequential safe → explicit sequential fallback; capability
unavailable + no safe fallback → actionable stop. All preserve ordinary
`/run` behavior and report the selected mode/reason.

## Test Infrastructure

### Existing patterns to follow

- `internal/sharkdata/embed_test.go`: `t.TempDir`, embedded bundle materialization, `Init`, `Upgrade`, `Validate`, path traversal and symlink/security fixtures.
- `internal/cli/commands/sharkdata_cmd_test.go`: CLI command tests with injected/mocked dependencies and output assertions.
- `internal/cli/commands/sprint_test.go`: production-shaped mocked sprint service, including `GetNextTask(ctx, agentType)` and compile-time interface checks.
- `internal/services/sprint_service_test.go` and `internal/services/claim_service_test.go`: service tests with mocked repositories/claim seams; no real DB outside repository tests.
- `internal/sharkdata/resolve_at_test.go`: embedded/override resolution contract coverage.
- `internal/sharkdata/default_data/manifest.yaml`: manifest and embedded content are the source of shipped skill identity.

### New fixtures/helpers needed

- Add `tests/contracts/e38_f04_interactions_test.go` at the exact shared
  pointer in `spec.md`, with deterministic roster/message/artifact fixtures and
  TC-001 through TC-004 aliases/comments required by the map.
- Add a focused F04 validator/content test file under
  `internal/sharkdata/` if `embed_test.go` becomes unwieldy; use temp roots and
  fixture YAML/Markdown, not the project `shark-tasks.db`.
- Add a council fixture helper that writes only bounded structured artifacts
  and can capture redacted log fields. It must reject secrets and path escapes.
- Add a role-pull fixture using the existing sprint/claim mock interfaces and
  production CLI/service argument shape. Do not add a second claim store.
- Add a small path/secret attack corpus covering absolute paths, `..`, symlink
  escape, bearer/API-key patterns, rendered prompt markers, unrestricted
  stdout, invalid Shark keys, duplicate IDs, and conflicting artifact IDs.

No database migration is expected. Repository tests may use the real test DB,
but F04 content, CLI, service, and contract tests must use temp files, pure
fixtures, or mocked repository/service seams per `.claude/rules/testing/architecture.md`.

## Codex Test-Plan Red-Team

**Verdict:** NOT RUN — codex command was not supplied to this worker
**Issues raised:** 0
**Issues addressed before dev:** 0
**Issues deferred:** 1 — parent/orchestrator should run the configured Codex
red-team command before development if available; this is non-blocking because
the plan explicitly enumerates techniques, attack classes, ISO cells,
observability, negative cases, and caller-path contracts.

The required `codex_command` input was absent, and the parent-loop contract
prohibits spawning an additional external AI worker. This plan therefore
records the tooling exception rather than claiming a review was executed.

## Recommendations

- [x] Ready for development after parent records the missing-task-spec assumption.
- [ ] Needs BA refinement.
- [ ] Needs tech refinement.
- [ ] Run the configured Codex red-team review when the parent supplies it; resolve any BLOCKER before implementation.
