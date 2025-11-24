# E2E Release Test Status

## 🚀 Release Initiated

**Tag:** v0.21.1
**Date:** November 24, 2024
**Status:** ⏳ GitHub Actions Running

---

## What Was Done

### 1. Git Tag Created and Pushed ✅
```bash
git tag -a v0.21.1 -m "Enterprise GenAI Toolbox v0.21.1..."
git push origin v0.21.1
```

**Result:** Tag successfully pushed to GitHub

### 2. GitHub Actions Workflow Triggered ✅
The push triggered `.github/workflows/release.yml` which will:
- Build binaries for all platforms (Linux, macOS, Windows)
- Create release packages (.tar.gz, .zip)
- Generate SHA256 checksums
- Publish to GitHub Releases

**Status:** Workflow running (typically takes 5-10 minutes)

---

## Current Status

### GitHub Actions Workflow
**URL:** https://github.com/sethdford/genai-toolbox/actions

The workflow is currently:
1. ✅ Checking out code
2. ⏳ Setting up Go 1.25
3. ⏳ Building binaries for all platforms
4. ⏳ Creating release packages
5. ⏳ Generating checksums
6. ⏳ Publishing release

**Expected Duration:** 5-10 minutes

### Release Page
**URL:** https://github.com/sethdford/genai-toolbox/releases/tag/v0.21.1

Will be created when workflow completes.

---

## Next Steps

### Step 1: Wait for Workflow to Complete ⏳

Monitor at: https://github.com/sethdford/genai-toolbox/actions

Look for:
- Green checkmark ✅ (success)
- Red X ❌ (failure - needs debugging)

### Step 2: Run E2E Test Script

Once the workflow completes, run:
```bash
./scripts/test-release.sh
```

This will test:
- ✓ Release exists
- ✓ All assets are present (6 binaries + checksums)
- ✓ Binary downloads work
- ✓ Checksums are correct
- ✓ Binary is executable
- ✓ Installation script is accessible
- ✓ NPM package.json is correct

### Step 3: Test Installation Methods

#### One-Line Installer
```bash
curl -fsSL https://raw.githubusercontent.com/sethdford/genai-toolbox/main/scripts/install.sh | bash
```

**Expected:**
- Downloads correct binary for your platform
- Installs to `~/.local/bin/genai-toolbox`
- Binary runs: `genai-toolbox --version`

#### Direct Download (Manual Test)
```bash
# macOS ARM64
curl -L https://github.com/sethdford/genai-toolbox/releases/download/v0.21.1/genai-toolbox-darwin-arm64.tar.gz -o genai-toolbox.tar.gz
tar -xzf genai-toolbox.tar.gz
./genai-toolbox --version
```

#### NPM Package (Optional)
```bash
# Publish to npm (requires credentials)
npm publish

# Users can then install
npm install -g @genai-toolbox/server
```

### Step 4: Verify All Platforms

Check that all platform binaries are available:
- ✓ Linux amd64
- ✓ Linux arm64
- ✓ macOS amd64 (Intel)
- ✓ macOS arm64 (Apple Silicon)
- ✓ Windows amd64
- ✓ checksums.txt

---

## Testing Checklist

### Release Artifacts
- [ ] Release v0.21.1 published on GitHub
- [ ] All 6 platform binaries present
- [ ] checksums.txt present
- [ ] Release notes generated

### Binary Validation
- [ ] Darwin arm64 binary downloads
- [ ] Darwin arm64 binary extracts
- [ ] Darwin arm64 binary executes
- [ ] Version output correct
- [ ] Checksum matches

### Installation Scripts
- [ ] One-line installer accessible
- [ ] Shell installer works end-to-end
- [ ] Correct binary downloaded for platform
- [ ] Binary installed to correct location

### Documentation
- [ ] Release notes accurate
- [ ] Installation instructions in README work
- [ ] Links to release work

---

## Troubleshooting

### If Workflow Fails

1. Check workflow logs:
   https://github.com/sethdford/genai-toolbox/actions

2. Common issues:
   - Go version mismatch
   - Build errors (check go.mod)
   - Permission issues (check GITHUB_TOKEN)
   - Missing files (check Makefile targets)

3. Fix and retry:
   ```bash
   # Delete the tag
   git tag -d v0.21.1
   git push origin :refs/tags/v0.21.1

   # Fix issues, commit, and retry
   git tag -a v0.21.1 -m "..."
   git push origin v0.21.1
   ```

### If Binaries Don't Download

- Check URL format
- Verify release is public
- Check asset names match expected

### If Installation Script Fails

- Test URL accessibility
- Check script syntax
- Verify platform detection logic
- Test with specific VERSION env var

---

## Success Criteria

### Workflow Success ✅
- All builds complete
- All packages created
- Release published
- No errors in logs

### E2E Test Success ✅
- `./scripts/test-release.sh` passes all tests
- Binary downloads and runs
- Checksum verification passes
- Installation instructions work

### User Experience ✅
- One-line install works
- Binary runs on all platforms
- Documentation is accurate
- No manual compilation needed

---

## Timeline

| Time | Event | Status |
|------|-------|--------|
| T+0m | Tag pushed | ✅ Complete |
| T+0m | Workflow triggered | ✅ Complete |
| T+5-10m | Workflow completes | ⏳ In Progress |
| T+10m | Release published | ⏳ Pending |
| T+12m | E2E tests run | ⏳ Pending |
| T+15m | Installation tested | ⏳ Pending |

---

## Commands Quick Reference

```bash
# Check workflow status (requires gh CLI)
gh run list --workflow=release.yml

# Run E2E tests
./scripts/test-release.sh

# Test one-line installer
curl -fsSL https://raw.githubusercontent.com/sethdford/genai-toolbox/main/scripts/install.sh | bash

# Verify installation
genai-toolbox --version
genai-toolbox --help

# Test with example config
genai-toolbox --tools-file examples/aws-tools.yaml
```

---

## Current Action Required

**⏳ Wait for GitHub Actions workflow to complete (5-10 minutes)**

Then run:
```bash
./scripts/test-release.sh
```

Monitor progress at:
- Actions: https://github.com/sethdford/genai-toolbox/actions
- Release: https://github.com/sethdford/genai-toolbox/releases/tag/v0.21.1
