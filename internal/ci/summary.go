package ci

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/QuesmaOrg/git-prompt-story/internal/git"
	"github.com/QuesmaOrg/git-prompt-story/internal/note"
	"github.com/QuesmaOrg/git-prompt-story/internal/session"
)

const transcriptsRef = "refs/notes/prompt-story-transcripts"
const notesRef = "refs/notes/commits"

// PromptEntry represents a single prompt or action in a session
type PromptEntry struct {
	Time      time.Time `json:"time"`
	Type      string    `json:"type"` // PROMPT, TOOL_REJECT, COMMAND
	Text      string    `json:"text"`
	Truncated bool      `json:"truncated,omitempty"`
}

// SessionSummary represents a summarized session within a commit
type SessionSummary struct {
	Tool     string        `json:"tool"`
	ID       string        `json:"id"`
	Start    time.Time     `json:"start"`
	End      time.Time     `json:"end"`
	Prompts  []PromptEntry `json:"prompts"`
}

// CommitSummary represents prompts for a single commit
type CommitSummary struct {
	SHA       string           `json:"sha"`
	ShortSHA  string           `json:"short_sha"`
	Subject   string           `json:"subject"`
	Sessions  []SessionSummary `json:"sessions"`
	StartWork time.Time        `json:"start_work"`
	EndWork   time.Time        `json:"end_work"`
}

// Summary represents the full analysis result
type Summary struct {
	Commits           []CommitSummary `json:"commits"`
	TotalPrompts      int             `json:"total_prompts"`
	CommitsWithNotes  int             `json:"commits_with_notes"`
	CommitsAnalyzed   int             `json:"commits_analyzed"`
}

// GenerateSummary analyzes commits in a range and extracts prompt data
func GenerateSummary(commitRange string, full bool) (*Summary, error) {
	// Resolve commit range to list of SHAs
	commits, err := resolveCommitRange(commitRange)
	if err != nil {
		return nil, err
	}

	summary := &Summary{
		Commits:         make([]CommitSummary, 0),
		CommitsAnalyzed: len(commits),
	}

	for _, sha := range commits {
		cs, err := analyzeCommit(sha, full)
		if err != nil {
			// Skip commits without notes
			continue
		}
		if len(cs.Sessions) > 0 {
			summary.Commits = append(summary.Commits, *cs)
			summary.CommitsWithNotes++
			for _, sess := range cs.Sessions {
				summary.TotalPrompts += len(sess.Prompts)
			}
		}
	}

	return summary, nil
}

// resolveCommitRange parses a commit range and returns the list of SHAs
func resolveCommitRange(spec string) ([]string, error) {
	// Check for range (contains ..)
	if strings.Contains(spec, "..") {
		commits, err := git.RevList(spec)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve range %s: %w", spec, err)
		}
		if len(commits) == 0 {
			return nil, fmt.Errorf("no commits in range %s", spec)
		}
		return commits, nil
	}

	// Single commit reference
	sha, err := git.ResolveCommit(spec)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve commit %s: %w", spec, err)
	}
	return []string{sha}, nil
}

// analyzeCommit extracts prompt data for a single commit
func analyzeCommit(sha string, full bool) (*CommitSummary, error) {
	// Get note attached to commit
	noteContent, err := git.GetNote(notesRef, sha)
	if err != nil {
		return nil, fmt.Errorf("no prompt-story note found for commit %s", sha[:7])
	}

	// Parse note JSON
	var psNote note.PromptStoryNote
	if err := json.Unmarshal([]byte(noteContent), &psNote); err != nil {
		return nil, fmt.Errorf("failed to parse note: %w", err)
	}

	// Get commit subject
	subject, _ := getCommitSubject(sha)

	// Get commit timestamp (end of work period)
	endWork, _ := git.GetPreviousCommitTimestamp(sha)
	if endWork.IsZero() {
		for _, sess := range psNote.Sessions {
			if sess.Modified.After(endWork) {
				endWork = sess.Modified
			}
		}
	}

	cs := &CommitSummary{
		SHA:       sha,
		ShortSHA:  sha[:7],
		Subject:   subject,
		Sessions:  make([]SessionSummary, 0),
		StartWork: psNote.StartWork,
		EndWork:   endWork,
	}

	// Process each session
	for _, sess := range psNote.Sessions {
		ss, err := analyzeSession(sess, psNote.StartWork, endWork, full)
		if err != nil {
			continue
		}
		if len(ss.Prompts) > 0 {
			cs.Sessions = append(cs.Sessions, *ss)
		}
	}

	return cs, nil
}

