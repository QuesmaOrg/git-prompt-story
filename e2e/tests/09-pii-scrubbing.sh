#!/bin/bash
set -euo pipefail
source /e2e/lib/helpers.sh

echo "[9/9] PII Scrubbing Test"

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

# Step 3: Install hooks
echo "  Step 3: Installing hooks..."
git-prompt-story install-hooks

# Step 4: Create mock session with various PII types
echo "  Step 4: Creating mock session with PII data..."
create_mock_session_with_pii "pii-session" "2025-01-15T09:15:00Z" "2025-01-15T09:25:00Z"

# Step 5: Commit to trigger scrubbing
echo "  Step 5: Creating commit (triggers PII scrubbing)..."
echo "feature" >> file.txt
git add file.txt
export GIT_AUTHOR_DATE="2025-01-15T10:30:00Z"
export GIT_COMMITTER_DATE="2025-01-15T10:30:00Z"
faketime '2025-01-15 10:30:00' git commit -m "Add feature with PII in session"
unset GIT_AUTHOR_DATE GIT_COMMITTER_DATE

# Step 6: Retrieve stored transcript
echo "  Step 6: Retrieving stored transcript..."
TRANSCRIPT=$(git cat-file -p "refs/notes/prompt-story-transcripts:claude-code/pii-session.jsonl")

echo "  Step 7: Verifying PII was scrubbed..."

# Check that original PII is NOT present
echo "    Checking original PII is removed..."

if echo "$TRANSCRIPT" | grep -q "john.doe@example.com"; then
    fail "Email was not scrubbed"
fi
echo "    - Email scrubbed"

if echo "$TRANSCRIPT" | grep -q "/Users/jacek/"; then
    fail "User path was not scrubbed"
fi
echo "    - User path scrubbed"

if echo "$TRANSCRIPT" | grep -q "AKIAIOSFODNN7EXAMPLE"; then
    fail "AWS key was not scrubbed"
fi
echo "    - AWS key scrubbed"

if echo "$TRANSCRIPT" | grep -q "sk-or-v1-abcdefghijklmnopqrstuvwxyz1234567890abcd"; then
    fail "OpenRouter key was not scrubbed"
fi
echo "    - OpenRouter key scrubbed"

if echo "$TRANSCRIPT" | grep -q "sk-ant-api03-abcdefghijklmnopqrstuvwxyz1234567890abcdefgh"; then
    fail "Anthropic key was not scrubbed"
fi
echo "    - Anthropic key scrubbed"

if echo "$TRANSCRIPT" | grep -q "ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij"; then
    fail "GitHub token was not scrubbed"
fi
echo "    - GitHub token scrubbed"

if echo "$TRANSCRIPT" | grep -q "4111111111111111"; then
    fail "Credit card was not scrubbed"
fi
echo "    - Credit card scrubbed"

if echo "$TRANSCRIPT" | grep -q "supersecret123"; then
    fail "Password was not scrubbed"
fi
echo "    - Password scrubbed"

if echo "$TRANSCRIPT" | grep -q "AIzaSyA1234567890abcdefghijklmnopqrstuv"; then
    fail "Google API key was not scrubbed"
fi
echo "    - Google API key scrubbed"

# Check that replacements ARE present (using unicode-escaped versions since JSON encodes < and >)
echo "    Checking replacements are present..."

# Note: JSON marshaling escapes <> to \u003c and \u003e
if ! echo "$TRANSCRIPT" | grep -qE '(<EMAIL>|\\u003cEMAIL\\u003e)'; then
    fail "EMAIL replacement not found"
fi
echo "    - <EMAIL> replacement present"

if ! echo "$TRANSCRIPT" | grep -qE '(<REDACTED>|\\u003cREDACTED\\u003e)'; then
    fail "REDACTED replacement not found"
fi
echo "    - <REDACTED> replacement present"

if ! echo "$TRANSCRIPT" | grep -qE '(<AWS_ACCESS_KEY>|\\u003cAWS_ACCESS_KEY\\u003e)'; then
    fail "AWS_ACCESS_KEY replacement not found"
fi
echo "    - <AWS_ACCESS_KEY> replacement present"

if ! echo "$TRANSCRIPT" | grep -qE '(<OPENROUTER_API_KEY>|\\u003cOPENROUTER_API_KEY\\u003e)'; then
    fail "OPENROUTER_API_KEY replacement not found"
fi
echo "    - <OPENROUTER_API_KEY> replacement present"

if ! echo "$TRANSCRIPT" | grep -qE '(<ANTHROPIC_API_KEY>|\\u003cANTHROPIC_API_KEY\\u003e)'; then
    fail "ANTHROPIC_API_KEY replacement not found"
fi
echo "    - <ANTHROPIC_API_KEY> replacement present"

if ! echo "$TRANSCRIPT" | grep -qE '(<GITHUB_TOKEN>|\\u003cGITHUB_TOKEN\\u003e)'; then
    fail "GITHUB_TOKEN replacement not found"
fi
echo "    - <GITHUB_TOKEN> replacement present"

if ! echo "$TRANSCRIPT" | grep -qE '(<CREDIT_CARD>|\\u003cCREDIT_CARD\\u003e)'; then
    fail "CREDIT_CARD replacement not found"
fi
echo "    - <CREDIT_CARD> replacement present"

if ! echo "$TRANSCRIPT" | grep -qE '(<PASSWORD>|\\u003cPASSWORD\\u003e)'; then
    fail "PASSWORD replacement not found"
fi
echo "    - <PASSWORD> replacement present"

if ! echo "$TRANSCRIPT" | grep -qE '(<GOOGLE_API_KEY>|\\u003cGOOGLE_API_KEY\\u003e)'; then
    fail "GOOGLE_API_KEY replacement not found"
fi
echo "    - <GOOGLE_API_KEY> replacement present"

# Verify transcript is still valid JSONL
echo "    Verifying transcript is valid JSONL..."
while IFS= read -r line; do
    echo "$line" | jq -e . > /dev/null 2>&1 || fail "Invalid JSON line: $line"
done <<< "$TRANSCRIPT"
echo "    - All lines are valid JSON"

echo ""
echo "  All PII scrubbing assertions passed!"
