---
feature_key: E28-F03-tag-vocabulary-service-and-cli
epic_key: E28
document_type: test-plan
title: "Test Plan — Tag Vocabulary Service and CLI (E28-F03)"
---

# Test Plan — Tag Vocabulary Service and CLI (E28-F03)

This document defines the complete testing strategy for feature **E28-F03**.
Every section traces back to a specific acceptance criterion (AC-N) from
`spec.md §1.3`. No test case is orphaned from a requirement.

References:
- Spec: `docs/plan/E28-entity-tagging-with-managed-vocabulary/E28-F03-tag-vocabulary-service-and-cli/spec.md`
- Epic UAT plan: `docs/plan/E28-entity-tagging-with-managed-vocabulary/uat-plan.md`
- Testing architecture: `.claude/rules/testing/architecture.md`
- Service testing patterns: `.claude/rules/services/testing.md`
- CLI testing patterns: `.claude/rules/testing/cli-tests.md`

---

## 1. AC Test Matrix

Each AC from `spec.md §1.3` has one or more test cases. Test cases are
grouped by test suite type (service unit, CLI unit, integration, static).

### Suite Legend

| Suite | File | DB? | Mocks |
|---|---|---|---|
| **SVC** | `internal/services/tag_service_test.go` | None | `tag.TagRepositoryInterface`, `tag.EntityTagRepositoryInterface`, `maintainer.Gate` |
| **CLI** | `internal/cli/commands/tags_test.go` | None | local `tagServiceIface` |
| **INT** | `internal/cli/tag_global_test.go` | Real (TempDir) | None |
| **STATIC** | In-process `go list` assertions | None | None |

---

### AC-1 — `ListTags` returns tags in ascending name order; gate never called

**Requirement:** REQ-F-002, REQ-F-004. `ListTags` is open to all users.

**Suite:** SVC

| # | Test case | Setup | Assertion |
|---|---|---|---|
| 1.1 | Happy path — ordered output | `tagRepo.List` returns `[{name:"voice"},{name:"audio"}]` out of order; gate records all method calls | Result slice is `["audio","voice"]`; gate call count is 0 |
| 1.2 | Empty vocabulary | `tagRepo.List` returns `[]` | Returns empty slice, nil error, gate never called |
| 1.3 | Repository error propagation | `tagRepo.List` returns `fmt.Errorf("db error")` | Error is returned unwrapped; gate never called |

**Edge cases:**
- Single-tag vocabulary: result is a one-element slice.
- 100-tag vocabulary: ordering is lexicographic, not insert-order.

---

### AC-2 — `AddTag` normalizes, authorizes, creates, records success in order

**Requirement:** REQ-F-002, REQ-F-003, REQ-F-004.

**Suite:** SVC

| # | Test case | Setup | Assertion |
|---|---|---|---|
| 2.1 | Happy path with mixed-case input | Input `"Voice"`, gate returns nil, `repo.Create` captures name | Name passed to `Create` is `"voice"` (lowercased + trimmed); call order is Authorize → Create → RecordSuccess |
| 2.2 | Leading/trailing whitespace | Input `"  audio  "` | Normalized to `"audio"` before `Create` |
| 2.3 | Lowercase input unchanged | Input `"audio"` | `"audio"` passed to `Create`; no double-normalization |
| 2.4 | Conflict from repository | `repo.Create` returns `ErrTagConflict` | Returns `*ConflictError{Name:"audio"}`; RecordSuccess NOT called |
| 2.5 | RecordSuccess error is swallowed | `gate.RecordSuccess` returns `fmt.Errorf("disk full")` | Method returns nil error; created tag is returned |

**Edge cases:**
- Name that becomes exactly 64 chars after normalization: accepted.
- Name with a leading digit after trim (e.g., `"1voice"`): accepted (`^[a-z0-9]` regex).

---

### AC-3 — `AddTag` with invalid name: Authorize called first, Create NOT called

**Requirement:** REQ-F-004 (authorize-first order, D1), REQ-F-003.

**Suite:** SVC

