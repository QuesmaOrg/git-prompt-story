package provider

import (
	"time"
)

// RawSession represents a discovered session from any tool
type RawSession struct {
	ID       string    // Session/composer UUID
	Tool     string    // Provider name (e.g., "claude-code", "cursor")
	Path     string    // Source path (file path or DB key)
	Created  time.Time // First timestamp
	Modified time.Time // Last timestamp
	RepoPath string    // Derived workspace path (for filtering)
}

// Provider discovers and reads LLM session data from a specific tool
type Provider interface {
	// Name returns the tool identifier (e.g., "claude-code", "cursor")
	Name() string

	// DiscoverSessions finds sessions for a repo path within the time window.
	// Returns sessions that belong to the repo (filtering by repo path).
	DiscoverSessions(repoPath string, startWork, endWork time.Time) ([]RawSession, error)

	// ReadTranscript reads the raw transcript content for storage.
	// Returns the native format (JSONL for Claude, JSON for Cursor).
	ReadTranscript(session RawSession) ([]byte, error)

	// FileExtension returns the file extension for stored transcripts
	FileExtension() string
}
