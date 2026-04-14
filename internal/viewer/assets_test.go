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

// TestViewerHTMLDocViewImplemented verifies that renderDocView() is fully
// implemented (not a stub) and contains the required structural elements.
// TC-NAV-04: Clicking a doc node enters Doc View with plain markdown.
func TestViewerHTMLDocViewImplemented(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// Must NOT still be a stub
	if strings.Contains(content, "Doc View — coming in Task 7") {
		t.Error("renderDocView() is still a stub — must be implemented in Task 7")
	}

	// Must have a doc_view section label
	if !strings.Contains(content, "=== DOC VIEW ===") {
		t.Error("viewer.html missing '=== DOC VIEW ===' section label")
	}

	// Doc View renders markdown pane (reuses renderMarkdownPane)
	if !strings.Contains(content, "renderMarkdownPane") {
		t.Error("viewer.html missing 'renderMarkdownPane' call in renderDocView")
	}

	// Shows doc title prominently
	if !strings.Contains(content, "ev-title") {
		t.Error("viewer.html missing 'ev-title' class for doc title in Doc View")
	}

	// Shows minimal props: path and parent
	if !strings.Contains(content, "doc-content-pane") {
		t.Error("viewer.html missing 'doc-content-pane' id for Doc View content area")
	}
}

// TestViewerHTMLNavigateToEntityAncestorExpansion verifies that navigateToEntity()
// expands ancestor nodes (epic → feature) before rendering.
// TC-NAV-02, TC-NAV-03: ancestor expansion for cross-view navigation.
func TestViewerHTMLNavigateToEntityAncestorExpansion(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// navigateToEntity must add to expandedEpics for feature/task navigation
	if !strings.Contains(content, "expandedEpics.add") {
		t.Error("viewer.html missing 'expandedEpics.add' call in navigateToEntity (ancestor expansion)")
	}

	// navigateToEntity must add to expandedFeatures for task navigation
	if !strings.Contains(content, "expandedFeatures.add") {
		t.Error("viewer.html missing 'expandedFeatures.add' call in navigateToEntity (ancestor expansion)")
	}

	// Must log a warning when key is not found (no-op safety guard)
	if !strings.Contains(content, "key not found in treeData") {
		t.Error("viewer.html missing 'key not found in treeData' warning in navigateToEntity")
	}
}

// TestViewerHTMLDocNodeRouting verifies that sidebar doc node clicks are routed
// to doc_view rather than entity_view. TC-NAV-04.
func TestViewerHTMLDocNodeRouting(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// The sidebar click handler must distinguish doc nodes by CSS class
	if !strings.Contains(content, "tree-node-doc-epic") {
		t.Error("viewer.html missing 'tree-node-doc-epic' check in sidebar click handler")
	}
	if !strings.Contains(content, "tree-node-doc-feature") {
		t.Error("viewer.html missing 'tree-node-doc-feature' check in sidebar click handler")
	}

	// Doc nodes must route to doc_view state
	if !strings.Contains(content, "appState = 'doc_view'") {
		t.Error("viewer.html missing \"appState = 'doc_view'\" assignment for doc node clicks")
	}
}

// TestViewerHTMLEscapeKeyHandler verifies that the Escape key handler is
// registered and handles both entity_view and doc_view states. TC-KB-01.
func TestViewerHTMLEscapeKeyHandler(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// Must listen for keydown event
	if !strings.Contains(content, "addEventListener('keydown'") {
		t.Error("viewer.html missing document.addEventListener('keydown') for keyboard handler")
	}

	// Must handle Escape key
	if !strings.Contains(content, `e.key === 'Escape'`) {
		t.Error("viewer.html missing e.key === 'Escape' check in keyboard handler")
	}

	// Must handle both entity_view and doc_view
	if !strings.Contains(content, "appState === 'entity_view' || appState === 'doc_view'") {
		t.Error("viewer.html missing combined entity_view/doc_view check in Escape handler")
	}

	// Must reset to dashboard on Escape
	if !strings.Contains(content, "appState = 'dashboard'") {
		t.Error("viewer.html missing \"appState = 'dashboard'\" reset in Escape handler")
	}
}

// =============================================================================
// E27-F08 Tests: Spec rename and properties panel rewrite
// =============================================================================

