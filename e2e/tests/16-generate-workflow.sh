#!/bin/bash
set -euo pipefail
source /e2e/lib/helpers.sh

echo "[16/16] Generate GitHub Workflow"

# ============================================
# Test 1: Generate workflow with Pages disabled (default)
# ============================================
echo "  Test 1: Generate workflow with Pages disabled (default)..."

rm -rf /workspace/test-repo
mkdir -p /workspace/test-repo
cd /workspace/test-repo
git init

# Verify no workflow exists yet
[ ! -f .github/workflows/prompt-story.yml ] || fail "workflow should not exist yet"

# Run command with Enter (default = no for Pages)
echo "" | git-prompt-story generate-github-workflow

# Verify workflow created
[ -f .github/workflows/prompt-story.yml ] || fail "workflow not created"

# Verify uses prompt-story action (not prompt-story-with-pages)
grep -q "prompt-story@main" .github/workflows/prompt-story.yml || fail "should use prompt-story action"
! grep -q "prompt-story-with-pages" .github/workflows/prompt-story.yml || fail "should not use prompt-story-with-pages action"

# Verify essential workflow content
grep -q "name: Prompt Story" .github/workflows/prompt-story.yml || fail "missing workflow name"
grep -q "pull_request:" .github/workflows/prompt-story.yml || fail "missing pull_request trigger"
grep -q "github-token:" .github/workflows/prompt-story.yml || fail "missing github-token"

# Verify no cleanup job (only needed for pages)
! grep -q "cleanup-old-previews" .github/workflows/prompt-story.yml || fail "should not have cleanup job"

echo "    - Workflow created with Pages disabled"

# ============================================
# Test 2: Generate workflow with Pages enabled
# ============================================
echo "  Test 2: Generate workflow with Pages enabled..."

rm -rf /workspace/test-repo
mkdir -p /workspace/test-repo
cd /workspace/test-repo
git init

# Run command with 'y' to enable Pages
echo "y" | git-prompt-story generate-github-workflow

# Verify workflow created
[ -f .github/workflows/prompt-story.yml ] || fail "workflow not created"

# Verify uses prompt-story-with-pages action
grep -q "prompt-story-with-pages@main" .github/workflows/prompt-story.yml || fail "should use prompt-story-with-pages action"

# Verify cleanup job is present
grep -q "cleanup-old-previews" .github/workflows/prompt-story.yml || fail "should have cleanup job"

# Verify pages permission
grep -q "pages: read" .github/workflows/prompt-story.yml || fail "should have pages permission"

echo "    - Workflow created with Pages enabled"

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

echo "" | git-prompt-story generate-github-workflow

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

echo "" | git-prompt-story generate-github-workflow

# Verify new content replaced old
! grep -q "old content" .github/workflows/prompt-story.yml || fail "old content should be replaced"
grep -q "name: Prompt Story" .github/workflows/prompt-story.yml || fail "new content missing"

echo "    - Existing workflow file overwritten"

echo "  All generate-github-workflow tests passed"
