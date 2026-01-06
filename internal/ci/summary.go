package ci

import (
	"encoding/json"
	"fmt"
	"html"
	"sort"
	"strings"
	"time"

	"github.com/QuesmaOrg/git-prompt-story/internal/git"
	"github.com/QuesmaOrg/git-prompt-story/internal/note"
	"github.com/QuesmaOrg/git-prompt-story/internal/session"
)

const (
	// Size limits for markdown output to stay under GitHub's 65KB comment limit
	maxUserPromptsSize = 20000 // Budget for user prompts section
	maxAllStepsSize    = 40000 // Budget for all steps section
)

// PromptEntry represents a single prompt or action in a session
type PromptEntry struct {
	Time         time.Time `json:"time"`
	Type         string    `json:"type"` // PROMPT, COMMAND, TOOL_REJECT, ASSISTANT, TOOL_USE, TOOL_RESULT, DECISION
	Text         string    `json:"text"`
	Truncated    bool      `json:"truncated,omitempty"`
	InWorkPeriod bool      `json:"in_work_period"` // true if within commit's work period
	ToolID       string    `json:"tool_id,omitempty"`     // For TOOL_USE/TOOL_RESULT: links them together
	ToolName     string    `json:"tool_name,omitempty"`   // For TOOL_USE: the tool name (Bash, Edit, etc.)
	ToolInput    string    `json:"tool_input,omitempty"`  // For TOOL_USE: the tool input/command
	ToolOutput   string    `json:"tool_output,omitempty"` // For TOOL_RESULT: the tool output
	// For DECISION entries (AskUserQuestion)
	DecisionHeader string `json:"decision_header,omitempty"` // Question header (e.g., "Version")
	DecisionAnswer string `json:"decision_answer,omitempty"` // User's selected answer
}

// SessionSummary represents a summarized session within a commit
type SessionSummary struct {
	Tool     string        `json:"tool"`
	ID       string        `json:"id"`
	IsAgent  bool          `json:"is_agent"`  // True if this is an agent/subagent session
	Start    time.Time     `json:"start"`
	End      time.Time     `json:"end"`
	Prompts  []PromptEntry `json:"prompts"`
}

// IsAgentSession returns true if the session ID indicates an agent session
// Agent sessions have IDs prefixed with "agent-"
func IsAgentSession(sessionID string) bool {
	return strings.HasPrefix(sessionID, "agent-")
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
	Commits          []CommitSummary `json:"commits"`
	TotalPrompts     int             `json:"total_prompts"`       // Kept for backward compatibility (equals TotalSteps)
	TotalUserPrompts int             `json:"total_user_prompts"`  // Count of user actions in main sessions only
	TotalAgentPrompts int            `json:"total_agent_prompts"` // Count of user actions in agent sessions
	TotalSteps       int             `json:"total_steps"`         // Count of all entries
	TotalAgentSessions int           `json:"total_agent_sessions"` // Count of agent sessions
	CommitsWithNotes int             `json:"commits_with_notes"`
	CommitsAnalyzed  int             `json:"commits_analyzed"`
}

// GenerateSummary analyzes commits in a range and extracts prompt data
func GenerateSummary(commitRange string, full bool) (*Summary, error) {
	// Resolve commit range to list of SHAs
	commits, err := git.ResolveCommitSpec(commitRange)
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
				stepCount := len(sess.Prompts)
				userPromptCount := countUserPrompts(sess.Prompts)
				summary.TotalSteps += stepCount
				summary.TotalPrompts += stepCount // Keep for backward compatibility

				// Separate counts for main vs agent sessions
				if sess.IsAgent {
					summary.TotalAgentPrompts += userPromptCount
					summary.TotalAgentSessions++
				} else {
					summary.TotalUserPrompts += userPromptCount
				}
			}
		}
	}

	return summary, nil
}

