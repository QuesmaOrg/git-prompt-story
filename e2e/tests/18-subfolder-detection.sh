#!/bin/bash
set -euo pipefail
source /e2e/lib/helpers.sh

echo "[18] Subfolder Session Detection"

# Test that sessions started from various locations are correctly detected
#
# Timeline:
#   09:00       - Initial commit
#   09:15-10:25 - Session from repo root - SHOULD BE DETECTED
#   09:15-10:25 - Session from /workspace/test-repo/src subfolder - SHOULD BE DETECTED
#   09:15-10:25 - Session from /workspace (external/parent dir) editing repo files - SHOULD BE DETECTED
#   09:15-10:25 - Session from /workspace/test-repo-v2 (different repo) - should NOT be detected
#   10:30       - Feature commit

# Clean up any previous sessions
cleanup_sessions

# Step 1: Create fresh test repo
echo "  Step 1: Creating test repo..."
rm -rf /workspace/test-repo
rm -rf /workspace/test-repo-v2
mkdir -p /workspace/test-repo/src
mkdir -p /workspace/test-repo-v2
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

# Step 4: Create mock sessions
echo "  Step 4: Creating mock sessions..."

# Session 1: From repo root - SHOULD BE DETECTED
create_mock_session "session-root" "2025-01-15T09:15:00Z" "2025-01-15T10:25:00Z"

# Session 2: From src subfolder - SHOULD BE DETECTED
create_mock_session_for_subfolder "src" "session-subfolder" "2025-01-15T09:15:00Z" "2025-01-15T10:25:00Z"

# Session 3: From external/parent directory editing files in repo - SHOULD BE DETECTED
# This tests that sessions started from /workspace that edit /workspace/test-repo files are detected
create_mock_session_for_external_dir "/workspace" "/workspace/test-repo" "session-external" "2025-01-15T09:15:00Z" "2025-01-15T10:25:00Z"

# Session 4: From similarly-named but different repo - should NOT be detected
# This tests that we don't match /workspace/test-repo-v2 when looking for /workspace/test-repo
create_mock_session_for_path "/workspace/test-repo-v2" "session-other-repo" "2025-01-15T09:15:00Z" "2025-01-15T10:25:00Z"

# List created sessions for debugging
echo "  Session directories created:"
ls -la ~/.claude/projects/ | grep workspace

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

echo "    Checking correct sessions are detected..."
NOTE=$(git notes --ref=refs/notes/prompt-story show HEAD)

# Should have exactly 3 sessions (root, subfolder, and external)
SESSION_COUNT=$(echo "$NOTE" | jq '.sessions | length')
if [[ "$SESSION_COUNT" != "3" ]]; then
    echo "    ERROR: Expected 3 sessions, got $SESSION_COUNT"
    echo "    Note content:"
    echo "$NOTE" | jq .
    fail "Wrong number of sessions detected"
fi
echo "    - Exactly 3 sessions detected"

# Session-root should be detected
echo "$NOTE" | jq -e '.sessions[] | select(.id == "session-root")' > /dev/null || fail "session-root not detected"
echo "    - session-root (repo root) detected"

# Session-subfolder should be detected
echo "$NOTE" | jq -e '.sessions[] | select(.id == "session-subfolder")' > /dev/null || fail "session-subfolder not detected"
echo "    - session-subfolder (src/) detected"

# Session-external should be detected (from parent dir /workspace editing /workspace/test-repo files)
echo "$NOTE" | jq -e '.sessions[] | select(.id == "session-external")' > /dev/null || fail "session-external not detected"
echo "    - session-external (/workspace editing repo) detected"

# Session-other-repo should NOT be detected
if echo "$NOTE" | jq -e '.sessions[] | select(.id == "session-other-repo")' > /dev/null 2>&1; then
    echo "    ERROR: session-other-repo (from /workspace/test-repo-v2) should NOT be detected"
    echo "    Note content:"
    echo "$NOTE" | jq .
    fail "session-other-repo incorrectly detected"
fi
echo "    - session-other-repo (test-repo-v2) correctly NOT detected"

echo ""
echo "  All assertions passed!"
