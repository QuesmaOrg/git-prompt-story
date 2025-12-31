#!/bin/bash
set -euo pipefail
source /e2e/lib/helpers.sh

echo "[8/8] Cloud Integration"

# Clean up any previous state
cleanup_sessions
cleanup_mock_cloud 2>/dev/null || true

# Step 1: Setup mock credentials and cloud session
echo "  Step 1: Setting up mock cloud environment..."
setup_mock_credentials
create_mock_cloud_session "session_cloud123" "Test Cloud Session" "main" "2025-01-15T09:00:00Z" "2025-01-15T10:00:00Z"

# Step 2: Start mock API server
echo "  Step 2: Starting mock API server..."
start_mock_api 9999

# Step 3: Create test repository
echo "  Step 3: Creating test repo..."
rm -rf /workspace/test-repo
mkdir -p /workspace/test-repo
cd /workspace/test-repo
git init

# Step 4: Create initial commit
echo "  Step 4: Creating initial commit..."
echo "initial" > file.txt
git add file.txt
git commit -m "Initial commit"

# Step 5: Test list-cloud command
echo "  Step 5: Testing list-cloud command..."
LIST_OUTPUT=$(git-prompt-story list-cloud 2>&1) || fail "list-cloud command failed"
echo "$LIST_OUTPUT" | grep -q "session_cloud123" || fail "list-cloud should show mock session ID"
echo "$LIST_OUTPUT" | grep -q "Test Cloud Session" || fail "list-cloud should show session title"
echo "    - list-cloud shows mock sessions"

# Step 6: Test annotate-cloud with session ID
echo "  Step 6: Testing annotate-cloud --session-id..."
git-prompt-story annotate-cloud HEAD --session-id=session_cloud123 || fail "annotate-cloud failed"
echo "    - annotate-cloud command succeeded"

# Step 7: Verify note attachment
echo "  Step 7: Verifying note attachment..."
git notes --ref=refs/notes/prompt-story show HEAD > /dev/null 2>&1 || fail "No note attached to HEAD"
echo "    - Note is attached to HEAD"

# Step 8: Verify note content
echo "  Step 8: Verifying note content..."
NOTE=$(git notes --ref=refs/notes/prompt-story show HEAD)

echo "$NOTE" | jq -e '.v == 1' > /dev/null || fail "Invalid note version"
echo "    - Note version is 1"

echo "$NOTE" | jq -e '.sessions | length == 1' > /dev/null || fail "Expected exactly 1 session"
echo "    - Note contains 1 session"

echo "$NOTE" | jq -e '.sessions[0].tool == "claude-cloud"' > /dev/null || fail "Session tool should be claude-cloud"
echo "    - Session tool is claude-cloud"

echo "$NOTE" | jq -e '.sessions[0].id == "session_cloud123"' > /dev/null || fail "Wrong session ID"
echo "    - Session ID is correct"

# Step 9: Verify transcript storage
echo "  Step 9: Verifying transcript storage..."
git ls-tree refs/notes/prompt-story-transcripts 2>/dev/null | grep -q "claude-cloud" || fail "No claude-cloud subtree in transcripts"
echo "    - claude-cloud subtree exists"

git cat-file -e "refs/notes/prompt-story-transcripts:claude-cloud/session_cloud123.jsonl" 2>/dev/null || fail "Transcript not stored"
echo "    - Transcript stored at claude-cloud/session_cloud123.jsonl"

# Step 10: Verify transcript content
echo "  Step 10: Verifying transcript content..."
TRANSCRIPT=$(git cat-file -p "refs/notes/prompt-story-transcripts:claude-cloud/session_cloud123.jsonl")
echo "$TRANSCRIPT" | grep -q "Test prompt from mock" || fail "Transcript missing user message"
echo "    - Transcript contains user message"

# Cleanup
echo "  Cleaning up..."
stop_mock_api

echo ""
echo "  All assertions passed!"