// analyzeCommit extracts prompt data for a single commit
func analyzeCommit(sha string, full bool) (*CommitSummary, error) {
	// Get note attached to commit
	noteContent, err := note.GetNote(sha)
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

// analyzeSession extracts all entries from a session, marking which are in work period
func analyzeSession(sess note.SessionEntry, startWork, endWork time.Time, full bool) (*SessionSummary, error) {
	// Extract relative path from full ref path
	relPath := strings.TrimPrefix(sess.Path, note.TranscriptsRef+"/")

	// Fetch transcript content
	content, err := git.GetBlobContent(note.TranscriptsRef, relPath)
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
		IsAgent: IsAgentSession(sess.ID),
		Start:   sess.Created,
		End:     sess.Modified,
		Prompts: make([]PromptEntry, 0),
	}

	// Map to track tool use entries by ID for linking with results
	toolUseEntries := make(map[string]*PromptEntry)

	// Map to track AskUserQuestion entries by tool ID for linking with answers
	// Key is tool ID, value is slice of indices into ss.Prompts for the DECISION entries
	askUserQuestionEntries := make(map[string][]int)

	for _, entry := range entries {
		// Get timestamp
		ts := entry.Timestamp
		if ts.IsZero() && entry.Snapshot != nil {
			ts = entry.Snapshot.Timestamp
		}

		// Skip entries without timestamp
		if ts.IsZero() {
			continue
		}

		// Determine if in work period
		inWorkPeriod := !ts.Before(startWork) && !ts.After(endWork)

		switch entry.Type {
		case "user":
			if entry.Message != nil {
				msgText := entry.Message.GetTextContent()

				// Check for commands
				if strings.HasPrefix(msgText, "<command-name>") {
					start := strings.Index(msgText, "<command-name>") + len("<command-name>")
					end := strings.Index(msgText, "</command-name>")
					if end > start {
						cmdName := msgText[start:end]
						cmdName = strings.TrimPrefix(cmdName, "/")
						pe := PromptEntry{
							Time:         ts,
							Type:         "COMMAND",
							Text:         "/" + cmdName,
							InWorkPeriod: inWorkPeriod,
						}
						if inWorkPeriod {
							ss.Prompts = append(ss.Prompts, pe)
						}
						continue
					}
				}

				// Skip local command output entries
				if strings.HasPrefix(msgText, "<local-command-stdout>") {
					continue
				}

				// Skip meta/system-injected messages
				if entry.IsMeta {
					continue
				}

				// Check for tool results
				toolResults := parseToolResults(entry.Message.RawContent)
				if len(toolResults) > 0 {
					hasRejection := false
					for _, tr := range toolResults {
						// Check if this is a tool rejection (is_error=true with rejection message)
						if tr.IsError && strings.Contains(tr.Content, "tool use was rejected") {
							// Extract user's rejection message if present
							text := "User rejected tool call"
							if idx := strings.Index(tr.Content, "user said:\n"); idx != -1 {
								text = strings.TrimSpace(tr.Content[idx+len("user said:\n"):])
							}
							pe := PromptEntry{
								Time:         ts,
								Type:         "TOOL_REJECT",
								Text:         text,
								InWorkPeriod: inWorkPeriod,
							}
							if inWorkPeriod {
								ss.Prompts = append(ss.Prompts, pe)
							}
							hasRejection = true
							continue
						}
						// Find and update the corresponding tool use entry
						if toolUse, ok := toolUseEntries[tr.ToolUseID]; ok {
							toolUse.ToolOutput = tr.Content
						}
						// Check if this is an answer to AskUserQuestion
						if indices, ok := askUserQuestionEntries[tr.ToolUseID]; ok {
							// Get answers from ToolUseResult (preserved by scrubber for decisions)
							if entry.ToolUseResult != nil && entry.ToolUseResult.Answers != nil {
								for _, idx := range indices {
									if idx < len(ss.Prompts) {
										question := ss.Prompts[idx].Text
										if answer, found := entry.ToolUseResult.Answers[question]; found {
											ss.Prompts[idx].DecisionAnswer = answer
										}
									}
								}
							}
						}
					}
					if !hasRejection {
						continue
					}
				}

				// Regular user prompt
				if msgText != "" {
					pe := PromptEntry{
						Time:         ts,
						Type:         "PROMPT",
						Text:         msgText,
						InWorkPeriod: inWorkPeriod,
					}
					if !full && len(pe.Text) > 2000 {
						pe.Text = pe.Text[:2000] + "...[TRUNCATED]"
						pe.Truncated = true
					}
					if inWorkPeriod {
						ss.Prompts = append(ss.Prompts, pe)
					}
				}
			}

		case "tool_reject":
			text := "User rejected tool call"
			if entry.Message != nil {
				if t := entry.Message.GetTextContent(); t != "" {
					text = t
				}
			}
			pe := PromptEntry{
				Time:         ts,
				Type:         "TOOL_REJECT",
				Text:         text,
				InWorkPeriod: inWorkPeriod,
			}
			if inWorkPeriod {
				ss.Prompts = append(ss.Prompts, pe)
			}

		case "assistant":
			if entry.Message != nil {
				entryType, text, toolUses := parseAssistantContent(entry.Message.RawContent)

				if len(toolUses) > 0 {
					// Create an entry for each tool use
					for _, tool := range toolUses {
						// Special handling for AskUserQuestion - create DECISION entries
						if tool.Name == "AskUserQuestion" {
							var askInput AskUserQuestionInput
							if err := json.Unmarshal(tool.RawInput, &askInput); err == nil && len(askInput.Questions) > 0 {
								var indices []int
								for _, q := range askInput.Questions {
									pe := PromptEntry{
										Time:           ts,
										Type:           "DECISION",
										Text:           q.Question,
										ToolID:         tool.ID,
										DecisionHeader: q.Header,
										InWorkPeriod:   inWorkPeriod,
									}
									if inWorkPeriod {
										ss.Prompts = append(ss.Prompts, pe)
										indices = append(indices, len(ss.Prompts)-1)
									}
								}
								if len(indices) > 0 {
									askUserQuestionEntries[tool.ID] = indices
								}
								continue
							}
						}

						pe := PromptEntry{
							Time:         ts,
							Type:         "TOOL_USE",
							Text:         tool.Name,
							ToolID:       tool.ID,
							ToolName:     tool.Name,
							ToolInput:    tool.Input,
							InWorkPeriod: inWorkPeriod,
						}
						if !full && len(pe.ToolInput) > 500 {
							pe.ToolInput = pe.ToolInput[:500] + "...[TRUNCATED]"
							pe.Truncated = true
						}
						if inWorkPeriod {
							ss.Prompts = append(ss.Prompts, pe)
							// Track for linking with results
							toolUseEntries[tool.ID] = &ss.Prompts[len(ss.Prompts)-1]
						}
					}
				} else if entryType == "ASSISTANT" && text != "" {
					pe := PromptEntry{
						Time:         ts,
						Type:         "ASSISTANT",
						Text:         text,
						InWorkPeriod: inWorkPeriod,
					}
					if !full && len(pe.Text) > 2000 {
						pe.Text = pe.Text[:2000] + "...[TRUNCATED]"
						pe.Truncated = true
					}
					if inWorkPeriod {
						ss.Prompts = append(ss.Prompts, pe)
					}
				}
			}

		case "queue-operation":
			// Messages typed by user while Claude is working
			// Only include "enqueue" operations with actual content
			if entry.Operation == "enqueue" && entry.Content != "" {
				// Skip system notifications (bash notifications, etc.)
				if strings.HasPrefix(entry.Content, "<bash-notification>") {
					continue
				}
				// Skip commands (they'll be processed as separate entries)
				if strings.HasPrefix(entry.Content, "/") {
					continue
				}
				pe := PromptEntry{
					Time:         ts,
					Type:         "PROMPT",
					Text:         entry.Content,
					InWorkPeriod: inWorkPeriod,
				}
				if !full && len(pe.Text) > 2000 {
					pe.Text = pe.Text[:2000] + "...[TRUNCATED]"
					pe.Truncated = true
				}
				if inWorkPeriod {
					ss.Prompts = append(ss.Prompts, pe)
				}
			}
		}
	}

	return ss, nil
}

