---
feature_key: E27-F07
doc_type: test-plan
status: draft
spec: ./spec.md
epic_uat_plan: ../uat-plan.md
---

# E27-F07 Test Plan: Inline Markdown Editor

## Overview

This plan covers all automated tests required before the feature ships. Every acceptance criterion in `spec.md §1.3` has at least one named test case. Tests are assigned to the two new test files called out in the spec (`edit_service_test.go`, `edit_handler_test.go`) plus the existing CORS test file that must be updated.

Tests are **not** end-to-end browser tests — those belong to the epic UAT plan (`../uat-plan.md`). This plan covers Go unit and HTTP-handler tests only.

---

## 1. AC Test Matrix

### AC-01 — Edit button appears; clicking shows textarea with raw content and Save/Cancel buttons

This acceptance criterion is verified by the browser/SPA layer and belongs entirely to the epic UAT (scenario D1 extended). No Go unit test covers client-side DOM state. Covered at system level by UAT Area D.

**Note:** The backend prerequisite is that `GET /api/v1/viewer/file/{key}` and `GET /api/v1/viewer/file-by-path` return a `path` field in the response (already implemented in ViewerService). No new backend test is needed for AC-01 alone.

---

### AC-02 — Clicking Save writes content to disk; viewer returns to rendered markdown

**Traces to:** REQ-F-001, REQ-F-002, REQ-F-003, REQ-F-006.

#### TC-F07-001: EditService.WriteFile — happy path

| Field | Value |
|---|---|
| File | `internal/services/edit_service_test.go` |
| Test function | `TestEditService_WriteFile_HappyPath` |
| Setup | `t.TempDir()` as `projectRoot`; create a real file `docs/spec.md` inside it with known content |
| Input | `relPath = "docs/spec.md"`, `content = "# Updated\n"` |
| Expected | No error; file on disk contains `"# Updated\n"` exactly; result `BytesWritten` equals `len("# Updated\n")`; result `Path` equals the relative path |
| Edge case | Content is empty string (zero-byte write succeeds) |

#### TC-F07-002: EditHandler.PutFile — success returns 200 with JSON body

| Field | Value |
|---|---|
| File | `internal/api/viewer/edit_handler_test.go` |
| Test function | `TestEditHandler_PutFile_Success` |
| Setup | `MockEditServicer` with `WriteFileFunc` returning `&WriteFileResult{Path:"docs/spec.md", BytesWritten:12}` |
| Input | `PUT /api/v1/edit/file` body `{"path":"docs/spec.md","content":"hello world\n"}` |
| Expected | HTTP 200; response body JSON has `path` and `bytes_written` fields; no error field |

---

### AC-03 — Cancel restores previous rendered view; no filesystem write

This is pure SPA state logic — no server-side write is issued when Cancel is clicked. Verified at UAT level (browser interaction). Backend guarantee: the `PUT /api/v1/edit/file` endpoint is never called by Cancel. No Go test.

---

### AC-04 — Read-only file: Save shows inline error; textarea stays open

**Traces to:** REQ-F-005.

#### TC-F07-003: EditService.WriteFile — permission denied returns error

| Field | Value |
|---|---|
| File | `internal/services/edit_service_test.go` |
| Test function | `TestEditService_WriteFile_PermissionDenied` |
| Setup | `t.TempDir()`; create file with `os.Chmod(path, 0o444)`; also make parent dir read-only if write is attempted there |
| Input | `relPath` pointing to the read-only file |
| Expected | Error returned; error is NOT a `SecurityError`; error message contains filesystem-level detail |
| Skip condition | `os.Getuid() == 0` (root ignores permissions; skip with `t.Skip`) |

#### TC-F07-004: EditHandler.PutFile — service write error maps to 500

| Field | Value |
|---|---|
| File | `internal/api/viewer/edit_handler_test.go` |
| Test function | `TestEditHandler_PutFile_WriteError` |
| Setup | `MockEditServicer` returning `fmt.Errorf("write failed: permission denied")` |
| Input | Valid JSON body |
| Expected | HTTP 500; response body has `error` field; `bytes_written` absent |