// analyzeSession extracts prompts from a session within the work period
func analyzeSession(sess note.SessionEntry, startWork, endWork time.Time, full bool) (*SessionSummary, error) {
	// Extract relative path from full ref path
	relPath := strings.TrimPrefix(sess.Path, transcriptsRef+"/")

	// Fetch transcript content
	content, err := git.GetBlobContent(transcriptsRef, relPath)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch transcript: %w", err)
	}

	// Parse messages
	entries, err := session.ParseMessages(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse messages: %w", err)
	}

	ss := &SessionSummary{
		Tool:    sess.Tool,
		ID:      sess.ID,
		Start:   sess.Created,
		End:     sess.Modified,
		Prompts: make([]PromptEntry, 0),
	}

	for _, entry := range entries {
		// Get timestamp
		ts := entry.Timestamp
		if ts.IsZero() && entry.Snapshot != nil {
			ts = entry.Snapshot.Timestamp
		}

		// Skip entries outside work period or without timestamp
		if ts.IsZero() || ts.Before(startWork) || ts.After(endWork) {
			continue
		}

		// Determine entry type and text
		var entryType, text string

		switch entry.Type {
		case "user":
			if entry.Message != nil {
				text = entry.Message.GetTextContent()
				if text != "" {
					entryType = "PROMPT"
				}
			}
		case "tool_reject":
			entryType = "TOOL_REJECT"
			if entry.Message != nil {
				text = entry.Message.GetTextContent()
			}
			if text == "" {
				text = "User rejected tool call"
			}
		}

		// Check for user commands
		if entry.Type == "user" && entry.Message != nil {
			msgText := entry.Message.GetTextContent()
			if strings.HasPrefix(msgText, "<command-name>") {
				start := strings.Index(msgText, "<command-name>") + len("<command-name>")
				end := strings.Index(msgText, "</command-name>")
				if end > start {
					cmdName := msgText[start:end]
					cmdName = strings.TrimPrefix(cmdName, "/")
					entryType = "COMMAND"
					text = "/" + cmdName
				}
			}
			// Skip local command output entries
			if strings.HasPrefix(msgText, "<local-command-stdout>") {
				entryType = ""
				text = ""
			}
		}

		if entryType != "" {
			pe := PromptEntry{
				Time: ts,
				Type: entryType,
				Text: text,
			}

			// Truncate if not full mode
			if !full && len(text) > 200 {
				pe.Text = text[:200]
				pe.Truncated = true
			}

			ss.Prompts = append(ss.Prompts, pe)
		}
	}

	return ss, nil
}

// getCommitSubject gets the first line of a commit message
func getCommitSubject(sha string) (string, error) {
	out, err := git.RunGit("log", "-1", "--format=%s", sha)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// RenderMarkdown generates markdown output for PR comment
func RenderMarkdown(summary *Summary, pagesURL string) string {
	var sb strings.Builder

	sb.WriteString("## Prompt Story\n\n")

	if summary.CommitsWithNotes == 0 {
		sb.WriteString("No prompt-story notes found in this PR.\n")
		return sb.String()
	}

	// Stats line
	sb.WriteString(fmt.Sprintf("**%d commit(s)** with LLM session data\n\n", summary.CommitsWithNotes))

	// Summary table
	sb.WriteString("| Commit | Tool | Prompts |\n")
	sb.WriteString("|--------|------|--------|\n")

	for _, commit := range summary.Commits {
		tools := make(map[string]bool)
		promptCount := 0
		for _, sess := range commit.Sessions {
			tools[formatToolName(sess.Tool)] = true
			promptCount += len(sess.Prompts)
		}
		var toolNames []string
		for t := range tools {
			toolNames = append(toolNames, t)
		}
		sb.WriteString(fmt.Sprintf("| %s | %s | %d |\n",
			commit.ShortSHA, strings.Join(toolNames, ", "), promptCount))
	}
	sb.WriteString("\n")

	// Expandable details for each commit
	for _, commit := range summary.Commits {
		subject := commit.Subject
		if len(subject) > 50 {
			subject = subject[:47] + "..."
		}
		sb.WriteString(fmt.Sprintf("<details>\n<summary><b>%s</b>: %s</summary>\n\n",
			commit.ShortSHA, subject))

		for i, sess := range commit.Sessions {
			if i > 0 {
				sb.WriteString("\n")
			}
			sb.WriteString(fmt.Sprintf("**Session %d** (%s) - %s to %s\n\n",
				i+1, formatToolName(sess.Tool),
				sess.Start.Local().Format("15:04"),
				sess.End.Local().Format("15:04")))

			for _, prompt := range sess.Prompts {
				text := prompt.Text
				// Escape markdown special chars in text
				text = strings.ReplaceAll(text, "\n", " ")
				if len(text) > 100 {
					text = text[:97] + "..."
				}
				sb.WriteString(fmt.Sprintf("- [%s] %s: %s\n",
					prompt.Time.Local().Format("15:04"), prompt.Type, text))
			}
		}
		sb.WriteString("\n</details>\n\n")
	}

	// Link to full transcripts
	if pagesURL != "" {
		sb.WriteString(fmt.Sprintf("[View full transcripts](%s)\n\n", pagesURL))
	}

	sb.WriteString("---\n*Generated by [git-prompt-story](https://github.com/QuesmaOrg/git-prompt-story)*\n")

	return sb.String()
}

// RenderJSON generates JSON output
func RenderJSON(summary *Summary) ([]byte, error) {
	return json.MarshalIndent(summary, "", "  ")
}

// formatToolName converts tool ID to display name
func formatToolName(tool string) string {
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