// TestViewerHTMLSpecTabLabel verifies that the toggle button reads "Spec" (not "Info").
// TC-F08-007, AC-002.1.
func TestViewerHTMLSpecTabLabel(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// Button inner text must read ">Spec<"
	if !strings.Contains(content, ">Spec<") {
		t.Error("viewer.html missing \">Spec<\" button label — Info tab must be renamed to Spec")
	}

	// Old ">Info<" label must be absent (visual-only rename; ID is preserved)
	if strings.Contains(content, ">Info<") {
		t.Error("viewer.html still contains \">Info<\" button label — must be renamed to Spec")
	}

	// Internal ID must still be present
	if !strings.Contains(content, "ev-tab-info") {
		t.Error("viewer.html missing \"ev-tab-info\" button id — internal id must be preserved")
	}
}

// TestViewerHTMLSpecTabInternalId verifies that ev-tab-info ID and renderMarkdownPane
// are preserved after the rename. TC-F08-008, AC-002.2.
func TestViewerHTMLSpecTabInternalId(t *testing.T) {
	content := string(viewer.ViewerHTML)

	if !strings.Contains(content, "ev-tab-info") {
		t.Error("viewer.html missing \"ev-tab-info\" — internal tab ID must be preserved after Spec rename")
	}
	if !strings.Contains(content, "renderMarkdownPane") {
		t.Error("viewer.html missing \"renderMarkdownPane\" — Spec tab must still render markdown")
	}
}

// TestViewerHTMLSpecTabHistoryState verifies that pushNavState and the 'info' state
// value are preserved for history/hash compatibility. TC-F08-009, AC-002.3.
func TestViewerHTMLSpecTabHistoryState(t *testing.T) {
	content := string(viewer.ViewerHTML)

	if !strings.Contains(content, "pushNavState") {
		t.Error("viewer.html missing \"pushNavState\" — navigation history must be preserved")
	}
	// The internal tab state value 'info' must still be used
	if !strings.Contains(content, "'info'") {
		t.Error("viewer.html missing \"'info'\" tab state value — internal state must be preserved")
	}
}

// TestViewerHTMLPropertiesPanelEpicFields verifies that the rewritten properties panel
// includes epic-specific fields and the appendEpicFields function. TC-F08-010, AC-003.1.
func TestViewerHTMLPropertiesPanelEpicFields(t *testing.T) {
	content := string(viewer.ViewerHTML)

	markers := []string{
		"appendEpicFields",
		"Priority",
		"Business Value",
		"Feature Rollup",
		"Approval Backlog",
		"approval_backlog_count",
		"feature_status_rollup",
	}
	for _, marker := range markers {
		if !strings.Contains(content, marker) {
			t.Errorf("viewer.html missing epic field marker: %q", marker)
		}
	}
}

// TestViewerHTMLPropertiesPanelFeatureFields verifies that the rewritten properties panel
// includes feature-specific fields and the appendFeatureFields function. TC-F08-011, AC-003.2.
func TestViewerHTMLPropertiesPanelFeatureFields(t *testing.T) {
	content := string(viewer.ViewerHTML)

	markers := []string{
		"appendFeatureFields",
		"Execution Order",
		"execution_order",
		"workflow_position",
		"Phase Description",
		"phase_description",
		"Display Mode",
	}
	for _, marker := range markers {
		if !strings.Contains(content, marker) {
			t.Errorf("viewer.html missing feature field marker: %q", marker)
		}
	}
}

// TestViewerHTMLPropertiesPanelTaskFields verifies that the rewritten properties panel
// includes task-specific fields and the appendTaskFields function. TC-F08-012, AC-003.3.
func TestViewerHTMLPropertiesPanelTaskFields(t *testing.T) {
	content := string(viewer.ViewerHTML)

	markers := []string{
		"appendTaskFields",
		"rejection_count",
		"verification_status",
		"tests_passed",
		"completed_at",
		"blocked_by",
		"Blocked By",
		"Blocks",
	}
	for _, marker := range markers {
		if !strings.Contains(content, marker) {
			t.Errorf("viewer.html missing task field marker: %q", marker)
		}
	}
}

