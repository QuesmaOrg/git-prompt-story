package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// FindSessions discovers Claude Code sessions for a given repo path
// Returns sessions sorted by modified time (most recent first)
// If trace is non-nil, it records discovery details for explainability.
func FindSessions(repoPath string, trace *TraceContext) ([]ClaudeSession, error) {
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return nil, err
	}

	claudeDir, err := getClaudeSessionDir(absPath)
	if err != nil {
		return nil, err
	}

	// Record trace info
	if trace != nil {
		trace.RepoPath = absPath
		trace.EncodedPath = encodePathForClaude(absPath)
		trace.SessionDir = claudeDir
	}

	// Check if directory exists
	if _, err := os.Stat(claudeDir); os.IsNotExist(err) {
		if trace != nil {
			trace.SessionDirExists = false
		}
		return nil, nil // No sessions directory = no sessions
	}

	if trace != nil {
		trace.SessionDirExists = true
	}

	files, err := filepath.Glob(filepath.Join(claudeDir, "*.jsonl"))
	if err != nil {
		return nil, err
	}

	if trace != nil {
		trace.FoundFiles = files
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

		// Initialize session trace
		if trace != nil {
			st := trace.FindOrCreateSessionTrace(id)
			st.Path = f
			st.Created = created
			st.Modified = modified
		}
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
// If trace is non-nil, it records the decision reason for each session.
func FilterSessionsByTime(sessions []ClaudeSession, startWork, endWork time.Time, trace *TraceContext) []ClaudeSession {
	var filtered []ClaudeSession
	for _, s := range sessions {
		// Session overlaps with work period if it was modified after work started
		// and created before work ended
		if !s.Modified.Before(startWork) && !s.Created.After(endWork) {
			filtered = append(filtered, s)
			if trace != nil {
				st := trace.FindOrCreateSessionTrace(s.ID)
				st.TimeFilterPassed = true
				st.TimeFilterReason = "PASS (overlaps work period)"
			}
		} else {
			if trace != nil {
				st := trace.FindOrCreateSessionTrace(s.ID)
				st.TimeFilterPassed = false
				if s.Modified.Before(startWork) {
					st.TimeFilterReason = "FAIL (modified before work start)"
				} else {
					st.TimeFilterReason = "FAIL (created after work end)"
				}
				st.FinalReason = st.TimeFilterReason
			}
		}
	}
	return filtered
}

// HasUserMessagesInRange checks if a session has any user messages within the time range
func HasUserMessagesInRange(sessionPath string, startWork, endWork time.Time) (bool, error) {
	has, _, err := CountUserMessagesInRangeForSession(sessionPath, startWork, endWork)
	return has, err
}

// CountUserMessagesInRangeForSession counts user messages in a single session within the time range
// Returns (hasMessages, count, error)
func CountUserMessagesInRangeForSession(sessionPath string, startWork, endWork time.Time) (bool, int, error) {
	content, err := ReadSessionContent(sessionPath)
	if err != nil {
		return false, 0, err
	}

	entries, err := ParseMessages(content)
	if err != nil {
		return false, 0, err
	}

	count := 0
	for _, entry := range entries {
		if entry.Type != "user" {
			continue
		}
		ts := entry.Timestamp
		if !ts.IsZero() && !ts.Before(startWork) && !ts.After(endWork) {
			count++
		}
	}
	return count > 0, count, nil
}

// FilterSessionsByUserMessages filters to only sessions with user messages in time range
// If trace is non-nil, it records the decision reason and message count for each session.
func FilterSessionsByUserMessages(sessions []ClaudeSession, startWork, endWork time.Time, trace *TraceContext) []ClaudeSession {
	var filtered []ClaudeSession
	for _, s := range sessions {
		hasMessages, count, err := CountUserMessagesInRangeForSession(s.Path, startWork, endWork)
		if err == nil && hasMessages {
			filtered = append(filtered, s)
			if trace != nil {
				st := trace.FindOrCreateSessionTrace(s.ID)
				st.UserMsgPassed = true
				st.UserMsgCount = count
				st.UserMsgReason = "PASS"
				st.Included = true
				st.FinalReason = "included"
			}
		} else {
			if trace != nil {
				st := trace.FindOrCreateSessionTrace(s.ID)
				st.UserMsgPassed = false
				st.UserMsgCount = count
				if strings.HasPrefix(s.ID, "agent-") {
					st.UserMsgReason = "FAIL (agent session)"
				} else {
					st.UserMsgReason = "FAIL (no user messages in range)"
				}
				st.FinalReason = st.UserMsgReason
			}
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

// CountUserActionsInRange counts actual user actions (prompts, commands, tool rejects)
// across all sessions within the time range, excluding agent sessions.
// This matches the counting logic used in CI summary.
func CountUserActionsInRange(sessions []ClaudeSession, startWork, endWork time.Time) int {
	count := 0
	for _, s := range sessions {
		// Skip agent sessions (IDs starting with "agent-")
		if strings.HasPrefix(s.ID, "agent-") {
			continue
		}

		content, err := ReadSessionContent(s.Path)
		if err != nil {
			continue
		}

		entries, err := ParseMessages(content)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			ts := entry.Timestamp
			if ts.IsZero() {
				continue
			}
			if ts.Before(startWork) || ts.After(endWork) {
				continue
			}

			if isUserActionEntry(entry) {
				count++
			}
		}
	}
	return count
}

// isUserActionEntry determines if a message entry represents a user action
// (prompt, command, or tool rejection) as opposed to tool results or system messages
func isUserActionEntry(entry MessageEntry) bool {
	// Skip meta/system-injected messages
	if entry.IsMeta {
		return false
	}

	// Only consider user and tool_reject entries
	switch entry.Type {
	case "tool_reject":
		return true
	case "user":
		// Continue to check content
	default:
		return false
	}

	if entry.Message == nil {
		return false
	}

	msgText := entry.Message.GetTextContent()

	// Skip local command outputs
	if strings.HasPrefix(msgText, "<local-command-stdout>") {
		return false
	}

	// Check if this is a tool_result (tool output returned as user message)
	// Rejections are tool_results with is_error=true, and they count as user actions
	isToolResult, isRejection := isToolResultContent(entry.Message.RawContent)
	if isToolResult && !isRejection {
		return false
	}
	if isRejection {
		return true
	}

	// Commands (starting with /) count as user actions
	if strings.HasPrefix(msgText, "<command-name>") {
		return true
	}

	// Regular prompts with non-empty content
	return msgText != ""
}

// isToolResultContent checks if raw content is a tool result and if it's a rejection
// Returns (isToolResult, isRejection)
func isToolResultContent(rawContent []byte) (bool, bool) {
	if len(rawContent) == 0 {
		return false, false
	}

	// Tool results are typically in an array with type "tool_result"
	var parts []struct {
		Type    string `json:"type"`
		IsError bool   `json:"is_error,omitempty"`
		Content string `json:"content,omitempty"`
	}
	if err := json.Unmarshal(rawContent, &parts); err != nil {
		return false, false
	}

	isToolResult := false
	isRejection := false
	for _, part := range parts {
		if part.Type == "tool_result" {
			isToolResult = true
			// Check if this is a rejection (is_error=true with rejection message)
			if part.IsError && strings.Contains(part.Content, "tool use was rejected") {
				isRejection = true
			}
		}
	}
	return isToolResult, isRejection
}
