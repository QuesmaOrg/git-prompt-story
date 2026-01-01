#!/bin/bash
set -euo pipefail
source /e2e/lib/helpers.sh

echo "[6/6] No User Messages in Work Period"

# Timeline:
#   09:00       - Initial commit (start of work period)
#   07:30       - Session user message (BEFORE work period)
#   09:15       - Session assistant response (WITHIN work period - updates modified time)
#   09:30       - Feature commit
#
# Expected: "Prompt-Story: none" since no user messages exist in work period (09:00-09:30)
# Bug behavior: "Prompt-Story: Used Claude Code" because session.modified (09:15) is in work period

# Clean up any previous sessions
cleanup_sessions

# Step 1: Create fresh test repo
echo "  Step 1: Creating test repo..."
rm -rf /workspace/test-repo
mkdir -p /workspace/test-repo
cd /workspace/test-repo
git init

# Step 2: Create initial commit at 09:00
echo "  Step 2: Creating initial commit at 09:00..."
echo "initial" > file.txt
git add file.txt
faketime '2025-01-15 09:00:00' git commit -m "Initial commit"

INITIAL_COMMIT_TIME=$(git log -1 --format=%ci)
echo "  Initial commit time: $INITIAL_COMMIT_TIME"

# Step 3: Install hooks
echo "  Step 3: Installing hooks..."
git-prompt-story install-hooks

# Step 4: Create stale session - user message at 07:30 (BEFORE work period), assistant at 09:15 (WITHIN)
echo "  Step 4: Creating stale session..."
create_stale_session "stale-session-1" "2025-01-15T07:30:00Z" "2025-01-15T09:15:00Z"

# Step 5: Create feature commit at 09:30
echo "  Step 5: Creating feature commit at 09:30..."
echo "feature code" >> file.txt
git add file.txt
export GIT_AUTHOR_DATE="2025-01-15T09:30:00Z"
export GIT_COMMITTER_DATE="2025-01-15T09:30:00Z"
faketime '2025-01-15 09:30:00' git commit -m "Add feature"
unset GIT_AUTHOR_DATE GIT_COMMITTER_DATE

FEATURE_COMMIT_TIME=$(git log -1 --format=%ci)
echo "  Feature commit time: $FEATURE_COMMIT_TIME"

# Step 6: Verify results
echo "  Step 6: Verifying results..."

echo "    Checking commit message..."
COMMIT_MSG=$(git log -1 --format=%B)
echo "    Commit message: $(echo "$COMMIT_MSG" | grep "Prompt-Story:" || echo "no marker")"

# This is the key assertion: no user messages in work period means "Prompt-Story: none"
if echo "$COMMIT_MSG" | grep -q "Prompt-Story: none"; then
    echo "    - PASS: Commit correctly says 'Prompt-Story: none'"
else
    echo "    - FAIL: Expected 'Prompt-Story: none' but got:"
    echo "$COMMIT_MSG" | grep "Prompt-Story:" || echo "      (no Prompt-Story marker found)"
    fail "Should not attach session when no user messages in work period"
fi

echo "    Checking no note attached..."
if git notes --ref=refs/notes/prompt-story show HEAD > /dev/null 2>&1; then
    echo "    - FAIL: Note should NOT be attached when no user messages"
    fail "Note attached when it should not be"
else
    echo "    - PASS: No note attached (expected)"
fi

echo ""
echo "  All assertions passed!"
