package session

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// FindSessions discovers Claude Code sessions for a given repo path
// Returns sessions sorted by modified time (most recent first)
func FindSessions(repoPath string) ([]ClaudeSession, error) {
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return nil, err
	}

	claudeDir, err := getClaudeSessionDir(absPath)
	if err != nil {
		return nil, err
	}

	// Check if directory exists
	if _, err := os.Stat(claudeDir); os.IsNotExist(err) {
		return nil, nil // No sessions directory = no sessions
	}

	files, err := filepath.Glob(filepath.Join(claudeDir, "*.jsonl"))
	if err != nil {
		return nil, err
	}

	var sessions []ClaudeSession
	for _, f := range files {
		id := strings.TrimSuffix(filepath.Base(f), ".jsonl")
		created, modified, branch, err := ParseSessionMetadata(f)
		if err != nil {
			// Skip files we can't parse
			continue
		}
		sessions = append(sessions, ClaudeSession{
			ID:       id,
			Path:     f,
			Created:  created,
			Modified: modified,
			Branch:   branch,
		})
	}

	// Sort by modified time (most recent first)
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Modified.After(sessions[j].Modified)
	})

	return sessions, nil
}

// getClaudeSessionDir returns the Claude Code sessions directory for a repo
// Path encoding: /Users/jacek/git/myapp -> -Users-jacek-git-myapp
func getClaudeSessionDir(repoPath string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	encodedPath := encodePathForClaude(repoPath)
	return filepath.Join(homeDir, ".claude", "projects", encodedPath), nil
}

// encodePathForClaude converts /Users/jacek/git/myapp to -Users-jacek-git-myapp
func encodePathForClaude(repoPath string) string {
	return strings.ReplaceAll(repoPath, string(filepath.Separator), "-")
}

// FilterSessionsByTime filters sessions to only those overlapping with the work period
// A session overlaps if: session.Modified >= startWork AND session.Created <= endWork
func FilterSessionsByTime(sessions []ClaudeSession, startWork, endWork time.Time) []ClaudeSession {
	var filtered []ClaudeSession
	for _, s := range sessions {
		// Session overlaps with work period if it was modified after work started
		// and created before work ended
		if !s.Modified.Before(startWork) && !s.Created.After(endWork) {
			filtered = append(filtered, s)
		}
	}
	return filtered
}
