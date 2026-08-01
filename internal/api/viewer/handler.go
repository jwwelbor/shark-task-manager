package viewer

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/api"
	"github.com/jwwelbor/shark-task-manager/internal/keys"
	"github.com/jwwelbor/shark-task-manager/internal/services"
)

const errMsgPathEscapesRoot = "file path escapes project root"

// ViewerHandler handles the read-only dashboard API requests under /api/v1/viewer/.
// It is a thin wrapper: parse/validate → call service → format JSON response.
// All business logic lives in the service layer.
type ViewerHandler struct {
	svc ViewerServicer
}

// NewViewerHandler constructs a ViewerHandler with the given service.
func NewViewerHandler(svc ViewerServicer) *ViewerHandler {
	if svc == nil {
		panic("ViewerHandler: svc is required")
	}
	return &ViewerHandler{svc: svc}
}

// RegisterRoutes mounts all viewer routes on the given mux under prefix.
// All routes are wrapped with the localhost-only CORS middleware.
//
// Routes registered (with prefix = "/api/v1/viewer"):
//
//	GET /api/v1/viewer/summary
//	GET /api/v1/viewer/hierarchy
//	GET /api/v1/viewer/history/{key}
//	GET /api/v1/viewer/file/{key}
//	GET /api/v1/viewer/features/{key}/tasks
//	GET /api/v1/viewer/recent-activity
//	GET /api/v1/viewer/workflow-meta
//	GET /api/v1/viewer/folder-files/{path...}
//	GET /api/v1/viewer/notes/{key}
//	GET /api/v1/viewer/related-docs/{key}
//	GET /api/v1/viewer/tags
//	GET /api/v1/viewer/sprint/overview
//	GET /api/v1/viewer/sprint/plan
//	GET /api/v1/viewer/sprint/report
func (h *ViewerHandler) RegisterRoutes(mux *http.ServeMux, prefix string) {
	// Normalize prefix: trim trailing slash.
	prefix = strings.TrimRight(prefix, "/")

	wrap := WithLocalCORS

	mux.Handle("GET "+prefix+"/summary", wrap(http.HandlerFunc(h.Summary)))
	mux.Handle("GET "+prefix+"/hierarchy", wrap(http.HandlerFunc(h.Hierarchy)))
	mux.Handle("GET "+prefix+"/history/{key}", wrap(http.HandlerFunc(h.History)))
	mux.Handle("GET "+prefix+"/file/{key...}", wrap(http.HandlerFunc(h.File)))
	mux.Handle("GET "+prefix+"/features/{key}/tasks", wrap(http.HandlerFunc(h.FeatureTasks)))
	mux.Handle("GET "+prefix+"/recent-activity", wrap(http.HandlerFunc(h.RecentActivity)))
	mux.Handle("GET "+prefix+"/workflow-meta", wrap(http.HandlerFunc(h.WorkflowMeta)))
	mux.Handle("GET "+prefix+"/folder-files/{path...}", wrap(http.HandlerFunc(h.FolderFiles)))

	mux.Handle("GET "+prefix+"/notes/{key}", wrap(http.HandlerFunc(h.Notes)))
	mux.Handle("GET "+prefix+"/related-docs/{key}", wrap(http.HandlerFunc(h.RelatedDocs)))

	// Tag vocabulary endpoint (REQ-F-001, REQ-F-002): GET only — no mutations.
	mux.Handle("GET "+prefix+"/tags", wrap(http.HandlerFunc(h.Tags)))

	mux.Handle("GET "+prefix+"/sprint/overview", wrap(http.HandlerFunc(h.SprintOverview)))
	mux.Handle("GET "+prefix+"/sprint/plan", wrap(http.HandlerFunc(h.SprintPlan)))
	mux.Handle("GET "+prefix+"/sprint/report", wrap(http.HandlerFunc(h.SprintReport)))

	// Allow OPTIONS preflight for all viewer routes by catching the prefix.
	mux.Handle("OPTIONS "+prefix+"/", wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// WithLocalCORS handles OPTIONS by returning 204 before reaching here.
		// This handler is only reached for non-local origins; respond with 403.
		w.WriteHeader(http.StatusForbidden)
	})))
}

