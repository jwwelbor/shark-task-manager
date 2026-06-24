# F4 — Migration plan (mechanical conversion)

This document is the concrete migration plan for F4. It assumes F1 (extraction) and F3 (engine) are done; F4 is mostly mechanical.

## Source inventory (audit run 2026-05-10)

In `~/projects/shark-task-manager/shark-templates/`:

| Entity | `.tmpl` files | Files with `LOAD:` |
|--------|--------------|--------------------|
| epic | 23 | 7 |
| feature | 25 | 11 |
| task | 18 | 10 |
| bug | 7 | 1 |
| change | 8 | 0 |
| tech_debt | 6 | 0 |
| epic_short | 16 | 4 |
| feature_short | 24 | 9 |
| task_short | 13 | 1 |
| partials | 10 | 2 |
| **Total** | **150** | **45** |

Plus: `.sharkworkflow.json` (1750 lines) and `.sharkworkflow-short.json`.

## Target layout (after F4)

```
shark-data/
  workflow-config.yaml                 # profile selector per entity
  workflow/
    epic.yaml          epic.short.yaml
    feature.yaml       feature.short.yaml
    task.yaml          task.short.yaml
    bug.yaml
    change.yaml
    tech-debt.yaml
  prompts/
    epic/{status}.md         (23 files from epic/*.tmpl)
    epic_short/{status}.md   (16 files)
    feature/{status}.md      (25 files)
    feature_short/{status}.md (24 files)
    task/{status}.md         (18 files)
    task_short/{status}.md   (13 files)
    bug/{status}.md          (7 files)
    change/{status}.md       (8 files)
    tech-debt/{status}.md    (6 files)
    _partials/               (10 existing + 13 new from F1.c inventory + 4 codex variants)
  skills/{9 in-scope skills}/    (output of F1)
  agents/{9 in-scope agents}/    (output of F1's harness audit)
  overrides/                      (empty in canonical default)
```

## Mechanical conversion checklist

### Step 1 — File renames (.tmpl → .md)

```bash
cd shark-data/prompts
for f in $(find . -name "*.tmpl"); do
  mv "$f" "${f%.tmpl}.md"
done
```

Or perform in-place during the copy from `shark-templates/`. The engine (F2/E4) reads any extension and strips frontmatter, so `.md` is the new convention.

**Side effect**: existing `{{template "name" .}}` references stay the same — they're Go template definitions, not filename references.

### Step 2 — `LOAD:` replacements

There are 45 files with `LOAD:` references. Per the 2026-05-10 rendering-model decision, **skills are referenced by path text, not inlined.** Replace each `LOAD: <skill>` with a path-reference block in the prompt body:

```
Load skill: shark-data/skills/<skill>/<workflow>.md
```

The agent reads the skill file at runtime via filesystem tools. Only **agents** and **partials** use `{{include:}}` to inline content.

**Mapping table** (skill path resolutions):

| `LOAD:` text | Replace with (path reference text) |
|---|---|
| `LOAD: quality skill.` | `Load skill: shark-data/skills/quality/SKILL.md` (router) — prefer specific workflow file if prompt knows which it needs. |
| `LOAD: quality skill workflow review-code.md.` | `Load skill: shark-data/skills/quality/workflows/review-code.md` |
| `LOAD: quality skill (test planning).` | `Load skill: shark-data/skills/quality/workflows/test-planning.md` |
| `LOAD: architecture skill workflows (design-system, design-backend, design-frontend, design-database, design-security) as applicable.` | Multiple path references; agent picks applicable workflow. |
| `LOAD: assessment skill (complexity-triage workflow).` | `Load skill: shark-data/skills/assessment/SKILL.md` with `mode=complexity_triage` set in the prompt's input variables. |
| `LOAD: research skill (analyze-codebase, trace-dependencies, find-patterns, understand-feature).` | Multiple path references. |
| `LOAD: specification-writing workflow refine-task-requirements.md (architect path).` | `Load skill: shark-data/skills/specification-writing/workflows/refine-task-requirements.md` with `refinement_role=tech` set. |
| `LOAD: specification-writing skill (task creation).` | `Load skill: shark-data/skills/specification-writing/workflows/write-task.md` |
| `LOAD: specification-writing skill (feature creation).` | `Load skill: shark-data/skills/specification-writing/workflows/write-feature-prd.md` |
| `LOAD: specification-writing skill (write-epic-prd workflow).` | `Load skill: shark-data/skills/specification-writing/workflows/write-epic.md` |
| `LOAD: specification-writing workflow decompose-epic.md.` | `Load skill: shark-data/skills/specification-writing/workflows/decompose-epic.md` |
| `LOAD: implementation skill, test-driven-development skill.` | Two path references. |
| `LOAD: debugging skill, test-driven-development skill.` | Two path references. |

