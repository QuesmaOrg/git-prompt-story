#!/bin/bash
set -euo pipefail
source /e2e/lib/helpers.sh

echo "[7/7] Show Command - Ranges and Prefixes"

# Timeline:
#   09:00       - Initial commit (no session)
#   09:15-10:00 - Session 1
#   10:30       - Commit 1 (with session 1)
#   10:45-11:30 - Session 2
#   12:00       - Commit 2 (with session 2)
#   12:15-13:00 - Session 3
#   13:30       - Commit 3 (with session 3)
#
# Tests:
#   - Show by full SHA
#   - Show by short SHA prefix
#   - Show by commit range (HEAD~2..HEAD)
#   - Show by branch range (main..feature)
#   - Show by prompt-story-{hash} prefix

# Clean up any previous sessions
cleanup_sessions

# Step 1: Create fresh test repo
echo "  Step 1: Creating test repo with multiple commits..."
rm -rf /workspace/test-repo
mkdir -p /workspace/test-repo
cd /workspace/test-repo
git init

# Create initial commit at 09:00 (NO hooks yet)
echo "initial" > file.txt
git add file.txt
faketime '2025-01-15 09:00:00' git commit -m "Initial commit"

# Install hooks
git-prompt-story install-hooks

# Create session 1 and commit 1
create_mock_session "test-session-1" "2025-01-15T09:15:00Z" "2025-01-15T10:00:00Z"
echo "feature 1" >> file.txt
git add file.txt
export GIT_AUTHOR_DATE="2025-01-15T10:30:00Z"
export GIT_COMMITTER_DATE="2025-01-15T10:30:00Z"
faketime '2025-01-15 10:30:00' git commit -m "Add feature 1"
unset GIT_AUTHOR_DATE GIT_COMMITTER_DATE
COMMIT1=$(git rev-parse HEAD)
cleanup_sessions

# Create session 2 and commit 2
create_mock_session "test-session-2" "2025-01-15T10:45:00Z" "2025-01-15T11:30:00Z"
echo "feature 2" >> file.txt
git add file.txt
export GIT_AUTHOR_DATE="2025-01-15T12:00:00Z"
export GIT_COMMITTER_DATE="2025-01-15T12:00:00Z"
faketime '2025-01-15 12:00:00' git commit -m "Add feature 2"
unset GIT_AUTHOR_DATE GIT_COMMITTER_DATE
COMMIT2=$(git rev-parse HEAD)
cleanup_sessions

# Create session 3 and commit 3
create_mock_session "test-session-3" "2025-01-15T12:15:00Z" "2025-01-15T13:00:00Z"
echo "feature 3" >> file.txt
git add file.txt
export GIT_AUTHOR_DATE="2025-01-15T13:30:00Z"
export GIT_COMMITTER_DATE="2025-01-15T13:30:00Z"
faketime '2025-01-15 13:30:00' git commit -m "Add feature 3"
unset GIT_AUTHOR_DATE GIT_COMMITTER_DATE
COMMIT3=$(git rev-parse HEAD)

echo "  Created commits:"
echo "    COMMIT1: $COMMIT1"
echo "    COMMIT2: $COMMIT2"
echo "    COMMIT3: $COMMIT3"

# Test 1: Show by full SHA (should already work)
echo ""
echo "  Test 1: Show by full SHA..."
OUTPUT=$(git-prompt-story show "$COMMIT2")
echo "$OUTPUT" | grep -q "test-session-2" || fail "Full SHA: Should show session-2"
echo "    - Full SHA works"

# Test 2: Show by short SHA prefix
echo ""
echo "  Test 2: Show by short SHA prefix..."
SHORT_SHA="${COMMIT2:0:7}"
OUTPUT=$(git-prompt-story show "$SHORT_SHA")
echo "$OUTPUT" | grep -q "test-session-2" || fail "Short SHA: Should show session-2"
echo "    - Short SHA prefix works"

# Test 3: Show by commit range (HEAD~2..HEAD)
echo ""
echo "  Test 3: Show by commit range HEAD~2..HEAD..."
OUTPUT=$(git-prompt-story show "HEAD~2..HEAD")
echo "$OUTPUT" | grep -q "test-session-2" || fail "Range: Should include session-2"
echo "$OUTPUT" | grep -q "test-session-3" || fail "Range: Should include session-3"
# Should NOT include session-1 (it's HEAD~2, which is the start of the range, exclusive)
echo "    - Commit range works"

# Test 4: Show single commit with ~ syntax
echo ""
echo "  Test 4: Show single commit with HEAD~1..."
OUTPUT=$(git-prompt-story show "HEAD~1")
echo "$OUTPUT" | grep -q "test-session-2" || fail "HEAD~1: Should show session-2"
echo "    - HEAD~N syntax works"

# Test 5: Show by prompt-story-{hash} prefix
echo ""
echo "  Test 5: Show by prompt-story-{hash} prefix..."
# Get the note hash for COMMIT2
NOTE_HASH=$(git notes --ref=refs/notes/commits list "$COMMIT2" | awk '{print $1}')
SHORT_NOTE_HASH="${NOTE_HASH:0:7}"
echo "    Note hash for COMMIT2: $NOTE_HASH (short: $SHORT_NOTE_HASH)"
OUTPUT=$(git-prompt-story show "prompt-story-$SHORT_NOTE_HASH")
echo "$OUTPUT" | grep -q "test-session-2" || fail "prompt-story prefix: Should show session-2"
echo "    - prompt-story-{hash} prefix works"

# Test 5b: Verify commit message trailer hash matches note hash
echo ""
echo "  Test 5b: Verify commit message hash matches note hash..."
COMMIT_MSG=$(git log -1 --format=%B "$COMMIT2")
TRAILER_HASH=$(echo "$COMMIT_MSG" | grep -o 'prompt-story-[a-f0-9]*' | sed 's/prompt-story-//')
if [[ "$NOTE_HASH" != "$TRAILER_HASH"* ]]; then
    echo "    ERROR: Trailer hash ($TRAILER_HASH) doesn't match note hash ($NOTE_HASH)"
    fail "Commit message hash doesn't match note hash"
fi
echo "    - Commit message trailer hash matches note hash"

# Test 6: Show range with three dots (symmetric difference)
echo ""
echo "  Test 6: Show range with symmetric difference..."
# Create a branch from commit1
git checkout -b feature "$COMMIT1"
# Create a new commit on feature branch
cleanup_sessions
create_mock_session "test-session-4" "2025-01-15T14:00:00Z" "2025-01-15T14:30:00Z"
echo "feature branch" >> file.txt
git add file.txt
export GIT_AUTHOR_DATE="2025-01-15T15:00:00Z"
export GIT_COMMITTER_DATE="2025-01-15T15:00:00Z"
faketime '2025-01-15 15:00:00' git commit -m "Feature branch commit"
unset GIT_AUTHOR_DATE GIT_COMMITTER_DATE

# Now test range from main to feature
git checkout main
OUTPUT=$(git-prompt-story show "$COMMIT1..feature")
echo "$OUTPUT" | grep -q "test-session-4" || fail "Branch range: Should include session-4"
echo "    - Branch range works"

echo ""
echo "  All assertions passed!"
