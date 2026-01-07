package session

import (
	"time"
)

// Session is the common interface for all prompt tool sessions.
// Each prompt tool (Claude Code, Gemini CLI, Cursor, etc.) provides
// its own implementation of this interface.
type Session interface {
	// GetID returns the unique session identifier (e.g., UUID)
	GetID() string

	// GetPath returns the original file path to the session data
	GetPath() string

	// GetPromptTool returns the prompt tool identifier (e.g., "claude-code", "gemini-cli")
	GetPromptTool() string

	// GetCreated returns when the session was created (first timestamp)
	GetCreated() time.Time

	// GetModified returns when the session was last modified (last timestamp)
	GetModified() time.Time

	// ReadContent reads the raw session content from its source
	ReadContent() ([]byte, error)
}

// SessionDiscoverer discovers sessions for a specific prompt tool.
// Each prompt tool provides its own implementation to find sessions
// in its specific storage location and format.
type SessionDiscoverer interface {
	// PromptTool returns the prompt tool identifier this discoverer handles
	PromptTool() string

	// DiscoverSessions finds sessions for the given repo within the time range.
	// Returns tool-specific Session implementations.
	DiscoverSessions(repoPath string, startWork, endWork time.Time, trace *TraceContext) ([]Session, error)

	// FilterByUserMessages filters to sessions with user activity in time range
	FilterByUserMessages(sessions []Session, startWork, endWork time.Time, trace *TraceContext) []Session

	// CountUserActions counts user actions (prompts, commands, etc.) in time range
	CountUserActions(sessions []Session, startWork, endWork time.Time) int
}

// PromptEntry represents a single prompt or action in a session.
// This is the common format that all prompt tool parsers produce.
// Defined here to avoid circular imports with the ci package.
type PromptEntry struct {
	Time         time.Time `json:"time"`
	Type         string    `json:"type"` // PROMPT, COMMAND, TOOL_REJECT, ASSISTANT, TOOL_USE, TOOL_RESULT, DECISION
	Text         string    `json:"text"`
	Truncated    bool      `json:"truncated,omitempty"`
	InWorkPeriod bool      `json:"in_work_period"` // true if within commit's work period
	ToolID       string    `json:"tool_id,omitempty"`
	ToolName     string    `json:"tool_name,omitempty"`
	ToolInput    string    `json:"tool_input,omitempty"`
	ToolOutput   string    `json:"tool_output,omitempty"`
	// For DECISION entries (AskUserQuestion)
	DecisionHeader            string `json:"decision_header,omitempty"`
	DecisionAnswer            string `json:"decision_answer,omitempty"`
	DecisionAnswerDescription string `json:"decision_answer_description,omitempty"`
}

// SessionParser converts tool-specific session format to common PromptEntry.
// Each prompt tool provides its own parser to handle its specific format.
type SessionParser interface {
	// PromptTool returns the prompt tool identifier this parser handles
	PromptTool() string

	// ParseSession converts session content to common PromptEntry format.
	// If full is true, content is not truncated.
	ParseSession(content []byte, startWork, endWork time.Time, full bool) ([]PromptEntry, error)

	// ParseMetadata extracts session metadata without full parsing.
	// Returns created time, modified time, git branch, and any error.
	ParseMetadata(sessionPath string) (created, modified time.Time, branch string, err error)
}
