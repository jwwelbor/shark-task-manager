package viewer_test

import (
	"strings"
	"testing"
	"unicode/utf8"

	viewer "github.com/jwwelbor/shark-task-manager/internal/viewer"
)

// TestViewerHTMLEmbedded verifies that the go:embed directive populated
// ViewerHTML and that it contains the structural markers required by the SPA.
// TC-SMOKE-01 (full marker set verified once renderSidebar added in Task 3).
func TestViewerHTMLEmbedded(t *testing.T) {
	if len(viewer.ViewerHTML) == 0 {
		t.Fatal("ViewerHTML is empty — go:embed failed")
	}
	content := string(viewer.ViewerHTML)

	// Markers present after Task 1 + Task 2
	required := []string{
		"<!DOCTYPE html>",
		"<script>",
		"STATUS_COLORS",
		"getStatusColor",
		"renderDashboard",
		"renderEntityView",
		"renderPickFolder",
		"api/v1/viewer",
	}
	for _, marker := range required {
		if !strings.Contains(content, marker) {
			t.Errorf("viewer.html missing required marker: %q", marker)
		}
	}
}

// TestViewerHTMLContainsRequiredClasses verifies that key CSS IDs and class
// names used by the JavaScript are present in the HTML.
// TC-SMOKE-02.
func TestViewerHTMLContainsRequiredClasses(t *testing.T) {
	content := string(viewer.ViewerHTML)
	required := []string{
		`id="sidebar"`,
		`id="content"`,
		`id="header"`,
		"pick-folder",    // Pick Folder screen section
		"status-dot",     // Status indicator dots
		"status-badge",   // Status badge pills
		"tree-node",      // Sidebar tree nodes
		"markdown-body",  // Markdown content container
		"history-drawer", // History/transitions panel
	}
	for _, marker := range required {
		if !strings.Contains(content, marker) {
			t.Errorf("viewer.html missing required CSS class or ID: %q", marker)
		}
	}
}

// TestViewerHTMLIsComplete verifies that viewer.html is valid UTF-8, contains
// required closing tags, and meets the minimum size threshold (~30KB).
// TC-SMOKE-03.
func TestViewerHTMLIsComplete(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// Must be valid UTF-8
	if !utf8.ValidString(content) {
		t.Fatal("viewer.html is not valid UTF-8")
	}

	// Must have closing tags — not truncated
	if !strings.Contains(content, "</html>") {
		t.Fatal("viewer.html is missing </html> closing tag — file may be truncated")
	}
	if !strings.Contains(content, "</script>") {
		t.Fatal("viewer.html is missing </script> closing tag — file may be truncated")
	}

	// Minimum size check (should grow to ~2300 LOC ≈ 50KB+ when complete;
	// after Task 2 the file exceeds 30KB).
	if len(content) < 30000 {
		t.Errorf("viewer.html is suspiciously small (%d bytes) — expected at least 30000 bytes", len(content))
	}
}

// TestViewerHTMLAPIClientSection verifies that all 7 API fetch wrapper
// functions are present in the embedded file.
// Covers Task 2 requirement: 7 async fetch wrappers.
func TestViewerHTMLAPIClientSection(t *testing.T) {
	content := string(viewer.ViewerHTML)

	apiFunctions := []string{
		"apiGetWorkflowMeta",
		"apiGetHierarchy",
		"apiGetSummary",
		"apiGetRecentActivity",
		"apiGetFile",
		"apiGetHistory",
		"apiGetFeatureTasks",
	}
	for _, fn := range apiFunctions {
		if !strings.Contains(content, fn) {
			t.Errorf("viewer.html missing API function: %q", fn)
		}
	}

	// Section label must be present
	if !strings.Contains(content, "=== API CLIENT ===") {
		t.Error("viewer.html missing '=== API CLIENT ===' section label")
	}
}

// TestViewerHTMLLoadHelpers verifies that the UI feedback helpers added in
// Task 2 are present in the embedded file.
func TestViewerHTMLLoadHelpers(t *testing.T) {
	content := string(viewer.ViewerHTML)

	helpers := []string{
		"showError",
		"showLoading",
		"hideLoading",
		"loadProjectData",
	}
	for _, fn := range helpers {
		if !strings.Contains(content, fn) {
			t.Errorf("viewer.html missing helper function: %q", fn)
		}
	}
}
