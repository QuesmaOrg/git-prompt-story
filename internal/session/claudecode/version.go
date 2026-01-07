package claudecode

import (
	"path/filepath"
)

const (
	// PromptToolName is the identifier for Claude Code sessions
	PromptToolName = "claude-code"

	// TranscriptVersion is the current version of Claude Code transcript format.
	// Bump this when the transcript format changes.
	TranscriptVersion = "v1"
)

// TranscriptPath returns the relative path for storing a session transcript.
// Format: claude-code/v1/<session-id>.jsonl
func TranscriptPath(sessionID string) string {
	return filepath.Join(PromptToolName, TranscriptVersion, sessionID+".jsonl")
}
