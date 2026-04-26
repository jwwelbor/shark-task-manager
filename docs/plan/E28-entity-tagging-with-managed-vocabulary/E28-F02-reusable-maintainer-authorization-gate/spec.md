---
feature_key: E28-F02-reusable-maintainer-authorization-gate
epic_key: E28
document_type: spec
title: "Spec — Reusable Maintainer Authorization Gate"
---

# Spec — Reusable Maintainer Authorization Gate

Combined requirements + architecture specification for feature **E28-F02**.

**References (not restated here):**
- Epic PRD: `docs/plan/E28-entity-tagging-with-managed-vocabulary/epic.md` (§2 SC-3, SC-4, SC-9, §3 scope, §4 open questions O-2, O-3, §6 UAT-3, UAT-4, UAT-8)
- Epic architecture: `docs/plan/E28-entity-tagging-with-managed-vocabulary/architecture.md` (ADR-2 reusable gate, ADR-3 cache location, ADR-6 SHA-256 hashing, §1.3 service layering, §4.6 future-consumer contract, §6 observability rules)
- Feature description: `docs/plan/E28-entity-tagging-with-managed-vocabulary/E28-F02-reusable-maintainer-authorization-gate/feature.md`

**Scope of this feature.** Deliver the reusable, tag-agnostic authorization
package `internal/auth/maintainer` (`Gate` interface, SHA-256 comparison,
file-backed sudo-style cache), the corresponding `.sharkconfig.json`
`maintainer` object + `internal/config` wiring, the bootstrap command
`shark admin maintainer set-password`, and the CLI accessor
`cli.GetMaintainerGate()`. F03 is the first consumer; F02 ships with the
gate fully tested but with **no tag-vocabulary commands**.

---

## 1. Requirements

Requirements in this section are **incremental** over the epic PRD. They
specialize epic-level success criteria (SC-3, SC-4, SC-9) and the
acceptance scenarios in UAT-3, UAT-4, UAT-8 into testable deliverables
for this feature.

### 1.1 Functional Requirements

**REQ-F-001 — Package location and public surface.**
The gate MUST live in a new package at `internal/auth/maintainer`. The
package MUST expose the `Gate` interface and a concrete `FileGate`
implementation. The package MUST NOT import any shark-domain package
(no `internal/models`, no `internal/repository`, no `internal/services`).
Traces to epic ADR-2 and SC-9.

**REQ-F-002 — `Gate.Authorize` contract.**
`Authorize(ctx context.Context, providedPass string) error` MUST return
`nil` iff either (a) `providedPass` is non-empty and its SHA-256 hex
digest equals the configured `password_hash`, or (b) a live cache entry
exists in the session file and its stored hash equals the configured
`password_hash` and its `last_success` timestamp is within the
configured cache window. Otherwise it MUST return a typed
`*UnauthorizedError` whose `Error()` string tells the user how to
configure/obtain the maintainer password (including the path to
`.sharkconfig.json` and the exact `shark admin maintainer set-password`
command). Traces to epic SC-3 and UAT-3.

**REQ-F-003 — `Gate.RecordSuccess` contract.**
`RecordSuccess(ctx context.Context) error` MUST write a cache entry with
`last_success = time.Now().UTC()` and `pass_hash` equal to the currently
configured `password_hash`. The write MUST be a full-file replace
(create-temp + atomic rename), not an append. Callers MAY ignore the
returned error ("best-effort cache" — see ADR-3 §cache entry format).
Traces to epic SC-4 and UAT-4.

**REQ-F-004 — Cache location and permissions.**
The cache file MUST be located at
`<cache-root>/shark/<project-hash>/maintainer.session`, where
`<cache-root>` is `$XDG_CACHE_HOME` if set and non-empty, otherwise the
value returned by `os.UserCacheDir()`. The per-project directory MUST be
created with mode `0700`; the session file MUST be created with mode
`0600`. `<project-hash>` MUST be a stable lowercase-hex SHA-256 of the
absolute project root path. Traces to epic ADR-3 and open question O-3.

