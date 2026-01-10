# Git Prompt Story

**Automatically share LLM prompts in Pull Requests without changing your workflow.**

Git Prompt Story captures your AI sessions and links them to your commits. Your teammates get full context for code reviews, and you don't have to copy-paste a thing.

## Quick start

It requires git and Go (`brew install go`).

```bash
go install github.com/QuesmaOrg/git-prompt-story@latest
cd your_repository
git-prompt-story install-hooks --auto-push  # --global if for all
git-prompt-story generate-github-workflow
```

## Why

> "I've never felt this much behind as a programmer. The profession is being dramatically refactored."
> — [Andrej Karpathy](https://x.com/karpathy/status/2004607146781278521)

When LLM writes your code, the conversation is part of the story. See [Prompts are (not) the new source code](https://quesma.com/blog/prompts-source-code/) for a deeper dive.

**Review what matters.** Thousands of lines of generated code are hard to audit. The prompts that produced them? That's the real signal - the intent, constraints, and reasoning. seeing prompts in Pull Requests makes reviews faster and deeper.

**Learn from your teammates.** Prompts reveal problem-solving approaches, architectural decisions, and debugging strategies. Even Karpathy admits he could be "10X more powerful" by learning how others use AI tools.

**Intent verification.** Understand the "why" behind the code. Was this architectural choice deliberate, or just what the LLM spit out? The prompt reveals the difference.

**Require transparency.** Projects like [Ghostty now require AI disclosure](https://github.com/ghostty-org/ghostty/pull/8289) on PRs. We make it easy.

git-prompt-story captures LLM sessions in your git history - making prompts reviewable, searchable, and part of your project's permanent record.

## Principles

- **Open standard, not a product.** We're defining a convention for storing AI prompts in git, not building a walled garden. Fork it, extend it, build on it.
- **Vendor agnostic.** Claude Code today. Cursor, Codex, Gemini CLI tomorrow. One format for all tools.
- **Git-native.** No databases, no services, no accounts. Just git notes - portable, mergeable, already everywhere.
- **Privacy by default.** Notes stay local until you explicitly push. Review, redact, delete - you control what's shared.

## Features

- **Automatic capture** - Hooks detect active LLM sessions on commit
- **Privacy scrubbing** - Credentials, PII, and Read tool outputs redacted before storage ([configurable](docs/PII_SCRUBBING.md))
- **Review before push** - Curate or redact notes before sharing
- **Prompt tracking** - Commit messages show which tools and how many prompts were used

## Setup

### Individual Developer

Set up git-prompt-story on your machine to capture LLM sessions in all your repos.

```bash
# 1. Install
go install github.com/QuesmaOrg/git-prompt-story@latest

# 2. Install git hooks globally
# Add --auto-push to install pre-push hook that syncs notes automatically
# Without --auto-push, push notes manually with:
#   git push origin refs/notes/prompt-story +refs/notes/prompt-story-transcripts
git-prompt-story install-hooks --global --auto-push
```

That's it. Future commits will automatically capture active LLM sessions.

### Repository CI Integration

To add GitHub Action integration for your repository:

```bash
git-prompt-story generate-github-workflow
```

This interactively generates a workflow file that posts PR summaries and optionally deploys full transcripts to GitHub Pages.

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
│     ├── Save git note as blob {nid}                             │
│     ├── Generate summary (tools used)                           |
│     ├── Append to commit message:                               │
│     │   "Prompt-Story: Used Claude Code (N prompts)"            │
│     └── Save {nid} to .git/PENDING-PROMPT-STORY                 │
│                    │                                            │
│                    ▼                                            │
│  3. post-commit hook                                            │
│     ├── Read {nid} from .git/PENDING-PROMPT-STORY               │
│     ├── Attach note to HEAD via refs/notes/prompt-story         │
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

Git notes attached to commits in a dedicated namespace. Contains a lightweight JSON manifest:

```json
{
  "v": 1,
  "start_work": "2025-01-15T09:00:00Z",
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

### Local `git-prompt-story` (CLI) - Go

The capture tool runs locally via git hooks. It:

- Detects active LLM sessions
- Extracts conversation data
- Stores notes in `refs/notes/prompt-story`
- Adds summary to commit messages

Single binary, no runtime dependencies. Install once, works everywhere.

### GitHub Action

Two actions are available:

- **`prompt-story`** - Posts PR comment with markdown summary (default)
- **`prompt-story-with-pages`** - Posts PR comment + deploys full HTML transcripts to GitHub Pages

Both run on Pull Requests. Use `generate-github-workflow` to create the appropriate workflow.

## Viewer Integration

Notes are JSON - view them anywhere:

```bash
# Raw JSON
git notes --ref=prompt-story show HEAD

# Local pretty-print
git-prompt-story show HEAD

# Preview how it will look in GitHub Actions PR comment
git-prompt-story ci-preview              # Last commit
git-prompt-story ci-preview main..HEAD   # Current branch vs main (PR style)
```

## Privacy & Curation

Notes are local until you push them. Before sharing:

```bash
# Review what you're about to push
git-prompt-story review

# Interactive viewer - browse sessions and messages
git-prompt-story show HEAD
```

### Redacting Content

Use the interactive TUI or CLI flags to redact sensitive content:

```bash
# Interactive: press 'r' on a message to redact, 'D' to clear a session
git-prompt-story show HEAD

# CLI: Clear entire session (replaces content with empty file)
git-prompt-story show --clear-session "claude-code/session-id"

# CLI: Redact specific message by timestamp
git-prompt-story show --redact-message "claude-code/session-id@2025-01-15T10:00:00Z"
```

Redacted messages show `<REDACTED BY USER>` placeholder. Both git notes and local `~/.claude/projects/` files are updated.

**If notes were already pushed**, you'll need to force push:

```bash
git push -f origin refs/notes/prompt-story refs/notes/prompt-story-transcripts
```

Notes live in separate refs and can be explicitly pushed (unless you use `--auto-push`):

```bash
git push origin refs/notes/prompt-story +refs/notes/prompt-story-transcripts
```

## Roadmap

- [x] Claude Code support
- [x] Viewer (HTML export via `ci-html` and `ci-summary` commands)
- [x] GitHub Action for PR summaries and transcript pages
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
