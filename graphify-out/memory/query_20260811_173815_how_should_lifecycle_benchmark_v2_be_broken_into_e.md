---
type: "query"
date: "2026-08-11T17:38:15.597115+00:00"
question: "How should lifecycle benchmark v2 be broken into epics and features, including whether it belongs under E40?"
contributor: "graphify"
outcome: "useful"
source_nodes: ["FeatureService", ".CreateFeature()", ".maybeReopenParentEpic()", "Epic", "Feature"]
---

# Q: How should lifecycle benchmark v2 be broken into epics and features, including whether it belongs under E40?

## Answer

Keep lifecycle benchmark v2 under E40 as six outcome-oriented features: scenario corpus and adapters; stage evidence and evaluator isolation; replayable product-design prelude; canonical multi-entity lifecycle runner; calibrated evaluation and comparison identity; operator workflow and retained baseline. E40 already reserves later phases for broader entities and judging. Creating a feature under a terminal epic reopens the parent through aggregate maintenance, so use a new epic only if E40 must remain an immutable release milestone.

## Outcome

- Signal: useful

## Source Nodes

- FeatureService
- .CreateFeature()
- .maybeReopenParentEpic()
- Epic
- Feature