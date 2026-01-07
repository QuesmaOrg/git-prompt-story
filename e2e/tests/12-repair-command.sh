#!/bin/bash
set -euo pipefail
source /e2e/lib/helpers.sh

echo "[12/12] Add command (local sessions)"

# ============================================
# Test 1: Add to single commit with missing note
# ============================================
echo "  Test 1: Add to single commit with missing note..."

cleanup_sessions
rm -rf /workspace/test-repo
mkdir -p /workspace/test-repo
cd /workspace/test-repo
git init

# Create initial commit (no hooks installed)
echo "initial" > file.txt
git add file.txt
faketime '2025-01-15 09:00:00' git commit -m "Initial commit"

# Create session that overlaps with next commit's work period
create_mock_session "add-test-1" "2025-01-15T09:15:00Z" "2025-01-15T09:45:00Z"

# Create commit WITHOUT hooks (simulates missing note scenario)
echo "feature" >> file.txt
git add file.txt
export GIT_AUTHOR_DATE="2025-01-15T10:00:00Z"
export GIT_COMMITTER_DATE="2025-01-15T10:00:00Z"
faketime '2025-01-15 10:00:00' git commit -m "Add feature"
unset GIT_AUTHOR_DATE GIT_COMMITTER_DATE

COMMIT_SHA=$(git rev-parse HEAD)

# Verify no note exists yet
if git notes --ref=refs/notes/prompt-story show HEAD 2>/dev/null; then
    fail "Note should not exist before add"
fi

# Run add with local source
git-prompt-story add --source=local HEAD

# Verify note now exists
git notes --ref=refs/notes/prompt-story show HEAD > /dev/null 2>&1 || fail "Note should exist after add"
NOTE=$(git notes --ref=refs/notes/prompt-story show HEAD)
echo "$NOTE" | jq -e '.sessions | length == 1' > /dev/null || fail "Should have 1 session"
echo "$NOTE" | jq -e '.sessions[0].id == "add-test-1"' > /dev/null || fail "Session ID mismatch"

# ============================================
# Test 2: Add --dry-run doesn't create note
# ============================================
echo "  Test 2: Add --dry-run doesn't create note..."

cleanup_sessions
rm -rf /workspace/test-repo
mkdir -p /workspace/test-repo
cd /workspace/test-repo
git init

echo "initial" > file.txt
git add file.txt
faketime '2025-01-15 09:00:00' git commit -m "Initial commit"

create_mock_session "add-dry-run" "2025-01-15T09:15:00Z" "2025-01-15T09:45:00Z"

echo "feature" >> file.txt
git add file.txt
export GIT_AUTHOR_DATE="2025-01-15T10:00:00Z"
export GIT_COMMITTER_DATE="2025-01-15T10:00:00Z"
faketime '2025-01-15 10:00:00' git commit -m "Add feature"
unset GIT_AUTHOR_DATE GIT_COMMITTER_DATE

# Run add with --dry-run
OUTPUT=$(git-prompt-story add --source=local --dry-run HEAD 2>&1)
echo "$OUTPUT" | grep -q "would add" || fail "Dry-run should indicate what would be added"

# Verify note was NOT created
if git notes --ref=refs/notes/prompt-story show HEAD 2>/dev/null; then
    fail "Note should not exist after dry-run"
fi

# ============================================
# Test 3: Add skips commits with existing notes
# ============================================
echo "  Test 3: Add skips commits with existing notes..."

cleanup_sessions
rm -rf /workspace/test-repo
mkdir -p /workspace/test-repo
cd /workspace/test-repo
git init

echo "initial" > file.txt
git add file.txt
faketime '2025-01-15 09:00:00' git commit -m "Initial commit"

# Install hooks so note gets created
git-prompt-story install-hooks

create_mock_session "add-skip" "2025-01-15T09:15:00Z" "2025-01-15T09:45:00Z"

echo "feature" >> file.txt
git add file.txt
export GIT_AUTHOR_DATE="2025-01-15T10:00:00Z"
export GIT_COMMITTER_DATE="2025-01-15T10:00:00Z"
faketime '2025-01-15 10:00:00' git commit -m "Add feature"
unset GIT_AUTHOR_DATE GIT_COMMITTER_DATE

# Get original note SHA
ORIGINAL_NOTE=$(git notes --ref=refs/notes/prompt-story show HEAD)

# Run add (should skip)
OUTPUT=$(git-prompt-story add --source=local HEAD 2>&1)
echo "$OUTPUT" | grep -q "skipped" || fail "Should skip commit with existing note"

# Verify note unchanged
AFTER_NOTE=$(git notes --ref=refs/notes/prompt-story show HEAD)
[ "$ORIGINAL_NOTE" = "$AFTER_NOTE" ] || fail "Note should not have changed"