// ToolResultInfo holds extracted tool result information
type ToolResultInfo struct {
	ToolUseID string
	Content   string
	IsError   bool
}

// parseToolResults extracts tool_result entries from user message content
func parseToolResults(rawContent json.RawMessage) []ToolResultInfo {
	if len(rawContent) == 0 {
		return nil
	}

	// Tool results are typically in an array
	var parts []struct {
		Type       string `json:"type"`
		ToolUseID  string `json:"tool_use_id"`
		Content    any    `json:"content"`
		IsError    bool   `json:"is_error,omitempty"`
	}
	if err := json.Unmarshal(rawContent, &parts); err != nil {
		return nil
	}

	var results []ToolResultInfo
	for _, part := range parts {
		if part.Type == "tool_result" && part.ToolUseID != "" {
			content := extractToolResultContent(part.Content)
			if content != "" || part.IsError {
				results = append(results, ToolResultInfo{
					ToolUseID: part.ToolUseID,
					Content:   content,
					IsError:   part.IsError,
				})
			}
		}
	}

	return results
}

// extractToolResultContent extracts text content from tool result
func extractToolResultContent(content any) string {
	if content == nil {
		return ""
	}

	var result string

	// Content could be a string
	if s, ok := content.(string); ok {
		result = s
	} else if arr, ok := content.([]any); ok {
		// Content could be an array of content blocks
		var texts []string
		for _, item := range arr {
			if m, ok := item.(map[string]any); ok {
				if m["type"] == "text" {
					if text, ok := m["text"].(string); ok {
						texts = append(texts, text)
					}
				}
			}
		}
		result = strings.Join(texts, "\n")
	}

	// Clean up arrow characters from cat -n output format
	result = strings.ReplaceAll(result, "→", " ")

	return result
}

