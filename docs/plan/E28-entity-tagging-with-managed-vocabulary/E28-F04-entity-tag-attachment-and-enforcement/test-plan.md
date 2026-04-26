---
feature_key: E28-F04-entity-tag-attachment-and-enforcement
epic_key: E28
document_type: test-plan
title: Test Plan — Entity Tag Attachment and Enforcement (E28-F04)
---

# E28-F04 — Test Plan

Traceability: every test case below maps to at least one Acceptance Criterion
(AC-1..28) in `./spec.md` §1.3, and every AC has at least one test here. AC
traces that feed into epic-level UAT scenarios in `../uat-plan.md` are
cross-referenced in §2.

Testing rules in force (see `.claude/rules/testing/architecture.md`):
- Only **repository tests** use the real DB. F04 adds **zero** repository
  changes, so F04 adds **zero** repository tests.
- **Service tests** use mocked `TagRepositoryInterface`,
  `EntityTagRepositoryInterface`, `maintainer.Gate`, and
  `TagEnforcementConfig`. Per-entity service tests use a shared
  `MockTagService` (new file).
- **CLI tests** use a mocked `tagServiceIface` and entity service mocks. No
  real DB.
- **Integration-adjacent CLI tests** that need round-tripping stdout/stderr
  use the existing pattern in `internal/cli/commands/tags_test.go`
  (in-process Cobra invocation with a mocked service).

---

## 1. AC Test Matrix

One row per AC from spec.md §1.3. `Location` is the target test file and test
name. `Kind` mirrors the spec column. Edge-case rows are appended per AC
where applicable.

### 1.1 TagService pure methods (AC-1 .. AC-14, AC-28)