---

### AC-05 — Path containing `../` resolving outside project root returns 400; no file written

**Traces to:** REQ-NF-001.

#### TC-F07-005: EditService.WriteFile — absolute path rejected immediately

| Field | Value |
|---|---|
| File | `internal/services/edit_service_test.go` |
| Test function | `TestEditService_WriteFile_AbsolutePath` |
| Setup | Any `t.TempDir()` as root |
| Input | `relPath = "/etc/passwd"` |
| Expected | `SecurityError` returned; no file written |

#### TC-F07-006: EditService.WriteFile — `../` traversal outside root rejected

| Field | Value |
|---|---|
| File | `internal/services/edit_service_test.go` |
| Test function | `TestEditService_WriteFile_TraversalOutsideRoot` |
| Setup | `t.TempDir()` as root; target is `../../outside.md` which resolves above root |
| Input | `relPath = "../../outside.md"` |
| Expected | `SecurityError` returned; no file written in `outsideDir` |

#### TC-F07-007: EditService.WriteFile — symlink resolving outside root rejected

| Field | Value |
|---|---|
| File | `internal/services/edit_service_test.go` |
| Test function | `TestEditService_WriteFile_SymlinkEscape` |
| Setup | `t.TempDir()` as root; `outsideDir := t.TempDir()`; create symlink `root/link -> outsideDir/target.md` |
| Input | `relPath = "link"` |
| Expected | `SecurityError` returned; no file written to `outsideDir` |

#### TC-F07-008: EditHandler.PutFile — SecurityError maps to 400

| Field | Value |
|---|---|
| File | `internal/api/viewer/edit_handler_test.go` |
| Test function | `TestEditHandler_PutFile_PathTraversal` |
| Setup | `MockEditServicer` returning a `SecurityError` |
| Input | `{"path": "../../etc/passwd", "content": "evil"}` |
| Expected | HTTP 400; response body `error` = `"Bad Request"` |

---

### AC-06 — Request body exceeding 2 MiB returns 413

**Traces to:** REQ-NF-004.

#### TC-F07-009: EditHandler.PutFile — body > 2 MiB returns 413

| Field | Value |
|---|---|
| File | `internal/api/viewer/edit_handler_test.go` |
| Test function | `TestEditHandler_PutFile_BodyTooLarge` |
| Setup | Build a request body of `2*1024*1024 + 1` bytes |
| Input | JSON body that exceeds 2 MiB |
| Expected | HTTP 413; response body contains error message; `MockEditServicer.WriteFileFunc` is NOT called |
| Notes | `http.MaxBytesReader` is used in the handler before `json.Decode`; the mock should remain un-called to verify size limit fires before service call |

---

### AC-07 — Edit works for both entity files and standalone doc files

**Traces to:** REQ-F-007.

The backend endpoint is path-based (`{"path": "..."}`) and makes no distinction between entity files and standalone doc files — both are validated against `projectRoot` with identical logic. This is verified by TC-F07-001 (any relative path works) and by the security tests (TC-F07-005 through TC-F07-008).

No additional test case is needed beyond confirming that:

#### TC-F07-010: EditService.WriteFile — subdirectory path (standalone doc) succeeds

| Field | Value |
|---|---|
| File | `internal/services/edit_service_test.go` |
| Test function | `TestEditService_WriteFile_SubdirectoryPath` |
| Setup | `t.TempDir()` as root; create `docs/guide/intro.md` inside it |
| Input | `relPath = "docs/guide/intro.md"`, `content = "# Guide\n"` |
| Expected | No error; file content updated; `Path` in result equals input `relPath` |

---

### AC-08 — CORS middleware allows PUT requests from localhost origins

**Traces to:** REQ-NF-002.

#### TC-F07-011: WithLocalCORS — PUT from localhost origin includes PUT in Allow-Methods

