---
research_schema: 2
entity_key: E34-F09
entity_type: feature
recipe: universal
rigor: complex
categories:
  - backend
  - workflow_operations
  - documentation
related_work: true
---

# E34-F09 Research Report: Override Drift Visibility and WWGM Reconciliation

## Scope

E34-F09 adds visibility and baseline provenance for replace-only Shark data
overrides, then defines the cross-repository WWGM adoption boundary. It covers
the embedded canonical bundle, upgrade/status CLI, checksum metadata, current
WWGM overrides, and existing WWGM change cards. It does not automatically edit
WWGM or merge override content.

## Research checklist

- [x] `scope_vocabulary` — Evidence: `E34-F09-override-drift-visibility-and-wwgm-reconciliation/feature.md`; `internal/sharkdata/default_data/README.md`; `internal/cli/commands/sharkdata_cmd.go`. These sources define canonical data, replace-only override, upgrade, baseline, counterpart, and dry-run.
- [x] `affected_implementation_or_contract` — Evidence: `internal/sharkdata/embed.go`; `internal/cli/commands/sharkdata_cmd.go`; `internal/sharkdata/{embed_test.go,resolve_at_test.go}`. Upgrade skips overrides and has no drift classifier or baseline manifest.
- [x] `related_work` — Evidence: E34-F03, F06, F07, and F08 packets; E40 design; WWGM E04 proposal/inventory; WWGM CC-007 and CC-008.
- [x] `pattern_contract` — Evidence: SHA-256 prompt provenance, deterministic JSON CLI output patterns, project-root data resolution, path validation, and no-content upgrade summaries already used in the repository.
- [x] `dependency_impact` — Evidence: every override masks its canonical counterpart during prompt, skill, or workflow resolution; upgrade changes canonical bytes but intentionally preserves the masking file.
- [x] `cross_boundary_risks` — Evidence: status spans embedded binary data, on-disk canonical data, user-owned overrides, and metadata. An unsafe implementation could leak content, follow symlinks, mutate user files, or label unknown provenance current.
- [x] `alternatives` — Evidence: manual diffs identified WWGM drift but do not scale or survive baseline loss; automatic merges would make unsafe semantic assumptions about Markdown templates and workflow YAML.

## Capability map

| Capability | Evidence | Decision | E34-F09 responsibility |
|---|---|---|---|
| Replace-only override resolution | bundle README and resolver tests | REUSE | Preserve exact precedence and classify its maintenance risk. |
| Canonical upgrade and dry-run | `UpgradeAt` and admin command | EXTEND | Add baseline capture and drift summary without modifying overrides. |
| Stable digests | prompt SHA-256 contracts elsewhere in Shark | REUSE | Compare bytes without emitting content. |
| Override drift status | No command or service exists | NEW | Add five-state classification with deterministic JSON. |
| Baseline acknowledgement | No provenance mechanism exists | NEW | Record current canonical digest only after explicit reconciliation. |
| Reusable WWGM gate semantics | approval/rubric/epic review overrides | PROMOTE | Assign to F06–F08 and prove canonical coverage before removal. |
| WWGM workflow/model policy | workflow override diffs | RETAIN LOCAL | Rebase on new canonical YAML, preserving order and Sol assignments. |
| WWGM application safeguards | E04 proposal P1/P4/P5 | RETAIN LOCAL | Track in one WWGM item; do not place project commands in Shark defaults. |

## Findings

1. The current upgrade summary proves preservation, not safety. `skipped` does
   not tell an operator whether an override is intentional, stale, redundant,
   or orphaned.

2. Content comparison alone cannot distinguish current intentional divergence
   from an upstream change. A trustworthy baseline digest and an explicit
   acknowledgement operation are required; missing provenance must stay
   unknown.

3. Automatic merge is unsafe for prompt templates and route-based workflow
   YAML. Read-only classification plus explicit operator reconciliation is the
   correct authority boundary.

4. WWGM contains both reusable improvements and real local policy. Treating the
   whole tree as either “upstream all” or “delete all” would be wrong. The
   path-by-path matrix preserves the distinction.

5. CC-007 and CC-008 already track tier-artifact and staged/disposition gaps.
   F09 must link or resolve them through the single WWGM adoption item rather
   than create parallel records.

6. The proposed `rules.py` and editor hook have only one-project evidence.
   A thin WWGM `AGENTS.md` and project-local deterministic guards address the
   observed channel gap without prematurely adding general Shark machinery.

## Decisions

1. Add read-only five-state classification and stable JSON.
2. Track canonical baseline digests, never override contents.
3. Require explicit acknowledgement after manual reconciliation.
4. Never auto-merge, rewrite, remove, or disable an override.
5. Promote reusable WWGM behavior through F06–F08; keep commands, models,
   workflow order, standards, and lint local.
6. Use one linked WWGM adoption item and reuse CC-007/CC-008.
7. Keep E40 as later comparative validation.

## Sources

- `internal/sharkdata/default_data/README.md`
- `internal/sharkdata/embed.go`
- `internal/sharkdata/{embed_test.go,resolve_at_test.go}`
- `internal/cli/commands/{admin.go,sharkdata_cmd.go,sharkdata_cmd_test.go}`
- `docs/plan/E34-prompt-and-skill-improvements/E34-F0{3,6,7,8}-*/`
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/{epic.md,shark-bench-design.md}`
- WWGM `shark-data/overrides/`
- WWGM `docs/plan/changes/{CC-007.md,CC-008.md}`
- WWGM `dev-artifacts/2026-08-04-1530-e04-review-gap-analysis/{PROPOSAL.md,INVENTORY.md}`
