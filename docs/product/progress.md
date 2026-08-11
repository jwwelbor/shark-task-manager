# Product Progress

**Last updated:** 2026-08-11
**Updated by:** E40 lifecycle v2

## Cross-epic integration map

- Present: [cross-epic-integration-map.md](./cross-epic-integration-map.md)
- Last updated: 2026-08-11
- Updater: E40 lifecycle v2

## Decision log

| Date | Area | Decision |
|---|---|---|
| 2026-07-13 | Cross-epic integrations | Registered the global X-01 through X-05 rows for E38’s dispatch, workflow, sprint, observability, and Shark-data boundaries; mirrored them in E38-cross-epic-map.md. |
| 2026-07-30 | Cross-epic integrations | Added proposed X-06 for E39's Question lifecycle as E38-F09's future consumer contract; feature/activation ownership and consumer coverage are deferred to E39 decomposition. |
| 2026-08-05 | Cross-epic integrations | E40 design added X-07 (E22 `shark run` contract consumed by E40-F02, covered by a canary check), X-08 (E40-F04 extends `shark run` observability in the producer direction while preserving the stdout `RunResult` contract for existing E22 consumers), and proposed X-09 (E27-F15's unmerged usage decoder as Phase 2 G1's reuse candidate). X-07 and X-08 are assigned and mirrored in E40-cross-epic-map.md; X-09 remains `proposed` with owning feature and test coverage TBD, to be closed by Phase 2 decomposition or test planning. No E23 row was created: E40-F04's unconditional per-run log deliberately bypasses the observability config rather than extending E23's contract. |
| 2026-08-05 | Cross-epic integrations | E40 decomposition rework (post feature_review FAIL) named concrete feature-level ownership on both assigned E22 seams instead of an epic-wide audience: X-07's producer is E22-F08 "Simplify RunController to match architecture v2" — its title names RunController and its live tasks (`T-E22-F08-001/002/003`, `T-E22-F08-007`) touch the exact dispatch functions and worktree-isolation path that populate `RunResult`/`StageLog`. X-08 has no legitimate E22 consumer feature — `shark run --json` stdout consumers (skills, agents, CLI callers) are dispersed system-wide, not owned by any one E22 feature — so its activation owner is named as E40-F04 itself, self-accountable via its own UAT-06 stdout-preservation check, with a forward obligation recorded on E22-F08 to keep that assertion green when it next touches `controller.go`. Both maps updated verbatim-identically (E40-cross-epic-map.md and this file). |
| 2026-08-11 | Cross-epic integrations | E40 lifecycle v2 preserved X-07/X-08 as completed Phase 1 seams; assigned X-09 to E40-F06 with E40-F08 as runtime writer; and added X-10 (E36-F02 product-design action), X-11 (E38-F07/F09 keyed Rider loop), X-12 (E32-F04 canonical Shark-data identity), and X-13 (E39-F04 Question lifecycle). All new rows name E40 consumer ownership and UAT coverage; generic production changes remain under their producer epics. |
