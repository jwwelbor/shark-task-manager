---
feature_key: E28-F02-reusable-maintainer-authorization-gate
epic_key: E28
document_type: test-plan
title: "Test Plan — Reusable Maintainer Authorization Gate"
---

# Test Plan — Reusable Maintainer Authorization Gate (E28-F02)

**Feature:** Reusable Maintainer Authorization Gate
**Spec:** `docs/plan/E28-entity-tagging-with-managed-vocabulary/E28-F02-reusable-maintainer-authorization-gate/spec.md`
**Epic UAT Plan:** `docs/plan/E28-entity-tagging-with-managed-vocabulary/uat-plan.md`
**Testing Architecture Rule:** `.claude/rules/testing/architecture.md`

---

## 1. AC Test Matrix

Every acceptance criterion from spec.md §1.3 is covered here. Each row maps
one AC to one or more test cases, specifying test location, test type,
inputs/setup, expected outcome, and edge cases.

---

### AC-1: Correct password with empty cache returns nil

**Requirement traces:** REQ-F-002
**Epic UAT traces:** UAT-3, UAT-4 (gate foundation)

| Field | Detail |
|---|---|
| **Test file** | `internal/auth/maintainer/gate_test.go` |
| **Test function** | `TestFileGate_Authorize_CorrectPassword_EmptyCache` |
| **Test type** | Unit — real isolated filesystem (`t.TempDir()`), injected clock |

**Setup:**
- `XDG_CACHE_HOME` set to `t.TempDir()` (env var override)
- `MaintainerConfig.PasswordHash = sha256hex("hunter2")` (`a94a8fe5...`)
- No session file exists at the expected cache path
- `NewFileGate` constructed with test temp dir as project root

**Steps:**
1. Call `gate.Authorize(ctx, "hunter2")`

**Expected outcome:** Returns `nil` (no error)

**Edge cases:**
- Password is exactly one byte (minimum non-empty)
- Password contains Unicode / special chars (`"pässwörd!"`)
- Password contains shell metacharacters (`"'; DROP TABLE"`)
- Password matches hash only when exact (case-sensitive: `"Hunter2"` fails)

---

### AC-2: Wrong password with empty cache returns *UnauthorizedError

**Requirement traces:** REQ-F-002
**Epic UAT traces:** UAT-3

| Field | Detail |
|---|---|
| **Test file** | `internal/auth/maintainer/gate_test.go` |
| **Test function** | `TestFileGate_Authorize_WrongPassword_EmptyCache` |
| **Test type** | Unit |

**Setup:** Same as AC-1 but `providedPass = "wrong"`

**Steps:**
1. Call `gate.Authorize(ctx, "wrong")`

**Expected outcome:**
- Returns non-nil error
- Error is `*UnauthorizedError` (assert with `errors.As`)
- `err.Reason` is `"wrong_password"`
- `err.Error()` is a non-empty user-facing message

**Edge cases:**
- Empty string `""` with a configured hash and no cache — must return `*UnauthorizedError` (not nil, not a different error type)
- Password that is the correct hash string itself (not the password) must fail
- Password that is the hex-encoded hash: fails (SHA-256 of the hash is not the hash)

---

### AC-3: Nil config returns *UnauthorizedError with set-password hint

**Requirement traces:** REQ-F-002, REQ-F-007
**Epic UAT traces:** UAT-3

| Field | Detail |
|---|---|
| **Test file** | `internal/auth/maintainer/gate_test.go` |
| **Test function** | `TestFileGate_Authorize_NilConfig_ReturnsHint` |
| **Test type** | Unit |

**Setup:**
- `NewFileGate(projectRoot, nil, 0)` — nil `MaintainerConfig`

**Steps:**
1. Call `gate.Authorize(ctx, "anything")`