| # | Test case | Setup | Assertion |
|---|---|---|---|
| 3.1 | Name with special char | Input `"Voice!"`, gate returns nil | `Authorize` call count = 1; `Create` call count = 0; error is `*ValidationError{Field:"tag name"}` |
| 3.2 | Empty string | Input `""` | `Authorize` called; `Create` not called; `*ValidationError` returned |
| 3.3 | Whitespace-only | Input `"   "` (collapses to `""`) | Same as 3.2 |
| 3.4 | Name > 64 chars | Input of 65-char lowercase string | `Authorize` called; `Create` not called; `*ValidationError` returned |
| 3.5 | Name starts with hyphen | Input `"-voice"` (fails regex) | `Authorize` called; `Create` not called; `*ValidationError` returned |

**Edge cases:**
- `"a-b"` (hyphen in middle): valid — Authorize, then Create.
- `"a"` (single char, minimum): valid.

---

### AC-4 — `AddTag` unauthorized: gate error returned unwrapped, Create not called

**Requirement:** REQ-F-004, REQ-F-007.

**Suite:** SVC

| # | Test case | Setup | Assertion |
|---|---|---|---|
| 4.1 | Wrong password | Gate returns `*UnauthorizedError{Reason:"wrong_password"}` | `errors.As(err, &unauthorizedErr)` is true; `Create` not called |
| 4.2 | Missing config | Gate returns `*UnauthorizedError{Reason:"missing_config"}` | `errors.As(err, &unauthorizedErr)` is true; `Create` not called |
| 4.3 | Expired cache | Gate returns `*UnauthorizedError{Reason:"expired_cache"}` | `errors.As(err, &unauthorizedErr)` is true; `Create` not called |

**Edge cases:**
- Error is NOT double-wrapped: `errors.As` must find the type directly, not inside a `fmt.Errorf` wrapper.

---

### AC-5 — `RemoveTag` with in-use tag and `force=false`: `*TagInUseError` returned, Delete NOT called

**Requirement:** REQ-F-005.

**Suite:** SVC

| # | Test case | Setup | Assertion |
|---|---|---|---|
| 5.1 | 7 uses, force=false | `GetByName` returns tag; `CountByTag` returns 7; `Delete` records call | Returns `*TagInUseError{Name:"voice", Count:7}`; `Delete` call count = 0 |
| 5.2 | 1 use, force=false | `CountByTag` returns 1 | Returns `*TagInUseError{Name:"voice", Count:1}`; `Delete` not called |
| 5.3 | Error message includes count and --force hint | AC-5.1 error | `err.Error()` contains `"7"` and `"--force"` |

---

### AC-6 — `RemoveTag` with in-use tag and `force=true`: Delete(force=true) called, RecordSuccess called

**Requirement:** REQ-F-005.

**Suite:** SVC

| # | Test case | Setup | Assertion |
|---|---|---|---|
| 6.1 | 7 uses, force=true | `CountByTag` returns 7; `Delete` captures `(id, force)` args; gate returns nil | `Delete` called with `force=true`; `RecordSuccess` called once |
| 6.2 | Delete fails | `Delete` returns error | Method returns error; `RecordSuccess` NOT called |

---

### AC-7 — `RemoveTag` with zero uses and `force=false`: Delete(force=false) called

**Requirement:** REQ-F-005.

**Suite:** SVC

| # | Test case | Setup | Assertion |
|---|---|---|---|
| 7.1 | 0 uses, force=false | `CountByTag` returns 0; `Delete` records args | `Delete` called with `force=false`; `RecordSuccess` called once |
| 7.2 | Tag not found | `GetByName` returns `sql.ErrNoRows` or equivalent | Returns `*NotFoundError{Name:"nonexistent"}`; `Delete` not called |

---

### AC-8 — `RenameTag` collision: `*ConflictError` returned, `repo.Rename` NOT called

**Requirement:** REQ-F-006, ADR-8.

**Suite:** SVC

| # | Test case | Setup | Assertion |
|---|---|---|---|
| 8.1 | Target name exists | `GetByName("voice")` returns source tag; `GetByName("audio")` returns existing tag | Returns `*ConflictError{Name:"audio"}`; `Rename` call count = 0 |
| 8.2 | Pre-check protects against UNIQUE error | Same as 8.1 | Confirm `Rename` is never called even though the DB would reject it too |

---

### AC-9 — `RenameTag` with same old and new name (after normalization): `*ValidationError` returned

**Requirement:** REQ-F-006.

**Suite:** SVC