**REQ-F-005 — Cache window.**
The `FileGate` constructor MUST accept a `Window time.Duration`. Callers
MAY pass `0` to request the default of `60 * time.Second`. When the
wall-clock difference between `time.Now().UTC()` and the cache entry's
`last_success` is strictly greater than `Window`, the cache entry MUST
be treated as expired and MUST NOT satisfy `Authorize`. Traces to epic
ADR-3 and SC-4.

**REQ-F-006 — Password hashing.**
Password hashing MUST be SHA-256 (`crypto/sha256`) producing a
lowercase hex digest with no salt, no iteration count, no prefix.
Comparison MUST use `crypto/subtle.ConstantTimeCompare` to avoid
timing-based oracle attacks. Traces to epic ADR-6.

**REQ-F-007 — Config schema additions.**
`internal/config.Config` MUST gain a `Maintainer *MaintainerConfig`
field that marshals to `.sharkconfig.json` as a `maintainer` object
with two fields: `password_hash` (lowercase-hex string, omit-empty) and
`cache_window_seconds` (int, omit-empty). An unset / nil
`Maintainer` field MUST be equivalent to "no password configured" and
MUST cause `Authorize` to fail with a typed `*UnauthorizedError` whose
message instructs the user to run `shark admin maintainer set-password`.
Traces to epic §3.3 (config schema additions).

**REQ-F-008 — Bootstrap command `shark admin maintainer set-password`.**
A new command MUST exist at `shark admin maintainer set-password`. It
MUST accept a plaintext password from one of three sources, in
precedence order: (a) `--password <value>` flag, (b) interactive stdin
prompt with echo disabled (only when stdin is a terminal), (c)
`--password-stdin` flag reading exactly one line from stdin. The command
MUST compute the SHA-256 hex digest and write the result into the
`.sharkconfig.json` `maintainer.password_hash` field, preserving all
other config fields. On success the command MUST print neither the
plaintext password nor the stored hash to stdout/stderr. Traces to epic
ADR-6 "bootstrap ergonomic".

**REQ-F-009 — CLI accessor.**
`internal/cli/services_global.go` (or a companion file in the same
package) MUST expose `GetMaintainerGate() maintainer.Gate` that returns
a configured `FileGate` bound to the current project root and loaded
`Config.Maintainer`. The accessor MUST follow the existing lazy-init
pattern (one instance per CLI invocation; panics on DB/config failure
to match `GetTaskService`).

**REQ-F-010 — Future-consumer contract.**
Any future consumer (e.g., hypothetical `shark admin purge`) MUST be
able to gate itself with exactly this shape:
```go
gate := cli.GetMaintainerGate()
if err := gate.Authorize(ctx, providedPass); err != nil { return err }
defer func() { _ = gate.RecordSuccess(ctx) }()
```
No type assertion, no dependency on tags, no config re-read required at
the call site. Traces to epic §4.6 and SC-9 / UAT-8.

### 1.2 Non-Functional Requirements

**REQ-NF-001 — Observability (no secret leakage).**
`FileGate.Authorize` MUST emit exactly one OpenTelemetry span named
`maintainer.authorize` with a single outcome attribute
`maintainer.authorized` (bool). `FileGate.RecordSuccess` MUST emit a
span named `maintainer.record_success`. Spans MUST NOT include
attributes, events, or log statements that carry: the provided
password, the stored `password_hash`, the computed digest of the
provided password, or any prefix/suffix of any of these. Traces to
epic §6 (observability rules) and ADR-6.

**REQ-NF-002 — Timing-side-channel resistance.**
Hash comparison MUST use `crypto/subtle.ConstantTimeCompare`. Early
returns keyed off "password is empty" are permitted because that branch
does not leak the configured hash. Returning `*UnauthorizedError` from
the "missing config" branch and from the "wrong password" branch with
identical messages to the user is OPTIONAL; the threat model does not
require it.

