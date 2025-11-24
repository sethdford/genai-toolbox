# Fork Strategy: Enterprise GenAI Toolbox

## Current Situation

**Repository:** `sethdford/genai-toolbox` (fork of `googleapis/genai-toolbox`)

**Critical Issue:** The codebase has **mixed references** between the fork and original:
- Go module path: `github.com/googleapis/genai-toolbox`
- All internal imports: `github.com/googleapis/genai-toolbox/*`
- NPM package: `@genai-toolbox/server` (custom, not googleapis)
- Installation scripts: `sethdford/genai-toolbox`
- GitHub Actions: No repo-specific actions, uses standard Go tooling
- Documentation: Mixed references

---

## Why This Matters

### 1. **Go Module Path is CRITICAL**

The module path `github.com/googleapis/genai-toolbox` in `go.mod` is used by:
- **All internal imports** - Every `.go` file imports packages using this path
- **External dependencies** - Any project using this as a library
- **Build system** - Go toolchain resolves dependencies using this path

**Changing this breaks EVERYTHING** - All imports would need to be rewritten.

### 2. **Google-Specific Features**

The original project includes **many Google Cloud Platform integrations**:
- BigQuery, Spanner, Cloud SQL, Firestore, Bigtable
- AlloyDB, Dataproc, Dataplex, Looker
- Cloud Healthcare, Serverless Spark
- GCP authentication and IAM

**Your fork added AWS integrations**:
- DynamoDB, S3, Athena, CloudWatch, Neptune
- Redshift, QLDB, Timestream, DocumentDB
- Splunk, Honeycomb, Tableau

### 3. **Naming Consideration**

Current name "genai-toolbox" is ambiguous:
- Google's version: Focused on GCP only
- Your version: GCP + AWS + third-party platforms = **Enterprise-grade**

**Suggestion:** Rename to reflect enterprise scope

---

## Options Analysis

### Option 1: Keep as Fork, Don't Change Go Module Path ✅ RECOMMENDED

**Approach:**
- Keep `go.mod` as `github.com/googleapis/genai-toolbox`
- Continue development as a fork
- Only update user-facing references

**Pros:**
- No code changes needed
- All imports continue to work
- Can still sync with upstream
- Zero breaking changes

**Cons:**
- Module path points to googleapis (confusing)
- Looks like googleapis' project when it's not
- May conflict if googleapis makes conflicting changes

**What to Update:**
```bash
# User-facing only
- README.md installation URLs
- package.json repository URL
- scripts/install.sh REPO variable
- scripts/install.js GITHUB_REPO
- Documentation links
- server.json metadata
```

**What NOT to Change:**
```bash
# Keep as-is
- go.mod module path
- All *.go imports
- Dockerfile build flags
- .ci/continuous.release.cloudbuild.yaml (Google's CI, not ours)
```

---

### Option 2: Rename to Enterprise Fork, Keep Go Module Path ✅ ALSO GOOD

**Approach:**
- Rename: `genai-toolbox` → `genai-toolbox-enterprise`
- Keep `go.mod` as `github.com/googleapis/genai-toolbox` internally
- Update all external references to use new name

**Pros:**
- Clear differentiation from Google's version
- Communicates "enterprise-grade" value
- User-facing clarity
- Still no code changes needed

**Cons:**
- Module path still points to googleapis
- Slight confusion: package name ≠ module name
- Need to update more places

**What to Update:**
```bash
# Repository and user-facing
- GitHub repo name: genai-toolbox-enterprise
- NPM package: @genai-toolbox-enterprise/server
- Binary name: genai-toolbox-enterprise
- Installation scripts
- All documentation
- README title
```

**What NOT to Change:**
```bash
# Keep as-is
- go.mod module path (still googleapis/genai-toolbox)
- All *.go imports
```

---

### Option 3: Complete Fork with New Module Path ❌ NOT RECOMMENDED

