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

	// Only consider user, tool_reject, and queue-operation entries
	switch entry.Type {
	case "tool_reject":
		return true
	case "queue-operation":
		// Messages typed while Claude is working
		if entry.Operation == "enqueue" && entry.Content != "" {
			// Skip system notifications and commands
			if strings.HasPrefix(entry.Content, "<bash-notification>") || strings.HasPrefix(entry.Content, "/") {
				return false
			}
			return true
		}
		return false
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

// sessionBelongsToRepo checks if a session file belongs to the repo by:
// 1. Finding the first entry with cwd (may skip file-history-snapshot entries)
// 2. Checking if session started after endWork (skip if so)
// 3. Checking cwd relationship to repo:
//   - cwd == repo → INCLUDE
//   - cwd is subfolder of repo → INCLUDE
//   - repo is subfolder of cwd (parent folder case) → scan for Write/Edit operations
//   - else → SKIP
func sessionBelongsToRepo(sessionPath, repoPath string, endWork time.Time) bool {
	file, err := os.Open(sessionPath)
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	// Find first entry with cwd (skip file-history-snapshot entries that don't have cwd)
	var firstCwd string
	var firstTimestamp time.Time
	for scanner.Scan() {
		var entry struct {
			Timestamp time.Time `json:"timestamp"`
			Cwd       string    `json:"cwd"`
		}
		if json.Unmarshal(scanner.Bytes(), &entry) == nil && entry.Cwd != "" {
			firstCwd = entry.Cwd
			firstTimestamp = entry.Timestamp
			break
		}
	}

	if firstCwd == "" {
		return false
	}

	// Skip if session started after work ended
	if !firstTimestamp.IsZero() && firstTimestamp.After(endWork) {
		return false
	}

	// Check cwd using filepath for cross-OS portability
	cwd := filepath.Clean(firstCwd)
	repo := filepath.Clean(repoPath)

	// Exact match (repo root)
	if cwd == repo {
		return true
	}

	// Subfolder of repo
	if strings.HasPrefix(cwd, repo+string(filepath.Separator)) {
		return true
	}

	// Parent folder case: repo is under cwd
	// Scan subsequent lines for Write/Edit operations targeting the repo
	if strings.HasPrefix(repo, cwd+string(filepath.Separator)) {
		return scanForWritesToRepo(scanner, repo)
	}

	// External folder (not parent) - skip
	return false
}

// scanForWritesToRepo scans remaining lines for Write/Edit tool uses targeting the repo.
// Returns true if any Write or Edit operation has file_path inside repoPath.
func scanForWritesToRepo(scanner *bufio.Scanner, repoPath string) bool {
	for scanner.Scan() {
		var entry struct {
			Message struct {
				Content []struct {
					Type  string `json:"type"`
					Name  string `json:"name"`
					Input struct {
						FilePath string `json:"file_path"`
					} `json:"input"`
				} `json:"content"`
			} `json:"message"`
		}
		if json.Unmarshal(scanner.Bytes(), &entry) != nil {
			continue
		}

		for _, item := range entry.Message.Content {
			if item.Type != "tool_use" {
				continue
			}
			if item.Name != "Write" && item.Name != "Edit" {
				continue
			}
			filePath := filepath.Clean(item.Input.FilePath)
			// Check if file_path is inside repo (exact match or subfolder)
			if filePath == repoPath || strings.HasPrefix(filePath, repoPath+string(filepath.Separator)) {
				return true
			}
		}
	}
	return false
}