| Field | Value |
|---|---|
| File | `internal/api/viewer/cors_test.go` (update existing) |
| Test function | `TestWithLocalCORS_PUTLocalhost` |
| Setup | Wrap `innerHandler` with `WithLocalCORS`; send `PUT /` with `Origin: http://localhost:3000` |
| Expected | Inner handler called; `Access-Control-Allow-Methods` header contains `"PUT"` |

#### TC-F07-012: WithLocalCORS — OPTIONS preflight for PUT from localhost returns 204 with PUT in Allow-Methods

| Field | Value |
|---|---|
| File | `internal/api/viewer/cors_test.go` (update existing) |
| Test function | `TestWithLocalCORS_PreflightPUT` |
| Setup | `OPTIONS /` with `Origin: http://localhost:3000`, `Access-Control-Request-Method: PUT` |
| Expected | HTTP 204; `Access-Control-Allow-Methods` contains `"PUT"`; inner handler NOT called |

**Note:** These two tests will FAIL until `cors.go` is updated to include `"PUT"` in `Access-Control-Allow-Methods`. They are intentionally written first (red) per the TDD requirement in T-E27-F07-005.

---

## 2. Additional Edge Cases

The following edge cases arise from the spec but are not explicitly called out in the 8 ACs. Each is attached to the most relevant AC.

### 2a. Missing required fields in PUT body (attached to AC-02)

#### TC-F07-013: EditHandler.PutFile — missing `path` field returns 400

| Field | Value |
|---|---|
| File | `internal/api/viewer/edit_handler_test.go` |
| Test function | `TestEditHandler_PutFile_MissingPath` |
| Input | `{"content": "hello"}` (no `path`) |
| Expected | HTTP 400; `MockEditServicer.WriteFileFunc` not called |

#### TC-F07-014: EditHandler.PutFile — missing `content` field returns 400

| Field | Value |
|---|---|
| File | `internal/api/viewer/edit_handler_test.go` |
| Test function | `TestEditHandler_PutFile_MissingContent` |
| Input | `{"path": "docs/spec.md"}` (no `content`) |
| Expected | HTTP 400; `MockEditServicer.WriteFileFunc` not called |

#### TC-F07-015: EditHandler.PutFile — malformed JSON body returns 400

| Field | Value |
|---|---|
| File | `internal/api/viewer/edit_handler_test.go` |
| Test function | `TestEditHandler_PutFile_MalformedJSON` |
| Input | `{bad json` |
| Expected | HTTP 400 |

### 2b. Atomic write — `.tmp` file cleaned up on failure (attached to AC-02)

#### TC-F07-016: EditService.WriteFile — atomic: `.tmp` file does not persist on error

| Field | Value |
|---|---|
| File | `internal/services/edit_service_test.go` |
| Test function | `TestEditService_WriteFile_AtomicCleanup` |
| Setup | Force a rename failure by making the target directory read-only after the `.tmp` write succeeds (platform-specific). If the setup is not achievable portably, test instead that on a successful write no `.tmp` file remains. |
| Expected | After the call (success or failure), no `.tmp` file exists in the directory |

### 2c. Concurrent write does not corrupt final file (attached to AC-02)

#### TC-F07-017: EditService.WriteFile — concurrent writes serialize without corruption

| Field | Value |
|---|---|
| File | `internal/services/edit_service_test.go` |
| Test function | `TestEditService_WriteFile_ConcurrentWrites` |
| Setup | 10 goroutines each calling `WriteFile` with distinct content strings on the same file path |
| Expected | No error from any goroutine; final file content equals exactly one of the content strings (not a partial interleave) |
| Notes | `os.Rename` is atomic on POSIX, so the last rename wins — no corruption expected. This test confirms the atomicity guarantee in practice. |

---

## 3. Integration Scenarios

These verify cross-component contracts. They are NOT full end-to-end tests — each uses the real `EditService` or real HTTP handler wired to a real filesystem, but no database.