| AC | Test Name | Kind | Input / Setup | Expected Outcome | Location |
|---|---|---|---|---|---|
| AC-1 | `TestAttachMany_HappyPathMultipleTags` | Unit (mock repos) | `entityType=Task`, `entityID=42`, `names=["voice","auth"]`; both registered in vocabulary | Returns `nil`. Exactly 2 `GetByName` calls and 2 `Attach` calls in input order. `Attach` receives resolved `tag.ID` values, not names. | `internal/services/tag_service_test.go` |
| AC-2 | `TestAttachMany_AbortsOnUnregisteredBeforeAnyAttach` | Unit | `names=["voice","does-not-exist"]`; only `voice` registered | Returns `*UnregisteredTagError{Name:"does-not-exist"}`. Zero `Attach` calls total (both resolve-phase and attach-phase respected). | same file |
| AC-3 | `TestAttachMany_NilSliceIsNoOp` | Unit | `names=nil` | Returns `nil`. Zero repository calls (no `GetByName`, no `Attach`). Span still emitted. | same file |
| AC-3b | `TestAttachMany_EmptySliceIsNoOp` | Unit edge | `names=[]string{}` | Same as AC-3. | same file |
| AC-4 | `TestAttachMany_NormalizesNames` | Unit | `names=["Voice "]` | Normalizes via `ValidateName`; `GetByName("voice")` invoked exactly once; `Attach` invoked once; returns `nil`. | same file |
| AC-4b | `TestAttachMany_RejectsInvalidName` | Unit edge | `names=["VOICE!!"]` (fails ValidateName regex) | Returns `*ValidationError`. No `GetByName`, no `Attach`. | same file |
| AC-5 | `TestAttachMany_DuplicateInSameCallIssuesTwoAttaches` | Unit | `names=["voice","voice"]` | 2× `GetByName("voice")` (no in-call dedup) and 2× `Attach`. The second `Attach` is a no-op at the repo layer (INSERT OR IGNORE), but the service doesn't dedup. | same file |
| AC-6 | `TestDetachOne_HappyPath` | Unit | `name="voice"` registered and attached | 1× `GetByName("voice")`, 1× `Detach(entityType, entityID, tag.ID)`. Returns `nil`. | same file |
| AC-7 | `TestDetachOne_UnregisteredReturnsNotFound` | Unit | `name="voice"` not in vocabulary (`GetByName` returns `ErrTagNotFound`) | Returns `*NotFoundError{Name:"voice"}` (F03's type, not `UnregisteredTagError`). Zero `Detach` calls. | same file |
| AC-8 | `TestDetachOne_NotAttachedIsNoOp` | Unit | `name="voice"` registered but attachment absent (`Detach` returns nil per its no-op contract) | Returns `nil`. | same file |
| AC-8b | `TestDetachOne_NormalizesName` | Unit edge | `name=" Voice"` | `GetByName("voice")` invoked once with normalized form. | same file |
| AC-8c | `TestDetachOne_RejectsInvalidName` | Unit edge | `name="not valid!"` | Returns `*ValidationError`. Zero `GetByName`/`Detach`. | same file |
| AC-9 | `TestEnforceRequired_TypeRequiredAndTagsMissing` | Unit | `Config.TagRequiredFor=["task"]`, `entityType=Task`, `names=nil` | Returns `*TagRequiredError{EntityType:"task"}`. Zero repository calls (cfg-only path). | same file |
| AC-9b | `TestEnforceRequired_TypeRequiredAndTagsEmptySlice` | Unit edge | ..., `names=[]string{}` | Same as AC-9 (nil and `[]` behave identically). | same file |
| AC-10 | `TestEnforceRequired_TypeRequiredAndTagsPresent` | Unit | `Config.TagRequiredFor=["task"]`, `names=["voice"]` | Returns `nil`. Does NOT validate names in this method (AttachMany does). | same file |
| AC-11 | `TestEnforceRequired_OtherTypeNotRequired` | Unit | `Config.TagRequiredFor=["task"]`, `entityType=Epic`, `names=nil` | Returns `nil`. Enforcement does not leak across entity types. | same file |
| AC-12 | `TestEnforceRequired_EmptyConfigNoOp` | Unit | `Config.TagRequiredFor=nil` (and `=[]string{}`), `names=nil` | Returns `nil`. | same file |
| AC-12b | `TestEnforceRequired_CaseSensitiveTypeMatch` | Unit edge (ADR-F04-4) | `Config.TagRequiredFor=["Task"]` (wrong case), `entityType=Task`, `names=nil` | Returns `nil` — case-sensitive `==`, mis-cased entries silently no-op. Documents ADR-F04-4. | same file |
| AC-13 | `TestNewTagService_NilConfigPanics` | Unit | Construct with `cfg=nil` | Panics; message contains `requires a non-nil`. Matches existing `requireNonNil` pattern. | same file |
| AC-13b | `TestNewTagService_NilTagRepoPanics` / `NilEntityTagRepoPanics` / `NilGatePanics` | Unit (regression) | Existing constructor panics unchanged after new arg added. | Each panics with `requires a non-nil`. | same file |
| AC-14 | `TestAttachMany_DoesNotCallGate` | Unit | Construct with `MockGate` whose `Authorize` returns `*UnauthorizedError`. `names=["voice"]` registered. | Returns `nil`. `gate.Authorize` call count is 0. | same file |
| AC-14b | `TestDetachOne_DoesNotCallGate` | Unit | Same gate setup, `DetachOne` on registered+attached tag. | Returns `nil`; `gate.Authorize` count 0. | same file |
| AC-14c | `TestEnforceRequired_DoesNotCallGate` | Unit | Same. | Returns `nil`; `gate.Authorize` count 0. | same file |
| AC-28 | `TestUnregisteredTagError_Message` | Unit | `(&UnregisteredTagError{Name:"voice"}).Error()` | Equals literal `tag is not registered: voice`. | `internal/services/tag_errors_test.go` |
| AC-28b | `TestTagRequiredError_Message` | Unit | `(&TagRequiredError{EntityType:"task"}).Error()` | Equals literal `at least one tag is required for task`. | same file |

Additional non-AC TagService tests required by REQ-NF-002 (observability):

| Test Name | Kind | Purpose | Location |
|---|---|---|---|
| `TestAttachMany_EmitsSpanWithAttributes` | Unit | Uses `tracetest.NewInMemoryExporter` (pattern already present in `tag_service_test.go`) to verify span name `tag_service.attach_many` and attributes `entity.type`, `entity.id`, `tag.count`. | `internal/services/tag_service_test.go` |
| `TestDetachOne_EmitsSpanWithAttributes` | Unit | Span `tag_service.detach_one`; attributes include normalized `tag.name`. | same file |
| `TestEnforceRequired_EmitsSpan` | Unit | Span `tag_service.enforce_required`; no `tag.name` attribute; `tag.count` present. | same file |

### 1.2 Entity services Create/Update hooks (AC-15 .. AC-18)

Each row is multiplied by the six entity types
(task, feature, epic, bug, change, idea). The shape is identical; listing
each row once with the "×6" marker means the test must exist for each entity
service test file.

| AC | Test Name Template | Kind | Input / Setup | Expected Outcome | Location |
|---|---|---|---|---|---|
| AC-15 ×6 | `TestCreate<Entity>_NoTagsAndNoRequirement` | Unit | `input.Tags=nil`, `Config.TagRequiredFor=[]`, `tagSvc` wired with mock | Entity is persisted (mock `Create` invoked once). `mockTagSvc.EnforceRequiredCalls == 1` returning nil (fast path with empty names). `mockTagSvc.AttachManyCalls == 0`. | `internal/services/{task,feature,epic,bug,change_card,idea}_service_test.go` |
| AC-15b ×6 | `TestCreate<Entity>_NilTagSvcIsSkippedCleanly` | Unit | `tagSvc=nil` (optional-dep), `input.Tags=nil` | Create succeeds; no panic; no nil-deref. This is the graceful-degradation property from REQ-F-018. | same files |
| AC-16 ×6 | `TestCreate<Entity>_RequiredTypeMissingTagsAborts` | Unit | `Config.TagRequiredFor=["<entitytype>"]`, `input.Tags=nil` | Returns `*TagRequiredError`. Mock `repo.Create` invoked ZERO times (aborted before persistence). Mock `AttachMany` invoked ZERO times. | same files |
| AC-17 ×6 | `TestCreate<Entity>_TagsProvidedAttachAfterPersist` | Unit | `input.Tags=["voice"]`, tag registered | Ordering assertion: `repo.Create` invoked BEFORE `tagSvc.AttachMany`. `AttachMany` invoked exactly once with the post-insert ID. If `AttachMany` returns an error, it propagates unchanged (ADR-F04-2: entity stays persisted). | same files |
| AC-17b ×6 | `TestCreate<Entity>_AttachFailurePropagates` | Unit edge | `input.Tags=["voice"]`, mock `AttachMany` returns `*UnregisteredTagError` | Error returned to caller unchanged. Entity row is still persisted (caller visible via mock `Create` count == 1). Matches ADR-F04-2. | same files |
| AC-18 ×6 | `TestUpdate<Entity>_TagsAdditive` | Unit | `updates.Tags=["voice"]` | `AttachMany` invoked exactly once; `Detach*` never called. Returns nil. | same files |
| AC-18b ×6 | `TestUpdate<Entity>_EmptyTagsIsNoOp` | Unit | `updates.Tags=nil` AND separately `updates.Tags=[]string{}` | Both cases: `AttachMany` invoked ZERO times. | same files |

Ordering assertion strategy (AC-17): the `MockTagService` records a slice of
event strings (`"AttachMany"`, `"EnforceRequired"`); the entity-service mock
repo records `"Create"` into the same event log via a test-time callback.
Asserting `events == ["EnforceRequired", "Create", "AttachMany"]` covers the
"before key alloc" and "after insert" requirements in REQ-F-008 / REQ-F-009.

### 1.3 CLI integration (AC-19 .. AC-26)

CLI tests use the pattern in `internal/cli/commands/tags_test.go`: construct
the command via an exported factory, inject a mocked `tagServiceIface`, run
in-process via `cmd.Execute()`, capture stdout/stderr, assert exit code.
**No real DB.**

| AC | Test Name | Kind | Input / Setup | Expected Outcome | Location |
|---|---|---|---|---|---|
| AC-19 | `TestTaskCreate_WithRepeatedTagFlag` | CLI integration (mocked svc) | Args: `["create","E07","F01","x","--tag=voice","--tag=auth"]`; mock task service's `CreateTask` returns a task; mock tag svc records calls. | Exit 0. `CreateTask` called once with `input.Tags=["voice","auth"]`. After success, mocked `TagService.AttachMany` was invoked at the entity-service boundary (covered indirectly via the service mock in AC-17). | `internal/cli/commands/task_test.go` |
| AC-19b | `TestTaskCreate_CommaInTagValueIsSingleLiteral` | CLI edge (ADR-F04-5) | `--tag=voice,auth` (single flag; user error) | `input.Tags=["voice,auth"]` (one literal). Service-layer validation rejects it (invalid chars) → exit 3. Documents the "no comma split" decision. | same file |
| AC-20 | `TestTaskUpdate_DuplicateTagIsIdempotent` | CLI integration | `shark task update E07-F01-001 --tag=voice --tag=voice` with mock TagService; run TWICE. | First run: `AttachMany` called with `["voice","voice"]` (dedup at DB via INSERT OR IGNORE). Second run same. The `entity_tags` row count (simulated by the mock's internal state) is exactly 1. | same file |
| AC-21 | `TestTaskUpdate_UnregisteredTagRendersSnippet` | CLI integration | Mocked svc returns `*UnregisteredTagError{Name:"does-not-exist"}` on attach; mock list returns `["voice","auth"]`. | Exit 3. Stderr contains (a) all vocab names, (b) the literal substring `To add it: shark tags add does-not-exist`. | same file |
| AC-21b | `TestTaskUpdate_UnregisteredTagJSON` | CLI edge | Same as AC-21 but with `--json` | JSON body has `"code":"unregistered_tag"` (from REQ-F-016 mapping). | same file |
| AC-22 | `TestTaskCreate_TagRequiredByConfigFailsClearly` | CLI integration | `.sharkconfig.json` overridden in test to `{"tag_required_for":["task"]}`; mock task service returns `*TagRequiredError{EntityType:"task"}`; no `--tag` passed. Then repeat for `shark epic create` — returns success. | Task path: exit 3; stderr names `task` as requiring a tag. Epic path: exit 0. | `internal/cli/commands/task_test.go`, `epic_test.go` |
| AC-23 | `TestBugTagAdd_AttachesOnceThenIdempotent` | CLI integration | Factory-built `shark bug tag add B001 voice`; mocked svc with `AttachMany` returning nil. Run twice. | Exit 0 both times. `AttachMany` called twice with same args (idempotent at repo layer, surfaced in test by the mock counting attaches as unique via set semantics). | `internal/cli/commands/entity_tag_cmd_test.go` (new file) |
| AC-24 | `TestBugTagRm_RemovesThenIdempotent` | CLI integration | `shark bug tag rm B001 voice`; mocked svc with `DetachOne` returning nil both times. | Exit 0. Second invocation also 0 (no repeat detach error). | same new file |
| AC-25 | `TestBugTagRm_UnregisteredNameErrors` | CLI integration | `DetachOne` returns `*NotFoundError{Name:"does-not-exist"}`; mock vocab list `["voice","auth"]`. | Exit 1 (per REQ-F-016: NotFoundError→code 1). Stderr contains vocab snippet AND `To add it: shark tags add does-not-exist`. Uses the shared `handleVocabularyErrorWithSnippet` helper per spec §2.7. | same new file |
| AC-26 | `TestIdeaTagAdd_BehavesLikeOtherEntities` | CLI integration | `shark idea tag add <id> voice`, mock svc returns nil. | Exit 0. `AttachMany` called with `models.EntityTypeIdea`. | same new file |
| AC-26b | All six entity types × (add/rm) table-driven | CLI edge | Table row per entity type covering happy-path and unregistered-tag. | Per ADR-F04-1 this is the factory correctness gate. | same new file |

### 1.4 Config round-trip (AC-27)

| AC | Test Name | Kind | Input / Setup | Expected Outcome | Location |
|---|---|---|---|---|---|
| AC-27 | `TestConfig_TagRequiredFor_RoundTrip` | Unit | Marshal `Config{TagRequiredForTypes:["task","bug"]}` → unmarshal → re-marshal. | Slice preserved exactly (length, order, values). Re-marshal produces the same JSON. Pattern follows `TestConfig_Maintainer_RoundTrip` (line ~1020 of `config_test.go`). | `internal/config/config_test.go` |
| AC-27b | `TestConfig_TagRequiredFor_AbsentFieldIsNilSlice` | Unit edge | JSON with no `tag_required_for` key. | `cfg.TagRequiredFor()` returns `nil` (not `[]string{}`). Ensures `omitempty` correctness and nil-safe method. | same file |
| AC-27c | `TestConfig_TagRequiredFor_NilReceiver` | Unit edge (REQ-F-007) | `var c *Config; c.TagRequiredFor()` | Returns `nil`, does not panic. The method must handle nil receiver per spec §2.3. | same file |
| AC-27d | `TestConfig_TagRequiredFor_DefensiveCopy` | Unit edge | Call `TagRequiredFor()`, mutate returned slice, call again. | Second call returns original values — defensive copy per spec §2.3 comment. | same file |

### 1.5 Coverage summary

| AC | Test Count | Covered? |
|---|---|---|
| AC-1 through AC-28 | 28 ACs × 1 primary test each + ~25 edge/regression cases | Yes — every AC has ≥ 1 test; every edge case in §1.2 Non-Functional + §2.10 ADRs has ≥ 1 test. |

---

## 2. Integration Scenarios

These span multiple components and map to epic-level UAT scenarios in
`../uat-plan.md`. They are **not** fully automated in F04 (several require
F05's list/search pathway and F06's viewer, and by UAT design are executed
manually at epic release); F04 contributes the automated substrate.

### 2.1 Service→Service boundary: Entity service ↔ TagService

**What:** TaskService / FeatureService / EpicService / BugService /
ChangeCardService / IdeaService each hold an optional `*TagService` and
make the two-phase `EnforceRequired` → persist → `AttachMany` call
sequence.

**Verification at boundary:**
- Call order (event-log assertion on `MockTagService`).
- Nil tagSvc graceful skip (AC-15b).
- Error propagation: `TagRequiredError` aborts pre-insert; `AttachMany`
  error surfaces post-insert with entity already persisted (ADR-F04-2).
- The `models.EntityType` passed by each service matches the table in
  spec §2.6:

  | Service | Expected EntityType |
  |---|---|
  | TaskService | `EntityTypeTask` |
  | FeatureService | `EntityTypeFeature` |
  | EpicService | `EntityTypeEpic` |
  | BugService | `EntityTypeBug` |
  | ChangeCardService | `EntityTypeChange` |
  | IdeaService | `EntityTypeIdea` |

**Feeds into epic UAT:** UAT-1 (cross-entity tagging), UAT-INT-1 (apply +
rename + filter), UAT-6 (enforcement).

### 2.2 CLI→Service boundary: `--tag` flag parsing

**What:** Each of the 12 create/update commands reads `--tag` via
`StringSliceVar` and passes the slice to the DTO's new `Tags []string`
field.

**Verification at boundary:**
- Cobra parses `--tag=voice --tag=auth` into `[]string{"voice","auth"}`.
- Absence of `--tag` yields `nil`, not `[]string{}` (for `UpdateXxx`
  semantics — both are treated identically per AC-18b, but the wire shape
  is `nil` to distinguish from "empty JSON array" in future HTTP paths).
- `--tag=voice,auth` is **one literal** `"voice,auth"`, not split
  (ADR-F04-5) — covered by AC-19b.
- Service-layer invalid-char validation surfaces as exit 3.

**Feeds into epic UAT:** UAT-1, UAT-2, UAT-INT-2.

### 2.3 CLI→Service boundary: `shark <entity> tag add|rm`

**What:** The new factory `makeEntityTagCmd(entityType, resolveKey)` in
`internal/cli/commands/entity_tag_cmd.go` builds `add` and `rm`
sub-subcommands for six entity types.

**Verification at boundary:**
- `resolveKey` is invoked exactly once per invocation with the user-provided
  key; failure path returns exit 1 without touching `TagService`.
- `add` calls `TagService.AttachMany(ctx, type, id, []string{name})`.
- `rm` calls `TagService.DetachOne(ctx, type, id, name)`.
- Error rendering reuses `handleVocabularyErrorWithSnippet` for
  `UnregisteredTagError`/`NotFoundError`/`ValidationError`.
- Exit codes per REQ-F-016 table in spec §2.7.

**Feeds into epic UAT:** UAT-INT-1 step 6, UAT-INT-4.

### 2.4 Service→Config boundary: `TagEnforcementConfig` interface

**What:** `TagService` depends on a narrow `TagEnforcementConfig` interface
(one method `TagRequiredFor() []string`) so tests can stub it without
constructing a full `*config.Config`.

**Verification at boundary:**
- `*config.Config` satisfies the interface (compile-time test:
  `var _ services.TagEnforcementConfig = (*config.Config)(nil)` in
  `internal/services/tag_service_test.go`).
- Passing `nil` config panics (AC-13).
- Case-sensitive match against `entityType.String()` (AC-12b).
- Changes to config at runtime are picked up on next call (the service
  re-reads `cfg.TagRequiredFor()` each time — no caching) — regression
  test `TestEnforceRequired_ConfigReadFreshPerCall`.

**Feeds into epic UAT:** UAT-6.

### 2.5 CLI→Wiring boundary: `cli.GetTagService()` and per-entity accessors

**What:** Top-of-process wiring must pass the config into `NewTagService`
and must pass the `TagService` into each entity service constructor.

**Verification at boundary:**
- Code-review gate (no automated test): `cli.GetTagService()` in
  `internal/cli/tag_global.go` passes 4 arguments to `services.NewTagService`.
- Automated: compile-time guarantee — `NewTagService`'s signature change
  forces every call site to update; the build fails if any site is missed.
- Automated: `cmd/server/services.go` — add a smoke test
  `TestWireServices_ConstructsTagService` that instantiates `WireServices`
  with a test DB connection and asserts the returned bundle includes a
  non-nil `*TagService` and each entity service has its `tagSvc` field set.

**Feeds into epic UAT:** UAT-INT-6 (Turso parity, since the same wiring
path applies to both backends).

### 2.6 Error-rendering integration

**What:** The shared helper (spec §2.7) `handleVocabularyErrorWithSnippet`
is used by:
- `shark tags rm` (F03)
- `shark tags rename` (F03)
- `shark <entity> tag add` (F04)
- `shark <entity> tag rm` (F04)
- `shark <entity> create --tag=...` error path (F04)
- `shark <entity> update --tag=...` error path (F04)

**Verification at boundary:**
- A single table-driven test in `internal/cli/commands/tags_shared_test.go`
  (or extended `tags_test.go`) feeds each typed error
  (`UnregisteredTagError`, `NotFoundError`, `ValidationError`) and asserts
  identical stderr/exit-code output regardless of which command invoked it.
- F03 regression: existing `tags_test.go` tests still pass unchanged after
  the helper is renamed/relocated (REQ-F-015).

**Feeds into epic UAT:** UAT-2 (the SC-2 error shape is the headline AC).

### 2.7 Out-of-scope integration (for F04 automation)

The following are **epic-level** integration scenarios that F04 contributes
to but does not independently automate:

| UAT | F04 contribution | F04 test? |
|---|---|---|
| UAT-1 end-to-end with `shark list --tag=` | F04 creates the `entity_tags` rows. | Indirectly — AC-17, AC-19, AC-23. |
| UAT-5 rename + filter continuity | F04's attach is what the renamed tag preserves via tag_id reference. | No — UAT-5 requires F05's list --tag path to verify continuity. Covered by F05/F06. |
| UAT-7 viewer tag chips + filter | F04 persists tags; viewer is F06. | No. |
| UAT-INT-1 apply + rename + filter + rm | Four-step orchestration spans F03/F04/F05. | F04 covers steps 1–2, 6. |
| UAT-INT-5 cascade on delete | F01 triggers; no F04 work. | No. |
| UAT-INT-6 Turso parity | Shared SQL; no backend branches. | Covered by REQ-NF-006 + `make test` on Turso backend. |

---

## 3. Test Infrastructure

### 3.1 Existing patterns to reuse (with file paths)

| Pattern | Source file | F04 usage |
|---|---|---|
| Function-field mock for `TagRepositoryInterface` | `internal/services/tag_service_test.go` (`mockTagRepo`) | Extend with `Attach`/`Detach` expectation wrappers if needed; already covers `GetByName`. |
| Function-field mock for `EntityTagRepositoryInterface` | same file (`mockEntityTagRepo`) | Already has `attachFn`, `detachFn`. Extend with call-counting and argument capture for ordering assertions (AC-1, AC-2, AC-5). |
| Function-field mock for `maintainer.Gate` | `internal/services/tag_service_test.go` | Reuse for AC-14 (verify NOT called). |
| OTel in-memory exporter | `internal/services/tag_service_test.go` (`setupTracer`) | Reuse for `Test*_EmitsSpan*` tests. |
| Table-driven CLI test with in-process Cobra | `internal/cli/commands/tags_test.go` (`TestTagsAddCmd_HappyPath`, etc.) | Template for `task_test.go`, `entity_tag_cmd_test.go`. |
| Mock `tagServiceIface` (function-field) | `internal/cli/commands/tags_test.go` (`mockTagService`) | Extend with `attachManyFn`, `detachOneFn`, `enforceRequiredFn`. Or: new narrower `entityTagServiceIface` in `entity_tag_cmd.go` with just those three methods (cleaner). |
| Mock entity services for CLI create/update tests | `internal/cli/commands/task_test.go`, `bug_test.go`, etc. | Reuse — tests already exist for service calls; F04 extends with `--tag` expectations. |
| Config round-trip test | `internal/config/config_test.go` line ~1020 (`TestConfig_Maintainer_RoundTrip`) | Template for AC-27. |
| Constructor nil-panic test | `internal/services/tag_service_test.go` (existing `TestNewTagService_Nil*Panics`) | Extend with nil-config case (AC-13). |

### 3.2 New test helpers (required)

| Helper | File | Purpose |
|---|---|---|
| `MockTagService` | **NEW** `internal/services/mock_tag_service_test.go` (package-private, shared across `*_service_test.go` files) | Shared mock with function fields `attachManyFn`, `detachOneFn`, `enforceRequiredFn`, plus a call-log slice for ordering assertions (AC-17). Avoids 6× duplication across entity-service tests. |
| `stubTagEnforcementConfig` | `internal/services/tag_service_test.go` | One-liner `type stubCfg struct{ values []string }; func (s *stubCfg) TagRequiredFor() []string { return s.values }`. Used by AC-9..AC-12 and by entity-service tests that inject a real `TagService`. |
| `EventRecorder` | **NEW** `internal/services/mocks_event_recorder_test.go` OR inline in each service_test.go | Tiny append-only `[]string` used to record `"Create"`, `"EnforceRequired"`, `"AttachMany"` in call order (AC-17 ordering assertion). Not exported. |
| CLI fixture builder `buildTagCLIFixture(t, entityType)` | **NEW** `internal/cli/commands/entity_tag_cmd_test.go` | Returns `(*cobra.Command, *mockEntityTagService, *bytes.Buffer stderr)` wired per entity type; reduces per-table-row boilerplate in AC-23..AC-26. |
| `resolveKey` test stub | `entity_tag_cmd_test.go` | Deterministic stub returning a fixed int64 given the entity-key string; used in place of real entity-service lookups (per spec §2.7). |

### 3.3 No new repository tests

F04 adds no repository code; all existing tag repository tests in
`internal/repository/tag/*_test.go` (F01) must continue to pass unchanged.
`make test ./internal/repository/tag/...` is part of the F04 exit gate as
regression-only — no new cases.

### 3.4 Test data

- All tag names used in tests: `voice`, `auth`, `audio` (following UAT
  scenario examples for consistency with epic UAT).
- Entity keys: `E07-F01-001` (task), `B001` (bug), `CC-001` (change-card),
  numeric ideas.
- No `.sharkconfig.json` fixture files — tests construct `Config` values
  directly in Go and marshal when round-trip is needed.

---

## 4. Running the Test Suite

```bash
# Service-layer unit tests (all new F04 service logic)
go test -v ./internal/services/ -run 'TestAttachMany|TestDetachOne|TestEnforceRequired|TestNewTagService|TestUnregisteredTagError|TestTagRequiredError|TestCreate(Task|Feature|Epic|Bug|ChangeCard|Idea)_|TestUpdate(Task|Feature|Epic|Bug|ChangeCard|Idea)_'

# Config round-trip
go test -v ./internal/config/ -run TestConfig_TagRequiredFor

# CLI tests (new entity tag subcommand + --tag flag tests)
go test -v ./internal/cli/commands/ -run 'Test(Task|Feature|Epic|Bug|Change|Idea)(Create|Update|Tag)'
go test -v ./internal/cli/commands/ -run TestEntityTagCmd

# Full gate (MANDATORY per .claude/rules/development-workflows.md)
make fmt && make lint && make test
```

---

## 5. Exit Gate (Test Plan Itself)

- [x] Every AC (AC-1..AC-28) in `./spec.md` §1.3 has ≥ 1 test in §1.1–§1.4.
- [x] Edge cases identified for every AC cluster (AC-3b/3, AC-4b, AC-8b/8c,
      AC-9b, AC-12b, AC-13b, AC-14b/14c, AC-15b, AC-17b, AC-18b, AC-19b,
      AC-21b, AC-26b, AC-27b/27c/27d).
- [x] Integration scenarios cover every service↔service, CLI↔service, and
      service↔config boundary introduced by F04 (§2.1–§2.6).
- [x] Each integration scenario traces to an epic UAT scenario (table in
      §2.7, column "Feeds into epic UAT").
- [x] Test patterns reference existing infrastructure with file paths
      (§3.1).
- [x] New test helpers enumerated with locations (§3.2) — 4 new helpers,
      1 of which is a new file required by spec §2.13.
- [x] No orphaned tests — every test ties back to a spec requirement (via
      AC or NF-requirement).
- [x] No real-DB tests outside existing repository test suites (per
      `.claude/rules/testing/architecture.md`).

---

*Last Updated*: 2026-04-23
