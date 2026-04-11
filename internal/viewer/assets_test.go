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

// TestViewerHTMLDashboardSection verifies that the === DASHBOARD === section
// label and the renderDashboard orchestrator are present. TC-VIS-04.
func TestViewerHTMLDashboardSection(t *testing.T) {
	content := string(viewer.ViewerHTML)

	if !strings.Contains(content, "=== DASHBOARD ===") {
		t.Error("viewer.html missing '=== DASHBOARD ===' section label")
	}
	if !strings.Contains(content, "renderDashboard") {
		t.Error("viewer.html missing 'renderDashboard' orchestrator function")
	}
}

// TestViewerHTMLStatusBreakdownSection verifies that renderStatusBreakdown()
// is present and uses summaryData to build entity type cards. TC-VIS-04.
func TestViewerHTMLStatusBreakdownSection(t *testing.T) {
	content := string(viewer.ViewerHTML)

	if !strings.Contains(content, "renderStatusBreakdown") {
		t.Error("viewer.html missing 'renderStatusBreakdown' function")
	}
	// Data source: summaryData optional-chaining per spec §3
	if !strings.Contains(content, "summaryData") {
		t.Error("viewer.html missing 'summaryData' reference in status breakdown")
	}
	// Status Breakdown section title
	if !strings.Contains(content, "Status Breakdown") {
		t.Error("viewer.html missing 'Status Breakdown' section heading")
	}
	// Each card must display an uppercase entity label
	if !strings.Contains(content, "EPICS") {
		t.Error("viewer.html missing uppercase 'EPICS' label in Status Breakdown card")
	}
}

// TestViewerHTMLStatusBreakdownUsesGetStatusColor verifies that the badge
// colors in the Status Breakdown section are derived from getStatusColor().
// TC-COLOR-01, TC-VIS-05.
func TestViewerHTMLStatusBreakdownUsesGetStatusColor(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// renderStatusBreakdown must call getStatusColor for badge background
	if !strings.Contains(content, "getStatusColor") {
		t.Error("viewer.html missing 'getStatusColor' call (needed for status badge coloring in breakdown)")
	}
	// The function must use optional-chaining to guard against missing status_counts
	if !strings.Contains(content, "status_counts") {
		t.Error("viewer.html missing 'status_counts' reference for Status Breakdown badges")
	}
}

// TestViewerHTMLFeatureProgressSection verifies that renderFeatureProgress()
// is present and produces a progress bar list from treeData. TC-VIS-05.
func TestViewerHTMLFeatureProgressSection(t *testing.T) {
	content := string(viewer.ViewerHTML)

	if !strings.Contains(content, "renderFeatureProgress") {
		t.Error("viewer.html missing 'renderFeatureProgress' function")
	}
	// Section heading visible to users
	if !strings.Contains(content, "Feature Progress") {
		t.Error("viewer.html missing 'Feature Progress' section heading")
	}
	// Uses treeData (hierarchy) to derive task counts
	if !strings.Contains(content, "treeData") {
		t.Error("viewer.html missing 'treeData' reference in feature progress")
	}
	// Progress bar uses CSS width % with blue accent fill
	if !strings.Contains(content, "var(--accent)") {
		t.Error("viewer.html missing 'var(--accent)' for progress bar fill color")
	}
}

// TestViewerHTMLFeatureProgressClickHandler verifies that clicking a feature
// key in the Feature Progress list calls navigateToEntity(). TC-VIS-05.
func TestViewerHTMLFeatureProgressClickHandler(t *testing.T) {
	content := string(viewer.ViewerHTML)

	if !strings.Contains(content, "navigateToEntity") {
		t.Error("viewer.html missing 'navigateToEntity' stub function for feature key clicks")
	}
}

// TestViewerHTMLDashboardNullGuards verifies that the dashboard gracefully
// handles null summaryData and empty treeData. TC-API-06.
func TestViewerHTMLDashboardNullGuards(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// Null summaryData guard — show inline message
	if !strings.Contains(content, "Failed to load status data") {
		t.Error("viewer.html missing 'Failed to load status data' message for null summaryData")
	}
	// Empty treeData guard — show inline message
	if !strings.Contains(content, "No features found") {
		t.Error("viewer.html missing 'No features found' message for empty treeData")
	}
}

// TestViewerHTMLDashboardStubsForTask5 verifies that stubs for Active
// Transitions and Stale Entities sections are present (to be completed in
// Task 5). TC-VIS-04.
func TestViewerHTMLDashboardStubsForTask5(t *testing.T) {
	content := string(viewer.ViewerHTML)

	if !strings.Contains(content, "renderActiveTransitions") {
		t.Error("viewer.html missing 'renderActiveTransitions' stub function (Task 5 target)")
	}
	if !strings.Contains(content, "renderStaleEntities") {
		t.Error("viewer.html missing 'renderStaleEntities' stub function (Task 5 target)")
	}
}

