#!/bin/bash
# E2E Test Helper Functions

# Fail with error message and exit
fail() {
    echo "FAIL: $1" >&2
    exit 1
}

# Create a mock Claude Code session with realistic JSONL format
# Usage: create_mock_session <session_id> <start_time> <end_time>
# Example: create_mock_session "test-session-1" "2025-01-15T09:15:00Z" "2025-01-15T10:25:00Z"
create_mock_session() {
    local session_id="$1"
    local start_time="$2"
    local end_time="$3"

    local repo_path=$(pwd)
    local encoded_path=$(echo "$repo_path" | tr '/' '-')
    local session_dir="$HOME/.claude/projects/$encoded_path"

    mkdir -p "$session_dir"

    cat > "$session_dir/$session_id.jsonl" << EOF
{"type":"user","sessionId":"$session_id","timestamp":"$start_time","cwd":"$repo_path","gitBranch":"main","message":{"role":"user","content":"How do I implement feature X?"}}
{"type":"assistant","sessionId":"$session_id","timestamp":"$end_time","message":{"role":"assistant","content":[{"type":"text","text":"Here's how to implement feature X..."}]}}
EOF

    echo "  Created mock session: $session_dir/$session_id.jsonl"
}

# Create a mock Claude Code session with a long user prompt (>250 chars)
# Usage: create_mock_session_with_long_prompt <session_id> <start_time> <end_time>
create_mock_session_with_long_prompt() {
    local session_id="$1"
    local start_time="$2"
    local end_time="$3"

    local repo_path=$(pwd)
    local encoded_path=$(echo "$repo_path" | tr '/' '-')
    local session_dir="$HOME/.claude/projects/$encoded_path"

    mkdir -p "$session_dir"

    # Create a prompt that is definitely longer than 250 characters
    local long_prompt="I need help implementing a comprehensive authentication system for my web application. The system should support multiple authentication methods including username/password, OAuth2 with Google and GitHub providers, and magic link email authentication. Additionally, I want to implement role-based access control with at least three roles: admin, editor, and viewer."

    cat > "$session_dir/$session_id.jsonl" << EOF
{"type":"user","sessionId":"$session_id","timestamp":"$start_time","cwd":"$repo_path","gitBranch":"main","message":{"role":"user","content":"$long_prompt"}}
{"type":"assistant","sessionId":"$session_id","timestamp":"$end_time","message":{"role":"assistant","content":[{"type":"text","text":"I'll help you implement a comprehensive authentication system..."}]}}
EOF

    echo "  Created mock session with long prompt: $session_dir/$session_id.jsonl"
}

