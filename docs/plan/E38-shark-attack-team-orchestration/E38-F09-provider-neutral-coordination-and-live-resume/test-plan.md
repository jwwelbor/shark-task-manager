# Test Plan: E38-F09 — Provider-Neutral Coordination and Live Resume

**Created:** 2026-08-01
**Feature PRD:** [feature.md](feature.md) / [spec.md](spec.md) (incremental over [epic.md](../epic.md) and [architecture.md](../architecture.md))
**Research report:** [research-report.md](research-report.md)
**Status:** APPROVED

No `test-plan.md` existed for this feature prior to this pass (confirmed by
directory listing before drafting). This is a feature-level plan — no task
spec exists yet — so Step 3 "task spec vs. feature PRD" drift detection is
not applicable; instead this plan validates spec.md's own internal
traceability against epic.md/architecture.md and the interaction maps, per
the READ/PRODUCE instructions in the dispatch prompt.

---

## Spec Drift Analysis

### Drift findings

spec.md already carries an unusually complete self-audit (its own
Exit-gate checklist, the "Degraded upstream dependencies" table, and a
Requirement→Traceability column on every REQ row). Re-verified against
epic.md, architecture.md, `E38-interaction-map.md`, `E38-cross-epic-map.md`,
`E39-cross-epic-map.md`, and the live production tree:

1. **No scope creep or narrowing found.** Every REQ-F-### traces to a named
   epic §2/§3 clause, an I-## row, or an X-## row. `internal/runner/` is
   explicitly out of scope (D-002) and the file table lists exactly two
   production Go files (`internal/cli/commands/next.go` for the provenance
   wire change, and `internal/sharkdata/embed.go` for the `capability_profile`
   roster-validator extension and `model_tier` deprecation warning — both
   declared explicitly, see spec.md Architecture) plus three new test files
   and a skill-content tree — verified against the live tree: no
   `prompt_sha256`, `PromptSHA256`, or `--prompt-out` symbol exists anywhere
   in `internal/` today, and `skills/shark-rider/verbs/run.md` contains zero
   occurrences of "question" (grep-verified), confirming REQ-F-011 and
   REQ-F-005/006's gap claims are accurate, not aspirational. `embed.go`'s
   inclusion is consistent with this plan's own TC-003-04..07 table row and
   the Integration Scenarios row below, both of which already target
   `embed.go:1169`'s `allowedMember` map directly.
2. **Confirmed, not drift: F08/F05 degraded-upstream handling.** spec.md's
   claim that E38-F08 and E38-F05 report Shark `completed` with no
   implementation reachable from `main` is independently re-verifiable
   (`internal/models/council_artifact.go`, `internal/repository/council_artifact_repository.go`,
   `internal/services/council_artifact_service.go`,
   `internal/cli/commands/admin_council.go` all absent; `related_docs.go` has
   no tech-debt route). REQ-F-018 and the "Degraded upstream dependencies"
   table correctly scope F09 to build its own provenance rather than assume
   F08's Tranche A landed. This test plan asserts F09's independence from
   both gaps directly (TC-009).
3. **Minor spec gap (non-blocking): AC-016's "sync helper" is not named as a
   file to create.** The architecture component table lists only
   `internal/sharkdata/shark_attack_parity_test.go` (a Go parity *check*);
   no `scripts/`-level "sync helper" exists in the repo today (verified: no
   `scripts/*sync*shark-data*` file). AC-016 assumes "running the sync helper
   then the check is clean" as a two-step story. This plan treats the sync
   step as **implementation-defined** (a `cp -r skills/shark-attack/*
   internal/sharkdata/default_data/skills/shark-attack/` invocation, a
   Makefile target, or a Go helper — spec.md does not mandate the mechanism)
   and designs TC-008 to assert the *outcome* (post-sync byte parity) rather
   than a specific sync command. Flagged for the task-spec pass to either
   name the mechanism or confirm manual copy is intended.
4. **X-06 shape source confirmed live.** `internal/services/question_service.go`,
   `internal/cli/commands/question.go`, and `internal/models/question.go`
   are present and wired (`QuestionBlock` on `NextResponse` at
   `internal/cli/commands/next.go:175`). The consumer-activation obligation
   E39-F04 recorded against F09 is real and undischarged prior to this
   feature; TC-004 in this plan is new coverage, not a duplicate of
   `e39_interactions_test.go#TC-004` (verified: that test proves the
   *producer* read-only surface only — it never links a `question_blocks`
   edge to a **non-Question** entity via the full mint→route→respond→resolve
   loop through `shark next Q###` dispatch of the responder).

No BLOCKER-level drift found. spec.md is buildable as written.

### Traceability matrix

| Feature PRD requirement | Acceptance criteria | Covered? | Notes |
|---|---|---|---|
| REQ-F-001/002 (two-axis independence, sequential default) | AC-001, AC-002 | Yes | TC-005 |
| REQ-F-003 (control envelope, opaque `recommended_outcome`) | AC-003 | Yes | TC-003 |
| REQ-F-004/005/006 (Question mint/route/respond/resolve, no bespoke store) | AC-004, AC-007, AC-008, AC-009 | Yes | TC-004 |
| REQ-F-007 (heartbeat, no advance during consultation) | AC-005, AC-006, AC-010 | Yes | TC-001, TC-004 |
| REQ-F-008 (same-worker resume vs. bounded replacement) | AC-021 | Yes | TC-010 |
| REQ-F-009 (silent responder ping/replace/deadline) | AC-012 | Yes | TC-010 |
| REQ-F-010 (lease loss stops mutation) | AC-011 | Yes | TC-010 |
| REQ-F-011 (prompt hash/bytes, `--prompt-out`) | AC-013 | Yes | TC-002 |
| REQ-F-012 (capability discovery before topology) | AC-022 | Yes | TC-010 |
| REQ-F-013 (Codex/Claude Code references) | AC-015 | Yes | TC-003 |
| REQ-F-014 (`capability_profile` additive, `model_tier` deprecated) | AC-014 | Yes | TC-003 |
| REQ-F-015 (SKILL.md router restructure) | AC-017 | Yes | TC-003 (content-only) |
| REQ-F-016 (parity gate) | AC-016 | Yes | TC-008 |
| REQ-F-017 (`pull-by-role.md` retirement) | AC-018 | Yes | TC-007 |
| REQ-F-018 (degraded-upstream honesty) | — (structural) | Yes | TC-009 |
| REQ-NF-001…007 | AC-019, AC-020, AC-023, AC-024 | Yes | TC-002, TC-006, TC-009, TC-010, plus `go test ./internal/runner/...` diff-review gate |

---

## ISTQB Technique Application (per AC)

| AC | Technique(s) applied | Test cases | Rationale |
|---|---|---|---|
| AC-001, AC-002 | Decision Table (coordination level × topology, 2 independent axes) | TC-005-01..13 (full decision table added post-red-team) | Two-axis independence is a classic decision-table shape: each axis must vary without affecting the other |
| AC-003 | Equivalence Partitioning (known vocabulary vs. unknown/opaque outcome key) | TC-003-01..03 | `recommended_outcome` must round-trip regardless of whether it's in the control vocabulary |
| AC-004 | State Transition (`draft`→`open`→`answering`→`ready_for_resolution`→`resolved`) + Decision Table (link-before-configure vs. configure-before-link ordering) | TC-004-01..04 | Question lifecycle is a state machine; the load-bearing ordering constraint (D006, question_blocks:82-103) is a boundary between two orderings |
| AC-005, AC-006 | State Transition (blocked vs. unblocked gate) + Attack-class enumeration (bypass via `status set`) | TC-004-05..07 | Gate bypass is a defensive-property AC ("must not be advanceable") |
| AC-007, AC-008, AC-009 | Contract surface enumeration (question CLI: create/configure-workflow/next/respond/resolve) | TC-004-08..12 | Consumer-activation coverage over a documented producer contract |
| AC-010, AC-011 | State Transition (lease alive vs. lease lost) + BVA (heartbeat within/at/past TTL) | TC-001-01..03 | Claim lease is a bounded-lifetime resource |
| AC-012 | Decision Table (responder silent × ping × replace × deadline) | TC-010-01..04, TC-010-17..20 (full ladder added post-red-team) | Escalation ladder is a multi-condition decision table |
| AC-013 | BVA + Attack-class enumeration (adversarial byte fixture: newlines, code fences, quotes, Unicode, shell metacharacters) | TC-002-01..06 | Byte-exactness claim under adversarial input is the textbook case for adversarial BVA |
| AC-014 | Equivalence Partitioning (`capability_profile` valid/invalid values × `model_tier` present/absent × both/neither) | TC-003-04..07 | Roster validator accepts a bounded value set per field, with a legacy-compat class |
| AC-015 | Contract surface enumeration (per-provider supported/unsupported operation matrix) | TC-003-08..09 | Provider reference is itself a capability contract surface |
| AC-016 | Attack-class enumeration (byte drift injected into either tree; embedded-only file) | TC-008-01..03 | "No drift possible" is a defensive property |
| AC-017 | Contract surface enumeration (fresh-agent reachability for 3 scenario classes) + Attack-class enumeration (near-duplicate rule detection) | TC-003-11, TC-003-12 (content-only, redesigned post-red-team) | Router-file navigability across 3 known destinations; "no rule duplicated" requires a rule inventory, not a single-sentence spot check |
| AC-018 | Attack-class enumeration (any sanctioned-claim phrase surviving in `pull-by-role.md`, or a live child-claim authorization anywhere in the rendered corpus) | TC-007-01, TC-007-04 (content-only) | Confirms the *absence* of a previously-sanctioned path, corpus-wide |
| AC-019 | State Transition (single atomic commit, no intermediate red) | TC-CI-01 (build/CI-level, not unit) | Verified via `git log`/CI green-at-HEAD, not a Go test |
| AC-020 | Equivalence Partitioning (unchanged-diff class) | TC-CI-01 | Diff-scope assertion |
| AC-021 | Decision Table (resume-supported × resume-unsupported) | TC-010-05..07 | Capability flag drives exactly one of two disjoint code paths |
| AC-022 | Decision Table (isolation/follow-up/interrupt detected × undetected, 3 independent capabilities) | TC-010-08..11 | Capability-discovery-before-selection ordering across 3 independent capabilities |
| AC-023 | Decision Table (interrupt supported × unsupported) | TC-010-12..13 | Fallback behavior branches on one capability flag |
| AC-024 | Attack-class enumeration (rendered-prompt leakage across every artifact/note/telemetry surface) | TC-002-07, TC-010-14 | "Never persisted" is a defensive property requiring enumeration of every write surface |

ACs without a technique annotation: none — every AC above has at least one
technique. The "out of scope" bullets in spec.md require no test coverage.

---

## ISO 25010 Coverage Matrix

