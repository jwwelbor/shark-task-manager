#!/bin/bash
# Automated Manual Download and Checksum Verification Test Script
# Tests complete manual binary download and verification process

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
VERSION="${1:-v0.1.0-beta}"
GITHUB_REPO="jwwelbor/shark-task-manager"
BASE_URL="https://github.com/$GITHUB_REPO/releases/download/$VERSION"
TEST_DIR="/tmp/shark-manual-test-$(date +%s)"
TEST_RESULTS_FILE="manual-test-results.txt"

# Detect platform
detect_platform() {
    local os=$(uname -s | tr '[:upper:]' '[:lower:]')
    local arch=$(uname -m)

    case "$os" in
        linux)
            OS="linux"
            EXT="tar.gz"
            ;;
        darwin)
            OS="darwin"
            EXT="tar.gz"
            ;;
        msys*|mingw*|cygwin*|windows_nt)
            OS="windows"
            EXT="zip"
            ;;
        *)
            echo -e "${RED}ERROR: Unsupported OS: $os${NC}"
            exit 1
            ;;
    esac

    case "$arch" in
        x86_64|amd64)
            ARCH="amd64"
            ;;
        aarch64|arm64)
            ARCH="arm64"
            ;;
        *)
            echo -e "${RED}ERROR: Unsupported architecture: $arch${NC}"
            exit 1
            ;;
    esac
}

# Start timing
START_TIME=$(date +%s)

echo "========================================"
echo "Manual Download & Verification Test"
echo "========================================"
echo "Version: $VERSION"
echo "Repository: $GITHUB_REPO"
echo "Timestamp: $(date)"
echo ""

# Detect platform
detect_platform
echo "Detected platform: $OS $ARCH"
echo "Archive extension: .$EXT"
echo ""

# Test results tracking
PASSED=0
FAILED=0
DOWNLOAD_SIZES=()

# Create test directory
mkdir -p "$TEST_DIR"
cd "$TEST_DIR"
echo "Test directory: $TEST_DIR"
echo ""

# Cleanup function
cleanup() {
    echo ""
    echo "Cleaning up test files..."
    cd /
    rm -rf "$TEST_DIR"
    echo "Cleanup complete"
}

# Set trap for cleanup on exit
trap cleanup EXIT

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

echo "========================================"
echo "Phase 1: Download Assets"
echo "========================================"

# Construct filenames
ARCHIVE_NAME="shark_${VERSION#v}_${OS}_${ARCH}.${EXT}"
CHECKSUMS_NAME="checksums.txt"

ARCHIVE_URL="$BASE_URL/$ARCHIVE_NAME"
CHECKSUMS_URL="$BASE_URL/$CHECKSUMS_NAME"

echo "Downloading archive: $ARCHIVE_NAME"
DOWNLOAD_START=$(date +%s)

if curl -L -o "$ARCHIVE_NAME" "$ARCHIVE_URL" 2>/dev/null; then
    DOWNLOAD_END=$(date +%s)
    DOWNLOAD_DURATION=$((DOWNLOAD_END - DOWNLOAD_START))
    ARCHIVE_SIZE=$(stat -c%s "$ARCHIVE_NAME" 2>/dev/null || stat -f%z "$ARCHIVE_NAME")
    ARCHIVE_SIZE_MB=$((ARCHIVE_SIZE / 1024 / 1024))

    echo -e "${GREEN}SUCCESS${NC} - Downloaded in ${DOWNLOAD_DURATION}s (${ARCHIVE_SIZE_MB} MB)"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}FAILED${NC} - Could not download archive"
    echo "URL: $ARCHIVE_URL"
    FAILED=$((FAILED + 1))
    exit 1
fi

echo ""
echo "Downloading checksums: $CHECKSUMS_NAME"
CHECKSUM_DOWNLOAD_START=$(date +%s)

