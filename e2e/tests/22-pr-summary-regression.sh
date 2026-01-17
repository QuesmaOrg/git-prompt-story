#!/bin/bash
set -euo pipefail
source /e2e/lib/helpers.sh

echo "[22/22] PR Summary Regression Tests"

# This test suite verifies that pr summary correctly handles edge cases
# that previously caused false "Notes not found" warnings in GitHub Actions.
# Regression test for: commit 5f848e2 (Remove 'Notes not found' warning)

cleanup_sessions

# ============================================
# Test 1: No markers, no notes - should report 0 notes, 0 analyzed
# ============================================
echo "  Test 1: Commits with no markers and no notes..."

rm -rf /workspace/test-repo
mkdir -p /workspace/test-repo
cd /workspace/test-repo
git init

# Create initial commit (no hooks, no markers)
echo "initial" > file.txt
git add file.txt
faketime '2025-01-15 09:00:00' git commit -m "Initial commit"
INITIAL=$(git rev-parse HEAD)

# Create a feature commit without any AI involvement
echo "feature" >> file.txt
git add file.txt
faketime '2025-01-15 10:00:00' git commit -m "Add feature without AI"

# pr summary should report 0 commits with notes
OUTPUT=$(git-prompt-story pr summary "${INITIAL}..HEAD" --gha)
echo "$OUTPUT" | grep -q "commits-analyzed=1" || fail "Should analyze 1 commit"
echo "$OUTPUT" | grep -q "commits-with-notes=0" || fail "Should report 0 commits with notes"
echo "$OUTPUT" | grep -q "should-post-comment=false" || fail "Should not post comment"
echo "    - Correctly reports 0 notes for regular commits"

# ============================================
# Test 2: Prompt-Story: none marker - should NOT trigger false positive
# ============================================
echo "  Test 2: Commits with 'Prompt-Story: none' marker..."

rm -rf /workspace/test-repo
mkdir -p /workspace/test-repo
cd /workspace/test-repo
git init

echo "initial" > file.txt
git add file.txt
faketime '2025-01-15 09:00:00' git commit -m "Initial commit"
INITIAL=$(git rev-parse HEAD)

# Commit with explicit "Prompt-Story: none" marker (human-written, no AI)
echo "feature" >> file.txt
git add file.txt
faketime '2025-01-15 10:00:00' git commit -m "Add feature

Prompt-Story: none"

OUTPUT=$(git-prompt-story pr summary "${INITIAL}..HEAD" --gha)
echo "$OUTPUT" | grep -q "commits-analyzed=1" || fail "Should analyze 1 commit"
echo "$OUTPUT" | grep -q "commits-with-notes=0" || fail "Should report 0 commits with notes"
echo "    - Correctly handles 'Prompt-Story: none' marker"

# ============================================
# Test 3: Text that looks like marker but isn't - should NOT trigger false positive
# ============================================
echo "  Test 3: Commits with text resembling marker pattern..."

rm -rf /workspace/test-repo
mkdir -p /workspace/test-repo
cd /workspace/test-repo
git init

echo "initial" > file.txt
git add file.txt
faketime '2025-01-15 09:00:00' git commit -m "Initial commit"
INITIAL=$(git rev-parse HEAD)

# Commit message that mentions "Prompt-Story:" but isn't a real marker
echo "feature" >> file.txt
git add file.txt
faketime '2025-01-15 10:00:00' git commit -m "Update tracking doc

See the Prompt-Story: section in CONTRIBUTING.md for details"

OUTPUT=$(git-prompt-story pr summary "${INITIAL}..HEAD" --gha)
echo "$OUTPUT" | grep -q "commits-analyzed=1" || fail "Should analyze 1 commit"
echo "$OUTPUT" | grep -q "commits-with-notes=0" || fail "Should report 0 commits with notes"
echo "    - Correctly handles text that looks like marker pattern"

# ============================================
# Test 4: Multiple commits, none with actual notes
# ============================================
echo "  Test 4: Multiple commits without notes..."

rm -rf /workspace/test-repo
mkdir -p /workspace/test-repo
cd /workspace/test-repo
git init