// ToolUseInfo holds extracted information about a tool use
type ToolUseInfo struct {
	ID       string
	Name     string
	Input    string          // Formatted input string (e.g., command for Bash)
	RawInput json.RawMessage // Raw input JSON for AskUserQuestion parsing
}

// AskUserQuestionInput represents the input to AskUserQuestion tool
type AskUserQuestionInput struct {
	Questions []struct {
		Header   string `json:"header"`
		Question string `json:"question"`
	} `json:"questions"`
}

// parseAssistantContent parses assistant message content to determine type and text
// Returns: entryType, text, and slice of tool use info
func parseAssistantContent(rawContent json.RawMessage) (entryType, text string, toolUses []ToolUseInfo) {
	if len(rawContent) == 0 {
		return "", "", nil
	}

	// Try to parse as string first
	var strContent string
	if err := json.Unmarshal(rawContent, &strContent); err == nil {
		if strContent != "" {
			return "ASSISTANT", strContent, nil
		}
		return "", "", nil
	}

	// Try to parse as array of content parts
	var parts []struct {
		Type  string          `json:"type"`
		Text  string          `json:"text,omitempty"`
		ID    string          `json:"id,omitempty"`
		Name  string          `json:"name,omitempty"`
		Input json.RawMessage `json:"input,omitempty"`
	}
	if err := json.Unmarshal(rawContent, &parts); err == nil {
		var textParts []string

		for _, part := range parts {
			switch part.Type {
			case "text":
				if part.Text != "" {
					textParts = append(textParts, part.Text)
				}
			case "tool_use":
				if part.Name != "" {
					toolInfo := ToolUseInfo{
						ID:       part.ID,
						Name:     part.Name,
						Input:    formatToolInput(part.Name, part.Input),
						RawInput: part.Input,
					}
					toolUses = append(toolUses, toolInfo)
				}
			}
		}

		// If there are tool uses, report them
		if len(toolUses) > 0 {
			var names []string
			for _, t := range toolUses {
				names = append(names, t.Name)
			}
			return "TOOL_USE", strings.Join(names, ", "), toolUses
		}

		// Otherwise return text content
		if len(textParts) > 0 {
			return "ASSISTANT", textParts[0], nil
		}
	}

	return "", "", nil
}

// formatToolInput extracts the most relevant input field for display
func formatToolInput(toolName string, input json.RawMessage) string {
	if len(input) == 0 {
		return ""
	}

	// Parse input as map
	var inputMap map[string]any
	if err := json.Unmarshal(input, &inputMap); err != nil {
		return ""
	}

	// Extract the most relevant field based on tool type
	switch toolName {
	case "Bash":
		if cmd, ok := inputMap["command"].(string); ok {
			return cmd
		}
	case "Read":
		if path, ok := inputMap["file_path"].(string); ok {
			return path
		}
	case "Write":
		if path, ok := inputMap["file_path"].(string); ok {
			return path
		}
	case "Edit":
		if path, ok := inputMap["file_path"].(string); ok {
			return path
		}
	case "Glob":
		if pattern, ok := inputMap["pattern"].(string); ok {
			return pattern
		}
	case "Grep":
		if pattern, ok := inputMap["pattern"].(string); ok {
			return pattern
		}
	case "Task":
		if prompt, ok := inputMap["prompt"].(string); ok {
			if len(prompt) > 100 {
				return prompt[:97] + "..."
			}
			return prompt
		}
	case "WebFetch":
		if url, ok := inputMap["url"].(string); ok {
			return url
		}
	case "WebSearch":
		if query, ok := inputMap["query"].(string); ok {
			return query
		}
	default:
		// For unknown tools, return JSON representation
		if b, err := json.Marshal(inputMap); err == nil {
			s := string(b)
			if len(s) > 200 {
				return s[:197] + "..."
			}
			return s
		}
	}

	return ""
}