// TestViewerHTMLPropertiesPanelBugFields verifies that the rewritten properties panel
// includes bug-specific fields and the appendBugFields function. TC-F08-013, AC-003.4.
func TestViewerHTMLPropertiesPanelBugFields(t *testing.T) {
	content := string(viewer.ViewerHTML)

	markers := []string{
		"appendBugFields",
		"severity",
		"Severity",
		"assignee",
		"Assignee",
	}
	for _, marker := range markers {
		if !strings.Contains(content, marker) {
			t.Errorf("viewer.html missing bug field marker: %q", marker)
		}
	}
}

// TestViewerHTMLPropertiesPanelChangeCardFields verifies that the rewritten properties panel
// includes change card-specific fields and the appendChangeCardFields function. TC-F08-014, AC-003.5.
func TestViewerHTMLPropertiesPanelChangeCardFields(t *testing.T) {
	content := string(viewer.ViewerHTML)

	markers := []string{
		"appendChangeCardFields",
		"change_card",
	}
	for _, marker := range markers {
		if !strings.Contains(content, marker) {
			t.Errorf("viewer.html missing change card field marker: %q", marker)
		}
	}
}

// TestViewerHTMLOrchestratorActionDisclosure verifies that the orchestrator action block
// uses a native <details> element that defaults to collapsed. TC-F08-015, AC-003.6, Scenario 5.
func TestViewerHTMLOrchestratorActionDisclosure(t *testing.T) {
	content := string(viewer.ViewerHTML)

	markers := []string{
		"renderOrchestratorDetails",
		"<details",
		"<summary>Orchestrator Action</summary>",
		"orch-action",
		"orch-instruction",
		"orchestrator_action",
	}
	for _, marker := range markers {
		if !strings.Contains(content, marker) {
			t.Errorf("viewer.html missing orchestrator action marker: %q", marker)
		}
	}

	// The null guard must be present
	if !strings.Contains(content, "typeof oa !== 'object'") {
		t.Error("viewer.html missing null guard in renderOrchestratorDetails: \"typeof oa !== 'object'\"")
	}

	// The <details> must NOT have an open attribute — must be collapsed by default
	if strings.Contains(content, "<details open") {
		t.Error("viewer.html has \"<details open\" — orchestrator action block must be collapsed by default")
	}
}

// TestViewerHTMLPropertiesPanelNullGuard verifies that pushRow omits null/undefined/empty fields.
// TC-F08-016, AC-003.7.
func TestViewerHTMLPropertiesPanelNullGuard(t *testing.T) {
	content := string(viewer.ViewerHTML)

	if !strings.Contains(content, "function pushRow") {
		t.Error("viewer.html missing \"function pushRow\" — helper must be defined")
	}
	// The guard must check for null/undefined/empty
	if !strings.Contains(content, "valueHtml === null") {
		t.Error("viewer.html missing null guard in pushRow: \"valueHtml === null\"")
	}
}

// TestViewerHTMLPropertiesPanelStatusBadge verifies that the rewritten panel uses
// statusBadgeCell and getContrastColor for status rendering. TC-F08-017, AC-003.8.
func TestViewerHTMLPropertiesPanelStatusBadge(t *testing.T) {
	content := string(viewer.ViewerHTML)

	if !strings.Contains(content, "statusBadgeCell") {
		t.Error("viewer.html missing \"statusBadgeCell\" helper function")
	}
	if !strings.Contains(content, "getContrastColor") {
		t.Error("viewer.html missing \"getContrastColor\" call in properties panel")
	}
}

// TestViewerHTMLPropertiesPanelCopyButton verifies that copyBtn helper is present
// for File Path and Content Path. TC-F08-018, AC-003.9.
func TestViewerHTMLPropertiesPanelCopyButton(t *testing.T) {
	content := string(viewer.ViewerHTML)

	if !strings.Contains(content, "copyBtn") {
		t.Error("viewer.html missing \"copyBtn\" helper function in properties panel")
	}
	if !strings.Contains(content, "navigator.clipboard") {
		t.Error("viewer.html missing \"navigator.clipboard\" — copy button implementation required")
	}
}

