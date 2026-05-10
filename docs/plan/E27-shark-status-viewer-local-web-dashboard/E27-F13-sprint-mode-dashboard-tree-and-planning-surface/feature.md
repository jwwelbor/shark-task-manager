---
feature_key: E27-F13-sprint-mode-dashboard-tree-and-planning-surface
epic_key: E27
title: Sprint Mode Dashboard, Tree, and Planning Surface
description: Add a dedicated Sprint mode to shark web with overview, plan, and report subviews; sprint-tree navigation; and a read-first planning surface inside the existing local viewer.
size:
---

# Sprint Mode Dashboard, Tree, and Planning Surface

**Feature Key**: E27-F13

## Seed Description

Add a dedicated Sprint mode to the Shark local web viewer that complements the existing entity dashboard instead of replacing it. The initial concept is a read-first operational command board with three subviews, `Overview`, `Plan`, and `Report`, plus sprint-aware navigation in the left rail.

## Initial Scope For Refinement

- Add a top-level `Sprint` mode alongside the current dashboard/entity views.
- Make `Overview` the default landing state for current sprint health, live work, blockers, and recent changes.
- Add `Plan` and `Report` subviews for backlog shaping, capacity visibility, burndown, velocity, and trend reporting.
- Integrate a sprint tree into the existing left rail so sprint context and entity hierarchy remain visible together.
- Keep V1 frameworkless and aligned with the existing embedded `viewer.html` + vanilla JS architecture unless refinement finds a strong reason to change that.
- Treat V1 as interactive but read-first: filtering, inspection, selection, and scoped planning actions are in scope; deeper mutation can be deferred if needed.

## Why It Matters

The current web UI is useful for entity inspection, but it does not expose sprint work as a first-class surface. The attached recommendation and wireframes define a sprint-oriented view that makes current work, blockers, readiness, and capacity visible without forcing users back into CLI-first planning workflows for every inspection task.

## Source Documents

- [Sprint Planning Web UI Recommendation](../sprint-planning-web-ui-recommendation.md)
- [Sprint Planning Web UI Wireframes](../sprint-planning-web-ui-wireframes.md)

## Out of Scope For This Seed

- Writing the full PRD in this file ahead of the Shark refinement workflow.
- Committing to a framework migration before refinement.
- Expanding V1 into a fully mutation-centered planning tool before the refinement stages validate that scope.
