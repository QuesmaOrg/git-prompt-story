#!/bin/bash
set -euo pipefail
source /e2e/lib/helpers.sh

echo "[5/5] No Sessions"

# This test covers two scenarios:
# A) No sessions at all
# B) Session exists but is outside the work time window

# Clean up any previous sessions
cleanup_sessions

#######################################
# Scenario A: No sessions at all
#######################################
echo "  Scenario A: No sessions at all"

# Step 1: Create fresh test repo
echo "    Step 1: Creating test repo..."
rm -rf /workspace/test-repo
mkdir -p /workspace/test-repo
cd /workspace/test-repo
git init

# Step 2: Create initial commit at 09:00 (NO hooks yet)
echo "    Step 2: Creating initial commit at 09:00..."
echo "initial" > file.txt
git add file.txt
faketime '2025-01-15 09:00:00' git commit -m "Initial commit"

# Step 3: Install hooks
echo "    Step 3: Installing hooks..."
git-prompt-story install-hooks

# Step 4: Create feature commit at 10:30 (NO sessions)
echo "    Step 4: Creating feature commit at 10:30 (no sessions)..."
echo "feature code" >> file.txt
git add file.txt
export GIT_AUTHOR_DATE="2025-01-15T10:30:00Z"
export GIT_COMMITTER_DATE="2025-01-15T10:30:00Z"
faketime '2025-01-15 10:30:00' git commit -m "Add feature"
unset GIT_AUTHOR_DATE GIT_COMMITTER_DATE

# Step 5: Verify results
echo "    Step 5: Verifying results..."

COMMIT_MSG=$(git log -1 --format=%B)
echo "$COMMIT_MSG" | grep -q "Prompt-Story: none" || fail "Expected 'Prompt-Story: none' marker"
echo "      - Commit message says 'Prompt-Story: none'"

# No note should be attached when there are no sessions
if git notes --ref=refs/notes/prompt-story show HEAD > /dev/null 2>&1; then
    fail "Note should not be attached when no sessions"
fi
echo "      - No note attached (expected)"

# No pending file should remain
if [[ -f .git/PENDING-PROMPT-STORY ]]; then
    fail "PENDING-PROMPT-STORY file should not exist"
fi
echo "      - No PENDING-PROMPT-STORY file"

echo "    Scenario A passed!"
echo ""

#######################################
# Scenario B: Session outside time window
#######################################
echo "  Scenario B: Session outside time window"

# Clean up
cleanup_sessions

# Step 1: Create fresh test repo
echo "    Step 1: Creating test repo..."
rm -rf /workspace/test-repo
mkdir -p /workspace/test-repo
cd /workspace/test-repo
git init

# Step 2: Create initial commit at 09:00 (NO hooks yet)
echo "    Step 2: Creating initial commit at 09:00..."
echo "initial" > file.txt
git add file.txt
faketime '2025-01-15 09:00:00' git commit -m "Initial commit"

# Step 3: Install hooks
echo "    Step 3: Installing hooks..."
git-prompt-story install-hooks

# Step 4: Create mock session BEFORE the initial commit (08:00 - 08:30)
echo "    Step 4: Creating mock session (08:00 - 08:30) - before work window..."
create_mock_session "old-session" "2025-01-15T08:00:00Z" "2025-01-15T08:30:00Z"

# Step 5: Create feature commit at 10:30
echo "    Step 5: Creating feature commit at 10:30..."
echo "feature code" >> file.txt
git add file.txt
export GIT_AUTHOR_DATE="2025-01-15T10:30:00Z"
export GIT_COMMITTER_DATE="2025-01-15T10:30:00Z"
faketime '2025-01-15 10:30:00' git commit -m "Add feature"
unset GIT_AUTHOR_DATE GIT_COMMITTER_DATE

# Step 6: Verify results
echo "    Step 6: Verifying results..."

COMMIT_MSG=$(git log -1 --format=%B)
echo "$COMMIT_MSG" | grep -q "Prompt-Story: none" || fail "Expected 'Prompt-Story: none' marker (session outside window)"
echo "      - Commit message says 'Prompt-Story: none'"

# No note should be attached
if git notes --ref=refs/notes/prompt-story show HEAD > /dev/null 2>&1; then
    fail "Note should not be attached when session is outside window"
fi
echo "      - No note attached (expected)"

echo "    Scenario B passed!"

echo ""
echo "  All assertions passed!"
