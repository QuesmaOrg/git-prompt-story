# Git Prompt Story

**Automatically share LLM prompts in Pull Requests without changing your workflow.**

Git Prompt Story captures your AI sessions and links them to your commits. Your teammates get full context for code reviews, and you don't have to copy-paste a thing.

## Quick start

```bash
brew install QuesmaOrg/tap/git-prompt-story
cd your_repository
git-prompt-story install-hooks --auto-push  # --global if for all
git-prompt-story install-github-workflow
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

### 1. Install

**Homebrew (macOS):**
```bash
brew install QuesmaOrg/tap/git-prompt-story
```

**Go:**
```bash
go install github.com/QuesmaOrg/git-prompt-story@latest
```

### 2. Configure Repository

Navigate to your repository and install the hooks:

```bash
cd your_repository
git-prompt-story install-hooks --auto-push  # Add --global to install for all repos
```

The `--auto-push` flag installs a `pre-push` hook that automatically syncs your notes. If you omit it, you must push notes manually:

```bash
git push origin refs/notes/prompt-story +refs/notes/prompt-story-transcripts
```

### 3. GitHub Actions

Generate a workflow to post summaries on Pull Requests:

```bash
git-prompt-story install-github-workflow
```

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

Both run on Pull Requests. Use `install-github-workflow` to create the appropriate workflow.

## Storage Format

Git Prompt Story uses two storage locations to keep your main branch clean:

### 1. Commit Metadata (`refs/notes/prompt-story`)

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
    }
  ]
}
```

### 2. Transcripts (`refs/notes/prompt-story-transcripts`)

A tree ref containing raw session files, organized by tool:

```
refs/notes/prompt-story-transcripts/
├── claude-code/
│   ├── 113e0c55-64df-4b55-88f3-e06bcbc5b526.jsonl
│   └── 7f8a9b0c-1d2e-3f4a-5b6c-7d8e9f0a1b2c.jsonl
└── cursor/ (planned)
```

**Key design choices:**

- **Many-to-many**: One session can be referenced by many commits. One commit can reference many sessions.
- **Whole sessions**: Transcripts stored as-is, no slicing. Parsing happens in viewer.
- **Deduplication**: Same session blob is referenced, not copied.
- **Lightweight capture**: Minimal processing at commit time.

## Auto-Detection

git-prompt-story finds active sessions by checking:

| Tool        | Location                                    | Status  |
| ----------- | ------------------------------------------- | ------- |
| Claude Code | `~/.claude/projects/<encoded-path>/*.jsonl` | Done    |
| Cursor      | TBD                                         | Planned |
| Codex       | TBD                                         | Planned |
| Gemini CLI  | TBD                                         | Planned |

## View Notes

```bash
# Pretty-print notes for the last commit
git-prompt-story show HEAD

# Preview PR comment style
# You can compare any two commits, branches, or ranges
git-prompt-story pr preview main..HEAD
```

## Privacy

Notes are local until pushed.

```bash
# Interactive viewer: browse sessions, press 'r' to redact messages
git-prompt-story show HEAD
```

**Redaction**: Replaces sensitive content with `<REDACTED BY USER>` in git notes and local logs.

If you've already pushed sensitive notes, redact locally and force-push:

```bash
git push -f origin refs/notes/prompt-story refs/notes/prompt-story-transcripts
```

## How Claude Code Stores Sessions

Claude Code saves conversations as JSONL files in:

```
~/.claude/projects/<encoded-path>/<session-id>.jsonl
```

Where `<encoded-path>` is your project path with `/` replaced by `-`.
Example: `/home/user/myapp` → `-home-user-myapp`

Each line is a JSON event with timestamps, making delta computation straightforward.
`git-prompt-story` reads these files, computes the delta relevant to your commit, and links it.

## Roadmap

- [x] Claude Code support
- [x] Viewer (CLI & HTML export)
- [x] GitHub Action (PR summaries & transcript pages)
- [ ] Cursor integration
- [ ] VS Code extension (show prompts inline)

## License

[Apache License 2.0](https://www.apache.org/licenses/LICENSE-2.0)
