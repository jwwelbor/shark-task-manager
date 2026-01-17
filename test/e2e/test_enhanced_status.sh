#!/bin/bash

# E2E Tests for Enhanced Status Tracking Commands
# Tests: feature get, feature list, epic get commands
# Verifies all new fields are present in JSON output

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SHARK_BIN="$PROJECT_ROOT/bin/shark"
FAILED_TESTS=0
PASSED_TESTS=0

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Helper functions
log_success() {
    echo -e "${GREEN}✓${NC} $1"
    ((PASSED_TESTS++))
}

log_error() {
    echo -e "${RED}✗${NC} $1"
    ((FAILED_TESTS++))
}

log_info() {
    echo -e "${YELLOW}ℹ${NC} $1"
}

# Test 1: Feature Get Command - Verify Enhanced Fields
test_feature_get() {
    log_info "Testing: feature get command with enhanced fields"

    output=$($SHARK_BIN feature get E07-F23 --json 2>&1)

    # Verify core fields
    if echo "$output" | jq -e '.id' > /dev/null 2>&1; then
        log_success "feature get: .id present"
    else
        log_error "feature get: .id MISSING"
    fi

    if echo "$output" | jq -e '.key' > /dev/null 2>&1; then
        log_success "feature get: .key present"
    else
        log_error "feature get: .key MISSING"
    fi

    if echo "$output" | jq -e '.title' > /dev/null 2>&1; then
        log_success "feature get: .title present"
    else
        log_error "feature get: .title MISSING"
    fi

    # Verify enhanced fields
    if echo "$output" | jq -e '.status_breakdown' > /dev/null 2>&1; then
        log_success "feature get: .status_breakdown present"
    else
        log_error "feature get: .status_breakdown MISSING"
    fi

    if echo "$output" | jq -e '.work_summary' > /dev/null 2>&1; then
        log_success "feature get: .work_summary present"
    else
        log_error "feature get: .work_summary MISSING"
    fi

    if echo "$output" | jq -e '.action_items' > /dev/null 2>&1; then
        log_success "feature get: .action_items present"
    else
        log_error "feature get: .action_items MISSING"
    fi

    if echo "$output" | jq -e '.progress' > /dev/null 2>&1; then
        log_success "feature get: .progress present"
    else
        log_error "feature get: .progress MISSING"
    fi
}

# Test 2: Feature List Command - Verify Enhanced Fields
test_feature_list() {
    log_info "Testing: feature list command with enhanced fields"

    output=$($SHARK_BIN feature list E07 --json 2>&1)

    # Verify results structure
    echo "$output" | jq -e '.results | type == "array"' > /dev/null && log_success "feature list: .results is array" || log_error "feature list: .results is NOT array"
    echo "$output" | jq -e '.count' > /dev/null && log_success "feature list: .count present" || log_error "feature list: .count MISSING"

    # Verify first item has enhanced fields
    first=$(echo "$output" | jq '.results[0]' 2>/dev/null)
    if [ -n "$first" ] && [ "$first" != "null" ]; then
        echo "$first" | jq -e '.health' > /dev/null && log_success "feature list item: .health present" || log_error "feature list item: .health MISSING"
        echo "$first" | jq -e '.progress' > /dev/null && log_success "feature list item: .progress present" || log_error "feature list item: .progress MISSING"
        echo "$first" | jq -e '.notes' > /dev/null && log_success "feature list item: .notes present" || log_error "feature list item: .notes MISSING"
        echo "$first" | jq -e '.task_count' > /dev/null && log_success "feature list item: .task_count present" || log_error "feature list item: .task_count MISSING"
    fi
}

