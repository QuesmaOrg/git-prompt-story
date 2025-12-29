# Git Prompt Story

Store LLM prompts and sessions alongside your git commits using git notes.

## Why

> "I've never felt this much behind as a programmer. The profession is being dramatically refactored."
> — [Andrej Karpathy](https://x.com/karpathy/status/2004607146781278521)

When LLM writes your code, the conversation is part of the story.

**Learn from your teammates.** Prompts reveal problem-solving approaches, architectural decisions, and debugging strategies. Even Karpathy admits he could be "10X more powerful" by learning how others use AI tools.

**Review what matters.** Thousands of lines of generated code are hard to audit. The prompts that produced them? That's the real signal - the intent, constraints, and reasoning.

**Require transparency.** Projects like [Ghostty now require AI disclosure](https://github.com/ghostty-org/ghostty/pull/8289) on PRs. We make it easy.

git-prompt-story captures LLM sessions in your git history - making prompts reviewable, searchable, and part of your project's permanent record.

## Principles

- **Open standard, not a product.** We're defining a convention for storing AI prompts in git, not building a walled garden. Fork it, extend it, build on it.
- **Vendor agnostic.** Claude Code today. Cursor, Codex, Gemini CLI tomorrow. One format for all tools.
- **Git-native.** No databases, no services, no accounts. Just git notes - portable, mergeable, already everywhere.
- **Privacy by default.** Notes stay local until you explicitly push. Review, redact, delete - you control what's shared.

## Features

- **Automatic capture** - Hooks detect active LLM sessions on commit
- **Review before push** - Curate or redact notes before sharing
- **Viewer links** - Commit messages include links to rendered prompts

## Setup

### Individual Developer

Set up git-prompt-story on your machine to capture LLM sessions in all your repos.

```bash
# 1. Install
go install github.com/QuesmaOrg/git-prompt-story@latest

# 2. Install git hooks globally
git-prompt-story install-hooks --global

# 3. Enable automatic pushing of prompt notes
git config --global --add remote.origin.push 'refs/notes/prompt-story*:refs/notes/prompt-story*'

# 4. (Optional) Configure viewer URL - defaults to hosted service
git config --global prompt-story.viewer-url 'https://prompt-story.quesma.com/{owner}/{repo}/prompt/{note}'
```

That's it. Future commits will automatically capture active LLM sessions.

### Repository Owner

Set up git-prompt-story for your team by adding a setup script to your repository.

#### 1. Add setup script

Create `setup-prompt-story.sh` in your repo root:

```bash
#!/bin/bash
set -e

# Install git-prompt-story
go install github.com/QuesmaOrg/git-prompt-story@latest

# Install hooks for this repo
git-prompt-story install-hooks

# Enable automatic pushing of prompt notes
git config --add remote.origin.push 'refs/notes/prompt-story*:refs/notes/prompt-story*'

echo "git-prompt-story configured for this repository"
```

Contributors run `./setup-prompt-story.sh` after cloning.

#### 2. Install prompt server (optional for public repos, required for private)

For private repositories, host your own viewer:

```bash
docker run -d \
  -p 8080:8080 \
  -e GITHUB_TOKEN=your-token \
  ghcr.io/quesmaorg/git-prompt-story-server
```

Then configure the viewer URL in `setup-prompt-story.sh`:

```bash
git config prompt-story.viewer-url 'https://prompts.yourcompany.com/{owner}/{repo}/prompt/{note}'
```

Public repos can use the hosted service at `https://prompt-story.quesma.com`.

#### 3. Add CI check (optional)

Add a GitHub Action to verify prompt notes are attached to PRs:

```yaml
# .github/workflows/prompt-story.yml
name: Prompt Story Check

on:
  pull_request:
    types: [opened, synchronize]

jobs:
  check-prompts:
    runs-on: ubuntu-latest
    if: github.event.pull_request.merge_commit_sha == null  # Skip merge commits
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Fetch prompt notes
        run: git fetch origin 'refs/notes/prompt-story*:refs/notes/prompt-story*' || true

      - name: Check for prompt notes
        uses: quesmaorg/prompt-story-action@v1
        with:
          comment: true  # Post summary of LLM tools used
```

This action checks if commits have prompt notes attached and optionally posts a comment summarizing which LLM tools were used.

## How It Works

```
┌─────────────────────────────────────────────────────────────────┐
│                        COMMIT FLOW                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  1. You run: git commit                                         │
│                    │                                            │
│                    ▼                                            │
│  2. prepare-commit-msg hook                                     │
│     ├── Find active sessions for this repo                      │
│     ├── Store transcripts ref/notes/prompt-story-transcripts/*  │
│     ├── Save git note refs/notes/prompt-story/{nid}             │
│     ├── Generate summary (tools used)                           |
│     ├── Append to commit message:                               │
│     │   "Prompt-Story: Used Claude Code | <url>"                │
│     └── Save {nid} to .git/PENDING-PROMPT-STORY                 │
│                    │                                            │
│                    ▼                                            │
│  3. post-commit hook                                            │
│     ├── Read {nid} from .git/PENDING-PROMPT-STORY               │
│     ├── Attach refs/notes/prompt-story/{nid} as note to HEAD    │
│     └── Clean up .git/PENDING-PROMPT-STORY                      │
│                                                                 │
│  If no active sessions for this repo:                           │
│     └── Append: "Prompt-Story: none"                            │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Storage Format

Git Prompt Story uses two storage locations:

#### 1. Commit Metadata (`refs/notes/prompt-story`)

Standard git notes attached to commits. Contains a lightweight JSON manifest:

```json
{
  "v": 1,
  "start_work": "2025-01-15T09:00:00Z",
  "end_work": "2025-01-15T14:30:00Z",
  "sessions": [
    {
      "tool": "claude-code",
      "id": "113e0c55-64df-4b55-88f3-e06bcbc5b526",
      "path": "refs/notes/prompt-story-transcripts/claude-code/113e0c55-64df-4b55-88f3-e06bcbc5b526.jsonl",
      "created": "2025-01-15T09:15:00Z",
      "modified": "2025-01-15T14:22:00Z"
    },
    {
      "tool": "cursor",
      "id": "a1b2c3d4",
      "path": "refs/notes/prompt-story-transcripts/cursor/a1b2c3d4.json",
      "created": "2025-01-15T10:00:00Z",
      "modified": "2025-01-15T12:45:00Z"
    }
  ]
}
```

#### 2. Transcripts (`refs/notes/prompt-story-transcripts`)

A tree ref containing raw session files, organized by tool:

```
refs/notes/prompt-story-transcripts/
├── claude-code/
│   ├── 113e0c55-64df-4b55-88f3-e06bcbc5b526.jsonl
│   └── 7f8a9b0c-1d2e-3f4a-5b6c-7d8e9f0a1b2c.jsonl
├── cursor/
│   └── a1b2c3d4.json
└── codex/
    └── session-2025-01-15.jsonl
```

**Key design choices:**

- **Many-to-many**: One session can be referenced by many commits. One commit can reference many sessions.
- **Whole sessions**: Transcripts stored as-is, no slicing. Parsing happens in viewer.
- **Deduplication**: Same session blob is referenced, not copied.
- **Lightweight capture**: Minimal processing at commit time.

### Auto-Detection

git-prompt-story finds active sessions by checking:

| Tool        | Location                                    | Status  |
| ----------- | ------------------------------------------- | ------- |
| Claude Code | `~/.claude/projects/<encoded-path>/*.jsonl` | Done    |
| Cursor      | TBD                                         | Planned |
| Codex       | TBD                                         | Planned |
| Gemini CLI  | TBD                                         | Planned |

## Architecture

Git Prompt Story has two components:

### `git-prompt-story` (CLI) - Go

The capture tool runs locally via git hooks. It:

- Detects active LLM sessions
- Extracts conversation data
- Stores notes in `refs/notes/llm-prompts`
- Adds summary to commit messages

Single binary, no runtime dependencies. Install once, works everywhere.

### `git-prompt-story-server` (Viewer) - separate

Renders stored notes for the web. Options:

- **Self-hosted**: Run your own instance
- **Hosted service**: Use our hosted viewer (planned)
- **GitHub Action**: Auto-publish notes as PR comments (planned)

For local viewing without the server:

- **Raw**: `git notes --ref=llm-prompts show HEAD`
- **CLI**: `git-prompt-story show` (built into capture tool)

The CLI is essential. The server is optional.

## Commands (DRAFT)

### Capture

```bash
git-prompt-story init          # Install hooks in current repo
git-prompt-story init --global # Install hooks globally
```

### View

```bash
git-prompt-story show          # Show note for HEAD
git-prompt-story show abc123   # Show note for specific commit
git-prompt-story log           # List commits with notes
```

### Curate

```bash
git-prompt-story review        # Interactive review of unpushed notes
git-prompt-story edit HEAD     # Edit note content
git-prompt-story remove HEAD   # Delete note from commit
```

### Sync

```bash
git-prompt-story push          # Push notes to origin
git-prompt-story pull          # Pull notes from origin
```

## Configuration

Config lives in `.git-prompt-story.json` or `~/.config/git-prompt-story.json`:

```json
{
  "viewer_url": "https://prompt-story.quesma.com/{owner}/{repo}/prompt/{note}",
  "auto_push_notes": false
}
```

### Viewer URL Templates

Variables available:

- `{owner}` - Repository owner (e.g., QuesmaOrg)
- `{repo}` - Repository name (e.g., git-prompt-story)
- `{note}` - Note blob SHA (short)

## Viewer Integration

Notes are JSON - view them anywhere:

```bash
# Raw JSON
git notes --ref=prompt-story show HEAD

# Local pretty-print
git-prompt-story show HEAD

# Or use the hosted viewer
# https://prompt-story.quesma.com/{owner}/{repo}/prompt/{note-sha}
```

## Privacy & Curation

Notes are local until you push them. Before sharing:

```bash
# Review what you're about to push
git-prompt-story review

# Remove sensitive notes
git-prompt-story remove <commit>

# Edit to redact content
git-prompt-story edit <commit>
```

Notes live in separate refs and must be explicitly pushed:

```bash
git push origin refs/notes/prompt-story
git push origin refs/notes/prompt-story-transcripts
```

## Roadmap

- [ ] Claude Code support
- [ ] Local viewer (HTML export)
- [ ] Hosted viewer service
- [ ] Cursor integration
- [ ] Codex integration
- [ ] Gemini CLI integration
- [ ] VS Code extension (show prompts inline)

## How Claude Code Stores Sessions

Claude Code saves conversations as JSONL files in:

```
~/.claude/projects/<encoded-path>/<session-id>.jsonl
```

Where `<encoded-path>` is your project path with `/` replaced by `-`.
Example: `/home/user/myapp` → `-home-user-myapp`

Each line is a JSON event with timestamps, making delta computation straightforward.

## License

[Apache License 2.0](https://www.apache.org/licenses/LICENSE-2.0)