### 3a. Handler + Service integration (real EditService, real filesystem)

#### TC-F07-018: HTTP PUT through handler to real EditService writes file

| File | `internal/api/viewer/edit_handler_test.go` or a new `edit_integration_test.go` (same package) |
|---|---|
| Test function | `TestEditHandler_Integration_RealService` |
| Setup | `t.TempDir()` as root; create `docs/notes.md`; construct real `NewEditService(dir)` and `NewEditHandler(svc)` |
| Input | `PUT /api/v1/edit/file` body `{"path":"docs/notes.md","content":"new content"}` |
| Expected | HTTP 200; `os.ReadFile` confirms on-disk content equals `"new content"` |
| Notes | This is the only test that uses the real `EditService` from the handler layer. It validates the handler→service interface in one shot. |

### 3b. CORS middleware wrapping the edit route

#### TC-F07-019: OPTIONS preflight for `/api/v1/edit/file` returns 204 with correct headers

This is covered by TC-F07-012 at the middleware level. The wiring test (confirming the route is correctly wrapped in `server.go`) belongs to the server integration tests in T-E27-F07-003's task scope.

### 3c. Epic UAT scenarios this feature contributes to

| UAT Scenario | How F07 contributes |
|---|---|
| H6 — No write endpoints under `/api/v1/viewer/` | The edit route lives under `/api/v1/edit/`, NOT `/api/v1/viewer/`. TC-F07-002 confirms the correct path. Any PUT to `/api/v1/viewer/*` must still 405 (unchanged from pre-F07 viewer handler). |
| H4 — File endpoint path-traversal probe | TC-F07-005 through TC-F07-008 extend the same guarantee to the write endpoint. |
| H2/H3 — CORS rejects/accepts local origins | TC-F07-011 and TC-F07-012 verify PUT is allowed from localhost. TC-F07-019 covers the preflight. |
| D1 — Spec markdown renders after edit | UAT-only: the browser re-renders from textarea content on success (no extra fetch). No Go test. |

---

## 4. Test Infrastructure

### 4a. Existing patterns to follow

| Pattern | Location | Usage for F07 |
|---|---|---|
| `t.TempDir()` for filesystem isolation | `internal/services/viewer_service_test.go` (lines 148, 173, 193, …) | All `EditService` tests: create real files in temp dirs, no cleanup needed |
| Mock with function fields (`SummaryFunc`, `FileFunc`, etc.) | `internal/api/viewer/handler_test.go` (lines 17–90) | `MockEditServicer` follows the same struct+func-field pattern |
| `httptest.NewRequest` + `httptest.NewRecorder` | `internal/api/viewer/handler_test.go` and `cors_test.go` | All `EditHandler` tests |
| Error type assertion with `errors.As` | `internal/api/viewer/handler_test.go` | Handler test verifying `SecurityError` → 400 mapping |
| `t.Skip` for platform-specific tests | Standard Go | TC-F07-003 (root UID), TC-F07-016 (rename failure setup) |

### 4b. New mock needed

```go
// MockEditServicer — mirrors the established MockViewerServicer pattern
// Location: internal/api/viewer/edit_handler_test.go (or a shared mock file)
type MockEditServicer struct {
    WriteFileFunc func(ctx context.Context, path string, content string) (*services.WriteFileResult, error)
}

func (m *MockEditServicer) WriteFile(ctx context.Context, path string, content string) (*services.WriteFileResult, error) {
    if m.WriteFileFunc != nil {
        return m.WriteFileFunc(ctx, path, content)
    }
    return nil, errors.New("WriteFileFunc not set in mock")
}
```

### 4c. No new test utilities needed

`EditService` has no repository dependencies — it operates purely on the filesystem. `t.TempDir()` is sufficient isolation. No mock repositories, no `test.GetTestDB()`.

### 4d. Test file locations