if curl -L -o "$CHECKSUMS_NAME" "$CHECKSUMS_URL" 2>/dev/null; then
    CHECKSUM_DOWNLOAD_END=$(date +%s)
    CHECKSUM_DOWNLOAD_DURATION=$((CHECKSUM_DOWNLOAD_END - CHECKSUM_DOWNLOAD_START))
    echo -e "${GREEN}SUCCESS${NC} - Downloaded in ${CHECKSUM_DOWNLOAD_DURATION}s"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}FAILED${NC} - Could not download checksums"
    echo "URL: $CHECKSUMS_URL"
    FAILED=$((FAILED + 1))
    exit 1
fi

echo ""
echo "========================================"
echo "Phase 2: Checksum Verification"
echo "========================================"

# Verify archive is listed in checksums
echo "Checking if archive is in checksums.txt..."
if grep -q "$ARCHIVE_NAME" "$CHECKSUMS_NAME"; then
    echo -e "${GREEN}Archive found in checksums.txt${NC}"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}FAILED${NC} - Archive not found in checksums.txt"
    FAILED=$((FAILED + 1))
    exit 1
fi

# Verify checksum
echo "Verifying SHA256 checksum..."
CHECKSUM_START=$(date +%s)

if sha256sum -c "$CHECKSUMS_NAME" --ignore-missing --status 2>/dev/null; then
    CHECKSUM_END=$(date +%s)
    CHECKSUM_DURATION=$((CHECKSUM_END - CHECKSUM_START))
    echo -e "${GREEN}SUCCESS${NC} - Checksum verified in ${CHECKSUM_DURATION}s"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}FAILED${NC} - Checksum verification failed"
    echo ""
    echo "Expected checksum:"
    grep "$ARCHIVE_NAME" "$CHECKSUMS_NAME"
    echo ""
    echo "Actual checksum:"
    sha256sum "$ARCHIVE_NAME"
    FAILED=$((FAILED + 1))
    exit 1
fi

echo ""
echo "========================================"
echo "Phase 3: Extraction"
echo "========================================"

echo "Extracting archive..."
EXTRACT_START=$(date +%s)

if [ "$EXT" = "tar.gz" ]; then
    if tar -xzf "$ARCHIVE_NAME"; then
        EXTRACT_END=$(date +%s)
        EXTRACT_DURATION=$((EXTRACT_END - EXTRACT_START))
        echo -e "${GREEN}SUCCESS${NC} - Extracted in ${EXTRACT_DURATION}s"
        PASSED=$((PASSED + 1))
    else
        echo -e "${RED}FAILED${NC} - Extraction failed"
        FAILED=$((FAILED + 1))
        exit 1
    fi
elif [ "$EXT" = "zip" ]; then
    if unzip -q "$ARCHIVE_NAME"; then
        EXTRACT_END=$(date +%s)
        EXTRACT_DURATION=$((EXTRACT_END - EXTRACT_START))
        echo -e "${GREEN}SUCCESS${NC} - Extracted in ${EXTRACT_DURATION}s"
        PASSED=$((PASSED + 1))
    else
        echo -e "${RED}FAILED${NC} - Extraction failed"
        FAILED=$((FAILED + 1))
        exit 1
    fi
fi

echo ""
echo "Extracted files:"
ls -lh

# Find the binary
if [ -f "./shark" ]; then
    BINARY_PATH="./shark"
elif [ -f "./shark.exe" ]; then
    BINARY_PATH="./shark.exe"
else
    echo -e "${RED}FAILED${NC} - Binary not found after extraction"
    FAILED=$((FAILED + 1))
    exit 1
fi

echo "Binary found: $BINARY_PATH"
PASSED=$((PASSED + 1))

# Make binary executable (Unix-like systems)
if [ "$OS" != "windows" ]; then
    chmod +x "$BINARY_PATH"
fi

echo ""
echo "========================================"
echo "Phase 4: Binary Verification"
echo "========================================"

# Check binary size
BINARY_SIZE=$(stat -c%s "$BINARY_PATH" 2>/dev/null || stat -f%z "$BINARY_PATH")
BINARY_SIZE_MB=$((BINARY_SIZE / 1024 / 1024))

echo "Binary size: ${BINARY_SIZE_MB} MB"

