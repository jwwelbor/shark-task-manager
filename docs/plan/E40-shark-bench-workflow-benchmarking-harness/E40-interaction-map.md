---
type: interaction-map
epic: E40
last_updated: 2026-08-05
---

# E40 Cross-Feature Interaction Map

Four features, each with a real trigger, a production path, an observable UAT result, current prerequisites, and outputs for later consumers — tabulated in [architecture](architecture.md#delivery-boundaries-and-traceability). No Phase 1 acceptance criterion depends on later unbuilt behavior; the criteria that did (G6, UAT-03, UAT-04) are restated as Phase 2, not reassigned to a Phase 1 feature.

| ID | Producer feature | Consumer feature(s) | Shape | Payload | Style |
|---|---|---|---|---|---|
| I-01 | E40-F01 Benchmark corpus v1 | E40-F02 Bench harness | [Corpus and oracle contract](architecture.md#corpus-and-oracle-contract) | Machine-readable manifest (issue prompt, entity seed spec, P2P set id, reference patch), held-back F2P test files, pinned fixture base SHA, base-SHA test and lint ledgers | File artifact |
| I-02 | E40-F02 Bench harness | E40-F03 Baseline report and noise band | [Metric collection and artifact schema](architecture.md#metric-collection-and-artifact-schema) | One JSONL record per run: manifest block (item, variant, rep, SHAs, exact model IDs, timeout cap), per-stage records, post-run check results, rollup | File artifact |
| I-03 | E40-F04 shark run live progress and per-run log | E40-F02 Bench harness | [Run liveness contract](architecture.md#run-liveness-contract) | stderr NDJSON progress events (run id, entity key, stage status, action, agent/provider, event, stage elapsed, total elapsed) and `.shark/runs/<run_id>/run.log` | Process stream and file artifact |

All three rows use the default `live` gate mode. No `contract-only` row is declared, and none is available at this phase — that declaration belongs to feature specification.

## Sequencing note on I-01

E40-F01's task decomposition (11 tasks, `T-E40-F01-001` through `T-E40-F01-011`)
mirrors I-01 as `produces`/`validates` in every producer-side task — the
manifest, items, ledgers, `checkout-fixture.sh`, `diff-ledgers.sh`, and
`bench/README.md`. E40-F02, the named consumer, is still `execution_order` 3
and draft; it has not entered its own `task_generation` yet, so no F02 task
exists to declare `I-01: consumes`.

**This is build-order sequencing, not an open gap.** The consumer-side
mirror obligation is recorded as a decision note on E40-F02 (added
2026-08-05): F02's own `task_generation` must create at least one task that
declares `I-01: consumes`, copies the shape source
`architecture.md#corpus-and-oracle-contract` and the contract test
`tests/contracts/e40_i01_corpus_contract_test.go#TC-001` verbatim, and owns
the real caller path for the manifest, `checkout-fixture.sh`, and
`diff-ledgers.sh` (both modes). It cannot be satisfied earlier without
inventing F02 tasks ahead of F02's own spec and test plan.

This does not change I-01's gate mode: the row stays `live`, as declared
above — the edge is not staged or `contract-only` pending F02's
decomposition.

## Sequencing note on I-03

I-03 is consumed in code, not only by an operator. When the timeout cap kills a run, stdout never delivers a `RunResult`, and UAT-05 still requires F02's artifact to name the stalled stage. F04's liveness record is the primary source for that: it carries stage elapsed, agent, and provider, and needs no database access.

It is not a hard dependency. The controller advances status and writes the stage transcript only after the dispatch returns, so a killed run leaves the entity at the executing stage's status in the scratch DB, with the highest-numbered transcript bounding the last completed stage. F02 can meet UAT-05 from that fallback alone.

**Recommendation applied, not a blocker:** decomposition sequenced E40-F04 at `execution_order` 2, ahead of E40-F02 at `execution_order` 3, so F02 builds against F04's liveness record as its primary source instead of the fallback path first. This was weighed against F04's independent value to every `shark run` user (X-08), which does not depend on bench at all — F04 stands on its own merit regardless of ordering.

Decomposition must create feature boundaries matching these rows or update this map before advancing. Feature specs, task specs, feature review, task review, and QA reuse these stable IDs verbatim. Cross-epic seams use X-07 through X-09 in [E40-cross-epic-map.md](E40-cross-epic-map.md); the two ID spaces stay separate.
