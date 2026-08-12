---
research_schema: 2
entity_key: E40-F05
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

# Research report: Lifecycle scenario corpus and adapter contract

## Scope

E40-F05 defines **I-04**, the versioned scenario-package schema (scenario ID
and version, entity family, stage-applicability matrix, fixture SHA, adapter
identity, agent-visible input, replay/evaluator references, resource policy,
and admission result) that E40-F06, E40-F07, and E40-F08 consume read-only. It
adds one admitted seed for each of the four lifecycle families (feature, bug,
change-card, tech-debt) on a new controlled Python fixture, preserves the
existing Go fixture as a compatibility adapter, and defines a language-neutral
execution-adapter boundary so no generic lifecycle component branches on
Python vs. Go. The schema's shape is already pinned by
`architecture.md#lifecycle-scenario-package-contract` and the I-04 row in
`E40-interaction-map.md` — F05 implements a shape the epic has already fixed,
not an open design, the same posture the sibling F01 research report recorded
for I-01.

F05 consumes **I-01** (`bench/corpus/corpus.yaml`, the completed v1 corpus and
oracle contract) for its admission/versioning *principles* only;
`architecture.md` and `feature.md` both state explicitly that F05 must not
make the v1 Go schema the global format. It does not touch Shark's own Go
code — `architecture.md`'s component table lists F05 as "Extend the bench
corpus," the same "no Shark change" posture as F01–F03. Its output is new
schema, fixture, and adapter tooling under `bench/`, not `internal/`.

## Research checklist

