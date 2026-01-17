#!/bin/bash
set -euo pipefail
source /e2e/lib/helpers.sh

echo "[10/10] CI Commands"

# Clean up any previous sessions
cleanup_sessions

# Step 1: Create test repo with multiple commits
echo "  Step 1: Creating test repo with commits..."
rm -rf /workspace/test-repo
mkdir -p /workspace/test-repo
cd /workspace/test-repo
git init

# Initial commit at 09:00 (no hooks yet)
echo "initial" > file.txt
git add file.txt
faketime '2025-01-15 09:00:00' git commit -m "Initial commit"
INITIAL_COMMIT=$(git rev-parse HEAD)

# Install hooks
git-prompt-story install-hooks

# Step 2: Create first feature commit with session
echo "  Step 2: Creating first feature commit..."
create_mock_session "session-1" "2025-01-15T09:15:00Z" "2025-01-15T09:45:00Z"
echo "feature 1" >> file.txt
git add file.txt
export GIT_AUTHOR_DATE="2025-01-15T10:00:00Z"
export GIT_COMMITTER_DATE="2025-01-15T10:00:00Z"
faketime '2025-01-15 10:00:00' git commit -m "Add feature 1"
unset GIT_AUTHOR_DATE GIT_COMMITTER_DATE
COMMIT1=$(git rev-parse HEAD)

# Step 3: Create second feature commit with session
echo "  Step 3: Creating second feature commit..."
create_mock_session "session-2" "2025-01-15T10:15:00Z" "2025-01-15T10:45:00Z"
echo "feature 2" >> file.txt
git add file.txt
export GIT_AUTHOR_DATE="2025-01-15T11:00:00Z"
export GIT_COMMITTER_DATE="2025-01-15T11:00:00Z"
faketime '2025-01-15 11:00:00' git commit -m "Add feature 2"
unset GIT_AUTHOR_DATE GIT_COMMITTER_DATE
COMMIT2=$(git rev-parse HEAD)

# Step 4: Test pr summary with JSON output
echo "  Step 4: Testing pr summary JSON output..."
OUTPUT=$(git-prompt-story pr summary "${INITIAL_COMMIT}..HEAD" --format=json)

# Verify JSON structure
echo "$OUTPUT" | jq -e '.commits_analyzed == 2' > /dev/null || fail "Expected 2 commits analyzed"
echo "    - commits_analyzed = 2"

echo "$OUTPUT" | jq -e '.commits_with_notes == 2' > /dev/null || fail "Expected 2 commits with notes"
echo "    - commits_with_notes = 2"

echo "$OUTPUT" | jq -e '.commits | length == 2' > /dev/null || fail "Expected 2 commits in output"
echo "    - commits array has 2 entries"

# Verify first commit has sessions
echo "$OUTPUT" | jq -e '.commits[0].sessions | length > 0' > /dev/null || fail "First commit should have sessions"
echo "    - First commit has sessions"

# Verify new count fields exist
echo "$OUTPUT" | jq -e '.total_user_prompts >= 0' > /dev/null || fail "Should have total_user_prompts field"
echo "    - Has total_user_prompts field"

echo "$OUTPUT" | jq -e '.total_steps >= 0' > /dev/null || fail "Should have total_steps field"
echo "    - Has total_steps field"

# Step 5: Test pr summary with Markdown output
echo "  Step 5: Testing pr summary Markdown output..."
MD_OUTPUT=$(git-prompt-story pr summary "${INITIAL_COMMIT}..HEAD" --format=markdown)

echo "$MD_OUTPUT" | grep -q "| Commit | Subject | Tool(s) | User Prompts | Steps |" || fail "Markdown should have new table header"
echo "    - Has new table header"

# User prompts section uses markdown header "# N user prompts"
echo "$MD_OUTPUT" | grep -q "# .* user prompts" || fail "Markdown should have user prompts section header"
echo "    - Has user prompts section"

echo "$MD_OUTPUT" | grep -q "# All .* steps" || fail "Markdown should have All steps section"
echo "    - Has All steps section"

echo "$MD_OUTPUT" | grep -q "Claude Code" || fail "Markdown should mention Claude Code"
echo "    - Mentions Claude Code"