| # | Test case | Setup | Assertion |
|---|---|---|---|
| 9.1 | Identical after normalization | `oldName="voice"`, `newName="VOICE"` | Returns `*ValidationError` citing names must differ; `Rename` not called |
| 9.2 | Identical before normalization | `oldName="voice"`, `newName="voice"` | Same as 9.1 |
| 9.3 | Differ only by whitespace | `oldName="voice"`, `newName=" voice "` | Same as 9.1 after trim |

---

### AC-10 — `RenameTag` success: `repo.Rename` called once; no `EntityTagRepository` method called

**Requirement:** REQ-F-006, epic SC-5.

**Suite:** SVC

| # | Test case | Setup | Assertion |
|---|---|---|---|
| 10.1 | Happy path | `GetByName("voice")` returns source; `GetByName("audio")` returns not-found; `Rename(id,"audio")` returns updated tag | `Rename` call count = 1; any method on `entityTagRepo` mock call count = 0; returned tag has `Name:"audio"` |
| 10.2 | Race: Rename returns `ErrTagConflict` | `Rename` returns `tag.ErrTagConflict` | Returns `*ConflictError`; `RecordSuccess` not called |
| 10.3 | Source tag not found | `GetByName(oldName)` returns not-found | Returns `*NotFoundError{Name:"voice"}` |

---

### AC-11 — `RecordSuccess` error does NOT propagate as method failure; error is logged

**Requirement:** REQ-F-004.

**Suite:** SVC

| # | Test case | Setup | Assertion |
|---|---|---|---|
| 11.1 | AddTag: RecordSuccess fails | `gate.RecordSuccess` returns `fmt.Errorf("disk full")`; a test logger is installed | `AddTag` returns nil error; the created tag is returned; the logger captures the error |
| 11.2 | RemoveTag: RecordSuccess fails | Same pattern for `RemoveTag` | Method returns nil; logger captures the error |
| 11.3 | RenameTag: RecordSuccess fails | Same pattern for `RenameTag` | Method returns nil; logger captures the error |

**Note on logger verification:** The implementation must call a logger (e.g., `log.Printf` or a configurable logger field on the service). The test verifies this by capturing `log.Default()` output or by injecting a test logger. If the implementation does not expose a logger seam, the test verifies that `RecordSuccess`'s error is swallowed by confirming the method's return value is nil.

---

### AC-12 — `cli.GetTagService()` returns a working `*TagService` against a real TempDir DB

**Requirement:** REQ-F-012.

**Suite:** INT (real DB in `t.TempDir()`)

| # | Test case | Setup | Assertion |
|---|---|---|---|
| 12.1 | Smoke test against real DB | Create a temp project dir with an initialized DB (schema applied); set the global DB to that dir; call `cli.GetTagService().ListTags(ctx)` | Returns non-nil `*TagService`; `ListTags` returns `[]` or `nil` without error |
| 12.2 | Accessor constructs fresh instance per call | Call `GetTagService()` twice | Two distinct pointer values; neither panics |

**Setup pattern** (matches existing `tag_global_test.go` precedent in F02):
```go
func TestGetTagService_Smoke(t *testing.T) {
    tmpDir := t.TempDir()
    // Write minimal .sharkconfig.json (no maintainer block)
    // Initialize DB schema via cli.GetDB(ctx) with override
    defer cli.ResetDB()
    svc := cli.GetTagService()
    tags, err := svc.ListTags(context.Background())
    // assert no error, tags is empty slice
}
```

---

### AC-13 — `shark tags list --json` in empty project: `[]\n`, exit 0

**Requirement:** REQ-F-009, REQ-F-011.

**Suite:** CLI

| # | Test case | Setup | Assertion |
|---|---|---|---|
| 13.1 | Empty vocabulary, JSON flag | Mock service `ListTags` returns `[]` | stdout is `[]\n`; exit code 0 |
| 13.2 | Non-empty vocabulary, JSON flag | Mock service `ListTags` returns `[{name:"audio"},{name:"voice"}]` | stdout is `[{"name":"audio"},{"name":"voice"}]\n`; IDs and timestamps NOT in output |
| 13.3 | Non-empty vocabulary, plain text | Same mock, no `--json` | stdout contains `audio` and `voice` in a human-readable format; exit 0 |
| 13.4 | Service error | Mock `ListTags` returns `fmt.Errorf("db error")` | exit code non-zero; error message on stderr |

---

### AC-14 — `shark tags add voice --pass wrong`: exit 3, stderr contains gate error message

