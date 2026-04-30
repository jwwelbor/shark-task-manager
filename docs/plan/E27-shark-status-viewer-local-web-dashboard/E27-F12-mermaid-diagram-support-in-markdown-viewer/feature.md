---
feature_key: E27-F12-mermaid-diagram-support-in-markdown-viewer
epic_key: E27
title: Mermaid Diagram Support in Markdown Viewer
description: Render mermaid code blocks as diagrams in the web viewer's markdown panel.
---

# Mermaid Diagram Support in Markdown Viewer

**Feature Key**: E27-F12

## Observation

The web viewer renders markdown via `marked` (loaded from jsDelivr in `internal/viewer/assets/viewer.html:7`), but ` ```mermaid ` fenced code blocks are emitted as plain text instead of rendered diagrams. Architecture and design docs across the project rely on mermaid for sequence/flow/state diagrams, so they currently appear as unreadable source in the viewer.

## Where

- `internal/viewer/assets/viewer.html` — markdown rendering pipeline (search for `marked`, `markdown-body`).

## Why It Matters

Most epic and feature design docs (e.g. `docs/plan/E27-*/architecture.md` and others) embed mermaid blocks. Without rendering, the viewer's markdown panel is significantly less useful for design review.

## Out of Scope

To be decided during refinement.
