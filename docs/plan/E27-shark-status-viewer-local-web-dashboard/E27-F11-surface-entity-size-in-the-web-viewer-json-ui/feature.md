---
feature_key: E27-F11-surface-entity-size-in-the-web-viewer-json-ui
epic_key: E27
title: Surface entity size in the web viewer (JSON + UI)
description: Show entity size in shark web — extend FlatEntity DTO and render size in viewer.html.
---

# Surface entity size in the web viewer (JSON + UI)

**Feature Key**: E27-F11

## Context

E07-F42 added a numeric `size` field (Fibonacci 1/2/3/5/8/13, t-shirt labels XS/S/M/L/XL/XXL) to all entity tables and CLI commands. A follow-up branch (`feat/update-size-flag-and-td-size`) wired `--size` into the unified `shark update` dispatch and added full tech-debt size support (model/repo/service/CLI/migration v15→v16).

Today, `size` shows up correctly in the CLI (`shark <entity> get`, `shark task list`, etc.) but is only **partially visible in the web viewer**:

| Surface | Status |
|---|---|
| Hierarchy JSON for epic / feature / task | ✅ exposed automatically (DTOs embed `*models.Epic` / `*models.Feature` / `*models.Task`, whose `BaseEntity.Size` carries `json:"size,omitempty"`) |
| Hierarchy JSON for bug / change-card / idea | ❌ stripped — `FlatEntity` in `internal/services/viewer_service.go:195-202` only carries `key/title/status/status_color/tags` |
| `viewer.html` UI rendering | ❌ never reads `entity.size`. All 116 "size" references are file sizes, font sizes, SVG dimensions, or `Set.size` |

## Scope

1. **Backend (small)**: add `Size *int` (`json:"size,omitempty"`) to `FlatEntity` in `internal/services/viewer_service.go` and populate it from the bug / change-card / idea models at the three `&FlatEntity{...}` construction sites (currently around lines 988, 1006, 1027).
2. **Frontend**: render the size value in `viewer.html`. UX decision deferred to design — likely a small badge on hierarchy rows/cards and/or a "Size" row in the entity detail panel. Use the t-shirt label form (XS/S/M/L/XL/XXL) consistent with the CLI's `formatSize` helper in `internal/cli/commands/helpers.go`.

## Out of scope

- **Tech-debt in the web viewer** — tech-debt is not in `HierarchyResponse` at all; no `tech_debt` repo is wired into `ViewerService`. Adding it is a separate, larger feature ("Add tech-debt to the web view") — file separately when prioritized.
- Size-based filtering or sorting in the viewer (this feature is display-only).
- Editing size from the viewer UI (size mutation stays a CLI-only operation for now).

## Trigger / origin

Discovered while reviewing whether the F42 size work was end-to-end visible. CLI surface is complete; viewer surface is the remaining gap.
