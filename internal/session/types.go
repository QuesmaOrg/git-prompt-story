package session

import "time"

// ClaudeSession represents a discovered Claude Code session
type ClaudeSession struct {
	ID       string    // Session UUID (filename without .jsonl)
	Path     string    // Full path to JSONL file
	Created  time.Time // First timestamp in file
	Modified time.Time // Last timestamp in file
	Branch   string    // Git branch from session (optional)
}

// MessageEntry represents a single JSONL line from Claude Code
type MessageEntry struct {
	Type      string    `json:"type"`      // "user", "assistant", "file-history-snapshot"
	SessionID string    `json:"sessionId"`
	Timestamp time.Time `json:"timestamp"`
	CWD       string    `json:"cwd"`
	GitBranch string    `json:"gitBranch"`
	Snapshot  *Snapshot `json:"snapshot,omitempty"`
}

// Snapshot contains timestamp for file-history-snapshot entries
type Snapshot struct {
	Timestamp time.Time `json:"timestamp"`
}
