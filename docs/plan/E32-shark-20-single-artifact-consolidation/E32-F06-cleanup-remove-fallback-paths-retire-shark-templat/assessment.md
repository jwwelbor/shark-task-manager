# F06 Assessment — 2026-06-22

## 1. Engine fallback — shark-templates/ references in production Go code

The fallback is **fully present and active**. `internal/templates/orchestrator_renderer.go` defines `const defaultTemplateDir = "shark-templates"` (line 15) and explicitly implements a two-pass resolution: Pass 1 tries `shark-data/prompts/`, Pass 2 falls back to `shark-templates/` (line 109). `internal/includes.go` also carries legacy-mode logic for the fallback layout. Additionally, `internal/init/config.go` still writes `"workflow_config": "shark-templates/.sharkworkflow-short.json"` as the default on `shark init`, and the `sharkdata_cmd.go` comment documents that the legacy `.sharkworkflow*` path auto-migrates. In total, ~15 non-test production Go lines directly reference `shark-templates` as a live resolution path.

## 2. Deprecated slash commands (F05 work)

All 8 slash commands listed in the feature scope are present in `~/.claude/commands/` and **all carry correct F05 deprecation headers** (`> **DEPRECATED (E32-F05)** — ...This command remains functional for one release; it will be removed in F06.`). F05 is confirmed complete on this axis.

## 3. shark-templates/ references in docs

`grep -r "shark-templates" docs/ CLAUDE.md` returns **321 occurrences** across historical QA reports, archived feature specs (E04, E07, E20, E28), and CLAUDE.md workflow-config examples. The majority are in immutable historical artefacts (QA reports, old task files) and do not need to change; the actionable subset is CLAUDE.md and active onboarding docs. This is non-trivial but scoped cleanup work.

## 4. Release-window dependency assessment

F05 was completed today (2026-06-22). The feature file is explicit: "Don't merge F6 until at least one full release of F5 has been in daily use." The most recent git tag is `v1.5.4`, which pre-dates F05. **No release containing the F05 deprecation headers has shipped yet.** Executing F06 now would violate the back-compat contract the deprecation headers promised users ("functional for one release").

## 5. Recommendation

**Block F06.** Do not execute now. The release-window gate is not a bureaucratic formality — the deprecation headers told any user invoking `/run`, `/feature`, etc. that those commands would be removed "in F06," implying at least one intervening release. The correct sequence is: (a) cut a release that includes F05 changes, (b) confirm at least one day of normal use under that release, (c) then execute F06. Status should remain `in_assessment` (or be set to `blocked`) with a note that the gate condition is: first release tag after 2026-06-22 + one working day.