// getCommitSubject gets the first line of a commit message
func getCommitSubject(sha string) (string, error) {
	out, err := git.RunGit("log", "-1", "--format=%s", sha)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// TimelineEntry represents an entry with its commit context for timeline rendering
type TimelineEntry struct {
	Entry       PromptEntry
	CommitSHA   string
	CommitSubj  string
	CommitIndex int // Order of commit in the PR
}

// RenderMarkdown generates markdown output for PR comment
func RenderMarkdown(summary *Summary, pagesURL string, version string) string {
	var sb strings.Builder

	if summary.CommitsWithNotes == 0 {
		sb.WriteString("No prompt-story notes found in this PR.\n")
		return sb.String()
	}

	// Reverse commits to show oldest first (chronological order)
	commits := make([]CommitSummary, len(summary.Commits))
	for i, c := range summary.Commits {
		commits[len(summary.Commits)-1-i] = c
	}

	// Sort sessions within each commit by start time (earliest first)
	for i := range commits {
		sort.Slice(commits[i].Sessions, func(a, b int) bool {
			return commits[i].Sessions[a].Start.Before(commits[i].Sessions[b].Start)
		})
	}

	// Build timeline entries from all commits
	var userTimeline []TimelineEntry
	var fullTimeline []TimelineEntry

	for i, commit := range commits {
		for _, sess := range commit.Sessions {
			for _, p := range sess.Prompts {
				te := TimelineEntry{
					Entry:       p,
					CommitSHA:   commit.ShortSHA,
					CommitSubj:  commit.Subject,
					CommitIndex: i,
				}
				fullTimeline = append(fullTimeline, te)
				if isUserAction(p.Type) && !sess.IsAgent {
					userTimeline = append(userTimeline, te)
				}
			}
		}
	}

	// Sort userTimeline chronologically across all sessions
	sort.Slice(userTimeline, func(i, j int) bool {
		return userTimeline[i].Entry.Time.Before(userTimeline[j].Entry.Time)
	})

	// Render Prompts section - markdown header, show first 10, collapse rest
	if len(userTimeline) == 0 {
		sb.WriteString("*No user prompts in this PR*\n\n")
	} else {
		sb.WriteString(fmt.Sprintf("# %d user prompts\n\n", len(userTimeline)))

		if len(userTimeline) <= 10 {
			// Show all prompts
			if allPromptsShort(userTimeline) {
				renderTimeline(&sb, userTimeline, formatSimple)
			} else {
				userPromptsContent, _ := renderUserTimelineWithTruncation(userTimeline, maxUserPromptsSize)
				sb.WriteString(userPromptsContent)
			}
		} else {
			// Show first 10, collapse the rest
			first10 := userTimeline[:10]
			remaining := userTimeline[10:]

			// Render first 10
			if allPromptsShort(first10) {
				renderTimeline(&sb, first10, formatSimple)
			} else {
				content, _ := renderUserTimelineWithTruncation(first10, maxUserPromptsSize)
				sb.WriteString(content)
			}

			// Render remaining in collapsible section
			sb.WriteString(fmt.Sprintf("\n<details><summary>Show %d more...</summary>\n\n", len(remaining)))
			if allPromptsShort(remaining) {
				renderTimeline(&sb, remaining, formatSimple)
			} else {
				content, _ := renderUserTimelineWithTruncation(remaining, maxUserPromptsSize)
				sb.WriteString(content)
			}
			sb.WriteString("</details>\n\n")
		}
	}

	// Render All Steps section - markdown header with all steps collapsed
	sb.WriteString(fmt.Sprintf("# All %d steps\n\n", len(fullTimeline)))
	sb.WriteString("<details><summary>Show all...</summary>\n\n")
	allStepsContent, _, _ := renderAllSteps(commits, maxAllStepsSize, pagesURL)
	sb.WriteString(allStepsContent)
	sb.WriteString("</details>\n\n")

	// Link to full transcripts (only if not already shown in truncation message)
	if pagesURL != "" {
		sb.WriteString(fmt.Sprintf("[View full transcripts](%s)\n\n", pagesURL))
	}

	// Summary table (at the bottom)
	sb.WriteString("| Commit | Subject | Tool(s) | User Prompts | Steps |\n")
	sb.WriteString("|--------|---------|---------|--------------|-------|\n")

	for _, commit := range commits {
		// Collect unique tools
		tools := make(map[string]bool)
		userPromptCount := 0
		agentPromptCount := 0
		totalSteps := 0

		for _, sess := range commit.Sessions {
			tools[note.FormatToolName(sess.Tool)] = true
			prompts := countUserPrompts(sess.Prompts)
			if sess.IsAgent {
				agentPromptCount += prompts
			} else {
				userPromptCount += prompts
			}
			totalSteps += len(sess.Prompts)
		}

		// Format tool names
		toolDisplay := formatToolDisplay(tools)

		// Truncate subject for table
		subject := commit.Subject
		if len(subject) > 40 {
			subject = subject[:37] + "..."
		}
		subject = html.EscapeString(subject)

		// Format user prompts (main session only)
		promptDisplay := fmt.Sprintf("%d", userPromptCount)

		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %d |\n",
			commit.ShortSHA, subject, toolDisplay, promptDisplay, totalSteps))
	}
	sb.WriteString("\n")

	sb.WriteString(fmt.Sprintf("---\n*Generated by [git-prompt-story](https://github.com/QuesmaOrg/git-prompt-story) %s*\n", version))

	return sb.String()
}

