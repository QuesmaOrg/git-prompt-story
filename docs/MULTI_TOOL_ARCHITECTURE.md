# Multi-Tool Architecture

## Design Principles

1. **Write path is minimal** - Store raw data with minimal transformation
2. **Read path does heavy lifting** - Parse, normalize, and format on demand
3. **Tool-agnostic storage** - Common note format, tool-specific transcript storage
4. **Plugin pattern** - Each tool implements a simple interface

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                         WRITE PATH                               │
│                    (minimal, at commit time)                     │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│   ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│   │ Claude Code  │  │    Cursor    │  │   Future...  │          │
│   │   Provider   │  │   Provider   │  │   Provider   │          │
│   └──────┬───────┘  └──────┬───────┘  └──────┬───────┘          │
│          │                 │                 │                   │
│          └────────────┬────┴────────────────┘                   │
│                       ▼                                          │
│              ┌────────────────┐                                  │
│              │ Session Registry│  ← Collects sessions from all   │
│              │                │    providers                     │
│              └────────┬───────┘                                  │
│                       ▼                                          │
│              ┌────────────────┐                                  │
│              │  Store Raw     │  ← Each tool's native format     │
│              │  Transcripts   │    in separate subtree           │
│              └────────┬───────┘                                  │
│                       ▼                                          │
│              ┌────────────────┐                                  │
│              │  Create Note   │  ← Tool-agnostic metadata        │
│              │  (metadata)    │                                  │
│              └────────────────┘                                  │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                          READ PATH                               │
│                    (heavy lifting, on demand)                    │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│              ┌────────────────┐                                  │
│              │   Read Note    │  ← Get metadata for commit       │
│              │   (metadata)   │                                  │
│              └────────┬───────┘                                  │
│                       ▼                                          │
│              ┌────────────────┐                                  │
│              │ For each session│                                 │
│              │ in note:       │                                  │
│              └────────┬───────┘                                  │
│                       ▼                                          │
│   ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│   │ Claude Code  │  │    Cursor    │  │   Future...  │          │
│   │   Parser     │  │    Parser    │  │    Parser    │          │
│   └──────┬───────┘  └──────┬───────┘  └──────┬───────┘          │
│          │                 │                 │                   │
│          └────────────┬────┴────────────────┘                   │
│                       ▼                                          │
│              ┌────────────────┐                                  │
│              │ Unified Entry  │  ← Common format for display     │
│              │    Format      │                                  │
│              └────────┬───────┘                                  │
│                       ▼                                          │
│              ┌────────────────┐                                  │
│              │   Render       │  ← Markdown, JSON, terminal      │
│              └────────────────┘                                  │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

## Provider Interface

Each tool implements a simple provider interface:

```go
// internal/provider/provider.go

type Provider interface {
    // Name returns the tool identifier (e.g., "claude-code", "cursor")
    Name() string

    // DiscoverSessions finds sessions for a repo path
    // Returns sessions that might be relevant (filtering happens later)
    DiscoverSessions(repoPath string) ([]RawSession, error)

    // ReadTranscript reads raw transcript content for storage
    // Returns the native format (JSONL for Claude, JSON for Cursor)
    ReadTranscript(session RawSession) ([]byte, error)
}

type RawSession struct {
    ID          string
    Tool        string    // Provider name
    Path        string    // Source path (file or DB key)
    Created     time.Time
    Modified    time.Time
    RepoPath    string    // Derived workspace path (for filtering)
}
```

## Parser Interface (Read Path)

Each tool implements a parser for the read path:

```go
// internal/parser/parser.go

type Parser interface {
    // Name returns the tool identifier
    Name() string

    // Parse converts raw transcript to unified entries
    // Filtering by time window happens here
    Parse(content []byte, startWork, endWork time.Time) ([]UnifiedEntry, error)
}

type UnifiedEntry struct {
    Time        time.Time
    Type        EntryType  // User, Assistant, ToolUse, ToolResult, Command
    Role        string     // "user", "assistant"
    Text        string     // Display text

    // Tool call details (optional)
    ToolName    string
    ToolInput   string
    ToolOutput  string
    ToolID      string

    // Metadata
    Model       string     // AI model used (if available)
    Rejected    bool       // User rejected this action
}

type EntryType int
const (
    EntryUser EntryType = iota
    EntryAssistant
    EntryToolUse
    EntryToolResult
    EntryCommand
)
```

## Storage Format

### Transcript Tree Structure

```
refs/notes/prompt-story-transcripts
├── claude-code/
│   ├── {session-id-1}.jsonl    ← Native JSONL format
│   └── {session-id-2}.jsonl
├── cursor/
│   ├── {composer-id-1}.json    ← Native JSON format
│   └── {composer-id-2}.json
└── {future-tool}/
    └── ...
```

### Note Format (unchanged)

```json
{
  "v": 1,
  "start_work": "2025-01-15T09:00:00Z",
  "sessions": [
    {
      "tool": "claude-code",
      "id": "113e0c55-64df-4b55-88f3-e06bcbc5b526",
      "path": "claude-code/113e0c55-64df-4b55-88f3-e06bcbc5b526.jsonl",
      "created": "2025-01-15T09:15:00Z",
      "modified": "2025-01-15T14:22:00Z"
    },
    {
      "tool": "cursor",
      "id": "e70d8f24-a68f-4cd0-b43e-13e3833682ec",
      "path": "cursor/e70d8f24-a68f-4cd0-b43e-13e3833682ec.json",
      "created": "2025-01-15T10:00:00Z",
      "modified": "2025-01-15T14:00:00Z"
    }
  ]
}
```

