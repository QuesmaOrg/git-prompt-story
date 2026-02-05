package claude

import (
	"time"

	"github.com/QuesmaOrg/git-prompt-story/internal/parser"
	"github.com/QuesmaOrg/git-prompt-story/internal/session"
)

func init() {
	parser.Register(&Parser{})
}

// Parser implements the parser.Parser interface for Claude Code
type Parser struct{}

// Name returns the tool identifier
func (p *Parser) Name() string {
	return "claude-code"
}

// Parse converts JSONL content to unified entries
func (p *Parser) Parse(content []byte, startWork, endWork time.Time) ([]parser.UnifiedEntry, error) {
	entries, err := session.ParseMessages(content)
	if err != nil {
		return nil, err
	}

	var result []parser.UnifiedEntry
	for _, e := range entries {
		// Filter by time
		if !e.Timestamp.IsZero() {
			if e.Timestamp.Before(startWork) || e.Timestamp.After(endWork) {
				continue
			}
		}

		entry := convertEntry(e)
		result = append(result, entry)
	}

	return result, nil
}

// CountUserActions counts user actions in the content
func (p *Parser) CountUserActions(content []byte, startWork, endWork time.Time) int {
	entries, err := session.ParseMessages(content)
	if err != nil {
		return 0
	}

	count := 0
	for _, e := range entries {
		if !e.Timestamp.IsZero() {
			if e.Timestamp.Before(startWork) || e.Timestamp.After(endWork) {
				continue
			}
		}

		// Use the same logic as session.isUserActionEntry
		if isUserAction(e) {
			count++
		}
	}

	return count
}

// convertEntry converts a Claude Code message entry to a unified entry
func convertEntry(e session.MessageEntry) parser.UnifiedEntry {
	entry := parser.UnifiedEntry{
		Time:   e.Timestamp,
		IsMeta: e.IsMeta,
	}

	switch e.Type {
	case "user":
		entry.Type = parser.EntryUser
		entry.Role = "user"
		if e.Message != nil {
			entry.Text = e.Message.GetTextContent()
		}
	case "assistant":
		entry.Type = parser.EntryAssistant
		entry.Role = "assistant"
		if e.Message != nil {
			entry.Text = e.Message.GetTextContent()
		}
	case "tool_reject":
		entry.Type = parser.EntryToolResult
		entry.Role = "user"
		entry.Rejected = true
	case "queue-operation":
		if e.Operation == "enqueue" && e.Content != "" {
			entry.Type = parser.EntryUser
			entry.Role = "user"
			entry.Text = e.Content
		}
	default:
		entry.Type = parser.EntrySystem
		entry.Role = "system"
	}

	return entry
}

// isUserAction checks if an entry represents a user action
func isUserAction(e session.MessageEntry) bool {
	if e.IsMeta {
		return false
	}

	switch e.Type {
	case "tool_reject":
		return true
	case "queue-operation":
		if e.Operation == "enqueue" && e.Content != "" {
			return true
		}
		return false
	case "user":
		if e.Message == nil {
			return false
		}
		text := e.Message.GetTextContent()
		return text != ""
	default:
		return false
	}
}
