#!/bin/bash
set -euo pipefail
source /e2e/lib/helpers.sh

echo "[16/16] Install --workflow Flag"

# ============================================
# Test 1: Generate workflow with Pages enabled (default)
# ============================================
echo "  Test 1: Generate workflow with Pages enabled (default)..."

rm -rf /workspace/test-repo
mkdir -p /workspace/test-repo
cd /workspace/test-repo
git init

# Verify no workflow exists yet
[ ! -f .github/workflows/prompt-story.yml ] || fail "workflow should not exist yet"

# Run command with Enter (default = yes for Pages)
echo "" | git-prompt-story install --workflow

# Verify workflow created
[ -f .github/workflows/prompt-story.yml ] || fail "workflow not created"

# Verify deploy-pages is true
grep -q "deploy-pages: true" .github/workflows/prompt-story.yml || fail "deploy-pages should be true"

# Verify essential workflow content
grep -q "name: Prompt Story" .github/workflows/prompt-story.yml || fail "missing workflow name"
grep -q "pull_request:" .github/workflows/prompt-story.yml || fail "missing pull_request trigger"
grep -q "QuesmaOrg/git-prompt-story" .github/workflows/prompt-story.yml || fail "missing action reference"
grep -q "github-token:" .github/workflows/prompt-story.yml || fail "missing github-token"

echo "    - Workflow created with Pages enabled"

# ============================================
# Test 2: Generate workflow with Pages disabled
# ============================================
echo "  Test 2: Generate workflow with Pages disabled..."

rm -rf /workspace/test-repo
mkdir -p /workspace/test-repo
cd /workspace/test-repo
git init

# Run command with 'n' to disable Pages
echo "n" | git-prompt-story install --workflow

# Verify workflow created
[ -f .github/workflows/prompt-story.yml ] || fail "workflow not created"

# Verify deploy-pages is false
grep -q "deploy-pages: false" .github/workflows/prompt-story.yml || fail "deploy-pages should be false"

echo "    - Workflow created with Pages disabled"

# ============================================
# Test 3: Workflow directory creation
# ============================================
echo "  Test 3: Creates .github/workflows directory if missing..."

rm -rf /workspace/test-repo
mkdir -p /workspace/test-repo
cd /workspace/test-repo
git init

# Ensure no .github directory exists
[ ! -d .github ] || fail ".github should not exist yet"

echo "" | git-prompt-story install --workflow

# Verify directory structure created
[ -d .github/workflows ] || fail ".github/workflows directory not created"
[ -f .github/workflows/prompt-story.yml ] || fail "workflow file not created"

echo "    - Directory structure created correctly"

# ============================================
# Test 4: Overwrite existing workflow
# ============================================
echo "  Test 4: Overwrites existing workflow file..."

rm -rf /workspace/test-repo
mkdir -p /workspace/test-repo
cd /workspace/test-repo
git init

# Create an existing workflow with different content
mkdir -p .github/workflows
echo "old content" > .github/workflows/prompt-story.yml

echo "" | git-prompt-story install --workflow

# Verify new content replaced old
! grep -q "old content" .github/workflows/prompt-story.yml || fail "old content should be replaced"
grep -q "name: Prompt Story" .github/workflows/prompt-story.yml || fail "new content missing"

echo "    - Existing workflow file overwritten"

echo "  All install --workflow tests passed"
