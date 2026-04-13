---
feature_key: E27-F07-inline-markdown-editor-web-viewer-file-editing
epic_key: E27
title: Inline Markdown Editor - Web Viewer File Editing
description: Allow developers to edit any .md file displayed in the shark web viewer directly in the browser using a plain textarea, saving back to the local filesystem.
---

# Inline Markdown Editor - Web Viewer File Editing

**Feature Key**: E27-F07-inline-markdown-editor-web-viewer-file-editing

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)

---

## Goal

### Problem

When reviewing task specs or feature plans in the `shark web` viewer, developers must switch to their editor, find the file, make changes, save, and return to the browser to see updates. This context switch is friction — the viewer is read-only even though the server already knows exactly where every file lives.

### Solution

Add an Edit button to the file viewer panel. Clicking it replaces the rendered markdown with a plain textarea pre-filled with the raw content. The developer edits inline and clicks Save, which writes the file back to disk via a new `PUT /api/v1/edit/file/{key}` endpoint. Cancel discards changes and returns to the rendered view.

### Impact

- Spec files can be updated without leaving the browser
- No new dependencies (no editor libraries)
- Minimal surface area: one new service, one new handler, ~50 lines of JS

---

## User Stories

### Must-Have Stories

**Story 1**: As a developer using `shark web`, I want to click an Edit button on any displayed spec file so that I can update it without switching to my editor.

**Acceptance Criteria**:
- [ ] Edit button is visible when a file is displayed in the viewer
- [ ] Clicking Edit shows the raw markdown in a textarea
- [ ] Clicking Save writes the content to the filesystem and returns to rendered view
- [ ] Clicking Cancel discards changes and returns to rendered view without writing

**Story 2**: As a developer, when a save fails (e.g., file is read-only), I want to see an error message so I know the file was not updated.

**Acceptance Criteria**:
- [ ] Failed saves show an error message inline (no silent failures)
- [ ] Textarea remains open with my edits so I don't lose them

---

## Requirements

### Functional Requirements

1. **REQ-F-001**: Edit button in file viewer panel
   - **Description**: A button labeled "Edit" appears when a file is loaded in the viewer panel
   - **Priority**: Must-Have

2. **REQ-F-002**: Textarea edit mode
   - **Description**: Edit mode replaces rendered HTML with a full-width `<textarea>` containing the raw markdown content
   - **Priority**: Must-Have

3. **REQ-F-003**: Save writes to filesystem
   - **Description**: Save sends `PUT /api/v1/edit/file/{key}` with the textarea content; server writes the file atomically
   - **Priority**: Must-Have

4. **REQ-F-004**: Cancel discards changes
   - **Description**: Cancel returns to rendered view without any write; no confirmation dialog needed
   - **Priority**: Must-Have

5. **REQ-F-005**: Error feedback on save failure
   - **Description**: If the PUT returns an error, display the error message; textarea stays open
   - **Priority**: Must-Have

### Non-Functional Requirements

**Security**

1. **REQ-NF-001**: Path traversal prevention
   - **Description**: `EditService.WriteFile()` reuses the same path resolution and traversal checks as `ViewerService.File()`
   - **Risk Mitigation**: Prevents writes outside the project root

2. **REQ-NF-002**: Localhost-only access
   - **Description**: Edit route inherits the server's existing `127.0.0.1` bind and `WithLocalCORS` middleware
   - **Risk Mitigation**: Only the local developer can trigger writes

3. **REQ-NF-003**: Atomic writes
   - **Description**: `WriteFile` writes to a `.tmp` file then `os.Rename` — prevents partial content on crash
   - **Risk Mitigation**: No corrupted files if process dies mid-write

---

## Architecture

### New Components

**`internal/services/edit_service.go`**
```
EditService
  - projectRoot string
  - WriteFile(key string, content string) error
    - Resolve file path via entity key (reuse ViewerService path logic, or accept resolved path)
    - Path traversal check: resolved path must be under projectRoot
    - Atomic write: os.WriteFile to .tmp, then os.Rename
```

**`internal/api/edit/handler.go`**
```
EditHandler
  - editSvc *services.EditService
  - PutFile(w, r)  →  PUT /api/v1/edit/file/{key...}
    - Parse key from URL
    - Read body (content string, JSON or plain text)
    - Call editSvc.WriteFile(key, content)
    - Return 200 OK or error
```

**Server wiring** (`internal/viewer/server/wire.go`)
- Construct `EditService` with `projectRoot`
- Construct `EditHandler`
- Register `PUT /api/v1/edit/file/{key...}` route

**Frontend** (`internal/viewer/assets/viewer.html`)
- Add "Edit" button to file panel toolbar (shown when file is loaded)
- Edit state: hide rendered markdown, show `<textarea>`, show Save/Cancel buttons
- Save: `fetch('PUT', /api/v1/edit/file/'+currentKey, {body: textarea.value})`
- On success: re-render markdown from textarea value, return to view state
- On error: show error banner, keep textarea open

### What is NOT changing
- `ViewerService` stays read-only (ADR-E27-003 preserved)
- No changes to existing viewer API routes
- No new JS libraries

---

## Out of Scope

1. **Rich editor / syntax highlighting** — plain textarea only; polish can come later
2. **Create new files** — edit only, no file creation
3. **Delete files** — out of scope
4. **Rename / move files** — out of scope
5. **Undo history beyond Cancel** — git is the undo mechanism

---

## Acceptance Criteria (Feature-Level)

**Scenario 1: Happy path edit**
- **Given** a file is displayed in the viewer
- **When** the developer clicks Edit, modifies the textarea, and clicks Save
- **Then** the file on disk is updated with the new content
- **And** the viewer returns to rendered markdown view showing the new content

**Scenario 2: Cancel discards changes**
- **Given** Edit mode is open with modified content
- **When** the developer clicks Cancel
- **Then** no write occurs and the original rendered view is restored

**Scenario 3: Save failure**
- **Given** the file on disk is not writable
- **When** the developer clicks Save
- **Then** an error message is displayed inline
- **And** the textarea remains open with the developer's edits intact

**Scenario 4: Path traversal attempt**
- **Given** a malformed key containing `../`
- **When** a PUT is sent to the edit endpoint
- **Then** the server returns 400 Bad Request and no file is written

---

*Last Updated*: 2026-04-12
