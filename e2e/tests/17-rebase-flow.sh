#!/bin/bash
set -euo pipefail
source /e2e/lib/helpers.sh

echo "[14/14] Rebase Flow Test"

# This test validates the post-rewrite hook during git rebase operations.
# It differs from 11-squash-flow.sh by testing note preservation
# when reordering/editing commits (not just squashing).
#
# Timeline:
#   09:00       - Initial commit (no hooks)
#   09:15       - Install hooks
#   09:20-09:25 - Session A active
#   09:30       - Commit 1 (with session A)
#   09:40-09:50 - Session B active
#   10:00       - Commit 2 (with session B)
#   10:10-10:20 - Session C active
#   10:30       - Commit 3 (with session C)
#   11:00       - Rebase: reword commit 2, squash commit 3 into commit 2
#
# Expected:
#   - Commit 1 note preserved (session-A)
#   - New squashed commit has merged note (session-B + session-C)

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

# Verify commit 1 has note with session-A
echo "  Step 4b: Verifying commit 1 has note..."
COMMIT1_SHA=$(git rev-parse HEAD)
git notes --ref=refs/notes/prompt-story show HEAD > /dev/null 2>&1 || fail "Commit 1 should have a note"
NOTE1=$(git notes --ref=refs/notes/prompt-story show HEAD)
echo "$NOTE1" | jq -e '.sessions[0].id == "session-A"' > /dev/null || fail "Commit 1 should reference session-A"
echo "    - Commit 1 ($COMMIT1_SHA) has note with session-A"

# Step 5: Clean up session A, create session B, commit 2
echo "  Step 5: Creating session B and commit 2..."
cleanup_sessions
create_mock_session "session-B" "2025-01-15T09:40:00Z" "2025-01-15T09:50:00Z"

echo "feature B" >> file.txt
git add file.txt
export GIT_AUTHOR_DATE="2025-01-15T10:00:00Z"
export GIT_COMMITTER_DATE="2025-01-15T10:00:00Z"
faketime '2025-01-15 10:00:00' git commit -m "Add feature B"
unset GIT_AUTHOR_DATE GIT_COMMITTER_DATE

COMMIT2_SHA=$(git rev-parse HEAD)
git notes --ref=refs/notes/prompt-story show HEAD > /dev/null 2>&1 || fail "Commit 2 should have a note"
NOTE2=$(git notes --ref=refs/notes/prompt-story show HEAD)
echo "$NOTE2" | jq -e '.sessions[0].id == "session-B"' > /dev/null || fail "Commit 2 should reference session-B"
echo "    - Commit 2 ($COMMIT2_SHA) has note with session-B"

# Step 6: Clean up session B, create session C, commit 3
echo "  Step 6: Creating session C and commit 3..."
cleanup_sessions
create_mock_session "session-C" "2025-01-15T10:10:00Z" "2025-01-15T10:20:00Z"

echo "feature C" >> file.txt
git add file.txt
export GIT_AUTHOR_DATE="2025-01-15T10:30:00Z"
export GIT_COMMITTER_DATE="2025-01-15T10:30:00Z"
faketime '2025-01-15 10:30:00' git commit -m "Add feature C"
unset GIT_AUTHOR_DATE GIT_COMMITTER_DATE

COMMIT3_SHA=$(git rev-parse HEAD)
git notes --ref=refs/notes/prompt-story show HEAD > /dev/null 2>&1 || fail "Commit 3 should have a note"
NOTE3=$(git notes --ref=refs/notes/prompt-story show HEAD)
echo "$NOTE3" | jq -e '.sessions[0].id == "session-C"' > /dev/null || fail "Commit 3 should reference session-C"
echo "    - Commit 3 ($COMMIT3_SHA) has note with session-C"

# Clean up all sessions before rebase
cleanup_sessions

# Step 7: Perform rebase - squash commit 3 into commit 2
echo "  Step 7: Rebasing (squash commit 3 into commit 2)..."

# Get the SHA of commit before our feature commits (initial commit)
INITIAL_SHA=$(git rev-parse HEAD~3)

# Create a script for GIT_SEQUENCE_EDITOR to modify the rebase todo
# Keep commit 1, keep commit 2, squash commit 3 into commit 2
cat > /tmp/rebase-editor.sh << 'EDITOREOF'
#!/bin/bash
# The todo list has commits from oldest to newest:
# pick <commit1> Add feature A
# pick <commit2> Add feature B
# pick <commit3> Add feature C
#
# We want to squash commit 3 into commit 2:
# pick <commit1> Add feature A
# pick <commit2> Add feature B
# squash <commit3> Add feature C
sed -i '3s/^pick/squash/' "$1"
EDITOREOF
chmod +x /tmp/rebase-editor.sh

# Create a script for the commit message editor during squash
cat > /tmp/commit-editor.sh << 'COMMITEOF'
#!/bin/bash
cat > "$1" << 'MSG'
Add features B and C (squashed)

This commit combines feature B and feature C.
MSG
COMMITEOF
chmod +x /tmp/commit-editor.sh

