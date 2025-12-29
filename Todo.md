# Implementation Plan: git-prompt-story Collection Part

## Status: COMPLETE

All phases implemented. Run `go install` to install, then `git-prompt-story install-hooks` in any repo.

## Project Structure

```
git-prompt-story/
├── go.mod                           # github.com/QuesmaOrg/git-prompt-story
├── main.go                          # Entry point
├── cmd/
│   ├── root.go                      # Root command
│   ├── install_hooks.go             # install-hooks command
│   ├── prepare_commit_msg.go        # Hook command
│   └── post_commit.go               # Hook command
├── internal/
│   ├── session/
│   │   ├── discovery.go             # Find Claude Code sessions
│   │   ├── parser.go                # Parse JSONL metadata
│   │   └── types.go                 # Session types
│   ├── git/
│   │   ├── notes.go                 # Git notes operations
│   │   ├── objects.go               # hash-object, mktree
│   │   ├── refs.go                  # update-ref
│   │   ├── repo.go                  # Repo detection
│   │   └── config.go                # Git config
│   ├── hooks/
│   │   ├── prepare_commit_msg.go    # Hook logic
│   │   ├── post_commit.go           # Hook logic
│   │   └── installer.go             # Hook installation
│   └── note/
│       ├── metadata.go              # PromptStoryNote JSON
│       └── transcript.go            # Transcript storage
```

## Implementation Phases

### Phase 1: Foundation
- [x] `go mod init github.com/QuesmaOrg/git-prompt-story`
- [x] Add cobra: `go get github.com/spf13/cobra`
- [x] Create `main.go`
- [x] Create `cmd/root.go`

### Phase 2: Session Discovery
- [x] `internal/session/types.go` - ClaudeSession, MessageEntry structs
- [x] `internal/session/discovery.go` - FindSessions(repoPath)
- [x] `internal/session/parser.go` - ParseSessionMetadata (extract timestamps)

### Phase 3: Git Operations
- [x] `internal/git/repo.go` - GetRepoRoot(), GetGitDir()
- [x] `internal/git/objects.go` - HashObject(), CreateTree()
- [x] `internal/git/refs.go` - UpdateRef(), GetRef()
- [x] `internal/git/notes.go` - AddNote()
- [x] `internal/git/config.go` - GetViewerURL()

### Phase 4: Note Storage
- [x] `internal/note/metadata.go` - PromptStoryNote struct
- [x] `internal/note/transcript.go` - StoreTranscript(), UpdateTranscriptTree()

### Phase 5: Hook Logic
- [x] `internal/hooks/prepare_commit_msg.go`
- [x] `internal/hooks/post_commit.go`
- [x] `internal/hooks/installer.go`

### Phase 6: CLI Commands
- [x] `cmd/install_hooks.go` - `install-hooks [--global]`
- [x] `cmd/prepare_commit_msg.go` - `prepare-commit-msg <msg-file>`
- [x] `cmd/post_commit.go` - `post-commit`

## Usage

```bash
# Install the CLI
go install github.com/QuesmaOrg/git-prompt-story@latest

# Install hooks in current repo
git-prompt-story install-hooks

# Or install globally
git-prompt-story install-hooks --global
```

## How It Works

1. On `git commit`, the `prepare-commit-msg` hook:
   - Finds Claude Code sessions in `~/.claude/projects/<encoded-path>/`
   - Stores session transcripts as git blobs
   - Creates a metadata note with session info
   - Appends "Prompt-Story: Used Claude Code | <url>" to commit message

2. After commit, the `post-commit` hook:
   - Attaches the metadata note to the new commit
   - Cleans up the pending file

## Storage

- **Metadata**: `refs/notes/prompt-story` (attached to commits)
- **Transcripts**: `refs/notes/prompt-story-transcripts/claude-code/<id>.jsonl`
- **Session source**: `~/.claude/projects/<encoded-path>/*.jsonl`
