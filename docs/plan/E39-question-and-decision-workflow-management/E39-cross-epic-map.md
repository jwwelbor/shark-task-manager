---
type: cross-epic-integration-map
epic: E39
last_updated: 2026-07-30
---

# E39 Cross-Epic Integration Map

Architect and CX reviews found one cross-epic producer seam. E39 supplies the generic lifecycle; it does not repair E38 host adapters or continuation.

| ID | Producer epic | Consumer epic(s) | Integration purpose | Contract / shape source | UX / CX handoff notes | Owning feature | Status | Test coverage pointer |
|---|---|---|---|---|---|---|---|---|
| X-06 | E39 — Question and Decision Workflow Management (E39-F04) | E38 — Shark Attack Team Orchestration (E38-F09, activation owner) | Supply a durable serial Question lifecycle, scoped blocking visibility, and authoritative resolution for provider-neutral live-question handling | E39 architecture §2–§4; E38-F09 feature.md | One scoped responder prompt; compact blocked-work handoff; Question state rather than chat/council copies supports resume | Producer: E39-F04 Focused Question Read Surfaces and Consumer Handoff; consumer activation: E38-F09 Provider-Neutral Coordination and Live Resume | assigned | E39 uat-plan.md UAT-01–06 and X-06; E38-F09 remains blocked on E39 and must add consumer coverage when resumed |

No other X-## rows were identified during design on 2026-07-30. E39 decomposition assigns E39-F04 as the producer-facing contract owner and E38-F09 as the blocked activation owner; E38-F09 must add its consumer coverage when it resumes.
