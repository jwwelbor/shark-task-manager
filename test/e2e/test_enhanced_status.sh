#!/bin/bash
set -e

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

assert_jq_field() {
    local json="$1"
    local field="$2"
    local test_name="$3"

    if echo "$json" | jq -e "$field" > /dev/null 2>&1; then
        log_success "$test_name: field '$field' present"
        return 0
    else
        log_error "$test_name: field '$field' NOT FOUND"
        return 1
    fi
}

assert_jq_array_field() {
    local json="$1"
    local field="$2"
    local test_name="$3"

    if echo "$json" | jq -e "$field | type == \"array\"" > /dev/null 2>&1; then
        log_success "$test_name: field '$field' is array"
        return 0
    else
        log_error "$test_name: field '$field' is NOT an array"
        return 1
    fi
}

# Test 1: Feature Get Command - Verify Enhanced Fields
test_feature_get() {
    log_info "Testing: feature get command with enhanced fields"

    output=$($SHARK_BIN feature get E07-F23 --json 2>&1)

    if [ $? -ne 0 ]; then
        log_error "feature get E07-F23 failed"
        return 1
    fi

    # Verify core fields
    assert_jq_field "$output" '.id' "feature_get_core"
    assert_jq_field "$output" '.key' "feature_get_core"
    assert_jq_field "$output" '.title' "feature_get_core"
    assert_jq_field "$output" '.epic_id' "feature_get_core"

    # Verify progress fields
    assert_jq_field "$output" '.progress_pct' "feature_get_progress"
    assert_jq_field "$output" '.progress' "feature_get_progress"

    # Verify enhanced status tracking fields
    assert_jq_field "$output" '.status_breakdown' "feature_get_status_tracking"
    assert_jq_array_field "$output" '.status_breakdown' "feature_get_status_tracking"

    # Verify work summary fields
    assert_jq_field "$output" '.work_summary' "feature_get_work_summary"
    assert_jq_field "$output" '.work_summary.TotalTasks' "feature_get_work_summary"
    assert_jq_field "$output" '.work_summary.CompletedTasks' "feature_get_work_summary"
    assert_jq_field "$output" '.work_summary.AgentWork' "feature_get_work_summary"
    assert_jq_field "$output" '.work_summary.HumanWork' "feature_get_work_summary"
    assert_jq_field "$output" '.work_summary.BlockedWork' "feature_get_work_summary"

    # Verify action items fields
    assert_jq_field "$output" '.action_items' "feature_get_action_items"
    assert_jq_field "$output" '.action_items.AwaitingApproval' "feature_get_action_items"
    assert_jq_field "$output" '.action_items.Blocked' "feature_get_action_items"
    assert_jq_field "$output" '.action_items.InProgress' "feature_get_action_items"

    # Verify tasks are included
    assert_jq_array_field "$output" '.tasks' "feature_get_tasks"

    log_success "feature get command test completed"
}

# Test 2: Feature List Command - Verify Enhanced Fields
test_feature_list() {
    log_info "Testing: feature list command with enhanced fields"

    output=$($SHARK_BIN feature list E07 --json 2>&1)

    if [ $? -ne 0 ]; then
        log_error "feature list E07 failed"
        return 1
    fi

    # Verify results array
    assert_jq_array_field "$output" '.results' "feature_list_results"

    # Verify array has items
    count=$(echo "$output" | jq '.results | length')
    if [ "$count" -gt 0 ]; then
        log_success "feature list: results contain $count features"
    else
        log_error "feature list: results are empty"
        return 1
    fi

    # Get first feature result for detailed checks
    first_feature=$(echo "$output" | jq '.results[0]')

    # Verify health field (enhanced)
    assert_jq_field "$first_feature" '.health' "feature_list_health"

    # Verify progress field
    assert_jq_field "$first_feature" '.progress' "feature_list_progress"

    # Verify progress structure (WeightedPct, WeightedRatio, etc.)
    if echo "$first_feature" | jq -e '.progress.WeightedPct' > /dev/null 2>&1; then
        log_success "feature list: progress contains WeightedPct"
    else
        log_error "feature list: progress missing WeightedPct"
    fi

    # Verify notes field (summary of blocked/ready)
    assert_jq_field "$first_feature" '.notes' "feature_list_notes"

    # Verify task count field
    assert_jq_field "$first_feature" '.task_count' "feature_list_task_count"

    # Verify status_override field
    assert_jq_field "$first_feature" '.status_override' "feature_list_status_override"

    log_success "feature list command test completed"
}

