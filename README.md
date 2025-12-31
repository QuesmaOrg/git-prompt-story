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
- **PII scrubbing** - Sensitive data (emails, API keys, passwords) automatically redacted ([configurable](docs/PII_SCRUBBING.md))
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
git config --global --add remote.origin.push 'refs/notes/commits:refs/notes/commits'
git config --global --add remote.origin.push 'refs/notes/prompt-story-transcripts:refs/notes/prompt-story-transcripts'
```

That's it. Future commits will automatically capture active LLM sessions.

### Repository Owner

Set up git-prompt-story for your team by adding a setup script and configuring GitHub AutoLinks.

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
git config --add remote.origin.push 'refs/notes/commits:refs/notes/commits'
git config --add remote.origin.push 'refs/notes/prompt-story-transcripts:refs/notes/prompt-story-transcripts'

echo "git-prompt-story configured for this repository"
```

Contributors run `./setup-prompt-story.sh` after cloning.

#### 2. Configure GitHub AutoLinks

Set up GitHub AutoLinks to convert `prompt-story-{sha}` references into clickable links:

1. Go to your repository **Settings** > **Autolink references**
2. Click **Add autolink reference**
3. Configure:
   - **Reference prefix**: `prompt-story-`
   - **Target URL**: `https://prompt-story.quesma.com/OWNER/REPO/prompt/<num>`
   - Replace `OWNER` and `REPO` with your repository owner and name
4. Check **Alphanumeric** (the reference contains letters and numbers)
5. Click **Add autolink reference**

Now commit messages containing `prompt-story-abc1234` will automatically link to the viewer.

#### 3. Install prompt server (optional for public repos, required for private)

For private repositories, host your own viewer:

```bash
docker run -d \
  -p 8080:8080 \
  -e GITHUB_TOKEN=your-token \
  ghcr.io/quesmaorg/git-prompt-story-server
```

Then update your AutoLink target URL to point to your hosted viewer.

Public repos can use the hosted service at `https://prompt-story.quesma.com`.

#### 4. Add GitHub Action (optional)

Add a GitHub Action to analyze LLM sessions and post summaries on PRs:

```yaml
# .github/workflows/prompt-story.yml
name: Prompt Story

on:
  pull_request:
    types: [opened, synchronize, reopened]

permissions:
  contents: write       # For GitHub Pages deployment
  pull-requests: write  # For PR comments

jobs:
  prompt-story:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - uses: QuesmaOrg/git-prompt-story/.github/actions/prompt-story@main
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          comment: true        # Post summary comment on PR
          deploy-pages: true   # Deploy full transcripts to GitHub Pages
```

This action:
- Fetches git notes from your repository
- Generates a summary of LLM tools used in the PR
- Posts an interactive HTML transcript to GitHub Pages
- Adds a comment with links to the full transcript

**Prerequisites:**
1. Push your git notes to remote: `git push origin 'refs/notes/*'`
2. After the first workflow run, enable GitHub Pages:
   - Settings → Pages → Source: Deploy from a branch
   - Select `gh-pages` branch (created automatically by the action)

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
│     │   "Prompt-Story: Used Claude Code | prompt-story-{sha}"   │
│     └── Save {nid} to .git/PENDING-PROMPT-STORY                 │
│                    │                                            │
│                    ▼                                            │
│  3. post-commit hook                                            │
│     ├── Read {nid} from .git/PENDING-PROMPT-STORY               │
│     ├── Attach note to HEAD via refs/notes/commits              │
│     └── Clean up .git/PENDING-PROMPT-STORY                      │
│                                                                 │
│  If no active sessions for this repo:                           │
│     └── Append: "Prompt-Story: none"                            │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Storage Format

Git Prompt Story uses two storage locations:

#### 1. Commit Metadata (`refs/notes/commits`)

Standard git notes attached to commits (default notes ref). Contains a lightweight JSON manifest:

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
- Stores notes in `refs/notes/commits` (standard git notes)
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

## Configuration

Config lives in `.git-prompt-story.json` or `~/.config/git-prompt-story.json`:

```json
{
  "auto_push_notes": false
}
```

Viewer URLs are handled via GitHub AutoLinks (see Repository Owner setup above).

## Viewer Integration

Notes are JSON - view them anywhere:

```bash
# Raw JSON
git notes show HEAD

# Local pretty-print
git-prompt-story show HEAD

# Or use the hosted viewer via GitHub AutoLink
# prompt-story-{sha} links are automatically converted to viewer URLs
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

- [x] Claude Code support
- [x] Local viewer (HTML export via `ci-html` command)
- [x] GitHub Action for PR summaries and transcript pages
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
