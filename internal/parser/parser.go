package parser

import (
	"time"
)

// EntryType represents the type of conversation entry
type EntryType int

const (
	EntryUser EntryType = iota
	EntryAssistant
	EntryToolUse
	EntryToolResult
	EntryCommand
	EntrySystem
)

// UnifiedEntry represents a single conversation entry in a normalized format
type UnifiedEntry struct {
	Time      time.Time // When this entry occurred
	Type      EntryType // Type of entry
	Role      string    // "user", "assistant", "system"
	Text      string    // Display text
	IsMeta    bool      // System-injected message

	// Tool call details (optional)
	ToolName   string
	ToolInput  string
	ToolOutput string
	ToolID     string

	// Additional metadata
	Model    string // AI model used (if available)
	Rejected bool   // User rejected this action
}

// Parser converts raw transcript content to unified entries
type Parser interface {
	// Name returns the tool identifier (e.g., "claude-code", "cursor")
	Name() string

	// Parse converts raw transcript content to unified entries.
	// The time window (startWork, endWork) is used to filter entries.
	Parse(content []byte, startWork, endWork time.Time) ([]UnifiedEntry, error)

	// CountUserActions counts user actions (prompts, commands, rejections) in the content
	CountUserActions(content []byte, startWork, endWork time.Time) int
}
