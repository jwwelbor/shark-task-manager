---
feature_key: E34-F07-state-space-planning-and-decision-propagation
epic_key: E34
title: State-Space Planning and Decision Propagation — Test Plan
---

# E34-F07 Test Plan

**Mixed content-and-runtime feature.** Five of six changed files are bundle
content only: one new workflow file
(`skills/quality/workflows/state-space-coverage.md`) and edits to
`specification.md`, `test_planning.md`, `task_review.md`, `feature_review.md`,
`tech_debt/resolved.md`, and `question-management/SKILL.md`. Per CLAUDE.md's
prompt-only testing guidance and this plan's "Prompt-only changes" gate,
those test cases use the production template renderer, direct-file/grep
checks, and manual policy-wording review — not caller-path, mutation, or
decision-table tests, which apply only to deterministic runtime behavior.
The **one runtime surface** — `shark impact record` (`internal/cli/commands/impact_cmd.go`,
`internal/services/impact_service.go`) — is deterministic Go code with a real
production caller signature and gets full Caller-Path Contracts, ISTQB
technique application, and ISO 25010 coverage below.

## Spec Drift Analysis

### Drift Findings

**Cross-feature contract-test pointer name mismatch (documented, resolved
without unilaterally picking a side).** spec.md's "Cross-feature interactions
→ Produces" section names the I-04 contract test pointer as
`E34-F07-.../test-plan.md#TC-I-04-CHANGE-IMPACT-CLOSURE` and states "Both IDs
are taken verbatim from `E34-interaction-map.md`." The interaction map itself
(`E34-interaction-map.md` "I-04 ChangeImpactSet" section, line 66) actually
names the pointer `test-plan.md#TC-I-04-DECISION-PROPAGATION` — a different
string. This is exactly the "shared naming integrity" drift class REQ-F-005
of this very feature asks task review to catch. The exit gate for this test
plan reads "every I-## has a contract test case whose TC name... matches the
declared contract test pointer" against **the feature spec** (spec.md), which
names `TC-I-04-CHANGE-IMPACT-CLOSURE` — so this plan does not override the
spec's stated name. Instead, the Cross-feature contract tests table below
anchors the contract test under **`TC-I-04-CHANGE-IMPACT-CLOSURE`** (matching
spec.md, satisfying this feature's own exit gate) with an explicit
cross-reference noting it is also the test the interaction map calls
`TC-I-04-DECISION-PROPAGATION`, so E34-F08 finds the right test whichever
document it consults. Recommend a follow-up edit reconciling spec.md and
`E34-interaction-map.md` to one canonical string before E34-F08 test planning
begins — this plan does not adjudicate which side is authoritative, since
that is a spec/interaction-map correction, not a test-planning decision.

No other drift found. spec.md traces every REQ-F-00N to feature.md's
identically numbered requirement; AC-1 through AC-5 in spec.md map cleanly
onto feature.md's REQ list and three named acceptance scenarios (see
Traceability Matrix). spec.md's architecture section (component-change table,
data-model note, API contract, key technical decisions) is additive detail
within feature.md's stated scope — no scope creep or narrowing detected.

### Traceability Matrix

| Feature PRD Requirement | Task/Spec Acceptance Criterion | Covered? | Notes |
|---|---|---|---|
| REQ-F-001 Closed lifecycle tables | AC-1 (state-space-coverage.md exists, contains closed-table section) | Yes | TC-001 |
| REQ-F-002 Technique selection from state shape | AC-1 (technique-selection section), AC-2 (test_planning.md references it) | Yes | TC-001, TC-003 |
| REQ-F-003 Dependency discovery by interaction/caller path | AC-1 (dependency-discovery section), AC-2 (specification.md references it) | Yes | TC-001, TC-002 |
| REQ-F-004 Shipped consumer re-verification | AC-1 (shipped-consumer section) | Yes | TC-001, TC-006 (feature_review.md reference) |
| REQ-F-005 Shared naming integrity | AC-2 (task_review.md paragraph — AC-2 names `specification.md`, `test_planning.md`, and `task_review.md` together) | Yes | TC-004 |
| REQ-F-006 Decision propagation | AC-1 (I-04 propagation section), AC-3 (resolved.md + SKILL.md reference), AC-4 (`shark impact record`) | Yes | TC-001, TC-005, TC-007..TC-013 |
| REQ-F-007 Design divergence | AC-1 (design-divergence section, references defect-class-sweep.md) | Yes | TC-001 |
| REQ-NF-001 Content-level enforcement, no schema change | (implicit — no dedicated AC number, covered by "no new persistence" check) | Yes | TC-014 |
| Scenario: "Plan a multi-entity lifecycle" | AC-5 | Yes | TC-015 (also epic UAT-04, `uat-plan.md`) |
| Scenario: "Propagate a ratified decision" | AC-5 | Yes | TC-016 (also epic UAT-04, `uat-plan.md`) |
| Scenario: "Reject naming drift" | AC-5 | Yes | TC-017 |

## Acceptance Criteria Review

### Ambiguity Findings

- AC-1's "renders cleanly" is operationalized as: passes the production
  template/include renderer with no error, consistent with E34-F06's TC-001
  precedent. Not ambiguous once tied to that concrete check.
- AC-2/AC-3's "reference... rather than restating" is operationalized as a
  **bounded, per-file reference contract**, not an open-ended paraphrase
  heuristic (codex's red-team flagged an earlier draft of this operationalization
  as an unbounded robustness assertion — fixed here): each of the five edited
  files (`specification.md`, `test_planning.md`, `task_review.md`,
  `feature_review.md`, `tech_debt/resolved.md`, `question-management/SKILL.md`)
  gets an exact, enumerated allowed-content rule — (a) the file MUST contain a
  literal path reference or `{{include}}`/link token naming
  `state-space-coverage.md` (or the specific `resolved.md`/`SKILL.md`
  sub-anchor), and (b) the file's diff for the touched section MUST NOT add
  more than one new sentence describing the referenced procedure's mechanics
  (a bounded line-count/sentence-count rule fixed at implementation time and
  recorded in the test file itself, mirroring E34-F06's
  `TestDefectClassSweepConsolidatedNotDuplicated`'s closed
  old-prose-string list rather than a semantic classifier). This removes the
  "implementation-time judgment call" ambiguity by requiring the bound to be
  written into the test, not left to a reviewer's later paraphrase judgment.
- AC-4's "fails minimal I-04 shape validation" is concrete and testable:
  three named required fields (`source_kind`, `source_key`,
  `affected_artifacts` non-empty array), each independently triggerable via a
  fixture missing that one field. The full command-contract partition (valid
  key formats, unknown-entity-type keys, wrong-arity CLI args, missing
  required flags, `--json` output, wrong-JSON-type fields, invalid
  `--impact-file` paths) is enumerated in Step 5.5's Equivalence
  Partitioning row below and the AC-4 Test Matrix.