// getTypeEmoji returns an emoji for the entry type
func getTypeEmoji(entryType string) string {
	switch entryType {
	case "PROMPT":
		return "💬"
	case "TOOL_USE":
		return "🔧"
	case "ASSISTANT":
		return "🤖"
	case "TOOL_REJECT":
		return "❌"
	case "COMMAND":
		return "📋"
	case "DECISION":
		return "❓"
	default:
		return "📝" // Unknown type
	}
}

// Format modes for renderTimeline
const (
	formatRegular     = "regular"     // Truncated display (100 chars) - for "All Steps"
	formatCollapsible = "collapsible" // <details> tags for long prompts - for user prompts with some long
	formatSimple      = "simple"      // Full text, no details - for user prompts when all short
)

// renderTimeline renders a list of timeline entries with commit markers
func renderTimeline(sb *strings.Builder, entries []TimelineEntry, formatMode string) {
	lastCommitIndex := -1

	for _, te := range entries {
		// Insert commit marker when we cross to a new commit (including the first one)
		if te.CommitIndex != lastCommitIndex {
			subject := te.CommitSubj
			if len(subject) > 40 {
				subject = subject[:37] + "..."
			}
			subject = html.EscapeString(subject)
			sb.WriteString(fmt.Sprintf("\n#### %s: %s\n\n", te.CommitSHA, subject))
		}
		lastCommitIndex = te.CommitIndex

		// Format the entry based on mode
		switch formatMode {
		case formatCollapsible:
			if isUserAction(te.Entry.Type) {
				sb.WriteString(formatMarkdownEntryCollapsible(te.Entry))
			} else {
				sb.WriteString(formatMarkdownEntry(te.Entry))
			}
		case formatSimple:
			sb.WriteString(formatMarkdownEntrySimple(te.Entry))
		default: // formatRegular
			sb.WriteString(formatMarkdownEntry(te.Entry))
		}
	}
}

// renderAllSteps renders all steps grouped by session with truncation support
// Returns the rendered string and count of truncated sessions/steps
func renderAllSteps(commits []CommitSummary, maxSize int, pagesURL string) (string, int, int) {
	var sb strings.Builder
	truncatedSessions := 0
	truncatedSteps := 0

	for _, commit := range commits {
		// Format commit header
		subject := commit.Subject
		if len(subject) > 40 {
			subject = subject[:37] + "..."
		}
		subject = html.EscapeString(subject)
		commitHeader := fmt.Sprintf("\n#### %s: %s\n\n", commit.ShortSHA, subject)

		// Check if we can fit this commit header
		if sb.Len()+len(commitHeader) > maxSize {
			// Count remaining sessions and steps
			for _, sess := range commit.Sessions {
				truncatedSessions++
				truncatedSteps += len(sess.Prompts)
			}
			continue
		}
		sb.WriteString(commitHeader)

		for _, sess := range commit.Sessions {
			// Format session header
			toolName := note.FormatToolName(sess.Tool)
			startTime := sess.Start.Local().Format("15:04")
			endTime := sess.End.Local().Format("15:04")
			sessionHeader := fmt.Sprintf("**Session: %s** (%s-%s, %d steps)\n", toolName, startTime, endTime, len(sess.Prompts))

			// Estimate session size (header + entries)
			estimatedEntrySize := len(sess.Prompts) * 80 // rough estimate per entry
			if sb.Len()+len(sessionHeader)+estimatedEntrySize > maxSize {
				truncatedSessions++
				truncatedSteps += len(sess.Prompts)
				continue
			}

			sb.WriteString(sessionHeader)

			// Render entries with indent
			for _, p := range sess.Prompts {
				entryStr := formatMarkdownEntryIndented(p)
				if sb.Len()+len(entryStr) > maxSize {
					// Count remaining entries in this session
					truncatedSteps++
					continue
				}
				sb.WriteString(entryStr)
			}
			sb.WriteString("\n")
		}
	}

	// Add truncation notice if needed
	if truncatedSessions > 0 || truncatedSteps > 0 {
		notice := fmt.Sprintf("\n*...truncated %d older sessions with %d steps", truncatedSessions, truncatedSteps)
		if pagesURL != "" {
			notice += fmt.Sprintf(". [View full transcripts](%s)", pagesURL)
		}
		notice += "*\n"
		sb.WriteString(notice)
	}

	return sb.String(), truncatedSessions, truncatedSteps
}