// Summary returns entity-type counts with per-status color/phase metadata.
// GET /api/v1/viewer/summary
func (h *ViewerHandler) Summary(w http.ResponseWriter, r *http.Request) {
	result, err := h.svc.Summary(r.Context())
	if err != nil {
		slog.Error("viewer summary failed", "endpoint", "summary", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to load summary")
		return
	}
	respondJSON(w, http.StatusOK, result)
}

// SprintOverview returns the current sprint's operational bundle for the Overview subview.
// GET /api/v1/viewer/sprint/overview?key=S024 (key is optional; empty means active sprint)
func (h *ViewerHandler) SprintOverview(w http.ResponseWriter, r *http.Request) {
	rawKey := r.URL.Query().Get("key")
	key := strings.ToUpper(strings.TrimSpace(rawKey))
	if key != "" && !keys.IsSprintKey(key) {
		respondError(w, http.StatusBadRequest, "invalid sprint key: "+rawKey)
		return
	}

	result, err := h.svc.SprintOverview(r.Context(), key)
	if err != nil {
		if isNotFound(err) {
			respondError(w, http.StatusNotFound, "sprint not found: "+key)
			return
		}
		slog.Error("viewer sprint overview failed", "endpoint", "sprint_overview", "sprint_key", key, "error", err)
		respondError(w, http.StatusInternalServerError, "failed to load sprint overview")
		return
	}
	respondJSON(w, http.StatusOK, result)
}

// SprintPlan returns the planning bundle for the Plan subview.
// GET /api/v1/viewer/sprint/plan?key=S024 (key is optional; empty means active sprint)
func (h *ViewerHandler) SprintPlan(w http.ResponseWriter, r *http.Request) {
	rawKey := r.URL.Query().Get("key")
	key := strings.ToUpper(strings.TrimSpace(rawKey))
	if key != "" && !keys.IsSprintKey(key) {
		respondError(w, http.StatusBadRequest, "invalid sprint key: "+rawKey)
		return
	}

	result, err := h.svc.SprintPlan(r.Context(), key)
	if err != nil {
		if isNotFound(err) {
			respondError(w, http.StatusNotFound, "sprint not found: "+key)
			return
		}
		slog.Error("viewer sprint plan failed", "endpoint", "sprint_plan", "sprint_key", key, "error", err)
		respondError(w, http.StatusInternalServerError, "failed to load sprint plan")
		return
	}
	respondJSON(w, http.StatusOK, result)
}

// SprintReport returns the reporting bundle for the Report subview.
// GET /api/v1/viewer/sprint/report?key=S024 (key is optional; empty means active sprint)
func (h *ViewerHandler) SprintReport(w http.ResponseWriter, r *http.Request) {
	rawKey := r.URL.Query().Get("key")
	key := strings.ToUpper(strings.TrimSpace(rawKey))
	if key != "" && !keys.IsSprintKey(key) {
		respondError(w, http.StatusBadRequest, "invalid sprint key: "+rawKey)
		return
	}

	result, err := h.svc.SprintReport(r.Context(), key)
	if err != nil {
		if isNotFound(err) {
			respondError(w, http.StatusNotFound, "sprint not found: "+key)
			return
		}
		slog.Error("viewer sprint report failed", "endpoint", "sprint_report", "sprint_key", key, "error", err)
		respondError(w, http.StatusInternalServerError, "failed to load sprint report")
		return
	}
	respondJSON(w, http.StatusOK, result)
}

// Hierarchy returns the full epic → feature tree.
// GET /api/v1/viewer/hierarchy
// Accepts optional repeatable ?tag=<name> query params for tag-based filtering (REQ-F-011)
// and an optional ?include_terminal=true to mirror the CLI `--all` flag (terminal-status
// bugs and change cards are excluded by default; pass true to include them).
func (h *ViewerHandler) Hierarchy(w http.ResponseWriter, r *http.Request) {
	opts := services.HierarchyOptions{
		Tags:            parseTagsQuery(r),
		IncludeTerminal: r.URL.Query().Get("include_terminal") == "true",
	}
	result, err := h.svc.Hierarchy(r.Context(), opts)
	if err != nil {
		var unregErr *services.UnregisteredTagError
		if errors.As(err, &unregErr) {
			respondUnregisteredTagError(w, unregErr)
			return
		}
		slog.Error("viewer hierarchy failed", "endpoint", "hierarchy", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to load hierarchy")
		return
	}
	respondJSON(w, http.StatusOK, result)
}

// History returns the status-change audit trail for any supported entity.
// GET /api/v1/viewer/history/{key}
func (h *ViewerHandler) History(w http.ResponseWriter, r *http.Request) {
	rawKey := r.PathValue("key")
	key, err := validateAndNormalizeAnyKey(rawKey)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid entity key: "+rawKey)
		return
	}

	result, err := h.svc.History(r.Context(), key)
	if err != nil {
		if isNotFound(err) {
			respondError(w, http.StatusNotFound, "entity not found: "+key)
			return
		}
		slog.Error("viewer history failed", "entity", key, "endpoint", "history", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to load history")
		return
	}
	respondJSON(w, http.StatusOK, result)
}

// File returns the raw markdown content of an entity's spec file or any project file by path.
// GET /api/v1/viewer/file/{key...}
//
// Accepts two forms:
//  1. Entity key (e.g. "E07-F01-001") — looks up the entity's stored file_path.
//  2. Relative file path (e.g. "docs/plan/E27/spec.md") — served directly from project root.
func (h *ViewerHandler) File(w http.ResponseWriter, r *http.Request) {
	rawKey := r.PathValue("key")

	// Detect whether rawKey is a file path (contains "/" or has a file extension).
	isFilePath := strings.Contains(rawKey, "/") || strings.Contains(rawKey, ".")

	var result *services.FileResponse
	var err error

	if isFilePath {
		result, err = h.svc.FileByPath(r.Context(), rawKey)
	} else {
		key, keyErr := validateAndNormalizeAnyKey(rawKey)
		if keyErr != nil {
			respondError(w, http.StatusBadRequest, "invalid entity key or file path: "+rawKey)
			return
		}
		result, err = h.svc.File(r.Context(), key)
	}

	if err != nil {
		var secErr *services.SecurityError
		if errors.As(err, &secErr) {
			respondError(w, http.StatusForbidden, errMsgPathEscapesRoot)
			return
		}
		var largeErr *services.FileTooLargeError
		if errors.As(err, &largeErr) {
			respondError(w, http.StatusRequestEntityTooLarge, "file exceeds 2 MiB limit")
			return
		}
		if isNotFound(err) {
			respondError(w, http.StatusNotFound, "file not found: "+rawKey)
			return
		}
		slog.Error("viewer file failed", "key", rawKey, "endpoint", "file", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to load file")
		return
	}
	respondJSON(w, http.StatusOK, result)
}

// FeatureTasks returns the filterable task list for a feature.
// GET /api/v1/viewer/features/{key}/tasks
func (h *ViewerHandler) FeatureTasks(w http.ResponseWriter, r *http.Request) {
	rawKey := r.PathValue("key")
	key := strings.ToUpper(strings.TrimSpace(rawKey))

	// Validate feature key specifically.
	if !keys.IsFeatureKey(key) {
		respondError(w, http.StatusBadRequest, "invalid feature key: "+rawKey)
		return
	}

	q := r.URL.Query()

	// Parse 'blocked' query param: must be "true", "false", or absent.
	var blocked *bool
	if bStr := q.Get("blocked"); bStr != "" {
		switch bStr {
		case "true":
			v := true
			blocked = &v
		case "false":
			v := false
			blocked = &v
		default:
			respondError(w, http.StatusBadRequest, `"blocked" must be "true" or "false"`)
			return
		}
	}

	// Parse pagination params; clamp to valid range.
	limit := parseIntClamp(q.Get("limit"), 200, 1, 500)
	offset := parseIntClamp(q.Get("offset"), 0, 0, maxOffset)

	// Validate non-integer values result in 400.
	if q.Get("limit") != "" {
		if _, err := strconv.Atoi(q.Get("limit")); err != nil {
			respondError(w, http.StatusBadRequest, `"limit" must be an integer`)
			return
		}
	}
	if q.Get("offset") != "" {
		if _, err := strconv.Atoi(q.Get("offset")); err != nil {
			respondError(w, http.StatusBadRequest, `"offset" must be an integer`)
			return
		}
	}

	opts := services.FeatureTaskOptions{
		Status:  q.Get("status"),
		Agent:   q.Get("agent"),
		Blocked: blocked,
		Limit:   limit,
		Offset:  offset,
		Tags:    parseTagsQuery(r), // REQ-F-009: repeatable ?tag= params
	}

	result, err := h.svc.FeatureTasks(r.Context(), key, opts)
	if err != nil {
		var unregErr *services.UnregisteredTagError
		if errors.As(err, &unregErr) {
			respondUnregisteredTagError(w, unregErr)
			return
		}
		if isNotFound(err) {
			respondError(w, http.StatusNotFound, "feature not found: "+key)
			return
		}
		slog.Error("viewer feature tasks failed", "entity", key, "endpoint", "feature_tasks", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to load feature tasks")
		return
	}
	respondJSON(w, http.StatusOK, result)
}

// RecentActivity returns the N most recent status changes across all entity types.
// GET /api/v1/viewer/recent-activity
func (h *ViewerHandler) RecentActivity(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	// Validate 'entity_type' param against allowlist.
	entityType := q.Get("entity_type")
	if entityType != "" {
		validEntityTypes := map[string]bool{
			"epic": true, "feature": true, "task": true,
			"bug": true, "change_card": true, "tech_debt": true,
		}
		if !validEntityTypes[entityType] {
			respondError(w, http.StatusBadRequest, `invalid "entity_type": must be one of epic, feature, task, bug, change_card, tech_debt`)
			return
		}
	}

	// Validate non-integer limit is 400.
	if q.Get("limit") != "" {
		if _, err := strconv.Atoi(q.Get("limit")); err != nil {
			respondError(w, http.StatusBadRequest, `"limit" must be an integer`)
			return
		}
	}

	// Parse 'since' param (RFC3339 or RFC3339Nano).
	var since *time.Time
	if sinceStr := q.Get("since"); sinceStr != "" {
		t, err := time.Parse(time.RFC3339Nano, sinceStr)
		if err != nil {
			t, err = time.Parse(time.RFC3339, sinceStr)
			if err != nil {
				respondError(w, http.StatusBadRequest, `malformed "since": must be RFC3339 format`)
				return
			}
		}
		since = &t
	}

	limit := parseIntClamp(q.Get("limit"), 50, 1, 200)

	opts := services.RecentActivityOptions{
		Limit:      limit,
		EntityType: entityType,
		Since:      since,
	}

	result, err := h.svc.RecentActivity(r.Context(), opts)
	if err != nil {
		slog.Error("viewer recent activity failed", "endpoint", "recent_activity", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to load recent activity")
		return
	}
	respondJSON(w, http.StatusOK, result)
}

// WorkflowMeta returns the full workflow definition for the UI.
// GET /api/v1/viewer/workflow-meta
func (h *ViewerHandler) WorkflowMeta(w http.ResponseWriter, r *http.Request) {
	result, err := h.svc.WorkflowMeta(r.Context())
	if err != nil {
		slog.Error("viewer workflow meta failed", "endpoint", "workflow_meta", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to load workflow metadata")
		return
	}
	respondJSON(w, http.StatusOK, result)
}

// FolderFiles returns a directory listing for a relative path within the project root.
// GET /api/v1/viewer/folder-files/{path...}
func (h *ViewerHandler) FolderFiles(w http.ResponseWriter, r *http.Request) {
	rawPath := r.PathValue("path")
	if rawPath == "" {
		respondError(w, http.StatusBadRequest, "path is required")
		return
	}

	result, err := h.svc.FolderFiles(r.Context(), rawPath)
	if err != nil {
		var secErr *services.SecurityError
		if errors.As(err, &secErr) {
			respondError(w, http.StatusForbidden, "path escapes project root")
			return
		}
		slog.Error("viewer folder files failed", "path", rawPath, "error", err)
		respondError(w, http.StatusInternalServerError, "failed to list folder")
		return
	}
	respondJSON(w, http.StatusOK, result)
}

// Notes returns the notes for any supported entity.
// GET /api/v1/viewer/notes/{key}
func (h *ViewerHandler) Notes(w http.ResponseWriter, r *http.Request) {
	rawKey := r.PathValue("key")
	key, err := validateAndNormalizeAnyKey(rawKey)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid entity key: "+rawKey)
		return
	}

	result, err := h.svc.Notes(r.Context(), key)
	if err != nil {
		if isNotFound(err) {
			respondError(w, http.StatusNotFound, "entity not found: "+key)
			return
		}
		slog.Error("viewer notes failed", "entity", key, "endpoint", "notes", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to load notes")
		return
	}
	respondJSON(w, http.StatusOK, result)
}

// RelatedDocs returns the related documents for any supported entity.
// GET /api/v1/viewer/related-docs/{key}
func (h *ViewerHandler) RelatedDocs(w http.ResponseWriter, r *http.Request) {
	rawKey := r.PathValue("key")
	key, err := validateAndNormalizeAnyKey(rawKey)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid entity key: "+rawKey)
		return
	}

	result, err := h.svc.RelatedDocs(r.Context(), key)
	if err != nil {
		if isNotFound(err) {
			respondError(w, http.StatusNotFound, "entity not found: "+key)
			return
		}
		slog.Error("viewer related-docs failed", "entity", key, "endpoint", "related_docs", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to load related docs")
		return
	}
	respondJSON(w, http.StatusOK, result)
}

// ----- helpers -----

const maxOffset = 1<<31 - 1 // max int32, used as ceiling for offset clamping

// validateAndNormalizeAnyKey normalizes key to uppercase and validates it against
// all known entity key formats. Returns the normalized key or an error if the format
// is unrecognized.
func validateAndNormalizeAnyKey(rawKey string) (string, error) {
	upper := strings.ToUpper(strings.TrimSpace(rawKey))
	if upper == "" {
		return "", errors.New("empty key")
	}
	// Order matters: check short task keys before feature keys (strict prefix overlap).
	switch {
	case keys.IsShortTaskKey(upper), keys.IsTaskKey(upper):
		return upper, nil
	case keys.IsFeatureKey(upper):
		return upper, nil
	case keys.IsEpicKey(upper):
		return upper, nil
	case keys.IsBugKey(upper):
		return upper, nil
	case keys.IsChangeCardKey(upper):
		return upper, nil
	case keys.IsTechDebtKey(upper):
		return upper, nil
	case keys.NewKeyService().DetectEntityType(upper) == keys.EntityTypeQuestion:
		return upper, nil
	}
	return "", errors.New("unrecognized entity key format")
}

// parseIntClamp parses s as an integer. If s is empty or unparseable, defaultVal
// is returned. The result is clamped to [minVal, maxVal].
// Callers that need to distinguish "invalid format" from "out of range" should
// parse separately with strconv.Atoi before calling this.
func parseIntClamp(s string, defaultVal, minVal, maxVal int) int {
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	if v < minVal {
		return minVal
	}
	if v > maxVal {
		return maxVal
	}
	return v
}

// isNotFound reports whether err indicates a missing entity.
// The codebase uses "not found" suffix as the conventional signal.
func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "not found")
}

// respondJSON writes a JSON response. Delegates to the shared api package helper
// via a local wrapper to avoid importing internal/api from a sub-package.
func respondJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			slog.Error("failed to encode viewer JSON response", "error", err)
		}
	}
}

// respondError writes a JSON error response using the shared api.ErrorResponse shape.
func respondError(w http.ResponseWriter, statusCode int, message string) {
	resp := api.ErrorResponse{
		Error:   http.StatusText(statusCode),
		Message: message,
	}
	respondJSON(w, statusCode, resp)
}

// Tags returns the full tag vocabulary for the viewer filter UI.
// GET /api/v1/viewer/tags
// (REQ-F-001, REQ-F-002)
func (h *ViewerHandler) Tags(w http.ResponseWriter, r *http.Request) {
	result, err := h.svc.Tags(r.Context())
	if err != nil {
		slog.Error("viewer tags failed", "endpoint", "tags", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to load tags")
		return
	}
	respondJSON(w, http.StatusOK, result)
}

// parseTagsQuery extracts, trims, and de-empties the repeated ?tag= values.
// Case normalization is intentionally deferred to the service layer (REQ-NF-005).
func parseTagsQuery(r *http.Request) []string {
	raw := r.URL.Query()["tag"]
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		v = strings.TrimSpace(v)
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

// tagErrorResponse is the JSON shape returned on 400 for unregistered tag names.
// Defined locally to keep api.ErrorResponse free of tag-specific fields (spec §2.3.6).
type tagErrorResponse struct {
	Error            string   `json:"error"`
	Message          string   `json:"message"`
	UnregisteredTags []string `json:"unregistered_tags,omitempty"`
}

// respondUnregisteredTagError writes a 400 response with the unregistered_tags array.
// (REQ-F-011, REQ-F-012, spec §2.3.6)
func respondUnregisteredTagError(w http.ResponseWriter, err *services.UnregisteredTagError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	resp := tagErrorResponse{
		Error:            "Bad Request",
		Message:          err.Error(),
		UnregisteredTags: []string{err.Name},
	}
	if encErr := json.NewEncoder(w).Encode(resp); encErr != nil {
		slog.Error("failed to encode unregistered tag error response", "error", encErr)
	}
}