**Expected outcome:**
- Returns `*UnauthorizedError`
- `err.Error()` contains the literal substring `"shark admin maintainer set-password"`
- `err.Error()` also references `.sharkconfig.json` OR is phrased so the user knows where to configure the password

**Edge cases:**
- `MaintainerConfig` is non-nil but `PasswordHash` is empty string — must also return `*UnauthorizedError` with the same hint (no configured password is equivalent to nil config)
- Multiple calls with nil config all return the same message (deterministic)

---

### AC-4: Cache hit within window with empty provided password returns nil

**Requirement traces:** REQ-F-003, REQ-F-005
**Epic UAT traces:** UAT-4 (sudo-style cache admits burst)

| Field | Detail |
|---|---|
| **Test file** | `internal/auth/maintainer/gate_test.go` |
| **Test function** | `TestFileGate_Authorize_CacheHit_WithinWindow` |
| **Test type** | Unit — injected fake clock |

**Setup:**
- `MaintainerConfig.PasswordHash = sha256hex("hunter2")`
- Window: 60 seconds
- Fake clock starts at `T0`
- Call `gate.Authorize(ctx, "hunter2")` → succeeds → call `gate.RecordSuccess(ctx)` → writes cache with `last_success=T0`, `pass_hash=sha256hex("hunter2")`
- Advance fake clock to `T0 + 59s`

**Steps:**
1. Call `gate.Authorize(ctx, "")` (no password, cache should cover it)

**Expected outcome:** Returns `nil`

**Edge cases:**
- Cache written with correct hash, clock at exactly `T0 + 60s` — boundary: must fail (expired, strictly greater than window — see REQ-F-005)
- Cache written at `T0`, clock at `T0 + 1ns` — should succeed (within window)
- `providedPass = ""` but a valid cache exists — succeeds (cache is sufficient)

---

### AC-5: Cache entry expired at 61s returns *UnauthorizedError

**Requirement traces:** REQ-F-005
**Epic UAT traces:** UAT-4

| Field | Detail |
|---|---|
| **Test file** | `internal/auth/maintainer/gate_test.go` |
| **Test function** | `TestFileGate_Authorize_CacheExpired` |
| **Test type** | Unit — injected fake clock |

**Setup:** Same as AC-4 up through `RecordSuccess`. Advance fake clock to `T0 + 61s`.

**Steps:**
1. Call `gate.Authorize(ctx, "")` (no password, expired cache)

**Expected outcome:**
- Returns `*UnauthorizedError`
- `err.Reason` is `"expired_cache"` (or equivalent stable enum string)

**Edge cases:**
- Exactly at `T0 + 60s` (boundary) — must also fail (window is strict `>`)
- Expired cache with correct password provided simultaneously — correct password still wins; `nil` returned (cache expiry does not block explicit auth)

---

### AC-6: Rotated password_hash invalidates existing cache

**Requirement traces:** REQ-F-002, REQ-F-003
**Epic UAT traces:** UAT-4

| Field | Detail |
|---|---|
| **Test file** | `internal/auth/maintainer/gate_test.go` |
| **Test function** | `TestFileGate_Authorize_PasswordRotation_InvalidatesCache` |
| **Test type** | Unit — injected fake clock |

**Setup:**
- Gate with `PasswordHash = sha256hex("oldpass")`, window=60s, clock=T0
- Write cache entry: `{last_success: T0, pass_hash: sha256hex("oldpass")}`
- Reconstruct gate with `PasswordHash = sha256hex("newpass")` (simulating rotation)
- Clock still at `T0 + 5s` (within window, so only rotation explains the failure)

**Steps:**
1. Call `gate.Authorize(ctx, "")` with new-config gate

**Expected outcome:**
- Returns `*UnauthorizedError`
- `err.Reason` is `"hash_mismatch_after_rotation"` (or similar)

**Edge cases:**
- Cache has `pass_hash` matching new password but `last_success` is in the future — should still work (no future-time guard required, per spec)
- Cache file references a completely different project hash — must not be read (path isolation test; see AC-7)

