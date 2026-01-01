#!/bin/bash
set -euo pipefail
source /e2e/lib/helpers.sh

echo "[2/5] Session Detection"

# Timeline:
#   07:00-08:00 - Session 3 (old session, same project - BEFORE previous commit)
#   09:00       - Initial commit (previous commit)
#   09:15-10:25 - Session 1 (correct project, overlapping time - SHOULD BE DETECTED)
#   09:15-10:25 - Session 2 (different project - should NOT be detected)
#   10:30       - Feature commit (current commit)
#
# Expected: Only Session 1 should be detected

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

# Step 4: Create 3 mock sessions
echo "  Step 4: Creating mock sessions..."

# Session 1: Correct project, overlapping time (09:15 - 10:25) - SHOULD BE DETECTED
create_mock_session "session-1" "2025-01-15T09:15:00Z" "2025-01-15T10:25:00Z"

# Session 2: Different project directory (09:15 - 10:25) - should NOT be detected
create_mock_session_for_path "/workspace/other-repo" "session-2" "2025-01-15T09:15:00Z" "2025-01-15T10:25:00Z"

# Session 3: Same project, but BEFORE previous commit (07:00 - 08:00) - should NOT be detected
create_mock_session "session-3" "2025-01-15T07:00:00Z" "2025-01-15T08:00:00Z"

# Step 5: Add file and commit at 10:30
echo "  Step 5: Creating feature commit at 10:30..."
echo "feature code" >> file.txt
git add file.txt
export GIT_AUTHOR_DATE="2025-01-15T10:30:00Z"
export GIT_COMMITTER_DATE="2025-01-15T10:30:00Z"
faketime '2025-01-15 10:30:00' git commit -m "Add feature"
unset GIT_AUTHOR_DATE GIT_COMMITTER_DATE

# Step 6: Verify results
echo "  Step 6: Verifying results..."

echo "    Checking commit message..."
git log -1 --format=%B | grep -q "Prompt-Story: Used Claude Code" || fail "Missing Prompt-Story trailer"
echo "    - Commit message contains Prompt-Story trailer"

echo "    Checking note attachment..."
git notes --ref=refs/notes/prompt-story show HEAD > /dev/null 2>&1 || fail "No note attached to HEAD"
echo "    - Note is attached to HEAD"

echo "    Checking only session-1 is detected..."
NOTE=$(git notes --ref=refs/notes/prompt-story show HEAD)

# Should have exactly 1 session
SESSION_COUNT=$(echo "$NOTE" | jq '.sessions | length')
if [[ "$SESSION_COUNT" != "1" ]]; then
    echo "    ERROR: Expected 1 session, got $SESSION_COUNT"
    echo "    Note content:"
    echo "$NOTE" | jq .
    fail "Wrong number of sessions detected"
fi
echo "    - Exactly 1 session detected"

# Session should be session-1
echo "$NOTE" | jq -e '.sessions[0].id == "session-1"' > /dev/null || fail "Wrong session detected (expected session-1)"
echo "    - Correct session (session-1) detected"

# Verify session-2 is NOT in the note (different project)
if echo "$NOTE" | jq -e '.sessions[] | select(.id == "session-2")' > /dev/null 2>&1; then
    fail "session-2 (different project) should NOT be detected"
fi
echo "    - session-2 (different project) correctly NOT detected"

# Verify session-3 is NOT in the note (before previous commit)
if echo "$NOTE" | jq -e '.sessions[] | select(.id == "session-3")' > /dev/null 2>&1; then
    fail "session-3 (old session) should NOT be detected"
fi
echo "    - session-3 (before previous commit) correctly NOT detected"

# Verify timestamps
echo "    Verifying timestamps..."
echo "$NOTE" | jq -e '.sessions[0].created == "2025-01-15T09:15:00Z"' > /dev/null || fail "Wrong session created timestamp"
echo "    - session.created = 2025-01-15T09:15:00Z"

echo "$NOTE" | jq -e '.sessions[0].modified == "2025-01-15T10:25:00Z"' > /dev/null || fail "Wrong session modified timestamp"
echo "    - session.modified = 2025-01-15T10:25:00Z"

echo ""
echo "  All assertions passed!"
