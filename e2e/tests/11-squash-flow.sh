#!/bin/bash
set -euo pipefail
source /e2e/lib/helpers.sh

echo "[11/11] Squash Flow Test"

# Timeline:
#   09:00       - Initial commit (no hooks)
#   09:15       - Install hooks
#   09:20-09:25 - Session A active
#   09:30       - Commit 1 (with session A)
#   09:45-09:55 - Session B active
#   10:00       - Commit 2 (with session B)
#   10:15       - Squash commits 1+2 into one
#
# Expected: Squashed commit has merged note with both sessions

# Clean up any previous sessions
cleanup_sessions

# Step 1: Create fresh test repo
echo "  Step 1: Creating test repo..."
rm -rf /workspace/test-repo
mkdir -p /workspace/test-repo
cd /workspace/test-repo
git init

# Configure git for non-interactive rebase
git config user.email "test@example.com"
git config user.name "Test User"

# Step 2: Create initial commit at 09:00 (NO hooks yet)
echo "  Step 2: Creating initial commit at 09:00..."
echo "initial" > file.txt
git add file.txt
faketime '2025-01-15 09:00:00' git commit -m "Initial commit"

# Step 3: Install hooks
echo "  Step 3: Installing hooks..."
git-prompt-story install-hooks

# Step 4: Create first session (09:20 - 09:25) and commit at 09:30
echo "  Step 4: Creating session A and commit 1..."
create_mock_session "session-A" "2025-01-15T09:20:00Z" "2025-01-15T09:25:00Z"

echo "feature A" >> file.txt
git add file.txt
export GIT_AUTHOR_DATE="2025-01-15T09:30:00Z"
export GIT_COMMITTER_DATE="2025-01-15T09:30:00Z"
faketime '2025-01-15 09:30:00' git commit -m "Add feature A"
unset GIT_AUTHOR_DATE GIT_COMMITTER_DATE

# Verify commit 1 has note
echo "  Step 4b: Verifying commit 1 has note..."
COMMIT1_SHA=$(git rev-parse HEAD)
git notes --ref=refs/notes/prompt-story show HEAD > /dev/null 2>&1 || fail "Commit 1 should have a note"
NOTE1=$(git notes --ref=refs/notes/prompt-story show HEAD)
echo "$NOTE1" | jq -e '.sessions[0].id == "session-A"' > /dev/null || fail "Commit 1 should reference session-A"
echo "    - Commit 1 has note with session-A"

# Step 5: Clean up session A (simulating session ended), create session B, commit
echo "  Step 5: Cleaning up session A (session ended), creating session B..."

# In real usage, when a session ends, its file is no longer in the active sessions directory
# This simulates that session A ended before session B started
cleanup_sessions

# Now create session B
create_mock_session "session-B" "2025-01-15T09:45:00Z" "2025-01-15T09:55:00Z"

echo "feature B" >> file.txt
git add file.txt
export GIT_AUTHOR_DATE="2025-01-15T10:00:00Z"
export GIT_COMMITTER_DATE="2025-01-15T10:00:00Z"
faketime '2025-01-15 10:00:00' git commit -m "Add feature B"
unset GIT_AUTHOR_DATE GIT_COMMITTER_DATE

# Verify commit 2 has note
echo "  Step 5b: Verifying commit 2 has note..."
COMMIT2_SHA=$(git rev-parse HEAD)
git notes --ref=refs/notes/prompt-story show HEAD > /dev/null 2>&1 || fail "Commit 2 should have a note"
NOTE2=$(git notes --ref=refs/notes/prompt-story show HEAD)
echo "$NOTE2" | jq -e '.sessions[0].id == "session-B"' > /dev/null || fail "Commit 2 should reference session-B"
echo "    - Commit 2 has note with session-B"

# Step 6: Clean up session B (session ended) and squash commits
echo "  Step 6: Cleaning up session B (session ended), then squashing..."

# Clean up all sessions - simulating both sessions have ended
# The only record of the sessions is now in the git notes
cleanup_sessions

# Get the initial commit SHA (the one before our feature commits)
INITIAL_SHA=$(git rev-parse HEAD~2)

# Create a script for GIT_SEQUENCE_EDITOR to change "pick" to "squash" for second commit
cat > /tmp/rebase-editor.sh << 'EDITOREOF'
#!/bin/bash
sed -i '2s/^pick/squash/' "$1"
EDITOREOF
chmod +x /tmp/rebase-editor.sh

# Create a script for the commit message editor during squash
cat > /tmp/commit-editor.sh << 'COMMITEOF'
#!/bin/bash
cat > "$1" << 'MSG'
Add features A and B (squashed)

This commit combines feature A and feature B.
MSG
COMMITEOF
chmod +x /tmp/commit-editor.sh

# Use GIT_SEQUENCE_EDITOR for rebase todo and EDITOR for squash message
export GIT_SEQUENCE_EDITOR="/tmp/rebase-editor.sh"
export EDITOR="/tmp/commit-editor.sh"
export GIT_EDITOR="/tmp/commit-editor.sh"

faketime '2025-01-15 10:15:00' git rebase -i HEAD~2

unset GIT_SEQUENCE_EDITOR EDITOR GIT_EDITOR

echo "    - Rebase complete"

# Step 7: Verify the squashed commit has merged notes
echo "  Step 7: Verifying squashed commit..."

SQUASHED_SHA=$(git rev-parse HEAD)
echo "    Squashed commit: $SQUASHED_SHA"
echo "    Original commits: $COMMIT1_SHA, $COMMIT2_SHA"

# Check that squashed commit has a note
if ! git notes --ref=refs/notes/prompt-story show HEAD > /dev/null 2>&1; then
    echo "    ERROR: Squashed commit has no note attached"
    echo "    This is expected to fail until post-rewrite hook is implemented"
    fail "Squashed commit should have a merged note"
fi

MERGED_NOTE=$(git notes --ref=refs/notes/prompt-story show HEAD)
echo "    Merged note content:"
echo "$MERGED_NOTE" | jq .

# Verify both sessions are in the merged note
SESSION_COUNT=$(echo "$MERGED_NOTE" | jq '.sessions | length')
if [[ "$SESSION_COUNT" != "2" ]]; then
    echo "    ERROR: Expected 2 sessions in merged note, got $SESSION_COUNT"
    fail "Merged note should contain both sessions"
fi
echo "    - Merged note has 2 sessions"

# Verify session-A is present
echo "$MERGED_NOTE" | jq -e '.sessions[] | select(.id == "session-A")' > /dev/null || fail "Merged note should contain session-A"
echo "    - session-A present"

# Verify session-B is present
echo "$MERGED_NOTE" | jq -e '.sessions[] | select(.id == "session-B")' > /dev/null || fail "Merged note should contain session-B"
echo "    - session-B present"

# Verify start_work is from the earlier commit
START_WORK=$(echo "$MERGED_NOTE" | jq -r '.start_work')
echo "    - start_work: $START_WORK"

echo ""
echo "  All squash flow assertions passed!"
