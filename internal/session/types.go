package session

import (
	"encoding/json"
	"time"
)

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
	Message   *Message  `json:"message,omitempty"`
}

// Snapshot contains timestamp for file-history-snapshot entries
type Snapshot struct {
	Timestamp time.Time `json:"timestamp"`
}

// Message contains the actual prompt/response content
type Message struct {
	Role       string          `json:"role"` // "user", "assistant"
	RawContent json.RawMessage `json:"content"`
}

// GetTextContent extracts text content from the message
// Handles both string content and array of content parts
func (m *Message) GetTextContent() string {
	if m == nil || len(m.RawContent) == 0 {
		return ""
	}

	// Try to parse as string first
	var strContent string
	if err := json.Unmarshal(m.RawContent, &strContent); err == nil {
		return strContent
	}

	// Try to parse as array of content parts
	var parts []ContentPart
	if err := json.Unmarshal(m.RawContent, &parts); err == nil {
		var texts []string
		for _, part := range parts {
			if part.Type == "text" && part.Text != "" {
				texts = append(texts, part.Text)
			}
		}
		if len(texts) > 0 {
			return texts[0] // Return first text part
		}
	}

	return ""
}

// ContentPart represents a part of a message (text, tool use, etc.)
type ContentPart struct {
	Type string `json:"type"` // "text", "tool_use", "tool_result"
	Text string `json:"text,omitempty"`
}