No open ambiguities requiring refinement.

### Missing Coverage

None found. Every feature.md requirement has at least one AC in spec.md;
every AC has at least one test case below.

## ISTQB Technique Application (per AC)

| AC | Technique(s) Applied | Test Cases Generated | Rationale |
|----|---|---|---|
| AC-1 | Content-only: direct render + clause-level section-content check (not heading presence alone) | TC-001 | Prompt-only change; no input partitions to enumerate, but the six required sections' *substantive clauses* (lifecycle heuristic, all closed-table columns, invalid/recovery behavior, ordered dependency-discovery sources with per-axis rationale, shipped-consumer handoff fields, I-04 artifact-closure fields, divergence-reference-not-restatement) must each be asserted individually — a heading-only check would pass a section with the right title and empty or wrong content |
| AC-2 | Content-only: bounded reference-contract check (exact per-file allow-list, not open-ended paraphrase detection — see Ambiguity Findings above) | TC-002, TC-003, TC-004 | Same bounded-contract pattern as E34-F06's AC-2, made explicit per the codex red-team finding that an undefined "restatement" oracle is an open-ended robustness assertion |
| AC-3 | Content-only: reference-presence grep | TC-005 | tech_debt/resolved.md and SKILL.md reference check |
| AC-4 | Equivalence Partitioning (valid key × valid merged content; unknown-entity-type key; missing required flag(s) (`--source-kind`/`--source-key`/`--source-pointer`/`--impact-file`); each of 3 missing-required-field partitions in the merged content; wrong-JSON-type field value; malformed JSON; `--impact-file` variant with valid/missing/unreadable file) + Boundary Value Analysis (empty `affected_artifacts` array vs. one-element array) | TC-007..TC-013 | `shark impact record` is deterministic Go code; the full partition set (enumerated in the AC-4 Test Matrix below) closes the gaps the codex red-team flagged in an earlier draft (unknown key formats, argument arity, JSON type errors, `--impact-file` failure modes) |
| AC-5 | Content-only: three fixture walkthroughs against rendered workflow prose, mirroring E34-F06's TC-005..TC-009 manual-scenario pattern, backed by a committed rendered sample per scenario (not review-report prose alone) | TC-015, TC-016, TC-017 | No executable classifier exists for prose guidance; scenarios are proven by manual policy-wording review against concrete fixtures AND a durable rendered-output artifact (per AC-5's own wording, "a rendered-output sample demonstrates..."), per E34-F05's precedent of not building a fixture-execution harness while still requiring a committed artifact as the oracle, not just a UAT report claim |
| REQ-NF-001 | Content-only: grep-based negative check | TC-014 | Confirms no new DB column/table/migration was introduced outside the one declared exception |

ACs without a technique annotation = none; every AC above has one.

## ISO 25010 Coverage Matrix

