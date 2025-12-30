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
