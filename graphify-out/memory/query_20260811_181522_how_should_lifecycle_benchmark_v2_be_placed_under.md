---
type: "architecture"
date: "2026-08-11T18:15:22.259247+00:00"
question: "How should lifecycle benchmark v2 be placed under E40, and which epic and cross-feature contracts need updating?"
contributor: "graphify"
outcome: "useful"
source_nodes: ["FeatureService", ".CreateFeature()", ".maybeReopenParentEpic()", "Feature", "Epic"]
---

# Q: How should lifecycle benchmark v2 be placed under E40, and which epic and cross-feature contracts need updating?

## Answer

Created E40-F05 through E40-F10 as draft planning features. E40 reopened to active through FeatureService parent maintenance. Recorded internal and cross-epic dependencies, related documents, I-04 through I-08 contracts, X-09 through X-13 ownership, UAT-08 through UAT-15, execution orders 5 through 10, fail-closed evidence identity, replay and isolation boundaries, and safe retained-baseline requirements. Preserved F01-F04 as completed Phase 1 and did not start the new feature workflows.

## Outcome

- Signal: useful

## Source Nodes

- FeatureService
- .CreateFeature()
- .maybeReopenParentEpic()
- Feature
- Epic