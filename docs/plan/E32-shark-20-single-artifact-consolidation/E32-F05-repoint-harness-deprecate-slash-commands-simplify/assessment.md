# F05 Assessment — Repoint harness, deprecate slash commands, simplify shark/SKILL.md

**Date**: 2026-06-22
**Status at assessment**: `ready_for_assessment` (returned from UAT rejection 2026-05-12)

---

## Situation Summary

The UAT rejection (2026-05-12) documented a previous implementation attempt where:
- In-scope skills and agents were correctly deleted from `~/.claude/`
- The slash commands `/feature`, `/epic`, `/task`, `/prd`, `/dispatch`, `/develop`, `/release` were **deleted outright** rather than given deprecation headers
- `/run` was present but had no deprecation header

Since that rejection, the harness has been restored (or reset). The current machine state is effectively **a clean slate** — none of the F05 work is in place.

---

## Current State Audit

### Slash commands (`~/.claude/commands/`)

All 8 in-scope commands are **present with no deprecation headers**:

| Command | Present | Deprecation header |
|---------|---------|-------------------|
| `/run` | Yes | No |
| `/feature` | Yes | No |
| `/epic` | Yes | No |
| `/task` | Yes | No |
| `/prd` | Yes | No |
| `/dispatch` | Yes | No |
| `/develop` | Yes | No |
| `/release` | Yes | No |

All commands retain their original full functional content. None point to new entry points.

### In-scope skills (`~/.claude/skills/`)

All 10 in-scope skills are **present** (not deleted):

`specification-writing`, `quality`, `architecture`, `research`, `implementation`, `test-driven-development`, `assessment`, `uat`, `debugging`, `orchestration`

### In-scope agents (`~/.claude/agents/`)

All 9 in-scope agents are **present** (not deleted):

`architect.md`, `business-analyst.md`, `developer.md`, `qa.md`, `tech-lead.md`, `product-manager.md`, `tech-director.md`, `researcher.md`, `uat-agent.md`

### `~/.claude/skills/shark/SKILL.md`

Present. Contains a **complete, correct dispatch-loop rewrite** (~140+ lines). This content was produced by the previous F05 implementation and survived the harness restore. No action needed here.

### shark-data/ (F04 deliverables)

**Embedded canonical** (`internal/sharkdata/default_data/`): fully populated with skills, agents, workflows, prompts. This is the source of truth.

**Deployed copy** (`shark-data/` at project root): present but minimal — only `README.md`, `shark-SKILL.md`, and top-level directories. The deployed copy is not what F05 depends on; F05 depends on `shark init` pushing the embedded canonical to the project's `shark-data/`.

The embedded canonical has all 9 in-scope skills and all 9 in-scope agents. **F04 deliverables are sufficient to support the harness deletion.**

---

## Resolution Path Recommendation

**Restore + Deprecate** (Path A).

The current state is a clean slate — nothing from the previous F05 attempt remains harmful. The recommended path is straightforward:

1. The harness still has all the old slash commands and skills.
2. F06 still expects to delete deprecated (not already-gone) slash commands.
3. shark/SKILL.md is already correct — no work needed there.
4. The verification scenario "move quality aside, /run still works" is validatable again because `quality/` is present.

Choosing "Document Hard-Cutover" (Path B) is unnecessary and creates confusion in F06's scope, which explicitly says it will delete commands "that carried deprecation headers in F5." If we document a hard-cutover, F06 needs significant rewriting for a minimal gain.

---

## Tasks to Close F05

1. **Delete in-scope skills** from `~/.claude/skills/`: `specification-writing`, `quality`, `architecture`, `research`, `implementation`, `test-driven-development`, `assessment`, `uat`, `debugging`, `orchestration`

2. **Delete in-scope agents** from `~/.claude/agents/`: `architect.md`, `business-analyst.md`, `developer.md`, `qa.md`, `tech-lead.md`, `product-manager.md`, `tech-director.md`, `researcher.md`, `uat-agent.md`

3. **Add deprecation headers** to all 8 in-scope slash commands — a two-line YAML front-matter note + a one-paragraph redirect at the top of each file pointing to the new `shark run` / `shark` skill entry point. Leave the remaining body functional.

4. **Verify AC-2**: After deleting `quality/` from `~/.claude/skills/`, confirm `/run` on a workflow that invokes the quality skill still resolves correctly (via `shark-data/skills/quality/` inline at render time).

5. **Update F06 scope** to confirm it should delete the now-deprecated (not already-gone) slash commands — no F06 scope change required since F06 already expects this.

---

## Notes on `shark/SKILL.md`

The existing `~/.claude/skills/shark/SKILL.md` was rewritten by the previous F05 implementation and is **correct as-is**. It contains the dispatch loop and references `workflows/run.md` for the actual orchestration. No rewrite needed for the new implementation attempt — this is complete work that survived the harness restore.

---

## F04 Deliverable Sufficiency

Yes. `internal/sharkdata/default_data/` has all 9 in-scope skills and 9 agents. Running `shark init` on any project will populate `shark-data/` with these. The AC-2 verification scenario (move quality aside from `~/.claude/skills/`, run a quality workflow) is valid because the engine resolves from `shark-data/skills/` at render time, not from `~/.claude/skills/`.