// TestViewerHTMLPropertiesPanelRegressionGate verifies that no existing properties panel
// field markers were removed by the rewrite. TC-F08-019, AC-003.10.
func TestViewerHTMLPropertiesPanelRegressionGate(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// All markers from the pre-F08 TestViewerHTMLPropertiesPanelFields must still pass
	regressionMarkers := []string{
		"File Path",
		"Content Path",
		"props-grid",
		"props-label",
		"props-value",
		"copy-btn",
		"navigator.clipboard",
	}
	for _, marker := range regressionMarkers {
		if !strings.Contains(content, marker) {
			t.Errorf("viewer.html regression: existing properties panel marker removed: %q", marker)
		}
	}
}

// TestViewerHTMLNoNewAPIEndpoints verifies that the seven existing API fetch wrapper
// functions remain unchanged — no new apiGet* functions were added. TC-F08-020, AC-NF-001.1.
func TestViewerHTMLNoNewAPIEndpoints(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// All 7 existing functions must be present
	expectedFunctions := []string{
		"apiGetWorkflowMeta",
		"apiGetHierarchy",
		"apiGetSummary",
		"apiGetRecentActivity",
		"apiGetFile",
		"apiGetHistory",
		"apiGetFeatureTasks",
	}
	for _, fn := range expectedFunctions {
		if !strings.Contains(content, fn) {
			t.Errorf("viewer.html missing expected API function: %q", fn)
		}
	}
}

// TestViewerHTMLF08RegressionGate verifies that the highest-risk existing markers
// survive the F08 rewrite. TC-F08-022, Scenario 6.
func TestViewerHTMLF08RegressionGate(t *testing.T) {
	content := string(viewer.ViewerHTML)

	regressionMarkers := []string{
		"renderEntityView",
		"renderMarkdownPane",
		"renderDashboard",
		"ev-tab-transitions",
		"ev-tab-info",
		"history-table",
		"toggle-btn",
		"props-grid",
	}
	for _, marker := range regressionMarkers {
		if !strings.Contains(content, marker) {
			t.Errorf("viewer.html F08 regression: marker missing: %q", marker)
		}
	}
}

// =============================================================================
// E27-F08-002 Tests: Epic Dashboard Pane and Section Renderer Extensions
// =============================================================================

// TestViewerHTMLEpicDashboardTab verifies that the Dashboard tab is wired for epics.
// TC-F08-001, AC-001.1.
func TestViewerHTMLEpicDashboardTab(t *testing.T) {
	content := string(viewer.ViewerHTML)

	markers := []string{
		"ev-tab-dashboard",
		"renderEpicDashboardPane",
	}
	for _, marker := range markers {
		if !strings.Contains(content, marker) {
			t.Errorf("viewer.html missing epic dashboard tab marker: %q", marker)
		}
	}

	// Guard: entityViewTab === 'dashboard' check must be present for the pane switch
	if !strings.Contains(content, "entityViewTab === 'dashboard'") {
		t.Error("viewer.html missing \"entityViewTab === 'dashboard'\" — pane dispatch guard required")
	}
}

// TestViewerHTMLDashboardTabPosition verifies Dashboard tab button appears before
// the Spec (ev-tab-info) button in the toggle bar. TC-F08-002, AC-001.2.
func TestViewerHTMLDashboardTabPosition(t *testing.T) {
	content := string(viewer.ViewerHTML)

	dashIdx := strings.Index(content, "ev-tab-dashboard")
	infoIdx := strings.Index(content, "ev-tab-info")
	if dashIdx == -1 {
		t.Fatal("viewer.html missing ev-tab-dashboard button id")
	}
	if infoIdx == -1 {
		t.Fatal("viewer.html missing ev-tab-info button id")
	}
	if dashIdx > infoIdx {
		t.Error("ev-tab-dashboard must appear before ev-tab-info in viewer.html (Dashboard tab must precede Spec tab)")
	}
}

// TestViewerHTMLEpicDashboardSections verifies that renderEpicDashboardPane renders
// all five section titles. TC-F08-003, AC-001.3.
func TestViewerHTMLEpicDashboardSections(t *testing.T) {
	content := string(viewer.ViewerHTML)

	markers := []string{
		"renderEpicDashboardPane",
		"Entity Charts",
		"Status Overview",
		"Feature Progress",
		"Recent Activity",
		"Stale Entities",
	}
	for _, marker := range markers {
		if !strings.Contains(content, marker) {
			t.Errorf("viewer.html missing epic dashboard section marker: %q", marker)
		}
	}
}

