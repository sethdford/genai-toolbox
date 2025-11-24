#!/bin/bash
# E2E Release Testing Script
# Tests the complete release pipeline and installation methods

set -e

REPO="sethdford/genai-toolbox-enterprise"
VERSION="0.21.2"
RELEASE_URL="https://github.com/${REPO}/releases/tag/v${VERSION}"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

success() {
    echo -e "${GREEN}✓${NC} $1"
}

error() {
    echo -e "${RED}✗${NC} $1"
}

info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

echo "========================================="
echo "🚀 E2E Release Testing"
echo "========================================="
echo ""
echo "Repository: ${REPO}"
echo "Version: v${VERSION}"
echo "Release URL: ${RELEASE_URL}"
echo ""

# Test 1: Check if release exists
echo "Test 1: Checking if release exists..."
if curl -s -o /dev/null -w "%{http_code}" "https://api.github.com/repos/${REPO}/releases/tags/v${VERSION}" | grep -q "200"; then
    success "Release v${VERSION} exists"

    # Get release info
    RELEASE_JSON=$(curl -s "https://api.github.com/repos/${REPO}/releases/tags/v${VERSION}")
    RELEASE_NAME=$(echo "$RELEASE_JSON" | python3 -c "import sys, json; print(json.load(sys.stdin).get('name', 'N/A'))" 2>/dev/null || echo "N/A")
    info "Release name: $RELEASE_NAME"
else
    warning "Release not found yet (may still be building)"
    info "Check status at: https://github.com/${REPO}/actions"
    echo ""
    echo "GitHub Actions workflow may take 5-10 minutes to complete."
    echo "Re-run this script after the workflow completes."
    exit 0
fi
echo ""

# Test 2: Check for release assets
echo "Test 2: Checking release assets..."
ASSETS=$(echo "$RELEASE_JSON" | python3 -c "import sys, json; assets = json.load(sys.stdin).get('assets', []); print('\n'.join([a['name'] for a in assets]))" 2>/dev/null)

if [ -n "$ASSETS" ]; then
    success "Release has assets:"
    echo "$ASSETS" | while read asset; do
        info "  - $asset"
    done

    # Check for required assets
    required_assets=(
        "genai-toolbox-linux-amd64.tar.gz"
        "genai-toolbox-linux-arm64.tar.gz"
        "genai-toolbox-darwin-amd64.tar.gz"
        "genai-toolbox-darwin-arm64.tar.gz"
        "genai-toolbox-windows-amd64.zip"
        "checksums.txt"
    )

    for asset in "${required_assets[@]}"; do
        if echo "$ASSETS" | grep -q "$asset"; then
            success "  Found: $asset"
        else
            error "  Missing: $asset"
        fi
    done
else
    error "No assets found in release"
fi
echo ""

# Test 3: Download and verify a binary (macOS ARM64 for current platform)
echo "Test 3: Testing binary download..."
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/v${VERSION}/genai-toolbox-darwin-arm64.tar.gz"
TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

info "Downloading: $DOWNLOAD_URL"
if curl -L -o "$TMP_DIR/genai-toolbox.tar.gz" "$DOWNLOAD_URL" 2>/dev/null; then
    success "Download successful"

    # Extract
    info "Extracting..."
    tar -xzf "$TMP_DIR/genai-toolbox.tar.gz" -C "$TMP_DIR"

    if [ -f "$TMP_DIR/genai-toolbox" ]; then
        success "Binary extracted"

        # Make executable
        chmod +x "$TMP_DIR/genai-toolbox"

        # Test binary
        if "$TMP_DIR/genai-toolbox" --version > /dev/null 2>&1; then
            success "Binary is executable"
            VERSION_OUTPUT=$("$TMP_DIR/genai-toolbox" --version 2>&1)
            info "Version: $VERSION_OUTPUT"
        else
            error "Binary doesn't execute"
        fi
    else
        error "Binary not found after extraction"
    fi
else
    error "Download failed"
fi
echo ""

# Test 4: Test checksums
echo "Test 4: Verifying checksums..."
CHECKSUMS_URL="https://github.com/${REPO}/releases/download/v${VERSION}/checksums.txt"
if curl -L -s "$CHECKSUMS_URL" -o "$TMP_DIR/checksums.txt" 2>/dev/null; then
    success "Downloaded checksums"

    # Show checksums
    info "Checksums available:"
    cat "$TMP_DIR/checksums.txt" | head -5

    # Verify our downloaded binary
    cd "$TMP_DIR"
    if echo "$ASSETS" | grep -q "genai-toolbox-darwin-arm64.tar.gz"; then
        EXPECTED_SUM=$(grep "genai-toolbox-darwin-arm64.tar.gz" checksums.txt | awk '{print $1}')
        ACTUAL_SUM=$(shasum -a 256 genai-toolbox.tar.gz | awk '{print $1}')

        if [ "$EXPECTED_SUM" = "$ACTUAL_SUM" ]; then
            success "Checksum verified ✓"
        else
            error "Checksum mismatch!"
            info "Expected: $EXPECTED_SUM"
            info "Actual: $ACTUAL_SUM"
        fi
    fi
else
    error "Failed to download checksums"
fi
echo ""

# Test 5: Test installation script availability
echo "Test 5: Testing installation script..."
INSTALL_SCRIPT_URL="https://raw.githubusercontent.com/${REPO}/main/scripts/install.sh"
if curl -s -o /dev/null -w "%{http_code}" "$INSTALL_SCRIPT_URL" | grep -q "200"; then
    success "Installation script is accessible"
    info "URL: $INSTALL_SCRIPT_URL"

    warning "To test full installation (will install to ~/.local/bin):"
    echo "    curl -fsSL $INSTALL_SCRIPT_URL | bash"
else
    error "Installation script not accessible"
fi
echo ""

# Test 6: Check NPM package.json
echo "Test 6: Checking NPM package configuration..."
PACKAGE_JSON_URL="https://raw.githubusercontent.com/${REPO}/main/package.json"
if curl -s "$PACKAGE_JSON_URL" -o "$TMP_DIR/package.json" 2>/dev/null; then
    success "package.json accessible"
    PACKAGE_NAME=$(python3 -c "import json; print(json.load(open('$TMP_DIR/package.json'))['name'])" 2>/dev/null)
    PACKAGE_VERSION=$(python3 -c "import json; print(json.load(open('$TMP_DIR/package.json'))['version'])" 2>/dev/null)
    info "Package: $PACKAGE_NAME"
    info "Version: $PACKAGE_VERSION"

    warning "To publish to NPM:"
    echo "    npm publish"
else
    error "package.json not accessible"
fi
echo ""

echo "========================================="
echo "📋 Test Summary"
echo "========================================="
echo ""
echo "Release Status: ✓ Published"
echo "Release URL: $RELEASE_URL"
echo ""
echo "Next Steps:"
echo "  1. Visit: $RELEASE_URL"
echo "  2. Verify all assets are present"
echo "  3. Test installation:"
echo "     curl -fsSL https://raw.githubusercontent.com/${REPO}/main/scripts/install.sh | bash"
echo ""
echo "  4. (Optional) Publish to NPM:"
echo "     npm publish"
echo ""
echo "✅ E2E Release Test Complete!"
echo ""
