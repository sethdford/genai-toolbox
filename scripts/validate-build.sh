#!/bin/bash
# Validation script for build system
# Tests all Makefile targets and validates the build output

set -e

echo "🔍 Validating Enterprise GenAI Toolbox Build System"
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

success() {
    echo -e "${GREEN}✓${NC} $1"
}

error() {
    echo -e "${RED}✗${NC} $1"
}

info() {
    echo -e "${YELLOW}ℹ${NC} $1"
}

# Test 1: Makefile exists and is valid
echo "Test 1: Makefile validation"
if [ -f "Makefile" ]; then
    success "Makefile exists"
    make help > /dev/null 2>&1 && success "Makefile help target works" || error "Makefile help failed"
else
    error "Makefile not found"
    exit 1
fi
echo ""

# Test 2: Clean build
echo "Test 2: Clean build directory"
make clean > /dev/null 2>&1 && success "Clean target works" || error "Clean failed"
echo ""

# Test 3: Current platform build
echo "Test 3: Build for current platform"
if make build > /dev/null 2>&1; then
    success "Build completed"
    if [ -f "genai-toolbox" ] || [ -f "genai-toolbox.exe" ]; then
        success "Binary created"

        # Test binary
        if ./genai-toolbox --version > /dev/null 2>&1; then
            success "Binary is executable and runs"
            VERSION=$(./genai-toolbox --version 2>&1 | head -1)
            info "Version: $VERSION"
        else
            error "Binary doesn't execute properly"
        fi
    else
        error "Binary not found after build"
    fi
else
    error "Build failed"
fi
echo ""

# Test 4: Test suite
echo "Test 4: Run test suite"
if make test > /dev/null 2>&1; then
    success "Tests pass"
else
    error "Tests failed"
fi
echo ""

# Test 5: YAML validation
echo "Test 5: Example configuration validation"
if [ -f "examples/aws-tools.yaml" ]; then
    if python3 -c "import yaml; yaml.safe_load(open('examples/aws-tools.yaml'))" 2>/dev/null; then
        success "AWS tools YAML is valid"
    else
        error "AWS tools YAML has syntax errors"
    fi
else
    info "aws-tools.yaml not found (skipping)"
fi
echo ""

# Test 6: GitHub Actions workflow validation
echo "Test 6: GitHub Actions workflow validation"
if [ -f ".github/workflows/release.yml" ]; then
    if python3 -c "import yaml; yaml.safe_load(open('.github/workflows/release.yml'))" 2>/dev/null; then
        success "Release workflow YAML is valid"
    else
        error "Release workflow has YAML syntax errors"
    fi
else
    error "Release workflow not found"
fi
echo ""

# Test 7: Package.json validation
echo "Test 7: NPM package configuration"
if [ -f "package.json" ]; then
    if python3 -c "import json; json.load(open('package.json'))" 2>/dev/null; then
        success "package.json is valid JSON"
        PACKAGE_NAME=$(python3 -c "import json; print(json.load(open('package.json'))['name'])")
        info "Package: $PACKAGE_NAME"
    else
        error "package.json has syntax errors"
    fi
else
    error "package.json not found"
fi
echo ""

# Test 8: Installation scripts exist
echo "Test 8: Installation scripts"
if [ -f "scripts/install.sh" ] && [ -x "scripts/install.sh" ]; then
    success "Shell installer exists and is executable"
else
    error "Shell installer missing or not executable"
fi

if [ -f "scripts/install.js" ] && [ -x "scripts/install.js" ]; then
    success "NPM installer exists and is executable"
else
    error "NPM installer missing or not executable"
fi
echo ""

# Test 9: Documentation
echo "Test 9: Documentation files"
docs=("README.md" "INSTALL.md" "CONTRIBUTING.md" "LICENSE")
for doc in "${docs[@]}"; do
    if [ -f "$doc" ]; then
        success "$doc exists"
    else
        info "$doc not found (optional)"
    fi
done
echo ""

# Test 10: Clean up
echo "Test 10: Cleanup"
make clean > /dev/null 2>&1 && success "Cleanup successful" || error "Cleanup failed"
echo ""

echo "========================================="
echo "✅ Build validation complete!"
echo "========================================="
echo ""
echo "To build releases:"
echo "  make build-all      # All platforms"
echo "  make package        # Create release packages"
echo ""
echo "To test locally:"
echo "  make build          # Current platform"
echo "  ./genai-toolbox --tools-file examples/aws-tools.yaml"
echo ""
