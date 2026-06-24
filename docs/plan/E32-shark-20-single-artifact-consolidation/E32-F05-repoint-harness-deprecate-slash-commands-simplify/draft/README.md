# F5 draft artifacts

Drafts for review when F5 is ready to ship (after F4 lands).

## `shark-SKILL.md`

Replacement for `~/.claude/skills/shark/SKILL.md`. Goes from 194 lines (mix of user reference + dispatch hint) to ~70 lines of pure dispatch trigger.

### Design decisions captured in the draft

1. **User reference is split off** — the current SKILL.md doubles as a "how to use shark CLI" reference for users. The new design moves that content to `<shark-data>/skills/shark-cli-reference.md` (canonical, ships with binary). The harness skill is dispatch-only.
2. **Per-harness adapter section** — acknowledges the realistic edge in follow-up idea I-2026-05-10-08 ("shark/SKILL.md will grow beyond 5 lines"). The skill stays minimal but explicitly mentions where harness-specific logic lives.
3. **No workflow logic** — explicit "what this skill is NOT" section to prevent regression toward orchestration.
4. **Hooks** — sibling `HOOKS.md` referenced; hooks stay harness-side because they often need harness-specific signals.

## Slash command deprecation headers (not yet drafted)

Need to add deprecation headers to:

- `/run`, `/feature`, `/epic`, `/task`, `/prd`, `/dispatch`, `/develop`, `/release`

Header template (to be added at top of each `.md`):

```markdown
> **DEPRECATED in shark 2.0.** This slash command is functional for one release window
> while users migrate. Replacement: <see canonical entry point in shark-data/>.
> Removal scheduled for F6 cleanup.
```

These don't need to be drafted now — F5 step is mechanical (8 files, paste header).

## Pending decisions blocking F5

- **Agent dispatch routing** (idea I-2026-05-10-05) — F5 deletes in-scope agents from `~/.claude/agents/`. If the resolution is option 2 (copy on init), F5 needs to ensure `shark init` had already populated `.claude/agents/` from `shark-data/agents/` before deletion is safe.
- **One-release deprecation** vs hard cutover (idea I-2026-05-10-07) — F5 should ship harness skill *and* agent deletions with deprecation warnings for one release before actually deleting. Recommendation: F5 ships warnings; F6 does the actual deletion. This re-shapes F5's exit gate.

## Out of scope for this draft

- Actual `~/.claude/skills/shark/HOOKS.md` content — TBD; current HOOKS.md may need minor updates.
- Slash command deprecation headers (8 files; mechanical, do during F5 implementation).
- Agent definition edits — those go in F4 when agents move to `shark-data/agents/`.
