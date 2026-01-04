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
		created, modified, _, err := ParseSessionMetadata(f)
		if err != nil {
			// Skip files we can't parse
			continue
		}
		sessions = append(sessions, ClaudeSession{
			ID:       id,
			Path:     f,
			Created:  created,
			Modified: modified,
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

// HasUserMessagesInRange checks if a session has any user messages within the time range
func HasUserMessagesInRange(sessionPath string, startWork, endWork time.Time) (bool, error) {
	content, err := ReadSessionContent(sessionPath)
	if err != nil {
		return false, err
	}

	entries, err := ParseMessages(content)
	if err != nil {
		return false, err
	}

	for _, entry := range entries {
		if entry.Type != "user" {
			continue
		}
		ts := entry.Timestamp
		if !ts.IsZero() && !ts.Before(startWork) && !ts.After(endWork) {
			return true, nil
		}
	}
	return false, nil
}

// FilterSessionsByUserMessages filters to only sessions with user messages in time range
func FilterSessionsByUserMessages(sessions []ClaudeSession, startWork, endWork time.Time) []ClaudeSession {
	var filtered []ClaudeSession
	for _, s := range sessions {
		hasMessages, err := HasUserMessagesInRange(s.Path, startWork, endWork)
		if err == nil && hasMessages {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

// CountUserMessagesInRange counts user messages across all sessions within the time range
func CountUserMessagesInRange(sessions []ClaudeSession, startWork, endWork time.Time) int {
	count := 0
	for _, s := range sessions {
		content, err := ReadSessionContent(s.Path)
		if err != nil {
			continue
		}

		entries, err := ParseMessages(content)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.Type != "user" {
				continue
			}
			ts := entry.Timestamp
			if !ts.IsZero() && !ts.Before(startWork) && !ts.After(endWork) {
				count++
			}
		}
	}
	return count
}