**REQ-NF-003 — Concurrency safety.**
Two concurrent `RecordSuccess` calls against the same cache file MUST
NOT corrupt the file. The implementation MAY use last-write-wins (per
ADR-3's concurrency note). A concurrent `Authorize` reading a
half-written cache file MUST be tolerated: partial / malformed JSON in
the cache file MUST be treated as "no cache entry" (cache miss, no
error surfaced above the package).

**REQ-NF-004 — Stateless per invocation.**
The gate MUST NOT rely on any in-process state surviving across CLI
invocations. All session state is externalized to the cache file.
Traces to epic constraint #7 (CLI is stateless per invocation).

**REQ-NF-005 — Dual-backend neutral.**
The gate MUST NOT touch the database. It is backend-agnostic by
construction (pure filesystem + config). Traces to epic constraint #5.

### 1.3 Acceptance Criteria (testable)

Each criterion is phrased so a test can pass or fail it unambiguously.
Each cites the requirement(s) it verifies.

| # | Acceptance Criterion | Verification |
|---|---|---|
| AC-1 | With `Maintainer.PasswordHash = sha256("hunter2")` configured and an empty cache, `Authorize(ctx, "hunter2")` returns `nil`. | Unit test on `FileGate`. Verifies REQ-F-002. |
| AC-2 | With the same config and an empty cache, `Authorize(ctx, "wrong")` returns `*UnauthorizedError`. | Unit test. Verifies REQ-F-002. |
| AC-3 | With no `Maintainer` in config (nil), `Authorize(ctx, "anything")` returns `*UnauthorizedError` whose message contains the literal substring `shark admin maintainer set-password`. | Unit test. Verifies REQ-F-002, REQ-F-007. |
| AC-4 | After a successful `Authorize` followed by `RecordSuccess` with a 60s window, a subsequent `Authorize(ctx, "")` (no password) within 60 simulated seconds returns `nil`. | Unit test using a clock injected into `FileGate`. Verifies REQ-F-003, REQ-F-005. |
| AC-5 | With the same setup as AC-4, a subsequent `Authorize(ctx, "")` at `61 * time.Second` simulated returns `*UnauthorizedError`. | Unit test using injected clock. Verifies REQ-F-005. |
| AC-6 | After `password_hash` in config changes to a new value, a cache entry containing the OLD hash MUST NOT satisfy `Authorize`. | Unit test: write cache with old hash, change config, call `Authorize(ctx, "")` — expect `*UnauthorizedError`. Verifies REQ-F-002, REQ-F-003. |
| AC-7 | The cache file is created at `<XDG_CACHE_HOME-or-UserCacheDir>/shark/<sha256(project-root)>/maintainer.session` with mode `0600`; the per-project directory has mode `0700`. | Filesystem assertion in a repository-style test that sets `XDG_CACHE_HOME` to a `t.TempDir()`. Verifies REQ-F-004. |
| AC-8 | Hash comparison is implemented via `crypto/subtle.ConstantTimeCompare`. | `go vet` / static inspection + code review; MAY be asserted via a package-internal test that checks that `subtle` is imported by the authorize path. Verifies REQ-F-006, REQ-NF-002. |
| AC-9 | A malformed (half-written, truncated JSON, or non-JSON) cache file does NOT cause `Authorize` to return an error other than `*UnauthorizedError` when the caller passes an empty password. | Unit test that writes garbage bytes into the cache file, then calls `Authorize(ctx, "")`. Verifies REQ-NF-003. |
| AC-10 | `Authorize` emits exactly one span whose only non-intrinsic attribute is `maintainer.authorized` (bool). No attribute or event carries the password, hash, or any prefix. | Test using an in-memory OTel span recorder. Verifies REQ-NF-001. |
| AC-11 | `shark admin maintainer set-password --password "hunter2"` writes `password_hash` equal to `sha256("hunter2")` into `.sharkconfig.json` and leaves every other top-level key unchanged. | CLI test with mocked config writer (round-trips JSON). Verifies REQ-F-008. |
| AC-12 | `shark admin maintainer set-password --password "hunter2"` prints no occurrence of `"hunter2"` or the resulting hash on stdout or stderr. | CLI test asserting captured output. Verifies REQ-F-008. |
| AC-13 | `cli.GetMaintainerGate()` returns a non-nil `maintainer.Gate` whose `Authorize` works as specified above when the project has a valid `.sharkconfig.json` with a populated `Maintainer`. | Integration test that runs against a `t.TempDir()` project root. Verifies REQ-F-009. |
| AC-14 | The package `internal/auth/maintainer` has no import statement mentioning `internal/models`, `internal/repository`, `internal/services`, `internal/workflow`, or `internal/cli`. | Static test using `go list -f '{{.Imports}}' ./internal/auth/maintainer/...`. Verifies REQ-F-001, REQ-F-010. |

### 1.4 Out of Scope (for this feature)

Explicitly **deferred** to their own features or future epics; not
rejected forever, just not F02's responsibility:

- `shark tags list / add / rm / rename` commands and `TagService`
  integration — F03 (the first consumer).
- `--pass` flag plumbing on `tags` subcommands — F03.
- Argon2 / bcrypt / scrypt / salted hashing — future epic only if the
  threat model tightens (per epic ADR-6).
- Secret-manager / keychain integration for the password — future epic.
- Web viewer authentication — E27 / future epic; viewer stays read-only
  (epic §3 scope).
- Multi-maintainer roles, per-action permissions — future epic.
- Audit log of `Authorize` attempts (the OTel span is the current
  audit surface).

---

## 2. Architecture

This section details the design for F02. It adheres to CLAUDE.md:
service layering (CLI thin wrapper → service → repository → DB), input
sanitization, observability, testing discipline. Where the epic
architecture has already decided a point, this section cites it by ADR
number instead of restating the rationale.

### 2.1 Component Inventory (files created / modified)

**Created:**

- `internal/auth/maintainer/gate.go` — defines `Gate` interface and
  `FileGate` struct; implements `Authorize`, `RecordSuccess`.
- `internal/auth/maintainer/errors.go` — defines `*UnauthorizedError`
  (typed, exported; exposes `.UserHint()` method used by CLI to shape
  the final stderr message).
- `internal/auth/maintainer/cache.go` — defines `sessionEntry` struct
  (`LastSuccess time.Time`, `PassHash string`), JSON read/write helpers,
  project-hash computation.
- `internal/auth/maintainer/clock.go` — defines a `clock` interface
  (`Now() time.Time`) and a default `realClock` implementation; used
  for injecting time in tests.
- `internal/auth/maintainer/gate_test.go` — unit tests covering AC-1..10,
  using an in-memory clock and `t.TempDir()` for XDG root.
- `internal/auth/maintainer/doc.go` — package-level godoc stating the
  future-consumer contract (REQ-F-010 ten-line example).
- `internal/config/maintainer.go` — `MaintainerConfig` struct, default
  helpers (`cacheWindow()` returning `60s` when zero).
- `internal/cli/maintainer_global.go` — `GetMaintainerGate()` accessor;
  follows the atomic-container lazy-init pattern already used in
  `services_global.go`.
- `internal/cli/commands/admin_maintainer.go` — `shark admin maintainer`
  parent command + `set-password` subcommand.
- `internal/cli/commands/admin_maintainer_test.go` — CLI test verifying
  AC-11, AC-12.
- `internal/services/maintainer_bootstrap_service.go` — thin service
  that orchestrates "read config → compute hash → write config" for the
  bootstrap command (keeps the CLI command a thin wrapper per
  `cli-integration.md`).
- `internal/services/maintainer_bootstrap_service_test.go` — service
  test with mocked config reader/writer.

**Modified:**

- `internal/config/config.go` — add one field:
  `Maintainer *MaintainerConfig \`json:"maintainer,omitempty"\``.
  Register the field in the JSON-unknown-fields preservation path
  (the existing `RawData` map already handles this, but explicit tests
  are added to confirm round-trip).
- `internal/config/config_test.go` (existing) — add round-trip tests
  that serialize a `Config` with a populated `Maintainer` and parse it
  back; existing tests unchanged.
- `internal/cli/commands/admin.go` — register the `adminMaintainerCmd`
  as a child of `adminCmd` in its `init()` (one-line addition). Long
  help text updated to list `maintainer` in the subcommand list.
- `docs/cli-reference/setup-commands.md` — new section:
  "`shark admin maintainer set-password` — Set the maintainer password."
- `docs/cli-reference/configuration.md` — new section: "`maintainer`
  object: `password_hash`, `cache_window_seconds`." Includes the
  "plaintext is never stored" security note.

**Intentionally unchanged:**

- `internal/db/db.go` — no schema changes in F02 (tag tables land in
  F01, already merged). The gate does not touch the database.
- `internal/services/tag_service.go` — does not exist yet; lands in F03.
- `internal/repository/tag/` — untouched; already built in F01.

### 2.2 Package Shape

```
internal/auth/maintainer/
├── doc.go           // package-level doc + consumer example
├── gate.go          // Gate interface + FileGate
├── errors.go        // *UnauthorizedError
├── cache.go         // sessionEntry + JSON I/O + project hash
├── clock.go         // clock interface (for test injection)
└── gate_test.go     // unit tests
```

**Public API (normative — this is the F02 deliverable that F03
consumes):**

```go
package maintainer

// Gate is the authorization primitive. Future admin commands consume
// this interface. See doc.go for the ten-line adoption example.
type Gate interface {
    // Authorize verifies either an explicit password or a live cache
    // entry. Returns nil on success or *UnauthorizedError on failure.
    // Never returns an error other than *UnauthorizedError for
    // user-visible failures; infrastructure failures (disk I/O on the
    // cache file) are swallowed into "cache miss" per REQ-NF-003.
    Authorize(ctx context.Context, providedPass string) error

    // RecordSuccess extends the sudo-style window. Callers typically
    // invoke it after a gated operation completes. Returns an error
    // only on hard I/O failures the caller MAY choose to log; calling
    // contexts SHOULD treat the return as best-effort.
    RecordSuccess(ctx context.Context) error
}

// UnauthorizedError is the only user-visible error this package
// returns for Authorize failures.
type UnauthorizedError struct {
    // Reason is a stable enum-ish string for programmatic handling:
    // "missing_config", "wrong_password", "expired_cache",
    // "hash_mismatch_after_rotation".
    Reason string
}

func (e *UnauthorizedError) Error() string { /* composes a user-friendly message */ }

// FileGate is the filesystem-backed implementation.
type FileGate struct {
    projectRoot string
    cfg         *config.MaintainerConfig
    window      time.Duration
    clock       clock
}

// NewFileGate constructs a FileGate. A zero window is replaced with the
// default (60s).
func NewFileGate(projectRoot string, cfg *config.MaintainerConfig, window time.Duration) *FileGate
```

### 2.3 Data Model

F02 adds **zero** new database tables. The "data model" is:

**On disk (per project):**
```
$XDG_CACHE_HOME/shark/<project-hash>/maintainer.session
  (or $HOME/.cache/shark/<project-hash>/maintainer.session on Linux when
  XDG_CACHE_HOME is unset, per os.UserCacheDir fallback)
```

**File format (JSON):**
```json
{
  "last_success": "2026-04-23T15:04:05Z",
  "pass_hash": "a1b2c3..."
}
```

Unrecognized keys MUST be ignored on read (forward-compatible).

**In `.sharkconfig.json`:**
```json
{
  "maintainer": {
    "password_hash": "a1b2c3...",
    "cache_window_seconds": 60
  }
}
```

**In-memory (`internal/config/maintainer.go`):**
```go
type MaintainerConfig struct {
    PasswordHash       string `json:"password_hash,omitempty"`
    CacheWindowSeconds int    `json:"cache_window_seconds,omitempty"`
}

// CacheWindow returns the configured window with a 60s default.
func (m *MaintainerConfig) CacheWindow() time.Duration {
    if m == nil || m.CacheWindowSeconds <= 0 {
        return 60 * time.Second
    }
    return time.Duration(m.CacheWindowSeconds) * time.Second
}
```

### 2.4 Interface Contracts

**CLI command surface.**
```
shark admin maintainer set-password [--password <plaintext>] [--password-stdin]
```
Returns exit 0 on success, exit 1 on any parse/validation error, exit 2
on config-write failure. No `--force` flag in v1 (overwriting an
existing hash is the expected behavior).

**Gate interaction pattern for `shark admin maintainer set-password`:**
The set-password command itself is **not gated**. Rationale: the gate
cannot exist before the first password is set. This is the chicken-and-
egg bootstrap; it is consistent with how most unix `*-passwd` tools
behave when the target account has no password. Once a password exists,
rotating it is ALSO ungated in v1 (tracked as an open item for a future
epic to gate rotation by requiring the *current* password).

**Future-consumer pattern (REQ-F-010, normative example for doc.go):**
```go
func runAdminPurge(cmd *cobra.Command, args []string) error {
    pass, _ := cmd.Flags().GetString("pass")
    gate := cli.GetMaintainerGate()
    if err := gate.Authorize(cmd.Context(), pass); err != nil {
        return err // *UnauthorizedError renders correctly via cli.Error
    }
    defer func() { _ = gate.RecordSuccess(cmd.Context()) }()
    // ... destructive op ...
    return nil
}
```

### 2.5 Key Technical Decisions (F02-specific; defers to epic ADRs)

**F02-D1 — Clock injection via private `clock` interface.**
The `FileGate` holds an unexported `clock` field. `NewFileGate` binds a
`realClock{}`; tests use a `fakeClock` with a settable `now` field.
Rationale: testing cache expiry without `time.Sleep` (AC-5) is required;
exposing a public `WithClock` method is unnecessary surface area for
external callers.

**F02-D2 — Atomic cache write via temp-file + rename.**
`RecordSuccess` writes to
`<dir>/maintainer.session.<pid>-<nanotime>.tmp` then `os.Rename`s onto
the final path. This guarantees `Authorize` never observes a partially
written file (addresses REQ-NF-003's "tolerance" as a strengthening).
Rationale: `os.Rename` on POSIX is atomic within the same directory and
is the standard pattern used by the `fileops` package (see
`internal/fileops/writer.go`) — reuses an established project idiom.

**F02-D3 — Hash the `last_success` and `pass_hash` together? No.**
We considered HMAC-signing the cache entry so an attacker with write
access to the cache file cannot forge a "just authorized" entry. We
**rejected** this: the threat model in epic §4 (assumption 3) explicitly
excludes filesystem adversaries. An attacker who can write the cache
file can equally write `.sharkconfig.json`. Adding HMAC would introduce
key-management complexity with no defense against an in-scope threat.

**F02-D4 — Bootstrap service over direct file edit in the CLI command.**
Per `.claude/rules/services/cli-integration.md`, the
`shark admin maintainer set-password` command MUST remain a thin
wrapper. The "read config → compute hash → rewrite config" orchestration
lives in `internal/services/maintainer_bootstrap_service.go` even
though it only has one method. Rationale: keeps the test shape uniform
(service tests mock I/O; CLI tests mock the service); follows E15
service-layer goal; makes future enhancements (e.g., "if password
unchanged, skip write") easy to add without refactoring the command.

**F02-D5 — CacheWindow configurable via config, not via CLI flag.**
Rationale: the window is a project-wide policy, not a per-invocation
choice. Per-invocation override would create scripting footguns
("I set `--window=3600` and forgot"). Tests pass a window via
`NewFileGate` directly, not via `FileGate` mutator methods.

**F02-D6 — `project-hash` is SHA-256 of the absolute path.**
An alternative is `filepath.Base(projectRoot)`, which is shorter but
collides when two projects share the same basename
(`/home/a/shark-task-manager` vs `/home/b/shark-task-manager`). SHA-256
collision resistance is more than adequate; the hex digest is long but
never surfaced to users.

**F02-D7 — Set-password is not gated. Rotation is also not gated in v1.**
Recorded under §2.4; the trade-off is that an agent with write access
to `.sharkconfig.json` can set a password and then authorize. This is
consistent with the "no filesystem adversary" threat model. A follow-up
epic can add "current-password-required rotation" without changing the
`Gate` interface.

### 2.6 Integration With Existing Code

**Config loading.** `internal/cli/root.go`'s existing `GetConfig()`
(line 360) already unmarshals the full `.sharkconfig.json` into
`config.Config` and preserves unknown fields via `RawData`. Adding the
`Maintainer` field is a one-line struct extension; round-trip is
preserved because `RawData` is only used when we re-serialize to
compose unknown-fields, and the bootstrap service uses a full
`Config` struct, not `RawData`, for its writes.

**CLI accessor integration.** `GetMaintainerGate()` in the new file
`internal/cli/maintainer_global.go` follows the exact lazy-init shape
of `GetTaskService()`:
```go
func GetMaintainerGate() maintainer.Gate {
    projectRoot, err := FindProjectRoot()
    if err != nil {
        panic(fmt.Sprintf("failed to find project root: %v", err))
    }
    cfg, err := GetConfig()
    if err != nil {
        panic(fmt.Sprintf("failed to load config: %v", err))
    }
    var mc *config.MaintainerConfig
    if cfg != nil {
        mc = cfg.Maintainer
    }
    return maintainer.NewFileGate(projectRoot, mc, mc.CacheWindow())
}
```
No atomic container needed because the gate is cheap to re-construct;
it is not cached across calls. (This matches the `Get*Service()` pattern
"new instance per call".)

**Admin command tree.** The existing `adminCmd` in
`internal/cli/commands/admin.go` is the parent. A new file
`admin_maintainer.go` adds:
```go
var adminMaintainerCmd = &cobra.Command{Use: "maintainer", Short: "Manage maintainer password"}
var adminMaintainerSetPasswordCmd = &cobra.Command{Use: "set-password", RunE: runAdminMaintainerSetPassword}

func init() {
    adminMaintainerCmd.AddCommand(adminMaintainerSetPasswordCmd)
    adminCmd.AddCommand(adminMaintainerCmd)
}
```
This mirrors how `cloud.go` hangs `shark cloud init` under `adminCmd`
(cross-check: `internal/cli/commands/cloud.go`).

**Observability integration.** The span pattern uses the same
`repoutil.NewTracer` helper used by the tag repository
(`internal/repository/tag/tag_repository.go:19`). The gate's tracer
lives at the top of `gate.go`:
```go
var gateTracer = repoutil.NewTracer("internal/auth/maintainer")
```
`repoutil` is already imported from `internal/repository/repoutil/` and
does not create a cycle (the auth package lives outside `internal/
repository`; `repoutil` is a leaf package that imports only OTel).
**Caveat & mitigation:** if, on implementation, `repoutil.NewTracer`
pulls in a transitive import that would create a cycle with
`internal/auth/maintainer`, the implementer MUST extract the two-line
`otel.Tracer(name)` wrapper into a new leaf package
`internal/obs/tracing` (or inline it in `gate.go`) rather than creating
a cross-layer dependency. REQ-F-001 takes precedence over reusing
`repoutil`.

**Future tag-service consumption (anchor for F03).** `TagService` in
F03 will receive the gate via constructor injection:
```go
func NewTagService(
    tagRepo TagRepository,
    entityTagRepo EntityTagRepository,
    cfg *config.Config,
    gate maintainer.Gate, // <-- F02 deliverable
) *TagService { ... }
```
F02 does NOT ship this constructor; F03 does. But the `Gate` interface
shape here is the contract F03 will depend on. This is why the
interface is finalized in F02.

### 2.7 Testing Strategy

Per `.claude/rules/testing/architecture.md`:

| Layer | Test Type | What Is Mocked |
|---|---|---|
| `internal/auth/maintainer` | Unit, using `t.TempDir()` for XDG root and an in-memory clock | Nothing is mocked: filesystem is real but isolated per test; clock is injected |
| `internal/config` | Unit | Nothing — pure in-memory JSON round-trip |
| `internal/services/maintainer_bootstrap_service` | Unit | Config reader/writer (mocked via interface) |
| `internal/cli/commands/admin_maintainer` | Unit | `maintainerBootstrapService` (mocked) |
| `internal/cli` (`GetMaintainerGate`) | Integration, using `t.TempDir()` project root + written `.sharkconfig.json` | Nothing — end-to-end through real config and filesystem |

Notable test cases beyond the acceptance criteria:

- **Race.** `TestFileGate_RecordSuccess_Concurrent` runs 8 goroutines
  calling `RecordSuccess` and asserts the final file parses cleanly and
  contains a plausible `last_success`. Verifies REQ-NF-003.
- **XDG precedence.** `TestCachePath_XDGOverrides_HomeCache` sets
  `XDG_CACHE_HOME` explicitly and asserts the session file lands under
  it. Verifies REQ-F-004.
- **No secret in spans.** `TestAuthorize_SpansDoNotLeakPassword` uses
  an OTel span recorder, runs `Authorize` with password
  `"super-secret-xyz"`, asserts that `"super-secret-xyz"` does not
  appear as a substring of any attribute value, event name, or event
  attribute on the recorded span. Verifies REQ-NF-001.
- **Import firewall.** `TestPackage_HasNoSharkDomainImports` parses
  `go list -json` output at test time and asserts no forbidden
  imports. Can be implemented as a `go test` target or as a CI
  lint step. Verifies REQ-F-001, AC-14.

### 2.8 Migration & Rollout

- **No DB migration** (F02 touches no tables). The epic's overall
  schema-version bump is owned by F01 and is unaffected.
- **Config migration** is additive and optional: projects that ship
  without the `maintainer` object see `cfg.Maintainer == nil`, and
  `Authorize` fails with the "missing config" `*UnauthorizedError`
  message directing them to `shark admin maintainer set-password`.
- **No break** to any existing command, because no existing command
  is currently gated. F02 ships with zero call sites; F03 adds the
  first.
- **Documentation** updates for `docs/cli-reference/setup-commands.md`
  and `docs/cli-reference/configuration.md` land in this feature.
  Per-tag docs are explicitly F03/F07.

---

## 3. Exit Gate Cross-Check

| Exit-gate requirement | Where satisfied |
|---|---|
| Every requirement is testable | Each REQ-F/REQ-NF maps to at least one AC (table §1.3); ACs call out the test vehicle (unit/integration). |
| Every architecture decision references existing patterns or explains deviation | F02-D1..D7 each cite either an epic ADR, a CLAUDE.md rule, or an existing package (`fileops`, `repoutil`). F02-D3 explicitly explains a rejection. |
| File paths listed for all changes | §2.1 itemizes every file created/modified/unchanged with absolute-from-repo-root paths. |
| No TBDs in critical sections | Requirements, acceptance criteria, interface shape, file layout, and testing strategy are all concrete. |

---

*Last Updated*: 2026-04-23