**Requirement:** REQ-F-010, REQ-F-007.

**Suite:** CLI

| # | Test case | Setup | Assertion |
|---|---|---|---|
| 14.1 | Wrong password | Mock `AddTag` returns `*UnauthorizedError{Reason:"wrong_password"}` | exit 3; stderr contains `"incorrect maintainer password"` |
| 14.2 | JSON error format | Same, with `--json` | stderr JSON: `{"error":"unauthorized","message":"..."}` |

---

### AC-15 — `shark tags add voice` (no --pass, no cache): exit 3, stderr contains `UserHint()` text

**Requirement:** REQ-F-010, REQ-F-007, epic SC-3.

**Suite:** CLI

| # | Test case | Setup | Assertion |
|---|---|---|---|
| 15.1 | Missing config unauthorized | Mock `AddTag` returns `*UnauthorizedError{Reason:"missing_config"}` with `UserHint()` = `"run: shark admin maintainer set-password"` | exit 3; stderr contains `"shark admin maintainer set-password"` |
| 15.2 | Hint appears on second line | Same | stderr has the gate error on line 1 and the hint on line 2 (or visually separated) |
| 15.3 | No hint (expired cache) | Mock `AddTag` returns `*UnauthorizedError{Reason:"expired_cache"}` with `UserHint()` = `""` | exit 3; no second hint line emitted |

---

### AC-16 — `shark tags rm voice` with 7 uses: exit 3, stderr describes usage and --force

**Requirement:** REQ-F-010, REQ-F-005.

**Suite:** CLI

| # | Test case | Setup | Assertion |
|---|---|---|---|
| 16.1 | In-use tag | Mock `RemoveTag` returns `*TagInUseError{Name:"voice", Count:7}` | exit 3; stderr contains `"is in use by 7 entities"` and `"--force"` |
| 16.2 | JSON error format | Same, with `--json` | stderr JSON: `{"error":"in_use","message":"..."}` containing count and flag hint |

---

### AC-17 — `shark tags rm nonexistent --pass pw`: exit 1, stderr lists vocabulary and `shark tags add` hint

**Requirement:** REQ-F-008, REQ-F-010.

**Suite:** CLI

| # | Test case | Setup | Assertion |
|---|---|---|---|
| 17.1 | Not-found error | Mock `RemoveTag` returns `*NotFoundError{Name:"nonexistent"}`; mock `ListTags` returns `[{name:"audio"},{name:"voice"}]` | exit 1; stderr contains `"tag not found: nonexistent"`; stderr lists `audio` and `voice`; stderr contains `"shark tags add nonexistent"` or the `shark tags add` hint |
| 17.2 | More than 10 tags in vocabulary | Mock `ListTags` returns 15 tags | stderr shows first 10 names and `"…and 5 more"` |
| 17.3 | Same pattern for rename not-found | Mock `RenameTag` returns `*NotFoundError{Name:"oldname"}` | Same vocabulary snippet + hint in stderr |
| 17.4 | JSON error format | `--json` on 17.1 | stderr JSON: `{"error":"not_found","message":"..."}` |

---

### AC-18 — `shark tags rename voice audio --pass pw`: success output, JSON output

**Requirement:** REQ-F-009, REQ-F-011.

**Suite:** CLI

| # | Test case | Setup | Assertion |
|---|---|---|---|
| 18.1 | Plain text output | Mock `RenameTag` returns `*models.Tag{Name:"audio"}` | stdout contains `"Renamed voice to audio"` (or equivalent); exit 0 |
| 18.2 | JSON output | Same, with `--json` | stdout is `{"old":"voice","new":"audio"}\n`; exit 0 |
| 18.3 | Add success — plain text | Mock `AddTag` returns `*models.Tag{Name:"voice"}` | stdout contains `"Added tag voice"` or equivalent; exit 0 |
| 18.4 | Add success — JSON | Same, with `--json` | stdout is `{"name":"voice"}\n` |
| 18.5 | Rm success — plain text | Mock `RemoveTag` returns nil | stdout contains `"Removed tag voice"` or equivalent; exit 0 |
| 18.6 | Rm success — JSON | Same, with `--json` | stdout is `{"name":"voice","removed":true}\n` |

---

### AC-19 — `TagService` package imports include tag/auth repos; exclude cli and cobra

**Requirement:** REQ-F-001, REQ-NF-003.