# Test 3: Epic Get Command - Verify Enhanced Fields and Rollups
test_epic_get() {
    log_info "Testing: epic get command with rollup and impediments"

    output=$($SHARK_BIN epic get E07 --json 2>&1)

    if [ $? -ne 0 ]; then
        log_error "epic get E07 failed"
        return 1
    fi

    # Verify core fields
    assert_jq_field "$output" '.id' "epic_get_core"
    assert_jq_field "$output" '.key' "epic_get_core"
    assert_jq_field "$output" '.title' "epic_get_core"

    # Verify feature rollup (enhanced)
    assert_jq_field "$output" '.feature_status_rollup' "epic_get_rollup"
    if echo "$output" | jq -e '.feature_status_rollup | type == "object"' > /dev/null 2>&1; then
        log_success "epic get: feature_status_rollup is object"
    else
        log_error "epic get: feature_status_rollup is NOT an object"
    fi

    # Verify task rollup (enhanced)
    assert_jq_field "$output" '.task_status_rollup' "epic_get_rollup"
    if echo "$output" | jq -e '.task_status_rollup | type == "object"' > /dev/null 2>&1; then
        log_success "epic get: task_status_rollup is object"
    else
        log_error "epic get: task_status_rollup is NOT an object"
    fi

    # Verify impediments list (enhanced - blocked tasks with context)
    assert_jq_field "$output" '.impediments' "epic_get_impediments"
    assert_jq_array_field "$output" '.impediments' "epic_get_impediments"

    # If impediments exist, verify structure
    impediment_count=$(echo "$output" | jq '.impediments | length')
    if [ "$impediment_count" -gt 0 ]; then
        first_impediment=$(echo "$output" | jq '.impediments[0]')
        assert_jq_field "$first_impediment" '.task_key' "epic_get_impediment_structure"
        assert_jq_field "$first_impediment" '.title' "epic_get_impediment_structure"
        assert_jq_field "$first_impediment" '.blocked_since' "epic_get_impediment_structure"
        assert_jq_field "$first_impediment" '.reason' "epic_get_impediment_structure"
    else
        log_success "epic get: impediments is empty (no blocked tasks)"
    fi

    # Verify approval backlog count
    assert_jq_field "$output" '.approval_backlog_count' "epic_get_approval_backlog"

    # Verify features array
    assert_jq_array_field "$output" '.features' "epic_get_features"

    # Verify progress
    assert_jq_field "$output" '.progress_pct' "epic_get_progress"

    log_success "epic get command test completed"
}

# Test 4: Feature Get JSON Output Structure
test_feature_get_json_structure() {
    log_info "Testing: feature get JSON output structure validation"

    output=$($SHARK_BIN feature get E07-F23 --json 2>&1)

    if [ $? -ne 0 ]; then
        log_error "feature get E07-F23 failed for structure test"
        return 1
    fi

    # Validate JSON syntax
    if ! echo "$output" | jq empty 2>/dev/null; then
        log_error "feature get: JSON output is invalid"
        return 1
    fi
    log_success "feature get: JSON output is valid"

    # Verify all expected top-level fields exist
    expected_fields=(
        "id"
        "epic_id"
        "key"
        "title"
        "status"
        "progress_pct"
        "tasks"
        "status_breakdown"
        "progress"
        "work_summary"
        "action_items"
    )

    for field in "${expected_fields[@]}"; do
        if echo "$output" | jq -e ".$field" > /dev/null 2>&1; then
            log_success "feature get: top-level field '$field' present"
        else
            log_error "feature get: top-level field '$field' MISSING"
        fi
    done
}

