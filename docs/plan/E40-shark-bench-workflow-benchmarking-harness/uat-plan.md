---
epic: E40
title: Shark Bench UAT plan
date: 2026-08-05
---

# E40 UAT Plan

Scenarios verify observable behavior, not implementation. Phase 1 exit is gated on UAT-01, UAT-02, UAT-05, UAT-06, and UAT-07 only.

| ID | Scenario | Verify | Epic criterion |
|---|---|---|---|
| UAT-01 | Unattended baseline batch produces a report with a noise band | An operator starts the 10 x 3 batch and leaves. The batch completes with no intervention, every run yields a JSONL artifact, and the report states per headline metric both the baseline value and the observed run-to-run spread. Re-running the batch skips already-completed pairs instead of duplicating them. | G2, G4, G5 |
| UAT-02 | A broken corpus item is rejected at admission | A candidate whose reference patch leaves F2P red, or whose P2P set is already red at the base commit, is rejected with the failing check named, and appears in no benchmark run. Re-running the gate on a clean checkout gives the identical verdict. | G1 |
| UAT-05 | A stuck run is bounded, recorded, and does not hang the batch | A run whose stage exceeds the timeout cap terminates; its artifact records `outcome=timeout` **and names the stage it stalled in**; the batch proceeds to the next run. No stdout `RunResult` is available in this case, so the stage attribution comes from F04's liveness record, or failing that from the entity's status in the scratch DB. | G2, G4 |
| UAT-06 | An in-flight run is observable | Watching stderr or tailing `.shark/runs/<run_id>/run.log` during a `--json` run shows per-stage progress with the entity key, stage status, agent and provider, and stage plus total elapsed — while stdout still parses as exactly one `RunResult` document. During a cascade the child key is shown, not the parent's. A hung agent is distinguishable from a slow one: heartbeats continue with growing stage-elapsed and unchanged stage. | G3 |
| UAT-07 | Results reproduce from a stored manifest | Re-running a stored manifest — pinned fixture SHA, pinned model IDs, same variant bundle — reproduces the reported metrics within the published noise band. The report regenerates from the artifact directory alone, with no state outside it. | G7 |

## Deferred to Phase 2

UAT-03 (a delta inside the noise band is reported as "no detectable effect") and UAT-04 (a delta outside the band is reported as a measured change) both require variant bundles and the paired per-task comparison report. PRD section 3 defers those to Phase 2 and E40-F03 excludes them explicitly, so **G6, UAT-03, and UAT-04 are Phase 2 criteria** — see Q002. Phase 1 delivers the noise band those scenarios later measure against; asserting them at Phase 1 exit would assert behavior no Phase 1 feature builds.

## Cross-feature and cross-epic scenarios

- **I-01 (F01 → F02)**: the harness seeds and scores a run from the manifest with no manual step, and counts regressions and *new* lint issues against the base-SHA ledgers rather than absolute counts.
- **I-02 (F02 → F03)**: F03 aggregates from artifacts alone. A record missing a metric family fails aggregation loudly instead of being silently averaged away.
- **I-03 (F04 → F02)**: covered by UAT-05 and UAT-06 together — F04's liveness record is the primary way a timed-out run names its stalled stage, with the scratch-DB status as the fallback.
- **X-07 (E22 → E40-F02)**: the canary check fails loud when the `RunResult`, `StageLog`, or transcript byte format changes upstream, rather than yielding silently wrong metrics.
- **X-08 (E40-F04 → E22)**: existing `shark run --json` consumers — skills, agents, the runner's own callers — still parse stdout unchanged after F04 ships.

## Non-functional evidence

**Integrity.** Every run provisions through `scripts/shark-scratch-env.sh`; the live repo, its `.sharkconfig.json`, and the live database are untouched, and the config guardrail hook stays satisfied. Held-back tests never exist in the checkout the agent sees — verified by inspecting the working tree at dispatch time, not only by intent. Workers cannot self-advance status, so a recorded outcome is the workflow's, not the worker's.

**Cost and safety.** Every run terminates under the timeout cap, so a batch has a bounded worst case. Exact model IDs are recorded per run; a comparison spanning unpinned model versions is reported as invalid rather than as a result.

**Performance.** Not a product concern here — the harness is offline tooling. The one latency claim that matters is F04's: heartbeats appear at least every 10 seconds while a stage runs.
