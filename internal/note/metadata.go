package note

import (
	"fmt"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/QuesmaOrg/git-prompt-story/internal/git"
	"github.com/QuesmaOrg/git-prompt-story/internal/session"
	"gopkg.in/yaml.v3"
)

// TranscriptsRefPrefix is the git ref prefix for transcripts
const TranscriptsRefPrefix = "refs/notes/prompt-story-transcripts"

// PromptStoryNote is the YAML structure stored as a git note on commits
type PromptStoryNote struct {
	Version   int       `yaml:"v"`
	StartWork time.Time `yaml:"start_work"`
	Sessions  []string  `yaml:"sessions"` // e.g. "claude-code/uuid.jsonl"
}

// NewPromptStoryNote creates a new note from discovered sessions
// isAmend should be true when amending a commit (affects start_work calculation)
func NewPromptStoryNote(sessions []session.ClaudeSession, isAmend bool) *PromptStoryNote {
	n := &PromptStoryNote{
		Version:  1,
		Sessions: make([]string, 0, len(sessions)),
	}

	// Calculate work start time from git reflog
	// This is the most recent of: previous commit time or branch switch time
	n.StartWork, _ = git.CalculateWorkStartTime(isAmend)

	for _, s := range sessions {
		n.Sessions = append(n.Sessions, GetSessionPath("claude-code", s.ID))
	}

	return n
}

// ToYAML serializes the note to YAML
func (n *PromptStoryNote) ToYAML() ([]byte, error) {
	return yaml.Marshal(n)
}

// GenerateSummary creates the commit message line
// Returns: "Prompt-Story: Used Claude Code | prompt-story-{sha}" or "Prompt-Story: none"
func (n *PromptStoryNote) GenerateSummary(noteSHA string) string {
	if len(n.Sessions) == 0 {
		return "Prompt-Story: none"
	}

	// Build tool list from session paths (e.g. "claude-code/uuid.jsonl")
	tools := make(map[string]bool)
	for _, sessPath := range n.Sessions {
		tool, _ := ParseSessionPath(sessPath)
		if tool != "" {
			tools[tool] = true
		}
	}

	var toolNames []string
	for t := range tools {
		// Convert tool ID to display name
		name := t
		switch t {
		case "claude-code":
			name = "Claude Code"
		case "cursor":
			name = "Cursor"
		case "codex":
			name = "Codex"
		}
		toolNames = append(toolNames, name)
	}
	sort.Strings(toolNames) // Consistent ordering

	summary := fmt.Sprintf("Prompt-Story: Used %s", strings.Join(toolNames, ", "))

	// Add AutoLink reference (GitHub will convert to clickable link)
	if noteSHA != "" {
		shortSHA := noteSHA
		if len(shortSHA) > 7 {
			shortSHA = shortSHA[:7]
		}
		summary += " | prompt-story-" + shortSHA
	}

	return summary
}

// GetSessionPath returns the relative path for a session (e.g. "claude-code/uuid.jsonl")
func GetSessionPath(tool, sessionID string) string {
	return fmt.Sprintf("%s/%s.jsonl", tool, sessionID)
}

// GetFullTranscriptPath returns the full ref path for a transcript
func GetFullTranscriptPath(sessionPath string) string {
	return TranscriptsRefPrefix + "/" + sessionPath
}

// ParseSessionPath extracts tool and session ID from a session path
// e.g. "claude-code/uuid.jsonl" -> ("claude-code", "uuid")
func ParseSessionPath(sessionPath string) (tool, sessionID string) {
	dir, file := path.Split(sessionPath)
	tool = strings.TrimSuffix(dir, "/")
	sessionID = strings.TrimSuffix(file, ".jsonl")
	return
}
