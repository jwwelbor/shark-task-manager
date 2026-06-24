# F4 prompts-draft

Draft of the `shark-data/prompts/` directory contents for F4. Non-canonical — these are exemplars to validate the partials inventory and give F4 a concrete starting point.

When F4 begins, copy these into `internal/sharkdata/default_data/prompts/` and adjust based on actual rendering behavior.

## Rendering model (decided 2026-05-10)

The rendered response from `shark next` contains:

- **Agent prompt body** — INLINED via `{{include: agents/<type>.md}}`. The agent's persona, model, tool config arrive in the prompt itself.
- **Action prompt body** — INLINED. This is the prompt file's own content.
- **Partials** — INLINED via `{{template "_name" .}}` (in-tree) or `{{include: prompts/_partials/_name.md}}` (cross-tree).
- **Skill references** — *NOT inlined.* Skill paths are written as text in the prompt body. The agent reads skill files at runtime via filesystem tools.

The `{{include:}}` directive (engine E1) works for any path under `shark-data/`, but by convention we **don't** use it on `skills/*.md` for prompt size + caching reasons. Skills stay as path references that the agent reads on demand.

See E02 decision note for full rationale.

## Future extension (idea I-2026-05-10-11)

Path resolution for agents/prompts/skills will become configurable in a future release so that:
- Users can keep agents in their harness folder (`~/.claude/agents/`) instead of `shark-data/agents/`.
- Skills can come from a shared absolute path (cross-project canonical library).
- Whether the agent body is inlined into the prompt vs spawned via the harness becomes a config flag.

Not built now; revisit if friction emerges.

## Contents

- **`_partials/_fetch_entity_context.md`** — Tier 1 partial. Resolve entity ID to opaque IDs + spec paths.
- **`_partials/_resolve_spec_paths.md`** — Tier 1 partial. Domain-specific artifact path resolution.
- **`_partials/_advance.md`** — Tier 1 partial. PASS-side state advancement.
- **`_partials/_route_back_on_fail.md`** — Tier 1 partial. FAIL-side routing.
- **`_partials/_register_doc.md`** — Tier 1 partial. Register a produced doc with shark.
- **`_partials/_codex_qa_prompt.md`** — Tier 2 codex prompt body variant for QA.
- **`task/in_qa.md`** — Example prompt showing the rendering model: inlined agent + inlined action prompt + skill path reference.

## Validation against F1.c partials inventory

These exemplars confirm the inventory is implementable as written. Open questions for F2 to resolve (already shipped, just need real prompts to exercise):

- ✓ **`{{include:}}` vs `{{template ...}}` mixing semantics** — confirmed working side-by-side in E1.
- ✓ **`{{include:}}` parameter passing** — partials use `(dict "key" "value")` per Go template idiom.
- ✓ **Frontmatter behavior** — E4 strips frontmatter before render. Frontmatter is metadata for validation; not rendered.

## Decisions made (2026-05-10)

- **Agent dispatch routing** (idea I-2026-05-10-05): inline agent body + path-reference skills. Closed.
- **Stale `LOAD: build skill` ref**: drop entirely; user uses `feature_short/` exclusively. Closed.
- **Configurable path resolution** (idea I-2026-05-10-11): deferred follow-up.

## What's NOT in this draft

- All other prompts (75+ remaining task/feature/epic/bug/change/tech-debt prompts) — F4 fills these mechanically using the migration plan and partials.
- All other partials (Tier 2 except `_codex_qa_prompt`, all Tier 3) — F4 produces these from sidecars.
- The 10 existing partials in `shark-templates/partials/` — F4 audits each for consolidation opportunity vs preservation as project-specific scaffolding.
- Workflow YAML files (F4 step 4).
- Skills move (F4 step 5).
- Agents move (F4 step 7).
