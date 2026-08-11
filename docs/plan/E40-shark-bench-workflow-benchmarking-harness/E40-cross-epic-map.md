---
type: cross-epic-integration-map
epic: E40
last_updated: 2026-08-11
---

# E40 cross-epic integration map

E40 benchmarks production Shark and Rider behavior; it does not take ownership
of that behavior. The rows below name the feature that produces each external
contract and the E40 feature accountable for adapting, validating, and failing
loudly when that contract changes. Rows mirror
[docs/product/cross-epic-integration-map.md](../../product/cross-epic-integration-map.md).

| ID | Producer epic | Consumer epic(s) | Integration purpose | Contract / shape source | UX / CX handoff notes | Owning feature | Status | Test coverage pointer |
|---|---|---|---|---|---|---|---|---|
| X-07 | E22 — External Orchestration Runner (E22-F08 Simplify RunController to match architecture v2) | E40 — Shark Bench (E40-F02) | Preserve the completed Phase 1 `shark run` engine contract: JSON `RunResult`/`StageLog`, transcript bytes, and `--workdir` agent cwd | E40 architecture "Run lifecycle and isolation contract" and "Metric collection and artifact schema"; E22-F08 RunController surface | A v1 batch fails with the changed field named instead of publishing corrupt metrics | E40-F02 Bench harness: run driver and metric collection | assigned | E40 UAT-01, UAT-07, and the X-07 canary |
| X-08 | E40 — Shark Bench (E40-F04) | E22 — External Orchestration Runner; no single consumer feature owns all stdout callers | Extend `shark run` observability without changing JSON stdout: stderr progress, stage heartbeats, child labels, and an unconditional per-run log | E40 architecture "Run liveness contract"; `internal/cli/commands/run.go` progress callback and ticker | Every `shark run` user gains liveness while stdout remains one `RunResult`. E40-F04 owns the non-regression proof; E22-F08 must keep it green when changing the same path. | E40-F04 `shark run` live progress and per-run log | assigned | E40 UAT-06 and X-08 |
| X-09 | E27 — Shark Status Viewer (E27-F15 Codex and Claude cross-session usage tracking) | E40 — Shark Bench (E40-F06 contract owner; E40-F08 runtime writer) | Reuse the audited provider-usage field mapping for stage evidence and comparison identity; do not invent missing token, cost, model, session, or timing fields | E40 architecture "Stage evidence and isolation contract"; E27-F15 approved usage metadata contract and implementation artifacts | Internal. Missing required provider identity invalidates the run instead of degrading to an incomparable aggregate. | E40-F06 Stage evidence and evaluator isolation | assigned | E40 UAT-09 and UAT-14; consumer contract tests are owned by the E40-F06/E40-F08 workflows |
| X-10 | E36 — Project Layer and Consult Bridge (E36-F02 Project namespace and progress record) | E40 — Shark Bench (E40-F07) | Invoke the existing Shark Rider product-design action and progress record for D01-D05 rather than copying the methodology into the benchmark | E36-F02 feature contract; Shark Rider product-design adapter; E40 architecture "Product-design replay contract" | Scored feature scenarios use frozen inputs, but generated artifacts remain ordinary product-design artifacts and progress remains derived from disk. | E40-F07 Replayable product-design prelude | assigned | E40 UAT-10 |
| X-11 | E38 — Shark Attack Team Orchestration (E38-F07 Rider Execution and Escalation Loop; E38-F09 Provider-Neutral Coordination and Live Resume) | E40 — Shark Bench (E40-F08) | Reuse the canonical host-side keyed loop: dispatch response, claim, unchanged prompt, heartbeat, semantic outcome, transition, release, prompt provenance, and bounded resume | E38-F07 and E38-F09 feature contracts; Shark Rider run procedure; E40 architecture "Lifecycle v2 controller boundary" | The benchmark records and schedules the loop but does not create a second workflow engine, claim store, or prompt assembler. | E40-F08 Canonical multi-entity lifecycle runner | assigned | E40 UAT-11 and UAT-12 |
| X-12 | E32 — Shark 2.0 Single-Artifact Consolidation (E32-F04 Migrate canonical content into shark-data) | E40 — Shark Bench (E40-F09) | Derive one installed-content identity covering workflows, prompts, skills, and agents so comparisons cannot mix content bundles | E32-F04 Shark-data canonical-content contract; E40 architecture "Lifecycle evaluation record contract" | Operators see a comparison rejected with the differing content digest rather than a misleading quality delta. | E40-F09 Calibrated evaluation and comparison identity | assigned | E40 UAT-14 |
| X-13 | E39 — Question and Decision Workflow Management (E39-F04 Focused Question Read Surfaces and Consumer Handoff) | E40 — Shark Bench (E40-F08) | Use the durable Question lifecycle for unresolved decisions and replay-authorized responses during a scored lifecycle | E39 architecture and E39-F04 consumer handoff; E40 architecture "Lifecycle run record contract" | Missing authorized input remains a visible `unresolved_gate`; the benchmark never invents a decision or hides the blocking Question in transcript-only state. | E40-F08 Canonical multi-entity lifecycle runner | assigned | E40 UAT-12 |

## Notes

- X-07 and X-08 preserve the completed Phase 1 `shark run` path. Lifecycle v2
  does not delete or relabel that path.
- X-09 now has consumer ownership. E27-F15 is still active, so E40-F06 must
  verify its current artifact and implementation state before depending on it.
- X-10, X-11, and X-13 are behavioral reuse seams. Benchmark-only adapters live
  in E40; any missing generic production behavior is separate work under the
  producing epic.
- X-12 identifies installed Shark-data content. It does not require E32 to add a
  benchmark-specific digest API if E40 can deterministically hash the installed
  canonical tree.
- E40 adds no E23 observability row. Its retained evidence and run artifacts are
  benchmark outputs, not a change to the product observability contract.
