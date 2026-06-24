# Epic Acceptance Plan (UAT): E32 — Shark 2.0 Single-Artifact Consolidation

**Epic**: E32
**Date**: 2026-06-22
**Author**: architect

> This plan is the **epic-level acceptance gate**. Each scenario describes *what
> to verify*, not *how to implement*. Every scenario traces to one or more
> success criteria (SC1–SC8) and UAT scenarios (UAT-1–UAT-8) in `epic.md` §2/§6.
> E32 is acceptable when every Must-Pass scenario passes and no orphaned success
> criterion remains.

---

## 1. Coverage Matrix (success criteria → scenarios)

| Success Criterion (epic §2) | Epic UAT | Scenario(s) here |
|---|---|---|
| SC1 — fresh project runs from `shark-data/` + trigger only | UAT-1 | A1, A2 |
| SC2 — rendered prompts semantically equivalent pre/post migration | UAT-3 | A3 |
| SC3 — `shark next` returns self-contained prompts (skills inlined) | UAT-2 | A4 |
| SC4 — in-scope skills/agents removed from harness | UAT-5 | A5, A6 |
| SC5 — `shark validate` catches structural defects | UAT-1, UAT-6 | A2, A7 |
| SC6 — override survives upgrade and wins at render | UAT-4 | A8 |
| SC7 — legacy paths removed in final phase | UAT-7 | A9 |
| SC8 — no orphaned `_extracted/` sidecars as shipped craft | UAT-8 | A10 |
| (instrumentation, epic F07) | — | A11 |
| (cross-feature integration) | — | I1, I2, I3 |

No success criterion is left without a scenario; no scenario exists without a
criterion (except explicitly-labelled integration/perf scenarios).

---

## 2. Acceptance Scenarios

Each scenario: **Given / When / Then**, plus **Verify** (the observable evidence)
and **Trace**.

### A1 — Fresh-project bootstrap, end to end *(Must-Pass)*
- **Given** an empty temp directory with the `shark` binary on PATH.
- **When** the operator runs, in order: `shark init`, `shark validate`,
  `shark create epic "test"`, `shark next <new-epic-key> --json`.
- **Then** every command exits 0 and the final command returns a rendered prompt.
- **Verify**: each exit code is 0; `shark init` creates a `shark-data/` tree with
  non-empty `prompts/`, `skills/`, `agents/`, `workflow/`; the final JSON has a
  non-empty `prompt` field and a valid `action`.
- **Trace**: SC1, UAT-1.

### A2 — `shark init` lays down a complete, valid tree *(Must-Pass)*
- **Given** a fresh `shark init` from A1.
- **When** the operator runs `shark validate`.
- **Then** validation passes with zero error-level issues against the
  freshly-laid-down canonical tree.
- **Verify**: `ValidationReport` reports no `IssueLevelError`; `prompts/` is not
  merely a `.gitkeep` (the F04 embedded-corpus gap is closed); `workflow/`
  contains every shipped entity YAML.
- **Trace**: SC1, SC5, UAT-1; closes `research.md` Risk 1.

### A3 — Behavioral parity with the legacy stack *(Must-Pass)*
- **Given** a representative real project entity that has both a legacy rendered
  prompt and a post-migration rendered prompt.
- **When** the operator diffs `shark next <entity> --json | jq -r .prompt` against
  the pre-migration rendered prompt, and runs the entity through `/run`.
- **Then** the only differences are inlining (skill bodies now present); no
  instructions, gates, or referenced artifacts changed, and `/run` produces the
  same artifacts and behavior.
- **Verify**: textual diff limited to inlined skill/agent content; the set of
  gates and referenced artifact paths is identical; the post-migration `/run`
  output set matches the pre-migration baseline.
- **Trace**: SC2, UAT-3.

### A4 — Self-contained rendered prompt (skills inlined) *(Must-Pass)*
- **Given** a project initialized with `shark-data/`.
- **When** the operator runs `shark next <entity> --json | jq -r .prompt`.
- **Then** the output contains the full body of every skill the prompt references
  and the agent persona — no unresolved `LOAD:`/`{{include:}}`/`{{augment:}}`
  directives remain.
- **Verify**: grep the rendered prompt for residual `LOAD:`, `{{include:`,
  `{{augment:` — none found; a referenced skill's distinctive body text is present
  inline; the agent persona text is present.
