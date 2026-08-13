---
type: "query"
date: "2026-08-11T17:24:02.446531+00:00"
question: "review dev-artifacts/2026-08-11-lifecycle-benchmark-v2-plan.md"
contributor: "graphify"
outcome: "useful"
source_nodes: ["Execution mode: /shark-rider run", "Shark Attack protocol", "TransitionOptions", "Decomposition Criteria reference", "task_lifecycle_test.go"]
---

# Q: review dev-artifacts/2026-08-11-lifecycle-benchmark-v2-plan.md

## Answer

Expanded from original query via graph vocabulary: benchmark, lifecycle, oracle, scenario, artifact, workflow, fixture, decomposition, execution, provenance, snapshot, transition. Review found five implementation gaps: shark next is only one keyed dispatch step and the parent loop must own claim, heartbeat, exact prompt dispatch, semantic outcome advance, and release; D01 through D05 require a separate Rider and deterministic stakeholder or tool replay layer and do not apply uniformly to bug, change, and tech-debt entities; all reference artifacts must be evaluator-only rather than merely hiding execution tests; aggregate identity must pin the complete content, model, tool, adapter, environment, and judge configuration; provider-backed commands need explicit spend acknowledgement and bounded invalid-run ceilings.

## Outcome

- Signal: useful

## Source Nodes

- Execution mode: /shark-rider run
- Shark Attack protocol
- TransitionOptions
- Decomposition Criteria reference
- task_lifecycle_test.go