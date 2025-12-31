#!/bin/bash
set -euo pipefail
source /e2e/lib/helpers.sh

echo "[3/5] Amend Flow"

# Timeline:
#   09:00       - Initial commit (previous commit)
#   09:15-10:25 - Session 1 (overlapping time)
#   10:30       - Feature commit (original)
#   10:35       - Amend commit
#
# Expected: One Prompt-Story marker, correct session detected

# Clean up any previous sessions
cleanup_sessions

# Step 1: Create fresh test repo
echo "  Step 1: Creating test repo..."
rm -rf /workspace/test-repo
mkdir -p /workspace/test-repo
cd /workspace/test-repo
git init

# Step 2: Create initial commit at 09:00 (NO hooks yet)
echo "  Step 2: Creating initial commit at 09:00..."
echo "initial" > file.txt
git add file.txt
faketime '2025-01-15 09:00:00' git commit -m "Initial commit"

# Step 3: Install hooks
echo "  Step 3: Installing hooks..."
git-prompt-story install-hooks

# Step 4: Create mock session (09:15 - 10:25)
echo "  Step 4: Creating mock session (09:15 - 10:25)..."
create_mock_session "test-session-1" "2025-01-15T09:15:00Z" "2025-01-15T10:25:00Z"

# Step 5: Create feature commit at 10:30
echo "  Step 5: Creating feature commit at 10:30..."
echo "feature code" >> file.txt
git add file.txt
export GIT_AUTHOR_DATE="2025-01-15T10:30:00Z"
export GIT_COMMITTER_DATE="2025-01-15T10:30:00Z"
faketime '2025-01-15 10:30:00' git commit -m "Add feature"
unset GIT_AUTHOR_DATE GIT_COMMITTER_DATE

# Verify feature commit has marker
echo "  Step 5b: Verifying original commit..."
ORIGINAL_MSG=$(git log -1 --format=%B)
ORIGINAL_MARKER_COUNT=$(echo "$ORIGINAL_MSG" | grep -c "Prompt-Story:" || true)
if [[ "$ORIGINAL_MARKER_COUNT" != "1" ]]; then
    echo "    ERROR: Original commit should have exactly 1 Prompt-Story marker, got $ORIGINAL_MARKER_COUNT"
    echo "    Commit message:"
    echo "$ORIGINAL_MSG"
    fail "Wrong marker count in original commit"
fi
echo "    - Original commit has 1 Prompt-Story marker"

# Step 6: Make a small change and amend at 10:35
echo "  Step 6: Amending commit at 10:35..."
echo "# comment" >> file.txt
git add file.txt

export GIT_AUTHOR_DATE="2025-01-15T10:35:00Z"
export GIT_COMMITTER_DATE="2025-01-15T10:35:00Z"
# Use --no-edit to keep the original message (which has our Prompt-Story marker)
# This allows prepare-commit-msg hook to detect amend via the existing marker
faketime '2025-01-15 10:35:00' git commit --amend --no-edit
unset GIT_AUTHOR_DATE GIT_COMMITTER_DATE

# Step 7: Verify results
echo "  Step 7: Verifying results..."

echo "    Checking for single Prompt-Story marker..."
COMMIT_MSG=$(git log -1 --format=%B)
MARKER_COUNT=$(echo "$COMMIT_MSG" | grep -c "Prompt-Story:" || true)
if [[ "$MARKER_COUNT" != "1" ]]; then
    echo "    ERROR: Expected exactly 1 Prompt-Story marker, got $MARKER_COUNT"
    echo "    Commit message:"
    echo "$COMMIT_MSG"
    fail "Duplicate Prompt-Story markers after amend"
fi
echo "    - Exactly 1 Prompt-Story marker (no duplicates)"

# Also verify it's "Used Claude Code", not "none"
echo "$COMMIT_MSG" | grep -q "Prompt-Story: Used Claude Code" || fail "Expected 'Used Claude Code', got different marker"
echo "    - Marker says 'Used Claude Code'"

echo "    Checking note attachment..."
git notes --ref=refs/notes/prompt-story show HEAD > /dev/null 2>&1 || fail "No note attached to HEAD"
echo "    - Note is attached to HEAD"

echo "    Checking note content..."
NOTE=$(git notes --ref=refs/notes/prompt-story show HEAD)

# Should have exactly 1 session
SESSION_COUNT=$(echo "$NOTE" | yq '.sessions | length')
if [[ "$SESSION_COUNT" != "1" ]]; then
    echo "    ERROR: Expected 1 session, got $SESSION_COUNT"
    echo "    Note content:"
    echo "$NOTE"
    fail "Wrong number of sessions detected"
fi
echo "    - Exactly 1 session detected"

# Session should be session-1
echo "$NOTE" | yq -e '.sessions[0] == "claude-code/test-session-1.jsonl"' > /dev/null || fail "Wrong session detected"
echo "    - Correct session (test-session-1) detected"

# Verify work timestamps
echo "    Verifying timestamps..."
echo "$NOTE" | yq -e '.start_work == "2025-01-15T09:00:00Z"' > /dev/null || fail "Wrong start_work timestamp"
echo "    - start_work = 2025-01-15T09:00:00Z (previous commit)"

echo ""
echo "  All assertions passed!"