# ============================================
# Test 4: Add --force overwrites existing note
# ============================================
echo "  Test 4: Add --force overwrites existing note..."

# Create a different session for force test
cleanup_sessions
create_mock_session "add-force-new" "2025-01-15T09:20:00Z" "2025-01-15T09:50:00Z"

# Run add with --force
git-prompt-story add --source=local --force HEAD

# Verify note was replaced (should now reference the new session)
NEW_NOTE=$(git notes --ref=refs/notes/prompt-story show HEAD)
echo "$NEW_NOTE" | jq -e '.sessions[0].id == "add-force-new"' > /dev/null || fail "Note should have new session after force"

# ============================================
# Test 5: Add range of commits
# ============================================
echo "  Test 5: Add range of commits..."

cleanup_sessions
rm -rf /workspace/test-repo
mkdir -p /workspace/test-repo
cd /workspace/test-repo
git init

echo "initial" > file.txt
git add file.txt
faketime '2025-01-15 09:00:00' git commit -m "Initial commit"
INITIAL_SHA=$(git rev-parse HEAD)

# Create sessions for each commit
create_mock_session "range-session-1" "2025-01-15T09:15:00Z" "2025-01-15T09:45:00Z"

echo "commit1" >> file.txt
git add file.txt
export GIT_AUTHOR_DATE="2025-01-15T10:00:00Z"
export GIT_COMMITTER_DATE="2025-01-15T10:00:00Z"
faketime '2025-01-15 10:00:00' git commit -m "Commit 1"
unset GIT_AUTHOR_DATE GIT_COMMITTER_DATE
COMMIT1_SHA=$(git rev-parse HEAD)

create_mock_session "range-session-2" "2025-01-15T10:15:00Z" "2025-01-15T10:45:00Z"

echo "commit2" >> file.txt
git add file.txt
export GIT_AUTHOR_DATE="2025-01-15T11:00:00Z"
export GIT_COMMITTER_DATE="2025-01-15T11:00:00Z"
faketime '2025-01-15 11:00:00' git commit -m "Commit 2"
unset GIT_AUTHOR_DATE GIT_COMMITTER_DATE
COMMIT2_SHA=$(git rev-parse HEAD)

# Verify no notes exist
if git notes --ref=refs/notes/prompt-story show $COMMIT1_SHA 2>/dev/null; then
    fail "Commit 1 should not have note before add"
fi
if git notes --ref=refs/notes/prompt-story show $COMMIT2_SHA 2>/dev/null; then
    fail "Commit 2 should not have note before add"
fi

# Add to range
git-prompt-story add --source=local "${INITIAL_SHA}..HEAD"

# Verify both commits now have notes
git notes --ref=refs/notes/prompt-story show $COMMIT1_SHA > /dev/null 2>&1 || fail "Commit 1 should have note after add"
git notes --ref=refs/notes/prompt-story show $COMMIT2_SHA > /dev/null 2>&1 || fail "Commit 2 should have note after add"

# ============================================
# Test 6: Add --scan finds commits needing notes
# ============================================
echo "  Test 6: Add --scan finds commits needing notes..."

cleanup_sessions
rm -rf /workspace/test-repo
mkdir -p /workspace/test-repo
cd /workspace/test-repo
git init

echo "initial" > file.txt
git add file.txt
faketime '2025-01-15 09:00:00' git commit -m "Initial commit"

# Create commit with Prompt-Story marker but no actual note
# This simulates a commit that was made with hooks, then note was lost
create_mock_session "scan-session" "2025-01-15T09:15:00Z" "2025-01-15T09:45:00Z"

echo "feature" >> file.txt
git add file.txt
export GIT_AUTHOR_DATE="2025-01-15T10:00:00Z"
export GIT_COMMITTER_DATE="2025-01-15T10:00:00Z"
# Include marker in commit message to simulate lost note scenario
faketime '2025-01-15 10:00:00' git commit -m "Add feature

Prompt-Story: Used Claude Code (1 prompts)"
unset GIT_AUTHOR_DATE GIT_COMMITTER_DATE

# Verify no actual note exists
if git notes --ref=refs/notes/prompt-story show HEAD 2>/dev/null; then
    fail "Note should not exist (simulating lost note)"
fi

# Run scan mode
OUTPUT=$(git-prompt-story add --scan 2>&1)
echo "$OUTPUT" | grep -q "Found 1 commit" || fail "Scan should find 1 commit needing notes"

# Verify note was created
git notes --ref=refs/notes/prompt-story show HEAD > /dev/null 2>&1 || fail "Note should exist after scan"

echo "  All add command tests passed"