// TestViewerHTMLActiveTransitionsImplementation verifies that
// renderActiveTransitions() is fully implemented (Task 5). TC-VIS-06.
func TestViewerHTMLActiveTransitionsImplementation(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// Must NOT still be a stub
	if strings.Contains(content, "Active Transitions — coming in Task 5") {
		t.Error("renderActiveTransitions() is still a stub — must be implemented in Task 5")
	}

	// Empty-state placeholder
	if !strings.Contains(content, "No recent transitions") {
		t.Error("viewer.html missing 'No recent transitions' empty-state for Active Transitions")
	}

	// Must use recentActivity state variable
	if !strings.Contains(content, "recentActivity") {
		t.Error("viewer.html missing 'recentActivity' reference in renderActiveTransitions")
	}

	// Must use activity-table CSS class for the transitions table
	if !strings.Contains(content, "activity-table") {
		t.Error("viewer.html missing 'activity-table' CSS class in Active Transitions")
	}

	// Must use status badges with getStatusColor for from/to badges
	if !strings.Contains(content, "from_status") {
		t.Error("viewer.html missing 'from_status' field access in renderActiveTransitions")
	}
	if !strings.Contains(content, "to_status") {
		t.Error("viewer.html missing 'to_status' field access in renderActiveTransitions")
	}

	// Click handler: data-navigate-key attribute on the key span
	if !strings.Contains(content, "data-navigate-key") {
		t.Error("viewer.html missing 'data-navigate-key' attribute for key click navigation")
	}
}

// TestViewerHTMLStaleEntitiesImplementation verifies that renderStaleEntities()
// is fully implemented (Task 5). TC-STALE-01, TC-STALE-02, TC-STALE-03.
func TestViewerHTMLStaleEntitiesImplementation(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// Must NOT still be a stub
	if strings.Contains(content, "Stale Entities — coming in Task 5") {
		t.Error("renderStaleEntities() is still a stub — must be implemented in Task 5")
	}

	// Empty-state placeholder
	if !strings.Contains(content, "No stale entities found") {
		t.Error("viewer.html missing 'No stale entities found' empty-state for Stale Entities")
	}

	// Must use isTerminalStatus helper
	if !strings.Contains(content, "isTerminalStatus") {
		t.Error("viewer.html missing 'isTerminalStatus' call in renderStaleEntities")
	}

	// Must use daysSince helper
	if !strings.Contains(content, "daysSince") {
		t.Error("viewer.html missing 'daysSince' call in renderStaleEntities")
	}

	// Must guard with ?. on updated_at per spec
	if !strings.Contains(content, "updated_at") {
		t.Error("viewer.html missing 'updated_at' field access in renderStaleEntities")
	}

	// Must show "N days ago" text
	if !strings.Contains(content, "days ago") {
		t.Error("viewer.html missing 'days ago' text in Stale Entities rows")
	}

	// Sort descending by days (most stale first)
	if !strings.Contains(content, "b.days - a.days") {
		t.Error("viewer.html missing descending sort by days in renderStaleEntities (TC-STALE-03)")
	}
}

// TestViewerHTMLDataLoadingFixes verifies that the treeData and recentActivity
// assignments correctly unpack the API response objects. TC-VIS-06, TC-STALE-01.
func TestViewerHTMLDataLoadingFixes(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// treeData should unpack hierarchy.epics (not treat the whole response as an array)
	if !strings.Contains(content, "hierarchy?.epics") {
		t.Error("viewer.html must unpack hierarchy?.epics into treeData (API returns {epics:[...]})")
	}

	// recentActivity should unpack activity.records
	if !strings.Contains(content, "activity?.records") {
		t.Error("viewer.html must unpack activity?.records into recentActivity (API returns {records:[...]})")
	}
}

// TestViewerHTMLEntityViewSection verifies that the === ENTITY VIEW === section
// label and the renderEntityView function are fully implemented. TC-VIS-08.
func TestViewerHTMLEntityViewSection(t *testing.T) {
	content := string(viewer.ViewerHTML)

	if !strings.Contains(content, "=== ENTITY VIEW ===") {
		t.Error("viewer.html missing '=== ENTITY VIEW ===' section label")
	}
	if !strings.Contains(content, "renderEntityView") {
		t.Error("viewer.html missing 'renderEntityView' function")
	}
	// Must NOT still be a stub
	if strings.Contains(content, "Entity View — coming in Task 6") {
		t.Error("renderEntityView() is still a stub — must be implemented in Task 6")
	}
}