---

### AC-7: Cache file at correct XDG path with correct permissions

**Requirement traces:** REQ-F-004
**Epic UAT traces:** S-2 (cache file is private), S-3 (project scope)

| Field | Detail |
|---|---|
| **Test file** | `internal/auth/maintainer/gate_test.go` |
| **Test function** | `TestFileGate_CachePath_XDGOverride_Permissions` |
| **Test type** | Unit — real filesystem, `t.TempDir()`, `os.Setenv("XDG_CACHE_HOME", ...)` |

**Setup:**
- `xdgRoot := t.TempDir()`
- `t.Setenv("XDG_CACHE_HOME", xdgRoot)` (restored after test)
- Provide a `projectRoot` path (absolute; can be another `t.TempDir()`)
- Compute expected project hash: `sha256hex(projectRoot)`
- Gate constructed and `RecordSuccess` called to trigger cache creation

**Steps:**
1. Assert session file exists at: `xdgRoot/shark/<projectHash>/maintainer.session`
2. Assert session file mode bits: `0600` (`-rw-------`)
3. Assert per-project directory mode bits: `0700` (`drwx------`)

**Expected outcome:** All three assertions pass.

**Edge cases:**
- `XDG_CACHE_HOME` not set — file should land under `os.UserCacheDir()/shark/<projectHash>/maintainer.session`
- `XDG_CACHE_HOME` set to empty string `""` — same as unset (falls back to `os.UserCacheDir()`)
- Two projects with different absolute roots produce different hashes (non-collision test)
- Project root is the same path but different case (on case-sensitive FS they differ; on case-insensitive FS they must resolve to same hash) — document expected behavior

---

### AC-8: ConstantTimeCompare is used for hash comparison

**Requirement traces:** REQ-F-006, REQ-NF-002
**Epic UAT traces:** N/A (code-review / static verification)

| Field | Detail |
|---|---|
| **Test file** | `internal/auth/maintainer/gate_test.go` |
| **Test function** | `TestPackage_UsesConstantTimeCompare` |
| **Test type** | Static import assertion (parsed via `go list -json` or `go/ast` at test time) |

**Setup:**
- At test init, run `go list -json ./internal/auth/maintainer/...` and parse imports

**Steps:**
1. Assert `"crypto/subtle"` appears in the imports of the `maintainer` package
2. Optionally: source-code grep for `subtle.ConstantTimeCompare` in `gate.go`

**Expected outcome:** Assertion passes.

**Note:** This test is a compile-time / static guard. It does NOT verify the
comparison is correct by timing — that is a property of the standard library.
The test documents the design invariant for reviewers and CI.

**Edge cases:**
- `subtle` imported but not used on the comparison path (e.g., dead import) — code review must catch this; the static test only confirms the import

---

### AC-9: Malformed cache file treated as cache miss, not an error

**Requirement traces:** REQ-NF-003
**Epic UAT traces:** N/A (resilience)

| Field | Detail |
|---|---|
| **Test file** | `internal/auth/maintainer/gate_test.go` |
| **Test function** | `TestFileGate_Authorize_MalformedCacheFile_TreatedAsMiss` |
| **Test type** | Unit — real isolated filesystem |

**Setup:**
- `MaintainerConfig.PasswordHash = sha256hex("hunter2")`
- Manually write garbage bytes to the expected cache path (e.g., `"NOT JSON AT ALL"`)

**Steps:**
1. Call `gate.Authorize(ctx, "")` (no password, rely on cache)

**Expected outcome:**
- Returns `*UnauthorizedError` (not nil, not a non-UnauthorizedError error)
- No panic

**Variations (sub-tests):**
- Empty file (0 bytes)
- Valid JSON but missing expected fields: `{}`
- Valid JSON with wrong types: `{"last_success": 12345, "pass_hash": []}`
- Truncated JSON mid-write: `{"last_suc`
- File is a directory instead of a regular file