# Test 3: Epic Get Command - Verify Enhanced Fields
test_epic_get() {
    log_info "Testing: epic get command with enhanced fields"

    output=$($SHARK_BIN epic get E07 --json 2>&1)

    # Verify core fields
    echo "$output" | jq -e '.id' > /dev/null && log_success "epic get: .id present" || log_error "epic get: .id MISSING"
    echo "$output" | jq -e '.key' > /dev/null && log_success "epic get: .key present" || log_error "epic get: .key MISSING"

    # Verify enhanced fields
    echo "$output" | jq -e '.feature_status_rollup | type == "object"' > /dev/null && log_success "epic get: .feature_status_rollup is object" || log_error "epic get: .feature_status_rollup is NOT object"
    echo "$output" | jq -e '.task_status_rollup | type == "object"' > /dev/null && log_success "epic get: .task_status_rollup is object" || log_error "epic get: .task_status_rollup is NOT object"
    echo "$output" | jq -e '.impediments | type == "array"' > /dev/null && log_success "epic get: .impediments is array" || log_error "epic get: .impediments is NOT array"
    echo "$output" | jq -e '.approval_backlog_count' > /dev/null && log_success "epic get: .approval_backlog_count present" || log_error "epic get: .approval_backlog_count MISSING"
}

# Test 4: Verify JSON Validity
test_json_validity() {
    log_info "Testing: JSON output validity"

    output1=$($SHARK_BIN feature get E07-F23 --json 2>&1)
    echo "$output1" | jq empty > /dev/null && log_success "feature get: JSON is valid" || log_error "feature get: JSON is INVALID"

    output2=$($SHARK_BIN feature list E07 --json 2>&1)
    echo "$output2" | jq empty > /dev/null && log_success "feature list: JSON is valid" || log_error "feature list: JSON is INVALID"

    output3=$($SHARK_BIN epic get E07 --json 2>&1)
    echo "$output3" | jq empty > /dev/null && log_success "epic get: JSON is valid" || log_error "epic get: JSON is INVALID"
}

# Test 5: Verify Work Summary Fields
test_work_summary() {
    log_info "Testing: work_summary field structure"

    output=$($SHARK_BIN feature get E07-F23 --json 2>&1)

    # Verify all work summary fields
    echo "$output" | jq -e '.work_summary.TotalTasks' > /dev/null && log_success "work_summary: TotalTasks present" || log_error "work_summary: TotalTasks MISSING"
    echo "$output" | jq -e '.work_summary.CompletedTasks' > /dev/null && log_success "work_summary: CompletedTasks present" || log_error "work_summary: CompletedTasks MISSING"
    echo "$output" | jq -e '.work_summary.AgentWork' > /dev/null && log_success "work_summary: AgentWork present" || log_error "work_summary: AgentWork MISSING"
    echo "$output" | jq -e '.work_summary.HumanWork' > /dev/null && log_success "work_summary: HumanWork present" || log_error "work_summary: HumanWork MISSING"
    echo "$output" | jq -e '.work_summary.BlockedWork' > /dev/null && log_success "work_summary: BlockedWork present" || log_error "work_summary: BlockedWork MISSING"
}

# Test 6: Verify Action Items Fields
test_action_items() {
    log_info "Testing: action_items field structure"

    output=$($SHARK_BIN feature get E07-F23 --json 2>&1)

    # Verify all action items fields
    echo "$output" | jq -e '.action_items.AwaitingApproval' > /dev/null && log_success "action_items: AwaitingApproval present" || log_error "action_items: AwaitingApproval MISSING"
    echo "$output" | jq -e '.action_items.Blocked' > /dev/null && log_success "action_items: Blocked present" || log_error "action_items: Blocked MISSING"
    echo "$output" | jq -e '.action_items.InProgress' > /dev/null && log_success "action_items: InProgress present" || log_error "action_items: InProgress MISSING"
}