| AC | Functional | Performance | Compat | Usability | Reliability | Security | Maintainability | Portability |
|---|---|---|---|---|---|---|---|---|
| AC-001/002 | ✅ TC-005-01..13 | N/A | N/A | N/A | ✅ TC-005-05/09 (both independent degrade-to-sequential paths) | N/A | N/A | N/A |
| AC-003 | ✅ TC-003-01..03 | N/A | ✅ TC-003-02 (unknown outcome key doesn't break existing consumers) | N/A | N/A | N/A | N/A | N/A |
| AC-004 | ✅ TC-004-01..04 | N/A | N/A | N/A | N/A | N/A | N/A | N/A |
| AC-005/006 | ✅ TC-004-05..07 | N/A | N/A | N/A | N/A | ✅ TC-004-06 (`status set` bypass documented, not silently open) | N/A | N/A |
| AC-007/008/009 | ✅ TC-004-08..12 | N/A | N/A | N/A | N/A | ✅ TC-004-11 (mismatched-lease response rejected) | N/A | N/A |
| AC-010/011 | ✅ TC-001-01..03 | N/A | N/A | N/A | ✅ TC-001-02..03 (lease loss stops mutation, no partial-authority write) | ✅ TC-001-03 (no answer delivery under lost authority) | N/A | N/A |
| AC-012 | ✅ TC-010-01..04, TC-010-17..20 | N/A | N/A | N/A | ✅ TC-010-02..04, TC-010-20 (bounded escalation, deadline boundary) | N/A | N/A | N/A |
| AC-013 | ✅ TC-002-01..06 | N/A | N/A | N/A | N/A | N/A | N/A | N/A |
| AC-014 | ✅ TC-003-04..07 | N/A | ✅ TC-003-06 (legacy `model_tier` rosters keep validating) | ✅ TC-003-05 (deprecation warning surfaced) | N/A | N/A | N/A | N/A |
| AC-015 | ✅ TC-003-08..09 | N/A | N/A | N/A | N/A | N/A | N/A | ✅ TC-003-08/09 (installed-host evidence per provider) |
| AC-016 | ✅ TC-008-01..03 | N/A | N/A | N/A | N/A | N/A | ✅ TC-008-01..03 (drift becomes structurally impossible) | N/A |
| AC-017 | ✅ TC-003-11/12 | N/A | N/A | ✅ TC-003-12 (fresh-agent reachability) | N/A | N/A | ✅ TC-003-11 (rule-inventory no-duplication check) | N/A |
| AC-018 | ✅ TC-007-01 | N/A | N/A | N/A | N/A | N/A | N/A | N/A |
| AC-019/020 | ✅ TC-CI-01 | N/A | ✅ TC-CI-01 (`internal/runner/...` unchanged) | N/A | N/A | N/A | N/A | N/A |
| AC-021 | ✅ TC-010-05..07 | N/A | N/A | N/A | ✅ TC-010-05..07 (exactly-one-worker invariant under both paths) | N/A | N/A | N/A |
| AC-022 | ✅ TC-010-08..11 | N/A | N/A | N/A | N/A | ✅ TC-010-08..11 (no unverified command issued on undetected capability) | N/A | N/A |
| AC-023 | ✅ TC-010-12..13 | N/A | N/A | N/A | N/A | ✅ TC-010-13 (no unverified provider command on unsupported interrupt) | N/A | N/A |
| AC-024 | ✅ TC-002-07, TC-010-14 | N/A | N/A | N/A | N/A | ✅ TC-002-07, TC-010-14 (no prompt text in any durable artifact) | N/A | N/A |
| REQ-NF-007 (hash latency) | N/A | ✅ TC-002-08 (single-pass SHA-256, reasoned review not benchmark — spec explicitly allows either) | N/A | N/A | N/A | N/A | N/A | N/A |

No empty cells. Performance is N/A everywhere except REQ-NF-007 because F09
adds no new query paths, loops, or network calls — the only measurable cost
is one SHA-256 pass already covered there. Compatibility applies wherever a
legacy shape (`model_tier`, unchanged `internal/runner/`, unknown outcome
keys) must keep working. Portability applies only to the two named provider
references.

### Coverage gaps

None identified. Every AC has at least one ✅ cell; every N/A is justified
by the AC's own shape (e.g., AC-013's byte-exactness claim has no usability
dimension).

---

## Observability Design (per behavior)

| Behavior | Metric | Log | Trace span | Alert threshold | Test assertion |
|---|---|---|---|---|---|
| Keyed-next prompt hashed and byte length recorded | reuse existing `prompt_bytes` OTel int attribute (`next.go:352`) — no new metric | none required (hash is a wire field, not an event) | existing `shark.next` span (`next.go:274`) carries `prompt_bytes`; F09 adds no new span | N/A | TC-002-08 asserts the wire `prompt_sha256`/`prompt_bytes` values equal the span attribute and the `--prompt-out` file hash |
| Question mint → gate → route → respond → resolve loop | none required — Question lifecycle is already the observability surface for E39; F09 adds no counter of its own | reuse Question's own history/audit trail (each `question respond`/`resolve` call is itself the evidence) | none required | N/A | TC-004 asserts state transitions land in `question_history`/status reads, which is the durable evidence |
| Lease loss stops mutation workers | none new — existing claim/heartbeat telemetry already covers lease lifecycle | none new | none new | N/A | TC-001-02/03 asserts no write occurs after simulated lease loss, using existing claim state as evidence |
| Bounded replacement creates exactly one worker | internal — no observability | internal — no observability | internal — no observability | N/A | TC-010-05..07 asserts worker-count invariant directly against the fixture harness, not runtime telemetry (this behavior lives entirely in skill/prose + fixture-level Go tests, not in a running service) |
| Provenance never contains rendered prompt text | internal — no observability (this is a negative/absence assertion, not an emitted signal) | internal — no observability | internal — no observability | N/A | TC-002-07/TC-010-14 grep every artifact write path in the fixture for prompt content |

**Implementation hook:** F09 introduces no new metric, log, or span — every
runtime behavior it adds either reuses existing OTel attributes already on
`shark.next` (`prompt_bytes`) or is proven structurally (parity gate,
byte-exactness, worker-count invariants) rather than through new runtime
instrumentation. This is a deliberate consequence of D-002 (no code under
`internal/runner/`, so there is no new dispatch runtime to instrument) and
should be called out explicitly if a later task spec proposes adding new
telemetry — it would be scope the feature spec does not require.

**Deferred CONCERN (codex red-team, not fixed in this pass):** codex flagged
that lease-loss, ping/interrupt, replacement, deadline-release, and
worker-stop events (AC-010–AC-012, AC-021–AC-023) have no production-runtime
observability — diagnosis would depend on reading skill prose or test
fixtures, not a running signal. This plan does not add a new operational
record for these events, because spec.md's own architecture section commits
to zero new Go runtime code (D-002) and lists no new metric/log/span in its
component-changes table — adding one would be scope this test plan cannot
unilaterally introduce without amending spec.md. **Deferred to: the task-spec
author / feature owner**, to decide before task decomposition whether a
bounded, non-secret operational record (entity/Q key, session/worker
identity, event type, capability fallback selected, timestamp, reason) is
in-scope for F09 or explicitly out-of-scope pending F10/F11's provider
conformance work. If accepted, it becomes a new REQ-NF row and this test
plan's observability table gains real metric/log/span cells for those five
rows instead of "internal — no observability."

**Decision recorded (task-decomposition rework pass):** explicitly
out-of-scope for F09, deferred to F10/F11's provider conformance work — no
new REQ-NF row added, no new metric/log/span introduced. Rationale:
D-002/D-011 hold F09 to zero new `internal/runner/` code, and F10/F11 own the
real provider-process integration where a running signal would first exist;
adding instrumentation now would be speculative ahead of that work. Recorded
in T-E38-F09-016's Brownfield Context (the task owning the ping/interrupt/
replacement/deadline procedures).

---

## Caller-Path Contracts (per test case)

