# `_partials/` inventory — F1.c

Aggregated from sidecars across all in-scope skills. Identifies recurring scaffolding patterns that are candidates for shared `_partials/` files in F4's `shark-data/prompts/_partials/` directory.

> **Status**: Final. All 9 in-scope skills (TDD, debugging, architecture, implementation, research, quality, uat, assessment, specification-writing) processed under F1.b. Sidecars under each skill's `_extracted/` directory.

## How partials are used

A partial is reusable scaffolding that appears identically (or near-identically) across multiple prompt files. F4 lands them at `shark-data/prompts/_partials/_<name>.md` and prompts include them via either:

- `{{template "_name" .}}` for Go-template partial (in-tree mechanism the engine already supports).
- `{{include: prompts/_partials/_name.md}}` for cross-tree inlining (new F2 mechanism).

The choice depends on whether the partial needs variable substitution (template) or pure inlining (include). Most of these partials need variable substitution — use the template form.

---

## Tier 1 — universal partials (appear in ≥6 prompts)

These cover most prompts. Highest leverage to extract.

### `_fetch_entity_context.md`

**What**: Resolve a shark entity ID to its parent context (epic, feature, task) and load all related paths (spec, PRD, related docs).

**Pattern**:
```bash
TASK_JSON=$(shark get $ID --json)
EPIC_ID=$(echo "$TASK_JSON" | jq -r '.epic_id')
FEATURE_ID=$(echo "$TASK_JSON" | jq -r '.feature_id')
TASK_SPEC_PATH=$(echo "$TASK_JSON" | jq -r '.path + "/" + .filename')
ACCEPTANCE_CRITERIA=$(echo "$TASK_JSON" | jq -r '.acceptance_criteria // []')
```

**Sidecars feeding this** (12+):
- quality/qa-testing.md, quality/review-code.md, quality/test-planning.md, quality/validate-design.md
- specification-writing/SKILL, write-epic, write-feature-prd, write-task, refine-task-requirements
- specification-writing/plan/check-ba-docs, check-tech-docs, epic-ba-plan, feature-ba-plan, epic-tech-plan, feature-tech-plan
- uat/SKILL.md (Phase 1 + Phase 4)
- research/consult-related-work
- assessment/SKILL.md (multi-mode)

**Variants**:
- `_fetch_task_context.md` — entity is task (most common form).
- `_fetch_feature_context.md` — entity is feature.
- `_fetch_epic_context.md` — entity is epic.

**Recommendation**: one base `_fetch_entity_context.md` parameterized on entity type via Go-template variable; the three named entry points are thin wrappers that bind the type.

---

### `_resolve_spec_paths.md`

**What**: Compute filesystem paths for spec/PRD/related-doc destinations from epic/feature/task IDs using the project's `docs/plan/...` convention. Including per-domain conventions:

```bash
EPIC_PATH="docs/plan/$EPIC_ID"
FEATURE_PATH="docs/plan/$EPIC_ID/$FEATURE_ID"

# Per-domain (only the relevant ones get used per workflow)
QA_REPORTS_DIR="$FEATURE_PATH/qa_reports"
CODE_REVIEW_DIR="$FEATURE_PATH/code_review"
TEST_PLANS_DIR="$FEATURE_PATH/test_plans"
UAT_DIR="docs/uat/$EPIC_ID"
ARCH_DIR="docs/architecture"

# Timestamped artifact filename
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
ARTIFACT_PATH="$DOMAIN_DIR/${TIMESTAMP}-${TASK_ID}-${SUFFIX}.md"
```

**Sidecars feeding this** (essentially every sidecar — 25+).

**Recommendation**: combine with `_fetch_entity_context.md` since they always co-occur. Keep separate to allow "fetch but don't write" use cases (read-only research workflows).

---

### `_advance.md`

**What**: Advance the entity's status with a rationale note pointing at any artifact the prompt produced.

**Pattern**:
```bash
shark task note add $ID --type <type> --content "<verdict>: <summary> — see <artifact-path>"
shark status advance $ID
```