**Approach:**
- Change module path: `github.com/googleapis/genai-toolbox` → `github.com/sethdford/genai-toolbox-enterprise`
- Update ALL imports in ALL .go files
- Completely separate from upstream

**Pros:**
- Complete independence
- No googleapis references
- Clear ownership

**Cons:**
- **MASSIVE BREAKING CHANGE**
- Need to update ~774+ files with imports
- Breaks any external packages depending on this
- Can never sync with googleapis upstream again
- High risk of errors
- Time-consuming (hours of work)

**What to Update:**
```bash
# Everything
- go.mod module path
- Every single .go file's imports (774+ files)
- go.sum regeneration
- All documentation
- All build scripts
- All external references
```

---

## Recommendation

### **Choose Option 1 or 2** (Your Call)

Both are viable and avoid the massive breaking change of Option 3.

#### **Option 1: Minimal Changes** ⭐ Best for Quick Ship

Keep everything as-is, just update user-facing references:

```bash
# Quick updates (15 minutes)
1. Update scripts/install.sh REPO to sethdford/genai-toolbox
2. Update scripts/install.js GITHUB_REPO to sethdford/genai-toolbox
3. Update package.json repository URL
4. Update README.md installation URLs
5. Update documentation links
```

**Best if:** You want to ship fast and don't care about Google references in code.

#### **Option 2: Enterprise Branding** ⭐ Best for Long-term

Rename everything user-facing to "Enterprise GenAI Toolbox":

```bash
# Comprehensive updates (1-2 hours)
1. Rename GitHub repo: genai-toolbox → genai-toolbox-enterprise
2. Update binary name to genai-toolbox-enterprise
3. Update NPM package to @genai-toolbox-enterprise/server
4. Update all installation scripts
5. Update all documentation
6. Update README title and branding
7. Add "Enterprise" branding throughout
```

**Best if:** You want clear differentiation and are building this as a product.

---

## What Actually Needs Changing

### Files with `sethdford` or Wrong Repo References

Based on grep analysis, these files reference the fork specifically:

#### **Installation Scripts** (MUST UPDATE)
```bash
scripts/install.sh:9:REPO="sethdford/genai-toolbox"
scripts/install.js:19:const GITHUB_REPO = 'sethdford/genai-toolbox';
scripts/test-release.sh:7:REPO="sethdford/genai-toolbox"
```

#### **Package Metadata** (MUST UPDATE)
```json
// package.json
{
  "name": "@genai-toolbox/server",
  "repository": "https://github.com/sethdford/genai-toolbox"
}
```

#### **Documentation** (SHOULD UPDATE)
```
README.md - Multiple installation URLs
INSTALL.md - Installation instructions
E2E_RELEASE_STATUS.md - Test URLs
VALIDATION_RESULTS.md - Installation URLs
docs/guides/*.md - Multiple references
bin/genai-toolbox.js - Error message URLs
```

#### **Server Metadata** (SHOULD UPDATE if used)
```json
// server.json - MCP server registry metadata
{
  "name": "io.github.googleapis/genai-toolbox",
  "websiteUrl": "https://github.com/googleapis/genai-toolbox"
}
```

### Files with `googleapis` References

#### **DO NOT CHANGE** (Core Code)
```bash
# These MUST stay as googleapis/genai-toolbox
go.mod:1:module github.com/googleapis/genai-toolbox
main.go:18:import "github.com/googleapis/genai-toolbox/cmd"
# + 700+ other .go files with imports
```

#### **CAN IGNORE** (Google's CI, Not Ours)
```yaml
# .ci/continuous.release.cloudbuild.yaml
# This is Google's Cloud Build config, not used by our GitHub Actions
# Safe to leave as-is or delete
```

#### **CAN UPDATE** (User-Facing)
```
README.md - Google links (keep for attribution)
CHANGELOG.md - Original project history (keep for history)
docs/*.md - Installation instructions (update)
```

---

## Immediate Action Plan

### Phase 1: Fix Current Release (v0.21.2) - 15 Minutes

