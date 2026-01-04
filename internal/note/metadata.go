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
	Tool     string    `json:"tool"`
	ID       string    `json:"id"`
	Path     string    `json:"path"`
	Created  time.Time `json:"created"`
	Modified time.Time `json:"modified"`
}

// NewPromptStoryNote creates a new note from discovered sessions
// isAmend should be true when amending a commit (affects start_work calculation)
func NewPromptStoryNote(sessions []session.ClaudeSession, isAmend bool) *PromptStoryNote {
	note := &PromptStoryNote{
		Version:  1,
		Sessions: make([]SessionEntry, 0, len(sessions)),
	}

	// Calculate work start time from git reflog
	// This is the most recent of: previous commit time or branch switch time
	note.StartWork, _ = git.CalculateWorkStartTime(isAmend)

	for _, s := range sessions {
		note.Sessions = append(note.Sessions, SessionEntry{
			Tool:     "claude-code",
			ID:       s.ID,
			Path:     GetTranscriptPath("claude-code", s.ID),
			Created:  s.Created,
			Modified: s.Modified,
		})
	}

	return note
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

	// Build tool list
	tools := make(map[string]bool)
	for _, s := range n.Sessions {
		tools[s.Tool] = true
	}

	var toolNames []string
	for t := range tools {
		// Convert tool ID to display name
		name := t
		switch t {
		case "claude-code":
			name = "Claude Code"
		case "claude-cloud":
			name = "Claude Cloud"
		case "cursor":
			name = "Cursor"
		case "codex":
			name = "Codex"
		}
		toolNames = append(toolNames, name)
	}
	sort.Strings(toolNames) // Consistent ordering

	return fmt.Sprintf("Prompt-Story: Used %s (%d prompts)", strings.Join(toolNames, ", "), promptCount)
}

// NewPromptStoryNoteWithTime creates a new note with an explicit start time
// Used by repair command when recreating notes for past commits
func NewPromptStoryNoteWithTime(sessions []session.ClaudeSession, startWork time.Time) *PromptStoryNote {
	note := &PromptStoryNote{
		Version:   1,
		StartWork: startWork,
		Sessions:  make([]SessionEntry, 0, len(sessions)),
	}

	for _, s := range sessions {
		note.Sessions = append(note.Sessions, SessionEntry{
			Tool:     "claude-code",
			ID:       s.ID,
			Path:     GetTranscriptPath("claude-code", s.ID),
			Created:  s.Created,
			Modified: s.Modified,
		})
	}

	return note
}

// GetTranscriptPath returns the path within the transcript tree for a session
func GetTranscriptPath(tool, sessionID string) string {
	return fmt.Sprintf("%s/%s.jsonl", tool, sessionID)
}