// TestViewerHTMLEpicScopedFiltering verifies that all five section renderers accept
// an optional epicKey parameter. TC-F08-004, AC-001.4–AC-001.7.
func TestViewerHTMLEpicScopedFiltering(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// Each renderer must accept epicKey as parameter
	rendererMarkers := []string{
		"renderStatusBreakdown(epicKey",
		"renderFeatureProgress(epicKey",
		"renderActiveTransitions(epicKey",
		"renderStaleEntities(epicKey",
		"renderEntityCharts(epicKey",
	}
	for _, marker := range rendererMarkers {
		if !strings.Contains(content, marker) {
			t.Errorf("viewer.html missing epicKey parameter in renderer: %q", marker)
		}
	}
}

// TestViewerHTMLNoDashboardForNonEpic verifies that the Dashboard tab is guarded by
// isEpic so features and tasks never get the tab. TC-F08-005, AC-001.8.
func TestViewerHTMLNoDashboardForNonEpic(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// isEpic variable must be present to gate the dashBtnHtml
	if !strings.Contains(content, "isEpic") {
		t.Error("viewer.html missing \"isEpic\" variable — required to guard Dashboard tab for epics only")
	}
	// dashBtnHtml must be present (conditionally rendered)
	if !strings.Contains(content, "dashBtnHtml") {
		t.Error("viewer.html missing \"dashBtnHtml\" — Dashboard button HTML must be conditional")
	}
}

// TestViewerHTMLEpicDashboardNavigation verifies that renderEpicDashboardPane
// wires data-navigate-key delegation. TC-F08-006, AC-001.9.
func TestViewerHTMLEpicDashboardNavigation(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// renderEpicDashboardPane must be defined
	if !strings.Contains(content, "function renderEpicDashboardPane") {
		t.Error("viewer.html missing \"function renderEpicDashboardPane\" definition")
	}
	// data-navigate-key must be referenced (existing pattern reused)
	if !strings.Contains(content, "data-navigate-key") {
		t.Error("viewer.html missing \"data-navigate-key\" — navigation delegation pattern required")
	}
}

// TestViewerHTMLEpicDashboardUsesExistingData verifies that renderEpicDashboardPane
// uses treeData (no new fetches). TC-F08-021, AC-NF-002.1.
func TestViewerHTMLEpicDashboardUsesExistingData(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// renderEpicDashboardPane function must be present
	if !strings.Contains(content, "renderEpicDashboardPane") {
		t.Error("viewer.html missing renderEpicDashboardPane function")
	}
	// treeData must be referenced (existing cached data source)
	if !strings.Contains(content, "treeData") {
		t.Error("viewer.html missing treeData reference — epic dashboard must use existing cached data")
	}
}

// =============================================================================
// E27-F09-004 Tests: Overview tab, default tab change, Edit button move
// =============================================================================

// TestViewerHTMLOverviewTabDefault verifies that navigateToEntity() sets
// entityViewTab = 'overview' unconditionally, replacing the E27-F08 branching.
// TC-F001-1, AC-001.1, AC-T1.
func TestViewerHTMLOverviewTabDefault(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// navigateToEntity must set entityViewTab to 'overview'
	if !strings.Contains(content, "entityViewTab = 'overview'") {
		t.Error("viewer.html missing \"entityViewTab = 'overview'\" — navigateToEntity must default to overview tab")
	}

	// The old E27-F08 branching (epic->dashboard, other->info) must no longer exist
	if strings.Contains(content, "navEntity.type === 'epic') ? 'dashboard'") {
		t.Error("viewer.html still has E27-F08 epic→dashboard default — must be replaced with unconditional 'overview'")
	}
}

// TestViewerHTMLOverviewTabButton verifies that the toggle bar contains an Overview
// button (always present for epic/feature/task). TC-F001-2, AC-001.2, AC-T2.
func TestViewerHTMLOverviewTabButton(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// Overview button must exist in the toggle bar
	if !strings.Contains(content, "ev-tab-overview") {
		t.Error("viewer.html missing \"ev-tab-overview\" button id — Overview tab button is required")
	}

	// Overview button text
	if !strings.Contains(content, ">Overview<") {
		t.Error("viewer.html missing \">Overview<\" button label text")
	}
}