## File Structure

```
internal/
├── provider/
│   ├── provider.go          # Provider interface
│   ├── registry.go          # Provider registration
│   ├── claude/
│   │   └── claude.go        # Claude Code provider
│   └── cursor/
│       └── cursor.go        # Cursor provider
├── parser/
│   ├── parser.go            # Parser interface + UnifiedEntry
│   ├── registry.go          # Parser registration
│   ├── claude/
│   │   └── claude.go        # Claude Code parser
│   └── cursor/
│       └── cursor.go        # Cursor parser
├── session/
│   ├── types.go             # RawSession (slim, tool-agnostic)
│   └── filter.go            # Time/repo filtering logic
├── note/
│   ├── metadata.go          # Note creation (unchanged)
│   └── transcript.go        # Transcript storage (uses providers)
├── hooks/
│   └── prepare_commit_msg.go # Uses provider registry
├── show/
│   └── show.go              # Uses parser registry
└── ci/
    └── summary.go           # Uses parser registry
```

## Write Path Flow

```go
// internal/hooks/prepare_commit_msg.go

func PrepareCommitMsg(msgFile, source, sha string) error {
    repoPath := git.GetRepoRoot()

    // 1. Discover sessions from all providers
    var allSessions []session.RawSession
    for _, p := range provider.All() {
        sessions, err := p.DiscoverSessions(repoPath)
        if err != nil {
            log.Printf("warning: %s discovery failed: %v", p.Name(), err)
            continue
        }
        allSessions = append(allSessions, sessions...)
    }

    // 2. Filter by time window
    startWork := calculateStartWork(sha)
    endWork := time.Now()
    filtered := session.FilterByTime(allSessions, startWork, endWork)

    // 3. Store raw transcripts (native format per tool)
    blobs := make(map[string]string)
    for _, sess := range filtered {
        p := provider.Get(sess.Tool)
        content, _ := p.ReadTranscript(sess)
        content = scrubber.Scrub(content)  // PII scrubbing
        blobSHA := git.HashBlob(content)
        blobs[sess.Tool+"/"+sess.ID+ext(sess.Tool)] = blobSHA
    }

    // 4. Update transcript tree
    note.UpdateTranscriptTree(blobs)

    // 5. Create note
    n := note.NewPromptStoryNote(filtered, startWork)
    // ... store note
}

func ext(tool string) string {
    switch tool {
    case "cursor":
        return ".json"
    default:
        return ".jsonl"
    }
}
```

## Read Path Flow

```go
// internal/show/show.go

func ShowPrompts(commitRef string, full bool) error {
    commits := resolveCommitSpec(commitRef)

    for _, sha := range commits {
        n := note.GetNote(sha)
        endWork := git.GetCommitTime(sha)

        for _, sess := range n.Sessions {
            // Get parser for this tool
            p := parser.Get(sess.Tool)

            // Fetch raw transcript
            content := note.GetTranscript(sess.Path)

            // Parse to unified format (heavy lifting here)
            entries, _ := p.Parse(content, n.StartWork, endWork)

            // Display
            for _, e := range entries {
                displayEntry(e, full)
            }
        }
    }
}
```

## Adding a New Tool

To add support for a new tool (e.g., Windsurf, Copilot):

### 1. Create Provider

```go
// internal/provider/windsurf/windsurf.go

type Provider struct{}

func (p *Provider) Name() string { return "windsurf" }

func (p *Provider) DiscoverSessions(repoPath string) ([]session.RawSession, error) {
    // Find Windsurf session files/database
    // Extract workspace path from sessions
    // Filter to sessions matching repoPath
    // Return RawSession list
}

func (p *Provider) ReadTranscript(sess session.RawSession) ([]byte, error) {
    // Read raw content in native format
}

func init() {
    provider.Register(&Provider{})
}
```

### 2. Create Parser

```go
// internal/parser/windsurf/windsurf.go

type Parser struct{}

func (p *Parser) Name() string { return "windsurf" }

func (p *Parser) Parse(content []byte, start, end time.Time) ([]parser.UnifiedEntry, error) {
    // Parse native format
    // Convert to UnifiedEntry
    // Filter by time window
    // Return unified entries
}

func init() {
    parser.Register(&Parser{})
}
```

### 3. Import in Main

```go
// main.go or cmd/root.go

import (
    _ "git-prompt-story/internal/provider/claude"
    _ "git-prompt-story/internal/provider/cursor"
    _ "git-prompt-story/internal/provider/windsurf"  // Add this

    _ "git-prompt-story/internal/parser/claude"
    _ "git-prompt-story/internal/parser/cursor"
    _ "git-prompt-story/internal/parser/windsurf"    // Add this
)
```

## Benefits

1. **Isolation** - Each tool's logic is self-contained
2. **Native storage** - No lossy conversion on write
3. **Flexible parsing** - Each parser handles its own format quirks
4. **Easy testing** - Providers and parsers can be tested independently
5. **Graceful degradation** - One tool's failure doesn't break others
6. **Lazy parsing** - Heavy work only happens when reading

## Migration Path

1. Current code continues to work (Claude Code only)
2. Refactor existing code into `provider/claude` and `parser/claude`
3. Add `provider/cursor` and `parser/cursor`
4. Update hooks and show/ci to use registries
5. Existing notes remain compatible (tool field already exists)
