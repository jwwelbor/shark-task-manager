#!/bin/bash
# Automated Homebrew Installation Test Script
# Tests complete Homebrew tap installation and functionality

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
TAP_NAME="jwwelbor/shark"
FORMULA_NAME="shark"
EXPECTED_VERSION="${1:-v0.1.0-beta}"
TEST_RESULTS_FILE="homebrew-test-results.txt"

# Start timing
START_TIME=$(date +%s)

echo "========================================"
echo "Homebrew Installation Test"
echo "========================================"
echo "Expected Version: $EXPECTED_VERSION"
echo "Timestamp: $(date)"
echo ""

# Cleanup function
cleanup() {
    echo ""
    echo "Cleaning up test installation..."
    brew uninstall $FORMULA_NAME 2>/dev/null || true
    brew untap $TAP_NAME 2>/dev/null || true
    echo "Cleanup complete"
}

# Set trap for cleanup on exit
trap cleanup EXIT

# Test results tracking
PASSED=0
FAILED=0

# Test function
test_step() {
    local description="$1"
    local command="$2"

    echo -n "Testing: $description... "

    if eval "$command" > /dev/null 2>&1; then
        echo -e "${GREEN}PASS${NC}"
        PASSED=$((PASSED + 1))
        return 0
    else
        echo -e "${RED}FAIL${NC}"
        FAILED=$((FAILED + 1))
        return 1
    fi
}

# Timed test function
timed_test() {
    local description="$1"
    local command="$2"
    local max_seconds="$3"

    echo -n "Testing: $description (max ${max_seconds}s)... "

    local cmd_start=$(date +%s)
    if eval "$command" > /dev/null 2>&1; then
        local cmd_end=$(date +%s)
        local duration=$((cmd_end - cmd_start))

        if [ $duration -le $max_seconds ]; then
            echo -e "${GREEN}PASS${NC} (${duration}s)"
            PASSED=$((PASSED + 1))
            return 0
        else
            echo -e "${YELLOW}PASS${NC} but slow (${duration}s > ${max_seconds}s target)"
            PASSED=$((PASSED + 1))
            return 0
        fi
    else
        local cmd_end=$(date +%s)
        local duration=$((cmd_end - cmd_start))
        echo -e "${RED}FAIL${NC} (${duration}s)"
        FAILED=$((FAILED + 1))
        return 1
    fi
}

echo "========================================"
echo "Phase 1: Tap Addition"
echo "========================================"

# Add tap
echo "Adding tap: $TAP_NAME"
TAP_START=$(date +%s)
if brew tap $TAP_NAME; then
    TAP_END=$(date +%s)
    TAP_DURATION=$((TAP_END - TAP_START))
    echo -e "${GREEN}SUCCESS${NC} - Tap added in ${TAP_DURATION}s"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}FAILED${NC} - Could not add tap"
    FAILED=$((FAILED + 1))
    exit 1
fi

echo ""
echo "========================================"
echo "Phase 2: Installation"
echo "========================================"

# Install with timing (should be < 30 seconds)
echo "Installing $FORMULA_NAME..."
INSTALL_START=$(date +%s)
if brew install $FORMULA_NAME; then
    INSTALL_END=$(date +%s)
    INSTALL_DURATION=$((INSTALL_END - INSTALL_START))

    if [ $INSTALL_DURATION -le 30 ]; then
        echo -e "${GREEN}SUCCESS${NC} - Installed in ${INSTALL_DURATION}s (target: <30s)"
    else
        echo -e "${YELLOW}SUCCESS${NC} - Installed in ${INSTALL_DURATION}s (slower than 30s target)"
    fi
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}FAILED${NC} - Installation failed"
    FAILED=$((FAILED + 1))
    exit 1
fi

echo ""
echo "========================================"
echo "Phase 3: Installation Verification"
echo "========================================"

# Verify binary exists
test_step "Binary exists in PATH" "which $FORMULA_NAME"

# Verify version
test_step "Version command works" "$FORMULA_NAME --version"