# Test 7: Verify Progress Fields
test_progress_info() {
    log_info "Testing: progress field structure"

    output=$($SHARK_BIN feature get E07-F23 --json 2>&1)

    # Verify all progress fields
    echo "$output" | jq -e '.progress.WeightedPct' > /dev/null && log_success "progress: WeightedPct present" || log_error "progress: WeightedPct MISSING"
    echo "$output" | jq -e '.progress.CompletionPct' > /dev/null && log_success "progress: CompletionPct present" || log_error "progress: CompletionPct MISSING"
    echo "$output" | jq -e '.progress.WeightedRatio' > /dev/null && log_success "progress: WeightedRatio present" || log_error "progress: WeightedRatio MISSING"
    echo "$output" | jq -e '.progress.CompletionRatio' > /dev/null && log_success "progress: CompletionRatio present" || log_error "progress: CompletionRatio MISSING"
    echo "$output" | jq -e '.progress.TotalTasks' > /dev/null && log_success "progress: TotalTasks present" || log_error "progress: TotalTasks MISSING"
}

# Test 8: Verify Status Breakdown
test_status_breakdown() {
    log_info "Testing: status_breakdown field structure"

    output=$($SHARK_BIN feature get E07-F23 --json 2>&1)

    # Verify status breakdown is an array
    echo "$output" | jq -e '.status_breakdown | type == "array"' > /dev/null && log_success "status_breakdown: is array" || log_error "status_breakdown: is NOT array"

    # Verify first item has required fields
    first=$(echo "$output" | jq '.status_breakdown[0]' 2>/dev/null)
    if [ -n "$first" ] && [ "$first" != "null" ]; then
        echo "$first" | jq -e '.status' > /dev/null && log_success "status_breakdown item: .status present" || log_error "status_breakdown item: .status MISSING"
        echo "$first" | jq -e '.count' > /dev/null && log_success "status_breakdown item: .count present" || log_error "status_breakdown item: .count MISSING"
    fi
}

# Test 9: Verify Epic Rollups
test_epic_rollups() {
    log_info "Testing: epic rollup structures"

    output=$($SHARK_BIN epic get E07 --json 2>&1)

    # Verify feature rollup is object with status keys
    feature_rollup=$(echo "$output" | jq '.feature_status_rollup' 2>/dev/null)
    if [ -n "$feature_rollup" ]; then
        log_success "epic: feature_status_rollup is present"
    else
        log_error "epic: feature_status_rollup is MISSING"
    fi

    # Verify task rollup is object with status keys
    task_rollup=$(echo "$output" | jq '.task_status_rollup' 2>/dev/null)
    if [ -n "$task_rollup" ]; then
        log_success "epic: task_status_rollup is present"
    else
        log_error "epic: task_status_rollup is MISSING"
    fi
}

# Main test execution
main() {
    echo "═══════════════════════════════════════════════════════"
    echo "E2E Tests: Enhanced Status Tracking Commands"
    echo "═══════════════════════════════════════════════════════"
    echo ""

    # Verify shark binary exists
    if [ ! -f "$SHARK_BIN" ]; then
        echo -e "${RED}Error: shark binary not found at $SHARK_BIN${NC}"
        echo "Run 'make shark' to build it"
        exit 1
    fi

    log_info "Using shark binary: $SHARK_BIN"
    log_info "Project root: $PROJECT_ROOT"
    echo ""

    # Run all tests
    test_feature_get
    echo ""

    test_feature_list
    echo ""

    test_epic_get
    echo ""

    test_json_validity
    echo ""

    test_work_summary
    echo ""

    test_action_items
    echo ""

    test_progress_info
    echo ""

    test_status_breakdown
    echo ""

    test_epic_rollups
    echo ""

    # Print summary
    echo "═══════════════════════════════════════════════════════"
    echo "Test Results Summary"
    echo "═══════════════════════════════════════════════════════"
    echo -e "${GREEN}Passed: $PASSED_TESTS${NC}"
    echo -e "${RED}Failed: $FAILED_TESTS${NC}"
    echo "═══════════════════════════════════════════════════════"

    if [ $FAILED_TESTS -eq 0 ]; then
        echo -e "${GREEN}✓ All tests passed!${NC}"
        exit 0
    else
        echo -e "${RED}✗ Some tests failed!${NC}"
        exit 1
    fi
}

# Run main
main