---

### AC-10: Authorize emits one span with only bool attribute, no secret leakage

**Requirement traces:** REQ-NF-001
**Epic UAT traces:** S-1 (password never appears in output)

| Field | Detail |
|---|---|
| **Test file** | `internal/auth/maintainer/gate_test.go` |
| **Test function** | `TestFileGate_Authorize_SpanAttributes_NoSecretLeakage` |
| **Test type** | Unit — in-memory OTel span recorder |

**Setup:**
- Use `go.opentelemetry.io/otel/sdk/trace` + `tracetest.NewSpanRecorder()` as test exporter
- Password used: `"super-secret-xyz"` (a string not expected anywhere else in test output)
- Hash stored in config: `sha256hex("super-secret-xyz")`

**Steps:**
1. Call `gate.Authorize(ctx, "super-secret-xyz")` (succeeds)
2. Retrieve recorded spans from the span recorder
3. Assert exactly one span with name `"maintainer.authorize"` was emitted
4. Assert the span has exactly one non-intrinsic attribute: `maintainer.authorized = true` (bool)
5. Serialize all span attributes + event attributes + event names to a string
6. Assert `"super-secret-xyz"` does NOT appear as a substring in the serialized span data
7. Assert the stored hash string does NOT appear in the serialized span data

**Expected outcome:** All assertions pass.

**Variations (sub-tests):**
- Failed authorize (wrong password): span emitted with `maintainer.authorized = false`
- `RecordSuccess` emits span named `"maintainer.record_success"` with no attributes carrying the hash

**Edge cases:**
- OTel tracer is nil / not configured — `Authorize` must not panic; it may silently skip the span

---

### AC-11: set-password writes correct hash, preserves other config fields

**Requirement traces:** REQ-F-008
**Epic UAT traces:** UAT-8 (bootstrap)

| Field | Detail |
|---|---|
| **Test file** | `internal/cli/commands/admin_maintainer_test.go` |
| **Test function** | `TestAdminMaintainerSetPassword_WritesHashPreservesOtherFields` |
| **Test type** | Unit CLI test — mocked `MaintainerBootstrapService` |

**Setup:**
- Mock bootstrap service with captured arguments
- Pre-existing config contains other top-level keys (`workflow_config`, `database`, etc.)
- Invoke command with `--password "hunter2"`

**Steps:**
1. Run `runAdminMaintainerSetPassword` with `--password "hunter2"`
2. Assert mock service called with `providedPassword = "hunter2"`
3. Assert service call returned no error
4. Assert the written config still contains all original top-level keys

**Expected outcome:** Command exits 0; mock service received `"hunter2"`.

**Note on service test:** `internal/services/maintainer_bootstrap_service_test.go`
tests the actual hash computation and config-round-trip with a mocked config
reader/writer. That test asserts:
- Written `password_hash` equals `sha256hex("hunter2")`
- All other keys in the original JSON are preserved verbatim (round-trip via `json.Marshal` / `json.Unmarshal`)

**Edge cases:**
- Config file does not exist yet — service creates it with only `maintainer` key
- Existing config has `maintainer` with different hash — hash is overwritten, `cache_window_seconds` preserved if set
- Concurrent `set-password` calls (future: not tested in v1 but documented as last-write-wins)

---

### AC-12: set-password prints no password or hash to stdout/stderr

**Requirement traces:** REQ-F-008
**Epic UAT traces:** S-1

| Field | Detail |
|---|---|
| **Test file** | `internal/cli/commands/admin_maintainer_test.go` |
| **Test function** | `TestAdminMaintainerSetPassword_NoSecretInOutput` |
| **Test type** | Unit CLI test — captured stdout/stderr |

**Setup:**
- Mock bootstrap service returns `nil`
- Capture stdout and stderr by overriding `os.Stdout` / `os.Stderr` with `bytes.Buffer` writers during test

