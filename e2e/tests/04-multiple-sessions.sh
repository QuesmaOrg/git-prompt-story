#!/bin/bash
set -euo pipefail
source /e2e/lib/helpers.sh

echo "[4/5] Multiple Sessions"

# Timeline:
#   09:00       - Initial commit
#   09:15-10:00 - Session 1
#   09:30-10:15 - Session 2
#   09:45-10:20 - Session 3
#   10:30       - Feature commit
#
# Expected: All 3 sessions captured in note

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

# Step 4: Create 3 mock sessions with overlapping times
echo "  Step 4: Creating 3 mock sessions..."
create_mock_session "test-session-1" "2025-01-15T09:15:00Z" "2025-01-15T10:00:00Z"
create_mock_session "test-session-2" "2025-01-15T09:30:00Z" "2025-01-15T10:15:00Z"
create_mock_session "test-session-3" "2025-01-15T09:45:00Z" "2025-01-15T10:20:00Z"

# Step 5: Create feature commit at 10:30
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
COMMIT_MSG=$(git log -1 --format=%B)
MARKER_COUNT=$(echo "$COMMIT_MSG" | grep -c "Prompt-Story:" || true)
if [[ "$MARKER_COUNT" != "1" ]]; then
    echo "    ERROR: Expected exactly 1 Prompt-Story marker, got $MARKER_COUNT"
    fail "Wrong marker count"
fi
echo "    - Exactly 1 Prompt-Story marker"

echo "$COMMIT_MSG" | grep -q "Prompt-Story: Used Claude Code" || fail "Missing 'Used Claude Code' in marker"
echo "    - Marker says 'Used Claude Code'"

echo "    Checking note attachment..."
git notes --ref=refs/notes/commits show HEAD > /dev/null 2>&1 || fail "No note attached to HEAD"
echo "    - Note is attached to HEAD"

echo "    Checking note content..."
NOTE=$(git notes --ref=refs/notes/commits show HEAD)

# Should have exactly 3 sessions
SESSION_COUNT=$(echo "$NOTE" | yq '.sessions | length')
if [[ "$SESSION_COUNT" != "3" ]]; then
    echo "    ERROR: Expected 3 sessions, got $SESSION_COUNT"
    echo "    Note content:"
    echo "$NOTE"
    fail "Wrong number of sessions"
fi
echo "    - Note contains 3 sessions"

# Verify all session paths
echo "$NOTE" | yq -e '.sessions[] | select(. == "claude-code/test-session-1.jsonl")' > /dev/null || fail "Missing test-session-1"
echo "    - test-session-1 found"
echo "$NOTE" | yq -e '.sessions[] | select(. == "claude-code/test-session-2.jsonl")' > /dev/null || fail "Missing test-session-2"
echo "    - test-session-2 found"
echo "$NOTE" | yq -e '.sessions[] | select(. == "claude-code/test-session-3.jsonl")' > /dev/null || fail "Missing test-session-3"
echo "    - test-session-3 found"

echo "    Checking transcript storage..."
git cat-file -e "refs/notes/prompt-story-transcripts:claude-code/test-session-1.jsonl" 2>/dev/null || fail "Transcript 1 not stored"
echo "    - test-session-1.jsonl stored"
git cat-file -e "refs/notes/prompt-story-transcripts:claude-code/test-session-2.jsonl" 2>/dev/null || fail "Transcript 2 not stored"
echo "    - test-session-2.jsonl stored"
git cat-file -e "refs/notes/prompt-story-transcripts:claude-code/test-session-3.jsonl" 2>/dev/null || fail "Transcript 3 not stored"
echo "    - test-session-3.jsonl stored"

echo ""
echo "  All assertions passed!"
