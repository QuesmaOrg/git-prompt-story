package claudecode

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/QuesmaOrg/git-prompt-story/internal/session"
)

// Discoverer implements session.SessionDiscoverer for Claude Code.
type Discoverer struct{}

// NewDiscoverer creates a new Claude Code session discoverer.
func NewDiscoverer() *Discoverer {
	return &Discoverer{}
}

// PromptTool returns the prompt tool identifier.
func (d *Discoverer) PromptTool() string {
	return PromptToolName
}

// DiscoverSessions finds Claude Code sessions for the given repo within the time range.
func (d *Discoverer) DiscoverSessions(repoPath string, startWork, endWork time.Time, trace *session.TraceContext) ([]session.Session, error) {
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

	var sessions []session.Session
	skippedByMtime := 0

	for _, f := range allFiles {
		// Fast pre-filter: check file mtime before reading content
		info, err := os.Stat(f)
		if err != nil {
			continue
		}
		mtime := info.ModTime()
		if mtime.Before(startWork) {
			skippedByMtime++
			continue
		}

		// Verify session belongs to this repo
		if !sessionBelongsToRepo(f, absPath, endWork) {
			continue
		}

		id := strings.TrimSuffix(filepath.Base(f), ".jsonl")
		created, modified, _, err := session.ParseSessionMetadata(f)
		if err != nil {
			continue
		}

		// Time filter: session must overlap with work period
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

		sessions = append(sessions, NewSession(id, f, created, modified))

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
		return sessions[i].GetModified().After(sessions[j].GetModified())
	})

	return sessions, nil
}

// FilterByUserMessages filters to sessions with user messages in time range.
func (d *Discoverer) FilterByUserMessages(sessions []session.Session, startWork, endWork time.Time, trace *session.TraceContext) []session.Session {
	var filtered []session.Session
	for _, s := range sessions {
		hasMessages, count, err := countUserMessagesInRangeForSession(s.GetPath(), startWork, endWork)
		if err == nil && hasMessages {
			filtered = append(filtered, s)
			if trace != nil {
				st := trace.FindOrCreateSessionTrace(s.GetID())
				st.UserMsgPassed = true
				st.UserMsgCount = count
				st.UserMsgReason = "PASS"
				st.Included = true
				st.FinalReason = "included"
			}
		} else {
			if trace != nil {
				st := trace.FindOrCreateSessionTrace(s.GetID())
				st.UserMsgPassed = false
				st.UserMsgCount = count
				if strings.HasPrefix(s.GetID(), "agent-") {
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

// CountUserActions counts user actions across all sessions within the time range.
func (d *Discoverer) CountUserActions(sessions []session.Session, startWork, endWork time.Time) int {
	count := 0
	for _, s := range sessions {
		// Skip agent sessions
		if strings.HasPrefix(s.GetID(), "agent-") {
			continue
		}

		content, err := s.ReadContent()
		if err != nil {
			continue
		}

		entries, err := session.ParseMessages(content)
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

// countUserMessagesInRangeForSession counts user messages in a single session.
func countUserMessagesInRangeForSession(sessionPath string, startWork, endWork time.Time) (bool, int, error) {
	content, err := session.ReadSessionContent(sessionPath)
	if err != nil {
		return false, 0, err
	}

	entries, err := session.ParseMessages(content)
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

// isUserActionEntry determines if a message entry represents a user action.
func isUserActionEntry(entry session.MessageEntry) bool {
	if entry.IsMeta {
		return false
	}

	switch entry.Type {
	case "tool_reject":
		return true
	case "queue-operation":
		if entry.Operation == "enqueue" && entry.Content != "" {
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

	if strings.HasPrefix(msgText, "<local-command-stdout>") {
		return false
	}

	isToolResult, isRejection := isToolResultContent(entry.Message.RawContent)
	if isToolResult && !isRejection {
		return false
	}
	if isRejection {
		return true
	}

	if strings.HasPrefix(msgText, "<command-name>") {
		return true
	}

	return msgText != ""
}

// isToolResultContent checks if raw content is a tool result and if it's a rejection.
func isToolResultContent(rawContent []byte) (bool, bool) {
	if len(rawContent) == 0 {
		return false, false
	}

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

// encodePathForClaude converts /Users/jacek/git/myapp to -Users-jacek-git-myapp
func encodePathForClaude(repoPath string) string {
	return strings.ReplaceAll(repoPath, string(filepath.Separator), "-")
}

// sessionBelongsToRepo checks if a session file belongs to the repo.
func sessionBelongsToRepo(sessionPath, repoPath string, endWork time.Time) bool {
	file, err := os.Open(sessionPath)
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

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

	if !firstTimestamp.IsZero() && firstTimestamp.After(endWork) {
		return false
	}

	cwd := filepath.Clean(firstCwd)
	repo := filepath.Clean(repoPath)

	if cwd == repo {
		return true
	}

	if strings.HasPrefix(cwd, repo+string(filepath.Separator)) {
		return true
	}

	if strings.HasPrefix(repo, cwd+string(filepath.Separator)) {
		return scanForWritesToRepo(scanner, repo)
	}

	return false
}

// scanForWritesToRepo scans for Write/Edit tool uses targeting the repo.
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
			if filePath == repoPath || strings.HasPrefix(filePath, repoPath+string(filepath.Separator)) {
				return true
			}
		}
	}
	return false
}
