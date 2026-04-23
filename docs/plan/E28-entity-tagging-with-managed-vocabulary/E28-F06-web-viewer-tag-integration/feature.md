---
feature_key: E28-F06-web-viewer-tag-integration
epic_key: E28
title: Web Viewer Tag Integration
description: Surface tags in the E27 web viewer (read-only), add `tags[]` to every entity response DTO, accept `?tag=` query params on list endpoints (AND semantics), expose `GET /api/v1/viewer/tags` for the vocabulary, and render tag chips + a tag filter control in the UI. Vocabulary management intentionally stays CLI-only.
order: 6
---

# Web Viewer Tag Integration

**Feature Key**: E28-F06-web-viewer-tag-integration

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)
- **Epic Architecture**: [Architecture](../../architecture.md)

---

## Thin Description

Deliver the viewer parity for tags without exposing any mutation paths. On the API side (`internal/api/viewer`, `internal/services/viewer_service.go`): every entity DTO returned by `Summary`, `Hierarchy`, and `FeatureTasks` handlers gains a `tags []string` field that is always present (`[]` when empty, never `null`) so clients skip null checks; list handlers accept repeated `?tag=<name>` query parameters and intersect results using the AND semantics from F05's `EntityIDsByTags`; a new `GET /api/v1/viewer/tags` endpoint returns the full vocabulary as `{"tags": [{"name": "voice"}, ...]}`. On the UI side (`internal/viewer`): entity detail views render tag chips in the header area; list views gain a multi-select tag filter control that loads the vocabulary from the new endpoint and appends selected tags as `?tag=` params on list fetches. No create/rename/delete controls are built — those remain CLI-only in v1 per PRD scope. `WireServices` in `cmd/server/services.go` injects `TagService` into the viewer handler; the `MaintainerGate` is intentionally not injected.

**Integration points:** `internal/api/viewer/handler.go`, `internal/services/viewer_service.go`, viewer templates/assets in `internal/viewer/`, `cmd/server/services.go` wiring.

**Architecture refs:** §4.2 (viewer service wiring), §4.5 (API and UI integration, read-only invariant), §5.5 (backward compatibility).

**Execution order:** 6 — depends on F05 (needs `EntityIDsByTags` + per-entity `Tags` population). Parallel with F07.

---

*Last Updated*: 2026-04-22