**Suite:** STATIC

| # | Test case | Method | Assertion |
|---|---|---|---|
| 19.1 | Imports include tag repository | `go list -f '{{.Imports}}' ./internal/services/` | Import list contains `github.com/jwwelbor/shark-task-manager/internal/repository/tag` |
| 19.2 | Imports include auth/maintainer | Same | Import list contains `github.com/jwwelbor/shark-task-manager/internal/auth/maintainer` |
| 19.3 | Imports exclude internal/cli | Same | `internal/cli` NOT in import list |
| 19.4 | Imports exclude cobra | Same | `github.com/spf13/cobra` NOT in import list |

**Pattern:** In-process test using `os/exec` to run `go list -f '{{.Imports}}' ./internal/services/` and assert on stdout. Follows the pattern established in `internal/auth/maintainer` package static tests from F02.

---

### AC-20 — CLI `tags.go` does NOT import tag repo or database/sql directly

**Requirement:** REQ-NF-001.

**Suite:** STATIC

| # | Test case | Method | Assertion |
|---|---|---|---|
| 20.1 | No direct repo import | `go list -f '{{.Imports}}' ./internal/cli/commands/` | `internal/repository/tag` NOT in import list |
| 20.2 | No database/sql | Same | `database/sql` NOT in import list for the commands package |
| 20.3 | auth/maintainer only via errors.As | Manual review or grep | Only import of `internal/auth/maintainer` is for `*maintainer.UnauthorizedError` type assertion |

---

### AC-21 — Documentation files exist and are complete

**Requirement:** REQ-F-013.

**Suite:** Manual review / docs gate check

| # | Check | How to verify |
|---|---|---|
| 21.1 | `docs/cli-reference/tags.md` exists | `os.Stat` in a test or filesystem check |
| 21.2 | `tags.md` documents all four subcommands | File contains `list`, `add`, `rm`, `rename` as section headings |
| 21.3 | `tags.md` documents `--pass` flag | File contains `--pass` |
| 21.4 | `tags.md` documents `--force` flag on `rm` | File contains `--force` |
| 21.5 | `tags.md` documents JSON output shapes for all four commands | File contains all JSON shapes from REQ-F-011 |
| 21.6 | `docs/cli-reference/README.md` has a link to `tags.md` | File contains `tags.md` as a link or table entry |

---

### AC-22 — `TagService` spans contain tag name but NOT password attributes

**Requirement:** REQ-NF-002.

**Suite:** SVC (in-memory OTel recorder)

| # | Test case | Setup | Assertion |
|---|---|---|---|
| 22.1 | `AddTag` span exists | Call `AddTag("voice", "pw")`; use `sdktrace` in-memory exporter | A span named `tag_service.add_tag` (or `TagService.AddTag`) is recorded |
| 22.2 | Span contains `tag.name` attribute | Same | Span attributes include `tag.name = "voice"` |
| 22.3 | Span does NOT contain password attributes | Same | No attribute key matches `pass`, `password`, `hash`, or `maintainer.*` |
| 22.4 | `RemoveTag` span has `tag.force` attribute | Call `RemoveTag("voice", true, "pw")` | Span attributes include `tag.force = true` |
| 22.5 | Spans for all mutating methods | `AddTag`, `RemoveTag`, `RenameTag`, `ListTags` | Each has exactly one span with the correct name |

**OTel test setup pattern** (matches `feature_service_tracing_test.go`):
```go
func newTestTracerProvider() (*sdktrace.TracerProvider, *tracetest.SpanRecorder) {
    sr := tracetest.NewSpanRecorder()
    tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
    return tp, sr
}
```

---

## 2. Integration Scenarios

These scenarios verify cross-component interactions between `TagService`,
the `maintainer.Gate`, and the CLI command layer.

### INT-1: Full happy-path round-trip — add, list, rename, remove (service level)

**Components:** `TagService` ↔ mock `TagRepository` ↔ mock `Gate`
**UAT scenarios contributed to:** UAT-1, UAT-5

**Flow:**
1. `AddTag(ctx, "voice", "pw")` → returns `*models.Tag{Name:"voice"}`
2. `ListTags(ctx)` → returns `[{name:"voice"}]`
3. `RenameTag(ctx, "voice", "audio", "pw")` → returns `*models.Tag{Name:"audio"}`
4. `ListTags(ctx)` → returns `[{name:"audio"}]`
5. `RemoveTag(ctx, "audio", false, "pw")` with `CountByTag=0` → returns nil