# Step 5b: Test pr summary with long user prompt (should use <details> format)
echo "  Step 5b: Testing pr summary with long user prompt..."
cleanup_sessions
rm -rf /workspace/test-repo
mkdir -p /workspace/test-repo
cd /workspace/test-repo
git init

echo "initial" > file.txt
git add file.txt
faketime '2025-01-15 09:00:00' git commit -m "Initial commit"
LONG_PROMPT_INITIAL=$(git rev-parse HEAD)

git-prompt-story install-hooks

# Create session with a long prompt (>250 chars)
create_mock_session_with_long_prompt "long-session" "2025-01-15T09:15:00Z" "2025-01-15T09:45:00Z"
echo "feature" >> file.txt
git add file.txt
export GIT_AUTHOR_DATE="2025-01-15T10:00:00Z"
export GIT_COMMITTER_DATE="2025-01-15T10:00:00Z"
faketime '2025-01-15 10:00:00' git commit -m "Add feature with long prompt"
unset GIT_AUTHOR_DATE GIT_COMMITTER_DATE

LONG_MD=$(git-prompt-story pr summary "${LONG_PROMPT_INITIAL}..HEAD" --format=markdown)

# Section should use markdown header
echo "$LONG_MD" | grep -q "# 1 user prompts" || fail "Long prompts should have markdown header"
echo "    - Has markdown header for user prompts"

# Individual long prompts should have collapsible <details> entries
echo "$LONG_MD" | grep -q "<details><summary>" || fail "Long prompts should have collapsible entries"
echo "    - Has collapsible entries for long prompts"

# Step 6: Test pr summary with pages-url option
echo "  Step 6: Testing pr summary with pages-url..."
PAGES_MD=$(git-prompt-story pr summary "${LONG_PROMPT_INITIAL}..HEAD" --format=markdown --pages-url="https://example.github.io/repo/pr-42/")

echo "$PAGES_MD" | grep -q "https://example.github.io/repo/pr-42/" || fail "Markdown should contain pages URL"
echo "    - Contains pages URL"

# Step 7: Test pr html generation
echo "  Step 7: Testing pr html generation..."
rm -rf /tmp/html-test
LONG_PROMPT_COMMIT=$(git rev-parse HEAD)
git-prompt-story pr html "${LONG_PROMPT_INITIAL}..HEAD" --output-dir=/tmp/html-test --pr=42

test -f /tmp/html-test/index.html || fail "index.html should be created"
echo "    - index.html created"

# Check that commit page was created (short SHA format)
SHORT_SHA=$(echo "$LONG_PROMPT_COMMIT" | cut -c1-7)

test -f "/tmp/html-test/${SHORT_SHA}.html" || fail "${SHORT_SHA}.html should be created"
echo "    - ${SHORT_SHA}.html created"

# Verify HTML content
grep -q "PR #42" /tmp/html-test/index.html || fail "index.html should contain PR number"
echo "    - index.html contains PR #42"

grep -q "Prompt Story" /tmp/html-test/index.html || fail "index.html should contain title"
echo "    - index.html contains title"

# Step 8: Test pr summary output to file
echo "  Step 8: Testing pr summary file output..."
rm -f /tmp/summary.md
git-prompt-story pr summary "${LONG_PROMPT_INITIAL}..HEAD" --format=markdown --output=/tmp/summary.md

test -f /tmp/summary.md || fail "Output file should be created"
echo "    - Output file created"

grep -q "| Commit | Subject | Tool(s) | User Prompts | Steps |" /tmp/summary.md || fail "Output file should have content"
echo "    - Output file has content"

# Step 9: Test with no notes (should handle gracefully)
echo "  Step 9: Testing with commit range with no notes..."
# Create a commit without sessions
cleanup_sessions
echo "no session" >> file.txt
git add file.txt
faketime '2025-01-15 12:00:00' git commit -m "Commit without session"
NO_SESSION_COMMIT=$(git rev-parse HEAD)

# pr summary for just this commit should work but show 0 commits with notes
NO_NOTES_OUTPUT=$(git-prompt-story pr summary "${LONG_PROMPT_COMMIT}..HEAD" --format=json)
echo "$NO_NOTES_OUTPUT" | jq -e '.commits_with_notes == 0' > /dev/null || fail "Should show 0 commits with notes"
echo "    - Handles commits without notes gracefully"

echo ""
echo "  All assertions passed!"
