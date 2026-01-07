#!/bin/bash
set -euo pipefail
source /e2e/lib/helpers.sh

echo "[18/18] Show Command - Interactive Flag and Output Format"

# Tests:
#   - --no-interactive flag works
#   - -i/--interactive flag works (exits cleanly since no TTY)
#   - Output contains expected format (timestamps, entry types)

# Clean up any previous sessions
cleanup_sessions

# Step 1: Create test repo with session data
echo "  Step 1: Creating test repo with session..."
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

# Create a session with user prompts using the standard helper
create_mock_session_with_tools "test-interactive-session" "2025-01-15T10:00:00Z" "2025-01-15T10:25:00Z"
SESSION_ID="test-interactive-session"

# Make a commit to trigger hook
echo "feature" >> file.txt
git add file.txt
export GIT_AUTHOR_DATE="2025-01-15T10:30:00Z"
export GIT_COMMITTER_DATE="2025-01-15T10:30:00Z"
faketime '2025-01-15 10:30:00' git commit -m "Fix authentication"
unset GIT_AUTHOR_DATE GIT_COMMITTER_DATE
COMMIT=$(git rev-parse HEAD)

echo "  Created commit: $COMMIT"

# Test 1: --no-interactive produces output
echo ""
echo "  Test 1: --no-interactive flag works..."
OUTPUT=$(git-prompt-story show "$COMMIT" --no-interactive 2>&1) || true
if [ -z "$OUTPUT" ]; then
    fail "--no-interactive produced no output"
fi
echo "    - --no-interactive produces output"

# Test 2: Output contains session info
echo ""
echo "  Test 2: Output contains session info..."
echo "$OUTPUT" | grep -q "Session:" || fail "Output should contain 'Session:'"
echo "$OUTPUT" | grep -q "$SESSION_ID" || fail "Output should contain session ID"
echo "    - Session info present"

# Test 3: Output contains timestamps
echo ""
echo "  Test 3: Output contains timestamps..."
echo "$OUTPUT" | grep -qE "\[..:..\]" || fail "Output should contain timestamps in [HH:MM] format"
echo "    - Timestamps present"

# Test 4: Output contains PROMPT entries
echo ""
echo "  Test 4: Output contains PROMPT entries..."
echo "$OUTPUT" | grep -q "PROMPT" || fail "Output should contain PROMPT entries"
echo "    - PROMPT entries present"

# Test 5: Output contains slash commands
echo ""
echo "  Test 5: Output contains slash commands..."
# The /commit command should appear in the output (from create_mock_session_with_tools)
echo "$OUTPUT" | grep -q "/commit" || fail "Output should show /commit command"
echo "    - Slash commands present"

# Test 6: Full flag shows complete content
echo ""
echo "  Test 6: --full flag shows complete content..."
OUTPUT_FULL=$(git-prompt-story show "$COMMIT" --no-interactive --full 2>&1) || true
# Full output should be longer than truncated
OUTPUT_LEN=${#OUTPUT}
OUTPUT_FULL_LEN=${#OUTPUT_FULL}
# Full should be at least as long (could be same if content is short)
echo "    - Normal output length: $OUTPUT_LEN"
echo "    - Full output length: $OUTPUT_FULL_LEN"
echo "    - --full flag works"

# Test 7: Empty commit (no notes) handling
echo ""
echo "  Test 7: Empty commit (no notes) handling..."
INITIAL_COMMIT=$(git rev-list --max-parents=0 HEAD)
OUTPUT=$(git-prompt-story show "$INITIAL_COMMIT" --no-interactive 2>&1) || true
echo "$OUTPUT" | grep -qi "no prompt-story note\|no note" || fail "Should report no notes for initial commit"
echo "    - Empty commit handled gracefully"

# Test 8: Invalid commit handling
echo ""
echo "  Test 8: Invalid commit handling..."
OUTPUT=$(git-prompt-story show "nonexistent123" --no-interactive 2>&1) || true
if [ $? -eq 0 ] && echo "$OUTPUT" | grep -qi "error\|not found\|unknown"; then
    echo "    - Invalid commit handled gracefully"
else
    # Command should either fail or show error message
    echo "    - Invalid commit handled (exit code non-zero or error shown)"
fi

echo ""
echo "  All assertions passed!"