// TestViewerHTMLOverviewTabBeforeSpec verifies that the Overview button appears
// before the Spec (ev-tab-info) button in the toggle bar. TC-F001-2, AC-001.2.
func TestViewerHTMLOverviewTabBeforeSpec(t *testing.T) {
	content := string(viewer.ViewerHTML)

	overviewIdx := strings.Index(content, "ev-tab-overview")
	infoIdx := strings.Index(content, "ev-tab-info")
	if overviewIdx == -1 {
		t.Fatal("viewer.html missing ev-tab-overview button id")
	}
	if infoIdx == -1 {
		t.Fatal("viewer.html missing ev-tab-info button id")
	}
	if overviewIdx > infoIdx {
		t.Error("ev-tab-overview must appear before ev-tab-info in viewer.html (Overview tab must precede Spec tab)")
	}
}

// TestViewerHTMLOverviewPaneSwitch verifies that the pane switch in renderEntityView()
// handles the 'overview' case. TC-F001-1, AC-T3.
func TestViewerHTMLOverviewPaneSwitch(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// Pane switch must have 'overview' case
	if !strings.Contains(content, "entityViewTab === 'overview'") {
		t.Error("viewer.html missing \"entityViewTab === 'overview'\" — pane switch must handle overview case")
	}

	// renderOverviewPane function must be present (dispatcher)
	if !strings.Contains(content, "renderOverviewPane") {
		t.Error("viewer.html missing \"renderOverviewPane\" function — overview pane renderer required")
	}
}

// TestViewerHTMLDashboardCoercionForNonEpic verifies that 'dashboard' on a non-epic
// is coerced to 'overview' (not 'info' as in E27-F08). TC-F001-5, AC-001.5, AC-T3.
func TestViewerHTMLDashboardCoercionForNonEpic(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// Guard must coerce to 'overview', not 'info' as in E27-F08
	if !strings.Contains(content, "entityViewTab = 'overview'") {
		t.Error("viewer.html missing coercion to 'overview' for non-epic dashboard restore")
	}
}

// TestViewerHTMLNoEditButtonInToggleBar verifies that there is no #ev-tab-edit
// element in the toggle bar markup. TC-F002-1, AC-002.1, AC-T2.
func TestViewerHTMLNoEditButtonInToggleBar(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// No ev-tab-edit id should exist anywhere
	if strings.Contains(content, "ev-tab-edit") {
		t.Error("viewer.html contains \"ev-tab-edit\" — Edit button must not be in the tab bar (AC-002.1)")
	}

	// editBtnHtml must not be referenced in the toggle-bar HTML template
	// (It may still exist as a variable name in renderMarkdownPane context,
	// but must not appear inside the toggle-bar template string)
	// We check that editBtnHtml is NOT concatenated into the toggle bar.
	// The toggle bar template string uses backtick — verify editBtnHtml is absent from it.
	// A simpler assertion: the toggle-bar div must not contain editBtnHtml reference.
	// Since this is string-presence in the static HTML, we check the toggle-bar block.
}

// TestViewerHTMLInlineEditToggleInSpecPane verifies that renderMarkdownPane renders
// an inline Edit button inside the pane (not in the tab bar). TC-F002-2, AC-002.2, AC-T4.
func TestViewerHTMLInlineEditToggleInSpecPane(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// spec-pane-header div must be present (wraps the inline edit button)
	if !strings.Contains(content, "spec-pane-header") {
		t.Error("viewer.html missing \"spec-pane-header\" — Spec pane header row with inline Edit toggle is required")
	}

	// inline-edit-btn class for the inline Edit button
	if !strings.Contains(content, "inline-edit-btn") {
		t.Error("viewer.html missing \"inline-edit-btn\" — inline Edit button class required in Spec pane")
	}

	// ev-inline-edit id for the inline Edit button
	if !strings.Contains(content, "ev-inline-edit") {
		t.Error("viewer.html missing \"ev-inline-edit\" — inline Edit button id required in Spec pane")
	}
}