**Sidecars feeding this** (10+):
- quality/qa-testing (PASS), review-code (PASS), test-planning (PASS), validate-design, validate-tasks
- spec-writing/write-epic, write-feature-prd, write-task, refine-task-requirements
- spec-writing/plan/check-ba-docs, check-tech-docs (after PASS verdict)
- research/project-init, consult-related-work
- implementation (after completion)
- assessment/SKILL (after each mode completes)

---

### `_route_back_on_fail.md`

**What**: Route a task back to development on FAIL, including bug-fix flag, blocker note, and rationale doc reference.

**Pattern**:
```bash
shark task note add $ID --type blocker --content "<verdict>: <reason> — see <artifact-path>"
shark task context set $ID --field bug_fix --value true
shark status set $ID ready_for_development \
  --reason="<reason summary>" \
  --reason-doc="<artifact-path>"
```

**Sidecars feeding this** (6):
- quality/qa-testing (FAIL), review-code (FAIL), test-planning (FAIL), validate-design, validate-tasks
- uat/SKILL.md (Step 5.6 rejected tasks — variant uses ready_for_development too)

---

### `_register_doc.md`

**What**: After a workflow creates / updates a document, register it as a related-doc for its parent entity in shark.

**Pattern**:
```bash
shark related-docs add "<friendly name>" "<path>" --{epic|feature}=$PARENT_ID
```

