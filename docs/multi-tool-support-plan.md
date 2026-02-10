# Multi-Tool Support Implementation Plan

## Summary
Add a generic `ToolProvider` interface with a new `Step` abstraction to support multiple LLM tools (Claude Code, Cursor, etc.). All Claude-specific code moves to the Claude provider.

## Key Design Decisions
- **New Step interface** - Common abstraction for user prompts, tool calls, responses
- **Provider parses JSONL** - Each provider converts its JSONL format to `[]Step`
- **Provider maps tools** - Each provider converts internal tool names to display names
- **Auto-detect** with config-based disabling via `git config`

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    Common Abstractions                       │
│  internal/step/step.go     - Step interface & types         │
│  internal/provider/        - ToolProvider interface         │
└─────────────────────────────────────────────────────────────┘
                              │
         ┌────────────────────┼────────────────────┐
         ▼                    ▼                    ▼
┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
│  Claude Code    │  │     Cursor      │  │    (Future)     │
│    Provider     │  │    Provider     │  │    Providers    │
├─────────────────┤  ├─────────────────┤  └─────────────────┘
│ - Discovery     │  │ - Discovery     │
│ - JSONL parsing │  │ - JSONL parsing │
│ - Tool mapping  │  │ - Tool mapping  │
│ - Filtering     │  │ - Filtering     │
└─────────────────┘  └─────────────────┘
```

---

## Implementation Steps

### Step 1: Create Step Abstraction
**File**: `internal/step/step.go` (NEW)

```go
package step

type StepType string

const (
    UserPrompt     StepType = "user_prompt"
    UserCommand    StepType = "user_command"
    UserDecision   StepType = "user_decision"
    ToolReject     StepType = "tool_reject"
    AssistantText  StepType = "assistant_text"
    ToolUse        StepType = "tool_use"
    ToolResult     StepType = "tool_result"
)

// Step is the common abstraction for all displayable entries
type Step struct {
    Time           time.Time
    Type           StepType
    Text           string    // Main display text
    ToolID         string    // Links ToolUse with ToolResult
    ToolName       string    // Display name (provider maps this)
    ToolInput      string    // Formatted input summary
    ToolOutput     string    // Tool result output
    DecisionHeader string    // For UserDecision
    DecisionAnswer string
    Truncated      bool
}

// IsUserAction returns true for user-initiated steps
func (s Step) IsUserAction() bool {
    return s.Type == UserPrompt || s.Type == UserCommand ||
           s.Type == UserDecision || s.Type == ToolReject
}
```

### Step 2: Create ToolProvider Interface
**File**: `internal/provider/provider.go` (NEW)

```go
package provider

type Session struct {
    ToolID   string
    ID       string
    Path     string
    Created  time.Time
    Modified time.Time
}