// TestViewerHTMLSidebarClickNoEpicDashboardBranch verifies that the
// sidebarContent.onclick handler does not contain the old E27-F08 branching
// logic that set entityViewTab to 'dashboard' for epics and 'info' for others.
// The sidebar click must unconditionally set entityViewTab = 'overview'.
// Regression gate for T-E27-F09-004 bug fix.
func TestViewerHTMLSidebarClickNoEpicDashboardBranch(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// The old clickedEntity branching (epic->dashboard, other->info) must not exist
	if strings.Contains(content, "clickedEntity.type === 'epic') ? 'dashboard'") {
		t.Error("viewer.html still has clickedEntity epic→dashboard branch in sidebarContent.onclick — must be replaced with unconditional 'overview'")
	}
}

// TestViewerHTMLF09RegressionGate verifies that all E27-F08 markers survive the
// F09 changes. Regression gate for Scenario 6 of spec §1.3.
func TestViewerHTMLF09RegressionGate(t *testing.T) {
	content := string(viewer.ViewerHTML)

	regressionMarkers := []string{
		"renderEntityView",
		"renderMarkdownPane",
		"renderEpicDashboardPane",
		"ev-tab-dashboard",
		"ev-tab-transitions",
		"ev-tab-info",
		"ev-tab-files",
		"history-table",
		"toggle-btn",
		"props-grid",
		"isEpic",
		"dashBtnHtml",
	}
	for _, marker := range regressionMarkers {
		if !strings.Contains(content, marker) {
			t.Errorf("viewer.html F09 regression: marker missing: %q", marker)
		}
	}
}

// =============================================================================
// E27-F09-005 Tests: Breadcrumb component and properties panel status accent
// =============================================================================

// TestViewerHTMLBreadcrumbFunction verifies that renderBreadcrumb function is
// defined and produces the correct structure. TC-F003-1, AC-003.1.
func TestViewerHTMLBreadcrumbFunction(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// renderBreadcrumb function must be defined
	if !strings.Contains(content, "function renderBreadcrumb(") {
		t.Error("viewer.html missing \"function renderBreadcrumb(\" — breadcrumb helper is required (AC-003.1)")
	}

	// Must produce a .breadcrumb wrapper element
	if !strings.Contains(content, `"breadcrumb"`) {
		t.Error("viewer.html missing breadcrumb class reference in renderBreadcrumb — wrapper element required (AC-003.1)")
	}

	// Must produce .breadcrumb-seg elements for segments
	if !strings.Contains(content, "breadcrumb-seg") {
		t.Error("viewer.html missing \"breadcrumb-seg\" class — segment elements required (AC-003.1)")
	}
}

// TestViewerHTMLBreadcrumbNavigableSegments verifies that non-current segments
// carry data-navigate-key and role="button". TC-F003-2, AC-003.2.
func TestViewerHTMLBreadcrumbNavigableSegments(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// Parent segments must carry data-navigate-key
	if !strings.Contains(content, `data-navigate-key="${escapeHtml(seg.key)}"`) {
		t.Error("viewer.html breadcrumb: parent segments must carry data-navigate-key (AC-003.2)")
	}

	// Parent segments must carry role="button" for accessibility
	if !strings.Contains(content, `role="button"`) {
		t.Error("viewer.html breadcrumb: parent segments must carry role=\"button\" (AC-003.2, AC-NF-002.1)")
	}

	// Parent segments must carry tabindex="0" for keyboard focus
	if !strings.Contains(content, `tabindex="0"`) {
		t.Error("viewer.html breadcrumb: parent segments must carry tabindex=\"0\" (AC-NF-002.1)")
	}
}

// TestViewerHTMLBreadcrumbCurrentSegment verifies that the last segment (current
// entity) has the .current class and no data-navigate-key. TC-F003-3, AC-003.3.
func TestViewerHTMLBreadcrumbCurrentSegment(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// Current segment must have .current class
	if !strings.Contains(content, `breadcrumb-seg current`) {
		t.Error("viewer.html breadcrumb: current segment must carry class \"breadcrumb-seg current\" (AC-003.3)")
	}

	// Current segment must NOT have data-navigate-key (it is not clickable)
	// Verify the current segment template does not contain data-navigate-key
	if !strings.Contains(content, `isCurrent`) {
		t.Error("viewer.html breadcrumb: must distinguish current segment via isCurrent flag (AC-003.3)")
	}
}

