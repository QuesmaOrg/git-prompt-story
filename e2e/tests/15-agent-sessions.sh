#!/bin/bash
set -euo pipefail
source /e2e/lib/helpers.sh

echo "[15/15] Agent Session Handling Test"

# This test validates that agent sessions are:
# 1. Captured alongside main sessions
# 2. Detected correctly by ID pattern (agent-* prefix)
# 3. Displayed with proper badges in HTML output
# 4. Counted separately from main session prompts
# 5. Toggleable in HTML output
#
# Timeline:
#   09:00       - Initial commit (no hooks)
#   09:15       - Install hooks
#   09:20-09:25 - Main session active
#   09:22-09:23 - Agent session 1 (explore1) spawned from main
#   09:24-09:25 - Agent session 2 (explore2) spawned from main
#   09:30       - Commit with main + agent sessions

# Clean up any previous sessions
cleanup_sessions

# Step 1: Create fresh test repo
echo "  Step 1: Creating test repo..."
rm -rf /workspace/test-repo
mkdir -p /workspace/test-repo
cd /workspace/test-repo
git init

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

# Step 4: Create main session (UUID format) and agent sessions (agent-* format)
echo "  Step 4: Creating main session and agent sessions..."

# Main session: 09:20 - 09:25, UUID format
create_mock_session "fb813892-a738-4fc4-bcf8-b6f175a27a93" "2025-01-15T09:20:00Z" "2025-01-15T09:25:00Z"

# Agent session 1: 09:22 - 09:23, agent-* format
create_mock_agent_session "explore1" "2025-01-15T09:22:00Z" "2025-01-15T09:23:00Z"

# Agent session 2: 09:24 - 09:25, agent-* format
create_mock_agent_session "explore2" "2025-01-15T09:24:00Z" "2025-01-15T09:25:00Z"

# Step 5: Create commit with all sessions
echo "  Step 5: Creating commit with main + agent sessions..."
echo "feature with agents" >> file.txt
git add file.txt
export GIT_AUTHOR_DATE="2025-01-15T09:30:00Z"
export GIT_COMMITTER_DATE="2025-01-15T09:30:00Z"
faketime '2025-01-15 09:30:00' git commit -m "Add feature with agent assistance"
unset GIT_AUTHOR_DATE GIT_COMMITTER_DATE

COMMIT_SHA=$(git rev-parse HEAD)
echo "    - Commit SHA: $COMMIT_SHA"

# Step 6: Verify note contains all three sessions
echo "  Step 6: Verifying note contains all sessions..."

NOTE=$(git notes --ref=refs/notes/prompt-story show HEAD)
SESSION_COUNT=$(echo "$NOTE" | jq '.sessions | length')
echo "    - Session count in note: $SESSION_COUNT"

if [[ "$SESSION_COUNT" != "3" ]]; then
    echo "    ERROR: Expected 3 sessions, got $SESSION_COUNT"
    echo "$NOTE" | jq .
    fail "Note should contain 3 sessions (1 main + 2 agents)"
fi
echo "    - Note has 3 sessions"

# Verify session IDs
echo "$NOTE" | jq -e '.sessions[] | select(.id == "fb813892-a738-4fc4-bcf8-b6f175a27a93")' > /dev/null || fail "Should have main session"
echo "    - Main session found (UUID format)"

echo "$NOTE" | jq -e '.sessions[] | select(.id == "agent-explore1")' > /dev/null || fail "Should have agent-explore1"
echo "    - Agent session 1 found (agent-explore1)"

echo "$NOTE" | jq -e '.sessions[] | select(.id == "agent-explore2")' > /dev/null || fail "Should have agent-explore2"
echo "    - Agent session 2 found (agent-explore2)"

# Step 7: Verify pr summary counts
echo "  Step 7: Verifying pr summary output..."

# Use HEAD~1..HEAD as commit range (from initial commit to current)
SUMMARY=$(git-prompt-story pr summary HEAD~1..HEAD --format=json)
echo "$SUMMARY" | jq .

# Check total_agent_sessions
AGENT_SESSIONS=$(echo "$SUMMARY" | jq '.total_agent_sessions')
if [[ "$AGENT_SESSIONS" != "2" ]]; then
    echo "    ERROR: Expected total_agent_sessions=2, got $AGENT_SESSIONS"
    fail "pr summary should report 2 agent sessions"
fi
echo "    - total_agent_sessions: 2"

# Check total_agent_prompts (2 agents x 1 user prompt each = 2)
AGENT_PROMPTS=$(echo "$SUMMARY" | jq '.total_agent_prompts')
if [[ "$AGENT_PROMPTS" != "2" ]]; then
    echo "    ERROR: Expected total_agent_prompts=2, got $AGENT_PROMPTS"
    fail "pr summary should report 2 agent prompts"
fi
echo "    - total_agent_prompts: 2"