**Sidecars feeding this** (8+):
- spec-writing/write-epic (after PRD written)
- spec-writing/write-feature-prd
- spec-writing/write-task (for design refs)
- spec-writing/plan/epic-ba-plan, feature-ba-plan, epic-tech-plan, feature-tech-plan (after each planned doc is registered)
- research/project-init (after architecture docs)
- architecture/* (after each design doc)

---

## Tier 2 — high-frequency partials (appear in 4-6 prompts)

### `_codex_invocation.md`

**What**: Build the `codex exec ...` command with model selection, reasoning effort, read-only sandbox, skip-git-repo-check, and inject the workflow-specific codex prompt body.

**Pattern**:
```bash
codex exec -m gpt-5.2-codex -s read-only \
  -c model_reasoning_effort=high \
  --skip-git-repo-check \
  "$(cat <<'PROMPT'
{{ include: prompts/_partials/_codex_<workflow>_prompt.md }}
PROMPT
)"
```

**Sidecars feeding this** (4): qa-testing (Step 5.7), review-code (Step 9.5), test-planning (codex preflight), uat (Step 5.2).

**Variants** (the prompt body, not the invocation framing):
- `_codex_qa_prompt.md` — ENUMERATE per-AC, A/B/C/D categorization, attack/error class enumeration.
- `_codex_review_prompt.md` — DRY/SOLID/standards/idiom/complexity/tests/risk enumeration.
- `_codex_test_planning_prompt.md` — drift detection, AC coverage gaps, prd-completeness.
- `_codex_uat_prompt.md` — evidence + spec cross-check, integration verification, severity-tagged findings.

The invocation framing is shared; the body diverges per workflow. **Recommendation**: ship `_codex_invocation.md` parameterized on the body partial path.

---

### `_codex_red_team_gate.md`

**What**: Codex result must be PASS (or blockers acknowledged) before advancing. Per user memory, this is mandatory at QA and UAT — and the same discipline carries over to code review and test-planning.

**Pattern**:
```bash
if [ "$CODEX_VERDICT" != "PASS" ] && [ -z "$CODEX_BLOCKERS_ACK" ]; then
  echo "GATE FAIL: codex red-team did not return PASS and blockers were not acknowledged."
  VERDICT=FAIL
fi
```

**Sidecars feeding this** (4): qa-testing, review-code, uat, test-planning.

---

### `_loop_guard_check.md`

**What**: Check whether this task has been at this phase before; if ≥1 prior rejection, escalate to user instead of mechanically re-rejecting.

**Pattern**:
```bash
PRIOR=$(shark notes search "" --task $TASK_ID --type rejection --json | \
  jq '[.[] | select(.note.metadata.from_status == "<phase>")] | length')
if [ "$PRIOR" -ge 1 ]; then
  echo "WARNING: Prior rejection at <phase> for $TASK_ID. Escalate to user."
fi
```

**Sidecars feeding this** (1 explicit + universal applicability):
- review-code (explicit)
- All rejection-capable phases inherit this implicitly via the orchestrator's 2-strikes guard.

**Recommendation**: ship as a partial; orchestrator and prompt both reference it.

---

### `_create_tech_debt.md`

**What**: Loop through `non_blockers_to_triage` and create a tech-debt task per item under the parent feature, with fallback to a feature-level note if creation fails.

**Pattern**:
```bash
for finding in $NON_BLOCKERS; do
  NEW_TASK=$(shark create task $EPIC_ID $FEATURE_ID \
    "Tech debt: <summary>" \
    --description="From <source-workflow> of $TASK_ID at $TIMESTAMP. Finding: <text>. Source: $REPORT_PATH" \
    --json | jq -r '.task_id')
done
# Fallback: if shark create task fails, write a single feature-level note
```

**Sidecars feeding this** (3+): review-code (Step 11), and analogous patterns in qa-testing (BUG findings), uat (non-blocking findings via /triage).

---

### `_persist_decision_log.md`

**What**: Store key decisions made during a write/design/refinement workflow as shark notes for traceability.

**Pattern**:
```bash
for decision in "${DECISIONS[@]}"; do
  shark notes add $ID --type decision --content "$decision"
done
```

**Sidecars feeding this** (5): spec-writing/write-epic, write-feature-prd, write-task, refine-task-requirements, plan/* (after planning each doc).

---

### `_check_doc_exists.md`

**What**: Check whether a related doc (PRD, design, architecture, etc.) exists at the expected path or via shark `related-docs list`. If missing, gate or surface.

**Pattern**:
```bash
DOC_PATH=$(shark related-docs list --feature=$FEATURE_ID --json | \
  jq -r '.[] | select(.type=="<doc-type>") | .path' | head -1)
[ -z "$DOC_PATH" ] && DOC_PATH="docs/plan/$EPIC_ID/$FEATURE_ID/<convention-filename>.md"
if [ ! -f "$DOC_PATH" ]; then ...; fi
```

**Sidecars feeding this** (4): spec-writing/plan/check-ba-docs, check-tech-docs, validate-design, uat (Phase 4.2).

---

## Tier 3 — narrower partials (appear in 2-3 prompts)

### `_plan_context_get_set.md`

**What**: PLAN gate context fetch + persist — read `remaining_steps` and `completed_steps` from shark `<entity> context get`, mutate, then `context set` back.

**Pattern**:
```bash
PLAN_CTX=$(shark <entity> context get $ID --field plan --json)
REMAINING=$(echo "$PLAN_CTX" | jq -r '.remaining_steps[]')
COMPLETED=$(echo "$PLAN_CTX" | jq -r '.completed_steps[]')
# ... do work ...
shark <entity> context set $ID --field plan --value "<updated json>"
```

**Sidecars feeding this** (4+): spec-writing/write-epic, write-feature-prd, plan/check-ba-docs, plan/check-tech-docs.

---

### `_create_then_resolve_path.md`

**What**: Create an entity in shark, then `shark get --json` to learn its assigned `path` and `filename` (since shark determines them — never hardcode `prd.md`).

**Pattern**:
```bash
NEW_KEY=$(shark create <entity> ... --json | jq -r '.key')
ENT_JSON=$(shark get $NEW_KEY --json)
ENT_PATH=$(echo "$ENT_JSON" | jq -r '.path')
ENT_FILE=$(echo "$ENT_JSON" | jq -r '.filename')
DEST="$ENT_PATH/$ENT_FILE"
```

**Sidecars feeding this** (3): spec-writing/write-feature-prd, write-task, decompose-epic.

---

### `_consult_prior_art_preflight.md`

**What**: Run the research/consult-related-work workflow before tech planning, take its `prior_art_report_path` output, and feed it as input to tech-plan workflows.

**Pattern**:
```bash
# Host invokes consult-related-work first; result fed into next prompt
PRIOR_ART_PATH=$(shark related-docs list --feature=$FEATURE_ID --json | \
  jq -r '.[] | select(.type=="prior_art") | .path' | head -1)
# If missing, run consult-related-work first
```

**Sidecars feeding this** (2): spec-writing/plan/epic-tech-plan, feature-tech-plan.

---

### `_sibling_enumeration.md`

**What**: Enumerate sibling entities under the same parent (feature's siblings under epic, task's siblings under feature) for cross-reference / integration / ordering context.

**Pattern**:
```bash
SIBLINGS=$(shark list $PARENT_ID --json | \
  jq -r '.[] | select(.key != "'"$ID"'") | {key, title, status, integration_notes}')
```

**Sidecars feeding this** (3): research/consult-related-work, uat (Phase 4 sibling features), spec-writing/write-task (sibling tasks).

---

## Tier 4 — single-use scaffolding (do NOT extract)

These appear in only 1 sidecar. Inline at the prompt site.

- `_frontend_detection.md` — qa-testing only.
- `_first_task_detection.md` — test-planning only ("is this the first task in the feature so PRD-completeness gates apply?").
- `_design_ref_trace.md` — uat only (verifying design refs from PRD appear in task spec; the trace itself is a workflow-level invariant, not a partial-worthy operation).
- `_orchestrator_action_routing.md` — assessment/SKILL.md only (mode discriminator).
- `_test_plan_lookup.md` — qa-testing + test-planning only; small enough to inline.

---

## Stale references (cross-repo — recorded for fix in shark-task-manager)

The actual cleanup happens in `~/projects/shark-task-manager/shark-templates/.sharkworkflow.json` (separate repo, not edited from this branch):

- `discovery` → `research` — 4 refs (lines 162, 179, 200, 217)
- `build` → drop or replace with `devops` — 1 ref (line 973 in `feature_short/ready_to_build/instruction_template` loads)

None of the in-scope skills refer to themselves as `discovery`. The stale refs are purely in the workflow JSON.

---

## Effort calibration (actual vs prompt estimate)

The prompt budgeted 4-6 days for F1.b. **Actual: ~2 hours of orchestration** (one main thread + 8 parallel agents over ~30-45 minutes wall-time, with model effort ~12 hours equivalent if serialized).

Why the prompt over-estimated:

1. The pre-flight audit was correct — most skills were lightly coupled (TDD: 0 hits, debugging: 0, architecture: 4, implementation: 3, research: 35 but concentrated). The prompt's 4-6 day estimate was based on the spec-writing 401-hit claim; actual was 216 hits and the bulk are in 1-2 files.
2. Parallelism — agents in parallel, each handling one skill, dropped serial wall-time dramatically.
3. The pilot (qa-testing + craft + scaffolding + README) gave agents a concrete pattern that scaled cleanly. Grey-zone calls were rare and sidecar-documented.

**Implication for F4**: similar parallel-agent pattern works. Each prompt file can be drafted by an agent given its sidecar + the partials inventory. Estimate: ~1 day with similar parallel orchestration.

---

## F4 next steps (informed by this inventory)

1. Create `shark-data/prompts/_partials/` directory.
2. Implement Tier 1 partials first (5 files: `_fetch_entity_context`, `_resolve_spec_paths`, `_advance`, `_route_back_on_fail`, `_register_doc`).
3. Implement Tier 2 partials (4 files + 4 codex variant prompts: `_codex_invocation`, `_codex_red_team_gate`, `_loop_guard_check`, `_create_tech_debt`, `_persist_decision_log`, `_check_doc_exists`).
4. Implement Tier 3 partials (4 files: `_plan_context_get_set`, `_create_then_resolve_path`, `_consult_prior_art_preflight`, `_sibling_enumeration`).
5. Skip Tier 4 — inline at the single-use prompt site.
6. Each prompt becomes much shorter — most are `{{include:}}` of partials + skill workflow + workflow-specific glue. Rough math: original `.tmpl` files average 80-150 lines; expected post-F4 average is 30-50 lines.

Estimated `_partials/` size: 13 files (5 + 4 + 4) plus 4 codex prompt body variants = 17 files total.

---

*Last updated*: 2026-05-10 (final after all 9 in-scope skills processed)