**Boundary verifications:**
- Gate is called exactly once per mutating step.
- `RecordSuccess` is called exactly once per mutating step.
- `EntityTagRepository` is never called during `RenameTag` (AC-10).
- `List` returns updated name after rename.

---

### INT-2: Gate reuse — CLI command wires gate once, passes password through to service

**Components:** CLI `tags add` handler → `cli.GetTagService()` → `TagService.gate.Authorize`
**UAT scenario:** UAT-8 (reusability of gate)

**What to verify at the boundary:**
- `tags.go` never constructs a `FileGate` or reads `PasswordHash` directly.
- `tags.go` receives `--pass` from the flag and passes it as-is to `svc.AddTag(ctx, name, pass)`.
- The service is the only caller of `gate.Authorize`.

**Test approach:** Static check (AC-20) plus a CLI test where the mock `tagServiceIface.AddTag` captures the `pass` argument and asserts it equals the CLI flag value.

---

### INT-3: Error translation at CLI boundary — typed errors map to exit codes

**Components:** CLI error handler ↔ service typed errors
**UAT scenarios:** UAT-2, UAT-3, UAT-INT-4

**Boundary table:**

| Service error type | Expected exit code | Expected stderr content |
|---|---|---|
| `*NotFoundError` | 1 | `"tag not found: <name>"` |
| `*maintainer.UnauthorizedError` | 3 | gate reason + optional hint |
| `*ConflictError` | 3 | `"tag already exists: <name>"` |
| `*TagInUseError` | 3 | `"is in use by N entities"` + `"--force"` |
| `*ValidationError` | 3 | `"invalid <field>: <message>"` |
| generic `error` | 2 | generic error message |

**Test approach:** One CLI table-driven test exercises all six rows by feeding mock `tagServiceIface` errors and checking exit code + stderr content.

---

### INT-4: `RemoveTag` force-cascade preserves gate + RecordSuccess ordering

**Components:** `TagService` ↔ mock `EntityTagRepository`
**UAT scenario:** UAT-INT-4 (remove-in-use safety net)

**Flow:**
1. `RemoveTag(ctx, "voice", true, "pw")` with `CountByTag=3`.
2. Verify call order: `Authorize` → `GetByName` → `CountByTag` → `Delete(id, true)` → `RecordSuccess`.
3. Verify `Delete` is called with `force=true` (which internally removes entity_tags rows per F01 spec).
4. Verify `RecordSuccess` is called after `Delete` succeeds.

**What is NOT tested here:** The actual entity_tags deletion (that is F01's repository responsibility, tested in F01's repository tests).

---

### INT-5: Vocabulary snippet surfaced in CLI error for rm/rename not-found

**Components:** CLI `rm` command → `tagServiceIface.RemoveTag` + `tagServiceIface.ListTags`
**UAT scenario:** UAT-2 (actionable error text)

**What to verify at the boundary:**
- When `RemoveTag` returns `*NotFoundError`, the CLI handler calls `ListTags` (via a second service call) to assemble the vocabulary snippet.
- The vocabulary snippet in stderr contains up to 10 names with a "…and N more" suffix if the list is longer.
- The `shark tags add <notFoundName>` command is included literally in stderr.

**Test approach:** CLI test with a mock `tagServiceIface` whose `RemoveTag` returns `*NotFoundError` and whose `ListTags` returns a configurable vocabulary. Assert stderr contents.

---

## 3. Test Infrastructure

### 3.1 Existing infrastructure to follow

| File / Pattern | What it provides | Use for F03 |
|---|---|---|
| `internal/services/bug_service_test.go` | Function-field mock pattern (e.g., `mockBugRepo.createFn`) | Template for `mockTagRepo`, `mockEntityTagRepo`, `mockGate` |
| `internal/cli/commands/admin_maintainer_test.go` | CLI test with injectable service interface (`maintainerBootstrapServiceIface`) and cobra command construction helper | Template for `buildTagsCmdWithMock(svc tagServiceIface)` and all CLI tests |
| `internal/services/feature_service_tracing_test.go` | OTel in-memory span recording with `newTestTracer` | Template for AC-22 tracing tests |
| `internal/auth/maintainer/*_test.go` | Static import checks via `go list` in-process | Template for AC-19 and AC-20 static tests |
| `internal/cli/tag_global_test.go` (F02 analogue) | TempDir integration test for accessor smoke test | Template for AC-12 |