**Steps:**
1. Run `runAdminMaintainerSetPassword` with `--password "hunter2"`
2. Assert `"hunter2"` does NOT appear in captured stdout or stderr
3. Assert `sha256hex("hunter2")` does NOT appear in captured stdout or stderr

**Expected outcome:** No plaintext or hash in any output stream.

**Edge cases:**
- Verbose mode (`--verbose`) enabled — hash must still not appear
- Error path (mock service returns error) — error message must not include the password

---

### AC-13: GetMaintainerGate() returns a functional Gate for a valid project

**Requirement traces:** REQ-F-009
**Epic UAT traces:** UAT-8

| Field | Detail |
|---|---|
| **Test file** | `internal/cli/maintainer_global_test.go` (new file) |
| **Test function** | `TestGetMaintainerGate_ReturnsWorkingGate` |
| **Test type** | Integration test — `t.TempDir()` project root, real `.sharkconfig.json` |

**Setup:**
- Create a temp dir as `projectRoot`
- Write `.sharkconfig.json` with `{"maintainer":{"password_hash":"<sha256hex of 'hunter2'>","cache_window_seconds":60}}`
- Override project root discovery to point at temp dir (or use `--config` flag equivalent in test)

**Steps:**
1. Call `GetMaintainerGate()` (or call `maintainer.NewFileGate` directly with the config)
2. Assert returned value is non-nil
3. Call `gate.Authorize(ctx, "hunter2")` — assert `nil`
4. Call `gate.Authorize(ctx, "wrong")` — assert `*UnauthorizedError`

**Expected outcome:** Gate is functional end-to-end through config loading.

**Edge cases:**
- Config missing `maintainer` field — `GetMaintainerGate()` returns a gate that always returns `*UnauthorizedError` (no panic)
- Project root not found — accessor panics per the lazy-init pattern (consistent with other `Get*Service()` accessors)

---

### AC-14: internal/auth/maintainer has no shark-domain imports

**Requirement traces:** REQ-F-001, REQ-F-010
**Epic UAT traces:** UAT-8

| Field | Detail |
|---|---|
| **Test file** | `internal/auth/maintainer/gate_test.go` (or a dedicated `imports_test.go`) |
| **Test function** | `TestPackage_HasNoSharkDomainImports` |
| **Test type** | Static — `go list -json` at test time |

**Setup:**
- Forbidden import prefixes:
  - `github.com/jwwelbor/shark-task-manager/internal/models`
  - `github.com/jwwelbor/shark-task-manager/internal/repository`
  - `github.com/jwwelbor/shark-task-manager/internal/services`
  - `github.com/jwwelbor/shark-task-manager/internal/workflow`
  - `github.com/jwwelbor/shark-task-manager/internal/cli`

**Steps:**
1. Execute `go list -json ./internal/auth/maintainer/...`
2. Parse the `Imports` and `TestImports` fields
3. Assert none of the above forbidden prefixes appear

**Expected outcome:** No forbidden imports found.

**Edge cases:**
- `repoutil` import: allowed (leaf observability package; see spec §2.6 caveat). If `repoutil` transitively pulls in a forbidden package, the test catches it.
- Future addition of a forbidden import: test fails, blocking the PR

---

## 2. Additional Unit Tests (Beyond AC Coverage)

These tests are specified in spec.md §2.7 but do not map to a single AC.
They are mandatory for the test suite.

### 2.1 Concurrent RecordSuccess — Race Safety

| Field | Detail |
|---|---|
| **Test file** | `internal/auth/maintainer/gate_test.go` |
| **Test function** | `TestFileGate_RecordSuccess_Concurrent` |
| **Test type** | Unit — real isolated filesystem, `t.Parallel()` goroutines |
| **Go test flag** | Run with `-race` in CI |

**Setup:** 8 goroutines each call `gate.RecordSuccess(ctx)` concurrently. Fake clock (static `T0`).