// formatMarkdownEntryIndented formats a single entry with indentation for session grouping
func formatMarkdownEntryIndented(entry PromptEntry) string {
	timeStr := entry.Time.Local().Format("15:04")
	emoji := getTypeEmoji(entry.Type)
	text := strings.ReplaceAll(entry.Text, "\n", " ")
	if len(text) > 100 {
		text = text[:97] + "..."
	}
	text = html.EscapeString(text)

	switch entry.Type {
	case "TOOL_USE":
		if entry.ToolName != "" {
			input := entry.ToolInput
			if len(input) > 60 {
				input = input[:57] + "..."
			}
			input = strings.ReplaceAll(input, "\n", " ")
			input = html.EscapeString(input)
			return fmt.Sprintf("  - %s %s %s: %s\n", timeStr, emoji, entry.ToolName, input)
		}
		return fmt.Sprintf("  - %s %s %s\n", timeStr, emoji, text)
	case "DECISION":
		header := entry.DecisionHeader
		if header == "" {
			header = "Question"
		}
		answer := entry.DecisionAnswer
		if answer == "" {
			answer = "(no answer)"
		}
		answer = html.EscapeString(answer)
		return fmt.Sprintf("  - %s %s %s: %s → %s\n", timeStr, emoji, header, text, answer)
	default:
		if entry.Type == "PROMPT" || entry.Type == "ASSISTANT" || entry.Type == "COMMAND" || entry.Type == "TOOL_REJECT" {
			return fmt.Sprintf("  - %s %s %s\n", timeStr, emoji, text)
		}
		return fmt.Sprintf("  - %s %s %s: %s\n", timeStr, emoji, entry.Type, text)
	}
}

// renderUserTimelineWithTruncation renders user prompts with size limit
// Returns the rendered string and count of truncated prompts
func renderUserTimelineWithTruncation(entries []TimelineEntry, maxSize int) (string, int) {
	var sb strings.Builder
	truncatedCount := 0
	lastCommitIndex := -1

	for _, te := range entries {
		// Insert commit marker when we cross to a new commit
		if te.CommitIndex != lastCommitIndex {
			subject := te.CommitSubj
			if len(subject) > 40 {
				subject = subject[:37] + "..."
			}
			subject = html.EscapeString(subject)
			header := fmt.Sprintf("\n#### %s: %s\n\n", te.CommitSHA, subject)
			if sb.Len()+len(header) > maxSize {
				truncatedCount++
				continue
			}
			sb.WriteString(header)
		}
		lastCommitIndex = te.CommitIndex

		// Format the entry
		entryStr := formatMarkdownEntryCollapsible(te.Entry)
		if sb.Len()+len(entryStr) > maxSize {
			truncatedCount++
			continue
		}
		sb.WriteString(entryStr)
	}

	// Add truncation notice if needed
	if truncatedCount > 0 {
		notice := fmt.Sprintf("\n*...truncated %d older user prompts*\n", truncatedCount)
		sb.WriteString(notice)
	}

	return sb.String(), truncatedCount
}

// formatMarkdownEntry formats a single entry for markdown display
func formatMarkdownEntry(entry PromptEntry) string {
	timeStr := entry.Time.Local().Format("15:04")
	emoji := getTypeEmoji(entry.Type)
	text := strings.ReplaceAll(entry.Text, "\n", " ")
	if len(text) > 100 {
		text = text[:97] + "..."
	}
	// Escape HTML to prevent breaking markdown structure
	text = html.EscapeString(text)

	switch entry.Type {
	case "TOOL_USE":
		if entry.ToolName != "" {
			input := entry.ToolInput
			if len(input) > 60 {
				input = input[:57] + "..."
			}
			input = strings.ReplaceAll(input, "\n", " ")
			input = html.EscapeString(input)
			return fmt.Sprintf("- %s %s %s: %s\n", timeStr, emoji, entry.ToolName, input)
		}
		return fmt.Sprintf("- %s %s %s\n", timeStr, emoji, text)
	case "DECISION":
		header := entry.DecisionHeader
		if header == "" {
			header = "Question"
		}
		answer := entry.DecisionAnswer
		if answer == "" {
			answer = "(no answer)"
		}
		answer = html.EscapeString(answer)
		return fmt.Sprintf("- %s %s %s: %s → %s\n", timeStr, emoji, header, text, answer)
	default:
		// For known types (PROMPT, ASSISTANT), just show emoji + text
		// For unknown types, show emoji + type + text
		if entry.Type == "PROMPT" || entry.Type == "ASSISTANT" || entry.Type == "COMMAND" || entry.Type == "TOOL_REJECT" {
			return fmt.Sprintf("- %s %s %s\n", timeStr, emoji, text)
		}
		return fmt.Sprintf("- %s %s %s: %s\n", timeStr, emoji, entry.Type, text)
	}
}

