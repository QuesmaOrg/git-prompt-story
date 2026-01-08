package claudecode

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/QuesmaOrg/git-prompt-story/internal/session"
)

// Parser implements session.SessionParser for Claude Code.
type Parser struct{}

// NewParser creates a new Claude Code session parser.
func NewParser() *Parser {
	return &Parser{}
}

// PromptTool returns the prompt tool identifier.
func (p *Parser) PromptTool() string {
	return PromptToolName
}

// ParseMetadata extracts session metadata without full parsing.
func (p *Parser) ParseMetadata(sessionPath string) (created, modified time.Time, branch string, err error) {
	return session.ParseSessionMetadata(sessionPath)
}

// ParseSession converts Claude Code session content to common PromptEntry format.
func (p *Parser) ParseSession(content []byte, startWork, endWork time.Time, full bool) ([]session.PromptEntry, error) {
	entries, err := session.ParseMessages(content)
	if err != nil {
		return nil, err
	}

	var prompts []session.PromptEntry

	// Map to track tool use entries by ID for linking with results
	toolUseEntries := make(map[string]*session.PromptEntry)

	// Map to track AskUserQuestion entries by tool ID
	askUserQuestionEntries := make(map[string][]int)

	// Map to track options for each question
	questionOptions := make(map[string][]AskUserQuestionOption)

	for _, entry := range entries {
		ts := entry.Timestamp
		if ts.IsZero() && entry.Snapshot != nil {
			ts = entry.Snapshot.Timestamp
		}

		if ts.IsZero() {
			continue
		}

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
						pe := session.PromptEntry{
							Time:         ts,
							Type:         "COMMAND",
							Text:         "/" + cmdName,
							InWorkPeriod: inWorkPeriod,
						}
						if inWorkPeriod {
							prompts = append(prompts, pe)
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
						if tr.IsError && strings.Contains(tr.Content, "tool use was rejected") {
							text := "User rejected tool call"
							if idx := strings.Index(tr.Content, "user said:\n"); idx != -1 {
								text = strings.TrimSpace(tr.Content[idx+len("user said:\n"):])
							}
							pe := session.PromptEntry{
								Time:         ts,
								Type:         "TOOL_REJECT",
								Text:         text,
								InWorkPeriod: inWorkPeriod,
							}
							if inWorkPeriod {
								prompts = append(prompts, pe)
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
							if entry.ToolUseResult != nil && entry.ToolUseResult.Answers != nil {
								for _, idx := range indices {
									if idx < len(prompts) {
										question := prompts[idx].Text
										if answer, found := entry.ToolUseResult.Answers[question]; found {
											prompts[idx].DecisionAnswer = answer
											if opts, hasOpts := questionOptions[question]; hasOpts {
												for _, opt := range opts {
													if opt.Label == answer {
														prompts[idx].DecisionAnswerDescription = opt.Description
														break
													}
												}
											}
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
					pe := session.PromptEntry{
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
						prompts = append(prompts, pe)
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
			pe := session.PromptEntry{
				Time:         ts,
				Type:         "TOOL_REJECT",
				Text:         text,
				InWorkPeriod: inWorkPeriod,
			}
			if inWorkPeriod {
				prompts = append(prompts, pe)
			}

		case "assistant":
			if entry.Message != nil {
				entryType, text, toolUses := parseAssistantContent(entry.Message.RawContent)

				if len(toolUses) > 0 {
					for _, tool := range toolUses {
						// Special handling for AskUserQuestion
						if tool.Name == "AskUserQuestion" {
							var askInput AskUserQuestionInput
							if err := json.Unmarshal(tool.RawInput, &askInput); err == nil && len(askInput.Questions) > 0 {
								var indices []int
								for _, q := range askInput.Questions {
									pe := session.PromptEntry{
										Time:           ts,
										Type:           "DECISION",
										Text:           q.Question,
										ToolID:         tool.ID,
										DecisionHeader: q.Header,
										InWorkPeriod:   inWorkPeriod,
									}
									if inWorkPeriod {
										prompts = append(prompts, pe)
										indices = append(indices, len(prompts)-1)
									}
									if len(q.Options) > 0 {
										questionOptions[q.Question] = q.Options
									}
								}
								if len(indices) > 0 {
									askUserQuestionEntries[tool.ID] = indices
								}
								continue
							}
						}

						pe := session.PromptEntry{
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
							prompts = append(prompts, pe)
							toolUseEntries[tool.ID] = &prompts[len(prompts)-1]
						}
					}
				} else if entryType == "ASSISTANT" && text != "" {
					pe := session.PromptEntry{
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
						prompts = append(prompts, pe)
					}
				}
			}

		case "queue-operation":
			if entry.Operation == "enqueue" && entry.Content != "" {
				if strings.HasPrefix(entry.Content, "<bash-notification>") {
					continue
				}
				if strings.HasPrefix(entry.Content, "/") {
					continue
				}
				pe := session.PromptEntry{
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
					prompts = append(prompts, pe)
				}
			}
		}
	}

	return prompts, nil
}

// ToolResultInfo holds extracted tool result information.
type ToolResultInfo struct {
	ToolUseID string
	Content   string
	IsError   bool
}

// parseToolResults extracts tool_result entries from user message content.
func parseToolResults(rawContent json.RawMessage) []ToolResultInfo {
	if len(rawContent) == 0 {
		return nil
	}

	var parts []struct {
		Type      string `json:"type"`
		ToolUseID string `json:"tool_use_id"`
		Content   any    `json:"content"`
		IsError   bool   `json:"is_error,omitempty"`
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

// extractToolResultContent extracts text content from tool result.
func extractToolResultContent(content any) string {
	if content == nil {
		return ""
	}

	var result string

	if s, ok := content.(string); ok {
		result = s
	} else if arr, ok := content.([]any); ok {
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

// ToolUseInfo holds extracted information about a tool use.
type ToolUseInfo struct {
	ID       string
	Name     string
	Input    string
	RawInput json.RawMessage
}

// AskUserQuestionOption represents a single option in AskUserQuestion.
type AskUserQuestionOption struct {
	Label       string `json:"label"`
	Description string `json:"description"`
}

// AskUserQuestionInput represents the input to AskUserQuestion tool.
type AskUserQuestionInput struct {
	Questions []struct {
		Header   string                  `json:"header"`
		Question string                  `json:"question"`
		Options  []AskUserQuestionOption `json:"options"`
	} `json:"questions"`
}

// parseAssistantContent parses assistant message content.
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

		if len(toolUses) > 0 {
			var names []string
			for _, t := range toolUses {
				names = append(names, t.Name)
			}
			return "TOOL_USE", strings.Join(names, ", "), toolUses
		}

		if len(textParts) > 0 {
			return "ASSISTANT", textParts[0], nil
		}
	}

	return "", "", nil
}

// formatToolInput extracts the most relevant input field for display.
func formatToolInput(toolName string, input json.RawMessage) string {
	if len(input) == 0 {
		return ""
	}

	var inputMap map[string]any
	if err := json.Unmarshal(input, &inputMap); err != nil {
		return ""
	}

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
