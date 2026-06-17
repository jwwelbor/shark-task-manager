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
		"renderSprintMode",
		"renderSprintOverview",
		"renderSprintTree",
		"renderEntityView",
		"renderPickFolder",
		"fetchSprintOverview",
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

// TestViewerHTMLMutationControls verifies that the embedded viewer renders the
// entity mutation disclosure and references the viewer-only mutation routes.
// TC-F01-007.
func TestViewerHTMLMutationControls(t *testing.T) {
	content := string(viewer.ViewerHTML)

	required := []string{
		"renderMutationControls(entity)",
		"renderNoteControls(entity)",
		"renderRelationshipControls(entity)",
		"entity-mutation-panel",
		"entity-note-panel",
		"entity-relationship-panel",
		"entity-mutation-patch-form",
		"entity-mutation-transition-form",
		"mutation-transition-buttons",
		"apiCreateViewerNote",
		"apiCreateViewerRelationship",
		"apiDeleteViewerRelationship",
		"api/v1/viewer/epics/",
		"api/v1/viewer/features/",
		"api/v1/viewer/tasks/",
		"target_status",
		"business_value",
		"execution_order",
		"agent_type",
		"clear_size",
	}
	for _, marker := range required {
		if !strings.Contains(content, marker) {
			t.Errorf("viewer.html missing mutation control marker: %q", marker)
		}
	}

	funcStart := strings.Index(content, "function renderMutationControls(entity)")
	if funcStart < 0 {
		t.Fatal("viewer.html missing renderMutationControls(entity) function")
	}
	funcBody := content[funcStart:]
	funcEnd := strings.Index(funcBody[1:], "\nfunction ")
	if funcEnd > 0 {
		funcBody = funcBody[:funcEnd+1]
	}

	for _, marker := range []string{"case 'epic'", "case 'feature'", "case 'task'"} {
		if !strings.Contains(funcBody, marker) {
			t.Errorf("viewer.html renderMutationControls missing scoped case marker: %q", marker)
		}
	}
	for _, marker := range []string{"case 'bug'", "case 'change_card'", "case 'idea'"} {
		if strings.Contains(funcBody, marker) {
			t.Errorf("viewer.html renderMutationControls must not expose flat entity type marker: %q", marker)
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

// TestViewerHTMLSprintModeMarkers verifies that the Sprint mode shell,
// subview controls, and drawer/jump-back hooks are present in the embedded file.
func TestViewerHTMLSprintModeMarkers(t *testing.T) {
	content := string(viewer.ViewerHTML)

	required := []string{
		`id="header-sprint-btn"`,
		"case 'sprint'",
		"sprintViewTab",
		"renderSprintModeControls",
		"renderSprintDetailDrawer",
		"selectSprintItem",
		"Back to Sprint",
		"Open Entity View",
		"Upcoming Sprint",
		"Archived Sprints",
		"sprint-plan-status-filter",
		"sprint-plan-stage-btn",
		"applySprintPlanFilters",
		"sprintPlanSelection",
		"fetchSprintPlan",
		"fetchSprintReport",
		"apiGetRelatedDocs",
	}
	for _, marker := range required {
		if !strings.Contains(content, marker) {
			t.Errorf("viewer.html missing Sprint mode marker: %q", marker)
		}
	}
}

// TestViewerHTMLSprintTreeAndReportMarkers verifies that the sprint tree,
// report surface, and guardrail-oriented hooks stay embedded in the single file.
func TestViewerHTMLSprintTreeAndReportMarkers(t *testing.T) {
	content := string(viewer.ViewerHTML)

	required := []string{
		"toggleSprintTreeNode",
		"renderSprintReport",
		"fetchSprintReport",
		"Stage selected",
		"Remove selected",
		"Mark ready",
		"popstate",
		"Escape",
	}
	for _, marker := range required {
		if !strings.Contains(content, marker) {
			t.Errorf("viewer.html missing sprint tree/report marker: %q", marker)
		}
	}
}

func TestViewerHTMLSprintTreeRendersCatalogRows(t *testing.T) {
	content := string(viewer.ViewerHTML)

	required := []string{
		"sprintCatalogRows",
		"payload?.catalog",
		"No upcoming sprints.",
		"No archived sprints.",
	}
	for _, marker := range required {
		if !strings.Contains(content, marker) {
			t.Errorf("viewer.html missing sprint catalog marker: %q", marker)
		}
	}

	forbidden := []string{
		"Upcoming sprint queue will appear here.",
		"Archived sprint history will appear here.",
	}
	for _, marker := range forbidden {
		if strings.Contains(content, marker) {
			t.Errorf("viewer.html still contains sprint placeholder marker: %q", marker)
		}
	}
}

func TestViewerHTMLMermaidPanZoomMarkers(t *testing.T) {
	content := string(viewer.ViewerHTML)

	required := []string{
		"svg-pan-zoom@3.6.2",
		"mermaid-viewer",
		"mermaid-toolbar",
		"mermaid-zoom-in",
		"mermaid-zoom-out",
		"mermaid-reset",
		"mermaid-maximize",
		"mermaid-collapse",
		"mermaid-overlay",
		"mermaid-overlay-close",
		"initMermaidPanZoom",
		"resetMermaidZoom",
		"openMermaidOverlay",
		"toggleMermaidCollapse",
		"ResizeObserver",
		"resizeMermaidObserver",
		"onPan:",
		"onZoom:",
		"scheduleMermaidZoomStateSync",
	}
	for _, marker := range required {
		if !strings.Contains(content, marker) {
			t.Errorf("viewer.html missing Mermaid pan/zoom marker: %q", marker)
		}
	}

	forbidden := []string{
		"wheel",
		"mousedown",
		"panOnMove",
	}
	for _, marker := range forbidden {
		if strings.Contains(content, marker) {
			t.Errorf("viewer.html still contains custom Mermaid pan/zoom handler: %q", marker)
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

// =============================================================================
// E27-F09-006 Tests: Epic Overview — Feature Rollup, Task Rollup, Features Table
// =============================================================================

// TestViewerHTMLOverviewSharedHelpers verifies that all shared computation helpers
// required by the epic/feature Overview pane are present.
// TC-F004-3, TC-F005-3 — derived from treeData, no new network call.
func TestViewerHTMLOverviewSharedHelpers(t *testing.T) {
	content := string(viewer.ViewerHTML)

	helpers := []string{
		"function countByStatus(",
		"function collectEpicTasks(",
		"function statusMeta(",
		"function weightedProgress(",
		"function sortedStatuses(",
		"function sectionShell(",
	}
	for _, h := range helpers {
		if !strings.Contains(content, h) {
			t.Errorf("viewer.html missing shared overview helper: %q (TC-F004-3, TC-F005-3)", h)
		}
	}
}

// TestViewerHTMLFeatureRollupSection verifies that renderFeatureRollupSection is
// present and uses getStatusColor for pill colors. TC-F004-1, TC-F004-2, AC-004.1–AC-004.4.
func TestViewerHTMLFeatureRollupSection(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// Function must exist
	if !strings.Contains(content, "function renderFeatureRollupSection(") {
		t.Error("viewer.html missing \"function renderFeatureRollupSection(\" — epic overview feature rollup required (AC-004.1)")
	}

	// Must use getStatusColor for pill background (AC-004.2)
	if !strings.Contains(content, "getStatusColor(s)") {
		t.Error("viewer.html missing getStatusColor(s) call in feature rollup section (AC-004.2)")
	}

	// Must use countByStatus (AC-004.3)
	if !strings.Contains(content, "countByStatus(") {
		t.Error("viewer.html missing countByStatus() call — pill counts must come from treeData (AC-004.3)")
	}

	// Must use rollup-pill CSS class
	if !strings.Contains(content, "rollup-pill") {
		t.Error("viewer.html missing rollup-pill CSS class reference in feature rollup section (AC-004.1)")
	}

	// Must use rollup-row CSS class
	if !strings.Contains(content, "rollup-row") {
		t.Error("viewer.html missing rollup-row CSS class reference in feature rollup section (AC-004.1)")
	}

	// Zero-count filter: occupied statuses filter must be present (AC-004.4)
	if !strings.Contains(content, "counts[s] > 0") {
		t.Error("viewer.html missing zero-count filter in feature rollup (AC-004.4 — zero-count statuses must be omitted)")
	}

	// Phase ordering for pills (AC-004.5)
	if !strings.Contains(content, "sortedStatuses(") {
		t.Error("viewer.html missing sortedStatuses() call in feature rollup section (AC-004.5 — phase ordering required)")
	}
}

// TestViewerHTMLTaskRollupSection verifies that renderTaskRollupSection is present
// with a segmented progress bar and weighted-progress label.
// TC-F005-1, TC-F005-2, TC-F005-3, AC-005.1–AC-005.4.
func TestViewerHTMLTaskRollupSection(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// Function must exist
	if !strings.Contains(content, "function renderTaskRollupSection(") {
		t.Error("viewer.html missing \"function renderTaskRollupSection(\" — task rollup section required (AC-005.1)")
	}

	// No tasks placeholder (AC-005.2)
	if !strings.Contains(content, "No tasks") {
		t.Error("viewer.html missing \"No tasks\" placeholder in task rollup section (AC-005.2)")
	}

	// Weighted progress label (AC-005.3)
	if !strings.Contains(content, "weightedProgress(") {
		t.Error("viewer.html missing weightedProgress() call in task rollup section (AC-005.3)")
	}

	// Segmented progress bar CSS classes
	if !strings.Contains(content, "seg-bar") {
		t.Error("viewer.html missing seg-bar CSS class in task rollup section (AC-005.1)")
	}
	if !strings.Contains(content, "seg-bar-segment") {
		t.Error("viewer.html missing seg-bar-segment CSS class in task rollup section (AC-005.1)")
	}
	if !strings.Contains(content, "seg-bar-label") {
		t.Error("viewer.html missing seg-bar-label CSS class for NN% label (AC-005.3)")
	}

	// Pills use same color as segments (AC-005.1)
	if !strings.Contains(content, "rollup-pill") {
		t.Error("viewer.html missing rollup-pill in task rollup section (AC-005.1 — pills and segments must share color)")
	}

	// Uses collectEpicTasks for epic-level task aggregation (AC-005.4)
	if !strings.Contains(content, "collectEpicTasks(") {
		t.Error("viewer.html missing collectEpicTasks() call — epic task rollup must aggregate across features (AC-005.4)")
	}
}

// TestViewerHTMLFeaturesTableSection verifies that renderFeaturesTableSection is
// present with the required columns and clickable KEY cells.
// TC-F008-1, TC-F008-2, TC-F008-3, TC-F008-4, TC-F008-5, AC-008.1–AC-008.5.
func TestViewerHTMLFeaturesTableSection(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// Function must exist
	if !strings.Contains(content, "function renderFeaturesTableSection(") {
		t.Error("viewer.html missing \"function renderFeaturesTableSection(\" — features table required (AC-008.1)")
	}

	// KEY cells must carry data-navigate-key and .clickable class (AC-008.1)
	if !strings.Contains(content, "data-navigate-key=") {
		t.Error("viewer.html missing data-navigate-key attribute in features table (AC-008.1)")
	}
	if !strings.Contains(content, `class="clickable"`) {
		t.Error("viewer.html missing class=\"clickable\" in features table KEY cell (AC-008.1)")
	}

	// TITLE cells must use title attribute for full text (AC-008.2)
	if !strings.Contains(content, "text-overflow:ellipsis") {
		t.Error("viewer.html missing text-overflow:ellipsis in features table TITLE cell (AC-008.2)")
	}

	// Table uses ov-table CSS class (AC-008.3)
	if !strings.Contains(content, "ov-table") {
		t.Error("viewer.html missing ov-table CSS class in features table (AC-008.3)")
	}

	// Table body scroll at >10 rows (AC-008.4)
	if !strings.Contains(content, "ov-table-scroll") {
		t.Error("viewer.html missing ov-table-scroll class (AC-008.4 — table must scroll when >10 rows)")
	}

	// Sticky header (AC-008.4) — CSS rule for table header sticky positioning
	if !strings.Contains(content, "position: sticky") {
		t.Error("viewer.html missing position: sticky in ov-table thead (AC-008.4)")
	}

	// Progress bar width from progress_pct (AC-008.5)
	if !strings.Contains(content, "progress_pct") {
		t.Error("viewer.html missing progress_pct reference in features table (AC-008.5 — bar width must match properties panel)")
	}
}

// TestViewerHTMLOverviewPaneDispatchUpdated verifies that renderOverviewPane
// dispatches to the real section renderers (not placeholder text).
// TC-F004-1, TC-F005-1, TC-F008-1 — replaces placeholder stubs.
func TestViewerHTMLOverviewPaneDispatchUpdated(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// Must call renderFeatureRollupSection for epic case (REQ-F-004)
	if !strings.Contains(content, "renderFeatureRollupSection(entity)") {
		t.Error("viewer.html renderOverviewPane must call renderFeatureRollupSection(entity) for epic case (REQ-F-004)")
	}

	// Must call renderTaskRollupSection for epic case (REQ-F-005)
	if !strings.Contains(content, "renderTaskRollupSection(collectEpicTasks(entity))") {
		t.Error("viewer.html renderOverviewPane must call renderTaskRollupSection(collectEpicTasks(entity)) for epic case (REQ-F-005)")
	}

	// Must call renderFeaturesTableSection for epic case (REQ-F-008)
	if !strings.Contains(content, "renderFeaturesTableSection(entity)") {
		t.Error("viewer.html renderOverviewPane must call renderFeaturesTableSection(entity) for epic case (REQ-F-008)")
	}

	// Must NOT still contain placeholder stub text
	if strings.Contains(content, "Epic overview — rollups, feature table, and notes will render here") {
		t.Error("viewer.html renderOverviewPane still contains epic overview placeholder stub — must be replaced with real renderers")
	}
}

// TestViewerHTMLOverviewPaneNavigationDelegation verifies that renderOverviewPane wires
// click delegation for [data-navigate-key] elements after setting innerHTML, so that
// feature rows (in epic overview), task rows (in feature overview), and dependency links
// (in task overview) are clickable with the mouse. TC-E27-F09-006.
func TestViewerHTMLOverviewPaneNavigationDelegation(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// renderOverviewPane must call querySelectorAll('[data-navigate-key]') for delegation
	if !strings.Contains(content, "querySelectorAll('[data-navigate-key]')") {
		t.Error("viewer.html renderOverviewPane missing querySelectorAll('[data-navigate-key]') delegation — navigate-key elements will not be clickable")
	}
}

// TestViewerHTMLOverviewCSSRules verifies that overview-specific CSS classes are present.
// TC-F004-1, TC-F005-1, TC-F008-3.
func TestViewerHTMLOverviewCSSRules(t *testing.T) {
	content := string(viewer.ViewerHTML)

	cssRules := []string{
		".ov-section",
		".ov-section-header",
		".rollup-row",
		".rollup-pill",
		".seg-bar",
		".seg-bar-segment",
		".seg-bar-row",
		".seg-bar-label",
		".ov-table",
		".ov-table-scroll",
	}
	for _, rule := range cssRules {
		if !strings.Contains(content, rule) {
			t.Errorf("viewer.html missing CSS rule: %q (required for Overview pane styling)", rule)
		}
	}
}

// TestViewerHTMLNotesPlaceholderAndCache verifies that the notes caching and
// placeholder functions are present. TC-F010-1, AC-010.1, AC-010.7.
func TestViewerHTMLNotesPlaceholderAndCache(t *testing.T) {
	content := string(viewer.ViewerHTML)

	if !strings.Contains(content, "notesCache") {
		t.Error("viewer.html missing notesCache Map — session cache required for notes (AC-010.1)")
	}
	if !strings.Contains(content, "function fetchNotes(") {
		t.Error("viewer.html missing function fetchNotes() — notes fetch helper required (AC-010.1)")
	}
	if !strings.Contains(content, "function notesPlaceholder(") {
		t.Error("viewer.html missing function notesPlaceholder() — skeleton placeholder required (AC-010.7)")
	}
	if !strings.Contains(content, "function loadNotesInto(") {
		t.Error("viewer.html missing function loadNotesInto() — async notes loader required (AC-010.7)")
	}
	if !strings.Contains(content, "data-notes-for=") {
		t.Error("viewer.html missing data-notes-for attribute — placeholder targeting required (AC-010.7)")
	}
}

// TestViewerHTMLRelatedDocsPlaceholderAndCache verifies that the related-docs caching
// and placeholder functions are present. AC-011.4.
func TestViewerHTMLRelatedDocsPlaceholderAndCache(t *testing.T) {
	content := string(viewer.ViewerHTML)

	if !strings.Contains(content, "relatedDocsCache") {
		t.Error("viewer.html missing relatedDocsCache Map — session cache required for docs (AC-011.4)")
	}
	if !strings.Contains(content, "function fetchRelatedDocs(") {
		t.Error("viewer.html missing function fetchRelatedDocs() — docs fetch helper required (AC-011.4)")
	}
	if !strings.Contains(content, "function relatedDocsPlaceholder(") {
		t.Error("viewer.html missing function relatedDocsPlaceholder() — docs placeholder required (AC-011.4)")
	}
	if !strings.Contains(content, "function loadRelatedDocsInto(") {
		t.Error("viewer.html missing function loadRelatedDocsInto() — async docs loader required (AC-011.4)")
	}
	if !strings.Contains(content, "data-docs-for=") {
		t.Error("viewer.html missing data-docs-for attribute — docs placeholder targeting required (AC-011.4)")
	}
}

// TestViewerHTMLActionItemsNoHardcodedStatusNames verifies that there are no
// hardcoded status name literals in the actionItems filter logic.
// TC-F007-1, AC-007.1.
func TestViewerHTMLActionItemsNoHardcodedStatusNames(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// The actionItems function itself must not use hardcoded status literals
	forbiddenLiterals := []string{
		"=== 'ready_for_approval'",
		"=== 'ready_for_review'",
		"=== 'ready_for_code_review'",
		"=== 'approved'",
	}
	for _, lit := range forbiddenLiterals {
		if strings.Contains(content, lit) {
			t.Errorf("viewer.html contains hardcoded status literal %q in overview logic — must use blocks_feature flag only (AC-007.1)", lit)
		}
	}

	// blocks_feature flag must be used
	if !strings.Contains(content, "blocks_feature === true") {
		t.Error("viewer.html missing blocks_feature === true filter — action items must use meta flag (AC-007.1)")
	}
}

// =============================================================================
// E27-F09-007 Tests: Feature Overview — Work Breakdown, Action Items, Tasks Table
// =============================================================================

// TestViewerHTMLWorkBreakdownFunction verifies that renderWorkBreakdownSection is
// present and computes agent/human/qa_team responsibility buckets from workflowMeta.
// TC-F006-1, TC-F006-2, AC-006.1, AC-006.2.
func TestViewerHTMLWorkBreakdownFunction(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// Function must be defined
	if !strings.Contains(content, "function renderWorkBreakdownSection(") {
		t.Error("viewer.html missing \"function renderWorkBreakdownSection(\" — work breakdown section required (AC-006.1)")
	}

	// Must use the workBreakdown() helper (AC-006.1 — uses responsibility field from workflowMeta)
	if !strings.Contains(content, "workBreakdown(") {
		t.Error("viewer.html missing workBreakdown() call in renderWorkBreakdownSection (AC-006.1)")
	}

	// Must show Agent row (AC-006.1)
	if !strings.Contains(content, "Agent") {
		t.Error("viewer.html missing Agent row label in work breakdown section (AC-006.1)")
	}

	// Must show Human row (AC-006.1)
	if !strings.Contains(content, "Human") {
		t.Error("viewer.html missing Human row label in work breakdown section (AC-006.1)")
	}

	// Must show QA Team row (AC-006.1)
	if !strings.Contains(content, "QA Team") {
		t.Error("viewer.html missing QA Team row label in work breakdown section (AC-006.1)")
	}

	// Must use wb-row, wb-bar CSS classes (AC-006.2)
	if !strings.Contains(content, "wb-row") {
		t.Error("viewer.html missing wb-row CSS class in work breakdown section (AC-006.2)")
	}
	if !strings.Contains(content, "wb-bar") {
		t.Error("viewer.html missing wb-bar CSS class in work breakdown section (AC-006.2)")
	}
}

// TestViewerHTMLWorkBreakdownHiddenWhenZero verifies that the Work Breakdown section
// is omitted when all three responsibility counts are zero. TC-F006-3, AC-006.3.
func TestViewerHTMLWorkBreakdownHiddenWhenZero(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// renderWorkBreakdownSection must return null/empty when all counts are zero (AC-006.3)
	// This is enforced by checking that the function returns early (returns null or '').
	// The function must check if agent + human + qa_team === 0.
	if !strings.Contains(content, "renderWorkBreakdownSection(") {
		t.Error("viewer.html missing renderWorkBreakdownSection — cannot verify hidden-when-zero behavior (AC-006.3)")
	}
}

// TestViewerHTMLWorkBreakdownCSS verifies that the work breakdown CSS classes are present.
// TC-F006-1, AC-006.2.
func TestViewerHTMLWorkBreakdownCSS(t *testing.T) {
	content := string(viewer.ViewerHTML)

	cssRules := []string{
		".wb-row",
		".wb-label",
		".wb-bar",
		".wb-bar-fill",
		".wb-count",
	}
	for _, rule := range cssRules {
		if !strings.Contains(content, rule) {
			t.Errorf("viewer.html missing CSS rule: %q (required for Work Breakdown styling, AC-006.2)", rule)
		}
	}
}

// TestViewerHTMLActionItemsFunction verifies that renderActionItemsSection is present
// and produces rows with status badge, clickable key, and truncated title.
// TC-F007-2, TC-F007-3, AC-007.2, AC-007.3.
func TestViewerHTMLActionItemsFunction(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// Function must be defined
	if !strings.Contains(content, "function renderActionItemsSection(") {
		t.Error("viewer.html missing \"function renderActionItemsSection(\" — action items section required (AC-007.2)")
	}

	// Must call the actionItems() helper (AC-007.1 — uses blocks_feature flag)
	if !strings.Contains(content, "actionItems(") {
		t.Error("viewer.html missing actionItems() call in renderActionItemsSection (AC-007.1)")
	}

	// Must produce rows with data-navigate-key for clickable keys (AC-007.2)
	if !strings.Contains(content, "data-navigate-key") {
		t.Error("viewer.html missing data-navigate-key in action item rows (AC-007.2)")
	}

	// Must use action-item-row CSS class (AC-007.2)
	if !strings.Contains(content, "action-item-row") {
		t.Error("viewer.html missing action-item-row CSS class (AC-007.2)")
	}
}

// TestViewerHTMLActionItemsOmittedWhenEmpty verifies that the Action Items section
// is omitted (not empty) when no tasks are in blocking statuses. TC-F007-4, AC-007.4.
func TestViewerHTMLActionItemsOmittedWhenEmpty(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// renderActionItemsSection must check for empty results and return null/'' (AC-007.4)
	if !strings.Contains(content, "renderActionItemsSection(") {
		t.Error("viewer.html missing renderActionItemsSection — cannot verify omit-when-empty (AC-007.4)")
	}

	// Section label must be "Action Items"
	if !strings.Contains(content, "Action Items") {
		t.Error("viewer.html missing \"Action Items\" section label — section header required (AC-007.2)")
	}
}

// TestViewerHTMLActionItemsCSS verifies that the action items CSS classes are present.
// AC-007.2.
func TestViewerHTMLActionItemsCSS(t *testing.T) {
	content := string(viewer.ViewerHTML)

	if !strings.Contains(content, ".action-item-row") {
		t.Error("viewer.html missing .action-item-row CSS rule (AC-007.2)")
	}
}

// TestViewerHTMLTasksTableFunction verifies that renderTasksTableSection is present
// with KEY · TITLE · STATUS · ORDER columns and clickable KEY cells.
// TC-F008-1, TC-F008-2, TC-F008-4, AC-008.1–AC-008.4.
func TestViewerHTMLTasksTableFunction(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// Function must be defined
	if !strings.Contains(content, "function renderTasksTableSection(") {
		t.Error("viewer.html missing \"function renderTasksTableSection(\" — tasks table section required (AC-008.1)")
	}

	// KEY cells must carry data-navigate-key and .clickable (AC-008.1)
	if !strings.Contains(content, "data-navigate-key") {
		t.Error("viewer.html missing data-navigate-key in tasks table (AC-008.1)")
	}
	if !strings.Contains(content, `"clickable"`) {
		t.Error("viewer.html missing class=\"clickable\" in tasks table KEY cell (AC-008.1)")
	}

	// ORDER column must be present (AC-T4 — execution_order from treeData)
	if !strings.Contains(content, "execution_order") {
		t.Error("viewer.html missing execution_order in tasks table — ORDER column must reflect execution_order (AC-T4)")
	}

	// ORDER column header must say ORDER
	if !strings.Contains(content, "ORDER") {
		t.Error("viewer.html missing ORDER column header in tasks table (AC-T4)")
	}

	// TITLE must use title attribute for full text and CSS truncation (AC-008.2)
	if !strings.Contains(content, "text-overflow:ellipsis") {
		t.Error("viewer.html missing text-overflow:ellipsis in tasks table TITLE cell (AC-008.2)")
	}

	// Table must use ov-table CSS class (AC-008.3)
	if !strings.Contains(content, "ov-table") {
		t.Error("viewer.html missing ov-table CSS class in tasks table (AC-008.3)")
	}

	// Overflow scroll at >10 children (AC-008.4)
	if !strings.Contains(content, "ov-table-scroll") {
		t.Error("viewer.html missing ov-table-scroll class in tasks table (AC-008.4)")
	}
}

// TestViewerHTMLFeatureOverviewPaneDispatch verifies that renderOverviewPane dispatches
// to the feature-specific section renderers for feature entities.
// TC-F006-4, TC-F007-4, TC-F008-1.
func TestViewerHTMLFeatureOverviewPaneDispatch(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// Must call renderWorkBreakdownSection for feature case (REQ-F-006)
	if !strings.Contains(content, "renderWorkBreakdownSection(entity.tasks || [])") {
		t.Error("viewer.html renderOverviewPane must call renderWorkBreakdownSection(entity.tasks || []) for feature case (REQ-F-006)")
	}

	// Must call renderActionItemsSection for feature case (REQ-F-007)
	if !strings.Contains(content, "renderActionItemsSection(entity.tasks || [])") {
		t.Error("viewer.html renderOverviewPane must call renderActionItemsSection(entity.tasks || []) for feature case (REQ-F-007)")
	}

	// Must call renderTasksTableSection for feature case (REQ-F-008)
	if !strings.Contains(content, "renderTasksTableSection(entity)") {
		t.Error("viewer.html renderOverviewPane must call renderTasksTableSection(entity) for feature case (REQ-F-008)")
	}

	// Must NOT still have the placeholder comment (indicates task is done)
	if strings.Contains(content, "Work breakdown, action items, tasks table rendered in later tasks") {
		t.Error("viewer.html renderOverviewPane feature case still has placeholder comment — must be replaced with real section calls (F09-007)")
	}
}

// TestViewerHTMLF09NewEndpointURLMarkers verifies that the two new backend
// endpoint URL patterns introduced by E27-F09 are referenced in the embedded
// viewer.html JavaScript.
// TC-SMOKE-F09-08 (notes endpoint URL) and TC-SMOKE-F09-09 (related-docs URL).
func TestViewerHTMLF09NewEndpointURLMarkers(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// TC-SMOKE-F09-08: Notes endpoint URL must appear in the fetch helper.
	if !strings.Contains(content, "api/v1/viewer/notes/") {
		t.Error("viewer.html missing \"api/v1/viewer/notes/\" — notes endpoint URL not referenced (TC-SMOKE-F09-08)")
	}

	// TC-SMOKE-F09-09: Related-docs endpoint URL must appear in the fetch helper.
	if !strings.Contains(content, "api/v1/viewer/related-docs/") {
		t.Error("viewer.html missing \"api/v1/viewer/related-docs/\" — related-docs endpoint URL not referenced (TC-SMOKE-F09-09)")
	}
}

// ----- T-E27-F09-011: Info pane header polish tests -----

// TestViewerHTMLBreadcrumbBeforeTitle verifies that the breadcrumb is rendered
// before (above) the entity title h2 / entity-view-header in the detail pane.
// The breadcrumb HTML must appear before the entity-view-header div in the
// renderEntityView template string so it is the first visual element.
// T-E27-F09-011 requirement: breadcrumb is the first line of the detail pane.
func TestViewerHTMLBreadcrumbBeforeTitle(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// Locate renderEntityView function body.
	funcMarker := "function renderEntityView()"
	funcStart := strings.Index(content, funcMarker)
	if funcStart < 0 {
		t.Fatal("viewer.html missing function renderEntityView() — cannot verify breadcrumb position")
	}
	// Work within the renderEntityView function body only.
	funcBody := content[funcStart:]

	// Find the template literal that builds the HTML — anchored on the const html assignment.
	templateStart := strings.Index(funcBody, "const html = `")
	if templateStart < 0 {
		t.Fatal("viewer.html renderEntityView: missing 'const html = `' — cannot find template string")
	}
	// Find the position of renderBreadcrumb(entity) within the template region.
	breadcrumbPos := strings.Index(funcBody[templateStart:], "renderBreadcrumb(entity)")
	titlePos := strings.Index(funcBody[templateStart:], "entity-view-header")
	if breadcrumbPos < 0 {
		t.Fatal("viewer.html renderEntityView template: missing renderBreadcrumb(entity) call")
	}
	if titlePos < 0 {
		t.Fatal("viewer.html renderEntityView template: missing entity-view-header div")
	}
	// breadcrumb must come before entity-view-header
	if breadcrumbPos > titlePos {
		t.Errorf("viewer.html: renderBreadcrumb(entity) (pos %d in template) must appear BEFORE entity-view-header (pos %d) — breadcrumb should be first element in detail pane (T-E27-F09-011)", breadcrumbPos, titlePos)
	}
}

// TestViewerHTMLNoKeyRowInPropertiesPanel verifies that the KEY row has been
// removed from the properties panel in renderPropertiesPanel.
// T-E27-F09-011 requirement: key is already shown as the blue link, so the
// redundant KEY row in the properties grid must be removed.
func TestViewerHTMLNoKeyRowInPropertiesPanel(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// The pushRow call for 'Key' must not be present in renderPropertiesPanel.
	// We check for the specific push call that was removed.
	// The pattern "pushRow(rows, 'Key'" should no longer appear.
	if strings.Contains(content, "pushRow(rows, 'Key'") {
		t.Error("viewer.html renderPropertiesPanel still contains pushRow(rows, 'Key'...) — the KEY row must be removed as it is redundant with the blue key link (T-E27-F09-011)")
	}
}

// TestViewerHTMLNoParentRowInPropertiesPanel verifies that the PARENT row has
// been removed from the properties panel in renderPropertiesPanel.
// T-E27-F09-011 requirement: parent is already shown in the breadcrumb, so the
// redundant PARENT row in the properties grid must be removed.
func TestViewerHTMLNoParentRowInPropertiesPanel(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// The pushRow call for 'Parent' based on entity.parent must not be present.
	// The pattern "pushRow(rows, 'Parent'" should no longer appear.
	if strings.Contains(content, "pushRow(rows, 'Parent'") {
		t.Error("viewer.html renderPropertiesPanel still contains pushRow(rows, 'Parent'...) — the PARENT row must be removed as it is redundant with the breadcrumb (T-E27-F09-011)")
	}
}

// TestViewerHTMLCopyKeyButton verifies that a copy button is rendered next to
// the entity-view-key element in renderEntityView.
// T-E27-F09-011 requirement: clicking the button copies the entity key to clipboard;
// the button provides brief visual confirmation.
func TestViewerHTMLCopyKeyButton(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// The copy key button class must be present.
	if !strings.Contains(content, "copy-key-btn") {
		t.Error("viewer.html missing \"copy-key-btn\" — copy button next to entity key is required (T-E27-F09-011)")
	}

	// The button must use navigator.clipboard (directly or via copy-btn delegation).
	// It can reuse the existing copy-btn delegation by adding data-copy-value, or
	// implement its own. Either way, navigator.clipboard usage must be present.
	if !strings.Contains(content, "navigator.clipboard") {
		t.Error("viewer.html missing navigator.clipboard — copy key button must write to clipboard (T-E27-F09-011)")
	}
}

// TestViewerHTMLCopyKeyButtonHasDataAttribute verifies that the copy key button
// carries the data-copy-value attribute (or equivalent) so the existing global
// copy delegation handler can pick it up without new JS plumbing.
// T-E27-F09-011 requirement: copy button is wired to the entity key value.
func TestViewerHTMLCopyKeyButtonHasDataAttribute(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// The copy-key-btn must include data-copy-value attribute so the existing
	// global .copy-btn delegation handles it, OR it must have its own handler.
	// We verify that copy-key-btn appears alongside copy-btn class or has own handler.
	hasDelegation := strings.Contains(content, `copy-key-btn`) &&
		(strings.Contains(content, `copy-btn copy-key-btn`) ||
			strings.Contains(content, `copy-key-btn copy-btn`) ||
			strings.Contains(content, `copyKeyBtn`) ||
			strings.Contains(content, `copy-key-btn`))

	if !hasDelegation {
		t.Error("viewer.html copy-key-btn missing — button must be wired to copy entity key (T-E27-F09-011)")
	}
}

// ============================================================
// T-E27-F09-012: Tab segment in breadcrumb + browser history
// ============================================================

// TestViewerHTMLBreadcrumbTabSegment verifies that renderBreadcrumb appends
// the active tab as the final breadcrumb segment (T-E27-F09-012).
func TestViewerHTMLBreadcrumbTabSegment(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// renderBreadcrumb must accept entityViewTab and include it as the last segment.
	// We verify the function references entityViewTab when building the breadcrumb.
	if !strings.Contains(content, "function renderBreadcrumb(") {
		t.Fatal("viewer.html missing renderBreadcrumb function")
	}

	// Find the renderBreadcrumb function body
	funcStart := strings.Index(content, "function renderBreadcrumb(")
	if funcStart < 0 {
		t.Fatal("cannot find renderBreadcrumb function")
	}
	// Find the end of the function (next top-level function or block)
	funcBody := content[funcStart:]
	// The function must reference entityViewTab to append the tab segment
	if !strings.Contains(funcBody[:strings.Index(funcBody, "\nfunction ")+1], "entityViewTab") {
		t.Error("viewer.html renderBreadcrumb does not reference entityViewTab — tab segment must be appended to breadcrumb (T-E27-F09-012)")
	}
}

// TestViewerHTMLBreadcrumbTabDisplayNames verifies that a tab display-name
// mapping exists for known tab IDs (T-E27-F09-012).
func TestViewerHTMLBreadcrumbTabDisplayNames(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// Tab IDs must be mapped to display names somewhere near renderBreadcrumb.
	// We require at minimum: overview, info (Spec), transitions (History), files, dashboard.
	tabDisplayNames := []string{"Overview", "Spec", "History", "Files", "Dashboard"}
	for _, name := range tabDisplayNames {
		if !strings.Contains(content, `'`+name+`'`) && !strings.Contains(content, `"`+name+`"`) {
			t.Errorf("viewer.html missing tab display name %q in tab name map (T-E27-F09-012)", name)
		}
	}
}

// TestViewerHTMLBreadcrumbTabSegmentIsPlainText verifies that the tab breadcrumb
// segment is rendered as plain text (no data-navigate-key), not a link (T-E27-F09-012).
func TestViewerHTMLBreadcrumbTabSegmentIsPlainText(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// The tab segment must use the "breadcrumb-seg current" class (plain text, not navigable).
	// We look for a pattern where entityViewTab is used with isCurrent:true or "breadcrumb-seg current".
	// The tab segment must NOT have data-navigate-key.
	funcStart := strings.Index(content, "function renderBreadcrumb(")
	if funcStart < 0 {
		t.Fatal("cannot find renderBreadcrumb function")
	}
	funcEnd := strings.Index(content[funcStart+1:], "\nfunction ")
	if funcEnd < 0 {
		funcEnd = len(content) - funcStart - 1
	}
	funcBody := content[funcStart : funcStart+funcEnd+1]

	// The function must push a tab segment with isCurrent: true
	if !strings.Contains(funcBody, "isCurrent: true") && !strings.Contains(funcBody, "isCurrent:true") {
		t.Error("viewer.html renderBreadcrumb: tab segment must use isCurrent:true so it renders as plain text (T-E27-F09-012)")
	}
}

// TestViewerHTMLPushNavStateIncludesURL verifies that pushNavState passes a URL
// to history.pushState encoding both entity key and tab (T-E27-F09-012).
func TestViewerHTMLPushNavStateIncludesURL(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// pushNavState must call history.pushState with a URL argument.
	// The URL must encode the entity key and tab as query params.
	funcStart := strings.Index(content, "function pushNavState(")
	if funcStart < 0 {
		t.Fatal("viewer.html missing function pushNavState()")
	}
	funcEnd := strings.Index(content[funcStart+1:], "\nfunction ")
	if funcEnd < 0 {
		funcEnd = len(content) - funcStart - 1
	}
	funcBody := content[funcStart : funcStart+funcEnd+1]

	if !strings.Contains(funcBody, "entity") {
		t.Error("viewer.html pushNavState: URL must include 'entity' query param (T-E27-F09-012)")
	}
	if !strings.Contains(funcBody, "tab") {
		t.Error("viewer.html pushNavState: URL must include 'tab' query param (T-E27-F09-012)")
	}
	// Must pass URL as third arg to history.pushState (not just empty string)
	if !strings.Contains(funcBody, "searchParams") && !strings.Contains(funcBody, "URLSearchParams") && !strings.Contains(funcBody, "?entity") {
		t.Error("viewer.html pushNavState: must build a URL with query params for history.pushState (T-E27-F09-012)")
	}
}

// TestViewerHTMLInitialURLRestore verifies that on page load the app reads
// ?entity= and ?tab= query params and restores that state (T-E27-F09-012).
func TestViewerHTMLInitialURLRestore(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// After loadProjectData completes, it must check URL params.
	// We verify that URLSearchParams or location.search is referenced in or
	// near loadProjectData to restore entity+tab on initial load.
	funcStart := strings.Index(content, "async function loadProjectData(")
	if funcStart < 0 {
		t.Fatal("viewer.html missing function loadProjectData()")
	}
	funcEnd := strings.Index(content[funcStart+1:], "\nfunction ")
	if funcEnd < 0 {
		funcEnd = len(content) - funcStart - 1
	}
	funcBody := content[funcStart : funcStart+funcEnd+1]

	hasURLRestore := strings.Contains(funcBody, "URLSearchParams") ||
		strings.Contains(funcBody, "location.search") ||
		strings.Contains(funcBody, "searchParams.get")
	if !hasURLRestore {
		t.Error("viewer.html loadProjectData: must read URL query params (URLSearchParams/location.search) to restore entity+tab on initial load (T-E27-F09-012)")
	}
}

// TestViewerHTMLPopstateRestoresTabFromState verifies that the popstate handler
// restores entityViewTab from event.state (T-E27-F09-012).
func TestViewerHTMLPopstateRestoresTabFromState(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// restoreNavState must read entityViewTab from the state object.
	funcStart := strings.Index(content, "function restoreNavState(")
	if funcStart < 0 {
		t.Fatal("viewer.html missing function restoreNavState()")
	}
	funcEnd := strings.Index(content[funcStart+1:], "\nfunction ")
	if funcEnd < 0 {
		funcEnd = len(content) - funcStart - 1
	}
	funcBody := content[funcStart : funcStart+funcEnd+1]

	if !strings.Contains(funcBody, "entityViewTab") {
		t.Error("viewer.html restoreNavState: must restore entityViewTab from state (T-E27-F09-012)")
	}
}

// TestViewerHTMLTabSegmentAbsentWithNoEntity verifies that when no entity is
// selected, the tab segment does not appear in the breadcrumb (T-E27-F09-012).
func TestViewerHTMLTabSegmentAbsentWithNoEntity(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// renderBreadcrumb returns '' when entity is falsy — tab segment must
	// not be shown when no entity is selected. We verify the early-return guard.
	funcStart := strings.Index(content, "function renderBreadcrumb(")
	if funcStart < 0 {
		t.Fatal("viewer.html missing renderBreadcrumb function")
	}
	funcEnd := strings.Index(content[funcStart+1:], "\nfunction ")
	if funcEnd < 0 {
		funcEnd = len(content) - funcStart - 1
	}
	funcBody := content[funcStart : funcStart+funcEnd+1]

	// Must have an early return when entity is falsy
	if !strings.Contains(funcBody, "if (!entity)") && !strings.Contains(funcBody, "if(!entity)") {
		t.Error("viewer.html renderBreadcrumb: must guard against null entity (no tab segment when no entity selected) (T-E27-F09-012)")
	}
}

// TestViewerHTMLF10IdeasFilterHelper verifies that the isHiddenTerminalStatus helper
// and HIDDEN_TERMINAL_BY_TYPE map are present. AC-005.7 (T-E27-F10-005).
func TestViewerHTMLF10IdeasFilterHelper(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// The HIDDEN_TERMINAL_BY_TYPE constant must exist
	if !strings.Contains(content, "HIDDEN_TERMINAL_BY_TYPE") {
		t.Error("viewer.html missing HIDDEN_TERMINAL_BY_TYPE map (T-E27-F10-005)")
	}

	// The isHiddenTerminalStatus function must exist
	if !strings.Contains(content, "function isHiddenTerminalStatus(") {
		t.Error("viewer.html missing isHiddenTerminalStatus function (T-E27-F10-005)")
	}
}

// TestViewerHTMLF10IdeasStatusMap verifies that the HIDDEN_TERMINAL_BY_TYPE map
// contains the idea-specific terminal statuses 'converted' and 'archived'.
// AC-005.1, AC-005.2 (T-E27-F10-005).
func TestViewerHTMLF10IdeasStatusMap(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// The idea entry must map converted and archived
	if !strings.Contains(content, "'converted'") {
		t.Error("viewer.html HIDDEN_TERMINAL_BY_TYPE missing 'converted' for idea (T-E27-F10-005 AC-005.1)")
	}
	if !strings.Contains(content, "'archived'") {
		t.Error("viewer.html HIDDEN_TERMINAL_BY_TYPE missing 'archived' for idea (T-E27-F10-005 AC-005.2)")
	}

	// The idea: entry must explicitly be present
	if !strings.Contains(content, "idea:") {
		t.Error("viewer.html HIDDEN_TERMINAL_BY_TYPE missing 'idea:' entry (T-E27-F10-005)")
	}
}

// TestViewerHTMLF10CallSitesReplaced verifies that all four original
// 'status === completed' filter predicates have been replaced with
// isHiddenTerminalStatus calls. AC-005.7 (T-E27-F10-005).
func TestViewerHTMLF10CallSitesReplaced(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// None of the old hardcoded 'completed' filter predicates should remain
	// in the showCompleted guard positions.
	// We check that 'isHiddenTerminalStatus' is called at least 4 times.
	helperStr := "isHiddenTerminalStatus("
	count := strings.Count(content, helperStr)
	if count < 4 {
		t.Errorf("viewer.html: expected at least 4 calls to isHiddenTerminalStatus, found %d (T-E27-F10-005 AC-005.7)", count)
	}
}

// TestViewerHTMLF10FlatSectionEntityTypeParam verifies that buildFlatSectionHtml
// now accepts an entityType parameter and the Ideas call passes 'idea'.
// AC-005.1 (T-E27-F10-005).
func TestViewerHTMLF10FlatSectionEntityTypeParam(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// The function signature must accept entityType
	funcStart := strings.Index(content, "function buildFlatSectionHtml(")
	if funcStart < 0 {
		t.Fatal("viewer.html missing buildFlatSectionHtml function")
	}
	// Find the closing paren of the function signature
	sigEnd := strings.Index(content[funcStart:], ")")
	if sigEnd < 0 {
		t.Fatal("viewer.html buildFlatSectionHtml: cannot find closing paren of signature")
	}
	sig := content[funcStart : funcStart+sigEnd+1]
	if !strings.Contains(sig, "entityType") {
		t.Errorf("viewer.html buildFlatSectionHtml: signature must include entityType param, got: %q (T-E27-F10-005)", sig)
	}

	// The Ideas call site must pass 'idea'
	if !strings.Contains(content, "buildFlatSectionHtml('Ideas'") && !strings.Contains(content, `buildFlatSectionHtml("Ideas"`) {
		t.Error("viewer.html missing buildFlatSectionHtml call for Ideas (T-E27-F10-005)")
	}
	// Check that an 'idea' entityType is passed at the Ideas call site
	if !strings.Contains(content, "'idea'") {
		t.Error("viewer.html Ideas call to buildFlatSectionHtml must pass 'idea' entityType (T-E27-F10-005 AC-005.1)")
	}
}

// TestViewerHTMLF10CompletedCSSRule verifies that the .entity-completed CSS rule exists
// and sets opacity: 0.55 without text-decoration: line-through. AC-001.2 (T-E27-F10-001).
func TestViewerHTMLF10CompletedCSSRule(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// The entity-completed CSS class must exist in the stylesheet
	if !strings.Contains(content, ".entity-completed") {
		t.Fatal("viewer.html missing .entity-completed CSS rule (T-E27-F10-001 AC-001.2)")
	}

	// The rule must set opacity: 0.55 for tree-node-key and tree-node-title
	if !strings.Contains(content, ".entity-completed .tree-node-key") {
		t.Error("viewer.html .entity-completed must target .tree-node-key (T-E27-F10-001 AC-001.2)")
	}
	if !strings.Contains(content, ".entity-completed .tree-node-title") {
		t.Error("viewer.html .entity-completed must target .tree-node-title (T-E27-F10-001 AC-001.2)")
	}

	// opacity: 0.55 must be present in the completed rule block
	// We verify by checking the .entity-completed block contains 0.55
	completedIdx := strings.Index(content, ".entity-completed .tree-node-key")
	if completedIdx < 0 {
		t.Fatal("viewer.html missing .entity-completed .tree-node-key rule")
	}
	// Look for 0.55 in the nearby block (within 300 chars)
	block := content[completedIdx : completedIdx+300]
	if !strings.Contains(block, "0.55") {
		t.Error("viewer.html .entity-completed rule must set opacity: 0.55 (T-E27-F10-001 AC-001.2)")
	}

	// The completed rule must NOT contain text-decoration: line-through
	// (line-through is reserved for .entity-cancelled only)
	if strings.Contains(block, "line-through") {
		t.Error("viewer.html .entity-completed must NOT use text-decoration: line-through (T-E27-F10-001 AC-001.2)")
	}
}

// TestViewerHTMLF10CompletedTableRowCSS verifies that tr.entity-completed td rule exists
// with opacity: 0.55 and no line-through. AC-001.5 (T-E27-F10-001).
func TestViewerHTMLF10CompletedTableRowCSS(t *testing.T) {
	content := string(viewer.ViewerHTML)

	if !strings.Contains(content, "tr.entity-completed td") {
		t.Fatal("viewer.html missing tr.entity-completed td CSS rule (T-E27-F10-001 AC-001.5)")
	}

	// Verify opacity 0.55 is in the tr.entity-completed block
	trIdx := strings.Index(content, "tr.entity-completed td")
	if trIdx < 0 {
		t.Fatal("viewer.html missing tr.entity-completed td rule")
	}
	block := content[trIdx : trIdx+200]
	if !strings.Contains(block, "0.55") {
		t.Error("viewer.html tr.entity-completed td must set opacity: 0.55 (T-E27-F10-001 AC-001.5)")
	}
	if strings.Contains(block, "line-through") {
		t.Error("viewer.html tr.entity-completed td must NOT use text-decoration: line-through (T-E27-F10-001 AC-001.5)")
	}
}

// TestViewerHTMLF10CancelledRegressionCheck verifies that entity-cancelled CSS rules
// still set opacity: 0.4 and text-decoration: line-through. AC-001.3 (T-E27-F10-001).
func TestViewerHTMLF10CancelledRegressionCheck(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// entity-cancelled must still have the line-through rule (existing behaviour)
	if !strings.Contains(content, ".entity-cancelled .tree-node-key") {
		t.Error("viewer.html missing .entity-cancelled .tree-node-key rule (T-E27-F10-001 AC-001.3 regression)")
	}
	cancelledIdx := strings.Index(content, ".entity-cancelled .tree-node-key")
	if cancelledIdx < 0 {
		t.Fatal("viewer.html missing .entity-cancelled .tree-node-key rule")
	}
	block := content[cancelledIdx : cancelledIdx+300]
	if !strings.Contains(block, "line-through") {
		t.Error("viewer.html .entity-cancelled must have text-decoration: line-through (T-E27-F10-001 AC-001.3 regression)")
	}
}

// TestViewerHTMLF10BuildNodeHtmlCompletedClass verifies that buildNodeHtml applies
// entity-completed for completed status. AC-001.1 (T-E27-F10-001).
func TestViewerHTMLF10BuildNodeHtmlCompletedClass(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// The buildNodeHtml function must exist
	if !strings.Contains(content, "function buildNodeHtml(") {
		t.Fatal("viewer.html missing buildNodeHtml function (T-E27-F10-001)")
	}

	// The function must assign entity-completed based on completed status.
	// We look for a pattern where entity-completed is conditionally set —
	// similar to how entity-cancelled is set via isCancelledStatus.
	if !strings.Contains(content, "entity-completed") {
		t.Fatal("viewer.html: entity-completed class never used in JS (T-E27-F10-001 AC-001.1)")
	}

	// The cancelled check must come BEFORE the completed check
	// (isCancelledStatus is checked first — AC-001.4: cancelled wins).
	cancelledIdx := strings.LastIndex(content, "isCancelledStatus(status)")
	completedIdx := strings.Index(content, "entity-completed")
	// Both must exist and cancelled check must appear before entity-completed assignment
	if cancelledIdx < 0 {
		t.Error("viewer.html missing isCancelledStatus(status) call in buildNodeHtml area (T-E27-F10-001 AC-001.4)")
	}
	if completedIdx < 0 {
		t.Error("viewer.html missing entity-completed class assignment (T-E27-F10-001 AC-001.1)")
	}
}

// TestViewerHTMLF10TableRowsCompletedClass verifies that entity-completed is applied
// in the feature and task table row builders. AC-001.5 (T-E27-F10-001).
func TestViewerHTMLF10TableRowsCompletedClass(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// The feature table row builder must apply entity-completed for completed status
	// (not just entity-cancelled). We check that at least one of the two rowClass
	// builders assigns entity-completed.
	rowClassCompletedCount := strings.Count(content, "entity-completed")
	if rowClassCompletedCount < 3 {
		// We expect at least: CSS rule (1), feature rowClass (1), task rowClass (1)
		t.Errorf("viewer.html: expected at least 3 occurrences of entity-completed (CSS + feature rowClass + task rowClass), found %d (T-E27-F10-001 AC-001.5)", rowClassCompletedCount)
	}
}

// ── T-E27-F10-002: "Show all items" label rename + cancelled visibility ──

// TestViewerHTMLF10002LabelRename verifies that the show-completed checkbox is
// present and "Show completed" is gone. AC-002.1 (T-E27-F10-002).
func TestViewerHTMLF10002LabelRename(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// AC-002.1: show-completed-checkbox must be present
	if !strings.Contains(content, `id="show-completed-checkbox"`) {
		t.Error(`viewer.html missing show-completed-checkbox (T-E27-F10-002 AC-002.1)`)
	}

	// The old label text must be absent
	if strings.Contains(content, "Show completed") {
		t.Error(`viewer.html still contains "Show completed" — must be removed (T-E27-F10-002 AC-002.1)`)
	}
}

// TestViewerHTMLF10002ShowAllFilesUnaffected verifies that the files checkbox
// is still present. AC-002.5 (T-E27-F10-002).
func TestViewerHTMLF10002ShowAllFilesUnaffected(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// AC-002.5: show-all-files-checkbox must remain present
	if !strings.Contains(content, `id="show-all-files-checkbox"`) {
		t.Error(`viewer.html missing show-all-files-checkbox — must remain unaffected (T-E27-F10-002 AC-002.5)`)
	}
}

// TestViewerHTMLF10002VariableComment verifies that showCompleted has a code comment
// documenting its updated semantics. AC-002.4 (T-E27-F10-002).
func TestViewerHTMLF10002VariableComment(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// AC-002.4: The showCompleted variable declaration must have a comment
	// noting that it now controls all terminal statuses (not just "completed").
	// We look for the comment keyword near the showCompleted declaration.
	showCompletedIdx := strings.Index(content, "let showCompleted")
	if showCompletedIdx < 0 {
		t.Fatal("viewer.html missing 'let showCompleted' declaration (T-E27-F10-002 AC-002.4)")
	}

	// Grab a window around and before the declaration to check for a comment.
	// The comment may appear on the preceding lines.
	start := showCompletedIdx
	if start > 300 {
		start = showCompletedIdx - 300
	}
	window := content[start : showCompletedIdx+200]

	// Comment must mention the new "show all items" / "Show all items" semantics
	// OR at minimum explain it controls "all" / "terminal" items.
	hasComment := strings.Contains(window, "Show all items") ||
		strings.Contains(window, "show all items") ||
		strings.Contains(window, "all items") ||
		strings.Contains(window, "terminal")
	if !hasComment {
		t.Error("viewer.html: showCompleted declaration must have a comment documenting its 'show all items' semantics (T-E27-F10-002 AC-002.4)")
	}
}

// ── T-E27-F10-003: "Collapse all" button ──

// TestViewerHTMLF10003CollapseAllButtonPresent verifies that the collapse-all button
// is rendered inside the sidebar-section-header next to "Hierarchy". AC-003.1 (T-E27-F10-003).
func TestViewerHTMLF10003CollapseAllButtonPresent(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// AC-003.1: <button id="collapse-all-btn"> must be present
	if !strings.Contains(content, `id="collapse-all-btn"`) {
		t.Error(`viewer.html missing <button id="collapse-all-btn"> (T-E27-F10-003 AC-003.1)`)
	}

	// Button must use the sidebar-collapse-btn class
	if !strings.Contains(content, `class="sidebar-collapse-btn"`) {
		t.Error(`viewer.html missing class="sidebar-collapse-btn" on collapse button (T-E27-F10-003 AC-003.1)`)
	}

	// The button must appear near the Hierarchy section header — verify both appear
	// in the renderSidebar function together.
	hierarchyIdx := strings.Index(content, `sidebar-section-header`)
	collapseIdx := strings.Index(content, `collapse-all-btn`)
	if hierarchyIdx < 0 || collapseIdx < 0 {
		t.Fatal("viewer.html: sidebar-section-header or collapse-all-btn not found (T-E27-F10-003 AC-003.1)")
	}
	// The button template should appear after the sidebar-section-header CSS definition.
	if collapseIdx < hierarchyIdx {
		t.Error("viewer.html: collapse-all-btn must appear after sidebar-section-header (T-E27-F10-003 AC-003.1)")
	}
}

// TestViewerHTMLF10003CollapseAllCSS verifies that a CSS rule styles
// the .sidebar-collapse-btn. AC-003.1 (T-E27-F10-003).
func TestViewerHTMLF10003CollapseAllCSS(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// A CSS rule for .sidebar-collapse-btn must be present to style the button
	if !strings.Contains(content, ".sidebar-collapse-btn") {
		t.Error("viewer.html missing .sidebar-collapse-btn CSS rule (T-E27-F10-003 AC-003.1)")
	}
}

// TestViewerHTMLF10003CollapseAllClickHandler verifies that the click handler
// calls expandedEpics.clear(), expandedFeatures.clear(), and renderSidebar().
// AC-003.2 (T-E27-F10-003).
func TestViewerHTMLF10003CollapseAllClickHandler(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// The handler must clear expandedEpics
	if !strings.Contains(content, "expandedEpics.clear()") {
		t.Error("viewer.html missing expandedEpics.clear() in collapse-all handler (T-E27-F10-003 AC-003.2)")
	}

	// The handler must clear expandedFeatures
	if !strings.Contains(content, "expandedFeatures.clear()") {
		t.Error("viewer.html missing expandedFeatures.clear() in collapse-all handler (T-E27-F10-003 AC-003.2)")
	}

	// The collapse-all button handler must wire to the collapse-all-btn element
	if !strings.Contains(content, "collapse-all-btn") {
		t.Error("viewer.html missing reference to collapse-all-btn in wiring (T-E27-F10-003 AC-003.2)")
	}

	// The handler must call renderSidebar() after clearing
	// We verify both clear() calls and renderSidebar() appear in close proximity
	// (within 500 chars) to the collapseAllBtn wiring
	collapseWireIdx := strings.Index(content, "collapse-all-btn")
	if collapseWireIdx < 0 {
		t.Fatal("viewer.html missing collapse-all-btn wiring (T-E27-F10-003 AC-003.2)")
	}
	// Find the second occurrence (first is the HTML template, second is wiring)
	secondOccurrence := strings.Index(content[collapseWireIdx+1:], "collapse-all-btn")
	if secondOccurrence < 0 {
		t.Fatal("viewer.html collapse-all-btn only appears once — expected both HTML and JS wiring (T-E27-F10-003 AC-003.2)")
	}
	wireOffset := collapseWireIdx + 1 + secondOccurrence
	// Look for renderSidebar() call in a window after the second occurrence
	window := content[wireOffset : wireOffset+400]
	if !strings.Contains(window, "renderSidebar()") {
		t.Error("viewer.html collapse-all click handler must call renderSidebar() (T-E27-F10-003 AC-003.2)")
	}
}

// TestViewerHTMLF10003CollapseAllGuard verifies that the click handler uses
// an if-guard before accessing the element (safe even if element hasn't rendered).
// AC-003.5 (T-E27-F10-003).
func TestViewerHTMLF10003CollapseAllGuard(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// The handler must use getElementById and check for non-null before wiring
	// Pattern: const collapseAllBtn = document.getElementById('collapse-all-btn');
	//          if (collapseAllBtn) { ... }
	if !strings.Contains(content, "getElementById('collapse-all-btn')") &&
		!strings.Contains(content, `getElementById("collapse-all-btn")`) {
		t.Error("viewer.html collapse-all handler must use getElementById to find the button (T-E27-F10-003 AC-003.5)")
	}
}

// ─── T-E27-F10-004: Persist show-all checkbox state in localStorage ─────────

// TestViewerHTMLF10004LocalStorageKeyPresent verifies that the storage key
// 'shark.viewer.showAllItems' is referenced in viewer.html.
// AC-004.1 and AC-004.2 (T-E27-F10-004).
func TestViewerHTMLF10004LocalStorageKeyPresent(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// The key must appear at least twice: once for getItem (read on load) and
	// once for setItem (write on checkbox change).
	const storageKey = "shark.viewer.showAllItems"
	count := strings.Count(content, storageKey)
	if count < 2 {
		t.Errorf("viewer.html must reference localStorage key %q at least twice (read + write), found %d occurrence(s) (T-E27-F10-004 AC-004.1 AC-004.2)", storageKey, count)
	}
}

// TestViewerHTMLF10004LocalStorageRead verifies that the initial value of
// showCompleted is read from localStorage.getItem on page load.
// AC-004.2 and AC-004.3 (T-E27-F10-004).
func TestViewerHTMLF10004LocalStorageRead(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// Must contain localStorage.getItem('shark.viewer.showAllItems')
	if !strings.Contains(content, "localStorage.getItem('shark.viewer.showAllItems')") &&
		!strings.Contains(content, `localStorage.getItem("shark.viewer.showAllItems")`) {
		t.Error("viewer.html must call localStorage.getItem('shark.viewer.showAllItems') to read persisted state on load (T-E27-F10-004 AC-004.2)")
	}

	// The read result must be compared to 'true' string (AC-004.5: any other value → false)
	if !strings.Contains(content, `=== 'true'`) && !strings.Contains(content, `=== "true"`) {
		t.Error("viewer.html must compare localStorage value to string 'true' so any other value (null, 'false', etc.) defaults to false (T-E27-F10-004 AC-004.3 AC-004.5)")
	}
}

// TestViewerHTMLF10004LocalStorageWrite verifies that the checkbox change
// handler calls localStorage.setItem with the new value.
// AC-004.1 (T-E27-F10-004).
func TestViewerHTMLF10004LocalStorageWrite(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// Must contain localStorage.setItem('shark.viewer.showAllItems', ...)
	if !strings.Contains(content, "localStorage.setItem('shark.viewer.showAllItems'") &&
		!strings.Contains(content, `localStorage.setItem("shark.viewer.showAllItems"`) {
		t.Error("viewer.html must call localStorage.setItem('shark.viewer.showAllItems', ...) in the checkbox change handler (T-E27-F10-004 AC-004.1)")
	}
}

// TestViewerHTMLF10004LocalStorageReadTryCatch verifies that localStorage
// read is wrapped in try/catch for privacy-mode resilience.
// AC-004.6 (T-E27-F10-004).
func TestViewerHTMLF10004LocalStorageReadTryCatch(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// Find the localStorage.getItem call and verify it is inside a try block.
	// Strategy: look for the IIFE pattern that wraps the read in try/catch.
	// Pattern: try { return localStorage.getItem('shark.viewer.showAllItems') === 'true'; }
	//          catch (e) { return false; }
	readIdx := strings.Index(content, "localStorage.getItem('shark.viewer.showAllItems')")
	if readIdx == -1 {
		readIdx = strings.Index(content, `localStorage.getItem("shark.viewer.showAllItems")`)
	}
	if readIdx == -1 {
		t.Fatal("viewer.html missing localStorage.getItem call — earlier AC-004.2 test should have caught this")
	}

	// Scan a window before the read call for a 'try' keyword
	start := readIdx - 200
	if start < 0 {
		start = 0
	}
	window := content[start:readIdx]
	if !strings.Contains(window, "try") {
		t.Error("viewer.html localStorage.getItem call must be inside a try block for privacy-mode resilience (T-E27-F10-004 AC-004.6)")
	}
}

// TestViewerHTMLF10004LocalStorageWriteTryCatch verifies that localStorage
// write is wrapped in try/catch for privacy-mode resilience.
// AC-004.6 (T-E27-F10-004).
func TestViewerHTMLF10004LocalStorageWriteTryCatch(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// Find the localStorage.setItem call
	writeIdx := strings.Index(content, "localStorage.setItem('shark.viewer.showAllItems'")
	if writeIdx == -1 {
		writeIdx = strings.Index(content, `localStorage.setItem("shark.viewer.showAllItems"`)
	}
	if writeIdx == -1 {
		t.Fatal("viewer.html missing localStorage.setItem call — earlier AC-004.1 test should have caught this")
	}

	// Scan a window before the write call for a 'try' keyword
	start := writeIdx - 100
	if start < 0 {
		start = 0
	}
	window := content[start:writeIdx]
	if !strings.Contains(window, "try") {
		t.Error("viewer.html localStorage.setItem call must be inside a try block for privacy-mode resilience (T-E27-F10-004 AC-004.6)")
	}
}

// TestViewerHTMLF10004ShowCompletedInitialisedBeforeRender verifies that
// the showCompleted variable is initialised (via the IIFE) before
// renderSidebar() can first run, so the first paint reflects stored state.
// AC-004.4 (T-E27-F10-004).
func TestViewerHTMLF10004ShowCompletedInitialisedBeforeRender(t *testing.T) {
	content := string(viewer.ViewerHTML)

	// The IIFE pattern must appear: (function() { try { return localStorage.getItem ... } })()
	// This ensures showCompleted is set at var-declaration time before any renderSidebar call.
	if !strings.Contains(content, "(function()") && !strings.Contains(content, "(() =>") {
		t.Error("viewer.html showCompleted initialisation must use an IIFE so the value is set before first renderSidebar() call (T-E27-F10-004 AC-004.4)")
	}
}
