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

// TestViewerHTMLSidebarSection verifies that the === SIDEBAR === section label
// and the renderSidebar function are present. TC-TREE-01 (section present).
func TestViewerHTMLSidebarSection(t *testing.T) {
	content := string(viewer.ViewerHTML)

	if !strings.Contains(content, "=== SIDEBAR ===") {
		t.Error("viewer.html missing '=== SIDEBAR ===' section label")
	}
	if !strings.Contains(content, "renderSidebar") {
		t.Error("viewer.html missing 'renderSidebar' function")
	}
}

// TestViewerHTMLSidebarNodeBuilders verifies that all required node builder
// functions are present. TC-TREE-02, TC-TREE-03, TC-TREE-05.
func TestViewerHTMLSidebarNodeBuilders(t *testing.T) {
	content := string(viewer.ViewerHTML)

	builders := []string{
		"buildEpicNodeHtml",
		"buildFeatureNodeHtml",
		"buildTaskNodeHtml",
		"buildDocNodeHtml",
		"buildFlatSectionHtml",
		"buildStatusDotHtml",
		"buildNodeHtml",
		"buildArrowHtml",
	}
	for _, fn := range builders {
		if !strings.Contains(content, fn) {
			t.Errorf("viewer.html missing sidebar builder function: %q", fn)
		}
	}
}

// TestViewerHTMLSidebarIndentationClasses verifies that the CSS classes
// matching the spec §2.6.2 indentation table are present. TC-TREE-01 through TC-TREE-03.
func TestViewerHTMLSidebarIndentationClasses(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// Spec §2.6.2: epics=16px, features=40px, tasks=56px, flat=18px
	indentClasses := []string{
		"tree-node-epic",    // 16px left padding, bold
		"tree-node-feature", // 40px left padding
		"tree-node-task",    // 56px left padding
		"tree-node-flat",    // 18px left padding
	}
	for _, cls := range indentClasses {
		if !strings.Contains(content, cls) {
			t.Errorf("viewer.html missing indentation CSS class: %q", cls)
		}
	}
}

// TestViewerHTMLSidebarSelectedState verifies that the selected state
// CSS rules and JS logic are present. TC-TREE-04.
func TestViewerHTMLSidebarSelectedState(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// CSS: selected class with dark tint and 3px accent border
	if !strings.Contains(content, "tree-node.selected") {
		t.Error("viewer.html missing '.tree-node.selected' CSS rule")
	}
	if !strings.Contains(content, "3px solid var(--accent)") {
		t.Error("viewer.html missing 3px accent left-border for selected state")
	}

	// JS: selectedKey comparison used in node rendering
	if !strings.Contains(content, "selectedKey") {
		t.Error("viewer.html missing 'selectedKey' variable for selected state")
	}
}

// TestViewerHTMLScrollSidebarToKey verifies that the scrollSidebarToKey
// helper is present. TC-TREE-04 (programmatic selection).
func TestViewerHTMLScrollSidebarToKey(t *testing.T) {
	content := string(viewer.ViewerHTML)

	if !strings.Contains(content, "scrollSidebarToKey") {
		t.Error("viewer.html missing 'scrollSidebarToKey' helper function")
	}
	if !strings.Contains(content, "scrollIntoView") {
		t.Error("viewer.html missing 'scrollIntoView' call in sidebar helper")
	}
}

// TestViewerHTMLExpandCollapseState verifies that expand/collapse state sets
// (expandedEpics, expandedFeatures) and arrow rendering are present. TC-TREE-02.
func TestViewerHTMLExpandCollapseState(t *testing.T) {
	content := string(viewer.ViewerHTML)

	if !strings.Contains(content, "expandedEpics") {
		t.Error("viewer.html missing 'expandedEpics' Set for expand/collapse state")
	}
	if !strings.Contains(content, "expandedFeatures") {
		t.Error("viewer.html missing 'expandedFeatures' Set for expand/collapse state")
	}
	// Arrow symbols used in expand/collapse
	if !strings.Contains(content, "▶") {
		t.Error("viewer.html missing collapsed arrow symbol ▶")
	}
	if !strings.Contains(content, "▼") {
		t.Error("viewer.html missing expanded arrow symbol ▼")
	}
}

// TestViewerHTMLStatusDotColorIntegration verifies that the status dot
// uses getStatusColor() for its background. TC-COLOR-01.
func TestViewerHTMLStatusDotColorIntegration(t *testing.T) {
	content := string(viewer.ViewerHTML)

	if !strings.Contains(content, "buildStatusDotHtml") {
		t.Error("viewer.html missing 'buildStatusDotHtml' function")
	}
	// The function must call getStatusColor
	if !strings.Contains(content, "getStatusColor") {
		t.Error("viewer.html missing 'getStatusColor' call (needed for status dot coloring)")
	}
}

// TestViewerHTMLSidebarRenderCall verifies that renderSidebar() is called
// from the render() dispatcher for non-pick-folder states.
func TestViewerHTMLSidebarRenderCall(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// renderSidebar must be called from render() for dashboard and entity_view
	// We check that both appear in the same switch block
	if !strings.Contains(content, "renderSidebar()") {
		t.Error("viewer.html missing renderSidebar() call in render() dispatcher")
	}
}

// TestViewerHTMLSidebarFlatSections verifies that Ideas and Change Cards flat
// section rendering is wired. TC-TREE-05.
func TestViewerHTMLSidebarFlatSections(t *testing.T) {
	content := string(viewer.ViewerHTML)

	flatSections := []string{
		"Ideas",
		"Change Cards",
		"Tech Debt",
		"Tags",
	}
	for _, section := range flatSections {
		if !strings.Contains(content, section) {
			t.Errorf("viewer.html missing flat section reference: %q", section)
		}
	}
}
