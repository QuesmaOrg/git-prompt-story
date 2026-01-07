#!/bin/bash
set -euo pipefail
source /e2e/lib/helpers.sh

echo "[19] Show Command - Redaction and Session Clearing"

# Tests:
#   - --clear-session flag clears session in both git notes and local file
#   - --redact-message flag redacts specific message with <REDACTED BY USER>
#   - Other sessions are not affected when clearing one session

cleanup_sessions

# Step 1: Create test repo with session data
echo "  Step 1: Creating test repo with sessions..."
rm -rf /workspace/test-repo
mkdir -p /workspace/test-repo
cd /workspace/test-repo
git init

# Create initial commit
echo "initial" > file.txt
git add file.txt
faketime '2025-01-15 09:00:00' git commit -m "Initial commit"

# Install hooks
git-prompt-story install-hooks

# Create two sessions - one to clear, one to keep
create_mock_session "session-to-clear" "2025-01-15T10:00:00Z" "2025-01-15T10:10:00Z"
create_mock_session "session-to-keep" "2025-01-15T10:05:00Z" "2025-01-15T10:15:00Z"

# Make a commit to trigger hook (attaches notes)
echo "feature" >> file.txt
git add file.txt
export GIT_AUTHOR_DATE="2025-01-15T10:30:00Z"
export GIT_COMMITTER_DATE="2025-01-15T10:30:00Z"
faketime '2025-01-15 10:30:00' git commit -m "Add feature"
unset GIT_AUTHOR_DATE GIT_COMMITTER_DATE

COMMIT=$(git rev-parse HEAD)
echo "  Created commit: $COMMIT"

# Test 1: Verify both sessions exist in transcript refs
echo ""
echo "  Test 1: Verify sessions exist before clearing..."
TRANSCRIPT1=$(git cat-file -p "refs/notes/prompt-story-transcripts:claude-code/session-to-clear.jsonl" 2>/dev/null) || fail "session-to-clear should exist"
TRANSCRIPT2=$(git cat-file -p "refs/notes/prompt-story-transcripts:claude-code/session-to-keep.jsonl" 2>/dev/null) || fail "session-to-keep should exist"
echo "$TRANSCRIPT1" | grep -q "How do I implement feature X" || fail "session-to-clear should have content"
echo "$TRANSCRIPT2" | grep -q "How do I implement feature X" || fail "session-to-keep should have content"
echo "    - Both sessions have content"

# Test 2: Clear one session
echo ""
echo "  Test 2: Clear session..."
git-prompt-story show --clear-session "claude-code/session-to-clear"
echo "    - clear-session command succeeded"

# Test 3: Verify cleared session is empty
echo ""
echo "  Test 3: Verify cleared session is empty..."
TRANSCRIPT1=$(git cat-file -p "refs/notes/prompt-story-transcripts:claude-code/session-to-clear.jsonl" 2>/dev/null) || true
if [ -n "$TRANSCRIPT1" ]; then
    fail "session-to-clear should be empty after clearing"
fi
echo "    - Cleared session is empty in git notes"

# Test 4: Verify other session is untouched
echo ""
echo "  Test 4: Verify other session is untouched..."
TRANSCRIPT2=$(git cat-file -p "refs/notes/prompt-story-transcripts:claude-code/session-to-keep.jsonl" 2>/dev/null) || fail "session-to-keep should still exist"
echo "$TRANSCRIPT2" | grep -q "How do I implement feature X" || fail "session-to-keep should still have content"
echo "    - Other session still has content"

# Test 5: Verify local file is also cleared
echo ""
echo "  Test 5: Verify local file is cleared..."
LOCAL_FILE="$HOME/.claude/projects/-workspace-test-repo/session-to-clear.jsonl"
if [ -s "$LOCAL_FILE" ]; then
    fail "Local file should be empty after clearing"
fi
echo "    - Local file is empty"

# Test 6: Message redaction (use session-to-keep which is already stored)
echo ""
echo "  Test 6: Message redaction..."

# Verify session exists before redaction
TRANSCRIPT3=$(git cat-file -p "refs/notes/prompt-story-transcripts:claude-code/session-to-keep.jsonl" 2>/dev/null) || fail "session-to-keep should exist"
echo "$TRANSCRIPT3" | grep -q "How do I implement feature X" || fail "session-to-keep should have original content"

# Redact the user message (timestamp is 2025-01-15T10:05:00Z from session creation)
git-prompt-story show --redact-message "claude-code/session-to-keep@2025-01-15T10:05:00Z"
echo "    - redact-message command succeeded"

# Verify message is redacted
TRANSCRIPT3=$(git cat-file -p "refs/notes/prompt-story-transcripts:claude-code/session-to-keep.jsonl" 2>/dev/null) || fail "session should still exist"
# JSON escapes < and > as \u003c and \u003e
if echo "$TRANSCRIPT3" | grep -q "How do I implement feature X"; then
    fail "Original message should be redacted"
fi
if ! echo "$TRANSCRIPT3" | grep -qE '(REDACTED BY USER|\\u003cREDACTED BY USER\\u003e)'; then
    fail "Redaction placeholder should be present"
fi
echo "    - Message redacted with placeholder"

# Test 7: Assistant message should be untouched
echo ""
echo "  Test 7: Verify assistant message is untouched..."
echo "$TRANSCRIPT3" | grep -q "Here's how to implement feature X" || fail "Assistant message should be untouched"
echo "    - Assistant message preserved"

echo ""
echo "  All assertions passed!"