- **Trace**: SC3, UAT-2; confirms ADR-4 inline-agent model.

### A5 — Harness resolves from `shark-data/` only — skills *(Must-Pass)*
- **Given** an in-scope harness skill directory (e.g. `~/.claude/skills/quality/`)
  is moved aside.
- **When** a `/run` workflow that uses that skill executes.
- **Then** it still works, resolving the skill body from `shark-data/skills/`.
- **Verify**: the run completes; the rendered prompt still contains the skill
  body; no "skill not found" error.
- **Trace**: SC4, UAT-5.

### A6 — In-scope skills and agents deleted from harness *(Must-Pass)*
- **Given** F05 cutover is complete.
- **When** the operator inspects `~/.claude/skills/` and `~/.claude/agents/`.
- **Then** the nine in-scope skills (`specification-writing`, `quality`,
  `architecture`, `research`, `implementation`, `test-driven-development`,
  `assessment`, `uat`, `debugging`), the `orchestration` skill, and the nine
  in-scope agents are absent; only helper agents and untouched skills remain;
  `shark/SKILL.md` is the tiny trigger.
- **Verify**: directory listings confirm absence; `shark/SKILL.md` is materially
  smaller (~trigger-sized); out-of-scope skills (epic §3 list) still present.
- **Trace**: SC4, UAT-5.

### A7 — Validation rejects structural defects *(Must-Pass)*
- **Given** a deliberately broken canonical tree in three independent variants:
  (a) a skill that references a shark status name, (b) a dangling `{{include:}}`
  path, (c) a workflow YAML naming a missing `agent_type`/`prompt`.
- **When** the operator runs `shark validate` against each variant.
- **Then** each run fails and names the offending file and the specific defect.
- **Verify**: non-zero outcome / error-level issue per variant; the report cites
  the file path and defect class; per-file accumulation means all defects in a
  variant are reported, not just the first.
- **Trace**: SC5, UAT-6.

### A8 — Override survives upgrade and wins at render *(Must-Pass)*
- **Given** a customization at
  `shark-data/overrides/skills/quality/workflows/review-code.md`.
- **When** the operator runs `shark upgrade` (and `--dry-run` first).
- **Then** the override file is byte-unchanged after upgrade and continues to win
  over the canonical default at render time.
- **Verify**: file hash identical before/after `shark upgrade`; `DiffSummary`
  reports it under `SkippedOverrides`; `shark next` renders the override content,
  not the default.
- **Trace**: SC6, UAT-4; enforces ADR-3 replace-only + upgrade-skip invariant.

### A9 — Legacy paths retired *(Must-Pass)*
- **Given** the F06 cleanup phase is complete.
- **When** the operator runs `grep -r "shark-templates" cmd/ internal/` and
  attempts to load a legacy `.sharkworkflow.json`.
- **Then** the grep returns nothing and the legacy load is refused with a
  deprecation error.
- **Verify**: empty grep result; `shark-templates/` directory removed; the
  `findTemplateDir()` Pass-2 fallback branch is gone; legacy JSON load returns a
  clear deprecation error (not a silent fallthrough).
- **Trace**: SC7, UAT-7; gated on A2/A3/A4 passing first (do not remove the
  fallback before F04 coverage is proven — `research.md` Risk 2).

### A10 — Canonical tree is clean (no orphaned scaffolding) *(Must-Pass)*
- **Given** the epic is ready to close.
- **When** the operator inspects `shark-data/skills/` (and the embedded
  `default_data/skills/`).
- **Then** no `_extracted/` scaffolding sidecars remain as *shipped* craft — each
  is consumed into prompts/workflow or explicitly marked non-shipped.
- **Verify**: no `_extracted/` directories ship as skill content; the
  specification-writing purity fix (2 stray `shark status advance` lines moved per
  `research.md` Risk 5) is in place; `_partials_inventory.md` exists.
- **Trace**: SC8, UAT-8.

### A11 — Dispatch instrumentation observable *(Should-Pass, F07)*
- **Given** observability enabled with `exporter: file_jsonl`.
- **When** the operator runs `shark next <entity>` several times.
- **Then** dispatch events appear in `shark-data/.stats/events.jsonl` with the
  expected fields and unresolved-placeholder detection works.