**Expected outcome:**
- No data race detected by `-race`
- Final session file is valid JSON
- `last_success` in final file is a plausible timestamp
- No file is left as a `.tmp` file (atomic rename committed)

### 2.2 XDG Precedence Over UserCacheDir

| Field | Detail |
|---|---|
| **Test file** | `internal/auth/maintainer/gate_test.go` |
| **Test function** | `TestCachePath_XDGOverrides_HomeCache` |

**Setup:** `XDG_CACHE_HOME` set to an explicit temp dir.

**Expected outcome:** Session file path starts with `XDG_CACHE_HOME` value.

### 2.3 Project Hash Isolation (Different Projects Don't Share Cache)

| Field | Detail |
|---|---|
| **Test file** | `internal/auth/maintainer/gate_test.go` |
| **Test function** | `TestCachePath_DifferentProjects_DifferentHashes` |

**Setup:** Two `t.TempDir()` paths used as project roots.

**Expected outcome:** SHA-256 of path A != SHA-256 of path B; session file paths are distinct.

### 2.4 CacheWindow Default — Zero Means 60s

| Field | Detail |
|---|---|
| **Test file** | `internal/config/maintainer_test.go` |
| **Test function** | `TestMaintainerConfig_CacheWindow_ZeroDefault` |

**Setup:** `MaintainerConfig{CacheWindowSeconds: 0}`

**Expected outcome:** `cfg.CacheWindow()` returns `60 * time.Second`.

### 2.5 Config Round-Trip — Maintainer Field Preserved

| Field | Detail |
|---|---|
| **Test file** | `internal/config/config_test.go` (extension of existing file) |
| **Test function** | `TestConfig_Maintainer_RoundTrip` |

**Setup:** Marshal `Config{Maintainer: &MaintainerConfig{PasswordHash: "abc", CacheWindowSeconds: 120}}` to JSON and back.

**Expected outcome:** All fields survive the round-trip unchanged; other fields like `database` are unaffected.

---

## 3. Integration Scenarios

These cross-component scenarios verify that the pieces assemble correctly.
Each maps to one or more epic UAT scenarios.

### INT-1: Bootstrap → Authorize → RecordSuccess → Subsequent Authorize (full flow)

**Components involved:** `MaintainerBootstrapService` → `internal/config` → `FileGate`

**Maps to epic UAT:** UAT-4 (sudo-style cache burst), UAT-8 (reusable gate)

**Flow:**
1. Bootstrap service writes `sha256hex("mypass")` to config
2. `FileGate` constructed from config
3. `gate.Authorize(ctx, "mypass")` → nil (correct password)
4. `gate.RecordSuccess(ctx)` → nil (cache written)
5. `gate.Authorize(ctx, "")` within window → nil (cache hit)
6. Advance clock past window
7. `gate.Authorize(ctx, "")` → `*UnauthorizedError` (expired)

**Boundary to verify:**
- Config struct flows correctly into `FileGate` without loss (no default-value erasure on nil pointer)
- `RecordSuccess` actually persists a file that `Authorize` reads back correctly (not in-memory state)

### INT-2: GetMaintainerGate() Accessor → Gate Contract (CLI integration)

**Components involved:** `internal/cli/maintainer_global.go` → `internal/config` → `internal/auth/maintainer`

**Maps to epic UAT:** UAT-8 (AC-13 already covers the unit case; this is the integration assertion)

**Flow:**
1. Write a valid `.sharkconfig.json` in a temp project root
2. Call `cli.GetMaintainerGate()` using the real project root
3. Assert the returned `Gate` satisfies the `Authorize` / `RecordSuccess` contract

**What to verify at the boundary:**
- `GetMaintainerGate()` does not panic when `cfg.Maintainer` is nil
- `GetMaintainerGate()` creates a new instance each call (not shared state)
- The accessor follows the same panic-on-config-failure behavior as `GetTaskService()`