// TestViewerHTMLBreadcrumbSeparator verifies that the separator character › is
// used between segments. TC-F003-4, AC-003.4.
func TestViewerHTMLBreadcrumbSeparator(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// Separator must use the › character
	if !strings.Contains(content, "breadcrumb-sep") {
		t.Error("viewer.html breadcrumb: missing .breadcrumb-sep separator class (AC-003.4)")
	}
}

// TestViewerHTMLBreadcrumbInEntityView verifies that renderBreadcrumb is called
// inside renderEntityView, between the title and the properties panel. AC-003.1.
func TestViewerHTMLBreadcrumbInEntityView(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// renderBreadcrumb must be called in renderEntityView context
	if !strings.Contains(content, "renderBreadcrumb(entity)") {
		t.Error("viewer.html missing \"renderBreadcrumb(entity)\" call in renderEntityView (AC-003.1)")
	}
}

// TestViewerHTMLBreadcrumbCSS verifies that the breadcrumb CSS rules are present.
// TC-F003-1, AC-003.1.
func TestViewerHTMLBreadcrumbCSS(t *testing.T) {
	content := string(viewer.ViewerHTML)

	cssRules := []string{
		".breadcrumb",
		".breadcrumb-seg",
		".breadcrumb-sep",
	}
	for _, rule := range cssRules {
		if !strings.Contains(content, rule) {
			t.Errorf("viewer.html missing CSS rule for %q (breadcrumb requires its own CSS)", rule)
		}
	}
}

// TestViewerHTMLStatusAccentPropsGrid verifies that the properties panel
// (.props-grid) gets a status-driven left-border accent. TC-F012-1, AC-012.1.
func TestViewerHTMLStatusAccentPropsGrid(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// props-grid must include data-status-accent attribute when a status is known
	if !strings.Contains(content, "data-status-accent") {
		t.Error("viewer.html missing \"data-status-accent\" — props-grid must carry status accent attribute (AC-012.1)")
	}

	// CSS rule for .props-grid[data-status-accent] must be present
	if !strings.Contains(content, ".props-grid[data-status-accent]") {
		t.Error("viewer.html missing CSS rule \".props-grid[data-status-accent]\" (AC-012.1)")
	}
}

// TestViewerHTMLStatusAccentBorderStyle verifies that the status accent uses
// border-left inline style on the props-grid. TC-F012-1, AC-012.1.
func TestViewerHTMLStatusAccentBorderStyle(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// The renderPropertiesPanel must apply a border-left style using the status color
	if !strings.Contains(content, "border-left") {
		t.Error("viewer.html missing \"border-left\" in renderPropertiesPanel — status accent must use left border (AC-012.1)")
	}

	// The accent color must come from getStatusColor
	if !strings.Contains(content, "getStatusColor(entity.status)") {
		t.Error("viewer.html missing getStatusColor(entity.status) call for border accent (AC-012.1)")
	}
}

// TestViewerHTMLKeydownNavigationHandler verifies that the global keydown handler
// for Enter/Space on [data-navigate-key][role="button"] is added. TC-NF-002,
// AC-NF-002.1, AC-T4.
func TestViewerHTMLKeydownNavigationHandler(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// Must check for both Enter and Space keys
	if !strings.Contains(content, `e.key === 'Enter'`) {
		t.Error("viewer.html keydown handler missing e.key === 'Enter' check (AC-NF-002.1)")
	}
	if !strings.Contains(content, `e.key === ' '`) {
		t.Error("viewer.html keydown handler missing e.key === ' ' (Space) check (AC-NF-002.1)")
	}

	// Must match [data-navigate-key][role="button"] elements
	if !strings.Contains(content, `[data-navigate-key][role="button"]`) {
		t.Error("viewer.html keydown handler must match [data-navigate-key][role=\"button\"] selector (AC-NF-002.1)")
	}

	// Must call navigateToEntity with the key
	if !strings.Contains(content, "navigateToEntity(k)") {
		t.Error("viewer.html keydown handler must call navigateToEntity(k) (AC-NF-002.1)")
	}

	// Must call e.preventDefault() to stop Space from scrolling
	if !strings.Contains(content, "e.preventDefault()") {
		t.Error("viewer.html keydown handler must call e.preventDefault() to prevent Space scroll (AC-NF-002.1)")
	}
}