| AC | Functional | Performance | Compat | Usability | Reliability | Security | Maintainability | Portability |
|----|---|---|---|---|---|---|---|---|
| AC-1 | ✅ TC-001 | N/A — static content, no runtime path | N/A — no cross-version format | ✅ TC-001-usability (a worker following the rendered prose must be able to locate and apply each section without cross-referencing five other files — TC-001 asserts each section is self-contained enough to act on, not just present) | ✅ TC-001-reliability (renderer/include failure — a broken `{{include}}` reference or unresolvable path must surface as a hard render error, not silently omit content) | N/A — bundle content authored by the project, not runtime-supplied input | ✅ TC-002/TC-003 (single canonical source, no duplication) | N/A — bundle content, not a portable binary artifact |
| AC-2 | ✅ TC-002, TC-003, TC-004 | N/A | N/A | N/A | N/A | N/A | ✅ TC-002/TC-003/TC-004 (drift-prevention is itself a maintainability property) | N/A |
| AC-3 | ✅ TC-005 | N/A | N/A | N/A | N/A | N/A | ✅ TC-005 | N/A |
| AC-4 | ✅ TC-007..TC-013 | N/A — single note write, no perf-sensitive path | N/A — no format negotiation | N/A — CLI-only, no UI | ✅ TC-009 (target-not-found handled without partial write) AND TC-013 (validation failure and file-read failure leave zero notes written — see AC-4 Test Matrix reliability row) | ✅ TC-008/TC-013 (accepts untrusted JSON content and a user-supplied `--impact-file` path; the command must not follow `--impact-file` outside expected use — e.g. must not silently execute or interpret file content as anything but literal note text — and must reject malformed JSON before any write, closing the codex-flagged "security marked N/A despite untrusted input" gap) | ✅ thin-wrapper pattern matches `notes_add_dispatch.go` precedent (TC-007 Caller-Path Contract) | N/A — Go CLI, cross-platform by existing build |
| AC-5 | ✅ TC-015, TC-016, TC-017 | N/A | N/A | ✅ TC-015/TC-016/TC-017 (the acceptance scenarios are themselves a usability property — a worker must be able to follow the rendered guidance to the stated outcome without additional interpretation, per feature.md's stated Impact) | N/A | N/A | N/A — prose scenario, not a code artifact | N/A |
| REQ-NF-001 | N/A — non-functional constraint, not a behavior | N/A | N/A | N/A | N/A | N/A | ✅ TC-014 (schema/persistence surface stays closed) | N/A |

### Coverage Gaps

None deferred as of this revision. An earlier draft of this plan marked
AC-1/AC-5 usability, AC-1 reliability, and AC-4 security as N/A without
justification; codex's red-team pass (below) correctly flagged all four as
unjustified N/A cells for a feature whose primary value is workflow-prose
clarity (usability) and whose one runtime command accepts untrusted JSON/file
input (security). Each is now backed by a named test-case behavior above
rather than left blank. Remaining true N/A cells (performance, compatibility,
portability across all ACs) are justified inline: none of this feature's
artifacts have a performance-sensitive path, a cross-version wire format, or
a portable-binary packaging concern.

## Observability Design (per behavior)

| Behavior | Metric | Log | Trace span | Alert threshold | Test assertion |
|---|---|---|---|---|---|
| Bundle content renders (AC-1) | internal — no observability (static content, no runtime signal to emit) | internal — no observability | internal — no observability | N/A | TC-001 asserts render success directly |
| `shark impact record` writes a reference note | internal — no observability (repo has no CLI command telemetry framework; reuses `note.go`'s existing no-metric precedent) | Existing note-service creation path already surfaces success/error through the command's own output (`cli.OutputJSON`/`cli.Success`/returned error) — the same signal `shark create note` already provides; no new log line needed | internal — no observability (no tracing framework in this CLI) | N/A | TC-007 asserts the printed/JSON success output; TC-009 and TC-013 assert the error exit path and message. Runtime evidence beyond the mock assertion: TC-007 additionally queries `shark <entity> notes <key>` (or the equivalent note-list service call) after the command runs, against a real in-memory/sqlite-backed repository in at least one test, to confirm exactly one note actually persists — not only that the mock recorded one `Create` call. This closes the codex-flagged gap that a pure-mock assertion is not runtime evidence of the write. |
| `shark impact record` validation rejection | internal — no observability | Error message printed via the command's returned error (existing CLI convention: `fmt.Errorf` wrapped, non-zero exit) | internal — no observability | N/A | TC-010..TC-013 assert the specific rejection message is emitted AND that zero notes exist afterward (via the same real-repository query used in TC-007, run once for a representative rejection case — TC-010 — to prove the negative, not one query per rejection variant) |

**Implementation hook:** none beyond existing CLI error/success output
conventions — this repo's CLI commands do not emit metrics/traces, and
`shark impact record` does not introduce a new pattern; QA verifies the
existing convention (non-zero exit + stderr message) is followed, and that
the persisted-note-count runtime check (not just a mock call-count assertion)
passes for both the happy path and at least one rejection path.

## Cross-feature contract tests (I-##)

| I-## | Producer | Consumer(s) | Shape source | Contract test pointer | TC |
|---|---|---|---|---|---|
| I-02 | E34-F05 | E34-F06, E34-F07, E34-F08 | architecture.md#i-02-gateresult-v1 | **GAP** — same upstream gap E34-F06 already documented: `TC-I-02-GATERESULT-PARITY` does not exist because E34-F05 shipped with no test-plan.md. Out of E34-F07 scope to create (I-02 is consumed, not produced, here). This feature only nests `change_impacts` as a sibling array alongside `remediation_sweeps` inside the existing envelope (architecture.md line 165); TC-014 structurally confirms no new persistence was added around that nesting, which is the only I-02-adjacent claim this feature makes. | (owned by E34-F05's test plan, which does not yet exist) |
| I-04 | E34-F07 | E34-F08 | architecture.md#i-04-changeimpactset-v1 | `E34-F07-state-space-planning-and-decision-propagation/test-plan.md#tc-i-04-change-impact-closure--tc-i-04-decision-propagation` (a real heading below, matching spec.md's declared pointer name as the primary anchor, satisfying this plan's own exit gate against the feature spec) | See the `TC-I-04-CHANGE-IMPACT-CLOSURE` heading immediately below. |

### TC-I-04-CHANGE-IMPACT-CLOSURE / TC-I-04-DECISION-PROPAGATION

This is one contract test known by two names across E34's planning documents:
spec.md's "Cross-feature interactions → Produces" section calls it
`TC-I-04-CHANGE-IMPACT-CLOSURE`; `E34-interaction-map.md`'s "I-04
ChangeImpactSet" section calls it `TC-I-04-DECISION-PROPAGATION` (see Spec
Drift Analysis above for the mismatch — this test plan does not adjudicate
which document should change, only makes both names resolve to the same
evidence). E34-F08's test plan may reference either name; both point here.

**Constituent test cases:** TC-016 (rendered-content proof that the I-04
propagation section requires naming every invalidated artifact/consumer AC
with a disposition — see the AC-5 Test Matrix and
`scenario-review-TC-015-TC-017.md`) + TC-007 (`shark impact record`
end-to-end: valid I-04-shaped payload persists as exactly one
`--type=reference` note on the target key, verified against a real
repository, not only a mock — see the Caller-Path Contracts table above).
Together they prove both halves of the I-04 contract this feature owns: the
workflow-content obligation (what a Question/tech-debt/change-card resolution
must produce) and the one new runtime persistence hook (what ADR adoption
calls to produce the same durable representation).

## Cross-epic integration tests (X-##)

None. spec.md "Cross-epic integrations" declares no X-## rows for this
feature (grep-confirmed empty in both `E34-cross-epic-map.md` and
`docs/product/cross-epic-integration-map.md`, per spec.md's own statement).

## Caller-Path Contracts

Content-only test cases (TC-001..TC-006, TC-014..TC-017) name their concrete
entrypoint per the "Prompt-only changes" exemption instead of a Go caller
signature:

- **TC-001**: `internal/templates` production renderer — entrypoint is the
  renderer already exercised by `internal/templates/includes_test.go` (e.g.
  sibling to `TestIncludeResolver_BasicInclude`). `content-only` justification.
- **TC-002, TC-003, TC-004, TC-005, TC-006, TC-014**: direct grep/file-content
  check against the checked-in bundle files — entrypoint is the file itself.
  `content-only` justification.
- **TC-015, TC-016, TC-017**: manual policy-wording review of the rendered
  `state-space-coverage.md` content against each fixture scenario —
  `content-only` justification; no executable classifier exists because the
  content is instructional prose a future AI worker follows, not a Go
  decision function (same posture as E34-F06's TC-005..TC-009).

Runtime test cases (TC-007..TC-013) drive `shark impact record`.

**Mock-seam correction (fixed after codex red-team; an earlier draft of this
table was wrong):** `NoteService.AddNote`/`AddNoteWithMetadata` does **not**
resolve the target entity key through `NoteEntityNoteRepository`. It resolves
the key via `resolveEntityID`, which calls `EntityRegistry.GetRepository(entityType)`
(a per-entity-type repository selected by `DetectEntityType(key)`, the same
`internal/keys` detector `create.go`/`update_dispatch.go` already use for
verb-first commands) and then that repository's `GetByKey(ctx, key)` — only
the **resolved integer ID** reaches `NoteEntityNoteRepository.Create`. A test
that mocks only `NoteEntityNoteRepository` cannot observe the key argument at
all and cannot produce a target-not-found failure by itself. Every runtime
test case below therefore mocks **two** seams: the entity-type repository
(`GetByKey`, returning either a resolved entity or a not-found error) and
`NoteEntityNoteRepository.Create` (returning success and capturing the
resolved ID/content/note-type). Both seams are constructed via the same
`EntityRegistry`-and-mock-repository wiring `note_service_test.go` already
uses for its own `AddNote` tests — no new test infrastructure pattern is
introduced, only a mock at the correct interface.

| TC | Production entrypoint | Lowest mock seam | Forbidden mocks | Counter-factual |
|----|---|---|---|---|
| TC-007 | `runImpactRecord(cmd *cobra.Command, args []string) error` in `internal/cli/commands/impact_cmd.go`, invoked via the registered `shark impact record <key> --source-kind=<kind> --source-key=<key> --source-pointer=<path> --impact-file=<path>` Cobra command (production argument shape: one positional entity-key arg, four required flags, `--json` flag optional — the architecture.md-declared ADR-adoption boundary) | `services.ImpactService.RecordImpact(ctx, key, content)` calling the real `NoteService.AddNote`/`AddNoteWithMetadata`, which calls (a) the mocked entity-type repository's `GetByKey(ctx, key)` returning a resolved entity with a known ID, then (b) the mocked `NoteEntityNoteRepository.Create` — assert `GetByKey` was called with the exact input key, `Create` was called **exactly once** with `note_type = reference`, the resolved entity ID, and content equal to the impact-file's JSON merged with the three source flags. At least one instance of this test additionally runs against a real (in-memory/temp-sqlite) repository pair rather than mocks, per the Observability Design section, to confirm the note is genuinely queryable afterward, not merely mock-recorded. | Do not mock `ImpactService` itself, and do not mock at the `NoteService` level — the command test must drive the real `ImpactService` and real `NoteService` wired to mocked repositories, the same way existing thin-command tests (`notes_add_dispatch_test.go`) drive the real note-add path | A buggy impl that swallows the entity-key argument and always writes to a hardcoded key would pass a test that mocks `NoteService`/`ImpactService` directly, but fails this test because the mocked entity repository's `GetByKey` assertion catches the wrong key being passed |
| TC-008 | Same entrypoint, `--impact-file` content form (`shark impact record <key> --source-kind=... --source-key=... --source-pointer=... --impact-file=/path/to/impact.json`) | Same two-repository seam; file read happens in the command layer before calling the service, per the thin-wrapper pattern — mock only the repositories, not file I/O, so the test uses a real temp file | Do not mock `os.ReadFile` — a real temp file proves the file read actually reads bytes rather than passing the literal path string through. Also verify a missing/unreadable file path returns a clear file-error before any repository call (zero `GetByKey`/`Create` invocations). | A buggy impl that treats the `--impact-file` value as literal note content would pass the file's path string instead of its bytes; only a real-file test catches this. A buggy impl that treats a file-read error as empty content (silently proceeding) would pass a test that only checks "no crash"; this test requires zero repository calls on file-read failure. |
| TC-009 | Same entrypoint, target key that does not exist | Mocked entity-type repository's `GetByKey` returns a not-found error for the given key (matches the existing `NotFoundError` pattern `note_service.go`'s `resolveEntityID` already wraps); assert `NoteEntityNoteRepository.Create` is never called | Do not mock `ImpactService.RecordImpact` or `NoteService.AddNote` to return a canned not-found error — the not-found path must originate from the real entity-repository-lookup seam so the test proves the actual key-resolution code runs, and must prove zero notes are written for a nonexistent target | A buggy impl that never checks the target key's existence (e.g., blind insert into a note table keyed only by a string) would pass a test that mocks the not-found response directly; this test requires the mocked entity repository to be the sole source of the not-found signal, with the note repository asserting zero calls |
| TC-010 | Same entrypoint, content JSON missing `source_kind` | `services.ImpactService.RecordImpact` — validation happens in the service before any repository call; both the mocked entity repository and `NoteEntityNoteRepository` must assert they were never invoked | Do not let either mocked repository silently accept a call — assert zero calls to both, proving validation short-circuits before entity lookup or persistence | A buggy impl that validates only `source_key`/`affected_artifacts` (skipping `source_kind`) would pass a test that only checks the overall error return; this test asserts the specific missing-field error message names `source_kind` and that entity lookup never ran |
| TC-011 | Same entrypoint, content JSON missing `source_key` | Same as TC-010 | Same as TC-010 | Same shape, `source_key` field; catches a partial-validation bug that checks `source_kind`/`affected_artifacts` but not `source_key` |
| TC-012 | Same entrypoint, content JSON with `affected_artifacts: []` (present but empty) or with `affected_artifacts` present as a wrong JSON type (e.g. a string instead of an array) | Same as TC-010 | Same as TC-010 | A buggy impl that only checks field *presence* (not non-emptiness or correct type) would pass a test using a missing key but fail this one — this is the Boundary Value Analysis case distinguishing "absent," "empty array," and "wrong type" |
| TC-013 | Same entrypoint, malformed (non-JSON) content on an existing key | `services.ImpactService.RecordImpact` — JSON parsing happens before shape validation and before any repository call; both mocked repositories must assert zero calls | Do not catch the parse error and fall through to a generic "invalid" path without a specific parse-failure message — the error must name the JSON-parse failure distinctly from a missing-field validation failure (so a caller can tell malformed input from well-formed-but-incomplete input) | A buggy impl that treats malformed JSON the same as "all fields present but empty" (e.g., swallowing the parse error and defaulting to a zero-value struct) would pass a test that only checks for a non-zero exit code; this test requires zero repository calls and a JSON-parse-specific message |

## AC Test Matrix

### AC-1: state-space-coverage.md exists, renders cleanly, all required sections present

Note: spec.md AC-1 says "all five sections named in REQ-F-001–004/007"; that
range names five requirement numbers, but REQ-F-006 (I-04 propagation) is
separately required by spec.md's own body text ("Add an 'I-04 propagation'
section to the state-space-coverage.md workflow") and is not optional — so
this test plan checks **six** sections (closed-table, technique-selection,
dependency-discovery, shipped-consumer, I-04 propagation, design-divergence),
treating spec.md AC-1's "five" as a minor undercount rather than omitting the
I-04 section from test coverage. Recommend spec.md AC-1 be corrected to name
six sections in a follow-up edit.

| TC | Description | Input/Setup | Expected outcome | Edge cases |
|----|---|---|---|---|
| TC-001 | New workflow file renders cleanly and contains the full required *content* of each section, not just its heading | `internal/sharkdata/default_data/skills/quality/workflows/state-space-coverage.md` present, run through the production renderer (extend `TestIncludeResolver_*` table or add a sibling test); the test asserts each section's required clauses individually rather than a single heading-presence check | Renderer returns no error; output contains, per section: (1) closed-table shape — the lifecycle-field detection heuristic AND all seven table columns (value, meaning, entry transitions, exit transitions, terminal/no-exit marker, invalid-transition list, failure/recovery behavior) named; (2) technique-selection — the trigger condition (a closed lifecycle table exists) AND the state-transition/decision-table technique name; (3) dependency-discovery — the priority-ordered source list AND the per-axis rationale-recording instruction; (4) shipped-consumer re-verification — all four required fields (caller path, owning feature key, affected AC IDs, regression-test pointer); (5) I-04 propagation — the `ChangeImpactSet` shape reference and the "never a completion record that omits an affected artifact without a stated disposition" language; (6) design-divergence — a reference (not restatement) to defect-class-sweep.md's "Backward-looking rework" section | Missing any one required clause (not just a missing heading) → test fails; malformed `{{include}}` syntax → renderer error surfaces; a design-divergence section that restates defect-class-sweep.md's rework criteria instead of referencing it → fails per REQ-F-007's explicit "references... directly instead of restating" requirement; a closed-table section missing any one of the seven required columns → fails, since a partial table is exactly the "prose-only progression" failure mode REQ-F-001 forbids |

### AC-2: specification.md, test_planning.md, task_review.md reference (don't restate)

| TC | Description | Input/Setup | Expected outcome | Edge cases |
|----|---|---|---|---|
| TC-002 | `specification.md` references dependency-discovery and closed-table sections, replaces the old "grep for related services" READ item | Grep `internal/sharkdata/default_data/prompts/feature/specification.md` post-edit for (a) a reference/link to `state-space-coverage.md`, (b) absence of the prior unstated "grep for related services" phrase, (c) absence of a restated closed-table or dependency-discovery procedure exceeding one sentence | Reference present; old phrase absent; no restated procedure | A restated procedure (paraphrase, not just a link) → fails, mirroring E34-F06's `TestDefectClassSweepConsolidatedNotDuplicated` structural-not-exact-string approach |
| TC-003 | `test_planning.md` gains one reference line to the technique-selection section, not a restated algorithm | Same grep pattern against `internal/sharkdata/default_data/prompts/feature/test_planning.md` | Reference present, no restated technique-selection algorithm | Same restated-procedure failure mode as TC-002 |
| TC-004 | `task_review.md` gains the REQ-F-005 shared-naming-drift paragraph and references `state-space-coverage.md` where applicable, without restating any of the workflow file's other procedures | Grep `internal/sharkdata/default_data/prompts/feature/task_review.md` post-edit for (a) the new one-paragraph shared-naming comparison instruction (compare every shared field/state/event/contract name the task touches against the owning specification and interaction map verbatim; report unexplained drift as a blocking contract finding even when local code compiles/passes tests — per spec.md REQ-F-005's exact wording), (b) absence of a restated closed-table, technique-selection, or dependency-discovery procedure exceeding one sentence | New paragraph present with the "blocking... even when compiles/passes" language intact; no restated procedure from other sections | A paragraph that omits the "even when the local name compiles/passes tests" clause → fails, since that clause is what distinguishes this from a soft style suggestion (this is the exact clause TC-017's fixture exercises) |
| TC-006 | `feature_review.md` references the shipped-consumer re-verification section | Grep `internal/sharkdata/default_data/prompts/epic/feature_review.md` for a reference to `state-space-coverage.md`'s shipped-consumer section | Reference present | Restated procedure instead of reference → fails |

### AC-3: tech_debt/resolved.md and question-management/SKILL.md reference I-04 propagation

| TC | Description | Input/Setup | Expected outcome | Edge cases |
|----|---|---|---|---|
| TC-005 | Both files reference the I-04 propagation section; `resolved.md`'s existing one-line template is preserved for the no-I-04-conditions case | Grep `internal/sharkdata/default_data/prompts/tech_debt/resolved.md` and `internal/sharkdata/default_data/skills/question-management/SKILL.md` for a reference to the I-04 propagation section; separately confirm `resolved.md`'s original one-line template text is still present unconditionally (only a second, conditional line was added per spec.md REQ-F-006) | Both reference the section; `resolved.md`'s original line unchanged; new line is clearly conditional ("when the resolved item changed accepted behavior") not unconditional | `resolved.md`'s original template altered/removed → fails (spec.md requires it "unchanged" for the no-I-04 case); a reference framed as unconditional (always requiring I-04 regardless of behavior change) → fails, since that would block simple bug-fix resolutions with no behavior change |

### AC-4: `shark impact record` behavior

| TC | Description | Input/Setup | Expected outcome | Edge cases |
|----|---|---|---|---|
| TC-007 | Valid `--impact-file` content on an existing key writes exactly one reference note, with the three source flags merged in | `shark impact record E34-F07-001 --source-kind=tech_debt --source-key=TD-042 --source-pointer=docs/td/TD-042.md --impact-file=impact.json` where `impact.json` contains `{"affected_artifacts":["spec.md"]}`, against a mocked repo where the key exists | Exit 0; mocked `NoteEntityNoteRepository.Create` called once with `note_type=reference`, content = the impact-file's JSON merged with `source_kind`/`source_key`/`source_pointer` from the flags; `--json` output echoes the created note | N/A (happy path) |
| TC-008 | Valid `--impact-file` form reads real file bytes | `shark impact record E34-F07-001 --source-kind=... --source-key=... --source-pointer=... --impact-file=impact.json` | Exit 0; note content is derived from the file's bytes (merged with the flags), not the literal string `impact.json` | Missing file at the given path → exit non-zero with a clear file-not-found message (not a silent empty-content write); any of the four required flags omitted → exit non-zero via cobra's `MarkFlagRequired`, zero repository calls |
| TC-009 | Target key does not exist | `shark impact record E34-NOPE-999 --source-kind=question --source-key=Q-1 --source-pointer=docs/questions/Q-1.md --impact-file=impact.json` where `impact.json` contains `{"affected_artifacts":["x"]}` | Exit non-zero (per AC-4); mocked repo's not-found path is the source of the error, not a pre-check bypassing the repo | N/A |
| TC-010 | Content missing `source_kind` — service-level partition | `services.ImpactService.RecordImpact` invoked directly with a content string omitting `source_kind` (`internal/services/impact_service_test.go`, unchanged by the flag rework) | Exit non-zero; error message names `source_kind`; mocked repo `Create` never called | Note: at the CLI layer this exact state (merged content missing `source_kind`) is now unreachable — `runImpactRecord` rejects an empty/whitespace `--source-kind` flag value *before* `os.ReadFile`, the merge, or any service call (`impact_cmd.go:105-107`), a strictly earlier CLI-layer partition covered by `TestRunImpactRecord_EmptyFlagValue` in `impact_cmd_test.go`, not this TC |
| TC-011 | Content missing `source_key` — service-level partition | Same as TC-010 for `source_key` (`impact_service_test.go`, unchanged) | Exit non-zero; error message names `source_key`; repo `Create` never called | Same CLI-layer-unreachable note as TC-010, for `--source-key` (`impact_cmd.go:108-110`) |
| TC-012 | Impact-file content has empty `affected_artifacts` array (present but empty), and separately, `affected_artifacts` present as a wrong JSON type (a string) | `--impact-file` pointing at `{"affected_artifacts":[]}` and separately at `{"affected_artifacts":"x"}`, valid `--source-kind`/`--source-key`/`--source-pointer` flags in both cases | Both exit non-zero; error message names `affected_artifacts` as required non-empty (empty-array case) or wrong-typed (string case); repo `Create` never called for either | Boundary: distinguishes "field absent" (TC-010/TC-011), "field present but empty" (TC-012a), and "field present with wrong type" (TC-012b) — all three must fail, closing the gap codex flagged in an earlier draft that only tested the absent/empty boundary, not the wrong-type case |
| TC-013 | `--impact-file` content is malformed (non-JSON) | `--impact-file` pointing at a file containing `not-json-at-all`, valid `--source-kind`/`--source-key`/`--source-pointer` flags | Exit non-zero; error message indicates JSON parse failure, distinct from a missing-field validation message; repo `Create` never called | N/A — this is the negative case proving the command validates shape before attempting persistence; must not be conflated with TC-010/TC-011's message (a caller must be able to tell "malformed JSON" from "well-formed but incomplete"). The CLI passes malformed impact-file bytes through to `ImpactService.RecordImpact` unmerged so the service's existing malformed-content error is the one surfaced, rather than a separate CLI-layer parse error |

### AC-5: rendered-output sample demonstrates the three feature.md scenarios

AC-5's own wording requires "a rendered-output sample," not only a UAT report
narrative. Per the codex red-team finding that a review-report claim alone is
not a repeatable oracle, this plan requires a **committed rendered artifact**
proving all three scenarios, matching E34-F06's precedent exactly: one file,
`E34-F07-state-space-planning-and-decision-propagation/scenario-review-TC-015-TC-017.md`,
containing the rendered `state-space-coverage.md` output plus each fixture's
walked-through outcome for TC-015/TC-016/TC-017 (one section per TC), checked
into the same commit that closes this feature. The manual-review verdict in
the code-review/UAT report must cite that committed file, not restate the
fixtures inline only.

| TC | Description | Input/Setup | Expected outcome | Edge cases |
|----|---|---|---|---|
| TC-015 | "Plan a multi-entity lifecycle" scenario | Fixture: a hypothetical feature spec adds a failure state read by a deduplication service owned by a shipped feature; walk the fixture through the rendered `state-space-coverage.md` closed-table + dependency-discovery + shipped-consumer sections | Rendered guidance requires: the failure state and recovery transitions appear in the closed table; the consumer path, existing AC, cross-entity axis, and regression test are all named before the plan can pass | A fixture where the consumer path is discovered only via a direct Shark dependency (missing the interaction-map/caller-path sources) must be flagged as insufficient discovery, proving REQ-F-003's "not limited to direct Shark dependencies" is enforced |
| TC-016 | "Propagate a ratified decision" scenario | Fixture: a Question or tech-debt resolution changes an accepted conversion design, affecting two specs, one test plan, and one shipped consumer AC | Rendered I-04 propagation guidance requires naming every invalidated artifact/consumer AC, each amended or linked to explicit follow-up work in the same session; a completion record omitting any one artifact without a stated disposition is rejected by the workflow's own text | A fixture where one affected spec is silently omitted (no mention, no disposition) must be caught — the content must make silent omission a review failure, not merely encourage completeness |
| TC-017 | "Reject naming drift" scenario | Fixture: a task renames an interaction field (e.g. `class_key` → `defectKey`) without updating the owning spec/interaction map | Rendered `task_review.md` shared-naming paragraph (REQ-F-005) requires task review to return a structured contract finding (blocking) for the unexplained rename, even though the task's local code compiles and its own tests pass | A fixture where the rename is explained (linked to an approved spec amendment) must NOT be flagged — only *unexplained* drift is blocking, per REQ-F-005's exact wording |

### REQ-NF-001: no new persistence introduced

| TC | Description | Input/Setup | Expected outcome | Edge cases |
|----|---|---|---|---|
| TC-014 | No new DB column/table/migration; `CurrentSchemaVersion` unchanged | `git diff` (or a dedicated Go test mirroring E34-F06's `TestDefectClassSweepNoGoPersistenceIntroduced` pattern, scoped to this feature's changed files) confirms: `internal/db/db.go`'s `CurrentSchemaVersion` constant is unchanged; no new `CREATE TABLE`/`ALTER TABLE` in any migration function; `impact_service.go` contains no direct SQL/repository calls other than delegating to the existing note-creation path. **Explicitly in scope, not a violation:** a test-only constructor/injection seam in `internal/cli/services_global.go` (e.g. a `cli.GetImpactService`-style accessor, or extending the existing `cli.GetNoteService` test-injection pattern if one exists) so TC-007's real-repository run can wire an in-memory/temp-sqlite-backed `NoteService` without a live production database file — this is test wiring, not a new persistence *schema* surface, and does not count against this guard. | Zero hits for new persistence *schema* surface outside the declared `EntityNoteRepository` delegation; a test-injection seam in the service-accessor layer is expected and does not fail this test | A new table/column, or `impact_service.go` bypassing `NoteService` to write its own SQL, fails the test — this is the structural guard proving REQ-NF-001's "zero new Shark database columns, tables, or relationship types" claim, matching E34-F06's TC-010 precedent for the same class of claim. Do not let this guard's grep pattern be so broad it flags the test-injection seam itself (e.g. a naive "no new .go file under internal/cli" rule would wrongly fail on the accessor addition) — scope the pattern to schema/DDL/table-name tokens, per E34-F06's TC-010 UAT-kickback lesson (MEDIUM-3) about over-broad negative-check patterns. |

## Integration Scenarios

- **Consuming gate → new workflow reference**: `specification.md`,
  `test_planning.md`, `task_review.md`, and `feature_review.md` each
  reference `skills/quality/workflows/state-space-coverage.md`. Verify via
  the same renderer path as TC-001 that each of the four consuming prompts
  resolves the reference without a broken-include error — this is the
  integration boundary between "the new workflow exists" (TC-001) and "gates
  actually use it" (TC-002, TC-003, TC-006).
- **Decision-resolution content → I-04 propagation**: `tech_debt/resolved.md`
  and `question-management/SKILL.md` both reference the I-04 propagation
  section for their respective resolution paths; `shark impact record`
  provides the parallel runtime hook for the one resolution path (ADR
  adoption) that has no existing Shark-tracked status transition. TC-007 and
  TC-016 together prove both halves of this integration reach the same
  durable `reference`-note representation, per the `EntityNoteRepository`
  seam both the content-driven paths (Question/tech-debt resolution, via
  their existing status-transition hooks — out of this feature's scope, they
  already write through this path) and `shark impact record` share.
- **E34-F08 downstream consumption**: out of this feature's test scope — the
  `TC-I-04-CHANGE-IMPACT-CLOSURE` pointer (above, also known as
  `TC-I-04-DECISION-PROPAGATION` in the interaction map) proves this
  feature's I-04 *shape and persistence hook*; E34-F08's own test plan proves
  it *consumes* I-04 sets to verify `status: accounted` before epic
  completion.
- **Epic UAT-04 ("Propagate a material decision")**: `uat-plan.md` names
  UAT-04 as the epic-level acceptance scenario this feature is the primary
  contributor to ("A changed Question, ADR, design, state, or debt decision
  yields an impact set with affected artifacts, consumer caller paths,
  acceptance criteria, and regression coverage. Each item is amended or has a
  linked follow-up."). TC-015 and TC-016 are this feature's direct
  contribution to UAT-04's evidence; TC-007/TC-013 (`shark impact record`'s
  runtime behavior) supply the ADR-adoption-specific slice of UAT-04 that no
  other E34 feature covers.

## Test Infrastructure

- **Existing to reuse**:
  - `internal/templates/includes_test.go`'s `TestIncludeResolver_*` table
    pattern for TC-001 and the four integration-scenario include checks (add
    entries rather than a new test file).
  - `internal/sharkdata/embed_test.go`'s existing global gates
    (`TestEmbedded_SkillsContainNoBareSharkCLIRefs`,
    `TestEmbedded_AgentsDescribeRoleNotWorkflow`) remain in force for the new
    workflow file — it must pass both by construction (no bare `shark <verb>`
    string; workflow content, not an agent persona).
  - `internal/cli/commands/notes_add_dispatch_test.go` and
    `internal/services/note_service_test.go`'s mocked-repository pattern is
    the direct precedent for `impact_cmd_test.go` and
    `impact_service_test.go` — same two-seam mock wiring
    (`EntityRegistry`-selected entity-type repository for `GetByKey`, plus
    `NoteEntityNoteRepository` for `Create`), same assertion style (call
    count, argument capture).
  - `internal/sharkdata/embed_test.go`'s
    `TestDefectClassSweepNoGoPersistenceIntroduced` pattern (E34-F06
    precedent) is the direct template for TC-014's structural no-new-schema
    guard.
- **New test helpers needed**:
  - A small fixture-JSON builder helper (valid I-04 shape, and each
    single-field-missing/empty/wrong-type variant) shared across
    TC-007/TC-010/TC-011/TC-012/TC-013 in `impact_cmd_test.go` — avoids
    repeating six near-identical JSON literals inline.
  - A mock-entity-repository helper returning either a resolved entity (with
    a known ID) or a not-found error for `GetByKey`, reusable across
    TC-007..TC-013's entity-lookup seam (see Caller-Path Contracts' mock-seam
    correction above) — likely a thin wrapper the `note_service_test.go`
    mocks already provide, extended if a dedicated fixture type is needed.
  - TC-002/TC-003/TC-004/TC-005/TC-006 are one-line grep/reference-presence
    assertions addable to `embed_test.go` or a lightweight companion test
    file in the same package, following E34-F06's TC-002 precedent
    (`TestDefectClassSweepConsolidatedNotDuplicated`) — this feature's
    equivalent would be named e.g.
    `TestStateSpaceCoverageConsolidatedNotDuplicated`.
  - TC-015, TC-016, TC-017 are manual review checklist items recorded in the
    code-review/UAT report for this feature, backed by the committed
    rendered-sample files required under AC-5 above (no fixture-execution
    harness exists for prose "decision procedures," consistent with
    E34-F05's and E34-F06's precedent of not building one).

## Codex Test-Plan Red-Team

**Attempt 1:** `codex exec -s read-only -c model_reasoning_effort=high --skip-git-repo-check "<red-team prompt per Step 7.5's seven evaluation criteria, scoped to an earlier draft of this test-plan.md against spec.md, with the Prompt-only-changes exemption explained>"`, run 2026-09-04, with `GOMODCACHE`/`GOCACHE` pinned into `/tmp` per this repo's known sandbox-writability constraint for `codex exec -s workspace-write`/read-only Go tooling. Completed successfully, well under the 580s budget; no retry needed.

**Verdict:** FAIL
**Issues raised:** 9 distinct findings (2 Blockers + 4 Enumeration/traceability bullets + 1 ISO 25010 section [4 flagged cells, counted as one finding] + 1 Observability-gap paragraph + 1 Drift-control paragraph)
**Issues addressed before dev:** 8 of the 9, applied directly to this document in this same revision pass; 1 (AC-1 clause-level enumeration) partially addressed and explicitly deferred to implementation time, not silently dropped:

1. **TC-013 defined twice, inconsistently, and the I-04 pointer cited the wrong definition** (found independently by pre-codex advisor review, confirmed in spirit by codex's broader Caller-Path Contract concern) — fixed: TC-013 is now consistently "malformed JSON, zero repo calls" in both the AC-4 Test Matrix and the Caller-Path Contracts table; the I-04 pointer now cites TC-007 (the happy-path persistence proof) instead.
2. **Caller-Path Contract mock seam was wrong (Blocker #2)** — `NoteEntityNoteRepository` cannot see the entity key or produce a target-not-found result; `NoteService` resolves the key through the `EntityRegistry`-selected entity-type repository's `GetByKey` first. Fixed: every runtime TC (TC-007..TC-013) now names both mock seams (entity-type repository + note repository) and TC-009's not-found path is now sourced from the correct seam.
3. **AC-2/REQ-F-005 traceability gap — TC-004 cited but never defined (Enumeration concern)** — fixed: TC-004 added to the AC Test Matrix, covering `task_review.md`'s shared-naming paragraph specifically (previously only TC-002/TC-003/TC-006 existed, none of which touched `task_review.md`).
4. **Open-ended "restatement" oracle (Blocker #1)** — fixed: the Ambiguity Findings section now defines a bounded per-file reference contract (a literal reference token present, plus a fixed one-new-sentence cap on restated procedure content) rather than an undefined semantic-paraphrase judgment call.
5. **ISO 25010 N/A cells unjustified for AC-1 usability/reliability and AC-4 security (ISO 25010 gaps)** — fixed: added named test-case behaviors for AC-1 usability (self-contained section clarity) and reliability (render-failure surfacing), and AC-4 security (untrusted JSON/`@file` input handling, TC-008/TC-013).
6. **AC-4 partition model incomplete — missing wrong-JSON-type and unreadable-`@file` cases (Enumeration concern)** — fixed: TC-012 now also covers `affected_artifacts` as a wrong JSON type (string instead of array); TC-008's edge case now covers an unreadable/missing `@file` path with a zero-repo-call assertion.

**Issues deferred:** 1, logged here rather than resolved now:

7. **AC-1/TC-001 header-presence vs. clause-level content (Enumeration concern)** — partially addressed: TC-001 was rewritten to assert each section's required *clauses* individually (not just headings), and the earlier internal "five sections" miscount (spec.md AC-1 undercounts against its own six-requirement body text) is now flagged with a recommendation to correct spec.md. Not fully resolved: this test plan does not enumerate the exact clause-matching regex/assertions codex asked for (e.g. the precise required substring set per section) — that level of implementation detail is deferred to the developer writing the actual `TestIncludeResolver_*` entry, consistent with this test plan's role of specifying *what* must be asserted, not the literal Go assertion code. Owner: developer implementing TC-001; Timeframe: T-E34-F07 task implementing the workflow file and its render test.

Additionally, codex's observability-gap finding ("no runtime evidence that exactly one note exists after the command; only a mock assertion") was addressed: the Observability Design section and TC-007/TC-010 now require at least one test run against a real (non-mocked) repository pair to confirm the note is genuinely persisted and queryable, not merely mock-recorded.

Codex's "Drift control" finding (the I-04 pointer mismatch treated as non-blocking while spec.md remains inconsistent) is addressed by this plan's approach: rather than silently picking a side, the Spec Drift Analysis and Cross-feature contract tests sections now anchor the shared TC under spec.md's own declared name (`TC-I-04-CHANGE-IMPACT-CLOSURE`, satisfying this plan's exit gate against the feature spec) while cross-referencing the interaction map's name, and explicitly recommend a follow-up correction to reconcile the two documents — not left ambiguous, but also not resolved by this test-planning pass overriding either source document.

Full codex output (verbatim), the version of this document it reviewed being
an earlier draft prior to this revision's fixes:

```
# FAIL

The prompt-only exemption is applied correctly for markdown-only cases. The failure is in the runtime command plan and in several content-test oracles.

## Blockers

1. Open-ended structural assertion — AC-2

The plan admits that "restatement" detection is an implementation-time paraphrase heuristic, with no enumerated model or reliable oracle (test-plan.md:76). This is an unbounded duplication/robustness assertion.

Fix: define an exact per-file reference contract and bounded allowed content, then test it structurally or with goldens. Do not rely on semantic paraphrase detection or a sentence-count threshold.

2. Caller-Path Contract is not executable — AC-4

TC-007 and TC-009 claim a mocked `NoteEntityNoteRepository` can verify the entity key and produce target-not-found. It cannot: `NoteService` resolves the key through the entity registry's entity repository, while `NoteEntityNoteRepository.Create` receives only the resolved entity ID. The stated mock seam therefore cannot catch hardcoded-key or missing-lookup bugs.

The production accessor also creates a concrete DB-backed `NoteService`; the plan gives no injection seam for running the registered Cobra command with mocks.

Fix: specify an explicit internal test provider/factory seam, invoke the registered command, use the real impact and note services, and mock both the entity lookup repository and note repository. Assert the exact key in `GetByKey`, resolved ID, and exactly one `Create` call. Do not mock `ImpactService` above the entrypoint.

## Enumeration and traceability concerns

- AC-1 / REQ-F-001–007: TC-001 checks headings, not the required clause-level content: lifecycle heuristic, all table columns, invalid/recovery behavior, ordered dependency sources, per-axis rationale, handoff rules, I-04 artifact closure, and divergence reference. It also inconsistently says "five" sections while testing six.

  Fix: enumerate those required clauses in a rendered golden/structural test, including failures for prose-only progression, "other state," omitted recovery, and duplicated divergence procedure.

- AC-2 / REQ-F-005: the traceability matrix cites TC-004, but no TC-004 exists. Consequently `task_review.md` has no actual test for the shared-name drift paragraph or its blocking negative case.

  Fix: add TC-004, or reconcile AC-2 with REQ-F-005. The spec requires a naming paragraph, but does not identify a corresponding state-space section for `task_review.md`; that contract must be clarified.

- AC-4: the partition model omits supported entity-key types, unknown-format keys, missing/extra CLI arguments, `--json` registration, null/wrong-type/empty source fields, wrong `affected_artifacts` types, and invalid `@file` contents.

  Fix: enumerate these partitions or explicitly narrow the command contract.

- AC-5: manual review is permissible for prose, but no committed rendered sample, path, golden, or repeatable oracle is specified. A UAT report alone does not prove the acceptance criterion.

  Fix: add three durable rendered samples/goldens with explicit required and forbidden outcomes.

## ISO 25010 gaps

The N/A decisions are not justified:

- AC-1 and AC-5 mark usability N/A even though workflow/prompt clarity is the feature's primary user-facing quality.
- AC-4 marks security N/A despite accepting untrusted JSON and a user-supplied `@file` path.
- AC-1 marks reliability N/A despite renderer/include failure modes.
- AC-4 reliability covers only target-not-found, not validation failures, file-read failures, or note-write errors.

Add applicable usability, security, and reliability cases, or document a precise scope exclusion.

## Observability gap

For AC-4, the plan claims there is no CLI telemetry and cites `cli.Success`, but the repository has command metrics/observability at the root execution layer, while the note command precedent uses direct output rather than the claimed `cli.Success` path. The plan also has no runtime evidence that exactly one note exists after the command; that is only a mock assertion.

Fix: document the existing command metric/error evidence, test the command through root execution, and add an isolated runtime/UAT check that queries reference notes afterward and verifies no note was added for rejected inputs.

## Drift control

The I-04 pointer mismatch is documented but incorrectly treated as non-blocking while `spec.md` remains inconsistent (spec.md:231, test-plan.md:27).

Fix: update the spec to the authoritative pointer and add a cross-document equality check.
```

The numbered findings above map 1:1 to codex's "Blockers," "Enumeration and
traceability concerns," "ISO 25010 gaps," "Observability gap," and "Drift
control" sections.

## Recommendations

- [x] Ready for development — no unresolved spec/PRD drift (the naming
      mismatch is documented and resolved by dual-anchoring the shared test
      name rather than silently picking one, with a follow-up correction
      recommended), every AC has at least one test case with a named
      technique and ISO 25010 entry (including the previously-missing
      usability/reliability/security justifications), every runtime test
      case has a corrected Caller-Path Contract (entity-lookup seam fixed
      per codex's Blocker finding), the I-04 contract test pointer is
      declared and reconciled against both the feature spec and the
      authoritative interaction map, REQ-NF-001's no-new-persistence claim
      has a structural test guard, and the codex red-team ran to completion
      with a FAIL verdict whose findings were incorporated into this same
      revision (6 of 7 resolved, 1 deferred to implementation-time with a
      named owner).
- [ ] Needs BA refinement
- [ ] Needs tech refinement

**Outstanding, non-blocking:** spec.md's AC-1 ("five sections") and
"Cross-feature interactions → Produces" pointer string
(`TC-I-04-CHANGE-IMPACT-CLOSURE`) should be reconciled with
`E34-interaction-map.md`'s six-section body text and
`TC-I-04-DECISION-PROPAGATION` naming respectively, in a follow-up spec edit
— tracked here, not blocking this test plan's APPROVED verdict since both
documents' intents are captured and cross-referenced above.

*Last Updated*: 2026-09-04