### 3.2 New test helpers needed

**`internal/services/tag_service_test.go` — in-file mocks:**

```go
// mockTagRepo implements tag.TagRepositoryInterface via function fields.
type mockTagRepo struct {
    createFn    func(ctx context.Context, name string) (*models.Tag, error)
    getByNameFn func(ctx context.Context, name string) (*models.Tag, error)
    getByIDFn   func(ctx context.Context, id int64) (*models.Tag, error)
    listFn      func(ctx context.Context) ([]*models.Tag, error)
    renameFn    func(ctx context.Context, id int64, newName string) (*models.Tag, error)
    deleteFn    func(ctx context.Context, id int64, force bool) error
}
// + method implementations delegating to function fields (nil → default no-op)

// mockEntityTagRepo implements tag.EntityTagRepositoryInterface.
type mockEntityTagRepo struct {
    countByTagFn func(ctx context.Context, tagID int64) (int64, error)
    // other methods as no-ops (never called by TagService directly)
}

// mockGate implements maintainer.Gate.
type mockGate struct {
    authorizeFn     func(ctx context.Context, pass string) error
    recordSuccessFn func(ctx context.Context) error
    authCalls       []string  // captures pass values for call-order verification
    recordCalls     int
}
```

**`internal/cli/commands/tags_test.go` — service mock:**

```go
// mockTagService implements tagServiceIface (the local package interface).
type mockTagService struct {
    listTagsFn   func(ctx context.Context) ([]*models.Tag, error)
    addTagFn     func(ctx context.Context, name, pass string) (*models.Tag, error)
    removeTagFn  func(ctx context.Context, name string, force bool, pass string) error
    renameTagFn  func(ctx context.Context, old, new, pass string) (*models.Tag, error)
}

// buildTagsCmdWithMock creates a fresh cobra root with injected service,
// mirrors buildSetPasswordCmdWithMock in admin_maintainer_test.go.
func buildTagsCmdWithMock(svc tagServiceIface) *cobra.Command { ... }
```

**Call-order recorder helper** (shared within `tag_service_test.go`):

```go
type callRecorder struct {
    mu    sync.Mutex
    calls []string
}

func (r *callRecorder) record(name string) { r.mu.Lock(); r.calls = append(r.calls, name); r.mu.Unlock() }
func (r *callRecorder) assertOrder(t *testing.T, expected ...string) {
    if !reflect.DeepEqual(r.calls, expected) {
        t.Errorf("call order: got %v, want %v", r.calls, expected)
    }
}
```

### 3.3 No new repository tests required

`TagService` tests are entirely mock-based (per testing architecture rule:
only repository tests use the real database). F01 already ships the
repository tests for `TagRepository` and `EntityTagRepository`. F03 adds
no schema changes, so no new migration-path repository tests are needed.

### 3.4 Test file inventory

| File | Suite type | New / Existing |
|---|---|---|
| `internal/services/tag_service_test.go` | SVC (service unit) | New |
| `internal/cli/commands/tags_test.go` | CLI (command unit) | New |
| `internal/cli/tag_global_test.go` | INT (integration smoke) | New |
| Static assertions (in `tag_service_test.go` or standalone `_imports_test.go`) | STATIC | New |

---

## 4. Exit Gate Checklist

Before advancing E28-F03 out of `in_test_planning`:

- [x] Every AC in `spec.md §1.3` (AC-1 through AC-22) has at least one test case in this plan.
- [x] Every test case specifies: setup, mocks, concrete assertion.
- [x] Edge cases are identified for each AC.
- [x] Integration scenarios cover all cross-component boundaries (service ↔ gate, CLI ↔ service, error translation layer).
- [x] Test infrastructure references existing patterns (no novel frameworks introduced).
- [x] New test helpers are described (function-field mocks, call recorder, cobra builder).
- [x] No real database in service tests or CLI tests (per `.claude/rules/testing/architecture.md`).
- [x] Epic UAT traceability: AC-2/13 → UAT-1; AC-17 → UAT-2; AC-14/15 → UAT-3; AC-10 → UAT-5; AC-19/20 → UAT-8; AC-21 → UAT-9.

---

*Last Updated: 2026-04-23*
