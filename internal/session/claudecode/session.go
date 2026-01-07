package claudecode

import (
	"os"
	"time"
)

// ClaudeCodeSession represents a discovered Claude Code session.
// Implements the session.Session interface.
type ClaudeCodeSession struct {
	id       string    // Session UUID (filename without .jsonl)
	path     string    // Full path to JSONL file
	created  time.Time // First timestamp in file
	modified time.Time // Last timestamp in file
}

// NewSession creates a new ClaudeCodeSession.
func NewSession(id, path string, created, modified time.Time) *ClaudeCodeSession {
	return &ClaudeCodeSession{
		id:       id,
		path:     path,
		created:  created,
		modified: modified,
	}
}

// GetID returns the session UUID.
func (s *ClaudeCodeSession) GetID() string {
	return s.id
}

// GetPath returns the full path to the session JSONL file.
func (s *ClaudeCodeSession) GetPath() string {
	return s.path
}

// GetPromptTool returns "claude-code".
func (s *ClaudeCodeSession) GetPromptTool() string {
	return PromptToolName
}

// GetCreated returns when the session was created.
func (s *ClaudeCodeSession) GetCreated() time.Time {
	return s.created
}

// GetModified returns when the session was last modified.
func (s *ClaudeCodeSession) GetModified() time.Time {
	return s.modified
}

// ReadContent reads the raw session content from the JSONL file.
func (s *ClaudeCodeSession) ReadContent() ([]byte, error) {
	return os.ReadFile(s.path)
}
