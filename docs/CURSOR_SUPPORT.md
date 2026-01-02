# Cursor Support Implementation Plan

## Overview

Add support for extracting Cursor IDE conversation logs alongside existing Claude Code support.

## Cursor Data Storage

### Location
- **macOS**: `~/Library/Application Support/Cursor/User/globalStorage/state.vscdb`
- **Windows**: `%APPDATA%/Cursor/User/globalStorage/state.vscdb`
- **Linux**: `~/.config/Cursor/User/globalStorage/state.vscdb`

### Format
SQLite database with table `cursorDiskKV`:
- **Key**: `composerData:<uuid>`
- **Value**: JSON blob containing conversation

### Conversation Structure
```json
{
  "composerId": "uuid",
  "createdAt": 1738420016413,
  "conversation": [
    {
      "type": 1,           // 1=user, 2=AI
      "bubbleId": "uuid",
      "text": "user prompt",
      "timingInfo": {"clientStartTime": 1738420122390, ...}
    },
    {
      "type": 2,
      "bubbleId": "uuid",
      "text": "AI response",
      "codeBlocks": [{"uri": {"fsPath": "/absolute/path/to/file.go"}}],
      "checkpoint": {"files": [{"uri": {"fsPath": "/absolute/path"}}]}
    }
  ]
}
```

## Workspace Linking

Cursor doesn't store workspace path directly. Derive it from:

1. **`codeBlocks[].uri.fsPath`** - absolute file paths from code edits
2. **`checkpoint.files[].uri.fsPath`** - created/modified files
3. **`relevantFiles`** - relative paths (less useful alone)

**Algorithm**:
1. Collect all `fsPath` values from conversation
2. Find common ancestor directory
3. Walk up to find `.git` directory (repo root)

## Implementation

### New Files
- `internal/session/cursor_discovery.go` - Find Cursor DB, list conversations
- `internal/session/cursor_parser.go` - Parse SQLite and extract conversations

### Changes to Existing Files
- `internal/session/types.go` - Add `Tool` field to `SessionInfo` ("claude-code" | "cursor")
- `internal/session/discovery.go` - Add `DiscoverCursorSessions()`
- `internal/hooks/prepare_commit_msg.go` - Call both discovery functions
- `internal/note/metadata.go` - Include tool type in session metadata

### Key Functions

```go
// cursor_discovery.go
func GetCursorDBPath() string
func DiscoverCursorSessions(repoPath string) ([]SessionInfo, error)

// cursor_parser.go
func ParseCursorConversation(composerData []byte) (*Conversation, error)
func ExtractWorkspacePath(conv *Conversation) string
func ConvertToJSONL(conv *Conversation) []byte  // Convert to Claude Code format for storage
```

## Data Flow

```
Cursor SQLite DB
       │
       ▼
┌─────────────────┐
│ Query composerData:* │
└─────────────────┘
       │
       ▼
┌─────────────────┐
│ Extract workspace │
│ from file paths   │
└─────────────────┘
       │
       ▼
┌─────────────────┐
│ Filter by repo  │
│ and time window │
└─────────────────┘
       │
       ▼
┌─────────────────┐
│ Convert to JSONL │
│ (unified format) │
└─────────────────┘
       │
       ▼
┌─────────────────┐
│ Store in git    │
│ notes (existing)│
└─────────────────┘
```

## Considerations

1. **No direct workspace link** - Must derive from file paths; conversations without edits can't be linked
2. **Timestamps** - Use `createdAt` (epoch ms) and `timingInfo.clientStartTime`
3. **Database locking** - Open SQLite in read-only mode; user should close Cursor
4. **Large conversations** - Some bubbles stored separately as `bubbleId:<composerId>:<bubbleId>`

## Existing Tools (Reference)

- [cursor-chat-export](https://github.com/somogyijanos/cursor-chat-export) - Python
- [cursor-history](https://github.com/S2thend/cursor-history) - Node.js
