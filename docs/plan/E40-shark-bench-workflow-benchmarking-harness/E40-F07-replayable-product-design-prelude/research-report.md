---
research_schema: 2
entity_key: E40-F07
entity_type: feature
recipe: universal
rigor: complex
categories:
  - backend
  - data
  - workflow_operations
  - documentation
related_work: true
---

# Research report: Replayable product-design prelude

## Scope

E40-F07 wraps the existing `/shark-rider project product-design` action
(D01-D05 only) so a **feature**-family benchmark scenario can run it without a
live stakeholder, a live web-research call, or an unrecorded operator
decision. It must not fork the D01-D05 methodology, must route every
`AskUserQuestion` and `WebSearch` call the bundle makes through a versioned
replay bundle, must stop with `unresolved_gate` when the bundle lacks an
authorized answer, must record response/artifact-consumption lineage and
interaction-volume proxies (never labeled as human minutes), and must record
D01-D05 as explicitly non-applicable for bug/change-card/tech-debt scenarios.
It consumes I-04 (E40-F05's scenario package, specifically
`stage_matrix.prelude` and `replay_reference`) and X-10 (the completed E36-F02
product-design action). It produces I-06, consumed only by E40-F08 — the
keyed Shark entity lifecycle that starts after the prelude, and Shark's own
Question lifecycle (X-13), are both explicitly out of this feature's scope.

## Research checklist