// TestViewerHTMLEntityViewHelpers verifies helper functions for the entity view
// are present. TC-VIS-08, TC-VIS-09.
func TestViewerHTMLEntityViewHelpers(t *testing.T) {
	content := string(viewer.ViewerHTML)

	helpers := []string{
		"findEntityByKey",
		"renderPropertiesPanel",
		"renderMarkdownPane",
		"showToast",
	}
	for _, fn := range helpers {
		if !strings.Contains(content, fn) {
			t.Errorf("viewer.html missing entity view helper function: %q", fn)
		}
	}
}

// TestViewerHTMLEntityViewToggle verifies that the Info/Transitions toggle
// buttons and the entityViewTab state variable are present. TC-VIS-08.
func TestViewerHTMLEntityViewToggle(t *testing.T) {
	content := string(viewer.ViewerHTML)

	if !strings.Contains(content, "entityViewTab") {
		t.Error("viewer.html missing 'entityViewTab' state variable")
	}
	if !strings.Contains(content, "ev-tab-info") {
		t.Error("viewer.html missing 'ev-tab-info' button id for Info tab")
	}
	if !strings.Contains(content, "ev-tab-transitions") {
		t.Error("viewer.html missing 'ev-tab-transitions' button id for Transitions tab")
	}
	if !strings.Contains(content, "toggle-btn") {
		t.Error("viewer.html missing 'toggle-btn' CSS class for Info/Transitions buttons")
	}
}

// TestViewerHTMLPropertiesPanelFields verifies that the properties panel
// renders the required metadata fields. TC-VIS-08.
func TestViewerHTMLPropertiesPanelFields(t *testing.T) {
	content := string(viewer.ViewerHTML)

	requiredFields := []string{
		"File Path",
		"Content Path",
		"props-grid",
		"props-label",
		"props-value",
		"copy-btn",
		"navigator.clipboard",
	}
	for _, field := range requiredFields {
		if !strings.Contains(content, field) {
			t.Errorf("viewer.html missing properties panel field/element: %q", field)
		}
	}
}

// TestViewerHTMLMarkdownRendering verifies that Marked.js CDN script tag and
// markdown rendering logic are present. TC-VIS-09, TC-API-08.
func TestViewerHTMLMarkdownRendering(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// Marked.js CDN script tag in <head>
	if !strings.Contains(content, "cdn.jsdelivr.net/npm/marked/marked.min.js") {
		t.Error("viewer.html missing Marked.js CDN script tag in <head>")
	}

	// marked.parse() call for rendering
	if !strings.Contains(content, "marked.parse(") {
		t.Error("viewer.html missing 'marked.parse()' call for markdown rendering")
	}

	// Fallback when CDN blocked
	if !strings.Contains(content, "typeof marked") {
		t.Error("viewer.html missing 'typeof marked' CDN availability check")
	}
	if !strings.Contains(content, "Markdown renderer unavailable (CDN blocked)") {
		t.Error("viewer.html missing CDN-blocked fallback warning banner text")
	}

	// No content placeholder
	if !strings.Contains(content, "No content available for this entity.") {
		t.Error("viewer.html missing 'No content available for this entity.' placeholder")
	}
}

// TestViewerHTMLEntityViewPropertiesStatusBadge verifies that the properties
// panel renders status as a colored badge using getStatusColor. TC-VIS-08.
func TestViewerHTMLEntityViewPropertiesStatusBadge(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// renderPropertiesPanel must use status-badge class + getStatusColor
	if !strings.Contains(content, "status-badge") {
		t.Error("viewer.html missing 'status-badge' class in properties panel")
	}
}

// TestViewerHTMLEntityViewHistoryTable verifies that the transitions/history
// view renders a table with from/to badges. TC-VIS-08.
func TestViewerHTMLEntityViewHistoryTable(t *testing.T) {
	content := string(viewer.ViewerHTML)

	if !strings.Contains(content, "history-table") {
		t.Error("viewer.html missing 'history-table' CSS class in entity view history")
	}
	if !strings.Contains(content, "history-date") {
		t.Error("viewer.html missing 'history-date' CSS class in entity view history")
	}
	if !strings.Contains(content, "apiGetHistory") {
		t.Error("viewer.html missing 'apiGetHistory' call in entity view transitions tab")
	}
}

// TestViewerHTMLEntityViewFindEntityByKey verifies that findEntityByKey walks
// epics, features and tasks to locate entities. TC-VIS-08.
func TestViewerHTMLEntityViewFindEntityByKey(t *testing.T) {
	content := string(viewer.ViewerHTML)

	if !strings.Contains(content, "findEntityByKey") {
		t.Error("viewer.html missing 'findEntityByKey' helper")
	}
	// Must walk treeData
	if !strings.Contains(content, "treeData") {
		t.Error("viewer.html missing 'treeData' reference in findEntityByKey")
	}
}
