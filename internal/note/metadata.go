package note

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/QuesmaOrg/git-prompt-story/internal/git"
	"github.com/QuesmaOrg/git-prompt-story/internal/provider"
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

// NewPromptStoryNoteMulti creates a new note from discovered sessions across multiple providers
// isAmend should be true when amending a commit (affects start_work calculation)
// Optional startTime can be provided to use an explicit start time instead of calculating from git
func NewPromptStoryNoteMulti(sessions []provider.RawSession, isAmend bool, startTime ...time.Time) *PromptStoryNote {
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
		p := provider.Get(s.Tool)
		ext := ".jsonl"
		if p != nil {
			ext = p.FileExtension()
		}
		n.Sessions = append(n.Sessions, SessionEntry{
			Tool:     s.Tool,
			ID:       s.ID,
			Path:     GetTranscriptPathWithExt(s.Tool, s.ID, ext),
			Created:  s.Created,
			Modified: s.Modified,
		})
	}

	return n
}

// ToJSON serializes the note to JSON
func (n *PromptStoryNote) ToJSON() ([]byte, error) {
	return json.MarshalIndent(n, "", "  ")
}

// GenerateSummary creates the commit message line
// Returns: "Prompt-Story: Used Claude Code (N prompts) [version]" or "Prompt-Story: none [version]"
func (n *PromptStoryNote) GenerateSummary(promptCount int, version string) string {
	if len(n.Sessions) == 0 {
		return fmt.Sprintf("Prompt-Story: none [%s]", version)
	}

	// Build tool list
	tools := make(map[string]bool)
	for _, s := range n.Sessions {
		tools[s.Tool] = true
	}

	var toolNames []string
	for t := range tools {
		toolNames = append(toolNames, FormatToolName(t))
	}
	sort.Strings(toolNames) // Consistent ordering

	return fmt.Sprintf("Prompt-Story: Used %s (%d user prompts) [%s]", strings.Join(toolNames, ", "), promptCount, version)
}

// GetTranscriptPath returns the path within the transcript tree for a session
func GetTranscriptPath(tool, sessionID string) string {
	return fmt.Sprintf("%s/%s.jsonl", tool, sessionID)
}

// GetTranscriptPathWithExt returns the path with a specific file extension
func GetTranscriptPathWithExt(tool, sessionID, ext string) string {
	return fmt.Sprintf("%s/%s%s", tool, sessionID, ext)
}

// FormatToolName converts a tool ID to its display name
func FormatToolName(tool string) string {
	switch tool {
	case "claude-code":
		return "Claude Code"
	case "claude-cloud":
		return "Claude Cloud"
	case "cursor":
		return "Cursor"
	case "codex":
		return "Codex"
	default:
		return tool
	}
}
