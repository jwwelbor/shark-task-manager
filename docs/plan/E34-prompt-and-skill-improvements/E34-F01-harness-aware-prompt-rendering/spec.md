---
feature_key: E34-F01-harness-aware-prompt-rendering
epic_key: E34
type: combined-requirements-and-architecture-spec
tier: STANDARD
last_updated: 2026-08-31
---

# E34-F01 Specification — Harness-aware prompt rendering

Combined requirements + architecture spec. Requirements are **incremental over
the epic**; architecture is **detailed for this feature only**.

---

## 0. Provenance and gate basis (read this first)

This feature has **no feature-level `research-report.md`**, unlike siblings
E34-F02 through E34-F09. That absence is deliberate, not an unmet gate:

- The **capability basis** is the epic research report's Capability map row for
  this feature — [research-report.md § Capability map](../research-report.md#capability-map):

  > `Harness-aware prompt rendering | F01 feature; internal/cli/commands/next.go; prompt-rendering tests | EXTEND | Preserve the dispatch/render handshake and add variants without rewriting every prompt.`

  The epic report cites *"F01 feature"* as F01's brownfield evidence, where the
  other rows cite *"F0N research report"*. The epic-level capability analysis
  for F01 was performed from `feature.md` on purpose.
- The **authorization** is Shark feature note #2837 (2026-08-31): *"Team-lead
  directed: resume E34-F01 build using feature.md+decisions.md as spec of
  record, per 2026-08-31 instruction superseding the 2026-08-25 planning-only
  hold."* Note #2838 records the closure of the three open non-decisions that
  preceded it.
- The **design basis** is [`decisions.md`](./decisions.md), including its
  2026-08-31 addendum closing the three non-decisions.

Tier is declared **STANDARD** (Shark reports `size_label: L`; no COMPLEXITY
note exists on the feature). Cross-feature and cross-epic sections below are
present and explicitly empty — see §7 and §8 — rather than silently omitted.

---

## 1. Context (referenced, not restated)

| Source | Use |
|---|---|
| [Epic PRD](../epic.md) § Scope → In scope | *"Canonical workflow, prompt-rendering, validation, parity-test, manifest, and index changes"* — the epic scope line this feature traces to. |
| [Epic PRD](../epic.md) § Scope → Out of scope | *"…coding standards, model selection…"* are out of Shark's canonical policy. This is the epic-level source of the client-owns-model-selection boundary in `decisions.md` Decision 1. |
| [Epic architecture](../architecture.md) § Component boundaries | System-level boundary this feature stays inside. F01 adds no new I-## payload. |
| [feature.md](./feature.md) | REQ-F-001, REQ-F-002, REQ-NF-001 — the requirements this spec refines. |
| [decisions.md](./decisions.md) | D1–D8 plus the 2026-08-31 addendum. |
| CLAUDE.md + `.claude/rules/architecture.md` | Command → service → repository layering; migration checklist in `.claude/rules/database-critical.md`. |

### Capabilities reused, not re-implemented

Per the epic Capability map's **EXTEND** disposition, this feature does **not**
re-implement any of the following. It only feeds them new data:

| Existing capability | Location | This feature's relationship |
|---|---|---|
| Go `text/template` prompt rendering with a curated FuncMap | `internal/templates/orchestrator_renderer.go` (`Render`, `orchestratorFuncs`) | **Reuse as-is.** `{{if}}` already exists; no new template language. |
| Conditional-branch precedent in prompts | `orchestratorFuncs`: `isSimple` / `isStandard` / `isComplex` | **Extend by analogy** — add harness predicates in the same shape. |
| Placeholder generation and alias augmentation | `internal/cli/commands/next.go:562-572`; `templates.AugmentPlaceholderAliases` | **Extend** — decorate the existing `vars` map. |
| Claim / session lease | `internal/services/claim_service.go`, `internal/repository/claim/claim_repository.go`, `entity_claims` table | **Extend** — three nullable columns; no new table, no new lifecycle. |
| Harness family vocabulary | `internal/runner/dispatcher.go` `Dispatcher.Name()` → `"claude"`, `"codex"` | **Reuse** as the canonical harness-type token set. |
| Prompt assembly and digesting | `assembleDispatchPrompt`, `PromptSHA256`/`PromptBytes` in `NextResponse` | **Untouched.** Harness metadata enters before assembly, so digests stay correct with no code change. |

Explicitly **not** re-implemented here: a Shark-side harness/model matrix, a
`model.class`/`model.effort` client-routing contract, a second config schema, or
any change to `agent_type` on the wire (all deferred — see §2.4 and
`decisions.md` addendum items 1 and 3).

---

## 2. Requirements (incremental over the epic)

### 2.1 Functional requirements

**REQ-F-001 — Harness identity is capturable at claim time.**
Refines `feature.md` REQ-F-001. `shark claim` accepts optional
`--harness`, `--harness-version`, and `--harness-model` values and persists
them on the entity's claim row so they outlive the claiming process.

- Values are optional and independently settable; supplying none is valid.
- Persisted values are scoped to the claim (one claim per entity, already
  enforced by `UNIQUE(entity_type, entity_key)`), so releasing the claim ends
  the metadata's validity for that entity.
- `--harness` is normalized (trimmed, lowercased). `--harness-version` and
  `--harness-model` are trimmed opaque strings.

**REQ-F-002 — Harness identity is resolvable at render time with a defined
precedence.**
When Shark renders a workflow prompt for an entity, harness metadata is
resolved from, highest precedence first:

1. explicit flags on the rendering command (`shark next --harness=…`,
   `--harness-version=…`, `--harness-model=…`);
2. the entity's active claim row (REQ-F-001);
3. environment: `SHARK_HARNESS`, `SHARK_HARNESS_VERSION`,
   `SHARK_HARNESS_MODEL`;
4. unset — every harness variable resolves to the empty string.

Precedence is evaluated **per field**, not per source, so a claim may supply
the type while a flag overrides only the model.

**REQ-F-003 — Harness metadata is exposed to prompt rendering.**
Resolved metadata is injected into the placeholder map consumed by both the
instruction template and the inlined agent body, under the keys
`harness`, `harness_version`, and `harness_model`, before the action is
populated.

**REQ-F-004 — Prompt authors can branch on harness metadata.**
Workflow prompts branch using the existing Go `text/template` `{{if}}`
construct over those variables. Three predicate helpers are added to the
orchestrator FuncMap, mirroring the existing tier helpers:
`isHarness <name>`, `isClaude`, `isCodex`.

**REQ-F-005 — Harness metadata is observable on the dispatch response.**
`shark next --json` reports the resolved values on `NextResponse` as
`harness`, `harness_version`, and `harness_model`, each `omitempty`, so a
prompt audit can verify rendered output against the metadata actually used.

**REQ-F-006 — `shark run` renders identically.**
The `shark run` controller path resolves and injects harness metadata through
the same resolver as `shark next`, so a prompt renders byte-identically under
both surfaces for the same entity and harness inputs. `shark run` accepts the
same three override flags as `shark next`, so all four precedence tiers of
REQ-F-002 are reachable from both surfaces.

### 2.2 Non-functional requirements

**REQ-NF-001 — Backward compatibility (additive only).**
Refines `feature.md` REQ-NF-001 and implements `decisions.md` addendum item 3.
No existing `NextResponse` field is removed, renamed, or demoted —
`agent_type` in particular is retained unchanged. A prompt containing no
harness branch renders byte-identically before and after this feature. A
database with no harness columns populated behaves exactly as today.

**REQ-NF-002 — Absent metadata never fails a render.**
Unset harness fields are injected as **empty strings, with the map keys always
present**, so `{{if isClaude .harness}}` is simply false and the generic branch
renders. No render path returns an error, and no dispatch is blocked, because
harness metadata is absent or unrecognized. This is the "defined fallback
behavior" required by `feature.md` REQ-F-001. The always-present requirement is
not cosmetic — omitting a key makes `text/template` fail the render outright;
see D-F01-07 for the measured behavior.

**REQ-NF-003 — No new per-render I/O on the hot path.**
Resolution performs at most one additional indexed read
(`idx_entity_claims_key`) per `shark next` invocation, reusing the already-open
global DB handle. No new file reads, no new config file.

**REQ-NF-004 — Input sanitization.**
Harness fields follow `.claude/rules/go/input-sanitization.md`: trimmed,
length-capped at 100 characters each, stored via parameterized queries, and
quoted with `%q` in any error message. They are treated as opaque data and are
never interpolated into SQL or shell commands.

### 2.3 Acceptance criteria

| ID | Given | When | Then |
|---|---|---|---|
| AC-01 | An unclaimed entity | `shark claim <key> --harness=claude --harness-version=2.1.0 --harness-model=opus` | The claim succeeds and `shark claims --json` reports the three values on that claim. |
| AC-02 | An entity claimed as in AC-01, whose step's prompt contains `{{if isClaude .harness}}A{{else}}B{{end}}` | `shark next <key> --json` | `prompt` contains `A` and not `B`. |
| AC-03 | The same entity and prompt, claimed with `--harness=codex` | `shark next <key> --json` | `prompt` contains `B` and not `A`. |
| AC-04 | An entity with **no** claim and no harness env vars | `shark next <key> --json` | The command exits 0, `prompt` contains `B` (generic branch), and `harness` is absent from the JSON. |
| AC-05 | An entity claimed with `--harness=codex`, and `SHARK_HARNESS=claude` exported | `shark next <key> --harness=claude --json` | `harness` is `claude` — flag beats claim beats env. |
| AC-06 | An entity claimed with `--harness=codex` only (no version), `SHARK_HARNESS_VERSION=9.9` exported | `shark next <key> --json` | `harness` is `codex` and `harness_version` is `9.9` — precedence resolves per field. |
| AC-07 | Any existing workflow prompt containing no harness branch | `shark next <key> --json` before and after this change | `prompt_sha256` is identical. |
| AC-08 | The same entity, claim, and step | `shark next <key> --json` and the `shark run` controller's rendered prompt | The two rendered prompts are byte-identical — asserted for all three precedence tiers, including identical `--harness` flags passed to each command, not only the claim/env-sourced case. |
| AC-09 | `shark claim <key> --harness="  CLAUDE  "` | `shark claims --json` | `harness` is `claude` (trimmed, lowercased). |
| AC-10 | `shark claim <key> --harness=<101 chars>` | The command | Fails with a validation error naming the field and quoting the input; no claim row is written. |
| AC-11 | A database at schema version 34 with existing claim rows | Any shark command | Migration to 35 succeeds, existing rows are preserved, and their harness columns are `NULL`. |
| AC-12 | A zero `HarnessIdentity` | `Vars()` | Returns all three keys `harness`, `harness_version`, `harness_model`, each mapped to `""` — no key is omitted (D-F01-07). |
| AC-13 | An entity with no claim and no harness env vars, and a step prompt using the bare form `{{.harness}}` | `shark next <key> --json` | The command exits 0 and `prompt` contains neither the literal `<no value>` nor a render error; the substitution is empty. |

### 2.4 Out of scope for this feature

Beyond `feature.md` § Out of Scope, which stands:

1. **The `model.class` / `model.effort` client-routing contract**
   (`decisions.md` D4–D6). This feature carries harness identity *into* Shark;
   it does not build the routing metadata Shark returns *out*. Deferred to a
   follow-up feature.
2. **Removing or demoting `agent_type`** (`decisions.md` Implications item 4).
   Explicitly deferred per addendum item 3 — removal requires a coordinated
   host migration that REQ-NF-001 forbids here.
3. **`model.prompt_profile`.** Optional and unimplemented in v1, per addendum
   item 2.
4. **A harness capability registry** (which tools/features each harness
   supports). Only identity — type, version, model — is captured. A capability
   model is speculative until at least two prompts need it.
5. **Rewriting any shipped workflow prompt to use a harness branch.** The
   mechanism ships with test fixtures only; adoption is incremental, per
   `feature.md` § Out of Scope item 1.

### 2.5 Durable unresolved decisions

**None material.** Per `skills/question-management/SKILL.md`, a Q### is created
only for a *material* unresolved decision. The three formerly open
non-decisions were closed in the `decisions.md` 2026-08-31 addendum and are
reflected above (§2.4 items 1–3). The one architecture choice this spec had to
settle itself — persist on the claim versus pass per-invocation — is **decided**
in D-F01-01 below with rationale, not deferred. No `TBD` remains in any section,
and no section relies on absence-of-TBD as a substitute for a decision.

---

## 3. Architecture

### 3.1 Data model changes

**Migration: schema version 34 → 35.** Follows the checklist in
`.claude/rules/database-critical.md`.

- New function `migrateEntityClaimsAddHarness(db *sql.DB)` in
  `internal/db/db.go`, called from `runMigrations()` immediately after the
  existing `migrateEntityClaimsTable(db)` call (currently `internal/db/db.go`
  line ~1107).
- Adds three nullable `TEXT` columns to `entity_claims`:
  `harness`, `harness_version`, `harness_model`. Uses the
  `ALTER TABLE … ADD COLUMN` + `PRAGMA table_info` guard pattern already used
  by `migrateSprintAssignmentsAddSprintOrder` (SQLite has no
  `ADD COLUMN IF NOT EXISTS`), so the migration is safe to rerun.
- Bump `CurrentSchemaVersion` from `34` to `35` and add the version-history
  comment line (`internal/db/db.go` line ~487). **This is the key step** —
  `ApplySchemaIfNeeded` uses it to run the migration under
  `skip_migrations: true`.
- No index is added: lookups go through the existing
  `idx_entity_claims_key` on `(entity_type, entity_key)`.
- No backfill. `NULL` is the correct "unknown harness" value and maps to the
  empty string in Go (AC-11).

Model change in `internal/models/entity_claim.go`:

- `EntityClaim` gains `Harness`, `HarnessVersion`, `HarnessModel string` with
  `json:"harness,omitempty"` / `harness_version,omitempty` /
  `harness_model,omitempty`.
- `Validate()` gains length checks only (≤100 chars each, per REQ-NF-004).
  Empty is valid. No enum allowlist — see D-F01-03.

### 3.2 Interface contracts

**`NextResponse` (wire, `internal/cli/commands/next.go:143`)** — three additive
fields, placed after `Effort` and before `Prompt`:

| Field | JSON | Notes |
|---|---|---|
| `Harness` | `harness,omitempty` | Resolved harness type. |
| `HarnessVersion` | `harness_version,omitempty` | Resolved version. |
| `HarnessModel` | `harness_model,omitempty` | Resolved model. |

All `omitempty`, so a run with no harness metadata emits a JSON object
byte-identical to today's (REQ-NF-001).

**`ClaimInput` (`internal/services/claim_service.go:84`)** — three additive
string fields `Harness`, `HarnessVersion`, `HarnessModel`, carried into the
`models.EntityClaim` constructed in `ClaimService.Claim`.

**New resolver type** in `internal/services/harness_service.go` (new file):

- `type HarnessIdentity struct { Type, Version, Model string }`
- `func (i HarnessIdentity) IsZero() bool`
- `func (i HarnessIdentity) Vars() map[string]string` — returns **all three**
  `harness*` keys **unconditionally**, using the empty string for any unset
  field. Keys are never omitted. This is load-bearing, not stylistic — see
  D-F01-07.
- `type HarnessResolver struct { claims ClaimReader }` with
  `func (r *HarnessResolver) Resolve(ctx context.Context, entityType, entityKey string, override HarnessIdentity) (HarnessIdentity, error)` —
  implements the per-field precedence of REQ-F-002. Errors reading the claim
  are logged and swallowed to the zero identity (REQ-NF-002); they never fail a
  render.
- `ClaimReader` is a one-method consumer-side interface
  (`Get(ctx, entityType, entityKey string) (*models.EntityClaim, error)`),
  satisfied as-is by the existing `claim.Repository.Get`
  (`internal/repository/claim/claim_repository.go:58`), per the
  accept-interfaces rule in `.claude/rules/go/patterns.md`.

**Template FuncMap additions** in `orchestratorFuncs()`
(`internal/templates/orchestrator_renderer.go:459`), in the same shape as the
existing `isSimple`/`isStandard`/`isComplex` block:

- `isHarness(want, got string) bool` — case-insensitive equality.
- `isClaude(got string) bool`, `isCodex(got string) bool` — convenience
  wrappers, matching the tier helpers' precedent.

Authors write `{{if isClaude .harness}}…{{else}}…{{end}}`. **No new template
syntax is introduced** — this closes `feature.md` § Out of Scope item 2 ("the
implementation can choose the simplest supported conditional rendering
approach") in favor of the engine already in the codebase.

### 3.3 Component changes — files to modify or create

| File | Change |
|---|---|
| `internal/db/db.go` | Add `migrateEntityClaimsAddHarness`; call it from `runMigrations()` after `migrateEntityClaimsTable`; bump `CurrentSchemaVersion` 34 → 35 with a history comment. |
| `internal/models/entity_claim.go` | Add three fields; extend `Validate()` with length caps. |
| `internal/repository/claim/claim_repository.go` | Include the three columns in the `Claim` insert/upsert (line 35), in the `scanOne` column list (line 175), and in the `Get`/`List`/`getByID` select lists (lines 58/142/163). Parameterized placeholders only. |
| `internal/services/claim_service.go` | Add three fields to `ClaimInput`; carry them onto the `models.EntityClaim` built in `Claim`. |
| `internal/services/harness_service.go` | **New.** `HarnessIdentity`, `ClaimReader`, `HarnessResolver`, per-field precedence, env-var reads. |
| `internal/cli/services_global.go` | Add `GetHarnessResolver()` following the existing lazy global-accessor pattern. |
| `internal/cli/commands/claim.go` | Register `--harness`, `--harness-version`, `--harness-model` on `claimCmd` (`init()`, line ~93); read and pass them in `runClaim` (line ~114). |
| `internal/cli/commands/next.go` | Add three `NextResponse` fields (line ~151); register the same three override flags on `newNextCommand()`; in `resolveEntity`, resolve and merge harness vars into `vars` immediately after `templates.AugmentPlaceholderAliases(vars)` (line 572) and **before** `GetStatusActionPopulated` (line 585); set the three response fields alongside `resp.Effort` in `applyWireAction` (line ~1015). |
| `internal/runner/controller.go` | Add an optional `HarnessResolver` dependency to the controller's config struct (alongside the existing `placeholders` field, line ~243). Merge the resolved harness vars into `vars` after `GeneratePlaceholders` (line ~450) and before `GetStatusActionPopulated` (line ~468), via the same resolver, satisfying REQ-F-006. When the resolver is nil, inject the zero identity's three empty keys — never an absent key (D-F01-07). |
| `internal/cli/commands/run.go` | Register the same three override flags on `runCmd` (`init()`, line ~70, beside `--workdir`/`--worktree`) and pass them into the controller as the resolver's override identity. **Required for REQ-F-006/AC-08**: without this, precedence tier 1 (flags) has no entry point under `shark run`, so a host passing `--harness` to `next` but not `run` would get two different prompts from the same entity. |
| `internal/templates/orchestrator_renderer.go` | Add `isHarness`, `isClaude`, `isCodex` to `orchestratorFuncs()`. |
| `docs/cli-reference/*.md` | Document the new flags on `claim` and `next`, and the new response fields. |

Tests to add (paths follow existing convention):
`internal/db/db_test.go` (migration/idempotence, AC-11),
`internal/models/entity_claim_test.go` (validation, AC-09/AC-10),
`internal/services/harness_service_test.go` (precedence matrix, AC-05/AC-06,
`Vars()` key completeness AC-12, error-swallowing per REQ-NF-002),
`internal/cli/commands/next_harness_test.go` (**new**, mocked services per
`.claude/rules/cli/commands.md` — AC-02/AC-03/AC-04/AC-07/AC-13),
`internal/templates/orchestrator_renderer_test.go` (the three helpers, plus a
regression case executing a harness-branching template against a map missing
the harness keys, asserting the failure mode recorded in D-F01-07),
plus an AC-08 parity assertion across all precedence tiers alongside the
existing `internal/cli/commands/next_golden_test.go` fixtures.

### 3.4 Integration with existing code

The decoration point is the **placeholder map**, and only the placeholder map.

```
shark claim --harness=…                shark next <key>
        │                                     │
        ▼                                     ▼
  ClaimService.Claim              resolveEntity (next.go:562-572)
        │                              GeneratePlaceholders → vars
        ▼                                     │
  entity_claims row  ───────────────►  HarnessResolver.Resolve
  (harness, version, model)            (flags → claim → env → zero)
                                              │
                                              ▼
                                   vars["harness"|"harness_version"|"harness_model"]
                                              │
                        ┌─────────────────────┴─────────────────────┐
                        ▼                                           ▼
        GetStatusActionPopulated                        assembleDispatchPrompt
        → PopulateTemplate → OrchestratorRenderer.Render   → attachAgentBody
          (text/template; {{if isClaude .harness}})          (RenderAndLintAgentBody)
                        └─────────────────────┬─────────────────────┘
                                              ▼
                                  PromptSHA256 / PromptBytes
                                     (unchanged, still correct)
```

Because the metadata is merged into `vars` *before* both consumers, the
instruction template and the inlined agent body both gain the branch capability
from one change. `assembleDispatchPrompt` and the SHA-256 digest need no
modification, and `annotateUnresolvedPlaceholders` continues to work: the
harness keys use the `{placeholder}` / `.field` form, not the `<token>` form it
scans for.

### 3.5 Key technical decisions

**D-F01-01 — Persist harness identity on the claim; do not pass it per
invocation only.**
`feature.md` REQ-F-001 requires the metadata to be "part of the claim/dispatch
context" and to "survive long enough to influence prompt rendering for the
claimed work." The claim is already Shark's lease and already the one
per-entity in-flight record (`route-based-workflow.md` §4), which makes it the
correct owner. Flags and env vars remain as overrides so a host that never
claims (or a one-off render) still works.
*Trade-off:* costs a schema migration. Accepted, because the alternative —
requiring every render call to re-supply identity — makes correctness depend on
every caller remembering, and `shark run` has no natural place to receive it.

**D-F01-02 — Reuse Go `text/template`; introduce no custom `if` syntax.**
`feature.md` § Out of Scope item 2 left the syntax open and mentioned a
possible custom `if` mechanism. The renderer already parses prompts as Go
templates with a curated FuncMap
(`internal/templates/orchestrator_renderer.go:221`), and
`isSimple`/`isStandard`/`isComplex` are an exact working precedent for
predicate-driven branching. Adding a parser would be new, unproven complexity
for a capability the codebase already ships. **Appropriate, Proven, Simple.**

**D-F01-03 — Harness type is an open, normalized string, not a DB enum.**
`.claude/rules/go/input-sanitization.md` prescribes enum allowlists for
fixed-vocabulary fields, and `"claude"`/`"codex"` are the known values
(`Dispatcher.Name()`). This spec deliberately **deviates**: the whole purpose is
to accept harnesses Shark does not yet know about, and memory item
`feedback_entity_type_check_constraints` records the cost of DB CHECK
constraints on extensible vocabularies. Enforcement is therefore normalization
plus a length cap at the app layer only — no CHECK constraint. `isClaude` /
`isCodex` are conveniences over the open string, not a closed set.

**D-F01-04 — Per-field precedence, not per-source.**
A host may claim once with its type and version, then vary only the model per
step. Resolving each field independently makes that work without re-claiming.
*Trade-off:* slightly more resolver logic than "first non-empty source wins";
justified by the concrete dispatch pattern.

**D-F01-05 — Resolution failures degrade to the generic prompt.**
A claim-read error yields the zero identity and a logged warning, never a
render error. This mirrors the existing graceful-degradation posture at
`next.go:585-601` (unknown status → `pause` rather than crash) and
`PopulateTemplate` (render error → warn and degrade), and it is what
REQ-NF-002 requires.

**D-F01-07 — The harness keys must always be present in the placeholder map,
never omitted.**
Verified empirically against Go 1.x `text/template` with a
`map[string]string` payload (the exact shape `OrchestratorRenderer.Render`
executes with, `internal/templates/orchestrator_renderer.go:446`):

| Template | Key absent | Key present but empty |
|---|---|---|
| `{{if isClaude .harness}}A{{else}}B{{end}}` | **execution error**: `at <.harness>: invalid value; expected string` | renders `B` |
| `{{.harness}}` | renders the literal `<no value>` | renders `` |
| `{{if eq .harness "claude"}}…` | renders `B` | renders `B` |

An absent key therefore does **not** degrade gracefully. It fails the typed
helper, which makes `Render` return an error, which makes `PopulateTemplate`
log and return `""` (`internal/config/action/orchestrator.go:207-218`), which
trips the empty-instruction guard at `internal/cli/commands/next.go:614` and
**fails the dispatch outright**. The bare-field form is quieter but worse: it
leaks the literal `<no value>` into the prompt, and
`annotateUnresolvedPlaceholders` will not flag it because it scans for the
`<token>` form.

Emitting all three keys unconditionally makes REQ-NF-002 and AC-04 hold for
every template form. AC-12 locks this behavior down so a later "tidy up the
empty values" refactor cannot silently reintroduce the failure.

**D-F01-06 — Additive wire changes only; `agent_type` untouched.**
Implements `decisions.md` addendum item 3 and satisfies REQ-NF-001. The
narrower client contract of `decisions.md` D8 is a real goal but requires a
coordinated host migration; it does not belong in the feature that merely
introduces harness identity.

---

## 4. Security

Harness fields are attacker-influenceable only by whoever can run `shark claim`
on the project — the same trust boundary as `--by` and `--session`, which are
already free strings. They are stored via parameterized queries, length-capped,
quoted with `%q` in errors, and never used to select a file path, build a shell
command, or resolve a template name. They **are** interpolated into rendered
prompt text when a template references them, which is the feature's purpose;
the 100-character cap bounds that surface, and no privileged instruction is
gated on the value.

---

## 5. Operations and observability

`shark next --json` reports the resolved identity (REQ-F-005), and
`shark claims` shows what each claim recorded — together these make a mis-set
harness diagnosable from the CLI without reading the database. Reusing the
existing OTel span in `runNext` (line ~399), harness type is added as a span
attribute; version and model are not, to avoid unbounded attribute cardinality.

---

## 6. Verification

Every AC in §2.3 maps to a named test in §3.3. AC-07 (byte-identical prompts
for unbranched workflows) and AC-08 (`next`/`run` parity) are the two
regression guards that make REQ-NF-001 and REQ-F-006 falsifiable rather than
aspirational; both belong in the golden-fixture suite so a future prompt or
renderer change cannot quietly break them. Detailed case decomposition belongs
to `test-plan.md`, produced by the next workflow step.

---

## 7. Cross-feature interactions

**None.** Verified against
[`E34-interaction-map.md`](../E34-interaction-map.md) in full: the map registers
I-01 through I-05, whose producers and consumers are E34-F02, F03, F05, F06,
F07, F08, and F09. E34-F01 appears in no producer or consumer column and owns
no I-## row, so this feature produces and consumes no cross-feature payload.

No I-## ID is invented here. All contracts changed by this feature
(`ClaimInput`, `NextResponse`, `HarnessResolver`, the FuncMap helpers) are
internal to Shark's own codebase and use local names, as the prompt's rules
permit.

## 8. Cross-epic integrations

**None.** Verified against [`E34-cross-epic-map.md`](../E34-cross-epic-map.md)
and [`docs/product/cross-epic-integration-map.md`](../../../product/cross-epic-integration-map.md).
Both register exactly one row for E34 — **X-14**, owned by
**E34-F09 Override Drift Visibility and WWGM Reconciliation**, status
`proposed`. E34-F01 neither produces, consumes, nor validates X-14, and no
other X-## row touches this feature. No X-## ID is invented here.

---

*Basis: `feature.md`, `decisions.md` (incl. 2026-08-31 addendum), epic
`research-report.md` § Capability map, Shark feature notes #2837/#2838.*
