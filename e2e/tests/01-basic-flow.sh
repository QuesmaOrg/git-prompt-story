#!/bin/bash
set -euo pipefail
source /e2e/lib/helpers.sh

echo "[1/5] Basic Hook Flow"

# Clean up any previous sessions
cleanup_sessions

# Step 1: Create fresh test repo
echo "  Step 1: Creating test repo..."
rm -rf /workspace/test-repo
mkdir -p /workspace/test-repo
cd /workspace/test-repo
git init

# Step 2: Create initial commit at controlled time (09:00) - NO hooks yet
echo "  Step 2: Creating initial commit at 09:00..."
echo "initial" > file.txt
git add file.txt
faketime '2025-01-15 09:00:00' git commit -m "Initial commit"

# Verify initial commit time
INITIAL_COMMIT_TIME=$(git log -1 --format=%ci)
echo "  Initial commit time: $INITIAL_COMMIT_TIME"

# Step 3: Install hooks
echo "  Step 3: Installing hooks..."
git-prompt-story install-hooks

# Verify hooks exist
test -x .git/hooks/prepare-commit-msg || fail "prepare-commit-msg hook not installed"
test -x .git/hooks/post-commit || fail "post-commit hook not installed"
echo "  Hooks installed successfully"

# Step 4: Create mock session with timestamps BETWEEN commits (09:15 - 10:25)
echo "  Step 4: Creating mock session (09:15 - 10:25)..."
create_mock_session "test-session-1" "2025-01-15T09:15:00Z" "2025-01-15T10:25:00Z"

# Step 5: Add file and commit at controlled time (10:30)
echo "  Step 5: Creating feature commit at 10:30..."
echo "feature code" >> file.txt
git add file.txt
# Use both faketime (for git) and GIT_COMMITTER_DATE (for Go code in hooks)
export GIT_AUTHOR_DATE="2025-01-15T10:30:00Z"
export GIT_COMMITTER_DATE="2025-01-15T10:30:00Z"
faketime '2025-01-15 10:30:00' git commit -m "Add feature"
unset GIT_AUTHOR_DATE GIT_COMMITTER_DATE

# Verify feature commit time
FEATURE_COMMIT_TIME=$(git log -1 --format=%ci)
echo "  Feature commit time: $FEATURE_COMMIT_TIME"

# Step 6: Verify results
echo "  Step 6: Verifying results..."

echo "    Checking commit message..."
COMMIT_MSG=$(git log -1 --format=%B)
echo "$COMMIT_MSG" | grep -q "Prompt-Story: Used Claude Code" || fail "Missing Prompt-Story trailer in commit message"
echo "    - Commit message contains Prompt-Story trailer"

echo "    Checking note attachment..."
git notes --ref=refs/notes/commits show HEAD > /dev/null 2>&1 || fail "No note attached to HEAD"
echo "    - Note is attached to HEAD"

echo "    Checking note content..."
NOTE=$(git notes --ref=refs/notes/commits show HEAD)
echo "$NOTE" | yq -e '.v == 1' > /dev/null || fail "Invalid note version"
echo "    - Note version is 1"

echo "$NOTE" | yq -e '.sessions | length == 1' > /dev/null || fail "Expected exactly 1 session"
echo "    - Note contains 1 session"

echo "$NOTE" | yq -e '.sessions[0] == "claude-code/test-session-1.jsonl"' > /dev/null || fail "Wrong session path"
echo "    - Session path is correct"

echo "    Verifying start_work matches previous commit time..."
echo "$NOTE" | yq -e '.start_work == "2025-01-15T09:00:00Z"' > /dev/null || fail "start_work should be previous commit time (09:00)"
echo "    - start_work = 2025-01-15T09:00:00Z (previous commit)"

echo "    Checking transcript storage..."
git cat-file -e "refs/notes/prompt-story-transcripts:claude-code/test-session-1.jsonl" 2>/dev/null || fail "Transcript not stored in refs/notes/prompt-story-transcripts"
echo "    - Transcript stored at refs/notes/prompt-story-transcripts:claude-code/test-session-1.jsonl"

echo ""
echo "  All assertions passed!"