- [x] `scope_vocabulary` — Evidence: `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F07-replayable-product-design-prelude/feature.md` (Scope/Contracts/Out-of-scope sections) and `docs/plan/E40-shark-bench-workflow-benchmarking-harness/architecture.md#product-design-replay-contract` (I-06 field list) define "replay adapter," "authorized entry," "replayed interaction proxy," and the D01-D05/D06-D14 boundary.
- [x] `affected_implementation_or_contract` — Evidence: `skills/shark-rider/verbs/product-design.md` (the owning Rider adapter — reads `docs/product/progress.md`, retrieves the bundle via `shark skill get product-design`, never invokes the CLI or another skill itself) and `internal/sharkdata/default_data/skills/product-design/{SKILL.md,workflows/d01-vision.md,workflows/d03-market-research.md,workflows/d05-stakeholder-insights.md}` (the exact `AskUserQuestion` and `WebSearch` call sites this feature must intercept).
- [x] `related_work` — Evidence: `docs/plan/E36-project-layer-and-consult-bridge/E36-F02-project-namespace-and-progress-record/feature.md` (X-10's producer: `docs/product/progress.md` derived-checklist/decision-log contract, status `completed`), `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F05-lifecycle-scenario-corpus-and-adapter-contract/{research-report.md,test-plan.md}` (I-04's producer; confirms X-10/E36-F02 already `completed`, unlike E27-F15), `bench/scenarios/packages/py-feature-recurring-tasks/package.yaml` and its `evaluator/replay/reference-bundle.json` (I-04's only feature-family seed, its `replay_reference` field explicitly deferred to E40-F07/I-06), and `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F08-canonical-multi-entity-lifecycle-runner/feature.md` (I-06's sole consumer).
- [x] `pattern_contract` — Evidence: `internal/runner/dispatcher.go` (`DefaultDisallowedTools`, `--disallowedTools`) and `internal/runner/claude_dispatcher.go:114-124` (arbitrary tool names, not only `Bash(...)` patterns, can be disallowed per-invocation) establish the codebase's one existing pattern for blocking a live tool from an agent session; `bench/scripts/replay-manifest.sh` (G7's replay-and-compare verification, `docs/plan/.../E40-F03-baseline-report-and-noise-band/`) establishes the epic's existing "replay a versioned input, verify reproduction" pattern, though for aggregate metrics rather than a scripted interaction sequence.
- [x] `dependency_impact` — Evidence: `bench/evidence/i05-schema.yaml` and `bench/scripts/verify-evidence-roots.sh` (E40-F06's completed three-root isolation model: agent-visible checkout / scratch Shark project / evaluator-only root) — I-06's artifacts (D01-D05 files, `docs/product/progress.md`) are written by a skill invocation, not by `shark run`, so this feature must state which of the three roots they belong to; `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-cross-epic-map.md` (X-10 row: `assigned`, `E40-F07`, `E40 UAT-10`) confirms E40-F07 is the sole owner of the X-10 seam.
- [x] `cross_boundary_risks` — Evidence: `internal/sharkdata/default_data/skills/product-design/SKILL.md:96-104` (Checkpoint boundary: the bundle "does not own ordering, resumability, progress records, CLI commands, or retrieval of another skill" — the replay adapter must sit in the Rider adapter layer, not inside the bundle, or it duplicates this ownership split) and `docs/plan/.../architecture.md` (I-07's "prompt and worker-result reference" is sourced from X-11's Shark-rendered workflow prompt, a different provenance than D01-D05's `shark skill get product-design` retrieval — the two "prompt digest" concepts must not be conflated when F08 later stitches I-06 into I-07).
- [x] `alternatives` — Evidence: `docs/plan/.../E40-F06-stage-evidence-and-evaluator-isolation/research-report.md` and `bench/scripts/verify-stage-evidence.sh` (considered and rejected: reusing a generic "stage evidence" capture path for D01-D05 instead of a dedicated I-06 contract) and `docs/plan/E39-.../` (Question lifecycle, X-13 — considered and rejected as the mechanism for D01-D05's human elicitation, since D01-D05 questions are in-session `AskUserQuestion` tool calls inside an interactive Rider action, not durable Shark Question entities).

## Capability map

| Capability | Brownfield evidence | Decision | E40-F07 responsibility |
|---|---|---|---|
| Shark Rider product-design action (D01-D05 methodology, checkpoint/resume, D04 stack-feedback) | `skills/shark-rider/verbs/product-design.md`; `internal/sharkdata/default_data/skills/product-design/{SKILL.md,workflows/*}` | REUSE | Invoke the action and bundle unmodified through X-10; wrap it, never fork or copy its methodology, exactly as the feature file requires. |
| `docs/product/progress.md` derived checklist + append-only decision log | `docs/plan/E36-project-layer-and-consult-bridge/E36-F02-project-namespace-and-progress-record/feature.md` (status `completed`) | REUSE | Let the wrapped action continue to seed/update this file as designed; F07 does not introduce a second progress record or a shark entity for it. |
| `AskUserQuestion` (D01, D02, D05) and `WebSearch` (D03, D04) call sites | `internal/sharkdata/default_data/skills/product-design/SKILL.md:110,133,136`; `workflows/d01-vision.md:11`; `workflows/d03-market-research.md:22`; `workflows/d05-stakeholder-insights.md:9-17,37` | EXTEND | These are the exact, and only, interaction points a scored run must route through the replay adapter and must disable live; F07's replay adapter has no other surface to cover for D01-D05. |
| `--disallowedTools` / `DefaultDisallowedTools` blocking pattern | `internal/runner/dispatcher.go:19-26`; `internal/runner/claude_dispatcher.go:114-124` | EXTEND, with a substitution gap | The existing pattern blocks a tool outright (agent then has no path forward); it does not supply a scripted answer in its place. F07 needs disallow-plus-substitute: block live `AskUserQuestion`/`WebSearch`, and give the dispatched session a local, authorized replay source to consult instead, stopping `unresolved_gate` only when that source lacks the entry. This is a benchmark-only extension of the pattern, not a Go-code change to `internal/runner` (F07 dispatches host-side, not through `shark run`). |
| I-04 scenario package `replay_reference` field | `bench/scenarios/scenarios.yaml`; `bench/scenarios/packages/py-feature-recurring-tasks/package.yaml:94` (`replay_reference: evaluator/replay/reference-bundle.json`); `evaluator/replay/reference-bundle.json`'s own header ("interior shape belongs to E40-F07/I-06, not typed [by F05]") | REUSE (path) / NEW (interior schema) | F05 already reserves the pointer and guarantees only that the path exists and is non-empty; F07 owns and must define I-06's interior shape — the authorized-response sequence, digests, and lineage fields — since nothing upstream defines them. |
| I-05 three-root evaluator-isolation model | `bench/evidence/i05-schema.yaml`; `bench/scripts/verify-evidence-roots.sh` (E40-F06, `completed` on this branch) | EXTEND | D01-D05 artifacts and `docs/product/progress.md` are written by a skill invocation outside `shark run`'s fixture checkout — F07 must state explicitly that they belong to the scratch-Shark-project root (planning documents, not fixture-repo code), not invent a fourth root. |
| E39-F04 durable Question lifecycle (X-13) | `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-cross-epic-map.md` (X-13 row assigned to E40-F08, not E40-F07); `docs/plan/.../E40-F08-canonical-multi-entity-lifecycle-runner/feature.md` ("Consumes X-13 ... use the E39-F04 Question lifecycle as the durable human-gate surface") | CONTRADICTS (not this feature's mechanism) | D01-D05's human elicitation is an in-session `AskUserQuestion` tool call inside one interactive Rider action, not a durable Shark Question entity created against a keyed entity. F07 must not route the prelude's questions through X-13; that surface belongs to F08's post-prelude keyed lifecycle. |
| `bench/scripts/replay-manifest.sh` reproducibility-verification pattern (G7) | `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F03-baseline-report-and-noise-band/` (script header, `research-report.md`) | REUSE (pattern only) | The "replay a stored, versioned input and verify byte-identical reproduction" shape F07's acceptance boundary requires ("two runs ... consume the same response sequence") already exists as an epic convention for aggregate metrics; F07 applies the same shape to an interaction sequence rather than copying the script. |

## Findings

1. The wrapped action already names its own boundary precisely: `SKILL.md`'s
   "Checkpoint boundary" section states the D01-D05 bundle "does not own
   ordering, resumability, progress records, CLI commands, or retrieval of
   another skill," and that ownership sits in
   `skills/shark-rider/verbs/product-design.md`. The replay adapter belongs at
   that same adapter layer — intercepting `AskUserQuestion`/`WebSearch` inside
   the bundle's own workflow files would duplicate an ownership split the
   codebase has already drawn deliberately.

2. D01-D05's live-input surface is small and fully enumerated by direct
   inspection: `AskUserQuestion` at D01 (vision elicitation), D02 (success
   criteria, per `SKILL.md:110`), and D05 (evidence-mode choice and interview
   synthesis, `d05-stakeholder-insights.md:9-17,37`); `WebSearch` at D03
   (market research) and D04 (feasibility/technical research, per
   `SKILL.md:136`). No other tool call in D01-D05 reaches outside the
   session. This is the complete interception surface — the feature's acceptance
   boundary ("a scored run cannot reach live research or an interactive
   question surface") reduces to exactly these five call sites.

3. The codebase's one existing "block a tool from an agent session" pattern
   (`DefaultDisallowedTools`, `--disallowedTools` in
   `internal/runner/claude_dispatcher.go:114-124`) blocks outright — it has no
   substitution mechanism, because its purpose (preventing self-advancement)
   never needs one. F07 needs the same blocking primitive plus a substitute
   answer source, which is new behavior at the benchmark-adapter layer, not a
   `internal/runner` Go change; F07 dispatches host-side over the Rider action,
   never through `shark run`.

4. I-04's `py-feature-recurring-tasks` package is the only seed scenario with
   `stage_matrix.prelude` all `true` and the only one carrying a
   `replay_reference` field — confirmed by its own header comment, which
   explicitly defers the field's interior shape to E40-F07/I-06 and states F05
   guarantees only that the path is a non-empty string. F07 has a real,
   already-wired pointer to build I-06's file format against, but no upstream
   feature has defined that format's fields.

5. X-13 (E39-F04's durable Question lifecycle) is explicitly assigned to
   E40-F08 in `E40-cross-epic-map.md`, and F08's own feature file names it as
   the "durable human-gate surface" for the *post-prelude* keyed lifecycle.
   F07's feature file consumes only I-04 and X-10 — never X-13 — which is
   consistent with D01-D05's human interaction being a live `AskUserQuestion`
   tool call inside one interactive session, not a durable per-entity Question
   record. Conflating the two would misroute F07's replay responses through a
   mechanism owned by a different, later feature.

6. I-05's three-root isolation model (agent-visible checkout / scratch Shark
   project / evaluator-only root, `bench/evidence/i05-schema.yaml`) was built
   for `shark run`-driven stage evidence over fixture-repo code. D01-D05
   artifacts and `docs/product/progress.md` are neither fixture-repo code nor
   held-back evaluator truth — they are planning documents written by a skill
   invocation against the harness's own scratch Shark project. F07 must state
   this placement explicitly rather than let F08 or F09 infer it later when
   they consume I-06 alongside I-05/I-07.

7. E36-F02 (X-10's producer) and E40-F05 (I-04's producer) are both
   `completed`, unlike E27-F15 (the epic's other, still-unmerged, dependency
   named in the epic-level report). F07 inherits no "unmerged branch" risk
   from either of its two direct upstream contracts.

## Decisions

1. **Place the replay adapter at the Rider-adapter layer, not inside the
   bundle.** Matches the wrapped action's own stated checkpoint boundary
   (Finding 1) and keeps the D01-D05 methodology unforked, as the feature file
   requires.

2. **Enumerate the interception surface as exactly five call sites**
   (`AskUserQuestion` at D01/D02/D05, `WebSearch` at D03/D04) rather than a
   generic "any tool call" policy. A narrower, evidence-backed surface is
   easier to verify complete and easier to test against the acceptance
   boundary's "cannot reach live research or an interactive question surface"
   requirement (Finding 2).

3. **Design disallow-plus-substitute as a benchmark-only extension of the
   existing block pattern, not a `internal/runner` Go change.** F07 dispatches
   host-side over the Rider action; reuse the `--disallowedTools` naming
   convention for consistency, but the substitute-answer mechanism is new,
   scenario-scoped harness code (Finding 3).

4. **Define I-06's interior shape as this feature's own deliverable**,
   anchored to I-04's already-reserved `replay_reference` pointer (Finding 4).
   The authoritative shape belongs in
   `architecture.md#product-design-replay-contract`; this feature's spec must
   include the versioned sequence of authorized entries (request match key,
   digest, response payload/reference), consumed-by lineage, and the
   interaction-volume/wait-classification/unresolved-gate proxy fields the
   feature file and UAT-10/UAT-18 already name.

5. **Never route D01-D05 human input through X-13.** Keep F07's Contracts
   section exactly as written (I-04 and X-10 only); any future temptation to
   reuse the Question lifecycle for D01-D05 elicitation should be treated as
   scope creep into F08's territory (Finding 5).

6. **State I-06 artifact placement as the scratch-Shark-project root
   explicitly in the spec**, so F08 and F09 do not have to infer it when they
   later join I-05, I-06, and I-07 (Finding 6).

7. **Proceed without an unmerged-dependency mitigation.** Both direct upstream
   contracts (X-10/E36-F02, I-04/E40-F05) are `completed`; no tracking note is
   needed beyond what the epic-level report already carries for E27-F15
   (Finding 7).

## Sources

- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F07-replayable-product-design-prelude/feature.md`
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/architecture.md` (`#product-design-replay-contract`, `#stage-evidence-and-isolation-contract`, `#lifecycle-run-record-contract`)
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/uat-plan.md` (UAT-10, UAT-18)
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-cross-epic-map.md` (X-10, X-13 rows)
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/research-report.md` (epic-level report)
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F05-lifecycle-scenario-corpus-and-adapter-contract/research-report.md`
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F06-stage-evidence-and-evaluator-isolation/research-report.md`
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F08-canonical-multi-entity-lifecycle-runner/feature.md`
- `docs/plan/E36-project-layer-and-consult-bridge/E36-F02-project-namespace-and-progress-record/feature.md`
- `skills/shark-rider/verbs/product-design.md`
- `internal/sharkdata/default_data/skills/product-design/SKILL.md`
- `internal/sharkdata/default_data/skills/product-design/workflows/d01-vision.md`
- `internal/sharkdata/default_data/skills/product-design/workflows/d03-market-research.md`
- `internal/sharkdata/default_data/skills/product-design/workflows/d05-stakeholder-insights.md`
- `internal/runner/dispatcher.go` (`DefaultDisallowedTools`)
- `internal/runner/claude_dispatcher.go:114-124` (`--allowedTools`/`--disallowedTools`)
- `bench/scenarios/scenarios.yaml`
- `bench/scenarios/packages/py-feature-recurring-tasks/package.yaml`
- `bench/scenarios/packages/py-feature-recurring-tasks/evaluator/replay/reference-bundle.json`
- `bench/evidence/i05-schema.yaml`
- `bench/scripts/verify-evidence-roots.sh`
- `bench/scripts/replay-manifest.sh`
- `internal/sharkdata/default_data/research/recipes.yaml`

RECOMMENDED OUTCOME: pass