| TC | Production entrypoint | Lowest mock seam | Forbidden mocks | Counter-factual |
|---|---|---|---|---|
| TC-002-01..06 (prompt provenance) | `resolveNext(ctx, cache *nextAdapterCache, entityType, key string, depth int) (NextResponse, error)` in `internal/cli/commands/next.go`, driven with the same `nextAdapterCache{entries, actionSvcRoot}` mock shape `TestResolveNext_ReturnsSelfContainedPrompt` already uses (`next_test.go:330`) | `action.MockActionService` (`GetStatusActionPopulatedFunc`) and `fixedNextTransitioner`/`fixedNextPlaceholders` — the same seam existing `next_test.go` mocks at | Do not call `assembleDispatchPrompt` directly with a hand-built string bypassing `resolveNext`; do not mock `sha256.Sum256` itself | An implementation that hashes the *pre-assembly* instruction (D-004's rejected alternative) would produce a hash that does not match `--prompt-out`'s bytes once the ownership preamble is prepended — this test's byte-for-byte comparison catches that |
| TC-002-04 (`--prompt-out` flag registration) | `nextCmd` cobra command registration + flag parsing (`internal/cli/commands/next.go` `init()`), verified the same way `TestNextCommandDoesNotExposeRemovedPreviewFlag` (`next_test.go:24`) verifies flag presence/absence | Cobra `Command.Flags()` introspection — no execution required for flag-registration assertions | Do not shell out to a built `shark` binary for this assertion — flag registration is a compile-time construct, not a process-boundary concern | An implementation that adds `--prompt-out` as a global persistent flag instead of scoping it to `next` would leak onto unrelated commands; the registration test would catch a flag appearing where the spec doesn't declare it |
| TC-002-09 (`--prompt-out` write path, **corrected per codex red-team BLOCKER**) | `runNext(cmd *cobra.Command, args []string) error` (`next.go:266`) driven end-to-end via real cobra `Execute()` against a temp SQLite DB (`db.InitDB`), i.e. the same real-process `tests/contracts` convention as TC-004 — flag introspection alone (the pre-fix TC-002-04 design) cannot prove file-write correctness, since `--prompt-out`'s file-write logic lives inside `runNext`'s body, not its flag declaration | Real `runNext` execution against a real temp-file DB and a real OS temp file target — no mock seam is valid here since the thing under test *is* the OS-level write | Do not assert only that `resp.Prompt`/`resp.PromptSHA256` match each other in-memory (that's TC-002-01..03's job) without also reading the actual bytes `runNext` wrote to disk | An implementation that computed the hash correctly but wrote `--prompt-out` through a text-mode writer (normalizing `\r\n`→`\n`) would pass every mocked `resolveNext`-level assertion while producing a file whose hash does not match; only a real-write, real-read-back test catches it |
| TC-003-01..03 (control envelope) | content-only — `skills/shark-attack/context/worker-control-schema.yaml` (new) rendered via `templates.NewIncludeResolver` the same way `TestTC004_X05EmbeddedSkillOverrideIsReplaceOnly` (`e38_f04_interactions_test.go:99`) resolves skill includes | `templates.NewIncludeResolver(dataRoot)` — real renderer, real embedded tree, no service mocks (content-only justification: this is a schema/prose contract, not a running envelope processor — F09 introduces no Go type for the control envelope) | n/a — content-only | A schema that named a `question_id` field on the `question` variant would violate D-005 (worker never mints identity); the renderer-level assertion that the field is absent catches an implementation that invents one |
| TC-003-04..07 (roster `capability_profile`/`model_tier`) | `internal/sharkdata` roster validator entrypoint at `embed.go:1169`'s `allowedMember` map, driven through the same `Validate...` path `internal/sharkdata/embed_test.go` already exercises for `model_tier` (`embed.go:1132-1133`) | The roster-YAML-parsing validator function itself — no repository or service seam exists here; roster validation is pure data-shape checking | Do not hand-construct a `RosterMember` struct bypassing YAML unmarshaling — the test must drive real YAML text through the real validator so a missing `allowedMember` entry is caught | An implementation that adds `capability_profile` to the struct but forgets the `allowedMember` map entry would reject every roster using the new field; a YAML-driven test (not struct-literal) is what catches that gap |
| TC-003-08..09 (provider references) | content-only — `skills/shark-attack/providers/{codex,claude-code}.md` (new), read via `sharkdata.ReadEmbedded`, same pattern as `readF07EmbeddedFile` (`e38_f07_interactions_test.go:26`) | `sharkdata.ReadEmbedded` — real embedded FS read | n/a — content-only | A provider reference that asserted a capability with no cited evidence line would pass a naive "file exists" check; the test greps for the evidence-citation marker adjacent to every claimed-supported operation |
| TC-003-11 (rule-inventory no-duplication, corrected per codex red-team BLOCKER) | content-only — all files under `skills/shark-attack/**` (restructured), read via `sharkdata.ReadEmbedded`, parsed against a maintained rule-inventory fixture (`{rule_id, canonical_file}`) | `sharkdata.ReadEmbedded` | n/a — content-only | A router that duplicated operating-model rules inline (violating "no rule stated twice") would still pass a naive single-sentence keyword-presence check; the inventory-driven near-duplicate scan across the *whole* tree is what a one-sentence spot check cannot catch |
| TC-003-12 (fresh-agent reachability) | content-only — `skills/shark-attack/SKILL.md`, read via `sharkdata.ReadEmbedded` | `sharkdata.ReadEmbedded` | n/a — content-only | A router missing a link for the Council scenario would strand a fresh agent with no path to `council.md`; the mechanical link-chain assertion per scenario class catches a missing or broken link the rule-inventory check wouldn't |
| TC-003-13 (host-adapter-contract field set) | content-only — `skills/shark-rider/context/host-adapter-contract.md`, read via a real repository file read (no embedded mirror), same convention as `readF07RepositoryFile` in `e38_f07_interactions_test.go` | real filesystem read — no service mocks; n/a for content-only | n/a — content-only | An adapter-contract file that renamed `prompt_sha256`/`prompt_bytes` or omitted the provenance fields TC-002 already proves at the `next.go` wire level would leave F10/F11 with an I-10 pointer describing a shape that doesn't match production; the field-name cross-check against TC-002 catches that drift |
| TC-004-01..12 (X-06 consumer activation) | `services.QuestionService.{ConfigureWorkflow,RecordResponse,Resolve}` and `internal/cli/commands` question/link/next/status commands, driven end-to-end against a real temporary SQLite DB via `db.InitDB` + CLI `Execute()`, following the exact pattern `TestTC004_X06ProducerPublicQuestionHandoffIsReadOnly` (`e39_interactions_test.go:898`) already uses for the producer side — this is the established `tests/contracts` package convention (black-box CLI+HTTP over a real temp DB), which is a distinct test category from `internal/cli/commands/*_test.go` unit tests and does not violate the mocked-CLI-test rule; see Test Infrastructure below | Real Shark CLI commands (`question create/configure-workflow/respond/resolve`, `link`, `next`, `status advance`) against a temp-file SQLite DB — this is the existing `tests/contracts` convention, not a new exception | Do not stub `QuestionService` — the whole point of X-06 consumer-activation coverage is proving the *actual* production wiring (`internal/services/question_blocker.go`, `status_group.go:545`) blocks and unblocks correctly | An implementation that links `question_blocks` before calling `configure-workflow` (reversing D-006's load-bearing order) would silently produce a non-blocking edge; this test's `draft`-state-link sub-case (TC-004-04) is the one that catches it |
| TC-005-01..06 (two-axis independence) | content-only — `skills/shark-attack/context/operating-model.md` (new): a scenario table asserted via `sharkdata.ReadEmbedded` plus, where the degrade-to-sequential rule is operationally testable, the same `resolveNext`/CLI seam as TC-004 for "no ownership evidence recorded" | `sharkdata.ReadEmbedded` for the scenario-table assertions; real CLI/DB seam (as TC-004) for the AC-002 degrade-to-sequential behavioral assertion, since that is the one sub-case with an actual runtime decision to verify | n/a for content rows; do not mock claim/ownership state for the AC-002 runtime sub-case — assert against the real absence of a recorded claim | A skill document that described "Batch implies parallel" would conflate the two axes; the scenario-table test asserts at least one `Batch`+non-parallel row exists verbatim |
| TC-006-01..04 (council routing threshold, I-04) | content-only — `skills/shark-attack/workflows/council.md` and `route-question.md`, read via `sharkdata.ReadEmbedded` | `sharkdata.ReadEmbedded` | n/a — content-only | A skill that routed every question (routine and material alike) through the council artifact path would create noise that buries real escalations; the routine-fixture sub-case (TC-006-01) asserts zero artifact creation for a scope-bounded question, which a "route everything to council" implementation would fail |
| TC-CI-01 (skill/contract-test atomicity, runner untouched — CI gate, not a numbered contract TC) | build/CI-level: `git diff --name-only <base>...<head>` scoped to `internal/runner/` (must be empty) plus `go test ./internal/runner/... ./tests/contracts/...` green at the same commit | n/a — this is a repository-state assertion, not a unit test with an injectable seam | Do not run this check against an intermediate commit — REQ-NF-006 requires the single-commit property, so the assertion must run against the final merged state | A PR that split the skill restructure from its replacement contract tests across two commits would show a transient red `e38_f04_interactions_test.go`/`e38_f07_interactions_test.go` at the intermediate commit; CI history (not just HEAD) is the evidence this counter-factual requires |
| TC-007-01 (`pull-by-role.md` retirement) | content-only — `skills/shark-attack/workflows/pull-by-role.md`, read via `sharkdata.ReadEmbedded`, same pattern as `e38_f07_interactions_test.go`'s `pullByRole` read | `sharkdata.ReadEmbedded` | n/a — content-only | A retirement that kept the phrase "the normal path" without the compatibility-reference framing would still describe a sanctioned claim route; the test asserts the specific reference-only framing, not just file existence |
| TC-007-04/05 (UAT round-1 corpus-wide claim-authorization sweep) | content-only — every `.md` file under the embedded `skills/shark-attack/` tree, read via `sharkdata.ReadEmbedded`, walked the same way TC-007-02's corpus scan already does | `sharkdata.ReadEmbedded` | n/a — content-only | TC-007-02 only scans files that cite `pull-by-role.md` by name; a live authorization added to a file that never mentions `pull-by-role.md` (as `SKILL.md` and `worker-ownership.md` both did) would pass TC-007-02 while still violating AC-018 — TC-007-04 closes that blind spot by scanning the whole corpus regardless of cross-reference, and TC-007-05 proves the fix didn't just delete the compatibility reference outright |
| TC-008-01/04 (parity gate, real trees) | `internal/sharkdata/shark_attack_parity_test.go` (new): `compareParity(authored, embedded fs.FS) []Drift` invoked with `os.DirFS("skills/shark-attack")` vs. an `fs.FS` adapter over `sharkdata.ReadEmbedded`'s backing embed.FS, mirroring the walk-and-compare shape research-report Findings #4 already performed manually (`diff -rq`) | the real authored/embedded FS pair — no service mocks apply, this is a filesystem/embed-FS parity check | Do not compare against a cached/snapshotted copy of either tree — both reads must be live at test time so a future edit to either tree is caught | An embedded-only file with no authored counterpart (a file added straight to `internal/sharkdata/default_data/skills/shark-attack/` without ever landing in `skills/shark-attack/`) would pass a naive "authored ⊆ embedded" check; the test walks the embedded tree too and fails on any embedded-only path |
| TC-008-02/03 (parity gate, comparator unit tests) | `compareParity(authored, embedded fs.FS) []Drift` invoked with two `testing/fstest.MapFS` fixtures — a compiled `go:embed` tree is immutable at test time, so fixture-injected drift/embedded-only cases must go through the pure-function seam, not the real embedded FS | `testing/fstest.MapFS` (standard library) | Do not skip the pure-function seam and try to mutate `sharkdata`'s embed.FS directly — it is compiled-in and cannot be written to | A comparator that only checked "every authored path exists in embedded" (one-directional) would miss an embedded-only file; the MapFS fixture with an embedded-only path is what catches a one-directional implementation |
| TC-009-01..02 (degraded-upstream behavior) | content-only + structural: `skills/shark-attack/**` prose asserting no `shark plan` read-only-probe instruction exists, and Go-level: `grep`-style source scan (or `go/ast` symbol check) confirming no import of `internal/models/council_artifact` (which does not exist) or the retired `parent_control.go` shape | `sharkdata.ReadEmbedded` for prose; direct `go list -deps` or import-scan for the Go-level absence assertion | n/a | An implementation that "helpfully" called `shark plan <root>` from the skill to double-check state before dispatch would silently mutate via `autoAdvanceCascadeParent` (`plan.go:631`) under F08's actual (unrepaired) trunk behavior; the prose-absence test catches the instruction, and a live-DB CLI assertion (reusing the TC-004 seam) can additionally prove `shark plan` is never invoked in F09's own dispatch path |
| TC-010-01..14 (resume lifecycle) | content-only for the skill-level fallback/ordering rules (`skills/shark-attack/workflows/resume.md`, `providers/{codex,claude-code}.md`), read via `sharkdata.ReadEmbedded`; where a concrete Go-level invariant exists (exactly-one-replacement-worker, capability-discovery-before-selection), assert it against the fixture harness described in the provider reference files rather than a live provider process — F09 ships no Go dispatcher changes (D-002), so "one worker created" is a documented-procedure invariant proven by prose/fixture-table assertion, not a running process count | `sharkdata.ReadEmbedded` | Do not spawn a real Codex/Claude Code process to prove worker-count invariants — that is F10/F11's conformance-fixture scope, not F09's; F09's own scope is the *contract*, proven at the documentation/fixture level | A resume procedure that silently started a second worker "just in case" while the first was still live would violate "exactly one" under the resume-supported branch; the fixture-table assertion checks the documented branch produces a single named action, not two |

---

## Acceptance Test Cases

### TC-001 — Parent-owned loop invariants during consultation

**Feature Requirement:** REQ-F-007; I-07 (parent-owned claim/heartbeat/transition boundary)
**Task Acceptance Criterion:** AC-010, AC-011
**Technique Applied:** State Transition (lease alive/lost) + BVA (heartbeat cadence vs. TTL)
**ISO 25010 Characteristic(s):** Functional Suitability, Reliability, Security

**Caller-Path Contract:** see table above (content-only for the prose invariant; runtime sub-case reuses TC-004's real-DB CLI seam for the claim/heartbeat state)

**Preconditions:** A dispatched entity is claimed by the parent; a `question_blocks` Question is `open`.

**Input / Expected Output (sub-cases):**
- TC-001-01: Heartbeat renewed mid-consultation → entity status and `task_history`/`feature_history` byte-unchanged (AC-010).
- TC-001-02: Simulated lease loss (claim `SessionID` no longer matches) mid-consultation → next attempt to record a responder answer via `question respond` is rejected (reuses `RecordResponse`'s `claim.SessionID == input.SessionID` check at `question_workflow_service.go:116`).
- TC-001-03: After lease loss, `status advance` on the dispatched entity is rejected and no council answer is delivered; recovery requires a fresh `shark claim` after a new keyed `shark next`.

**Observability Evidence:** none new — asserted via unchanged history rows (TC-001-01) and rejected mutation attempts (TC-001-02/03).

**Negative Cases:** A stale-session `question respond` call MUST be rejected, not silently accepted with the old session.

---

### TC-002 — Prompt provenance byte-exactness and worker identity

**Feature Requirement:** REQ-F-011, REQ-NF-003, REQ-NF-005, REQ-NF-007; I-10
**Task Acceptance Criterion:** AC-013, AC-024 (provenance-only slice), REQ-NF-007
**Technique Applied:** BVA + Attack-class enumeration (adversarial byte fixture)
**ISO 25010 Characteristic(s):** Functional Suitability, Security, Performance Efficiency (REQ-NF-007)

**Caller-Path Contract:** see table (`resolveNext` + `nextAdapterCache` mocks for the value path; cobra flag introspection for `--prompt-out` registration)

**Preconditions:** A mocked `nextAdapters` entry returns a fixed rendered prompt string.

**Input / Expected Output (sub-cases), fixture = a single adversarial prompt string containing:**
1. embedded newlines (`\n`, `\r\n`)
2. fenced code blocks with backticks (```` ``` ````)
3. single and double quotes
4. Unicode (emoji, combining characters, right-to-left marks)
5. shell metacharacters (`` ` ``, `$`, `;`, `|`, `&&`, `>`)
6. a trailing-newline-vs-no-trailing-newline pair (to isolate `--field prompt`'s documented one-newline difference)

- TC-002-01: `resp.PromptSHA256` equals `hex(sha256(resp.Prompt))` computed independently in the test.
- TC-002-02: `resp.PromptBytes` equals `len([]byte(resp.Prompt))`.
- TC-002-03: For each fixture variant, hashing is stable across two calls (determinism).
- TC-002-04: Flag-registration assertion only — `nextCmd.Flags().Lookup("prompt-out")` exists with the expected type/default, following `TestNextCommandDoesNotExposeRemovedPreviewFlag`'s pattern. This sub-case proves registration, **not** write behavior (write behavior moved to TC-002-09 below per codex red-team finding — flag introspection alone cannot prove file bytes are correct).
- TC-002-05: `--field prompt` output differs from the `--prompt-out` file by exactly one trailing newline (per AC-013's stated invariant) — this reuses the existing documented `--field` behavior, asserted rather than assumed.
- TC-002-06: Hash covers the **fully assembled** prompt (post-`assembleDispatchPrompt`, including the ownership preamble) — assert by varying only the ownership-preamble input and confirming the hash changes (proves D-004: hashing the pre-assembly instruction would NOT change here, so this sub-case is the one that would catch a regression to the rejected alternative).
- TC-002-07: Provenance record (whatever structure F09 persists — council handoff file, if any, per REQ-F-008) contains only prompt hash, byte length, entity key, provider worker/session identity, and timestamps; grep the written artifact for the literal fixture prompt text and assert absence (AC-024's provenance-only slice — the full leak-sink enumeration is TC-SEC-01 below).
- TC-002-08: `prompt_bytes` on the JSON wire response equals the existing `attribute.Int("prompt_bytes", ...)` OTel span attribute value (REQ-NF-007 "computed once" — same value reused, not recomputed) — reasoned code-review check that `assembleDispatchPrompt`'s return value is hashed exactly once per response, since spec explicitly permits "benchmark or reasoned review."
- TC-002-09 (**added per codex red-team BLOCKER**): a black-box, command-level test that invokes the real `runNext` path — `shark next <key> --prompt-out <tempfile>` executed against a temp SQLite DB seeded with a fixed entity/prompt fixture (reusing the `tests/contracts` real-DB convention, since this is the one TC-002 sub-case that must drive the actual `nextCmd.RunE`/`runNext`/flag-parsing/file-write chain, not the mocked `resolveNext` seam) — and asserts: (a) the file at `tempfile` contains exactly `resp.Prompt` bytes with no trailing newline, (b) `sha256(file) == resp.PromptSHA256` from the same invocation's `--json` output, (c) `len(file) == resp.PromptBytes`, (d) the CRLF fixture variant survives the write unmangled (catches a text-mode writer normalizing `\r\n`→`\n`), (e) an unwritable `--prompt-out` target (e.g. a directory path, or a path under a read-only directory) produces a clear error and non-zero exit, not a silent partial write.

**Negative Cases:** A fixture whose bytes are *not* rewritten (e.g., if `--prompt-out` accidentally goes through a text-mode writer that normalizes `\r\n`→`\n`) MUST cause the SHA-256 comparison to fail — the CRLF fixture variant (TC-002-09d) is what catches that class of bug. An unwritable target (TC-002-09e) MUST fail loudly, not silently skip the write while still reporting success.

---

### TC-SEC-01 — Prompt/credential leak-sink enumeration (supports AC-024, REQ-NF-003, REQ-NF-004)

**Added per codex red-team BLOCKER.** AC-024's "no rendered prompt in any
artifact" and REQ-NF-004's denylist reliance are open-ended robustness
assertions (Step 5's red-flag pattern) that TC-002-07/TC-010-14's original
single-artifact grep did not enumerate. This is F09-owned NFR coverage that
feeds evidence into the existing I-10 contract test pointers (TC-002,
TC-010) rather than a new spec-level `I-##`/`X-##` contract — spec.md's
Contract test index already cites TC-002 and TC-010 for REQ-NF-003/004/005,
so this test lands as additional sub-cases inside those two files, not as an
eleventh top-level TC number.

**Feature Requirement:** REQ-NF-003, REQ-NF-004; AC-024
**Technique Applied:** Attack-class enumeration (every durable/transport sink × every `ValidateQuestionBoundedText` denylist class)
**ISO 25010 Characteristic(s):** Security

**Caller-Path Contract:**
- Entrypoint: table-driven over (a) `internal/models.ValidateQuestionBoundedText(field, value, min, max)` directly (the real denylist function at `internal/models/question.go:190`) for the denylist-class dimension, and (b) each durable/transport sink for the sink dimension: `docs/council/` handoff/decision files written per F09 (REQ-F-008/009), Question `summary`/`evidence_pointer` fields via `question create`/`question respond`/`question resolve` CLI, `--prompt-out` file content, `--json`/`--field` CLI stdout, and existing OTel span attributes/events on `shark.next`.
- Lowest allowed mock seam: none for (a) — call the real validator directly; for (b), the same real-DB `tests/contracts` seam as TC-004, since sink enumeration requires real writes to inspect.
- Forbidden mocks: do not stub `ValidateQuestionBoundedText` — the denylist strings themselves (`api_key`, `password=`, `authorization:`, `bearer `, `system prompt`, `user prompt`, `assistant:`) including case and spacing variants (`API_KEY`, `Bearer  `, `Authorization :`) are the thing under test.
- Counter-factual: an implementation that validates `summary` but forgets `evidence_pointer` (or vice versa) would let a credential-shaped string through one field; the per-field table-driven sweep is what catches which specific field was missed.

**Sub-cases:**
- TC-SEC-01-01: For each denylist class × each of the 2 Question free-text fields (`summary`, `evidence_pointer` — reusing `ConfigureWorkflowInput`/`RecordQuestionResponseInput`/`ResolveQuestionInput`'s actual field set), a fixture containing the forbidden substring is rejected before persistence (assert no row/file is written on rejection).
- TC-SEC-01-02: Case and whitespace variants of each denylist string (`Bearer `, `BEARER:`, `password =`) are also rejected — proves the denylist isn't a naive exact-match.
- TC-SEC-01-03: After a full TC-004-style mint→respond→resolve lifecycle using a **non-denylisted** adversarial-shaped fixture (the TC-002 adversarial prompt fixture, truncated to fit field length limits), grep every sink in the enumerated list above (durable files under `docs/council/` if any were written, the `Q###` row's `context_data`, `--prompt-out` output, CLI stdout, and captured OTel span attributes from that run) for the literal fixture prompt text; assert zero matches in every sink, not just one.

---

---

### TC-003 — Control envelope, capability profile, provider references, router reachability

**Feature Requirement:** REQ-F-003, REQ-F-013, REQ-F-014, REQ-F-015; I-10
**Task Acceptance Criterion:** AC-003, AC-014, AC-015, AC-017
**Technique Applied:** Equivalence Partitioning (outcome vocabulary; roster field combinations) + Contract surface enumeration (provider ops)
**ISO 25010 Characteristic(s):** Functional Suitability, Compatibility, Usability, Maintainability, Portability

**Caller-Path Contract:** see table (content-only for envelope/provider/router; real YAML-driven roster validator for `capability_profile`/`model_tier`)

**Sub-cases:**
- TC-003-01: `worker-control-schema.yaml` round-trips a `kind: final, recommended_outcome: deep_verify` fixture unchanged through to `shark status advance --outcome deep_verify` — assert the outcome string passed to `status advance` equals the fixture's `recommended_outcome` verbatim, byte-for-byte, with no normalization/mapping table involved.
- TC-003-02: A `recommended_outcome` value **absent** from any control-vocabulary enum (e.g. `simple`, `standard`) still passes through — assert no enum-membership check exists in the envelope→advance path.
- TC-003-03: The `question` envelope variant schema carries no `question_id` field (D-005) — content assertion.
- TC-003-03b (**added per codex red-team CONCERN**, AC-006): grep every file under `skills/shark-attack/workflows/` and `skills/shark-rider/verbs/run.md` (the two directories that make up "any Rider path") for a `shark status set` invocation — assert **zero** occurrences, proving AC-006's "appears on no Rider path" claim across the whole executable-workflow surface, not just the one file TC-006's council-routing prose happens to cover.
- TC-003-04: Roster fixture `capability_profile: deep` (no `requirements`) validates cleanly.
- TC-003-05: Roster fixture with legacy `model_tier: opus` validates **and** produces a deprecation warning (assert the warning text/log line is emitted, not just that validation doesn't error).
- TC-003-06: Roster fixture with **both** `capability_profile` and `model_tier` set validates; `model_tier` produces no provider mapping (assert no code path reads `model_tier` to select a provider/model).
- TC-003-07: Roster fixture with **neither** field validates silently (no warning, no error).
- TC-003-08: `providers/codex.md` declares supported ops, unsupported ops, sequential fallback, and cites an installed-host evidence marker for each supported-op claim (grep for the evidence-citation pattern next to each claim).
- TC-003-08b (**added per codex red-team CONCERN**): `providers/codex.md` includes at least one operation with **no** captured evidence, and that operation is correctly listed under "unsupported" rather than silently omitted — proves the recorder doesn't just document what's easy to prove.
- TC-003-09: Same for `providers/claude-code.md` (including its own TC-003-09b no-evidence-marks-unsupported sub-case).
- TC-003-10: `SKILL.md` contains links to `context/operating-model.md`, `context/authority.md`, and each `workflows/{coordinate,direct,batch,council,route-question,execute-wave,resume}.md`; the operating-model rule text (e.g. the sequential-default sentence) appears in `operating-model.md` and does **not** also appear verbatim in `SKILL.md`.
- TC-003-11 (**redesigned per codex red-team BLOCKER** — the original TC-003-10 spot-checked one sentence, which cannot establish "no rule is duplicated" across the whole restructured tree): a **normative-rule inventory** — a table of `{rule_id, canonical source file, allowed cross-references}` covering every rule REQ-F-015's file table introduces (sequential-default, capability-discovery-before-selection, question-vs-council threshold, claim/heartbeat/authority boundary, parity requirement, etc.). The test parses every file under `skills/shark-attack/**` (post-restructure) and asserts: (a) each rule's canonical sentence/paragraph appears verbatim in exactly one file (its declared canonical source), (b) every other file that references the rule does so via a link/pointer, never a restated paraphrase matching >80% token overlap with the canonical text (a cheap near-duplicate heuristic, not full NLP), (c) `SKILL.md` itself contains zero rule statements from the inventory — only router prose and links.
- TC-003-12: Three fresh-agent-reachability scenario fixtures (Direct, Batch, Council) — for each, starting from only `SKILL.md`'s content, the test asserts a deterministic link-chain exists to the correct `workflows/*.md` file for that scenario class (mechanical link-following, not a semantic "would an agent understand this" judgment call).
- TC-003-13 (**added during task-decomposition rework** — closes T-017's I-10 orphan pointer gap): `skills/shark-rider/context/host-adapter-contract.md` declares the provider-neutral request/result field set and the prompt-hash provenance fields (`prompt_sha256`, `prompt_bytes`); read via a real repository file read (this path has no embedded mirror, unlike the rest of TC-003's `sharkdata.ReadEmbedded` rows). Cross-check: the declared field names match TC-002's `next.go` wire field names verbatim — no renamed or duplicate shape.

**Negative Cases:** A roster fixture with an invalid `capability_profile` value (e.g. `capability_profile: turbo`) MUST fail validation (equivalence class: invalid value outside `{fast,balanced,deep}`). A provider reference claiming a supported operation with zero cited evidence MUST be treated as a documentation defect (TC-003-08b's inverse) — if no unsupported-marked example exists in the shipped file, the test itself constructs one via a temp-file fixture copy to prove the *check* works, separate from asserting the shipped file's own content.

---

### TC-004 — X-06 consumer activation: mint → gate → route → respond → resolve → unblock

**Feature Requirement:** REQ-F-004, REQ-F-005, REQ-F-006, REQ-F-007; X-06
**Task Acceptance Criterion:** AC-004, AC-005, AC-006, AC-007, AC-008, AC-009
**Technique Applied:** State Transition + Contract surface enumeration + Decision Table (link-ordering)
**ISO 25010 Characteristic(s):** Functional Suitability, Security

**Caller-Path Contract:** see table — real temp-SQLite DB + CLI `Execute()`, mirroring `TestTC004_X06ProducerPublicQuestionHandoffIsReadOnly`

**Preconditions:** A seeded epic/feature (non-Question entity) exists, per the `e39_interactions_test.go#TC-004` seeding pattern.

**Sub-cases:**
- TC-004-01: `shark create question` → `Q001` created in `draft`.
- TC-004-02: `shark question configure-workflow Q001 --resolution-owner <owner> --responder <id>` → status becomes `open`.
- TC-004-03: `shark link Q001 <entity-key> --type=question_blocks` **after** configure-workflow → the edge qualifies (`QualifyQuestionBlock` returns non-nil); `shark next <entity-key> --json` returns `action: pause` with a populated `question_block` matching `{question_key, summary, resolution_owner, current_responder}` exactly (no `context_data`/`responses`/`evidence_pointer` leakage — reuse TC-004's forbidden-field list from `e39_interactions_test.go:926`).
- TC-004-04 (**the load-bearing ordering case**): Reverse the order — `shark link` the same edge type to a **second** Question that is still `draft` (never configured) → `shark next <that-entity>` returns **no** `question_block` (the edge is inert, per D-006's documented "silently inert" behavior) — this proves configure-before-link is enforced by the fixture, not asserted only in prose (AC-004's explicit requirement).
- TC-004-05: While `Q001` is `open`, `shark status advance <entity-key>` is rejected (question-blocked) — reuses `guardQuestionBlockedStatusAdvance` (`status_group.go:545`).
- TC-004-05b (**added per codex red-team CONCERN**): the same rejection holds while `Q001` is `answering` (a second responder still pending after the first responded) — the original sub-case only exercised the `open` state; `QualifyQuestionBlock` checks both `open` and `answering` (`question_blocker.go:86`), so this closes the gap between the checked states and the tested states.
- TC-004-06: `shark status set <entity-key> <next-status> --force` (or without `--force`, whichever the CLI requires) **succeeds** despite the open Question — asserting the documented bypass exists (AC-006) so the Rider-path prohibition can be enforced at the skill-prose level (TC-003/TC-005 content assertions) rather than at the CLI layer, which correctly does not block it.
- TC-004-07: `shark next Q001 --json` returns `action: spawn_agent` naming only the current pending responder (`entity_key: Q001`, no reference to the blocked entity's key in the payload) — reuses the TC-004 (E39) assertion pattern.
- TC-004-08: A second, competing `shark next Q001 --json` while a live claim exists on `Q001` collapses to `pause` (AC-007).
- TC-004-09: Parent (not the worker) calls `shark question respond Q001 --session <parent-sid> --responder <id> --summary ... --evidence-pointer ...` under the parent-held claim → succeeds, `Q001` transitions toward `ready_for_resolution` once all responders complete.
- TC-004-10: The **same** `question respond` call replayed with the identical `session-id`/`responder`/`summary`/`evidence-pointer` is idempotent (no error, no duplicate response row) — reuses `responseReplayMatches` (`question_workflow_service.go:156`).
- TC-004-11: A `question respond` call whose `--session` does **not** match the current claim's `SessionID` is rejected (AC-008's negative case).
- TC-004-12: `shark question resolve Q001 --owner <owner> --resolution-kind <kind> --resolution-pointer <ptr>` → `Q001` becomes `resolved`; the `question_blocks` predicate no longer qualifies; `shark status advance <entity-key>` now succeeds (AC-009).
- TC-004-13 (**added per codex red-team BLOCKER** — "zero bespoke records" was previously asserted only by not writing one in the test's own code, which proves nothing about the production schema): a **repository-state snapshot** test. Before the TC-004-01..12 lifecycle runs, enumerate `sqlite_master` table/index/trigger names in the temp DB (same `databaseSnapshotTC011`-style helper `e39_interactions_test.go` already uses at `databaseSnapshotTC011`/`before`/`after` comparisons). After the full mint→configure→link→route→respond→resolve→unblock lifecycle, re-enumerate: assert the **set of table names is identical** (no new table was created — a bespoke question/handoff/resolution store would require at least one new table, since F09 spec.md declares zero migrations and `CurrentSchemaVersion` stays at 32) and that every row written during the lifecycle lives in tables already owned by E39 (`questions`, `question_responses`-equivalent JSON in `context_data`, `task_history`/`question_history` if such a table exists) — assert zero rows in any table outside that named allowlist changed.
- TC-004-14 (**added per codex red-team BLOCKER**, REQ-NF-001 static guard): a source-level guard — `go list -deps ./internal/...` (or an equivalent import-graph check) run in CI asserts no package under `internal/` imports a name matching `*council_artifact*`, `*question_control*`, `*parent_control*`, `*roster_profile*` (the retired abandoned-branch type names research-report Findings #5 names) — catching an accidental re-introduction of the rejected bespoke `QuestionControl` design at compile-dependency granularity, not just by code review.

**Negative Cases:** `question respond` before `configure-workflow` (Question still `draft`) MUST be rejected ("Question workflow is not configured"). TC-004-13's table-name diff MUST be empty — any new table name appearing is treated as a hard failure, not a warning, since REQ-NF-001 forbids "a second workflow engine, or second lifecycle store" absolutely.

---

### TC-005 — Two-axis independence (coordination level × execution topology)

**Feature Requirement:** REQ-F-001, REQ-F-002; I-10
**Task Acceptance Criterion:** AC-001, AC-002
**Technique Applied:** Decision Table
**ISO 25010 Characteristic(s):** Functional Suitability, Reliability

**Caller-Path Contract:** see table (content scenario table + one runtime degrade-to-sequential sub-case)

**Sub-cases (scenario table asserted against `operating-model.md`):**
- TC-005-01: `Direct` + `Sequential` scenario row exists.
- TC-005-02: `Batch` + parallel-research-then-sequential-writes scenario row exists.
- TC-005-03: `Council` + mixed-topology scenario row exists.
- TC-005-04: Changing only the coordination-level column of a fixture row does not change the topology column value (independence assertion — parse the scenario table and assert no row implies coordination level determines topology).
- TC-005-05: A fixture row requesting a parallel topology with **no** recorded ownership/isolation evidence resolves to `Sequential` (AC-002) — this is the one sub-case with a live decision to make; assert against the documented degradation rule, and where a concrete Shark-side signal exists (no claim/ownership note recorded), reuse the TC-004 real-DB seam to show no parallel dispatch occurs.
- TC-005-06: Isolation evidence present does **not** by itself make logically dependent (producer/consumer contract-ordered) work run in parallel (D-007's explicit non-goal) — content assertion.

**Full decision table (added per codex red-team BLOCKER — the six sub-cases
above varied coordination level but never varied topology while holding
coordination constant, and omitted the ownership-only/isolation-only/both
evidence rows):**

| # | Requested coordination | Requested topology | Ownership evidence recorded? | Isolation evidence recorded? | Expected resolved topology | Sub-case |
|---|---|---|---|---|---|---|
| 1 | Direct | Sequential | n/a | n/a | Sequential | TC-005-01 |
| 2 | Batch | Parallel-with-ownership | yes | no | Parallel-with-ownership | TC-005-07 |
| 3 | Batch | Parallel-with-ownership | **no** | no | Sequential (degrade) | TC-005-05 |
| 4 | Batch | Parallel-with-isolation | no | yes | Parallel-with-isolation | TC-005-08 |
| 5 | Batch | Parallel-with-isolation | no | **no** | Sequential (degrade) | TC-005-09 |
| 6 | Batch | Parallel-with-isolation | yes | yes | Parallel-with-isolation (isolation still required; ownership alone is insufficient for the isolation-requesting row) | TC-005-10 |
| 7 | Council | Sequential | n/a | n/a | Sequential | TC-005-03 |
| 8 | Council | Parallel-with-ownership | yes | no | Parallel-with-ownership | TC-005-11 |
| 9 | Direct | Parallel-with-ownership | yes | no | Parallel-with-ownership (coordination level `Direct` does not itself forbid a parallel topology — proves the axes are independent in *both* directions, not just Batch/Council→parallel) | TC-005-12 |

- TC-005-07..12: assert each table row's resolved topology against the
  documented degradation rule in `operating-model.md`; rows 3 and 5 are the
  two independent degrade-to-`Sequential` paths (missing ownership evidence
  vs. missing isolation evidence) — a prior draft of this plan only tested
  the ownership-evidence-missing path (row 3 / original TC-005-05) and would
  have missed a regression that broke isolation-evidence detection alone
  (row 5).
- TC-005-13: Row 9 (`Direct` + a parallel topology request) additionally
  proves coordination level and topology vary independently *both* ways —
  the original sub-case set only showed `Batch`/`Council` could pair with a
  non-parallel topology, not that `Direct` could pair with parallel.

**Executable-procedure sub-cases (added during task-decomposition rework, T-023 —
`coordinate.md`/`direct.md`/`batch.md`/`execute-wave.md` implement the
scenario table above; TC-005-01..13 assert the rules exist in
`operating-model.md`, TC-005-14..16 assert the four procedures apply them):**

- TC-005-14: `coordinate.md` is the two-axis entry point and routes
  deterministically to `direct.md` (Direct), `batch.md` (Batch), and
  `council.md` (Council) per the selected coordination level — content
  assertion against `coordinate.md`'s routing table/links.
- TC-005-15: `direct.md` documents single-worker dispatch (no topology
  selection — Direct implies one worker); `batch.md`/`execute-wave.md`
  document the parallel-with-ownership and parallel-with-isolation wave
  shapes and link to, rather than restate, `operating-model.md`'s
  degradation rule.
- TC-005-16 (D-007 non-parallelization assertion): `batch.md` and
  `execute-wave.md` positively state that isolation evidence alone never
  authorizes running logically dependent (producer/consumer
  contract-ordered) work in parallel — grep both files for an explicit
  ordering-preserved statement and assert its presence; absence is a FAIL.
- TC-005-17 (regression guard, added during T-008 rework —
  code-review-2026-08-03T0801-E38-F09.md finding #1/blocker): row 9's
  "independent in both directions" claim is classification-time only.
  `operating-model.md` must never again assert that a `Direct`-classified
  request executes an actual parallel dispatch "exactly like"
  `Batch`/`Council` — that specific phrasing contradicted `coordinate.md`'s
  routing (topology never changes which procedure file is selected) and
  `direct.md`'s own "performs no topology-selection step of its own"
  statement. Assert the forbidden phrase is absent, that
  `operating-model.md` explicitly connects row 9 to `direct.md`'s
  classification-only behavior, and that row 9's resolved-topology cell
  itself says "classification only".

---

### TC-006 — Council routing threshold (I-04); routine answers create no artifact, material questions create one immutable record

**Feature Requirement:** I-04 (consumed); epic §4.5 council communication contract
**Task Acceptance Criterion:** supports AC-004's "no bespoke question/handoff store" boundary by proving the *other* side of the boundary — material questions still use I-04, not a diluted `Q###`
**Technique Applied:** Decision Table (material vs. routine threshold) + Contract surface enumeration
**ISO 25010 Characteristic(s):** Functional Suitability, Maintainability

**Caller-Path Contract:** see table — content-only, `sharkdata.ReadEmbedded` over `council.md`/`route-question.md`

**Sub-cases:**
- TC-006-01: A fixture question classified "routine" (scope-bounded, single-role, no architecture/quality/product impact) is documented as routing to the E39 `Q###` path and creating **no** `docs/council/` artifact.
- TC-006-02: A fixture question classified "material" (crosses a named threshold: scope, architecture, or quality-gate impact) is documented as routing through `council.md` **only** — `escalate.md` is retired by T-024 and must not appear in this assertion — and creating exactly **one** immutable decision/handoff record (I-04 shape), not a `Q###`.
- TC-006-03: The threshold rule itself (what makes a question "material") is stated once, in one file, and referenced — not duplicated — from the other (`route-question.md` points at `council.md`'s definition rather than restating it).
- TC-006-04: `route-question.md` and `council.md` agree on which vocabulary/category values (`product|requirements|architecture|quality|process` per REQ-F-003) map to which path — no category is orphaned (routable to neither) or ambiguous (routable to both without a tie-break rule).

---

### TC-CI-01 — Atomic skill-restructure landing; `internal/runner/` untouched (CI gate)

Not a numbered contract TC — spec.md's Contract test index allocates exactly
ten TC numbers (TC-001…TC-010), one per I-##/X-## row, and REQ-NF-002 /
REQ-NF-006's atomicity requirement is a **repository-state** gate rather than
a per-interaction contract test. It is verified alongside TC-006 in the same
CI run, not folded into TC-006's own assertions (council routing and diff
atomicity are unrelated concerns).

**Feature Requirement:** REQ-NF-002, REQ-NF-006
**Task Acceptance Criterion:** AC-019, AC-020
**Technique Applied:** Equivalence Partitioning (diff scope) + State Transition (no intermediate red)
**ISO 25010 Characteristic(s):** Functional Suitability, Compatibility

**Caller-Path Contract:** see table — build/CI-level, not a unit test

**Sub-cases:**
- TC-CI-01a: `go test ./internal/runner/...` passes, unmodified, at the feature's landing commit.
- TC-CI-01b: `git diff --name-only <pre-F09>...<post-F09>` touches zero files under `internal/runner/`.
- TC-CI-01c: At the landing commit, `tests/contracts/e38_f04_interactions_test.go` and `tests/contracts/e38_f07_interactions_test.go` both pass (REQ-NF-006) — CI history must show no red state at any intermediate commit within the F09 PR (verify via CI run list, not just final green).

---

### TC-007 — `pull-by-role.md` retirement from the sanctioned normal path

**Feature Requirement:** REQ-F-017; I-06
**Task Acceptance Criterion:** AC-018
**Technique Applied:** Attack-class enumeration (residual sanctioned-claim phrasing)
**ISO 25010 Characteristic(s):** Functional Suitability, Maintainability

**Caller-Path Contract:** see table — content-only, `sharkdata.ReadEmbedded`

**Sub-cases:**
- TC-007-01: `pull-by-role.md` (post-restructure) frames itself as a compatibility reference (not a sanctioned normal-path claim route) — reuse and extend the existing F07-owned assertions (`"Do not hand this child session to /shark-rider run."`, `"worker-owned child mode"`, `"not /shark-rider run"`) and additionally assert the file's introductory framing explicitly labels itself historical/compatibility-only.
- TC-007-02: No other file in the rendered `skills/shark-attack/` corpus references `pull-by-role.md` as the normal claim path (grep every workflow file for a still-sanctioning cross-reference).
- TC-007-03 (**added per codex red-team CONCERN**): an enumerated forbidden-vocabulary list, **derived from — not authored independently of — T-009's phrase-by-phrase adjudication of `internal/sharkdata/shark_attack_pull_test.go`'s 16 pinned phrases** (the subset T-009 marks "sanctioned claim route → retire", not the "authority description → keep" subset) — none of these phrases, nor any close paraphrase, appears anywhere in the post-restructure `pull-by-role.md` outside a clearly marked "historical reference" or "compatibility note" block. This replaces reliance on the two specific phrases the original sub-case checked with a maintained, explicit list that cannot silently diverge from the pinning test's own vocabulary.
- TC-007-04 (**added per UAT round-1 finding UAT-001, HIGH**): TC-007-02 only scans files that reference `pull-by-role.md` by name, so it missed a live worker-owned-child claim/heartbeat/release authorization that never cites `pull-by-role.md` at all — `SKILL.md`'s router pointed a reader at worker-owned child mode as something to "use ... when," and `context/worker-ownership.md` authorized a worker to "Claim, heartbeat and release" its own child lease. TC-007-04 sweeps the **complete** rendered `skills/shark-attack/` corpus for that phrase set, honoring the same clearly marked historical/compatibility section boundary `pull-by-role.md` already established (reused, not reinvented, so the two markers cannot diverge) — content before the marker, or the whole file when no marker exists, must not contain either forbidden phrase.
- TC-007-05: positive-control companion to TC-007-04 — the retired direct-claim phrase must still exist inside `worker-ownership.md`'s own historical/compatibility section (reference retained, not silently deleted), matching the retirement-not-deletion discipline TC-007-03 already proves for `pull-by-role.md`.

---

### TC-008 — Authored/embedded parity gate

**Feature Requirement:** REQ-F-016; X-05
**Task Acceptance Criterion:** AC-016
**Technique Applied:** Attack-class enumeration (injected drift; embedded-only file)
**ISO 25010 Characteristic(s):** Functional Suitability, Maintainability

**Caller-Path Contract (redesigned per codex red-team CONCERN — a compiled
`go:embed` tree is immutable at test time, so drift/embedded-only fixtures
cannot be injected directly into `sharkdata.ReadEmbedded`'s backing FS):**
the comparator itself must be implemented as a **pure function over two
`fs.FS` parameters** — `compareParity(authored, embedded fs.FS) []Drift` —
so fixture sub-cases can inject synthetic trees via `testing/fstest.MapFS`
(standard library, no new dependency) while the real gate (TC-008-01/04)
invokes the same function with `os.DirFS("skills/shark-attack")` and an
`fs.FS` adapter over `sharkdata.ReadEmbedded`'s backing embed.FS.

**Sub-cases:**
- TC-008-01: Baseline (post-sync) run, using the **real** authored tree (`os.DirFS`) vs. the **real** embedded tree (embed.FS adapter) — zero drift, zero embedded-only files. This is the actual CI gate; TC-008-02/03 below are `fstest.MapFS`-fixture unit tests of the comparator function itself, not of the real trees.
- TC-008-02: `compareParity` given two `fstest.MapFS` fixtures differing by one byte in one file → returns exactly one drift entry naming that path.
- TC-008-03: `compareParity` given an `embedded fstest.MapFS` fixture containing one path absent from the `authored fstest.MapFS` fixture → returns an "unexpected embedded-only file" entry naming that path (REQ-F-016's explicit second failure mode).
- TC-008-04: `skills/shark-rider/` (authored-only, no embedded counterpart per spec's explicit note) is correctly **excluded** from the parity gate's scope — assert the real-tree invocation (TC-008-01) does not false-positive on it, by asserting `compareParity`'s scope root is `skills/shark-attack/` only, not the full `skills/` tree.

---

### TC-009 — Degraded-upstream behavior (F08/F05 gap independence)

**Feature Requirement:** REQ-F-018; I-08, I-09
**Task Acceptance Criterion:** structural (no dedicated AC number; verified via REQ-F-018's own traceability plus AC-024's provenance-only assertion)
**Technique Applied:** Attack-class enumeration (accidental dependency on unmerged F08/F05 code)
**ISO 25010 Characteristic(s):** Functional Suitability, Reliability

**Caller-Path Contract:** see table — content-only prose scan + import/symbol-absence check

**Sub-cases:**
- TC-009-01: No file under `skills/shark-attack/**` (post-restructure) instructs a chair/parent to call `shark plan <key>` as a read-only inspection step (the documented F08 gap: `plan.go:631` still mutates via `autoAdvanceCascadeParent`).
- TC-009-02: No new F09 Go file imports `internal/models/council_artifact` or any package under a path that does not exist on `main` today (compile-time proof by construction — if the package doesn't exist, an import would fail `go build`, but this sub-case also greps for `admin council` CLI invocations in skill prose to catch a documentation-only reference that would compile fine but describe a nonexistent command).
- TC-009-03: F09's own durable decision/handoff artifacts land under `docs/council/` as plain files (not through a `shark admin council` subcommand) — assert no skill file instructs the parent to call that subcommand.

---

### TC-010 — Resume lifecycle: same-worker follow-up, bounded replacement, interrupt, isolation, capability-discovery ordering

**Feature Requirement:** REQ-F-008, REQ-F-009, REQ-F-010, REQ-F-012; I-10
**Task Acceptance Criterion:** AC-011, AC-012, AC-021, AC-022, AC-023, AC-024 (resume-half slice)
**Technique Applied:** Decision Table (resume-supported × unsupported; 3 independent capability flags; interrupt supported × unsupported) + Attack-class enumeration (prompt leakage into handoff)
**ISO 25010 Characteristic(s):** Functional Suitability, Reliability, Security

**Caller-Path Contract:** see table — content/fixture-level, `sharkdata.ReadEmbedded` plus documented fixture-table assertions (F09 ships no dispatcher code per D-002)

**Sub-cases:**
- TC-010-01: Silent responder → parent pings once (documented step, asserted in `route-question.md`).
- TC-010-02: No response after ping → parent interrupts/cancels where supported, else waits to deadline only.
- TC-010-03: One replacement responder is routed (not more than one).
- TC-010-04: No response before the consultation deadline → parent stops write workers, records a bounded unresolved handoff, records a blocker, releases the lease (assert all four actions are documented as an ordered, non-optional sequence — not "some of the above").
- TC-010-05: Fixture "host declares resume supported" → documented procedure delivers the answer to the same worker identity and creates **zero** new workers.
- TC-010-06: Fixture "host declares resume unsupported" → documented procedure creates **exactly one** replacement worker from a bounded immutable handoff.
- TC-010-07: The handoff (either path) carries entity key, question, answer, and evidence pointers, and explicitly **excludes** rendered-prompt content — grep the documented handoff schema for a prompt/transcript field and assert absence.
- TC-010-08: Fixture "isolation undetected" → resolves to `Sequential` **without** the procedure describing an isolation-detection command attempt beyond the capability-discovery step itself (REQ-F-012 ordering).
- TC-010-09: Fixture "follow-up undetected" → forces replacement path (not same-worker).
- TC-010-10: Fixture "interrupt undetected" → forces deadline-only expiry (no cancel attempt described).
- TC-010-11: Capability discovery is documented as the **first** step in the provider-reference procedure, strictly before any topology/coordination selection step (assert ordering in the rendered workflow file, e.g. section-index comparison).
- TC-010-12: Fixture "interrupt supported" → stale consultation is cancelled **before** the replacement responder is routed (ordering assertion); cancelling is documented as changing no Shark state (no claim/status side effect described).
- TC-010-13: Fixture "interrupt unsupported" → the documented fallback runs and no unverified provider command is invoked (cross-check against the provider reference's declared-unsupported-ops list from TC-003-08/09 — the fallback must not silently invoke an op the provider reference itself marked unsupported).
- TC-010-14: Across all documented handoff/decision/note templates in `skills/shark-attack/**`, none contains a `{{prompt}}`/`{{rendered_prompt}}`-style placeholder or field name suggesting prompt persistence (AC-024's resume-half).

**Capability-vector decision table (added per codex red-team BLOCKER — the
original TC-010-08..10 tested each capability missing in isolation; a real
host can have any combination of the three independent capabilities present
or absent, and the original set never tested combinations):**

| # | Isolation detected? | Follow-up detected? | Interrupt detected? | Expected resolved behavior | Sub-case |
|---|---|---|---|---|---|
| 1 | yes | yes | yes | Parallel-with-isolation eligible; same-worker follow-up on question; interrupt-then-replace on silent responder | TC-010-15 (baseline all-supported) |
| 2 | no | yes | yes | Sequential (isolation missing forces the topology degrade from TC-005); follow-up/interrupt behavior for the question loop is unaffected by the isolation gap | TC-010-08 |
| 3 | yes | no | yes | Topology unaffected; question loop forces bounded-replacement path (not same-worker) since follow-up is absent | TC-010-09 |
| 4 | yes | yes | no | Topology and follow-up unaffected; silent-responder handling forces deadline-only expiry, no cancel attempt | TC-010-10 |
| 5 | no | no | no | Sequential topology; bounded-replacement on question; deadline-only expiry on silent responder — all three fallbacks apply simultaneously, none masks another | TC-010-16 |

- TC-010-15/16: assert the three capability-driven behaviors (topology
  resolution, follow-up-vs-replacement, interrupt-vs-deadline) are decided
  **independently per capability flag** — row 5 (all undetected) is the
  case that would catch an implementation that only checks one flag and
  assumes the others follow from it.

**Responder-outcome ladder (added per codex red-team BLOCKER — the original
TC-010-01..04 covered only "silent throughout"; the full ladder has more
states):**

| # | Responder behavior | Expected result | Sub-case |
|---|---|---|---|
| 1 | Answers before any ping | No ping sent; answer recorded normally (fast path — not previously covered) | TC-010-17 |
| 2 | Silent, then answers after the one ping | No replacement routed; answer recorded from the original responder | TC-010-18 |
| 3 | Silent through ping, replacement responder answers | Replacement's answer recorded; original responder's pending state is closed out, not left dangling | TC-010-03/TC-010-19 |
| 4 | Silent through ping AND replacement, deadline reached | Bounded unresolved handoff, blocker recorded, lease released (TC-010-04) | TC-010-04 |
| 5 | Answers at exactly the deadline boundary (BVA edge) | Documented as either "counts" or "too late" — whichever the procedure states — and the fixture asserts that stated rule, not an assumed one; if the procedure is silent on the boundary, this sub-case is a FAIL and the gap is reported as a task-spec ambiguity, not silently resolved by the test | TC-010-20 |

---

## Integration Scenarios

| Scenario | Components | Boundary verified | Epic UAT tie-in |
|---|---|---|---|
| Keyed dispatch → prompt provenance → Rider dispatch | `internal/cli/commands/next.go` → `skills/shark-rider/verbs/run.md` | `resp.Prompt`/`resp.PromptSHA256`/`resp.PromptBytes` reach the harness unchanged; host adapter transports `response.prompt` verbatim per the existing UAT-03 contract | Extends UAT-03 (ordinary dispatch ownership preserved) with provenance |
| Worker question → parent-minted `Q###` → serial responder dispatch → parent-transcribed answer → resolve → unblock | `skills/shark-attack/workflows/route-question.md` → `internal/services/question_*` → `internal/cli/commands/{question,next,status_group}.go` | Parent never lets a worker call a Shark mutation command directly (ADR-005); the `question_blocks` gate is structural, not prose-only | New coverage — no prior E38 UAT scenario exercised the live-question loop end to end (the epic uat-plan.md predates E39) |
| Council escalation for material questions vs. routine E39 Question routing | `skills/shark-attack/workflows/council.md` (I-04) vs. `route-question.md` (X-06) | The threshold between "material → council artifact" and "routine → `Q###`" is documented and does not create two competing question stores | Extends UAT-04 (council memory survives worker refresh) — F09 must not regress it while adding the new routine path |
| Capability discovery → topology/coordination selection → provider dispatch | `skills/shark-attack/providers/{codex,claude-code}.md` → `skills/shark-attack/context/operating-model.md` | Capability discovery precedes selection; missing capability data causes documented fallback, never an invented command (REQ-F-012) | New — F10/F11 depend on this ordering being provably correct before they add more providers |
| Authored skill tree ↔ embedded bundle ↔ project override | `skills/shark-attack/**` ↔ `internal/sharkdata/default_data/skills/shark-attack/**` ↔ `shark-data/overrides/skills/shark-attack/**` | Parity gate (new) + existing replace-only override behavior (`TestTC004_X05EmbeddedSkillOverrideIsReplaceOnly`) both hold simultaneously | Extends UAT-08/UAT-09 (X-05) |
| Roster `capability_profile`/`model_tier` → selection/claim eligibility | `skills/shark-attack/context/roster-schema.yaml` → `internal/sharkdata/embed.go` validator → F06's claim/selection path | Neither field grants authority; F06's `ClaimService` race boundary and selection logic are untouched (I-06) | Extends UAT-01/UAT-02 |

---

## Test Infrastructure

### Existing patterns reused (no new helpers required beyond what's listed)

- **`tests/contracts` package convention** — black-box CLI(+HTTP) execution against a real temporary SQLite database via `db.InitDB(dbPath)` inside `t.TempDir()`, with a `runShark*` exec helper (pattern: `runSharkTC013` / `runSharkTC011Failure` in `e39_interactions_test.go`). This is the established convention for **cross-feature contract tests** in this codebase and is distinct from the `internal/cli/commands/*_test.go` mocked-service unit-test rule (`.claude/rules/testing/cli-tests.md`) — the golden rule ("ONLY repository tests use the real database") governs unit-test packages; `tests/contracts` is a deliberate black-box integration layer that predates F09 and is reused, not introduced, here. TC-004, TC-005-05, and TC-009's Go-level sub-case reuse this pattern. A new `runSharkF09*`-named helper (or reuse of the existing `e39_interactions_test.go` helpers if package-visible) is the only new test code required for this category.
- **`sharkdata.ReadEmbedded` + content-string assertions** — the pattern every existing `e38_f04`/`e38_f07_interactions_test.go` test uses for skill-prose contracts. All content-only TC sub-cases (TC-003, TC-005 scenario rows, TC-007, TC-009 prose, TC-010) reuse this directly; no new infrastructure needed.
- **`nextAdapterCache` mock shape** (`next_test.go:330`, `TestResolveNext_ReturnsSelfContainedPrompt`) — reused verbatim for TC-002's `resolveNext`-level assertions. This keeps `next_provenance_test.go` (the file spec.md's architecture table names) inside the golden-rule-compliant `internal/cli/commands` unit-test convention (mocked adapters, no real DB), as opposed to `tests/contracts`'s real-DB convention — the two files in spec's table (`next_provenance_test.go` vs. `e38_f09_interactions_test.go`) intentionally sit in different test categories for this reason, and this plan preserves that split.
- **`templates.NewIncludeResolver`** (`e38_f04_interactions_test.go:99`) — reused for TC-003-01/03's rendered-schema assertions if the control-envelope schema participates in `{{include:}}` resolution; otherwise plain `sharkdata.ReadEmbedded` suffices since the schema is declarative YAML, not a rendered template.

### New test helpers needed

- A `filepath.WalkDir`-based two-tree comparator in `internal/sharkdata/shark_attack_parity_test.go` (TC-008) — no prior art in this codebase compares authored vs. embedded byte-for-byte across an entire subtree (existing tests spot-check individual files); this is new, bounded utility code, not a new pattern.
- An adversarial prompt-fixture constant (TC-002) — a single `const adversarialPromptFixture = "..."` string literal covering the six adversarial classes listed in TC-002; no helper function needed beyond the constant itself.

---

## Cross-feature contract tests (I-##)

| I-## | Producer | Consumer(s) | Shape source | Contract test pointer (declared in spec.md) | TC in this plan |
|---|---|---|---|---|---|
| I-10 (produces) | E38-F09 | E38-F10, E38-F11 | [Provider-neutral adapter contract](../../../../dev-artifacts/shark-attack-v2-plan/implementation-plan.md#3-provider-neutral-adapter-contract) | `tests/contracts/e38_f09_interactions_test.go#TC-002`, `#TC-003`, `#TC-005`, `#TC-010` | TC-002, TC-003, TC-005, TC-010 (this plan's TC numbers match spec.md's contract-test-index numbers exactly — no twin naming) |
| I-04 (consumes) | E38-F04 | E38-F05, E38-F09 | E38 architecture §4.5 | `tests/contracts/e38_f09_interactions_test.go#TC-006` | TC-006 (council routing threshold — matches spec.md's index exactly) |
| I-06 (consumes) | E38-F06 | E38-F05, E38-F09 | F06 requirements + v2 authority model | `tests/contracts/e38_f09_interactions_test.go#TC-007` | TC-007 |
| I-07 (consumes) | E38-F07 | E38-F05, E38-F09 | Live question-and-resume protocol | `tests/contracts/e38_f09_interactions_test.go#TC-001` | TC-001 |
| I-08 (consumes) | E38-F08 | E38-F05, E38-F09, E38-F11 | Implementation plan §5 Tranche A | `tests/contracts/e38_f09_interactions_test.go#TC-009` | TC-009 |
| I-09 (consumes) | E38-F05 | E38-F09, E38-F11 | Implementation plan §5 Tranche B | `tests/contracts/e38_f09_interactions_test.go#TC-009` | TC-009 (shared pointer, per spec.md — F09 does not write a twin for I-09) |

This plan's Acceptance Test Cases section numbers its ten contract test
sections **TC-001 through TC-010, in the same order and covering the same
interaction as spec.md's own "Contract test index" table** — e.g. TC-001 is
the parent-owned-loop/I-07 test, TC-006 is the council-routing-threshold/I-04
test. REQ-NF-002/REQ-NF-006's atomicity requirement (AC-019/AC-020,
`internal/runner/` untouched, no intermediate red commit) is a
**repository-state CI gate**, not a per-interaction contract test — spec.md's
index allocates exactly ten TC numbers, one per I-##/X-## row, with no
eleventh slot for a cross-cutting repo-diff check. It is verified as
**TC-CI-01** (see the Acceptance Test Cases section) alongside, but separate
from, TC-006.

No twin tests: this plan's TC-004 (X-06) is new, distinct coverage from
`e39_interactions_test.go#TC-004` per spec.md's explicit "F09 adds distinct
consumer activation coverage... does not create a twin" note — verified
above in Drift Findings #4.

---

## Cross-epic integration tests (X-##)

| X-## | Producer | Consumer | Shape/contract source | Test coverage pointer (spec.md) | TC in this plan |
|---|---|---|---|---|---|
| X-06 | E39-F04 | E38-F09 (activation owner) | E39 architecture §2–§4; E38-F09 feature.md | `tests/contracts/e38_f09_interactions_test.go#TC-004`, citing `e39_interactions_test.go#TC-004` (`TestTC004_X06ProducerPublicQuestionHandoffIsReadOnly`) verbatim as the producer-shape proof | TC-004 (this plan) is the distinct consumer-activation coverage; producer-shape proof is cited, not re-tested |
| X-05 | E32 (owning feature E38-F04, completed) | E38-F09 | E38 architecture §2 ADR-007 and §5 Phase 4; E32 embedded bundle contract | `tests/contracts/e38_f09_interactions_test.go#TC-008`, alongside existing `e38_f04_interactions_test.go#TC-004` (`TestTC004_X05EmbeddedSkillOverrideIsReplaceOnly`), left intact | TC-008 (this plan) is new parity-gate coverage; the existing override test is unmodified regression coverage, re-run but not re-authored |

Both rows are `assigned` status in the global product map
(`docs/product/cross-epic-integration-map.md`) and `E38-cross-epic-map.md`.
No deferral applies to either — both have concrete test pointers above, no
`docs/product/progress.md` deferral entry needed.

**Owner-decision note (not a test gap, but must not be silently resolved by
this plan):** spec.md's Cross-epic §"Consumes/Validates" row for X-05 records
an unresolved tension — `GATE-F09-003` ("client-skill-only; remove embedded
bundle") vs. X-05's map-assigned requirement that F09 not remove the embedded
bundle. spec.md already resolves this correctly for *implementation*
purposes (ships both trees + parity gate, does not remove the bundle). This
test plan does not test for bundle removal (it would be testing a decision
spec.md explicitly declined to make) — TC-008 only proves parity, which holds
under either eventual owner decision.

---

## Codex Test-Plan Red-Team

**Verdict:** FAIL (initial pass) → issues incorporated below; plan now meets the exit gate.
**Issues raised:** 8 (5 BLOCKER, 3 CONCERN)
**Issues addressed before dev:** 7
**Issues deferred:** 1, with rationale (see below)

Full transcript: `dev-artifacts/2026-08-01-e38-f09-test-plan/analysis/codex-red-team-output.txt`
(prompt: `dev-artifacts/2026-08-01-e38-f09-test-plan/analysis/codex-red-team-prompt.md`).
Run via `codex exec -s read-only` against this repo, model `gpt-5.6-terra`,
reasoning effort medium, ~73k tokens.

Codex's summary line: *"the plan has strong traceability, and TC-004 is
genuinely new X-06 consumer coverage, but it leaves several
security/decision-table/runtime contracts untestable or only prose-tested."*

| # | Severity | Finding | Affected | Response |
|---|---|---|---|---|
| 1 | BLOCKER | Unbounded secret/prompt-leak assertions lack an attack-surface sink inventory; REQ-NF-004 had no denylist test | AC-024, REQ-NF-003/004 | **Fixed** — added TC-SEC-01, a table-driven test over every durable/transport sink × every `ValidateQuestionBoundedText` denylist class (incl. case/spacing variants), driven against the real validator |
| 2 | BLOCKER | TC-002 never exercised the production `--prompt-out` write path — flag introspection alone can't prove file-write correctness | AC-013, REQ-NF-005 | **Fixed** — split into TC-002-04 (registration only) and new TC-002-09 (real `runNext` execution, real file read-back, CRLF-survival and unwritable-target sub-cases) |
| 3 | BLOCKER | Decision tables for AC-001/002/012/021/022/023 were incomplete — TC-005 never varied topology while holding coordination constant; TC-010's capability flags were tested singly, never in combination; the responder ladder omitted several states | AC-001, AC-002, AC-012, AC-021, AC-022, AC-023 | **Fixed** — added full TC-005 decision table (9 rows, 2 independent degrade paths, both-directions independence) and TC-010 capability-vector table (5 combinations) plus a 5-state responder-outcome ladder including a deadline-boundary BVA case |
| 4 | BLOCKER | AC-017's "no rule stated twice" was tested with a single spot-checked sentence, which cannot prove absence of duplication across the whole restructured tree | AC-017, REQ-F-015 | **Fixed** — replaced with TC-003-11, a maintained rule-inventory (`{rule_id, canonical file}`) scanned across every file in the tree for near-duplicate (>80% token overlap) restatement, plus TC-003-12 for mechanical link-chain reachability |
| 5 | BLOCKER | AC-004's "zero bespoke records" was asserted by the test's own restraint (not writing one), not by inspecting production schema state; REQ-NF-001 had no concrete guard | AC-004, REQ-F-004, REQ-NF-001 | **Fixed** — added TC-004-13 (before/after `sqlite_master` table-set snapshot diff, zero new tables) and TC-004-14 (import-graph guard against retired bespoke-type package names) |
| 6 | CONCERN | Runtime observability for lease-loss/ping/interrupt/replacement/deadline events is test-only, not production-diagnosable | AC-010–012, AC-021–023 | **Deferred** — see Observability Design section's explicit deferral note; adding new telemetry is new scope beyond spec.md's D-002 commitment (zero new runtime code), not something this test plan can unilaterally add. Deferred to the task-spec author/feature owner. |
| 7 | CONCERN | Gate-state coverage incomplete (`answering` state untested), no `status set`-path scan, provider references never test a no-evidence-marks-unsupported case, retirement scan used only 2 ad hoc phrases | AC-005, AC-006, AC-015, AC-018 | **Fixed** — added TC-004-05b (`answering` state), TC-003-03b (`status set` grep across all Rider-executable paths), TC-003-08b/09b (no-evidence op correctly marked unsupported), TC-007-03 (enumerated forbidden-vocabulary list) |
| 8 | CONCERN | TC-008's parity-gate design compared a compiled (immutable) `go:embed` tree directly, which cannot host injected drift/embedded-only fixtures | AC-016, REQ-F-016 | **Fixed** — redesigned as a pure `compareParity(authored, embedded fs.FS) []Drift` function; real-tree invocation is the CI gate (TC-008-01/04), `fstest.MapFS` fixtures unit-test the comparator itself (TC-008-02/03) |

No second codex pass was run after applying fixes 1–5 and 7–8 (single-round
incorporation, per the workflow's guidance to enumerate by class and avoid
iterative one-finding-at-a-time loops); the fixes above were derived
directly from codex's own proposed remediation for each finding, not
re-guessed.

---

## Recommendations

- [x] Ready for development (no drift, spec is clear, every AC has a
      technique + ISO matrix entry + caller-path contract, observability
      designed, codex red-team above — 7 of 8 findings incorporated, 1
      explicitly deferred to the task-spec author with rationale).
- [ ] Needs BA refinement — not needed; no BLOCKER drift found.
- [ ] Needs tech refinement — one non-blocking spec gap noted (AC-016's sync
      helper mechanism, Drift Finding #3): the task-spec author should name
      the sync mechanism (script, Makefile target, or manual copy) explicitly
      rather than leaving it implementation-defined. Does not block
      development.