### INT-3: set-password Command → Config File → FileGate (CLI → config → gate)

**Components involved:** `admin_maintainer.go` → `MaintainerBootstrapService` → config file → `FileGate`

**Maps to epic UAT:** UAT-8 (bootstrap ergonomic), S-1 (no secret leakage)

**Flow:**
1. Start with a `.sharkconfig.json` containing only `{"workflow_config": "..."}`
2. Run `set-password --password "s3cr3t"` against the config
3. Read config back and parse `maintainer.password_hash`
4. Assert `password_hash == sha256hex("s3cr3t")`
5. Assert `workflow_config` key is unchanged (round-trip preservation)
6. Construct a `FileGate` from the updated config
7. Assert `gate.Authorize(ctx, "s3cr3t")` returns nil

**What to verify at the boundary:**
- The bootstrap service and config writer do not corrupt unrelated config keys (the `RawData` preservation path)
- A `FileGate` constructed from the just-written config correctly verifies the password set by the command

### INT-4: Future-Consumer Pattern Compiles and Works (REQ-F-010)

**Components involved:** `internal/auth/maintainer.Gate` interface

**Maps to epic UAT:** UAT-8

**Test type:** Compile-time + minimal smoke test

**What to verify:**
- The ten-line adoption pattern from `doc.go` / spec §1.1 REQ-F-010 is valid Go:
  ```go
  gate := cli.GetMaintainerGate()
  if err := gate.Authorize(ctx, pass); err != nil { return err }
  defer func() { _ = gate.RecordSuccess(ctx) }()
  ```
- This pattern can be instantiated in a test binary without type assertions
- `gate.Authorize` and `gate.RecordSuccess` are on the `Gate` interface (not only on `*FileGate`)

**Implementation note:** A minimal `TestFutureConsumerPattern_Compiles` test in
`internal/auth/maintainer/gate_test.go` that creates a `FileGate` via
`NewFileGate`, assigns it to a `Gate` variable, and calls both methods verifies
the interface at compile time. If it compiles, the pattern is valid.

---

## 4. Test Infrastructure

### 4.1 Existing Infrastructure to Reuse

| Pattern | Location | How F02 Uses It |
|---|---|---|
| `t.TempDir()` isolation | Throughout codebase | Isolate `XDG_CACHE_HOME` and project root per test; no cross-test pollution |
| Function-field mocks | `internal/services/task_service_test.go` | Bootstrap service mock follows the same `func` field pattern |
| Config JSON round-trip tests | `internal/config/config_test.go` | Extend existing `TestDatabaseConfig_Marshaling` style to cover `Maintainer` |
| CLI captured output | `internal/cli/commands/*_test.go` | Capture stdout/stderr for AC-12 |
| `ResetServices()` for CLI global state | `internal/cli/services_global_test.go` | Call after any test that sets `GetMaintainerGate` to clean up |
| `t.Setenv` for environment overrides | Go `testing` package | Override `XDG_CACHE_HOME` safely in AC-7 tests |
| OTel `tracetest` span recorder | `go.opentelemetry.io/otel/sdk/trace/tracetest` | In-memory span recording for AC-10 |

### 4.2 New Test Helpers Needed

| Helper | Purpose | Location |
|---|---|---|
| `sha256hex(s string) string` | Compute expected hash for assertions | Inline in `gate_test.go` (5 lines) |
| `fakeClock` struct implementing `clock` interface | Inject settable time for AC-4, AC-5, AC-6 | Inline in `gate_test.go` |
| `writeFakeSessionFile(t, path, content string)` | Write malformed cache for AC-9 | Inline helper in `gate_test.go` |
| `newTestGate(t, opts) *FileGate` | Construct gate with temp dirs and fake clock in one call | Inline helper in `gate_test.go` |

**No new shared test packages are needed.** All helpers are local to their test file.