// formatMarkdownEntryCollapsible formats an entry, making long ones collapsible
func formatMarkdownEntryCollapsible(entry PromptEntry) string {
	timeStr := entry.Time.Local().Format("15:04")
	emoji := getTypeEmoji(entry.Type)
	text := strings.ReplaceAll(entry.Text, "\n", " ")

	// DECISION entries: always show in full with answer
	if entry.Type == "DECISION" {
		header := entry.DecisionHeader
		if header == "" {
			header = "Question"
		}
		answer := entry.DecisionAnswer
		if answer == "" {
			answer = "(no answer)"
		}
		// Escape HTML
		text = html.EscapeString(text)
		answer = html.EscapeString(answer)
		return fmt.Sprintf("- <details open><summary>%s %s %s: %s → %s</summary></details>\n\n",
			timeStr, emoji, header, text, answer)
	}

	// Short prompts (≤250 chars): <details open> (expanded by default)
	if len(text) <= 250 {
		// Escape HTML to prevent breaking markdown structure
		text = html.EscapeString(text)
		return fmt.Sprintf("- <details open><summary>%s %s %s</summary></details>\n\n",
			timeStr, emoji, text)
	}

	// Long prompts: <details> (collapsed) with truncated summary
	summary := text[:247] + "..."
	continuation := strings.ReplaceAll(entry.Text[247:], "\n", " ") // Remove newlines to avoid nested details issues

	// Escape HTML in both summary and continuation
	summary = html.EscapeString(summary)
	continuation = html.EscapeString(continuation)

	return fmt.Sprintf("- <details><summary>%s %s %s</summary>...%s</details>\n\n",
		timeStr, emoji, summary, continuation)
}

// RenderJSON generates JSON output
func RenderJSON(summary *Summary) ([]byte, error) {
	return json.MarshalIndent(summary, "", "  ")
}

// isUserAction returns true if the entry type represents a user action
func isUserAction(entryType string) bool {
	switch entryType {
	case "PROMPT", "COMMAND", "TOOL_REJECT", "DECISION":
		return true
	default:
		return false
	}
}

// allPromptsShort returns true if all entries have short text (≤250 chars)
func allPromptsShort(entries []TimelineEntry) bool {
	for _, te := range entries {
		text := strings.ReplaceAll(te.Entry.Text, "\n", " ")
		if len(text) > 250 {
			return false
		}
	}
	return true
}

// formatMarkdownEntrySimple formats an entry as a simple bullet without details tags
func formatMarkdownEntrySimple(entry PromptEntry) string {
	timeStr := entry.Time.Local().Format("15:04")
	emoji := getTypeEmoji(entry.Type)
	text := strings.ReplaceAll(entry.Text, "\n", " ")
	text = html.EscapeString(text)

	// DECISION entries: show with answer
	if entry.Type == "DECISION" {
		header := entry.DecisionHeader
		if header == "" {
			header = "Question"
		}
		answer := entry.DecisionAnswer
		if answer == "" {
			answer = "(no answer)"
		}
		answer = html.EscapeString(answer)
		return fmt.Sprintf("- %s %s %s: %s → %s\n", timeStr, emoji, header, text, answer)
	}

	return fmt.Sprintf("- %s %s %s\n", timeStr, emoji, text)
}

// countUserPrompts counts user action entries in a slice
func countUserPrompts(prompts []PromptEntry) int {
	count := 0
	for _, p := range prompts {
		if isUserAction(p.Type) {
			count++
		}
	}
	return count
}

// formatToolDisplay formats tool names for display in the summary table
func formatToolDisplay(tools map[string]bool) string {
	if len(tools) == 1 {
		for t := range tools {
			return t
		}
	}
	return fmt.Sprintf("tools (%d)", len(tools))
}
