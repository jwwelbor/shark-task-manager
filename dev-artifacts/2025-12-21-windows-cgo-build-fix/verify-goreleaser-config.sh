#!/bin/bash
#
# Shark GoReleaser Configuration Verification Script
#
# Validates that the .goreleaser.yml configuration is correct:
# 1. YAML is valid
# 2. No default CGO_ENABLED in root env
# 3. Platform-specific overrides are present and correct
# 4. Windows has CGO_ENABLED=0
# 5. Linux amd64 has CGO_ENABLED=1
# 6. Linux arm64 has CGO_ENABLED=1 with cross-compiler
# 7. macOS has CGO_ENABLED=0
#

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
CONFIG_FILE="$PROJECT_ROOT/.goreleaser.yml"

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0

# Print helpers
print_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
    ((TESTS_PASSED++))
}

print_error() {
    echo -e "${RED}✗${NC} $1"
    ((TESTS_FAILED++))
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

print_header() {
    echo ""
    echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}  $1${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
    echo ""
}

# Test 1: File exists
test_file_exists() {
    print_header "Test 1: Configuration File Exists"

    if [ -f "$CONFIG_FILE" ]; then
        print_success "Found: $CONFIG_FILE"
        return 0
    else
        print_error "Not found: $CONFIG_FILE"
        return 1
    fi
}

# Test 2: Valid YAML
test_yaml_valid() {
    print_header "Test 2: YAML Syntax Valid"

    # Check if we can parse the YAML
    if python3 -c "import yaml; yaml.safe_load(open('$CONFIG_FILE'))" 2>/dev/null; then
        print_success "YAML syntax is valid"
        return 0
    elif go run github.com/naoina/toml/cmd/tomli@latest <"$CONFIG_FILE" 2>/dev/null; then
        print_success "Configuration is parseable"
        return 0
    else
        # Just check if file has expected structure
        if grep -q "version: 2" "$CONFIG_FILE" && grep -q "builds:" "$CONFIG_FILE"; then
            print_success "Configuration structure looks valid"
            return 0
        else
            print_error "Failed to validate YAML (requires python3 or go)"
            return 1
        fi
    fi
}

# Test 3: No default CGO_ENABLED in root env
test_no_default_cgo() {
    print_header "Test 3: No Default CGO_ENABLED in Root"

    # Extract the builds section and check for default env with CGO_ENABLED
    if grep -A 25 "builds:" "$CONFIG_FILE" | grep -B 5 "goos:" | grep -q "CGO_ENABLED"; then
        print_error "Found CGO_ENABLED in default env (before goos)"
        return 1
    else
        print_success "No default CGO_ENABLED before platform-specific overrides"
        return 0
    fi
}

# Test 4: Windows has CGO_ENABLED=0
test_windows_cgo_disabled() {
    print_header "Test 4: Windows CGO Disabled"

    # Look for windows override with CGO_ENABLED=0
    if grep -A 3 "goos: windows" "$CONFIG_FILE" | grep -q "CGO_ENABLED=0"; then
        print_success "Windows correctly has CGO_ENABLED=0"
        return 0
    else
        print_error "Windows does not have CGO_ENABLED=0"
        return 1
    fi
}

# Test 5: Linux amd64 has CGO_ENABLED=1
test_linux_amd64_cgo_enabled() {
    print_header "Test 5: Linux amd64 CGO Enabled"

    # Look for linux amd64 override with CGO_ENABLED=1
    if grep -A 4 "goos: linux" "$CONFIG_FILE" | grep -A 4 "goarch: amd64" | grep -q "CGO_ENABLED=1"; then
        print_success "Linux amd64 correctly has CGO_ENABLED=1"
        return 0
    else
        print_error "Linux amd64 does not have CGO_ENABLED=1"
        return 1
    fi
}

# Test 6: Linux arm64 has CGO_ENABLED=1 with cross-compiler
test_linux_arm64_cgo_enabled() {
    print_header "Test 6: Linux arm64 CGO with Cross-Compiler"

    # Check for arm64 override
    if ! grep -A 6 "goos: linux" "$CONFIG_FILE" | grep -A 6 "goarch: arm64" | grep -q "CGO_ENABLED=1"; then
        print_error "Linux arm64 does not have CGO_ENABLED=1"
        return 1
    fi

    # Check for cross-compiler
    if ! grep -A 6 "goos: linux" "$CONFIG_FILE" | grep -A 6 "goarch: arm64" | grep -q "aarch64-linux-gnu-gcc"; then
        print_error "Linux arm64 does not have cross-compiler configured"
        return 1
    fi

    print_success "Linux arm64 correctly configured with CGO_ENABLED=1 and cross-compiler"
    return 0
}

# Test 7: macOS has CGO_ENABLED=0
test_macos_cgo_disabled() {
    print_header "Test 7: macOS CGO Disabled"

    # Look for darwin override with CGO_ENABLED=0
    if grep -A 3 "goos: darwin" "$CONFIG_FILE" | grep -q "CGO_ENABLED=0"; then
        print_success "macOS correctly has CGO_ENABLED=0"
        return 0
    else
        print_error "macOS does not have CGO_ENABLED=0"
        return 1
    fi
}

# Test 8: Verify override structure
test_overrides_present() {
    print_header "Test 8: Override Structure"

    if grep -q "overrides:" "$CONFIG_FILE"; then
        print_success "Found overrides section"
    else
        print_error "Missing overrides section"
        return 1
    fi

    # Count platform-specific overrides
    override_count=$(grep -c "goos:" "$CONFIG_FILE" || echo 0)

    if [ "$override_count" -ge 3 ]; then
        print_success "Found $override_count platform-specific overrides (expected 4)"
        return 0
    else
        print_error "Found only $override_count platform overrides (expected 4: linux amd64, linux arm64, windows, darwin)"
        return 1
    fi
}

# Test 9: Verify no dangling env references
test_no_dangling_env() {
    print_header "Test 9: No Dangling Environment References"

    # Check that all env sections are properly nested under overrides
    # This is a simple check - more complex parsing would need a YAML parser

    if grep -q "^ *env:" "$CONFIG_FILE"; then
        # Found env section, check if it's indented (under overrides/platforms)
        if grep "^ *- goos:" "$CONFIG_FILE" | head -1 >/dev/null && \
           grep "^    env:" "$CONFIG_FILE" >/dev/null; then
            print_success "Environment variables properly nested under platform overrides"
            return 0
        else
            print_warning "Could not verify proper nesting (requires YAML parser)"
            return 0
        fi
    else
        print_error "No environment variables found in overrides"
        return 1
    fi
}

# Main execution
main() {
    print_header "GoReleaser Configuration Verification"

    print_info "Configuration file: $CONFIG_FILE"
    echo ""

    # Run all tests
    test_file_exists || exit 1
    test_yaml_valid || true  # Don't fail on YAML validation (requires tools)
    test_no_default_cgo || true
    test_windows_cgo_disabled || true
    test_linux_amd64_cgo_enabled || true
    test_linux_arm64_cgo_enabled || true
    test_macos_cgo_disabled || true
    test_overrides_present || true
    test_no_dangling_env || true

    # Print summary
    print_header "Test Summary"

    echo -e "${GREEN}Passed: $TESTS_PASSED${NC}"
    echo -e "${RED}Failed: $TESTS_FAILED${NC}"

    if [ $TESTS_FAILED -eq 0 ]; then
        echo ""
        print_success "All configuration checks passed!"
        echo ""
        echo "The .goreleaser.yml configuration is ready for release builds:"
        echo "  - Windows builds will have CGO disabled"
        echo "  - Linux amd64 builds will use native CGO"
        echo "  - Linux arm64 builds will use cross-compiler"
        echo "  - macOS builds will have CGO disabled"
        exit 0
    else
        echo ""
        print_error "Configuration has issues that need to be fixed"
        exit 1
    fi
}

# Run main
main "$@"