- [x] `scope_vocabulary` — Evidence: `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F05-lifecycle-scenario-corpus-and-adapter-contract/feature.md` (Scope, Acceptance boundary, Contracts, Out of scope) and `architecture.md#lifecycle-scenario-package-contract` define I-04's exact field list, the stage-applicability matrix, and the "no Python-specific commands in Shark's workflow contracts" boundary that names the adapter concept.
- [x] `affected_implementation_or_contract` — Evidence: `bench/corpus/corpus.yaml` (the I-01 schema F05 must not globalize — its `fixture.toolchain` block and `p2p_sets` are Go/golangci-lint-shaped); `bench/scripts/build-ledgers.sh` and `bench/scripts/diff-ledgers.sh` (hardcode `go test -json` / `golangci-lint run` with no language branch point — the concrete surface F05's adapter boundary must generalize); `bench/scripts/checkout-fixture.sh` and `.gitmodules` (the existing Go fixture-repo submodule pattern at a pinned SHA — the structural precedent for a second Python fixture); `tests/contracts/e40_i01_corpus_contract_test.go` (the Go-only I-01 validator, confirming there is no generic reader to extend); `internal/runner/dispatcher.go` (`AgentDispatcher` interface — the one existing "one interface, N concrete tool implementations" precedent in this codebase) and `internal/runner/claude_dispatcher.go` / `codex_dispatcher.go` (its two concrete implementations).
- [x] `related_work` — Evidence: `architecture.md` (Lifecycle scenario package contract, Delivery boundaries table, ADR list); `E40-interaction-map.md` (I-04 row: producer F05, consumers F06/F07/F08); `E40-cross-epic-map.md` (X-09 row: E27-F15 usage-field mapping, status `assigned`, owning feature F06 not F05); parent epic `research-report.md` (Capability map E27-F15/B051 rows, Decisions 2–3); sibling `E40-F01-.../research-report.md` (I-01's shape and its own "NEW, not reusable elsewhere" capability-map verdict); sibling feature files `E40-F06`, `E40-F07`, `E40-F08` (I-04 consumer contracts); `shark get <key> --field status` for E27-F15, B051, E38-F07, E38-F09, E39-F04, E36-F02, E32-F04; `git log`/`git branch --all --contains` for the E27-F15 branch and B051's landed fix.
- [x] `pattern_contract` — Evidence: `internal/runner/dispatcher.go`'s `AgentDispatcher` interface (`Name`, `BuildCommand`, `Dispatch`) implemented separately by `ClaudeDispatcher` and `CodexDispatcher` — a directly analogous "select the concrete tool behind one interface, driven by config" pattern already established in this codebase, even though the domain (agent-CLI dispatch) differs from fixture-command execution; `bench/corpus/corpus.yaml`'s own `schema_version` field and `bench/README.md`'s "Manifest schema" section — the local precedent for a versioned, documented corpus schema that I-04 should follow in form.
- [x] `cross_boundary_risks` — Evidence: `bench/scripts/build-ledgers.sh`/`diff-ledgers.sh` toolchain-identity block (Go version, `golangci-lint` version, `GOOS`/`GOARCH`, config hash) vs. the absence of any equivalent Python toolchain-identity concept anywhere in the repo; `feature.md`'s own boundary ("without embedding Python-specific commands in Shark's workflow contracts") vs. `architecture.md#stage-evidence-and-isolation-contract`'s dispatch-time absence requirement for evaluator-only material, which F06 inherits directly from however I-04 encodes evaluator references; the confirmed-unmerged E27-F15 branch (below) as a live boundary between F05's schema commitments and X-09's still-undecided usage-field mapping.
- [x] `alternatives` — Evidence: `architecture.md#lifecycle-scenario-package-contract` ("I-04 extends the Phase 1 corpus principles... not making the Go manifest the global benchmark shape") and direct inspection of `bench/corpus/corpus.yaml`'s Go-specific toolchain fields (rejects retrofitting I-01); `feature.md`'s Outcome statement (rejects embedding Python commands in workflow YAML); `internal/runner/dispatcher.go` as the chosen adapter shape versus an unproven alternative interface design.

## Capability map

| Capability | Brownfield evidence | Decision | F05 responsibility |
|---|---|---|---|
| I-01 corpus/oracle schema and admission principles (`bench/corpus/corpus.yaml`, `schema_version`, admission-gate semantics) | `bench/corpus/corpus.yaml`; `bench/README.md` "Manifest schema"; `tests/contracts/e40_i01_corpus_contract_test.go`; `architecture.md#lifecycle-scenario-package-contract` | EXTEND (principles only) | Reuse the versioned-schema and reproducible-admission *shape*, not the literal Go/golangci-lint fields — I-04 is its own schema, confirmed necessary by inspecting I-01's toolchain block, not just asserted by the architecture doc. |
| Go fixture-repo and its ledger/diff tooling (`bench/fixture-repo` submodule, `build-ledgers.sh`, `diff-ledgers.sh`, `checkout-fixture.sh`) | `.gitmodules`; `bench/scripts/build-ledgers.sh:180-194` (`go test -json`, `golangci-lint run`); `bench/scripts/diff-ledgers.sh:99-119` (golangci-lint version parsing, toolchain guard) | REUSE, unmodified, as the compatibility adapter | Keep these scripts working exactly as-is for the Go family; route selection of *which* command set runs through the new adapter rather than rewriting Go-specific logic. |
| Execution-adapter shape | `internal/runner/dispatcher.go` `AgentDispatcher` interface; `internal/runner/claude_dispatcher.go`, `codex_dispatcher.go` | REUSE (pattern only, not code) | Model the fixture execution-adapter as an analogous small interface (adapter name/version, command resolution) with one Go implementation and one Python implementation, matching this codebase's one existing precedent for the same kind of indirection. |
| Fixture-repo submodule convention | `.gitmodules` (`bench/fixture-repo` → pinned external GitHub repo) | REUSE | Add the Python fixture as a second submodule at a pinned SHA, following the same structural pattern rather than vendoring it inline. |
| `check_or_resume` tech-debt dispatch (B051) | `internal/runner/controller.go:517` (`case config.ActionSpawnAgent, config.ActionCheckOrResume:` — dispatches, not paused); `shark get B051 --field status` = `completed` | REUSE (now landed, no longer pending) | Admit the tech-debt seed scenario without a tracked precondition — the parent epic research report's "track as external dependency" caveat for B051 no longer applies; confirm by direct grep of `controller.go` rather than trusting the DB status alone, since both agree here. |
| Provider-usage envelope decode (X-09, E27-F15) | `shark get E27-F15 --field status` = `active`; `git branch --all --contains <E27-F15 commit>` shows it only on branch `E27-F15-cross-session-usage-tracking`; `internal/runner/claude_dispatcher.go` on `main` has zero hits for `modelUsage`/`num_turns`/`duration_api_ms`/`total_cost_usd` | CONTRADICTS current assumption if treated as decided | I-04's evaluator/usage reference fields must stay opaque references (per `architecture.md`'s "evaluator-only reference and execution-oracle references" wording), not typed fields naming specific decoded envelope keys — that decode remains F06/X-09's unresolved job, still genuinely unmerged today. |
| Rider execution loop (X-11), Question lifecycle (X-13), product-design action (X-10), canonical Shark-data bundle (X-12) | `shark get E38-F07/E38-F09/E39-F04/E36-F02/E32-F04 --field status` = `completed` for all five | REUSE (downstream, not F05's own surface) | I-04 must stay declarative (stage-applicability matrix, references) so F06–F09 can invoke these already-completed capabilities without I-04 encoding dispatch mechanics that would duplicate Rider/Question routing. |
| Top-level committed fixture directory convention (`test-fixtures/uat-test-project`) | Carried from sibling F01 research report Capability map (CONTRADICTS/limited transferability — different domain, a shark-project fixture not a benched service) | CONTRADICTS | Confirmed still not directly reusable; the fixture-repo submodule convention (above) is the relevant precedent instead. |

## Findings

1. **Nothing in `bench/`'s existing Go tooling has a language branch point today.** `build-ledgers.sh` and `diff-ledgers.sh` invoke `go test -json` and `golangci-lint run` unconditionally, with no config-driven command selection anywhere in the script. F05's "language-neutral adapter boundary" is a genuinely new abstraction layer these scripts lack, not a refactor of an existing one — confirming the feature scope's framing rather than merely restating it.

2. **I-01's schema is concretely, not just declaratively, Go-shaped.** `bench/corpus/corpus.yaml`'s `fixture.toolchain` block (`go_version`, `golangci_lint_version`, `goos`, `goarch`, `golangci_config_sha256`) and `p2p_sets` (Go package patterns, `-run` selectors) have no Python analog, and its validator (`tests/contracts/e40_i01_corpus_contract_test.go`) decodes exactly these Go-typed fields. `architecture.md`'s instruction not to make the v1 Go manifest the global format is therefore load-bearing, not precautionary — a literal reuse attempt would break on its own toolchain semantics before reaching any Python-specific gap.

3. **B051 is fixed and merged to `main` today, not merely code-reviewed on an unmerged branch as the parent epic research report (dated 2026-08-05, before this feature was created) recorded.** `internal/runner/controller.go:517` shows `ActionCheckOrResume` grouped with `ActionSpawnAgent` in the dispatch switch, matching `shark get B051 --field status` = `completed`. Tech-debt scenario admission — one of F05's four required seed families — is unblocked at the execution-engine layer with no outstanding merge risk.

4. **E27-F15 (X-09) remains genuinely unmerged and undecoded on `main`.** `git branch --all --contains` the E27-F15 commit shows it lives only on `E27-F15-cross-session-usage-tracking`; a grep of the on-`main` `internal/runner/claude_dispatcher.go` for `modelUsage`, `num_turns`, `duration_api_ms`, and `total_cost_usd` returns nothing. Unlike B051, this dependency's status is unchanged from the parent report's earlier caution — F05 should not assume any specific decoded-usage field name exists when shaping I-04's evaluator/usage reference fields.

5. **The codebase has exactly one adapter-shaped precedent for this exact kind of problem.** `internal/runner.AgentDispatcher` (three methods: `Name`, `BuildCommand`, `Dispatch`) is implemented separately by `ClaudeDispatcher` and `CodexDispatcher`, selected by configuration rather than by branching on tool identity inline. It is a different domain (agent-CLI dispatch, not fixture test/lint execution) so its code is not directly reusable, but it is the one local idiom this repo already uses for "swap the concrete tool behind one small interface."

6. **The controlled Python fixture is genuinely greenfield.** No `pytest`, `requirements.txt`, or `pyproject.toml` exists anywhere in this repository outside prose mentions in skill/prompt documentation (`skills/shark-rider/HOOKS.md`, various `internal/sharkdata/default_data/skills/*` files) — none of which are executable fixtures. The one transferable structural precedent is the existing Go fixture's submodule pattern (`.gitmodules` → `bench/fixture-repo`, pinned SHA, cloned by `checkout-fixture.sh`), which generalizes directly to a second, Python fixture submodule.

7. **All four other cross-epic capabilities I-04's consumers depend on are already `completed`, unlike E27-F15.** `shark get` confirms `completed` status for E38-F07, E38-F09 (X-11, Rider loop), E39-F04 (X-13, Question lifecycle), E36-F02 (X-10, product-design action), and E32-F04 (X-12, canonical Shark-data bundle). None of F05's downstream consumers (F06–F09) inherit an unmerged-capability risk from these four the way F06 does from E27-F15/X-09.

## Decisions

1. **Design I-04 as its own versioned schema; reuse I-01's admission/versioning principles, not its fields.** Confirmed necessary, not just architecturally mandated, by inspecting `corpus.yaml`'s Go/golangci-lint-specific toolchain block directly.

2. **Model the execution-adapter interface on `internal/runner.AgentDispatcher`'s shape** (one small interface; one concrete implementation per language; selection driven by scenario package data, not inline branching) rather than inventing an unrelated abstraction — it is the one existing precedent in this codebase for the same category of problem.

3. **Route the existing Go-specific bench scripts (`build-ledgers.sh`, `diff-ledgers.sh`, `checkout-fixture.sh`) behind the new adapter unmodified rather than rewriting them.** The feature's own scope ("preserve the existing Go fixture as a compatibility adapter") is achievable by wrapping: nothing in the current scripts needs to change for the Go family to keep working; only the point where a generic lifecycle component chooses which command set to invoke needs adapter indirection.

4. **Keep I-04's evaluator/usage reference fields opaque (pointers, not typed decoded-usage fields).** E27-F15/X-09 remains unmerged and undecoded on `main` today; baking specific field names into I-04 now would commit F05 to a shape F06/X-09 has not yet verified.

5. **Treat B051 as resolved infrastructure when admitting the tech-debt seed family, not as a tracked external dependency.** The parent epic research report's caution predates this feature and predates the merge; direct inspection of `controller.go` on `main` confirms the fix has landed.

6. **Add the Python fixture as a second `.gitmodules` submodule at a pinned SHA**, following the same structural convention as `bench/fixture-repo`, rather than vendoring Python source inline or inventing a different fixture-hosting convention.

## Sources

- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F05-lifecycle-scenario-corpus-and-adapter-contract/feature.md`
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/epic.md`
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/architecture.md` (Lifecycle scenario package contract, Delivery boundaries, ADR-001–ADR-008)
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-interaction-map.md` (I-04 row)
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-cross-epic-map.md` (X-09 through X-13 rows)
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/shark-bench-design.md`
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/research-report.md` (parent epic report — Capability map E27-F15/B051 rows, Decisions 2–3)
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F01-benchmark-corpus-v1-fixture-repo-and-screened-task/feature.md` and `research-report.md`
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F06-stage-evidence-and-evaluator-isolation/feature.md`
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F07-replayable-product-design-prelude/feature.md`
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F08-canonical-multi-entity-lifecycle-runner/feature.md`
- `bench/corpus/corpus.yaml` (I-01 schema; `fixture.toolchain`, `p2p_sets`)
- `bench/README.md` (Manifest schema, ledger diff method)
- `tests/contracts/e40_i01_corpus_contract_test.go` (I-01 validator, Go-typed structs)
- `bench/scripts/build-ledgers.sh`, `bench/scripts/diff-ledgers.sh`, `bench/scripts/checkout-fixture.sh`
- `.gitmodules` (`bench/fixture-repo` submodule pattern)
- `internal/runner/dispatcher.go` (`AgentDispatcher` interface)
- `internal/runner/claude_dispatcher.go`, `internal/runner/codex_dispatcher.go`
- `internal/runner/controller.go:517` (`ActionCheckOrResume` dispatch grouping — B051 fix confirmed on `main`)
- `shark get E40-F05 --json` (complexity triage note: COMPLEX, score 19/27)
- `shark get B051 --field status`, `shark get E27-F15 --field status`, `shark get E38-F07/E38-F09/E39-F04/E36-F02/E32-F04 --field status`
- `git branch --all --contains <E27-F15 commit>`; `git log --oneline --all | grep -i "cross-session-usage"`
- `test-fixtures/uat-test-project` (carried-forward negative precedent from F01 research report)

RECOMMENDED OUTCOME: pass