| Test file | What it tests | Notes |
|---|---|---|
| `internal/services/edit_service_test.go` | `EditService.WriteFile` — all filesystem scenarios (TC-F07-001, 003, 005–007, 010, 016–017) | Uses real filesystem via `t.TempDir()`. No mocks. |
| `internal/api/viewer/edit_handler_test.go` | `EditHandler.PutFile` — HTTP layer (TC-F07-002, 004, 008–009, 013–015, 018) | Uses `MockEditServicer`. Same package as `cors_test.go`. |
| `internal/api/viewer/cors_test.go` | Updated CORS tests (TC-F07-011, 012) | Adds two new test functions to the existing file. |

---

## 5. Test Case Summary

| Test Case ID | AC | Test Function | File |
|---|---|---|---|
| TC-F07-001 | AC-02 | `TestEditService_WriteFile_HappyPath` | `edit_service_test.go` |
| TC-F07-002 | AC-02 | `TestEditHandler_PutFile_Success` | `edit_handler_test.go` |
| TC-F07-003 | AC-04 | `TestEditService_WriteFile_PermissionDenied` | `edit_service_test.go` |
| TC-F07-004 | AC-04 | `TestEditHandler_PutFile_WriteError` | `edit_handler_test.go` |
| TC-F07-005 | AC-05 | `TestEditService_WriteFile_AbsolutePath` | `edit_service_test.go` |
| TC-F07-006 | AC-05 | `TestEditService_WriteFile_TraversalOutsideRoot` | `edit_service_test.go` |
| TC-F07-007 | AC-05 | `TestEditService_WriteFile_SymlinkEscape` | `edit_service_test.go` |
| TC-F07-008 | AC-05 | `TestEditHandler_PutFile_PathTraversal` | `edit_handler_test.go` |
| TC-F07-009 | AC-06 | `TestEditHandler_PutFile_BodyTooLarge` | `edit_handler_test.go` |
| TC-F07-010 | AC-07 | `TestEditService_WriteFile_SubdirectoryPath` | `edit_service_test.go` |
| TC-F07-011 | AC-08 | `TestWithLocalCORS_PUTLocalhost` | `cors_test.go` |
| TC-F07-012 | AC-08 | `TestWithLocalCORS_PreflightPUT` | `cors_test.go` |
| TC-F07-013 | AC-02 | `TestEditHandler_PutFile_MissingPath` | `edit_handler_test.go` |
| TC-F07-014 | AC-02 | `TestEditHandler_PutFile_MissingContent` | `edit_handler_test.go` |
| TC-F07-015 | AC-02 | `TestEditHandler_PutFile_MalformedJSON` | `edit_handler_test.go` |
| TC-F07-016 | AC-02 | `TestEditService_WriteFile_AtomicCleanup` | `edit_service_test.go` |
| TC-F07-017 | AC-02 | `TestEditService_WriteFile_ConcurrentWrites` | `edit_service_test.go` |
| TC-F07-018 | AC-02 + AC-07 | `TestEditHandler_Integration_RealService` | `edit_handler_test.go` |
| TC-F07-019 | AC-08 | Covered by TC-F07-012 | `cors_test.go` |

**ACs covered:** AC-01 (UAT only), AC-02, AC-03 (UAT only), AC-04, AC-05, AC-06, AC-07, AC-08.
All 8 ACs are addressed. ACs 01 and 03 are browser-only and are explicitly delegated to epic UAT scenarios D1 and the Cancel button interaction.

---

## 6. Quality Gates

Before marking this feature `ready_for_task_generation`:

- [ ] Every TC listed in Section 5 has a corresponding test function stubbed or implemented.
- [ ] TC-F07-011 and TC-F07-012 (CORS PUT tests) are written first and confirmed RED before `cors.go` is changed.
- [ ] `make test` passes with zero failures.
- [ ] `make fmt && make lint` pass.
- [ ] No database is touched by any test in this feature (EditService is filesystem-only).
- [ ] `t.TempDir()` used for all filesystem isolation — no hard-coded temp paths.
