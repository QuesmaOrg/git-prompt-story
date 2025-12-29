# E2E Testing Strategy

## Goals

- Test the full hook flow (prepare-commit-msg → post-commit)
- Verify notes are attached correctly to commits
- Test `git-prompt-story show` displays session data
- Run in isolated Docker environment
- No real Claude Code needed - use mock session files

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     Docker Container                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  /workspace/                                                    │
│  └── test-repo/           <- Test git repository                │
│      ├── .git/                                                  │
│      │   └── hooks/       <- Installed hooks                    │
│      └── file.txt         <- Test file to commit                │
│                                                                 │
│  /root/.claude/                                                 │
│  └── projects/                                                  │
│      └── -workspace-test-repo/   <- Mock sessions dir           │
│          └── mock-session.jsonl  <- Mock Claude session         │
│                                                                 │
│  /usr/local/bin/                                                │
│  └── git-prompt-story     <- Built binary                       │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## Mock Session Format

Mock JSONL file simulating Claude Code session:

```jsonl
{"type":"user","sessionId":"test-session-1","timestamp":"2025-01-15T10:00:00Z","message":{"role":"user","content":"How do I implement feature X?"}}
{"type":"assistant","sessionId":"test-session-1","timestamp":"2025-01-15T10:01:00Z","message":{"role":"assistant","content":[{"type":"text","text":"Here's how to implement feature X..."}]}}
{"type":"user","sessionId":"test-session-1","timestamp":"2025-01-15T10:05:00Z","message":{"role":"user","content":"Can you add tests?"}}
{"type":"assistant","sessionId":"test-session-1","timestamp":"2025-01-15T10:06:00Z","message":{"role":"assistant","content":[{"type":"text","text":"Sure, here are the tests..."}]}}
```

## Test Cases

### 1. Basic Hook Flow
```bash
# Setup
create_mock_session "session-1" "2025-01-15T10:00:00Z" "2025-01-15T10:30:00Z"
echo "change" >> file.txt

# Act
git add file.txt
git commit -m "Test commit"

# Assert
git notes show HEAD | jq -e '.sessions | length > 0'
git log -1 --oneline | grep -q "Prompt-Story: Used Claude Code"
```

### 2. No Session = "Prompt-Story: none"
```bash
# Setup - no mock sessions

# Act
git commit --allow-empty -m "Empty commit"

# Assert
git log -1 --oneline | grep -q "Prompt-Story: none"
! git notes show HEAD 2>/dev/null  # No note attached
```

### 3. Show Command
```bash
# Setup
create_mock_session "session-1" "2025-01-15T10:00:00Z" "2025-01-15T10:30:00Z"
git commit --allow-empty -m "Test"

# Assert
git-prompt-story show HEAD | grep -q "USER: How do I implement"
git-prompt-story show --full HEAD | grep -q "Here's how to implement feature X"
```

### 4. Multiple Sessions
```bash
# Setup
create_mock_session "session-1" "2025-01-15T09:00:00Z" "2025-01-15T09:30:00Z"
create_mock_session "session-2" "2025-01-15T10:00:00Z" "2025-01-15T10:30:00Z"

# Act
git commit --allow-empty -m "Multi-session commit"

# Assert
git notes show HEAD | jq -e '.sessions | length == 2'
```

### 5. Transcript Storage
```bash
# Setup & commit with session

# Assert
git cat-file -p refs/notes/prompt-story-transcripts:claude-code/session-1.jsonl
```

## Directory Structure

```
e2e/
├── Dockerfile
├── run-tests.sh           # Entry point
├── lib/
│   └── helpers.sh         # create_mock_session, assert_*, etc.
├── fixtures/
│   └── mock-session.jsonl.template
└── tests/
    ├── 01-basic-flow.sh
    ├── 02-no-session.sh
    ├── 03-show-command.sh
    ├── 04-multiple-sessions.sh
    └── 05-transcript-storage.sh
```

## Dockerfile

```dockerfile
FROM golang:1.21-alpine

RUN apk add --no-cache git bash jq

WORKDIR /build
COPY . .
RUN go build -o /usr/local/bin/git-prompt-story .

WORKDIR /workspace
RUN git config --global user.email "test@test.com" && \
    git config --global user.name "Test User"

COPY e2e/ /e2e/
ENTRYPOINT ["/e2e/run-tests.sh"]
```

## Helper Functions

```bash
# lib/helpers.sh

create_mock_session() {
    local session_id="$1"
    local start_time="$2"
    local end_time="$3"

    local repo_path=$(pwd)
    local encoded_path=$(echo "$repo_path" | tr '/' '-')
    local session_dir="$HOME/.claude/projects/$encoded_path"

    mkdir -p "$session_dir"

    cat > "$session_dir/$session_id.jsonl" << EOF
{"type":"user","sessionId":"$session_id","timestamp":"$start_time","message":{"role":"user","content":"How do I implement feature X?"}}
{"type":"assistant","sessionId":"$session_id","timestamp":"$end_time","message":{"role":"assistant","content":[{"type":"text","text":"Here's how to implement feature X..."}]}}
EOF
}

init_test_repo() {
    rm -rf /workspace/test-repo
    mkdir -p /workspace/test-repo
    cd /workspace/test-repo
    git init
    git-prompt-story install-hooks
    echo "initial" > file.txt
    git add file.txt
    git commit -m "Initial commit"
}

cleanup_sessions() {
    rm -rf "$HOME/.claude/projects"
}
```

## Running Tests

```bash
# Build and run all tests
docker build -t git-prompt-story-test -f e2e/Dockerfile .
docker run --rm git-prompt-story-test

# Run specific test
docker run --rm git-prompt-story-test ./tests/01-basic-flow.sh

# Interactive debugging
docker run --rm -it git-prompt-story-test bash
```

## CI Integration

```yaml
# .github/workflows/e2e.yml
name: E2E Tests

on: [push, pull_request]

jobs:
  e2e:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Build test image
        run: docker build -t git-prompt-story-test -f e2e/Dockerfile .

      - name: Run E2E tests
        run: docker run --rm git-prompt-story-test
```

## Test Output

```
=== E2E Test Suite ===

[1/5] Basic Hook Flow
  ✓ Commit message contains Prompt-Story trailer
  ✓ Note attached to commit
  ✓ Note contains session data

[2/5] No Session
  ✓ Commit message contains "Prompt-Story: none"
  ✓ No note attached

[3/5] Show Command
  ✓ Shows user prompts
  ✓ --full shows complete content

[4/5] Multiple Sessions
  ✓ All sessions captured in note

[5/5] Transcript Storage
  ✓ Transcripts stored in refs/notes/prompt-story-transcripts

=== 5/5 tests passed ===
```