# Check actual version
ACTUAL_VERSION=$($FORMULA_NAME --version 2>&1 | grep -oE 'v[0-9]+\.[0-9]+\.[0-9]+(-[a-z]+)?' || echo "unknown")
echo "Installed version: $ACTUAL_VERSION"

if [ "$ACTUAL_VERSION" = "$EXPECTED_VERSION" ]; then
    echo -e "${GREEN}Version matches expected${NC}"
    PASSED=$((PASSED + 1))
else
    echo -e "${YELLOW}WARNING: Version mismatch${NC}"
    echo "  Expected: $EXPECTED_VERSION"
    echo "  Actual: $ACTUAL_VERSION"
fi

# Verify installation location
BINARY_PATH=$(which $FORMULA_NAME)
echo "Binary location: $BINARY_PATH"

if [[ "$BINARY_PATH" == *"/homebrew/"* ]] || [[ "$BINARY_PATH" == *"/local/bin/"* ]]; then
    echo -e "${GREEN}Binary in expected Homebrew location${NC}"
    PASSED=$((PASSED + 1))
else
    echo -e "${YELLOW}WARNING: Binary in unexpected location${NC}"
fi

echo ""
echo "========================================"
echo "Phase 4: Functional Tests"
echo "========================================"

# Test help command
test_step "Help command works" "$FORMULA_NAME --help"

# Test epic list command
test_step "Epic list command works" "$FORMULA_NAME epic list"

# Test task list command
test_step "Task list command works" "$FORMULA_NAME task list"

# Test database access
test_step "Database access works" "$FORMULA_NAME task list --status created"

echo ""
echo "========================================"
echo "Phase 5: Binary Analysis"
echo "========================================"

# Check binary size
BINARY_SIZE=$(stat -f%z "$BINARY_PATH" 2>/dev/null || stat -c%s "$BINARY_PATH" 2>/dev/null)
BINARY_SIZE_MB=$((BINARY_SIZE / 1024 / 1024))

echo "Binary size: ${BINARY_SIZE_MB} MB"

if [ $BINARY_SIZE_MB -le 12 ]; then
    echo -e "${GREEN}Binary size within target (<12 MB)${NC}"
    PASSED=$((PASSED + 1))
else
    echo -e "${YELLOW}WARNING: Binary larger than target (${BINARY_SIZE_MB} MB > 12 MB)${NC}"
fi

# Check binary type
echo "Binary info:"
file "$BINARY_PATH"

echo ""
echo "========================================"
echo "Test Summary"
echo "========================================"

END_TIME=$(date +%s)
TOTAL_DURATION=$((END_TIME - START_TIME))

echo "Total tests: $((PASSED + FAILED))"
echo -e "Passed: ${GREEN}$PASSED${NC}"
echo -e "Failed: ${RED}$FAILED${NC}"
echo "Total duration: ${TOTAL_DURATION}s"
echo ""
echo "Detailed timings:"
echo "  Tap addition: ${TAP_DURATION}s"
echo "  Installation: ${INSTALL_DURATION}s"
echo ""

# Save results to file
cat > "$TEST_RESULTS_FILE" << EOF
Homebrew Installation Test Results
====================================
Timestamp: $(date)
Expected Version: $EXPECTED_VERSION
Actual Version: $ACTUAL_VERSION

Performance Metrics:
- Tap Addition: ${TAP_DURATION}s
- Installation Time: ${INSTALL_DURATION}s (target: <30s)
- Binary Size: ${BINARY_SIZE_MB} MB (target: <12 MB)
- Total Test Duration: ${TOTAL_DURATION}s

Test Results:
- Tests Passed: $PASSED
- Tests Failed: $FAILED
- Success Rate: $(( (PASSED * 100) / (PASSED + FAILED) ))%

Binary Location: $BINARY_PATH
EOF

echo "Results saved to: $TEST_RESULTS_FILE"

# Exit with appropriate code
if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed${NC}"
    exit 1
fi
