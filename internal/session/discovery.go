package session

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// FindSessions discovers Claude Code sessions for a given repo path within the work period.
// Scans ALL session directories and greps for repo path references.
// Uses file mtime for fast pre-filtering before reading content.
// Returns sessions sorted by modified time (most recent first).
// If trace is non-nil, it records discovery details for explainability.
func FindSessions(repoPath string, startWork, endWork time.Time, trace *TraceContext) ([]ClaudeSession, error) {
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return nil, err
	}

	// Record trace info
	if trace != nil {
		trace.RepoPath = absPath
		trace.EncodedPath = encodePathForClaude(absPath)
	}

	// Find all session directories (full scan mode)
	candidateDirs, err := findAllSessionDirs()
	if err != nil {
		return nil, err
	}

	// Record candidate directories in trace
	if trace != nil {
		trace.CandidateDirs = candidateDirs
		if len(candidateDirs) > 0 {
			trace.SessionDir = candidateDirs[0] // Primary for backward compat
			trace.SessionDirExists = true
		} else {
			trace.SessionDirExists = false
		}
	}

	if len(candidateDirs) == 0 {
		return nil, nil
	}

	// Collect all session files from candidate directories
	var allFiles []string
	for _, dir := range candidateDirs {
		files, err := filepath.Glob(filepath.Join(dir, "*.jsonl"))
		if err != nil {
			continue
		}
		allFiles = append(allFiles, files...)
	}

	if trace != nil {
		trace.FoundFiles = allFiles
	}

	var sessions []ClaudeSession
	skippedByMtime := 0

	for _, f := range allFiles {
		// Fast pre-filter: check file mtime before reading content
		// If file hasn't been modified since before work started, skip it
		info, err := os.Stat(f)
		if err != nil {
			continue
		}
		mtime := info.ModTime()
		if mtime.Before(startWork) {
			skippedByMtime++
			continue
		}

		// Verify session belongs to this repo by checking first line cwd and timestamp
		if !sessionBelongsToRepo(f, absPath, endWork) {
			continue
		}

		id := strings.TrimSuffix(filepath.Base(f), ".jsonl")
		created, modified, _, err := ParseSessionMetadata(f)
		if err != nil {
			// Skip files we can't parse
			continue
		}

		// Time filter: session must overlap with work period
		// Session overlaps if: modified >= startWork AND created <= endWork
		if modified.Before(startWork) || created.After(endWork) {
			if trace != nil {
				st := trace.FindOrCreateSessionTrace(id)
				st.Path = f
				st.Created = created
				st.Modified = modified
				st.TimeFilterPassed = false
				if modified.Before(startWork) {
					st.TimeFilterReason = "FAIL (modified before work start)"
				} else {
					st.TimeFilterReason = "FAIL (created after work end)"
				}
				st.FinalReason = st.TimeFilterReason
			}
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
			st.TimeFilterPassed = true
			st.TimeFilterReason = "PASS (overlaps work period)"
		}
	}

	// Record mtime skip stats in trace
	if trace != nil {
		trace.SkippedByMtime = skippedByMtime
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
	if isToolResultContent(entry.Message.RawContent) {
		return false
	}

	// Commands (starting with /) count as user actions
	if strings.HasPrefix(msgText, "<command-name>") {
		return true
	}

	// Regular prompts with non-empty content
	return msgText != ""
}

// isToolResultContent checks if the raw content is a tool_result array
func isToolResultContent(rawContent []byte) bool {
	if len(rawContent) == 0 {
		return false
	}

	// Tool results are typically in an array with type "tool_result"
	var parts []struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(rawContent, &parts); err != nil {
		return false
	}

	for _, part := range parts {
		if part.Type == "tool_result" {
			return true
		}
	}
	return false
}

// findAllSessionDirs returns all session directories in ~/.claude/projects/
func findAllSessionDirs() ([]string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	projectsDir := filepath.Join(homeDir, ".claude", "projects")

	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var dirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, filepath.Join(projectsDir, entry.Name()))
		}
	}

	return dirs, nil
}

// sessionBelongsToRepo reads the first line of a session file to check:
// 1. If the session started after endWork (skip if so)
// 2. If the session's cwd is inside the repo path
// Returns true if session should be included.
// External folder sessions (cwd outside repo) are skipped for now (TODO: future enhancement).
func sessionBelongsToRepo(sessionPath, repoPath string, endWork time.Time) bool {
	file, err := os.Open(sessionPath)
	if err != nil {
		return false
	}
	defer file.Close()

	// Read first line only
	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	if !scanner.Scan() {
		return false
	}
	firstLine := scanner.Bytes()

	// Parse timestamp and cwd from first line
	var entry struct {
		Timestamp time.Time `json:"timestamp"`
		Cwd       string    `json:"cwd"`
	}
	if err := json.Unmarshal(firstLine, &entry); err != nil {
		return false
	}

	// Skip if session started after work ended
	if !entry.Timestamp.IsZero() && entry.Timestamp.After(endWork) {
		return false
	}

	// Check cwd using filepath for cross-OS portability
	cwd := filepath.Clean(entry.Cwd)
	repo := filepath.Clean(repoPath)

	// Exact match (repo root)
	if cwd == repo {
		return true
	}

	// Subfolder of repo
	if strings.HasPrefix(cwd, repo+string(filepath.Separator)) {
		return true
	}

	// External folder - skip for now (TODO: implement file path scanning)
	return false
}