type ToolProvider interface {
    // Identity
    ID() string
    DisplayName() string

    // Session Discovery
    FindSessions(repoPath string) ([]Session, error)
    ReadSessionContent(sessionPath string) ([]byte, error)

    // Parsing - provider converts its JSONL to common Step format
    ParseSteps(content []byte) ([]step.Step, error)

    // Filtering
    FilterSessionsByTime(sessions []Session, start, end time.Time) []Session
    FilterSessionsByUserActivity(sessions []Session, start, end time.Time) []Session
    CountUserStepsInRange(sessions []Session, start, end time.Time) int

    // Detection
    IsAgentSession(sessionID string) bool
}
```

### Step 3: Create Provider Registry
**File**: `internal/provider/registry.go` (NEW)

- `Register(p ToolProvider)` - called from provider `init()`
- `GetAll() []ToolProvider`
- `GetEnabled() []ToolProvider` - respects git config `prompt-story.tools.<id>`
- `GetByID(id string) ToolProvider`

### Step 4: Implement Claude Code Provider
**File**: `internal/provider/claudecode/provider.go` (NEW)

Move from existing files:
- From `session/discovery.go`: `FindSessions()`, `getClaudeSessionDir()`, `encodePathForClaude()`
- From `session/discovery.go`: `FilterSessionsByTime()`, `FilterSessionsByUserMessages()`, `CountUserMessagesInRange()`
- From `session/parser.go`: `ReadSessionContent()`, `ParseMessages()`
- From `session/types.go`: `MessageEntry`, `Message`, `ContentPart`, etc.

New logic:
- `ParseSteps()` - converts Claude JSONL to `[]step.Step`
- Tool name mapping (Bash, Read, Write, Edit, etc. stay as-is)
- `IsAgentSession()` - checks "agent-" prefix

### Step 5: Create Cursor Provider Skeleton
**File**: `internal/provider/cursor/provider.go` (NEW)

- Skeleton with stub implementations
- User will add:
  - `FindSessions()` - discover Cursor sessions
  - `ParseSteps()` - convert Cursor JSONL to common Steps
  - Tool name mapping for Cursor's tools

### Step 6: Update Session Package
**File**: `internal/session/types.go` (MODIFY)

- Remove `ClaudeSession` (moved to provider)
- Remove `MessageEntry` and related types (moved to provider)
- Keep shared utilities if any

**File**: `internal/session/discovery.go` (DELETE or DEPRECATE)
- Logic moved to Claude provider

**File**: `internal/session/parser.go` (DELETE or DEPRECATE)
- Logic moved to Claude provider

### Step 7: Update CI Summary to Use Steps
**File**: `internal/ci/summary.go` (MODIFY)

- Replace `PromptEntry` usage with `step.Step`
- Update `analyzeSession()` to call `provider.ParseSteps()`
- `SessionSummary.Prompts` becomes `[]step.Step`

**File**: `internal/ci/html.go` (MODIFY)

- Update templates to use `step.Step` fields
- `entryCategory()` maps `step.StepType` to CSS class

### Step 8: Update prepare-commit-msg Hook
**File**: `internal/hooks/prepare_commit_msg.go` (MODIFY)

```go
import (
    _ "github.com/QuesmaOrg/git-prompt-story/internal/provider/claudecode"
    _ "github.com/QuesmaOrg/git-prompt-story/internal/provider/cursor"
)

// Replace session.FindSessions() with:
var allSessions []provider.Session
for _, p := range provider.GetEnabled() {
    sessions, _ := p.FindSessions(repoRoot)
    allSessions = append(allSessions, sessions...)
}
```

### Step 9: Update Transcript Storage
**File**: `internal/note/transcript.go` (MODIFY)

- `StoreTranscripts()` accepts `[]provider.Session` and provider registry
- Groups blobs by `session.ToolID`
- Returns `map[toolID]map[sessionID]blobSHA`

**File**: `internal/note/metadata.go` (MODIFY)

- `NewPromptStoryNote()` uses `session.ToolID` from provider
- `FormatToolName()` delegates to provider registry

---

## Files Summary

| File | Action | Description |
|------|--------|-------------|
| `internal/step/step.go` | NEW | Step interface and types |
| `internal/provider/provider.go` | NEW | ToolProvider interface |
| `internal/provider/registry.go` | NEW | Provider registry |
| `internal/provider/claudecode/provider.go` | NEW | Claude Code provider |
| `internal/provider/claudecode/parser.go` | NEW | Claude JSONL parsing |
| `internal/provider/claudecode/types.go` | NEW | Claude-specific types (MessageEntry, etc.) |
| `internal/provider/cursor/provider.go` | NEW | Cursor provider skeleton |
| `internal/session/types.go` | MODIFY | Remove Claude-specific types |
| `internal/session/discovery.go` | DELETE | Moved to provider |
| `internal/session/parser.go` | DELETE | Moved to provider |
| `internal/ci/summary.go` | MODIFY | Use step.Step |
| `internal/ci/html.go` | MODIFY | Use step.Step |
| `internal/hooks/prepare_commit_msg.go` | MODIFY | Use provider registry |
| `internal/note/transcript.go` | MODIFY | Support multiple tools |
| `internal/note/metadata.go` | MODIFY | Use provider for tool names |

---

## Configuration

```bash
# Disable Cursor
git config prompt-story.tools.cursor false

# Re-enable
git config --unset prompt-story.tools.cursor
```

Default: All registered providers are enabled.