# Create a mock Claude Code session with tool uses for testing PR comment format
# Usage: create_mock_session_with_tools <session_id> <start_time> <end_time>
create_mock_session_with_tools() {
    local session_id="$1"
    local start_time="$2"
    local end_time="$3"

    local repo_path=$(pwd)
    local encoded_path=$(echo "$repo_path" | tr '/' '-')
    local session_dir="$HOME/.claude/projects/$encoded_path"

    mkdir -p "$session_dir"

    cat > "$session_dir/$session_id.jsonl" << EOF
{"type":"user","sessionId":"$session_id","timestamp":"$start_time","cwd":"$repo_path","gitBranch":"main","message":{"role":"user","content":"Add a new function to utils.go"}}
{"type":"assistant","sessionId":"$session_id","timestamp":"$start_time","message":{"role":"assistant","content":[{"type":"text","text":"I'll help you add a new function."},{"type":"tool_use","id":"tool1","name":"Read","input":{"file_path":"/utils.go"}}]}}
{"type":"user","sessionId":"$session_id","timestamp":"$start_time","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"tool1","content":"package utils\n\nfunc Existing() {}"}]}}
{"type":"assistant","sessionId":"$session_id","timestamp":"$start_time","message":{"role":"assistant","content":[{"type":"tool_use","id":"tool2","name":"Edit","input":{"file_path":"/utils.go"}}]}}
{"type":"user","sessionId":"$session_id","timestamp":"$end_time","message":{"role":"user","content":"<command-name>/commit</command-name>"}}
{"type":"assistant","sessionId":"$session_id","timestamp":"$end_time","message":{"role":"assistant","content":[{"type":"text","text":"Changes committed."}]}}
EOF

    echo "  Created mock session with tools: $session_dir/$session_id.jsonl"
}

# Create a mock Claude Code session for a specific project path
# Usage: create_mock_session_for_path <project_path> <session_id> <start_time> <end_time>
# Example: create_mock_session_for_path "/workspace/other-repo" "session-2" "2025-01-15T09:15:00Z" "2025-01-15T10:25:00Z"
create_mock_session_for_path() {
    local project_path="$1"
    local session_id="$2"
    local start_time="$3"
    local end_time="$4"

    local encoded_path=$(echo "$project_path" | tr '/' '-')
    local session_dir="$HOME/.claude/projects/$encoded_path"

    mkdir -p "$session_dir"

    cat > "$session_dir/$session_id.jsonl" << EOF
{"type":"user","sessionId":"$session_id","timestamp":"$start_time","cwd":"$project_path","gitBranch":"main","message":{"role":"user","content":"How do I implement feature X?"}}
{"type":"assistant","sessionId":"$session_id","timestamp":"$end_time","message":{"role":"assistant","content":[{"type":"text","text":"Here's how to implement feature X..."}]}}
EOF

    echo "  Created mock session for $project_path: $session_dir/$session_id.jsonl"
}

# Create mock session with user messages before work period, but file modified within
# This simulates a session where Claude responded (updating modified time) but user sent no new prompts
# Usage: create_stale_session <session_id> <user_msg_time> <assistant_time>
create_stale_session() {
    local session_id="$1"
    local user_msg_time="$2"      # When user messages occurred (before work period)
    local assistant_time="$3"      # When assistant responded (within work period)

    local repo_path=$(pwd)
    local encoded_path=$(echo "$repo_path" | tr '/' '-')
    local session_dir="$HOME/.claude/projects/$encoded_path"

    mkdir -p "$session_dir"

    cat > "$session_dir/$session_id.jsonl" << EOF
{"type":"user","sessionId":"$session_id","timestamp":"$user_msg_time","cwd":"$repo_path","gitBranch":"main","message":{"role":"user","content":"How do I implement feature X?"}}
{"type":"assistant","sessionId":"$session_id","timestamp":"$assistant_time","message":{"role":"assistant","content":[{"type":"text","text":"Here's how to implement feature X..."}]}}
EOF

    echo "  Created stale session: $session_dir/$session_id.jsonl"
    echo "    User message at: $user_msg_time (before work period)"
    echo "    Assistant response at: $assistant_time (within work period)"
}

# Clean up all mock sessions
cleanup_sessions() {
    rm -rf "$HOME/.claude/projects"
}

# Initialize a fresh test repository
# Usage: init_test_repo [path]
init_test_repo() {
    local repo_path="${1:-/workspace/test-repo}"

    rm -rf "$repo_path"
    mkdir -p "$repo_path"
    cd "$repo_path"
    git init

    echo "  Created test repo at: $repo_path"
}

# =============================================================================
# Cloud API Mocking Helpers
# =============================================================================

# Setup fake security command and claude.json for cloud API tests
setup_mock_credentials() {
    # Create fake security command that returns mock credentials
    cat > /usr/local/bin/security << 'SECEOF'
#!/bin/bash
if [[ "$*" == *"Claude Code-credentials"* && "$*" == *"-w"* ]]; then
    echo '{"claudeAiOauth":{"accessToken":"test-token-abc123"}}'
fi
SECEOF
    chmod +x /usr/local/bin/security

    # Create fake ~/.claude.json
    cat > ~/.claude.json << 'EOF'
{"oauthAccount":{"organizationUuid":"test-org-uuid-123"}}
EOF

    echo "  Mock credentials configured"
}

# Create a mock cloud session with API responses
# Usage: create_mock_cloud_session <session_id> <title> <branch> [created] [updated]
create_mock_cloud_session() {
    local session_id="$1"
    local title="$2"
    local branch="$3"
    local created="${4:-2025-01-15T09:00:00Z}"
    local updated="${5:-2025-01-15T10:00:00Z}"

    mkdir -p /tmp/mock-api

    # Session metadata response
    cat > "/tmp/mock-api/session_${session_id}.json" << EOF
{
  "id": "${session_id}",
  "title": "${title}",
  "created_at": "${created}",
  "updated_at": "${updated}",
  "session_status": "idle",
  "type": "internal_session",
  "session_context": {
    "model": "claude-opus-4",
    "outcomes": [{
      "type": "git_repository",
      "git_info": {"type": "github", "branches": ["${branch}"]}
    }]
  }
}
EOF

    # Events response
    cat > "/tmp/mock-api/session_${session_id}_events.json" << EOF
{
  "data": [
    {"type": "user", "uuid": "u1", "session_id": "${session_id}", "message": {"role": "user", "content": "Test prompt from mock"}},
    {"type": "assistant", "uuid": "a1", "session_id": "${session_id}", "message": {"role": "assistant", "content": [{"type": "text", "text": "Test response from mock"}]}}
  ],
  "has_more": false,
  "first_id": "u1",
  "last_id": "a1"
}
EOF

    # Sessions list response (for list-cloud)
    cat > "/tmp/mock-api/sessions_list.json" << EOF
{
  "data": [{
    "id": "${session_id}",
    "title": "${title}",
    "created_at": "${created}",
    "updated_at": "${updated}",
    "session_status": "idle",
    "type": "internal_session",
    "session_context": {
      "model": "claude-opus-4",
      "outcomes": [{
        "type": "git_repository",
        "git_info": {"type": "github", "branches": ["${branch}"]}
      }]
    }
  }],
  "has_more": false,
  "first_id": "${session_id}",
  "last_id": "${session_id}"
}
EOF

    echo "  Created mock cloud session: ${session_id}"
}

# Start a mock API server that routes requests to mock files
# Usage: start_mock_api [port]
start_mock_api() {
    local port="${1:-9999}"

    # Create a handler script that processes one HTTP request via stdin/stdout
    cat > /tmp/mock-api/handler.sh << 'HANDLEREOF'
#!/bin/bash
MOCK_DIR="/tmp/mock-api"

# Read the request line
read -r request_line

# Extract path from request
path=$(echo "$request_line" | awk '{print $2}')

# Log for debugging
echo "$(date): Request for $path" >> /tmp/mock-api/requests.log

# Skip remaining headers (read until empty line)
while IFS= read -r header; do
    # Check for end of headers (empty line or just CR)
    header=$(echo "$header" | tr -d '\r')
    [[ -z "$header" ]] && break
done

# Route request to appropriate response file
response_file=""
if [[ "$path" == "/v1/sessions/"*"/events"* ]]; then
    session_id=$(echo "$path" | sed -n 's|/v1/sessions/\([^/]*\)/events.*|\1|p')
    response_file="$MOCK_DIR/session_${session_id}_events.json"
elif [[ "$path" == "/v1/sessions?"* ]]; then
    response_file="$MOCK_DIR/sessions_list.json"
elif [[ "$path" == "/v1/sessions/"* ]]; then
    session_id=$(echo "$path" | sed -n 's|/v1/sessions/\([^?]*\).*|\1|p')
    response_file="$MOCK_DIR/session_${session_id}.json"
fi

# Send HTTP response
if [[ -f "$response_file" ]]; then
    content=$(cat "$response_file")
    content_length=${#content}
    printf "HTTP/1.1 200 OK\r\n"
    printf "Content-Type: application/json\r\n"
    printf "Connection: close\r\n"
    printf "Content-Length: %d\r\n" "$content_length"
    printf "\r\n"
    printf "%s" "$content"
else
    echo "$(date): 404 for $path (tried $response_file)" >> /tmp/mock-api/requests.log
    printf "HTTP/1.1 404 Not Found\r\n"
    printf "Connection: close\r\n"
    printf "Content-Length: 0\r\n"
    printf "\r\n"
fi
HANDLEREOF
    chmod +x /tmp/mock-api/handler.sh

    # Use socat to run the handler for each connection
    # socat forks for each connection and runs the handler script
    socat TCP-LISTEN:$port,reuseaddr,fork EXEC:/tmp/mock-api/handler.sh &
    echo $! > /tmp/mock-api.pid

    export CLAUDE_API_URL="http://localhost:$port"
    sleep 0.3
    echo "  Mock API server started on port $port (PID: $(cat /tmp/mock-api.pid))"
}

# Stop the mock API server
stop_mock_api() {
    if [[ -f /tmp/mock-api.pid ]]; then
        # Kill the socat server and all its children
        local pid=$(cat /tmp/mock-api.pid)
        pkill -P $pid 2>/dev/null || true
        kill $pid 2>/dev/null || true
        rm -f /tmp/mock-api.pid
    fi
    # Kill any stray socat processes
    pkill -f "socat TCP-LISTEN:9999" 2>/dev/null || true
    echo "  Mock API server stopped"
}

# Clean up mock cloud resources
cleanup_mock_cloud() {
    stop_mock_api
    rm -rf /tmp/mock-api
    rm -f ~/.claude.json
    echo "  Mock cloud resources cleaned up"
}

# =============================================================================
# PII Scrubbing Test Helpers
# =============================================================================

# Create mock session with various PII data for scrubbing tests
# Usage: create_mock_session_with_pii <session_id> <start_time> <end_time>
create_mock_session_with_pii() {
    local session_id="$1"
    local start_time="$2"
    local end_time="$3"

    local repo_path=$(pwd)
    local encoded_path=$(echo "$repo_path" | tr '/' '-')
    local session_dir="$HOME/.claude/projects/$encoded_path"

    mkdir -p "$session_dir"

    cat > "$session_dir/$session_id.jsonl" << 'EOF'
{"type":"user","sessionId":"pii-test","timestamp":"2025-01-15T09:15:00Z","cwd":"/workspace/test-repo","gitBranch":"main","message":{"role":"user","content":"My email is john.doe@example.com"}}
{"type":"user","sessionId":"pii-test","timestamp":"2025-01-15T09:16:00Z","cwd":"/workspace/test-repo","gitBranch":"main","message":{"role":"user","content":"Working in /Users/jacek/projects/secret-app/"}}
{"type":"user","sessionId":"pii-test","timestamp":"2025-01-15T09:17:00Z","cwd":"/workspace/test-repo","gitBranch":"main","message":{"role":"user","content":"AWS key is AKIAIOSFODNN7EXAMPLE"}}
{"type":"user","sessionId":"pii-test","timestamp":"2025-01-15T09:18:00Z","cwd":"/workspace/test-repo","gitBranch":"main","message":{"role":"user","content":"OpenRouter key: sk-or-v1-abcdefghijklmnopqrstuvwxyz1234567890abcd"}}
{"type":"user","sessionId":"pii-test","timestamp":"2025-01-15T09:19:00Z","cwd":"/workspace/test-repo","gitBranch":"main","message":{"role":"user","content":"Anthropic key: sk-ant-api03-abcdefghijklmnopqrstuvwxyz1234567890abcdefgh"}}
{"type":"user","sessionId":"pii-test","timestamp":"2025-01-15T09:20:00Z","cwd":"/workspace/test-repo","gitBranch":"main","message":{"role":"user","content":"GitHub token: ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij"}}
{"type":"user","sessionId":"pii-test","timestamp":"2025-01-15T09:21:00Z","cwd":"/workspace/test-repo","gitBranch":"main","message":{"role":"user","content":"Credit card: 4111111111111111"}}
{"type":"user","sessionId":"pii-test","timestamp":"2025-01-15T09:22:00Z","cwd":"/workspace/test-repo","gitBranch":"main","message":{"role":"user","content":"password=supersecret123"}}
{"type":"user","sessionId":"pii-test","timestamp":"2025-01-15T09:23:00Z","cwd":"/workspace/test-repo","gitBranch":"main","message":{"role":"user","content":"Google key: AIzaSyA1234567890abcdefghijklmnopqrstuv"}}
{"type":"assistant","sessionId":"pii-test","timestamp":"2025-01-15T09:25:00Z","message":{"role":"assistant","content":[{"type":"text","text":"I'll help with that sensitive data"}]}}
EOF

    # Update timestamps in the file
    sed -i "s/2025-01-15T09:15:00Z/$start_time/g" "$session_dir/$session_id.jsonl"
    sed -i "s/2025-01-15T09:25:00Z/$end_time/g" "$session_dir/$session_id.jsonl"

    echo "  Created mock session with PII: $session_dir/$session_id.jsonl"
}

# Create a mock agent session (with "agent-" prefix in session ID)
# Usage: create_mock_agent_session <agent_suffix> <start_time> <end_time>
# Example: create_mock_agent_session "explore1" "2025-01-15T09:15:00Z" "2025-01-15T09:20:00Z"
# Results in session ID: agent-explore1
create_mock_agent_session() {
    local agent_suffix="$1"
    local start_time="$2"
    local end_time="$3"
    local session_id="agent-${agent_suffix}"

    local repo_path=$(pwd)
    local encoded_path=$(echo "$repo_path" | tr '/' '-')
    local session_dir="$HOME/.claude/projects/$encoded_path"

    mkdir -p "$session_dir"

    cat > "$session_dir/$session_id.jsonl" << EOF
{"type":"user","sessionId":"$session_id","timestamp":"$start_time","cwd":"$repo_path","gitBranch":"main","message":{"role":"user","content":"[Agent task] Explore the codebase for pattern X"}}
{"type":"assistant","sessionId":"$session_id","timestamp":"$end_time","message":{"role":"assistant","content":[{"type":"text","text":"Found pattern X in the following files..."}]}}
EOF

    echo "  Created mock agent session: $session_dir/$session_id.jsonl"
}