These are **blocking the release** - already done but verify:

```bash
# Already updated
✅ scripts/install.sh REPO="sethdford/genai-toolbox"
✅ scripts/install.js GITHUB_REPO = 'sethdford/genai-toolbox'
✅ scripts/test-release.sh REPO="sethdford/genai-toolbox"
✅ package.json repository URL
✅ bin/genai-toolbox.js error message URL

# Not updated yet (non-blocking)
❓ server.json - MCP metadata
❓ README.md - Still has some googleapis links
❓ Other docs - Mixed references
```

### Phase 2: Decide on Branding - Discussion

**Option A: Keep "GenAI Toolbox"**
- Pros: Simpler, less changes
- Cons: Ambiguous, not differentiated

**Option B: Rename "Enterprise GenAI Toolbox"**
- Pros: Clear positioning, better branding
- Cons: More updates needed

**Questions to Consider:**
1. Do you want this to be seen as a separate product?
2. Are you planning to maintain this long-term independently?
3. Do you want to sync with googleapis upstream in the future?
4. Do you care about the googleapis references in code?

### Phase 3: Update User-Facing (If Decided)

If you choose Enterprise branding:

```bash
# Update these
1. GitHub repo name (rename on GitHub)
2. package.json name and version
3. Binary name in Makefile
4. README title and branding
5. All documentation
6. Installation scripts binary names
```

---

## My Recommendation

**For v0.21.2 Release:** ✅ Already Good

The current release is properly configured:
- Installation scripts point to sethdford/genai-toolbox ✅
- NPM package points to fork ✅
- Release assets will publish to fork ✅
- Go module path stays googleapis (fine for now) ✅

**For Future Versions:** Consider Option 2 (Enterprise Branding)

Why:
1. Your fork adds **significant** enterprise value (AWS + third-party)
2. Clear differentiation helps users understand the scope
3. Google's version is GCP-only, yours is multi-cloud
4. No breaking code changes needed (keep go.mod as-is)
5. Better long-term positioning

**Proposed New Names:**
- Repo: `genai-toolbox-enterprise`
- Package: `@genai-toolbox-enterprise/server` or `@enterprise/genai-toolbox`
- Binary: `genai-toolbox` (keep short for UX) or `genai-toolbox-enterprise`
- Branding: "Enterprise GenAI Toolbox"

---

## Summary

### What's Actually Broken? **Nothing Critical**

The current setup works:
- Installation scripts correctly point to your fork ✅
- NPM package works ✅
- Releases publish to your repo ✅
- Go module path (googleapis) is fine for internal use ✅

### What Needs Attention? **Branding & Clarity**

The confusion is:
- Is this Google's project or yours? (Unclear)
- What's the scope? (GCP only or enterprise?) (Unclear)
- Who maintains it? (Unclear)

### What Should You Do? **Ship v0.21.2, Then Decide**

1. **Now:** Let v0.21.2 release complete (in progress)
2. **Today:** Test installation and verify everything works
3. **This Week:** Decide if you want "Enterprise" branding
4. **Next Release:** Implement branding if decided

**Don't** change the go.mod module path - it's not worth the effort.

---

## Files That Need Updates (Summary)

### Must Update (Already Done)
- [x] scripts/install.sh
- [x] scripts/install.js
- [x] scripts/test-release.sh
- [x] package.json
- [x] bin/genai-toolbox.js

### Should Update (For Clarity)
- [ ] server.json (MCP metadata)
- [ ] README.md (mixed references)
- [ ] INSTALL.md (googleapis links)
- [ ] docs/guides/*.md (installation URLs)

### Do Not Change
- [ ] go.mod (keep as googleapis/genai-toolbox)
- [ ] Any *.go files (keep imports as-is)
- [ ] .ci/*.yaml (Google's CI config, not used)

---

**Last Updated:** November 24, 2024
**Status:** v0.21.2 release in progress with correct fork references
**Next Decision:** Enterprise branding for v0.22.0?
