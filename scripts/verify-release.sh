#!/bin/bash
#
# Shark Release Verification Script (Linux/macOS)
#
# This script downloads, verifies, and validates a Shark CLI release.
# It performs integrity checks and basic functionality tests.
#
# Usage:
#   ./verify-release.sh <version> [platform] [arch]
#
# Examples:
#   ./verify-release.sh v1.0.0                    # Auto-detect platform and arch
#   ./verify-release.sh v1.0.0 linux amd64        # Explicit platform
#   ./verify-release.sh v1.0.0 darwin arm64       # macOS Apple Silicon
#
# Exit Codes:
#   0 - Success (all checks passed)
#   1 - Invalid arguments or usage
#   2 - Download failed
#   3 - Checksum verification failed
#   4 - Binary validation failed
#

set -e  # Exit on error
set -u  # Exit on undefined variable

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
REPO="jwwelbor/shark-task-manager"
GITHUB_URL="https://github.com/${REPO}"
TEMP_DIR=""

# Print colored message
print_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

print_header() {
    echo ""
    echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}  $1${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
    echo ""
}

# Cleanup on exit
cleanup() {
    if [ -n "$TEMP_DIR" ] && [ -d "$TEMP_DIR" ]; then
        print_info "Cleaning up temporary files..."
        rm -rf "$TEMP_DIR"
    fi
}
trap cleanup EXIT

# Show usage information
usage() {
    cat << EOF
Shark Release Verification Script

Usage:
  $0 <version> [platform] [arch]

Arguments:
  version    Release version (e.g., v1.0.0)
  platform   Target platform: linux, darwin (auto-detected if omitted)
  arch       Target architecture: amd64, arm64 (auto-detected if omitted)

Examples:
  $0 v1.0.0                    # Auto-detect platform and arch
  $0 v1.0.0 linux amd64        # Verify Linux AMD64 build
  $0 v1.0.0 darwin arm64       # Verify macOS ARM64 build

Supported Platforms:
  linux   - Linux (AMD64, ARM64)
  darwin  - macOS (AMD64, ARM64)

This script will:
  1. Download the release binary and checksums
  2. Verify SHA256 checksum integrity
  3. Extract and test the binary
  4. Validate basic functionality (--version, --help)
  5. Report success or failure

Exit Codes:
  0 - All checks passed
  1 - Invalid arguments
  2 - Download failed
  3 - Checksum verification failed
  4 - Binary validation failed

EOF
    exit 1
}

# Detect platform
detect_platform() {
    case "$(uname -s)" in
        Linux*)     echo "linux" ;;
        Darwin*)    echo "darwin" ;;
        *)
            print_error "Unsupported platform: $(uname -s)"
            print_info "Supported: Linux, macOS (Darwin)"
            exit 1
            ;;
    esac
}

# Detect architecture
detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64)   echo "amd64" ;;
        aarch64|arm64)  echo "arm64" ;;
        *)
            print_error "Unsupported architecture: $(uname -m)"
            print_info "Supported: amd64 (x86_64), arm64 (aarch64)"
            exit 1
            ;;
    esac
}

