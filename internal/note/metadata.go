package note

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/QuesmaOrg/git-prompt-story/internal/git"
	"github.com/QuesmaOrg/git-prompt-story/internal/session"
)

// PromptStoryNote is the JSON structure stored as a git note on commits
type PromptStoryNote struct {
	Version   int            `json:"v"`
	StartWork time.Time      `json:"start_work"`
	Sessions  []SessionEntry `json:"sessions"`
}

// SessionEntry describes one LLM session referenced by the note
type SessionEntry struct {
	// PromptTool is the identifier of the prompt tool (e.g., "claude-code", "gemini-cli").
	// JSON tag is "tool" for backward compatibility with existing notes.
	PromptTool string    `json:"tool"`
	ID         string    `json:"id"`
	Path       string    `json:"path"`
	Created    time.Time `json:"created"`
	Modified   time.Time `json:"modified"`
}

// Tool is an alias for PromptTool for backward compatibility.
// Deprecated: Use PromptTool instead.
func (s SessionEntry) Tool() string { return s.PromptTool }

// NewPromptStoryNote creates a new note from discovered sessions.
// Accepts sessions implementing the session.Session interface.
// isAmend should be true when amending a commit (affects start_work calculation)
// Optional startTime can be provided to use an explicit start time instead of calculating from git
func NewPromptStoryNote(sessions []session.Session, isAmend bool, startTime ...time.Time) *PromptStoryNote {
	n := &PromptStoryNote{
		Version:  1,
		Sessions: make([]SessionEntry, 0, len(sessions)),
	}

	// Use explicit start time if provided, otherwise calculate from git
	if len(startTime) > 0 && !startTime[0].IsZero() {
		n.StartWork = startTime[0]
	} else {
		n.StartWork, _ = git.CalculateWorkStartTime(isAmend)
	}

	for _, s := range sessions {
		promptTool := s.GetPromptTool()
		n.Sessions = append(n.Sessions, SessionEntry{
			PromptTool: promptTool,
			ID:         s.GetID(),
			Path:       GetTranscriptPath(promptTool, s.GetID()),
			Created:    s.GetCreated(),
			Modified:   s.GetModified(),
		})
	}

	return n
}

// NewPromptStoryNoteFromClaudeSessions creates a new note from Claude Code sessions.
// Deprecated: Use NewPromptStoryNote with session.Session interface instead.
func NewPromptStoryNoteFromClaudeSessions(sessions []session.ClaudeSession, isAmend bool, startTime ...time.Time) *PromptStoryNote {
	// Convert to Session interface
	sessionInterfaces := make([]session.Session, len(sessions))
	for i, s := range sessions {
		sessionInterfaces[i] = s
	}
	return NewPromptStoryNote(sessionInterfaces, isAmend, startTime...)
}

// ToJSON serializes the note to JSON
func (n *PromptStoryNote) ToJSON() ([]byte, error) {
	return json.MarshalIndent(n, "", "  ")
}

// GenerateSummary creates the commit message line
// Returns: "Prompt-Story: Used Claude Code (N prompts)" or "Prompt-Story: none"
func (n *PromptStoryNote) GenerateSummary(promptCount int) string {
	if len(n.Sessions) == 0 {
		return "Prompt-Story: none"
	}

	// Build prompt tool list
	tools := make(map[string]bool)
	for _, s := range n.Sessions {
		tools[s.PromptTool] = true
	}

	var toolNames []string
	for t := range tools {
		toolNames = append(toolNames, FormatPromptToolName(t))
	}
	sort.Strings(toolNames) // Consistent ordering

	return fmt.Sprintf("Prompt-Story: Used %s (%d user prompts)", strings.Join(toolNames, ", "), promptCount)
}

// GetTranscriptPath returns the path within the transcript tree for a session
func GetTranscriptPath(tool, sessionID string) string {
	return fmt.Sprintf("%s/%s.jsonl", tool, sessionID)
}

// FormatPromptToolName converts a prompt tool ID to its display name.
func FormatPromptToolName(promptTool string) string {
	switch promptTool {
	case "claude-code":
		return "Claude Code"
	case "gemini-cli":
		return "Gemini CLI"
	case "cursor":
		return "Cursor"
	case "codex":
		return "Codex"
	default:
		return promptTool
	}
}

// FormatToolName is an alias for FormatPromptToolName for backward compatibility.
// Deprecated: Use FormatPromptToolName instead.
func FormatToolName(tool string) string {
	return FormatPromptToolName(tool)
}