# Test 5: Feature List JSON Output Structure
test_feature_list_json_structure() {
    log_info "Testing: feature list JSON output structure validation"

    output=$($SHARK_BIN feature list E07 --json 2>&1)

    if [ $? -ne 0 ]; then
        log_error "feature list E07 failed for structure test"
        return 1
    fi

    # Validate JSON syntax
    if ! echo "$output" | jq empty 2>/dev/null; then
        log_error "feature list: JSON output is invalid"
        return 1
    fi
    log_success "feature list: JSON output is valid"

    # Verify wrapper structure
    assert_jq_field "$output" '.results' "feature_list_wrapper"
    assert_jq_field "$output" '.count' "feature_list_wrapper"

    # Check first item structure if present
    first=$(echo "$output" | jq '.results[0]' 2>/dev/null)
    if [ -n "$first" ] && [ "$first" != "null" ]; then
        expected_fields=(
            "key"
            "title"
            "epic_id"
            "status"
            "health"
            "progress"
            "notes"
            "task_count"
        )

        for field in "${expected_fields[@]}"; do
            if echo "$first" | jq -e ".$field" > /dev/null 2>&1; then
                log_success "feature list item: field '$field' present"
            else
                log_error "feature list item: field '$field' MISSING"
            fi
        done
    fi
}

# Test 6: Epic Get JSON Output Structure
test_epic_get_json_structure() {
    log_info "Testing: epic get JSON output structure validation"

    output=$($SHARK_BIN epic get E07 --json 2>&1)

    if [ $? -ne 0 ]; then
        log_error "epic get E07 failed for structure test"
        return 1
    fi

    # Validate JSON syntax
    if ! echo "$output" | jq empty 2>/dev/null; then
        log_error "epic get: JSON output is invalid"
        return 1
    fi
    log_success "epic get: JSON output is valid"

    # Verify all expected top-level fields exist
    expected_fields=(
        "id"
        "key"
        "title"
        "status"
        "progress_pct"
        "features"
        "feature_status_rollup"
        "task_status_rollup"
        "impediments"
        "approval_backlog_count"
    )

    for field in "${expected_fields[@]}"; do
        if echo "$output" | jq -e ".$field" > /dev/null 2>&1; then
            log_success "epic get: top-level field '$field' present"
        else
            log_error "epic get: top-level field '$field' MISSING"
        fi
    done
}

# Test 7: Verify Numeric Values
test_numeric_values() {
    log_info "Testing: numeric fields are valid"

    output=$($SHARK_BIN feature get E07-F23 --json 2>&1)

    # Check that progress_pct is a number
    progress=$(echo "$output" | jq '.progress_pct' 2>/dev/null)
    if [[ $progress =~ ^[0-9]+(\.[0-9]+)?$ ]] || [ "$progress" = "null" ]; then
        log_success "feature get: progress_pct is numeric ($progress)"
    else
        log_error "feature get: progress_pct is not numeric ($progress)"
    fi

    # Check work_summary counts
    total_tasks=$(echo "$output" | jq '.work_summary.TotalTasks' 2>/dev/null)
    if [[ $total_tasks =~ ^[0-9]+$ ]]; then
        log_success "feature get: work_summary.TotalTasks is numeric ($total_tasks)"
    else
        log_error "feature get: work_summary.TotalTasks is not numeric"
    fi
}

# Test 8: Verify Field Types
test_field_types() {
    log_info "Testing: field types are correct"

    output=$($SHARK_BIN epic get E07 --json 2>&1)

    # Verify field types
    rollup=$(echo "$output" | jq '.feature_status_rollup | type' 2>/dev/null)
    if [ "$rollup" = '"object"' ]; then
        log_success "epic get: feature_status_rollup is object"
    else
        log_error "epic get: feature_status_rollup type is $rollup (expected object)"
    fi

    impediments=$(echo "$output" | jq '.impediments | type' 2>/dev/null)
    if [ "$impediments" = '"array"' ]; then
        log_success "epic get: impediments is array"
    else
        log_error "epic get: impediments type is $impediments (expected array)"
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

    test_feature_get_json_structure
    echo ""

    test_feature_list_json_structure
    echo ""

    test_epic_get_json_structure
    echo ""

    test_numeric_values
    echo ""

    test_field_types
    echo ""

    # Print summary
    echo "═══════════════════════════════════════════════════════"
    echo "Test Results Summary"
    echo "═══════════════════════════════════════════════════════"
    echo -e "${GREEN}Passed: $PASSED_TESTS${NC}"
    echo -e "${RED}Failed: $FAILED_TESTS${NC}"
    echo "═══════════════════════════════════════════════════════"

    if [ $FAILED_TESTS -eq 0 ]; then
        echo -e "${GREEN}All tests passed!${NC}"
        exit 0
    else
        echo -e "${RED}Some tests failed!${NC}"
        exit 1
    fi
}

# Run main
main