**Agent inclusion** (separate from skill loading): every prompt also gets `{{include: agents/<agent_type>.md}}` to inline the agent persona/config. The `agent_type` for each status comes from the workflow YAML.

### Step 3 — Stale reference fixes (decisions 2026-05-10)

| File | Line | Current | Resolution |
|------|------|---------|------------|
| `epic/in_research.tmpl` | 14 | `LOAD: discovery skill (market-research, feasibility-analysis) + research skill (analyze-codebase, find-patterns).` | Replace with path references to `shark-data/skills/research/workflows/analyze-codebase.md` and `find-patterns.md`. Drop the `discovery` half — folded into research. |
| `epic/ready_for_research.tmpl` | 7 | Same as above | Same as above |
| `feature/ready_to_build.tmpl` | 5 | `LOAD: build skill.` | **DROP entirely.** User has been using `feature_short/` exclusively; long-workflow `feature/` is dead code. F4 skips migrating long-workflow files. |
| `.sharkworkflow.json` line 973 | — | `"build"` in `feature_short/ready_to_build` loads array | **DROP.** No replacement. The `ready_to_build` step doesn't load any skill. |

### Step 4 — Workflow JSON → YAML

Convert `.sharkworkflow.json` (1750 lines) to per-entity YAML under `shark-data/workflow/`:

- Identify each entity's workflow tree by `entity_type` field.
- Round-trip-equivalent: same step names, same agent assignments, same `instruction_template` paths (now ending in `.md`), same routing keys, same `skip_for`.
- Preserve workflow_short profiles in `<entity>.short.yaml`.

Cross-repo stale-ref fix (recorded in `_partials_inventory.md`):

- 4 `discovery` refs at lines 162, 179, 200, 217 → `research`.
- 1 `build` ref at line 973 (in `feature_short/ready_to_build/instruction_template` loads) → drop or replace with `devops`.

Convert `.sharkworkflow-short.json` similarly.

### Step 5 — Move skills (output of F1)

```bash
# Pseudocode — run with rsync or git mv
for skill in specification-writing architecture research quality implementation \
            test-driven-development assessment uat debugging; do
  cp -r ~/.claude/skills/$skill shark-data/skills/$skill
done
```

The F1 work already extracted the craft and produced the sidecars. F4 takes the extracted craft only — sidecars at `_extracted/` were transient (their content moved into the prompts via Step 6).

**Note**: `_extracted/` directories should NOT be copied to `shark-data/skills/`. They're scratch space from F1; their content is already absorbed into the prompts and partials.

### Step 6 — Build `_partials/` from inventory

Per `_partials_inventory.md`:

**Tier 1** (5 files):
- `_fetch_entity_context.md`
- `_resolve_spec_paths.md`
- `_advance.md`
- `_route_back_on_fail.md`
- `_register_doc.md`

**Tier 2** (5 files + 4 codex variants):
- `_codex_invocation.md`
- `_codex_qa_prompt.md`, `_codex_review_prompt.md`, `_codex_test_planning_prompt.md`, `_codex_uat_prompt.md`
- `_codex_red_team_gate.md`
- `_loop_guard_check.md`
- `_create_tech_debt.md`
- `_persist_decision_log.md`
- `_check_doc_exists.md`

**Tier 3** (4 files):
- `_plan_context_get_set.md`
- `_create_then_resolve_path.md`
- `_consult_prior_art_preflight.md`
- `_sibling_enumeration.md`

Plus the 10 existing partials in `shark-templates/partials/` (carry over):
- `_client.md`, `_code_review_process.md`, `_commands.md`, `_exit_gate.md`, `_qa_process.md`, `_read_section.md`, `_resume_preamble.md`, `_sizing.md`, `_tdd_process.md`, `_uat_redteam_review.md`

