# Test Plan: E34-F01 - Harness-aware prompt rendering

**Created:** 2026-08-31
**Feature Spec:** [spec.md](./spec.md)
**Feature PRD:** [feature.md](./feature.md)
**Design basis:** [decisions.md](./decisions.md) (incl. 2026-08-31 addendum)
**Status:** APPROVED

## Spec Drift Analysis

### Drift Findings

None. `spec.md` §0–§2 is explicit about provenance: it refines `feature.md`
REQ-F-001/REQ-F-002/REQ-NF-001 rather than re-deriving them, and it closes the
three `decisions.md` non-decisions (config schema, `prompt_profile`, backward
compatibility) via the 2026-08-31 addendum rather than leaving them open. No
scope creep was found: `spec.md` §2.4 explicitly excludes the
`model.class`/`model.effort` client-routing contract, `agent_type`
removal/demotion, `model.prompt_profile`, a harness capability registry, and
rewriting shipped prompts — all four map to `feature.md` § Out of Scope items
1–2 or to `decisions.md` addendum items deliberately deferred. No narrowing
was found either: every `feature.md` acceptance-criterion bullet
(REQ-F-001's two bullets, REQ-F-002's two bullets, REQ-NF-001's measurement)
has a corresponding `spec.md` REQ-F-00N/REQ-NF-00N and AC-## row.

### Traceability Matrix

| Feature PRD Requirement | Task/Feature-Spec Acceptance Criterion | Covered? | Notes |
|---|---|---|---|
| REQ-F-001 bullet 1: metadata survives long enough to influence rendering | AC-01, AC-02, AC-03 | Yes | Claim-persisted harness identity, consumed by render |
| REQ-F-001 bullet 2: missing metadata has a defined fallback | AC-04, AC-12, AC-13 | Yes | Zero identity → all-empty `Vars()`, generic branch renders, no `<no value>` leak |
| REQ-F-002 bullet 1: authors have a supported branch mechanism | AC-02, AC-03 | Yes | `isClaude`/`isCodex`/`isHarness` FuncMap additions |
| REQ-F-002 bullet 2: default path still renders generic prompt with no variant | AC-04 | Yes | No claim, no env → `harness` absent from JSON, generic branch used |
| REQ-NF-001: existing workflows render unchanged | AC-07, AC-08 | Yes | Byte-identical prompt/digest regression + next/run parity |
| `decisions.md` addendum item 3 (additive-only wire contract, `agent_type` retained) | REQ-NF-001 / AC-07 | Yes | `omitempty` fields verified not to appear when unset |

### Missing Coverage

None found. `spec.md` §2.5 records no durable unresolved decisions (Q###) for
this feature, so there is no requirement awaiting clarification that would
leave a gap in this plan.

## AC Test Matrix

| AC | Test case(s) | Description | Input/setup | Expected outcome | Edge cases |
|---|---|---|---|---|---|
| AC-01 | TC-001, TC-002 | Claim persists optional harness fields | `shark claim` with all/none of `--harness*` | Values (or empty strings) recorded on the claim row | Only some of the three flags set (TC-001 edge case) |
| AC-02 | TC-003 | Claimed-Claude entity renders the Claude branch | Claim `harness=claude`; template `{{if isClaude .harness}}A{{else}}B{{end}}` | `prompt` has `A`, not `B` | `isHarness` general form vs. `isClaude` wrapper |
| AC-03 | TC-004 | Claimed-Codex entity renders the generic branch | Same template, claim `harness=codex` | `prompt` has `B`, not `A` | none beyond negative case (single enum input) |
| AC-04 | TC-005 | No claim/no env renders generic branch, `harness` absent from JSON | No claim, no env | Exit 0, `prompt` has `B`, no `harness` key | none beyond negative case (this TC *is* the boundary) |
| AC-05 | TC-006 | Flag beats claim beats env (type field) | Claim=codex, env=claude, flag=claude | `harness="claude"` | Flag vs. env disagreeing in the other direction (TC-006 edge case) |
| AC-06 | TC-007 | Per-field precedence (claim wins type, env wins version) | Claim=codex (no version), env version=9.9 | `harness="codex"`, `harness_version="9.9"` | none beyond negative case (this TC *is* the discriminating case) |
| AC-07 | TC-008 | Byte-identical prompts for unbranched workflows | Full golden corpus, pre/post digest | All digests identical | none — full corpus is already the exhaustive edge-case set |
| AC-08 | TC-009, TC-010, TC-011 | `next`/`run` parity at all three precedence tiers | Same fixture driven through both surfaces per tier | Byte-identical prompts | none beyond negative case per tier |
| AC-09 | TC-012 | Harness type trimmed/lowercased; version/model untouched | `--harness="  CLAUDE  "` | `harness="claude"` | Mixed case, no whitespace; version/model NOT lowercased |
| AC-10 | TC-013 | Oversized field rejected pre-write | `--harness=<101 chars>` | Non-zero exit, no claim row written | Exactly 100 chars accepted; same check for version/model fields |
| AC-11 | TC-014 | Migration v34→v35 preserves rows | Real SQLite seeded at v34 | v35, `NULL` harness columns, row preserved | Idempotent rerun |
| AC-12 | TC-015, TC-019 | `Vars()` always returns 3 keys; renderer fails loudly if a key is missing | Zero `HarnessIdentity`; renderer given a map missing `harness` | 3 keys always present; missing key raises the documented execution error | Non-zero `HarnessIdentity` still yields all 3 keys |
| AC-13 | TC-016 | Bare `{{.harness}}` form never leaks `<no value>` | No claim, no env, bare-form template | No `<no value>` leak, exit 0 | none beyond negative case (this TC *is* the edge case) |
| (§5, not numbered) | TC-017 | Harness type is an OTel span attribute; version/model are not | Claimed entity, span capture | `harness` attribute present; version/model absent | none — presence/absence is the whole check |
| (D-F01-05, not numbered) | TC-018 | Claim-read failure degrades to zero identity | `ClaimReader.Get` mock returns error | Exit 0, generic branch, warning logged | none beyond negative case |

## ISTQB Technique Application (per AC)

| AC | Technique(s) Applied | Test Cases Generated | Rationale |
|---|---|---|---|
| AC-01 | Equivalence Partitioning (populated vs. empty optional flags) | TC-001, TC-002 | `--harness`/`--harness-version`/`--harness-model` are three independently-optional fields; partition by "all set" vs "some set" vs "none set" |
| AC-02, AC-03 | Decision Table (harness value × template branch) | TC-003, TC-004 | Two-row decision table: `harness=claude` → branch A; `harness=codex` → branch B |
| AC-04 | Boundary Value Analysis (absence as the zero-value boundary) | TC-005 | "No claim, no env" is the empty-input boundary of the REQ-F-002 precedence chain |
| AC-05, AC-06 | Decision Table (precedence: flag × claim × env, per-field) | TC-006, TC-007 | Precedence is a 3-source × 3-field decision table; AC-05 exercises the type field, AC-06 exercises the version field to prove *per-field* (not per-source) resolution |
| AC-07 | State Transition / Regression (before-state vs. after-state prompt digest) | TC-008 | Compares rendered output across the "before this feature" and "after this feature" states via the existing golden corpus |
| AC-08 | Equivalence Partitioning across dispatch surfaces (`next` vs `run`), applied at all three precedence tiers | TC-009, TC-010, TC-011 | Two production entrypoints must be equivalence-partitioned identically; the spec explicitly requires "all three precedence tiers, including identical `--harness` flags" |
| AC-09 | Boundary Value Analysis (whitespace padding) + Equivalence Partitioning (case folding) | TC-012 | Input normalization is a boundary/equivalence case, not new business logic |
| AC-10 | Boundary Value Analysis (100 vs. 101 characters) | TC-013 | Classic off-by-one boundary at the stated length cap |
| AC-11 | State Transition (schema v34 → v35) | TC-014 | Migration is inherently a state-transition test: pre-state (v34, existing rows) → transition → post-state (v35, NULL columns) |
| AC-12 | Equivalence Partitioning (zero value) + Attack-class enumeration (key omission) | TC-015, TC-019 | The class under test is "does any refactor silently drop a map key" — enumerated explicitly per D-F01-07, covered from both the `Vars()` producer side (TC-015) and the `Render` consumer side (TC-019) |
| AC-13 | Attack-class enumeration (template-form robustness: `{{if}}` vs. bare `{{.field}}`) | TC-016 | D-F01-07's own table names two distinct render-failure classes; AC-13 pins down the bare-field form specifically since AC-04/AC-12 already cover the typed-helper form |

Every runtime AC in `spec.md` §2.3 has at least one technique above; none are
untestable.

## ISO 25010 Coverage Matrix

| AC | Functional | Performance | Compat | Usability | Reliability | Security | Maintainability | Portability |
|---|---|---|---|---|---|---|---|---|
| AC-01 | ✅ TC-001, TC-002 | N/A | N/A | N/A | N/A | ✅ TC-002 (empty-flag path never errors) | N/A | N/A |
| AC-02/AC-03 | ✅ TC-003, TC-004 | N/A | N/A | N/A | N/A | N/A | N/A | N/A |
| AC-04 | ✅ TC-005 | N/A | ✅ TC-005 (REQ-NF-001 no-op case) | N/A | ✅ TC-005 (never blocks dispatch) | N/A | N/A | N/A |
| AC-05/AC-06 | ✅ TC-006, TC-007 | N/A | N/A | N/A | N/A | N/A | N/A | N/A |
| AC-07 | ✅ TC-008 | N/A | ✅ TC-008 (byte-identical regression gate, REQ-NF-001) | N/A | N/A | N/A | ✅ TC-008 (regression guard prevents silent drift) | N/A |
| AC-08 | ✅ TC-009..011 | N/A | ✅ TC-009..011 (REQ-F-006 dual-surface parity) | N/A | N/A | N/A | N/A | N/A |
| AC-09 | ✅ TC-012 | N/A | N/A | N/A | N/A | ✅ TC-012 (input normalization, `.claude/rules/go/input-sanitization.md`) | N/A | N/A |
| AC-10 | ✅ TC-013 | N/A | N/A | ✅ TC-013 (error names the field, quotes input per REQ-NF-004) | N/A | ✅ TC-013 (length cap enforced) | N/A | N/A |
| AC-11 | ✅ TC-014 | ✅ TC-014 (no added per-render I/O beyond the existing indexed read, REQ-NF-003 — asserted qualitatively, not benchmarked) | N/A | N/A | ✅ TC-014 (existing rows preserved, idempotent rerun) | N/A | N/A | N/A |
| AC-12 | ✅ TC-015, TC-019 | N/A | N/A | N/A | ✅ TC-015, TC-019 (locks D-F01-07 contract against regression on both the `Vars()` and `Render` sides) | N/A | ✅ TC-015, TC-019 (regression guard) | N/A |
| AC-13 | ✅ TC-016 | N/A | N/A | N/A | ✅ TC-016 (no render-error / no leaked sentinel) | N/A | N/A | N/A |

### Coverage Gaps

None deferred. Performance is marked `N/A` for most rows because REQ-NF-003
("no new per-render I/O on the hot path") is a code-shape property verified by
TC-014's assertion that resolution reuses `idx_entity_claims_key` — this is
checked once at the DB-repository layer rather than re-measured per AC, since
every AC above exercises the same resolver call path.

## Observability Design (per behavior)

| Behavior | Metric | Log | Trace span | Alert threshold | Test assertion |
|---|---|---|---|---|---|
| Harness identity resolved for a dispatch | N/A (no new metric; low-cardinality field) | N/A | `runNext` OTel span (existing, `next.go` line ~399) gains a `harness` attribute per `spec.md` §5 | N/A | TC-017 asserts the span carries a `harness` attribute equal to the resolved type when non-empty, and that `harness_version`/`harness_model` are **not** added as span attributes (bounded cardinality per §5) |
| Claim-read failure during resolution (REQ-NF-002/D-F01-05) | N/A | A warning is logged (mirrors the existing `next.go:585-601` unknown-status degrade-and-warn posture) | N/A | N/A | TC-018 asserts a claim-read error degrades to the zero identity and a warning line, and does **not** propagate as a render error |
| Harness metadata length-cap rejection (AC-10) | N/A | Validation error returned to the caller (not a log line — CLI error path) | N/A | N/A | Covered by TC-013 |

**Implementation hook:** the `harness` span attribute and the claim-read
warning log become hard requirements in the task spec derived from this test
plan — the developer instruments both as part of the change; QA verifies the
span attribute is present in a `next`/`run` test and that the warning log
line appears when a claim-read is forced to fail.

## Cross-feature contract tests (I-##)

**None.** `spec.md` §7 verifies against `E34-interaction-map.md` in full: I-01
through I-05 are owned by E34-F02/F03/F05/F06/F07/F08/F09; E34-F01 appears in
no producer/consumer column for any registered I-##. No I-## row exists for
this feature to test against, and none is invented here.

## Cross-epic integration tests (X-##)

**None.** `spec.md` §8 verifies against `E34-cross-epic-map.md` and
`docs/product/cross-epic-integration-map.md`: the only E34 row is X-14, owned
by E34-F09 (status `proposed`). E34-F01 neither produces, consumes, nor
validates X-14. No X-## row exists for this feature to test against.

## Integration Scenarios

Cross-component boundaries this feature introduces, per `spec.md` §3.4's
integration diagram:

| Boundary | Components | What to verify | Covering TC(s) |
|---|---|---|---|
| Claim capture | `claim.go` (`runClaim`) → `ClaimService.Claim` → `claim.Repository.Claim` → `entity_claims` table | Harness flags flow unmodified (except type normalization) from CLI args through the service layer into the persisted row; optional fields don't force a sentinel value | TC-001, TC-002, TC-012, TC-013 |
| Claim persistence → migration | `entity_claims` schema (v34→v35) ↔ `claim.Repository` column list (`Claim`/`Get`/`List`/`getByID`) | New nullable columns are included in every repository SQL statement that touches the table, and existing rows survive the migration with `NULL` harness columns | TC-014 |
| Render-time resolution | `next.go resolveEntity` (or `controller.go` step 3/4) → `HarnessResolver.Resolve` → `ClaimReader.Get` (`claim.Repository.Get`) + env vars → merged into `vars` | Per-field precedence (flag > claim > env > zero) holds at the boundary between the resolver and its two upstream sources (claim row, env vars) and its one downstream consumer (`vars` map) | TC-005, TC-006, TC-007, TC-018 |
| Placeholder map → template engine | `vars["harness"\|"harness_version"\|"harness_model"]` → `GetStatusActionPopulated` → `OrchestratorRenderer.Render` (`text/template` + `orchestratorFuncs()`) | All three keys are always present (never conditionally omitted) before the map reaches `Render`; the new `isHarness`/`isClaude`/`isCodex` FuncMap entries evaluate correctly against real map values | TC-003, TC-004, TC-015, TC-016, TC-019 |
| Rendered prompt → digest/assembly | `OrchestratorRenderer.Render` output → `assembleDispatchPrompt` → `PromptSHA256`/`PromptBytes` | Harness metadata entering `vars` before assembly does not perturb `assembleDispatchPrompt` or digest computation for prompts that don't reference `.harness` | TC-008 |
| Dual dispatch surfaces | `next.go` (`shark next`) vs. `internal/runner/controller.go` (`shark run`) — both consume the same `HarnessResolver` via their respective `RunControllerDeps`/`resolveEntity` wiring | Both surfaces resolve and inject harness metadata identically, at all three precedence tiers, given identical inputs (REQ-F-006) | TC-009, TC-010, TC-011 |
| Render-time resolution → observability | `HarnessResolver.Resolve` result → `runNext`'s existing OTel span (`next.go` line ~399) | The resolved harness type (not version/model) is attached as a span attribute per `spec.md` §5's bounded-cardinality decision | TC-017 |

**Epic UAT scenario contribution:** `uat-plan.md` was searched in full for
`F01` and `harness` references. The only relevant hit is UAT-09 ("Preserve
reusable policy handoffs"), whose criterion text is generic to the whole
epic — "Rendered prompts and layered skill consumers preserve the documented
harness and evidence boundaries" — and does not name E34-F01 or any of its
ACs specifically. No epic-level UAT scenario is scoped narrowly enough to
name as "this feature's contribution" beyond that generic mention, and X-14
(the epic's only cross-epic row) is owned by E34-F09, not F01 (confirmed
again in the Cross-epic integration tests section above). This test plan
therefore treats TC-008 (byte-identical unbranched rendering) and TC-009–011
(next/run parity) as the concrete evidence that satisfies UAT-09's "preserve
... harness ... boundaries" language for this feature's slice of the epic.

## Caller-Path Contracts (per test case — Step 5.8)

| TC | Production entrypoint | Lowest mock seam | Forbidden mocks | Counter-factual |
|---|---|---|---|---|
| TC-001 | `runClaim(cmd, args)` in `internal/cli/commands/claim.go`, driven via Cobra's `claimCmd.Execute()` with `--harness=claude --harness-version=2.1.0 --harness-model=opus` | `services.ClaimService.Claim` mocked at the service-interface layer is **forbidden**; mock only `ClaimRepository` (the interface `claim_service.go` depends on) | Do not mock `ClaimService` itself — a buggy `runClaim` that never populates `ClaimInput.Harness*` would still "pass" if the service call is stubbed to echo back whatever the test hardcodes | A buggy `runClaim` that forgets to read the three new flags would submit a `ClaimInput` with empty `Harness*` fields even though the flags were set; the repository-level mock captures the actual `ClaimInput` passed and catches this |
| TC-002 | `runClaim(cmd, args)` with no harness flags | Same as TC-001 | Same as TC-001 | A buggy implementation that defaults empty flags to a non-empty sentinel (e.g. `"unknown"`) instead of `""` is caught by asserting the captured `ClaimInput.Harness == ""` |
| TC-003, TC-004 | `runNext(cmd, args)` in `internal/cli/commands/next.go` end-to-end through `resolveEntity` → `GetStatusActionPopulated` → `OrchestratorRenderer.Render`, driven with a claimed entity and a workflow step whose prompt template is `{{if isClaude .harness}}A{{else}}B{{end}}` | Mock `ClaimReader.Get` (the claim-lookup seam) and the `TaskRepository`/`WorkflowSvc` seams already used by existing `next_test.go` mocks; do **not** mock `OrchestratorRenderer.Render` or `orchestratorFuncs()` | Forbidden: mocking `Render` or stubbing `isClaude`/`isCodex` directly — that would prove the mock branches correctly, not that the FuncMap addition and precedence wiring do | A buggy FuncMap registration (e.g. `isClaude` always returns `false`) or a resolver that never sets `vars["harness"]` would still pass a test that mocks the renderer; driving the real renderer against the real template catches both |
| TC-005 | `runNext(cmd, args)` with no claim and no `SHARK_HARNESS*` env vars set, against the same template as TC-003/004 | Same seam as TC-003/004 | Same as TC-003/004 | A buggy resolver that returns an error (instead of the zero identity) on "no claim found" would make the command exit non-zero; a buggy renderer that omits the `harness` key entirely would fail with `text/template`'s "invalid value" error (D-F01-07) — this test's exit-0 assertion catches both |
| TC-006, TC-007 | `runNext(cmd, args)` with `--harness=claude` flag (TC-006) / no flag (TC-007), a claim carrying `harness=codex`, and `SHARK_HARNESS=claude` / `SHARK_HARNESS_VERSION=9.9` set via `t.Setenv` | Mock only `ClaimReader.Get`; env vars are read through the real `os.Getenv` (via `t.Setenv`, not a mocked env-reader) so the resolver's actual precedence logic runs | Forbidden: mocking `HarnessResolver.Resolve` directly — the resolver **is** the unit under test; mocking it would make the test tautological | A resolver that resolves per-*source* instead of per-*field* (D-F01-04) would return the codex claim's version alongside the codex type in TC-007, when the correct behavior takes `harness=codex` from the claim but `harness_version=9.9` from the env — catching exactly the "per-source" bug the decision rejected |
| TC-008 | `TestRenderedPromptsGolden` in `internal/cli/commands/next_golden_test.go`, extended to compute `PromptSHA256` before and after this feature's change set for every shipped prompt containing no harness branch | None — the golden harness already drives the real `OrchestratorRenderer` and template corpus; no mocking above the renderer | Forbidden: reducing this to a single hand-picked prompt fixture instead of the full existing corpus — the golden suite's value is covering every shipped prompt | A change that alters `vars` construction order, or that appends the harness keys in a way that changes map iteration behavior fed into a template, could shift byte output for prompts that never reference `.harness`; the full-corpus digest comparison catches drift the hand-picked case would miss |
| TC-009, TC-010, TC-011 | `runNext(cmd, args)` (`next.go`) and the `RunController.Run` step-3/4 path (`internal/runner/controller.go`), both invoked against the same entity/claim/step fixture, for each of the three precedence tiers (flag, claim, env) plus the explicit-`--harness`-flag-to-both-surfaces case AC-08 calls out | Mock `EntityTransitioner`, `PlaceholderGenerator`, `ClaimReader` — the same interfaces both `next.go` and `controller.go` already accept per `RunControllerDeps`; do not mock `HarnessResolver.Resolve` itself | Forbidden: asserting parity by comparing two calls to the *same* internal resolver function — the contract is that **both command surfaces** wire the resolver identically; each must be driven through its own real entrypoint (`runNext` vs `controller.Run`) | A fix that adds the resolver call to `next.go` but forgets `internal/runner/controller.go` (as `spec.md`'s "Required for REQ-F-006/AC-08" callout on `run.go` warns against) produces two different prompts from the identical fixture; driving both real entrypoints is the only way this is caught |
| TC-012 | `runClaim(cmd, args)` with `--harness="  CLAUDE  "` | Same seam as TC-001 | Same as TC-001 | A resolver/model layer that stores the raw string would surface `"  CLAUDE  "` (or `"CLAUDE"`) instead of `"claude"` in `shark claims --json`; asserting the round-tripped, displayed value (not just an internal normalization function in isolation) catches a forgotten call site |
| TC-013 | `runClaim(cmd, args)` with `--harness=<101 chars>` | Same seam as TC-001; also assert the repository mock's `Claim` method is **never called** | Forbidden: asserting only that `models.EntityClaim.Validate()` returns an error in isolation — the CLI's non-zero exit, the error message content (field name + quoted input per REQ-NF-004), and "no claim row written" are all part of AC-10 and must be observed at the command boundary | A validator that returns an error but whose caller (`runClaim`) ignores it and proceeds to call `Claim` anyway would still "look right" if only `Validate()` is unit-tested; asserting the repository's `Claim` was never invoked catches that |
| TC-014 | `db.ApplySchemaAndMigrations` (or `db.ApplySchemaIfNeeded` per the `skip_migrations` path) driven against a real SQLite file seeded at schema v34 with pre-existing `entity_claims` rows, following the existing pattern in `internal/db/db_test.go` for `migrateSprintAssignmentsAddSprintOrder` | None — migration tests in this codebase drive the real `*sql.DB`; per CLAUDE.md, repository/DB-layer tests use a real (temp-file or in-memory) database, not a mock | Forbidden: mocking `*sql.DB` — migrations are pure SQL-shape correctness and must run against a real SQLite engine, consistent with existing `db_test.go` convention | A migration that adds the columns but forgets `CurrentSchemaVersion`'s bump would never run on an existing v34 database under `skip_migrations: true`; seeding a real pre-migration DB and asserting `PRAGMA table_info` post-run is the only way to catch that specific failure mode (as opposed to testing the migration function in isolation, which would pass even if nothing calls it) |
| TC-015 | `HarnessIdentity{}.Vars()` — direct call; this is a pure value-object method with no wider caller-path to drive | internal — function under test **is** the entrypoint | n/a | A future "simplify" refactor that switches to `omitempty`-style conditional key insertion (e.g. building the map with an `if v != ""` guard) would silently reintroduce the D-F01-07 failure mode; asserting `len(m) == 3` and each key's presence (not just its value) locks this down |
| TC-016 | `runNext(cmd, args)` against a step prompt using the bare form `{{.harness}}`, no claim, no env | Mock `ClaimReader.Get`, `TaskRepository` — same seam as TC-003/005 | Forbidden: mocking `OrchestratorRenderer.Render` — the literal `<no value>` leak is a `text/template` execution-time behavior that only a real `Render` call can produce or avoid | A resolver that supplies the harness keys only when a `{{if}}`-style template is detected (a plausible but wrong optimization) would still leak `<no value>` for the bare-field form; driving the real renderer with the bare-field fixture is required to catch it |
| TC-017 | `runNext(cmd, args)` for a claimed entity with `harness=claude`, asserting on the emitted OTel span via the existing test-tracer-provider pattern already used for `runNext`'s span (if one exists in `next_test.go`); otherwise, a new minimal in-memory span exporter | Mock the same seams as TC-003 for entity/claim data; do not mock the tracer/span-recording | internal-only for the *attribute-value* assertion (no external caller drives spans) but the *span itself* is emitted by the real `runNext` code path, not by a test-only tracer wrapper | A developer who resolves harness metadata but forgets the `spec.md` §5 span-attribute requirement would pass every functional AC while leaving the feature undiagnosable in production; this test is what makes that omission a QA-visible gap rather than a silent one |
| TC-018 | `runNext(cmd, args)` with a `ClaimReader.Get` mock that returns an error | Mock `ClaimReader.Get` to return an error (the one seam the resolver is contractually allowed to fail from, per D-F01-05) | Forbidden: mocking `HarnessResolver.Resolve` to directly return the zero identity — that proves the mock, not the resolver's actual error-swallowing behavior | A resolver that propagates the claim-read error as a render error instead of degrading (D-F01-05) would make the command exit non-zero; this test's exit-0 + warning-log assertion catches that regression |
| TC-019 | `OrchestratorRenderer.Render(template, vars)` in `internal/templates/orchestrator_renderer_test.go`, called directly with a `map[string]string` that omits the `harness` key entirely, against the fixture `{{if isClaude .harness}}A{{else}}B{{end}}` | internal — `Render` is itself the entrypoint `spec.md` §3.3 names for this exact regression case | Forbidden: pre-populating the map with `harness: ""` before calling `Render` — that tests TC-016's scenario (key present, empty), not this one (key absent) | A future change to how `vars` is constructed upstream (e.g. a resolver or a `next.go`/`controller.go` refactor that conditionally skips merging harness keys) would silently degrade from "always three keys" back to "sometimes missing key"; this test pins the exact D-F01-07 failure string (`at <.harness>: invalid value; expected string`) so that regression fails loudly at the renderer-unit level, not just via the higher-level TC-005/TC-018 exit-0 checks which would also fail but wouldn't localize the cause |

## Test Infrastructure

**Existing patterns to follow:**

- `internal/cli/commands/next_test.go` — mocked-service pattern for `runNext`;
  reuse its `EntityTransitioner`/`PlaceholderGenerator`/`ActionSvc` mock
  scaffolding for TC-003 through TC-007, TC-016, TC-017, TC-018.
- `internal/cli/commands/next_golden_test.go` — golden-corpus digest
  comparison; TC-008 extends `goldenVars()` and the existing digest-comparison
  loop rather than introducing a parallel mechanism. The `-update` flag
  convention is preserved.
- `internal/db/db_test.go`'s existing coverage of
  `migrateSprintAssignmentsAddSprintOrder` — the `ADD COLUMN` +
  `PRAGMA table_info` idempotence-and-preservation pattern is the direct
  template for TC-014.
- `internal/templates/orchestrator_renderer_test.go` — direct-call tests for
  FuncMap helpers (`isSimple`/`isStandard`/`isComplex` precedent); TC-003/004
  add the harness predicates here at the unit level in addition to the
  integration-level TC-003/004 above (belt-and-braces: FuncMap correctness in
  isolation, then wiring correctness through `runNext`).
- `.claude/rules/cli/commands.md` — CLI-command test-mocking convention (never
  a real database in CLI-layer tests); followed by every `next.go`/`claim.go`
  test case above.
- CLAUDE.md testing-architecture rule: repository/DB-layer tests use a real
  database (SQLite temp file or in-memory), service/CLI-layer tests use
  mocks. TC-014 is the one DB-layer test in this plan and is the only one
  that legitimately drives a real `*sql.DB`.

**New test helpers needed:**

- A small `fakeClaimReader` (or extend an existing `MockClaimRepository` if
  one exists) implementing the one-method `ClaimReader` interface
  (`Get(ctx, entityType, entityKey) (*models.EntityClaim, error)`) — required
  because `HarnessResolver` is a new type with no existing mock.
- A minimal fixture workflow YAML (or in-test template string) containing
  `{{if isClaude .harness}}A{{else}}B{{end}}` and a second fixture using the
  bare `{{.harness}}` form, for TC-003/004/005/013/016 — these branch shapes
  do not exist in the current shipped prompt corpus and must be added as test
  fixtures only (`spec.md` §2.4 item 5: "mechanism ships with test fixtures
  only").
- If `next_test.go` has no existing OTel span-capture harness, a minimal
  in-memory `trace.SpanExporter` for TC-017; otherwise reuse what exists.

## Acceptance Test Cases

### TC-001: Claim records all three harness fields when supplied

**Feature Requirement:** `feature.md` REQ-F-001 bullet 1 ("metadata survives
long enough to influence prompt rendering"); `spec.md` REQ-F-001, AC-01.
**Task Acceptance Criterion:** AC-01.
**Technique Applied:** Equivalence Partitioning (all-three-set partition).
**ISO 25010 Characteristic(s):** Functional Suitability.

**Caller-Path Contract:** see table above (TC-001).

**Preconditions:** Entity `E34-F01-001` exists and is unclaimed.

**Input:** `shark claim E34-F01-001 --by=agent1 --harness=claude --harness-version=2.1.0 --harness-model=opus --json`

**Expected Output:** Exit 0. `shark claims --json` reports a claim on
`E34-F01-001` with `harness="claude"`, `harness_version="2.1.0"`,
`harness_model="opus"`.

**Observability Evidence:** N/A (claim-time; no render span involved).

**Edge Cases:**
- Only `--harness` supplied (no version/model): claim row has
  `harness="claude"`, `harness_version=""`, `harness_model=""`.

**Negative Cases:** The claim must not fail or warn when harness flags are
absent — see TC-002.

---

### TC-002: Claim succeeds with no harness flags supplied

**Feature Requirement:** `feature.md` REQ-F-001 ("Values are optional...
supplying none is valid," `spec.md` REQ-F-001 bullet 1).
**Task Acceptance Criterion:** AC-01 (negative/edge partition).
**Technique Applied:** Equivalence Partitioning (all-unset partition).
**ISO 25010 Characteristic(s):** Functional Suitability, Security (no forced
input, no default sentinel value injected).

**Caller-Path Contract:** see table above (TC-002).

**Preconditions:** Entity `E34-F01-002` exists and is unclaimed.

**Input:** `shark claim E34-F01-002 --by=agent1 --json` (no `--harness*` flags).

**Expected Output:** Exit 0. The captured `ClaimInput` has
`Harness == "" && HarnessVersion == "" && HarnessModel == ""`. `shark claims
--json` output omits the three fields (`omitempty`) or reports them as empty
per the wire schema — not a placeholder string like `"unknown"`.

**Negative Cases:** No `"unknown"`, `"none"`, or `"unset"` sentinel value is
ever written.

---

### TC-003: Harness branch resolves to the Claude-specific instruction

**Feature Requirement:** `feature.md` REQ-F-002 bullet 1; `spec.md`
REQ-F-002/REQ-F-004, AC-02.
**Task Acceptance Criterion:** AC-02.
**Technique Applied:** Decision Table (harness=claude row).
**ISO 25010 Characteristic(s):** Functional Suitability.

**Caller-Path Contract:** see table above (TC-003/004).

**Preconditions:** Entity claimed via a fixture with `harness="claude"`. The
step's prompt fixture is `{{if isClaude .harness}}A{{else}}B{{end}}`.

**Input:** `shark next <key> --json`.

**Expected Output:** Exit 0. `prompt` contains the substring `A` and does not
contain `B`. `harness` in the response JSON is `"claude"`.

**Edge Cases:** Same assertion using `isHarness "claude" .harness` directly
(not just the `isClaude` convenience wrapper) to prove the general helper
also works.

**Negative Cases:** `prompt` must not contain both `A` and `B` (branch
exclusivity).

---

### TC-004: Harness branch resolves to the generic/Codex instruction

**Feature Requirement:** `feature.md` REQ-F-002 bullet 1; `spec.md` AC-03.
**Task Acceptance Criterion:** AC-03.
**Technique Applied:** Decision Table (harness=codex row — the table's second
row, completing the two-row decision table with TC-003).
**ISO 25010 Characteristic(s):** Functional Suitability.

**Caller-Path Contract:** see table above (TC-003/004).

**Preconditions:** Same entity/prompt fixture as TC-003, claimed with
`harness="codex"`.

**Input:** `shark next <key> --json`.

**Expected Output:** Exit 0. `prompt` contains `B` and not `A`. `harness` is
`"codex"`.

**Edge Cases:** None beyond the negative case — `harness` here is a
two-value enum for the purposes of this template fixture (`claude` in TC-003,
anything else falls to the generic branch), so TC-003/TC-004 together already
exhaust the equivalence classes.

**Negative Cases:** `isClaude` must return `false` for `"codex"` — this
guards against a helper that defaults to `true`.

---

### TC-005: No claim, no env — generic branch renders and harness is absent from JSON

**Feature Requirement:** `feature.md` REQ-F-002 bullet 2; REQ-NF-002;
`spec.md` AC-04.
**Task Acceptance Criterion:** AC-04.
**Technique Applied:** Boundary Value Analysis (the all-sources-empty
boundary of the four-tier precedence chain in REQ-F-002).
**ISO 25010 Characteristic(s):** Functional Suitability, Reliability
(never blocks dispatch), Compatibility (REQ-NF-001 no-op path).

**Caller-Path Contract:** see table above (TC-005).

**Preconditions:** Entity is unclaimed. No `SHARK_HARNESS*` env vars set —
use `os.Unsetenv("SHARK_HARNESS")` / `os.Unsetenv("SHARK_HARNESS_VERSION")` /
`os.Unsetenv("SHARK_HARNESS_MODEL")` (restoring any prior value via
`t.Cleanup`), **not** `t.Setenv(k, "")`. The two are observably different
states to a resolver implemented via `os.LookupEnv` (present-even-if-empty)
rather than "value is non-empty" — this test specifically exercises the
"genuinely absent" boundary.

**Input:** `shark next <key> --json`.

**Expected Output:** Exit code 0. `prompt` contains `B`. The JSON response
has no `harness` key (or it is `""`/omitted per `omitempty`) — AC-04's
literal wording "`harness` is absent from the JSON" is asserted via
`json.Unmarshal` into a `map[string]interface{}` and checking `_, ok :=
m["harness"]; !ok`.

**Observability Evidence:** N/A for this TC (see TC-017 for the span-omission
counterpart).

**Edge Cases:** None beyond the negative case for this TC's own precondition.
Design note for the developer: `os.Getenv` collapses "unset" and
"set-to-empty" into the same `""` return value, so the resolver's precedence
rule must be implemented as "value is non-empty wins," not "variable is
present (via `os.LookupEnv`) wins" — the two are indistinguishable for a
string-only field today but would diverge if a future field ever had a
meaningful non-string zero value.

**Negative Cases:** Command must not exit non-zero; must not print
`<no value>` anywhere in `prompt`.

---

### TC-006: Flag beats claim beats env (type field)

**Feature Requirement:** `spec.md` REQ-F-002, AC-05.
**Task Acceptance Criterion:** AC-05.
**Technique Applied:** Decision Table (3-source precedence, type field row).
**ISO 25010 Characteristic(s):** Functional Suitability.

**Caller-Path Contract:** see table above (TC-006/007).

**Preconditions:** Entity claimed with `harness="codex"`.
`SHARK_HARNESS=claude` exported via `t.Setenv`.

**Input:** `shark next <key> --harness=claude --json`.

**Expected Output:** `harness` in the JSON response is `"claude"` (flag
wins over both claim and env).

**Edge Cases:**
- `--harness=claude` with `SHARK_HARNESS=codex` (flag and env disagreeing in
  the *other* direction from the main scenario) — rules out a resolver that
  happens to match by luck when flag and claim/env values coincide.

**Negative Cases:** `harness` must not be `"codex"` (would indicate claim
incorrectly outranks the flag) and must not silently pick the env value by
coincidence.

---

### TC-007: Per-field precedence — claim wins the type field, env wins the version field

**Feature Requirement:** `spec.md` REQ-F-002 ("Precedence is evaluated
per field, not per source"), D-F01-04, AC-06.
**Task Acceptance Criterion:** AC-06.
**Technique Applied:** Decision Table (3-source precedence, version field
row) — this is the discriminating test between "per-field" (correct, D-F01-04)
and "per-source" (rejected alternative) resolution strategies.
**ISO 25010 Characteristic(s):** Functional Suitability.

**Caller-Path Contract:** see table above (TC-006/007).

**Preconditions:** Entity claimed with `harness="codex"` only (no version set
on the claim). `SHARK_HARNESS_VERSION=9.9` exported via `t.Setenv`.

**Input:** `shark next <key> --json` (no override flags).

**Expected Output:** `harness` is `"codex"` (from the claim — the only
non-empty type source), `harness_version` is `"9.9"` (from env — the only
non-empty version source for this field).

**Edge Cases:** None beyond the negative case — this TC is itself the
discriminating edge case between per-field and per-source resolution; there
is no further partition to add without duplicating TC-006's flag-tier
coverage.

**Negative Cases:** `harness_version` must not be `""` (would indicate a
per-source resolver stopping at "claim wins" for the whole record and never
consulting env for the version field it left blank).

---

### TC-008: Byte-identical prompts for unbranched workflows (regression gate)

**Feature Requirement:** `feature.md` REQ-NF-001; `spec.md` REQ-NF-001,
AC-07.
**Task Acceptance Criterion:** AC-07.
**Technique Applied:** State Transition / Regression (before/after digest
comparison across the full shipped-prompt corpus).
**ISO 25010 Characteristic(s):** Compatibility, Maintainability (regression
guard).

**Caller-Path Contract:** see table above (TC-008).

**Preconditions:** The existing `TestRenderedPromptsGolden` corpus in
`next_golden_test.go` (every shipped workflow prompt, none containing a
harness branch per `spec.md` §2.4 item 5).

**Input:** Run the golden test both against the current committed
`.golden` fixtures (captured before this feature's code change) and against
the post-change render output.

**Expected Output:** `PromptSHA256` values are identical pre/post change for
every prompt in the corpus. No `.golden` file requires a `-update` re-run as
part of this feature's change set.

**Edge Cases:** None beyond the negative case — the full shipped-prompt
corpus is already the exhaustive input set for this regression gate; there is
no narrower partition that adds coverage.

**Negative Cases:** If any digest changes, the feature has introduced a
non-additive rendering change and REQ-NF-001 is violated — this is a hard
gate, not an advisory finding.

---

### TC-009: `next`/`run` parity — flag-tier precedence

**Feature Requirement:** `feature.md` REQ-NF-001 (renders identically);
`spec.md` REQ-F-006, AC-08.
**Task Acceptance Criterion:** AC-08 (flag tier).
**Technique Applied:** Equivalence Partitioning across dispatch surfaces.
**ISO 25010 Characteristic(s):** Compatibility.

**Caller-Path Contract:** see table above (TC-009/010/011).

**Preconditions:** Same entity, claim, and step fixture driven through both
`runNext` and `RunController.Run`, both invoked with `--harness=claude
--harness-version=2.1.0 --harness-model=opus`.

**Input:** `shark next <key> --harness=claude --harness-version=2.1.0
--harness-model=opus --json` and the equivalent `shark run` controller
invocation with the same three flags.

**Expected Output:** The two rendered `prompt` strings are byte-identical.

**Edge Cases:** None beyond the negative case — TC-009/010/011 together
already partition the three precedence tiers exhaustively for this
equivalence class (dispatch-surface parity).

**Negative Cases:** A mismatch on any single character fails this test —
no "close enough" comparison.

---

### TC-010: `next`/`run` parity — claim-tier precedence

**Feature Requirement:** Same as TC-009.
**Task Acceptance Criterion:** AC-08 (claim tier).
**Technique Applied:** Equivalence Partitioning across dispatch surfaces.
**ISO 25010 Characteristic(s):** Compatibility.

**Caller-Path Contract:** see table above (TC-009/010/011).

**Preconditions:** Same entity/claim/step fixture, claimed with
`harness=codex`, no override flags on either surface.

**Input:** `shark next <key> --json` and the equivalent no-flag `shark run`
controller invocation.

**Expected Output:** Byte-identical rendered prompts, both reflecting the
claim's `codex` identity.

**Edge Cases:** None beyond the negative case (implicit: mismatch fails,
same as TC-009) — see TC-009's rationale for why the three-tier set is
already exhaustive.

---

### TC-011: `next`/`run` parity — env-tier precedence

**Feature Requirement:** Same as TC-009.
**Task Acceptance Criterion:** AC-08 (env tier).
**Technique Applied:** Equivalence Partitioning across dispatch surfaces.
**ISO 25010 Characteristic(s):** Compatibility.

**Caller-Path Contract:** see table above (TC-009/010/011).

**Preconditions:** Entity unclaimed on both surfaces. `SHARK_HARNESS=claude`
exported via `t.Setenv` for the duration of both calls.

**Input:** `shark next <key> --json` and the equivalent `shark run`
controller invocation, no flags, same env.

**Expected Output:** Byte-identical rendered prompts, both reflecting
`harness=claude` sourced from env.

**Edge Cases:** None beyond the negative case — see TC-009's rationale.

**Negative Cases:** This is the specific case `spec.md`'s `run.go` row
warns is easy to miss (`run.go` never wired to receive override flags would
still coincidentally pass tiers where env/claim already agree, but a real
regression here would only show up if `controller.go`'s resolver wiring were
missing entirely — which this test catches by using the *same* fixture as
TC-009/010, so any surface-specific gap surfaces as a TC-009/010/011
inconsistency rather than being individually invisible).

---

### TC-012: Harness type is trimmed and lowercased on claim

**Feature Requirement:** `spec.md` REQ-F-001 ("`--harness` is normalized
(trimmed, lowercased)"), AC-09.
**Task Acceptance Criterion:** AC-09.
**Technique Applied:** Boundary Value Analysis (leading/trailing whitespace)
+ Equivalence Partitioning (mixed case).
**ISO 25010 Characteristic(s):** Security (input normalization per
`.claude/rules/go/input-sanitization.md`), Functional Suitability.

**Caller-Path Contract:** see table above (TC-012).

**Preconditions:** Entity unclaimed.

**Input:** `shark claim <key> --harness="  CLAUDE  " --by=agent1 --json`.

**Expected Output:** `shark claims --json` reports `harness` as `"claude"`
(no surrounding whitespace, all lowercase).

**Edge Cases:**
- `--harness="Claude"` (mixed case, no whitespace) → `"claude"`.
- `--harness-version` and `--harness-model` are explicitly **not**
  lowercased (`spec.md` REQ-F-001: "trimmed opaque strings" only) — assert a
  mixed-case `--harness-model="Opus-4"` round-trips as `"Opus-4"` unchanged,
  to pin down that normalization is type-field-only.

---

### TC-013: Oversized harness value is rejected before any claim row is written

**Feature Requirement:** `spec.md` REQ-NF-004, AC-10.
**Task Acceptance Criterion:** AC-10.
**Technique Applied:** Boundary Value Analysis (100 vs. 101 characters — the
stated cap).
**ISO 25010 Characteristic(s):** Security, Usability (error names the field
and quotes the input).

**Caller-Path Contract:** see table above (TC-013).

**Preconditions:** Entity unclaimed.

**Input:** `shark claim <key> --harness=<101-char string> --by=agent1`.

**Expected Output:** Non-zero exit. Error message names the field
(`harness`) and quotes the offending input with `%q` (per
`.claude/rules/go/input-sanitization.md`). The mocked repository's `Claim`
method is never invoked.

**Edge Cases:**
- Exactly 100 characters is accepted (boundary-inclusive per REQ-NF-004
  "length-capped at 100 characters each").
- Repeat for `--harness-version` and `--harness-model` independently (each
  field is capped at 100 chars per REQ-NF-004, not just `--harness`).

**Negative Cases:** No partial claim row is written; a claim that fails
validation on `harness_model` must not still record `harness`/
`harness_version`.

---

### TC-014: Schema migration v34→v35 preserves existing claim rows

**Feature Requirement:** `spec.md` §3.1, AC-11; `.claude/rules/database-critical.md`.
**Task Acceptance Criterion:** AC-11.
**Technique Applied:** State Transition (pre-migration state → migration →
post-migration state), following the existing
`migrateSprintAssignmentsAddSprintOrder` test pattern.
**ISO 25010 Characteristic(s):** Reliability, Performance (idempotent rerun
costs no more than the guarded `ADD COLUMN` check).

**Caller-Path Contract:** see table above (TC-014).

**Preconditions:** A real (temp-file) SQLite database initialized at schema
version 34, with the pre-migration `entity_claims` table populated with at
least one existing row (no harness columns).

**Input:** Run `ApplySchemaAndMigrations` (or the `ApplySchemaIfNeeded` path
with a stale stored version, matching how `skip_migrations: true`
production databases actually upgrade).

**Expected Output:** Schema version is now 35. `PRAGMA table_info
(entity_claims)` includes `harness`, `harness_version`, `harness_model` as
nullable `TEXT` columns. The pre-existing row's original columns are
unchanged, and its three new harness columns are `NULL`.

**Edge Cases:**
- Rerunning the migration function a second time against an
  already-migrated database is a no-op and does not error (SQLite has no
  `ADD COLUMN IF NOT EXISTS`; the `PRAGMA table_info` guard is what this
  edge case exercises).

**Negative Cases:** Migration must not drop or truncate the pre-existing
row, and must not require a backfill value (`NULL` is correct per §3.1).

---

### TC-015: `HarnessIdentity{}.Vars()` always returns exactly three keys

**Feature Requirement:** `spec.md` §3.2 (`Vars()` contract), D-F01-07,
AC-12.
**Task Acceptance Criterion:** AC-12.
**Technique Applied:** Equivalence Partitioning (zero value) + Attack-class
enumeration (the "class" being: any code path that conditionally omits a
map key).
**ISO 25010 Characteristic(s):** Reliability, Maintainability (regression
guard against a future "tidy up empty values" refactor, explicitly
anticipated in `spec.md` D-F01-07).

**Caller-Path Contract:** see table above (TC-015) — internal-only,
justified: `Vars()` is a pure value-object method with no wider production
caller shape to drive; the function under test is itself the entrypoint.

**Preconditions:** None — `var id HarnessIdentity` (zero value).

**Input:** `id.Vars()`.

**Expected Output:** Returned map has exactly 3 entries:
`{"harness": "", "harness_version": "", "harness_model": ""}`. Assert both
`len(m) == 3` and each key's presence via `_, ok := m[key]; ok` (not just
value equality) — the whole point of D-F01-07 is that key *presence* is
load-bearing, independent of value.

**Edge Cases:**
- Non-zero `HarnessIdentity{Type: "claude"}` → `Vars()` still returns all
  three keys, with `harness_version`/`harness_model` present and empty.

**Negative Cases:** A future implementation using
`if v.Type != "" { m["harness"] = v.Type }`-style conditional insertion must
fail this test even though every *value* it produces might individually look
correct.

---

### TC-016: Bare `{{.harness}}` form never leaks `<no value>` or errors

**Feature Requirement:** `spec.md` REQ-NF-002, D-F01-07, AC-13.
**Task Acceptance Criterion:** AC-13.
**Technique Applied:** Attack-class enumeration (the second of the two
render-failure classes named in D-F01-07's table: typed-helper-absent-key
[covered by TC-005/TC-015] vs. bare-field-absent-key [this TC]).
**ISO 25010 Characteristic(s):** Reliability.

**Caller-Path Contract:** see table above (TC-016).

**Preconditions:** Entity unclaimed, no harness env vars. Step prompt
fixture is the bare form `{{.harness}}` (not wrapped in `{{if}}`).

**Input:** `shark next <key> --json`.

**Expected Output:** Exit 0. `prompt` does not contain the literal string
`<no value>`. No render error is returned. The substitution position in
`prompt` is empty (adjacent literal text on both sides of where `{{.harness}}`
appeared is contiguous with nothing between).

**Edge Cases:** None beyond the negative case — the bare-field form has only
one failure mode (the literal `<no value>` sentinel), which the negative
case below directly targets.

**Negative Cases:** This is exactly the failure D-F01-07's table predicts
for "key present but empty" vs. "key absent" — this TC specifically
constructs the "key absent" scenario. If harness keys were only injected
conditionally (as TC-015 also guards against from the `Vars()` side), this
TC catches the same defect class from the rendering side.

---

### TC-017: Resolved harness type is added as an OTel span attribute; version/model are not

**Feature Requirement:** `spec.md` §5 (Operations and observability).
**Task Acceptance Criterion:** Derived from §5, not a numbered AC — this is
the observability-design test required by Step 5.7/7.5 item 5 of the
test-planning workflow, closing the gap between "renders correctly" and
"is diagnosable in production."
**Technique Applied:** Equivalence Partitioning (attribute present vs.
absent) + explicit non-requirement check (cardinality bound).
**ISO 25010 Characteristic(s):** Reliability (observability), Security
(bounded attribute cardinality — unbounded free-text version/model strings
as span attributes would be a cardinality/cost concern, which is why §5
explicitly excludes them).

**Caller-Path Contract:** see table above (TC-017).

**Preconditions:** Entity claimed with `harness=claude`,
`harness_version=2.1.0`, `harness_model=opus`.

**Input:** `shark next <key> --json`, with the existing `runNext` OTel span
captured via an in-memory span exporter.

**Expected Output:** The emitted span for this `runNext` invocation has a
`harness` attribute with value `"claude"`. The span does **not** have
`harness_version` or `harness_model` attributes.

**Negative Cases:** If a future change adds `harness_version`/
`harness_model` as span attributes, this test fails and forces an explicit
decision (the §5 cardinality rationale must be revisited, not silently
overridden).

---

### TC-018: Claim-read failure degrades to the zero identity, never a render error

**Feature Requirement:** `spec.md` REQ-NF-002, D-F01-05.
**Task Acceptance Criterion:** Derived from REQ-NF-002/D-F01-05 (error
handling is asserted structurally within REQ-NF-002's coverage rather than
a separately numbered AC in `spec.md` §2.3, since AC-04 covers the "no
claim exists" case and this TC covers the distinct "claim lookup errors"
case D-F01-05 names explicitly).
**Technique Applied:** Attack-class enumeration (fault-injection: the one
external call in the resolve path, a claim-repository read, is forced to
fail).
**ISO 25010 Characteristic(s):** Reliability.

**Caller-Path Contract:** see table above (TC-018).

**Preconditions:** `ClaimReader.Get` mock configured to return a non-nil
error for the entity under test.

**Input:** `shark next <key> --json`.

**Expected Output:** Exit 0. `prompt` renders the generic branch (as if
unclaimed). A warning is logged (mirroring the `next.go:585-601` posture).
No `harness` key present in the JSON response (or empty, consistent with
AC-04's zero-identity behavior).

**Negative Cases:** Command must not exit non-zero; the claim-read error
must not be surfaced as a fatal dispatch failure.

---

### TC-019: Renderer fails loudly when the `harness` key is missing from the map (D-F01-07 regression pin)

**Feature Requirement:** `spec.md` §3.3 ("a regression case executing a
harness-branching template against a map missing the harness keys, asserting
the failure mode recorded in D-F01-07"), D-F01-07.
**Task Acceptance Criterion:** AC-12 (the renderer-side half of the
key-presence contract; TC-015 covers the `Vars()`-side half).
**Technique Applied:** Attack-class enumeration (the first of D-F01-07's two
named render-failure classes: typed-helper-absent-key).
**ISO 25010 Characteristic(s):** Reliability, Maintainability (this is
explicitly a regression pin against a plausible future refactor).

**Caller-Path Contract:** see table above (TC-019) — internal-only,
justified: `OrchestratorRenderer.Render` is the exact function `spec.md` §3.3
names as needing this case, and there is no wider production caller whose
signature adds discriminating power here; the point is to catch a defect at
the renderer unit level rather than only downstream.

**Preconditions:** A `map[string]string` containing every other key the
fixture template needs, but explicitly **omitting** `"harness"`.

**Input:** `OrchestratorRenderer.Render(tmpl, vars)` where `tmpl` is
`{{if isClaude .harness}}A{{else}}B{{end}}` and `vars` has no `"harness"` key.

**Expected Output:** `Render` returns a non-nil error whose message matches
the documented failure D-F01-07 records:
`at <.harness>: invalid value; expected string` (or the exact current Go
`text/template` wording — assert on the stable substring, not brittle full
match, since exact Go stdlib error text can shift across Go versions).

**Edge Cases:** None beyond the negative case — this test exists solely to
pin one specific, already-measured failure string; there is no broader
partition to explore.

**Negative Cases:** `Render` must **not** silently succeed with the
`{{else}}` branch — a renderer that swallowed the missing-key error and fell
back to the false branch would look like AC-04's correct behavior but would
actually be masking a caller bug (a caller that failed to inject the harness
keys at all, as opposed to correctly injecting them empty).

---

## Codex Test-Plan Red-Team

**Verdict:** NOT RUN
**Issues raised:** N/A
**Issues addressed before dev:** N/A
**Issues deferred:** N/A

This dispatch's workflow prompt (Step 7.5 of `quality/workflows/test-planning.md`)
calls for running a `codex_command` against this draft, but no `codex_command`
was supplied to this dispatched step, and the parent prompt's concrete
READ/PRODUCE/EXIT GATE instructions for this dispatch do not list a codex
run as an exit criterion. Logged here as a non-blocking gap per the
workflow's own two-failure fallback instruction, rather than fabricating a
codex run. If the parent loop wants the red-team pass, dispatch `/codex-exec`
against this file with the Step 7.5 prompt body (open-endedness, technique
fit, enumeration completeness, ISO 25010 coverage, observability, negative
cases, caller-path contract) before development starts.

## Recommendations

- [x] Ready for development (no drift, spec is clear, every AC has a
  technique + ISO matrix entry, observability designed).

All 13 acceptance criteria in `spec.md` §2.3 map to at least one test case
above (AC-01→TC-001/002, AC-02→TC-003, AC-03→TC-004, AC-04→TC-005,
AC-05→TC-006, AC-06→TC-007, AC-07→TC-008, AC-08→TC-009/010/011,
AC-09→TC-012, AC-10→TC-013, AC-11→TC-014, AC-12→TC-015/TC-019, AC-13→TC-016),
plus two observability/reliability test cases (TC-017, TC-018) derived from
§5 and D-F01-05 that are not separately numbered ACs but are required by the
test-planning workflow's observability-design step, and one regression-pin
test case (TC-019, for D-F01-07's renderer-side failure mode named explicitly
in `spec.md` §3.3) added during review of this draft to close a gap where
`spec.md`'s own test-file list had no corresponding TC. No orphaned tests:
every TC above traces to a `spec.md` requirement, and every `spec.md`
requirement traces to `feature.md`.