# Check total_user_prompts (main session only = 1)
USER_PROMPTS=$(echo "$SUMMARY" | jq '.total_user_prompts')
if [[ "$USER_PROMPTS" != "1" ]]; then
    echo "    ERROR: Expected total_user_prompts=1, got $USER_PROMPTS"
    fail "pr summary should report 1 user prompt (main session only)"
fi
echo "    - total_user_prompts: 1 (main session only)"

# Step 8: Verify pr html output contains badges and toggle
echo "  Step 8: Verifying pr html output..."

mkdir -p /tmp/ci-output
git-prompt-story pr html HEAD~1..HEAD --output-dir=/tmp/ci-output

# Check commit page for agent badges
COMMIT_SHORT=$(git rev-parse --short HEAD)
COMMIT_PAGE="/tmp/ci-output/${COMMIT_SHORT}.html"

if [[ ! -f "$COMMIT_PAGE" ]]; then
    echo "    ERROR: Commit page not found: $COMMIT_PAGE"
    fail "pr html should generate commit page"
fi

# Check for badge classes
if ! grep -q 'class="badge main"' "$COMMIT_PAGE"; then
    echo "    ERROR: Missing 'badge main' class in HTML"
    fail "HTML should have main session badge"
fi
echo "    - Found 'badge main' class"

if ! grep -q 'class="badge agent"' "$COMMIT_PAGE"; then
    echo "    ERROR: Missing 'badge agent' class in HTML"
    fail "HTML should have agent session badges"
fi
echo "    - Found 'badge agent' class"

# Check for agent toggle checkbox
if ! grep -q 'id="toggle-agent-sessions"' "$COMMIT_PAGE"; then
    echo "    ERROR: Missing agent sessions toggle"
    fail "HTML should have agent sessions toggle"
fi
echo "    - Found agent sessions toggle"

# Check for data-is-agent attributes
if ! grep -q 'data-is-agent="true"' "$COMMIT_PAGE"; then
    echo "    ERROR: Missing data-is-agent='true' attribute"
    fail "HTML should have data-is-agent attribute for agent sessions"
fi
echo "    - Found data-is-agent='true' attribute"

if ! grep -q 'data-is-agent="false"' "$COMMIT_PAGE"; then
    echo "    ERROR: Missing data-is-agent='false' attribute"
    fail "HTML should have data-is-agent attribute for main sessions"
fi
echo "    - Found data-is-agent='false' attribute"

# Step 9: Verify markdown output shows only main session prompts
echo "  Step 9: Verifying markdown output..."

MARKDOWN=$(git-prompt-story pr summary HEAD~1..HEAD --format=markdown)
echo "$MARKDOWN"

# Should show only main session prompt count (no agent indicator)
if echo "$MARKDOWN" | grep -q "(+[0-9]* agent)"; then
    echo "    ERROR: Markdown should NOT show agent count in User Prompts"
    fail "pr summary markdown should not indicate agent prompts"
fi
echo "    - User Prompts shows main session only (no agent indicator)"

# Verify user prompts header shows 1 (main session only)
if ! echo "$MARKDOWN" | grep -q "1 user prompts"; then
    echo "    ERROR: Should show '1 user prompts' (main session only)"
    fail "pr summary should show only main session prompts in header"
fi
echo "    - Header shows '1 user prompts' (main session only)"

# Step 10: Verify IsAgent field in session summary
echo "  Step 10: Verifying session IsAgent fields..."

# Check commits[0].sessions[*].is_agent
MAIN_SESSION_IS_AGENT=$(echo "$SUMMARY" | jq '.commits[0].sessions[] | select(.id == "fb813892-a738-4fc4-bcf8-b6f175a27a93") | .is_agent')
if [[ "$MAIN_SESSION_IS_AGENT" != "false" ]]; then
    echo "    ERROR: Main session should have is_agent=false, got $MAIN_SESSION_IS_AGENT"
    fail "Main session is_agent should be false"
fi
echo "    - Main session is_agent: false"

AGENT1_IS_AGENT=$(echo "$SUMMARY" | jq '.commits[0].sessions[] | select(.id == "agent-explore1") | .is_agent')
if [[ "$AGENT1_IS_AGENT" != "true" ]]; then
    echo "    ERROR: Agent session 1 should have is_agent=true, got $AGENT1_IS_AGENT"
    fail "Agent session is_agent should be true"
fi
echo "    - Agent session 1 is_agent: true"

AGENT2_IS_AGENT=$(echo "$SUMMARY" | jq '.commits[0].sessions[] | select(.id == "agent-explore2") | .is_agent')
if [[ "$AGENT2_IS_AGENT" != "true" ]]; then
    echo "    ERROR: Agent session 2 should have is_agent=true, got $AGENT2_IS_AGENT"
    fail "Agent session is_agent should be true"
fi
echo "    - Agent session 2 is_agent: true"

# Step 11: Cleanup
echo "  Step 11: Cleanup..."
cleanup_sessions
rm -rf /tmp/ci-output

echo ""
echo "  All agent session handling assertions passed!"