if [ $BINARY_SIZE_MB -le 12 ]; then
    echo -e "${GREEN}Binary size within target (<12 MB)${NC}"
    PASSED=$((PASSED + 1))
else
    echo -e "${YELLOW}WARNING: Binary larger than target (${BINARY_SIZE_MB} MB > 12 MB)${NC}"
fi

# Check if binary is executable
test_step "Binary is executable" "test -x $BINARY_PATH || [ '$OS' = 'windows' ]"

echo ""
echo "========================================"
echo "Phase 5: Functional Tests"
echo "========================================"

# Test version command
echo -n "Testing: Version command... "
if VERSION_OUTPUT=$("$BINARY_PATH" --version 2>&1); then
    echo -e "${GREEN}PASS${NC}"
    echo "  Output: $VERSION_OUTPUT"
    PASSED=$((PASSED + 1))

    # Verify version matches
    if echo "$VERSION_OUTPUT" | grep -q "${VERSION}"; then
        echo -e "  ${GREEN}Version matches expected${NC}"
        PASSED=$((PASSED + 1))
    else
        echo -e "  ${YELLOW}WARNING: Version mismatch${NC}"
        echo "    Expected: $VERSION"
    fi
else
    echo -e "${RED}FAIL${NC}"
    FAILED=$((FAILED + 1))
fi

# Test help command
test_step "Help command works" "$BINARY_PATH --help"

# Test basic command
test_step "Epic list command works" "$BINARY_PATH epic list"

echo ""
echo "========================================"
echo "Phase 6: Archive Analysis"
echo "========================================"

echo "Archive size: ${ARCHIVE_SIZE_MB} MB"

if [ $ARCHIVE_SIZE_MB -le 12 ]; then
    echo -e "${GREEN}Archive size within target (<12 MB)${NC}"
    PASSED=$((PASSED + 1))
else
    echo -e "${YELLOW}WARNING: Archive larger than target (${ARCHIVE_SIZE_MB} MB > 12 MB)${NC}"
fi

# Calculate compression ratio
COMPRESSION_RATIO=$(echo "scale=2; $BINARY_SIZE / $ARCHIVE_SIZE" | bc)
echo "Compression ratio: ${COMPRESSION_RATIO}x"

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
echo "  Archive download: ${DOWNLOAD_DURATION}s"
echo "  Checksums download: ${CHECKSUM_DOWNLOAD_DURATION}s"
echo "  Checksum verification: ${CHECKSUM_DURATION}s"
echo "  Extraction: ${EXTRACT_DURATION}s"
echo ""
echo "Size information:"
echo "  Archive: ${ARCHIVE_SIZE_MB} MB"
echo "  Binary: ${BINARY_SIZE_MB} MB"
echo "  Compression ratio: ${COMPRESSION_RATIO}x"
echo ""

# Save results to file (in original directory)
RESULTS_PATH="/tmp/$TEST_RESULTS_FILE"
cat > "$RESULTS_PATH" << EOF
Manual Download & Verification Test Results
============================================
Timestamp: $(date)
Version: $VERSION
Platform: $OS $ARCH

Performance Metrics:
- Archive Download: ${DOWNLOAD_DURATION}s
- Checksums Download: ${CHECKSUM_DOWNLOAD_DURATION}s
- Checksum Verification: ${CHECKSUM_DURATION}s
- Extraction: ${EXTRACT_DURATION}s
- Total Duration: ${TOTAL_DURATION}s

Size Metrics:
- Archive Size: ${ARCHIVE_SIZE_MB} MB (target: <12 MB)
- Binary Size: ${BINARY_SIZE_MB} MB (target: <12 MB)
- Compression Ratio: ${COMPRESSION_RATIO}x

Test Results:
- Tests Passed: $PASSED
- Tests Failed: $FAILED
- Success Rate: $(( (PASSED * 100) / (PASSED + FAILED) ))%

Download URLs:
- Archive: $ARCHIVE_URL
- Checksums: $CHECKSUMS_URL
EOF

echo "Results saved to: $RESULTS_PATH"

# Exit with appropriate code
if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed${NC}"
    exit 1
fi