- **Verify**: each `shark next` appends a JSONL line carrying prompt byte size,
  unresolved-placeholder count, `agent_type`, and `action`; an entity whose prompt
  contains an unresolved `<…>` placeholder surfaces a non-zero count and a stderr
  warning; `file_jsonl` is accepted by `ObservabilityConfig.Exporter` validation.
- **Trace**: epic F07 / SC-adjacent instrumentation; non-blocking for F01–F06.

---

## 3. Cross-Feature Integration Scenarios

### I1 — Full critical-path chain (F01→F04→F05→F06) *(Must-Pass)*
- **Given** all critical-path features merged.
- **When** the operator performs the full lifecycle on a clean checkout:
  build binary → `shark init` in a fresh dir → `shark validate` →
  create+drive a real entity through `/run` → `shark upgrade` (no-op diff) →
  confirm legacy paths gone.
- **Then** the chain completes with no fallback to `shark-templates/`, no
  unresolved directives, no override loss, and no legacy-path references.
- **Verify**: combines A1–A4, A8, A9 in one continuous run; the run never touches
  `shark-templates/` (confirm via the removed fallback in I1's post-F06 state).
- **Trace**: SC1–SC4, SC6, SC7.

### I2 — Override + upgrade + render interaction *(Must-Pass)*
- **Given** an override placed, then a simulated new embedded default for the same
  path (newer canonical content).
- **When** `shark upgrade` runs, then `shark next` renders the affected entity.
- **Then** the default is updated on disk *except* under `overrides/`, and the
  render uses the override (replace-only), proving the two mechanisms compose
  without merge drift.
- **Verify**: default file content changes under `prompts/`/`skills/`; the
  override file is untouched; rendered output reflects the override.
- **Trace**: SC6 + ADR-3; guards the epic's hard "replace, never merge" rule.

### I3 — Multi-harness self-containment smoke *(Should-Pass)*
- **Given** a rendered prompt from `shark next` for a representative entity.
- **When** the prompt is evaluated for host-independence (no `~/.claude/` runtime
  discovery required).
- **Then** the prompt is fully self-contained: all craft and persona inlined, no
  reference to a host skill/agent path that must be resolved at runtime.
- **Verify**: the prompt carries no instruction requiring the executing host to
  *find* a skill/agent; it would execute identically under a non-Claude harness.
- **Trace**: epic G3/§5 forward-looking benefit; validates ADR-2 portability goal
  at the prompt level (full Codex/Copilot/Gemini execution is a deferred epic).

---

## 4. Performance and Security Considerations

### Performance
- **`shark next` hot path**: A11/F07 file I/O must not measurably slow dispatch.
  Acceptance posture: buffered/append JSONL writes, flush on `Shutdown()`,
  synchronous span processor first for correctness, async only if latency is
  observed (`research.md` Risk 4). Exporter off by default.
- **Render size**: prompts inlining many skills must not blow past the 50KB
  warning threshold without a logged warning (ADR-7). A scenario-adjacent check:
  the largest shipped prompt renders without error and warnings are visible when
  the threshold is crossed.

### Security
- **No new external surface**: no auth, no network endpoints, no PII. Embedded
  content is trusted-source; no acceptance scenario introduces untrusted input.
- **Self-advancement guard intact**: the inline-agent prompt must not enable an
  agent to advance its own status — `DefaultDisallowedTools` continues to block
  `shark status advance*` / `shark task next-status*`. Acceptance check: a spawned
  agent cannot advance the entity it is working.
- **Override containment**: overrides resolve only within `shark-data/overrides/`;
  no path-escape outside the data root at render time (covered structurally by the
  resolver, asserted as a non-regression in A8/I2).

---

## 5. Exit Gate for E32

E32 is acceptable when:
1. All **Must-Pass** scenarios (A1–A10, I1, I2) pass.
2. Every success criterion SC1–SC8 has at least one passing scenario (matrix §1).
3. No orphaned requirement remains (every epic UAT-1–UAT-8 mapped).
4. `shark validate` is green against a fresh `shark init` (A2) — the F04 gate.
5. Override-preservation invariant holds (A8, I2) — release blocker if violated.
6. F07/F08 **Should-Pass** items (A11, I3) are either passing or explicitly
   deferred to follow-up with a tracked backlog entry (they are non-blocking per
   epic §3).
```