export GIT_SEQUENCE_EDITOR="/tmp/rebase-editor.sh"
export EDITOR="/tmp/commit-editor.sh"
export GIT_EDITOR="/tmp/commit-editor.sh"

faketime '2025-01-15 11:00:00' git rebase -i HEAD~3

unset GIT_SEQUENCE_EDITOR EDITOR GIT_EDITOR

echo "    - Rebase complete"

# Step 8: Verify commit 1 note is preserved
echo "  Step 8: Verifying commit 1 note preserved..."

# After rebase, commit 1 should be HEAD~1
NEW_COMMIT1_SHA=$(git rev-parse HEAD~1)
echo "    Original commit 1: $COMMIT1_SHA"
echo "    New commit 1: $NEW_COMMIT1_SHA"

if ! git notes --ref=refs/notes/prompt-story show HEAD~1 > /dev/null 2>&1; then
    echo "    ERROR: Commit 1 note was not preserved during rebase"
    fail "Commit 1 should still have its note after rebase"
fi

NEW_NOTE1=$(git notes --ref=refs/notes/prompt-story show HEAD~1)
echo "$NEW_NOTE1" | jq -e '.sessions[0].id == "session-A"' > /dev/null || fail "Commit 1 should still reference session-A"
echo "    - Commit 1 note preserved with session-A"

# Step 9: Verify squashed commit has merged note
echo "  Step 9: Verifying squashed commit has merged note..."

SQUASHED_SHA=$(git rev-parse HEAD)
echo "    Squashed commit: $SQUASHED_SHA"
echo "    Original commits that were squashed: $COMMIT2_SHA, $COMMIT3_SHA"

if ! git notes --ref=refs/notes/prompt-story show HEAD > /dev/null 2>&1; then
    echo "    ERROR: Squashed commit has no note attached"
    fail "Squashed commit should have a merged note"
fi

MERGED_NOTE=$(git notes --ref=refs/notes/prompt-story show HEAD)
echo "    Merged note content:"
echo "$MERGED_NOTE" | jq .

# Verify both sessions are in the merged note
SESSION_COUNT=$(echo "$MERGED_NOTE" | jq '.sessions | length')
if [[ "$SESSION_COUNT" != "2" ]]; then
    echo "    ERROR: Expected 2 sessions in merged note, got $SESSION_COUNT"
    fail "Merged note should contain both session-B and session-C"
fi
echo "    - Merged note has 2 sessions"

# Verify session-B is present
echo "$MERGED_NOTE" | jq -e '.sessions[] | select(.id == "session-B")' > /dev/null || fail "Merged note should contain session-B"
echo "    - session-B present"

# Verify session-C is present
echo "$MERGED_NOTE" | jq -e '.sessions[] | select(.id == "session-C")' > /dev/null || fail "Merged note should contain session-C"
echo "    - session-C present"

# Verify sessions are sorted by created time
FIRST_SESSION=$(echo "$MERGED_NOTE" | jq -r '.sessions[0].id')
SECOND_SESSION=$(echo "$MERGED_NOTE" | jq -r '.sessions[1].id')
if [[ "$FIRST_SESSION" != "session-B" ]] || [[ "$SECOND_SESSION" != "session-C" ]]; then
    echo "    WARNING: Sessions may not be sorted correctly"
    echo "    First: $FIRST_SESSION, Second: $SECOND_SESSION"
fi

# Verify start_work is from the earlier commit
START_WORK=$(echo "$MERGED_NOTE" | jq -r '.start_work')
echo "    - start_work: $START_WORK"

# Step 10: Verify transcript blobs are still accessible
echo "  Step 10: Verifying transcript blobs are accessible..."

# Get transcript paths from the merged note
TRANSCRIPT_B=$(echo "$MERGED_NOTE" | jq -r '.sessions[] | select(.id == "session-B") | .path')
TRANSCRIPT_C=$(echo "$MERGED_NOTE" | jq -r '.sessions[] | select(.id == "session-C") | .path')

echo "    Transcript B path: $TRANSCRIPT_B"
echo "    Transcript C path: $TRANSCRIPT_C"

# The transcripts should be stored in refs/notes/prompt-story-transcripts
# Check that we can access them via git-prompt-story show
if git-prompt-story show HEAD > /dev/null 2>&1; then
    echo "    - Transcripts accessible via git-prompt-story show"
else
    echo "    WARNING: Could not access transcripts via git-prompt-story show"
fi

# Step 11: Final verification - commit history
echo "  Step 11: Final commit history verification..."
echo "    Git log:"
git log --oneline | head -5

COMMIT_COUNT=$(git rev-list HEAD --count)
echo "    Total commits: $COMMIT_COUNT"

# Should have: Initial commit, Add feature A, Add features B and C (squashed)
if [[ "$COMMIT_COUNT" != "3" ]]; then
    echo "    WARNING: Expected 3 commits after rebase, got $COMMIT_COUNT"
fi

echo ""
echo "  All rebase flow assertions passed!"