echo "initial" > file.txt
git add file.txt
faketime '2025-01-15 09:00:00' git commit -m "Initial commit"
INITIAL=$(git rev-parse HEAD)

# Multiple commits without notes
echo "feature1" >> file.txt
git add file.txt
faketime '2025-01-15 10:00:00' git commit -m "Add feature 1"

echo "feature2" >> file.txt
git add file.txt
faketime '2025-01-15 11:00:00' git commit -m "Add feature 2

Prompt-Story: none"

echo "feature3" >> file.txt
git add file.txt
faketime '2025-01-15 12:00:00' git commit -m "Add feature 3"

OUTPUT=$(git-prompt-story pr summary "${INITIAL}..HEAD" --gha)
echo "$OUTPUT" | grep -q "commits-analyzed=3" || fail "Should analyze 3 commits"
echo "$OUTPUT" | grep -q "commits-with-notes=0" || fail "Should report 0 commits with notes"
echo "    - Correctly handles multiple commits without notes"

# ============================================
# Test 5: Mixed commits - some with notes, some without
# ============================================
echo "  Test 5: Mixed commits - some with notes, some without..."

rm -rf /workspace/test-repo
mkdir -p /workspace/test-repo
cd /workspace/test-repo
git init

echo "initial" > file.txt
git add file.txt
faketime '2025-01-15 09:00:00' git commit -m "Initial commit"
INITIAL=$(git rev-parse HEAD)

# Install hooks for AI-assisted commits
git-prompt-story install-hooks

# Commit 1: With AI session (will have notes)
create_mock_session "ai-session-1" "2025-01-15T09:30:00Z" "2025-01-15T09:45:00Z"
echo "ai-feature" >> file.txt
git add file.txt
export GIT_AUTHOR_DATE="2025-01-15T10:00:00Z"
export GIT_COMMITTER_DATE="2025-01-15T10:00:00Z"
faketime '2025-01-15 10:00:00' git commit -m "Add AI-assisted feature"
unset GIT_AUTHOR_DATE GIT_COMMITTER_DATE

# Commit 2: Without AI session (no notes)
cleanup_sessions
echo "manual-feature" >> file.txt
git add file.txt
faketime '2025-01-15 11:00:00' git commit -m "Add manual feature

Prompt-Story: none"

# Commit 3: Another without AI
echo "manual-feature-2" >> file.txt
git add file.txt
faketime '2025-01-15 12:00:00' git commit -m "Another manual feature"

OUTPUT=$(git-prompt-story pr summary "${INITIAL}..HEAD" --gha)
echo "$OUTPUT" | grep -q "commits-analyzed=3" || fail "Should analyze 3 commits"
echo "$OUTPUT" | grep -q "commits-with-notes=1" || fail "Should report 1 commit with notes"
echo "$OUTPUT" | grep -q "should-post-comment=true" || fail "Should post comment when notes exist"
echo "    - Correctly counts only commits with actual notes"

# ============================================
# Test 6: Verify markdown output doesn't include false warnings
# ============================================
echo "  Test 6: Markdown output for commits without notes..."

rm -rf /workspace/test-repo
mkdir -p /workspace/test-repo
cd /workspace/test-repo
git init

echo "initial" > file.txt
git add file.txt
faketime '2025-01-15 09:00:00' git commit -m "Initial commit"
INITIAL=$(git rev-parse HEAD)

echo "feature" >> file.txt
git add file.txt
faketime '2025-01-15 10:00:00' git commit -m "Add feature without AI

Prompt-Story: none"

MD_OUTPUT=$(git-prompt-story pr summary "${INITIAL}..HEAD")

# Should NOT contain warning messages
if echo "$MD_OUTPUT" | grep -q "Notes not found"; then
    fail "Markdown should NOT contain 'Notes not found' warning"
fi
if echo "$MD_OUTPUT" | grep -q "forgot to push"; then
    fail "Markdown should NOT contain 'forgot to push' warning"
fi
echo "    - Markdown output does not contain false warnings"

# The output should be empty or minimal for no-notes case
LINES=$(echo "$MD_OUTPUT" | wc -l)
echo "    - Markdown output has $LINES lines (expected minimal)"

echo ""
echo "  All regression tests passed!"