### 4.3 Test File Inventory

The complete set of test files for F02:

| File | Type | Tests |
|---|---|---|
| `internal/auth/maintainer/gate_test.go` | Unit + static | AC-1..10, AC-14, §2.1..2.3, INT-4 |
| `internal/config/maintainer_test.go` | Unit | §2.4 |
| `internal/config/config_test.go` (extension) | Unit | §2.5 |
| `internal/services/maintainer_bootstrap_service_test.go` | Unit + mocks | AC-11 (service half) |
| `internal/cli/commands/admin_maintainer_test.go` | Unit CLI | AC-11 (CLI half), AC-12 |
| `internal/cli/maintainer_global_test.go` | Integration | AC-13, INT-2 |

### 4.4 Test Execution

```bash
# Run all maintainer gate tests
go test -v -race ./internal/auth/maintainer/...

# Run config round-trip tests (includes new Maintainer tests)
go test -v ./internal/config/...

# Run service tests
go test -v ./internal/services/ -run TestMaintainerBootstrap

# Run CLI command tests
go test -v ./internal/cli/commands/ -run TestAdminMaintainer

# Run CLI accessor integration test
go test -v ./internal/cli/ -run TestGetMaintainerGate

# Full quality gate (mandatory before completion)
make fmt && make lint && make test
```

---

## 5. Quality Gates

The following must pass before F02 is marked complete:

| Gate | Criterion |
|---|---|
| AC coverage | All 14 ACs have at least one passing test |
| Race detector | `go test -race ./internal/auth/maintainer/...` is clean |
| Import firewall | AC-14 static test passes |
| Secret leakage | AC-10 and AC-12 tests pass |
| Round-trip | Config marshal/unmarshal test (§2.5) passes |
| Full test suite | `make fmt && make lint && make test` is green |
| Code review | UAT-8 (reusability) confirmed by reviewer per epic UAT plan |

---

## 6. Traceability Summary

| Spec AC | Tests | Epic UAT | Epic SC |
|---|---|---|---|
| AC-1 | `TestFileGate_Authorize_CorrectPassword_EmptyCache` | UAT-3, UAT-4 | SC-3, SC-4 |
| AC-2 | `TestFileGate_Authorize_WrongPassword_EmptyCache` | UAT-3 | SC-3 |
| AC-3 | `TestFileGate_Authorize_NilConfig_ReturnsHint` | UAT-3 | SC-3 |
| AC-4 | `TestFileGate_Authorize_CacheHit_WithinWindow` | UAT-4 | SC-4 |
| AC-5 | `TestFileGate_Authorize_CacheExpired` | UAT-4 | SC-4 |
| AC-6 | `TestFileGate_Authorize_PasswordRotation_InvalidatesCache` | UAT-4 | SC-4 |
| AC-7 | `TestFileGate_CachePath_XDGOverride_Permissions` | S-2, S-3 | SC-4 |
| AC-8 | `TestPackage_UsesConstantTimeCompare` | — | SC-3 (timing) |
| AC-9 | `TestFileGate_Authorize_MalformedCacheFile_TreatedAsMiss` | — | SC-4 (robustness) |
| AC-10 | `TestFileGate_Authorize_SpanAttributes_NoSecretLeakage` | S-1 | SC-9 (obs.) |
| AC-11 | `TestAdminMaintainerSetPassword_WritesHashPreservesOtherFields` (CLI + service) | UAT-8 | SC-9 |
| AC-12 | `TestAdminMaintainerSetPassword_NoSecretInOutput` | S-1 | SC-9 |
| AC-13 | `TestGetMaintainerGate_ReturnsWorkingGate` | UAT-8 | SC-9 |
| AC-14 | `TestPackage_HasNoSharkDomainImports` | UAT-8 | SC-9 |

All 14 ACs are covered. No orphaned tests exist — every test traces to a spec AC.

---

*Last Updated: 2026-04-23*