**Note**: some existing partials (`_qa_process.tmpl`, `_code_review_process.tmpl`) duplicate content the new partials cover. Audit for consolidation in F4. Probably the existing partials get re-expressed using the Tier 1-3 partials as building blocks — e.g., `_qa_process.md` becomes a thin wrapper that includes `_fetch_entity_context.md` + `_codex_invocation.md` + `_codex_qa_prompt.md` + `_advance.md` / `_route_back_on_fail.md`.

### Step 7 — Move agents

```bash
for agent in architect business-analyst developer qa tech-lead \
             product-manager tech-director researcher uat-agent; do
  cp ~/.claude/agents/$agent.md shark-data/agents/$agent.md
done
```

**Decision required before this step**: agent dispatch routing (per epic.md follow-up backlog idea I-2026-05-10-05). Three options:
- Inline agent definitions into rendered prompts (loses sub-agent isolation).
- Copy on init from `shark-data/agents/` to `.claude/agents/` (re-creates duplication).
- New harness↔shark agent contract (cleanest; means `shark/SKILL.md` is no longer "5 lines"; orchestration regrows in disguise).

**Recommendation pending user decision**: option 2 (copy on init) is the lowest-risk default. The duplication is one-way (init copies; upgrade refreshes); overrides land in `shark-data/overrides/agents/` which `init` skips re-copying.

### Step 8 — Update embedded FS in shark binary (F3 hand-off)

The F3 work shipped the `embed.FS` machinery; F4 just updates the embedded source path to point at the populated `shark-data/`. Single line change in `cmd/shark/main.go`:

```go
//go:embed shark-data/*
var defaultDataFS embed.FS
```

### Step 9 — Verification

Per F4 exit gate:

1. `shark init` lays down complete tree on a fresh project.
2. `/run E01-F02-???` on a real project exercises the new path end-to-end.
3. **Golden-output diff**: render representative dispatches before/after — semantically equivalent. (See follow-up idea I-2026-05-10-06 — recommended for F3 exit gate but practically lands here.)
4. All `LOAD:` references in prompts replaced with `{{include:}}`.
5. Stale refs (`discovery`, `build`) resolved.
6. `shark-templates/` audit confirmed: no external scripts hardcoding the directory name (already verified — see F4 reference note on E02-F04). Only `~/projects/shark-task-manager/scripts/` may need updating; that's in-repo for the engine work.

### Step 10 — Agent dispatch routing decision

Per follow-up idea I-2026-05-10-05 — **must decide before F4 ships**. Current recommendation: copy-on-init (option 2 above) as default; revisit if it bites.

## Effort estimate

| Step | Estimated effort |
|------|------------------|
| 1. File renames | < 1 hour (script) |
| 2. `LOAD:` replacements | 2-3 hours (45 files; mechanical with the mapping table) |
| 3. Stale ref fixes | 30 min |
| 4. Workflow JSON → YAML | 1 day (per-entity, with round-trip validation) |
| 5. Move skills | < 1 hour (script) |
| 6. Build `_partials/` | 4-6 hours (13 new partials + audit existing 10 for consolidation) |
| 7. Move agents | < 1 hour |
| 8. Embedded FS update | < 1 hour |
| 9. Verification | 1 day (golden-output diff + end-to-end run) |
| 10. Agent dispatch decision | 30 min discussion |
| **Total** | **~2-3 days of focused work** |

## Out of scope for F4

- Engine changes (F2/F3).
- Harness repointing (F5).
- Cleanup / removing fallback paths (F6).
- Override drift mitigation (follow-up epic).

## Risks

- **Stale partials**: existing `partials/_qa_process.tmpl` and `_code_review_process.tmpl` duplicate work that the new Tier 1-3 partials cover. Need to merge cleanly without losing project-specific scaffolding (e.g., `_qa_process.tmpl`'s targeted-test-runner approach).
- **Workflow JSON-YAML round-trip**: validate semantically, not just syntactically. Pick one entity (`task.yaml`) as proof per F2's E2 plan.
- **Agent dispatch decision**: not yet made; blocks final F4 commit.
- **Cross-repo coordination**: F4 spans `~/.claude/` and `~/projects/shark-task-manager/`. Workflow JSON conversion + stale-ref fixes + agent moves all happen in the engine repo; skill moves happen in the harness repo. Two-repo PR coordination needed.

---

*Last updated*: 2026-05-10