# Check required commands
check_dependencies() {
    local missing=()

    if ! command -v curl &> /dev/null && ! command -v wget &> /dev/null; then
        missing+=("curl or wget")
    fi

    if ! command -v sha256sum &> /dev/null && ! command -v shasum &> /dev/null; then
        missing+=("sha256sum or shasum")
    fi

    if ! command -v tar &> /dev/null; then
        missing+=("tar")
    fi

    if [ ${#missing[@]} -gt 0 ]; then
        print_error "Missing required commands: ${missing[*]}"
        print_info "Please install the missing tools and try again."
        exit 1
    fi
}

# Download file with fallback
download_file() {
    local url="$1"
    local output="$2"

    if command -v curl &> /dev/null; then
        curl -fsSL -o "$output" "$url"
    elif command -v wget &> /dev/null; then
        wget -q -O "$output" "$url"
    else
        print_error "No download tool available (curl or wget)"
        return 1
    fi
}

# Calculate SHA256 with fallback
calculate_sha256() {
    local file="$1"

    if command -v sha256sum &> /dev/null; then
        sha256sum "$file" | awk '{print $1}'
    elif command -v shasum &> /dev/null; then
        shasum -a 256 "$file" | awk '{print $1}'
    else
        print_error "No SHA256 tool available (sha256sum or shasum)"
        return 1
    fi
}

# Main verification function
verify_release() {
    local version="$1"
    local platform="$2"
    local arch="$3"

    print_header "Shark Release Verification: $version"

    print_info "Target: $platform/$arch"
    print_info "Repository: $REPO"
    echo ""

    # Create temporary directory
    TEMP_DIR=$(mktemp -d)
    print_info "Working directory: $TEMP_DIR"
    cd "$TEMP_DIR"

    # Construct filename
    local version_number="${version#v}"  # Remove 'v' prefix
    local filename="shark_${version_number}_${platform}_${arch}.tar.gz"
    local download_url="${GITHUB_URL}/releases/download/${version}/${filename}"
    local checksums_url="${GITHUB_URL}/releases/download/${version}/checksums.txt"

    # Step 1: Download binary
    print_header "Step 1: Downloading Release Assets"

    print_info "Downloading: $filename"
    if ! download_file "$download_url" "$filename"; then
        print_error "Failed to download binary"
        print_info "URL: $download_url"
        print_warning "Check if the release exists: ${GITHUB_URL}/releases/tag/${version}"
        exit 2
    fi
    print_success "Binary downloaded: $filename"

    print_info "Downloading: checksums.txt"
    if ! download_file "$checksums_url" "checksums.txt"; then
        print_error "Failed to download checksums"
        print_info "URL: $checksums_url"
        exit 2
    fi
    print_success "Checksums downloaded"

    # Step 2: Verify checksum
    print_header "Step 2: Verifying SHA256 Checksum"

    # Extract expected checksum from checksums.txt
    local expected_checksum=$(grep "$filename" checksums.txt | awk '{print $1}')

    if [ -z "$expected_checksum" ]; then
        print_error "Checksum not found in checksums.txt for: $filename"
        print_info "Contents of checksums.txt:"
        cat checksums.txt
        exit 3
    fi

    print_info "Expected checksum: $expected_checksum"

    # Calculate actual checksum
    local actual_checksum=$(calculate_sha256 "$filename")
    print_info "Actual checksum:   $actual_checksum"

    # Compare checksums
    if [ "$expected_checksum" = "$actual_checksum" ]; then
        print_success "Checksum verification PASSED"
        print_info "The binary is authentic and has not been tampered with."
    else
        print_error "Checksum verification FAILED"
        print_error "Expected: $expected_checksum"
        print_error "Actual:   $actual_checksum"
        print_warning "DO NOT use this binary!"
        exit 3
    fi

    # Step 3: Extract binary
    print_header "Step 3: Extracting Binary"

    print_info "Extracting: $filename"
    if ! tar -xzf "$filename"; then
        print_error "Failed to extract archive"
        exit 4
    fi
    print_success "Binary extracted"

    # Verify binary exists
    if [ ! -f "shark" ]; then
        print_error "Binary 'shark' not found in archive"
        print_info "Archive contents:"
        tar -tzf "$filename"
        exit 4
    fi

    # Make binary executable
    chmod +x shark
    print_success "Binary is executable"

    # Step 4: Validate binary
    print_header "Step 4: Validating Binary Functionality"

    # Test --version
    print_info "Running: ./shark --version"
    if ! version_output=$(./shark --version 2>&1); then
        print_error "Failed to run --version command"
        print_info "Output: $version_output"
        exit 4
    fi
    print_success "Version command executed successfully"
    print_info "Output: $version_output"

    # Verify version matches expected
    if echo "$version_output" | grep -q "${version#v}"; then
        print_success "Version matches release: ${version#v}"
    else
        print_warning "Version string doesn't match expected version"
        print_info "Expected to find: ${version#v}"
        print_info "Actual output: $version_output"
    fi

    # Test --help
    print_info "Running: ./shark --help"
    if ! help_output=$(./shark --help 2>&1); then
        print_error "Failed to run --help command"
        exit 4
    fi
    print_success "Help command executed successfully"

    # Verify help output contains expected content
    if echo "$help_output" | grep -q "shark"; then
        print_success "Help output looks valid"
    else
        print_warning "Help output may be incomplete"
    fi

    # Step 5: Summary
    print_header "Verification Summary"

    print_success "All checks passed!"
    echo ""
    print_info "Release Information:"
    echo "  Version:    $version"
    echo "  Platform:   $platform"
    echo "  Arch:       $arch"
    echo "  Filename:   $filename"
    echo "  Checksum:   $expected_checksum"
    echo ""
    print_info "The binary is verified and ready for use."
    print_info "Binary location: $TEMP_DIR/shark"
    echo ""
    print_info "To install system-wide, run:"
    echo "  sudo mv $TEMP_DIR/shark /usr/local/bin/"
    echo ""
}

# Parse arguments
main() {
    # Check minimum arguments
    if [ $# -lt 1 ]; then
        print_error "Missing required argument: version"
        echo ""
        usage
    fi

    # Check for help flag
    if [ "$1" = "-h" ] || [ "$1" = "--help" ]; then
        usage
    fi

    # Parse version
    local version="$1"
    if [[ ! "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-.*)?$ ]]; then
        print_error "Invalid version format: $version"
        print_info "Expected format: vX.Y.Z (e.g., v1.0.0)"
        exit 1
    fi

    # Parse or detect platform
    local platform="${2:-}"
    if [ -z "$platform" ]; then
        platform=$(detect_platform)
        print_info "Auto-detected platform: $platform"
    fi

    # Parse or detect architecture
    local arch="${3:-}"
    if [ -z "$arch" ]; then
        arch=$(detect_arch)
        print_info "Auto-detected architecture: $arch"
    fi

    # Validate platform
    if [[ ! "$platform" =~ ^(linux|darwin)$ ]]; then
        print_error "Invalid platform: $platform"
        print_info "Supported platforms: linux, darwin"
        exit 1
    fi

    # Validate architecture
    if [[ ! "$arch" =~ ^(amd64|arm64)$ ]]; then
        print_error "Invalid architecture: $arch"
        print_info "Supported architectures: amd64, arm64"
        exit 1
    fi

    # Check dependencies
    check_dependencies

    # Run verification
    verify_release "$version" "$platform" "$arch"
}

# Run main function
main "$@"
