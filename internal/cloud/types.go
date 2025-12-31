package cloud

import (
	"encoding/json"
	"time"
)

// Session represents a Claude Code Cloud session from the API
type Session struct {
	ID             string         `json:"id"`
	Title          string         `json:"title"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	SessionStatus  string         `json:"session_status"`
	Type           string         `json:"type"`
	EnvironmentID  string         `json:"environment_id"`
	SessionContext SessionContext `json:"session_context"`
}

// SessionContext contains context about the session's environment
type SessionContext struct {
	Model    string    `json:"model"`
	CWD      string    `json:"cwd"`
	Outcomes []Outcome `json:"outcomes"`
	Sources  []Source  `json:"sources"`
}

// Outcome represents a git repository outcome
type Outcome struct {
	Type    string  `json:"type"` // "git_repository"
	GitInfo GitInfo `json:"git_info"`
}

// GitInfo contains git repository information
type GitInfo struct {
	Type     string   `json:"type"` // "github"
	Repo     string   `json:"repo"` // e.g. "QuesmaOrg/git-prompt-story"
	Branches []string `json:"branches"`
}

// Source represents a session source
type Source struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

// Event represents an event from a cloud session
type Event struct {
	UUID            string          `json:"uuid"`
	Type            string          `json:"type"` // "user", "assistant", "system", "result", "env_manager_log"
	SessionID       string          `json:"session_id"`
	ParentToolUseID *string         `json:"parent_tool_use_id"`
	IsReplay        bool            `json:"isReplay,omitempty"`
	Message         *EventMessage   `json:"message,omitempty"`
	Data            json.RawMessage `json:"data,omitempty"` // For non-message events
}

// EventMessage represents the message content in an event
type EventMessage struct {
	ID             string          `json:"id,omitempty"`
	Role           string          `json:"role"` // "user", "assistant"
	Content        json.RawMessage `json:"content"`
	Model          string          `json:"model,omitempty"`
	Type           string          `json:"type,omitempty"`
	StopReason     *string         `json:"stop_reason,omitempty"`
	StopSequence   *string         `json:"stop_sequence,omitempty"`
	Usage          *Usage          `json:"usage,omitempty"`
}

// Usage contains token usage information
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// SessionsResponse is the paginated response from /v1/sessions
type SessionsResponse struct {
	Data    []Session `json:"data"`
	FirstID string    `json:"first_id"`
	LastID  string    `json:"last_id"`
	HasMore bool      `json:"has_more"`
}

// EventsResponse is the paginated response from /v1/sessions/{id}/events
type EventsResponse struct {
	Data    []Event `json:"data"`
	FirstID string  `json:"first_id"`
	LastID  string  `json:"last_id"`
	HasMore bool    `json:"has_more"`
}
