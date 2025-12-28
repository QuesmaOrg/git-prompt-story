# git-prompt-story

Store LLM prompts and sessions alongside your git commits using git notes.

## Why

When AI writes your code, the conversation is part of the story. git-prompt-story captures Claude Code sessions (and other LLM tools) in your git history - making prompts reviewable, searchable, and auditable without polluting your commit messages.

## Features

- **Git-native storage** - Uses git notes, no external databases
- **Automatic capture** - Hooks detect active LLM sessions on commit
- **Delta storage** - Only stores new conversation since last commit
- **Review before push** - Curate or redact notes before sharing
- **Viewer links** - Commit messages include links to rendered prompts
- **Multi-tool support** - Claude Code now, Cursor/Codex/Gemini planned

## Quick Start

```bash
pip install git-prompt-story

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
│  2. pre-commit hook                                             │
│     ├── Detect active LLM session                               │
│     ├── Compute delta since last commit                         │
│     ├── Create note blob (git hash-object -w)                   │
│     └── Save note ID to .git/GPS_PENDING_NOTE                   │
│                    │                                            │
│                    ▼                                            │
│  3. prepare-commit-msg hook                                     │
│     ├── Read note ID from .git/GPS_PENDING_NOTE                 │
│     ├── Generate summary from note content                      │
│     └── Append to commit message:                               │
│         "AI: 3 prompts, 847 tokens | View: <url>#<note-id>"     │
│                    │                                            │
│                    ▼                                            │
│  4. post-commit hook                                            │
│     ├── Read note ID from .git/GPS_PENDING_NOTE                 │
│     ├── Attach note to HEAD (git notes --ref=llm-prompts add)   │
│     └── Clean up .git/GPS_PENDING_NOTE                          │
│                                                                 │
│  If no active session or no changes since last commit:          │
│     └── Append minimal marker: "AI: none"                       │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Storage Format

Notes are stored as JSON in `refs/notes/llm-prompts`:

```json
{
  "v": 1,
  "source": "claude-code",
  "session": "a1b2c3",
  "range": [42, 58],
  "stats": {
    "prompts": 3,
    "tool_calls": 12,
    "input_tokens": 2048,
    "output_tokens": 4096
  },
  "summary": "Add user authentication with JWT...",
  "messages": [
    {"role": "user", "content": "Add user authentication..."},
    {"role": "assistant", "content": "I'll implement JWT-based..."}
  ]
}
```

### Auto-Detection

git-prompt-story finds active sessions by checking:

| Tool        | Location                                    | Status  |
|-------------|---------------------------------------------|---------|
| Claude Code | `~/.claude/projects/<encoded-path>/*.jsonl` | Done    |
| Cursor      | TBD                                         | Planned |
| Codex       | TBD                                         | Planned |
| Gemini CLI  | TBD                                         | Planned |

## Commands (DRAFT)

```bash
# Setup (DRAFT)
git-prompt-story init          # Install hooks in current repo
git-prompt-story init --global # Install hooks globally

# View (DRAFT)
git-prompt-story show          # Show note for HEAD
git-prompt-story show abc123   # Show note for specific commit
git-prompt-story log           # List commits with notes

# Curate (DRAFT - before pushing)
git-prompt-story review        # Interactive review of unpushed notes
git-prompt-story edit HEAD     # Edit note content
git-prompt-story remove HEAD   # Delete note from commit

# Push notes (DRAFT - git notes aren't pushed by default)
git-prompt-story push          # Push notes to origin
```

## Configuration

Config lives in `.git-prompt-story.json` or `~/.config/git-prompt-story.json`:

```json
{
  "viewer_url": "https://your-viewer.com/view/{repo}/{commit}",
  "summary_format": "AI: {prompts} prompts, {tokens} tokens",
  "include_messages": true,
  "auto_push_notes": false
}
```

### Viewer URL Templates

Variables available:
- `{repo}` - Repository name (owner/repo for GitHub)
- `{commit}` - Full commit SHA
- `{short}` - Short commit SHA (7 chars)

## Viewer Integration

Notes are JSON - view them anywhere:

```bash
# Raw JSON
git notes --ref=llm-prompts show HEAD

# Local pretty-print
git-prompt-story show HEAD

# Or host your own viewer and configure viewer_url
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

Notes live in a separate ref (`refs/notes/llm-prompts`) and must be explicitly pushed:

```bash
git push origin refs/notes/llm-prompts
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
