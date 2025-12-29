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

## Quick Start

```bash
# Install (single binary, no dependencies)
go install github.com/QuesmaOrg/git-prompt-story@latest
# or: brew install git-prompt-story
# or: download from releases

cd your-repo
git-prompt-story init
```

That's it. Future commits will automatically capture active LLM sessions.

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
│     ├── Generate summary (tools used)                           │
│     ├── Append to commit message:                               │
│     │   "Prompt-Story: Used Claude Code | [View](<url>)"        │
│     └── Save session list to .git/GPS_PENDING                   │
│                    │                                            │
│                    ▼                                            │
│  3. post-commit hook                                            │
│     ├── Read session list from .git/GPS_PENDING                 │
│     ├── Store transcripts (if new) in transcript tree           │
│     ├── Create metadata JSON referencing transcripts            │
│     ├── Attach metadata as note to HEAD                         │
│     └── Clean up .git/GPS_PENDING                               │
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
  "sessions": [
    {
      "tool": "claude-code",
      "id": "113e0c55-64df-4b55-88f3-e06bcbc5b526",
      "path": "claude-code/113e0c55-64df-4b55-88f3-e06bcbc5b526.jsonl"
    },
    {
      "tool": "cursor",
      "id": "a1b2c3d4",
      "path": "cursor/a1b2c3d4.json"
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

MIT
