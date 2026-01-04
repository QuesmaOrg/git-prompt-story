package cloud

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/QuesmaOrg/git-prompt-story/internal/session"
)

// EventsToJSONL converts cloud events to JSONL format compatible with local sessions
func EventsToJSONL(events []Event, sess *Session) ([]byte, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)

	for _, evt := range events {
		// Only convert user and assistant messages
		if evt.Type != "user" && evt.Type != "assistant" {
			continue
		}

		if evt.Message == nil {
			continue
		}

		entry := session.MessageEntry{
			Type:      evt.Type,
			SessionID: sess.ID,
			Timestamp: sess.CreatedAt, // Events don't have individual timestamps
			Message: &session.Message{
				Role:       evt.Message.Role,
				RawContent: evt.Message.Content,
			},
		}

		// Add git branch info if available
		for _, outcome := range sess.SessionContext.Outcomes {
			if outcome.Type == "git_repository" && len(outcome.GitInfo.Branches) > 0 {
				entry.GitBranch = outcome.GitInfo.Branches[0]
				break
			}
		}

		if err := encoder.Encode(entry); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// EventsToMessageEntries converts cloud events to MessageEntry slice
func EventsToMessageEntries(events []Event, sess *Session) []session.MessageEntry {
	var entries []session.MessageEntry

	for _, evt := range events {
		// Only convert user and assistant messages
		if evt.Type != "user" && evt.Type != "assistant" {
			continue
		}

		if evt.Message == nil {
			continue
		}

		entry := session.MessageEntry{
			Type:      evt.Type,
			SessionID: sess.ID,
			Timestamp: sess.CreatedAt,
			Message: &session.Message{
				Role:       evt.Message.Role,
				RawContent: evt.Message.Content,
			},
		}

		// Add git branch info if available
		for _, outcome := range sess.SessionContext.Outcomes {
			if outcome.Type == "git_repository" && len(outcome.GitInfo.Branches) > 0 {
				entry.GitBranch = outcome.GitInfo.Branches[0]
				break
			}
		}

		entries = append(entries, entry)
	}

	return entries
}

// ToClaudeSession converts a cloud Session to the local ClaudeSession format
func ToClaudeSession(sess *Session) session.ClaudeSession {
	return session.ClaudeSession{
		ID:       sess.ID,
		Path:     "", // Cloud sessions don't have local paths
		Created:  sess.CreatedAt,
		Modified: sess.UpdatedAt,
	}
}

// ExtractUserPrompts returns just the user messages from events
func ExtractUserPrompts(events []Event) []string {
	var prompts []string
	for _, evt := range events {
		if evt.Type == "user" && evt.Message != nil {
			// Try to get text content
			var content string
			if err := json.Unmarshal(evt.Message.Content, &content); err == nil {
				prompts = append(prompts, content)
			}
		}
	}
	return prompts
}

// GetSessionTimeRange returns the time range covered by events
func GetSessionTimeRange(sess *Session, events []Event) (start, end time.Time) {
	return sess.CreatedAt, sess.UpdatedAt
}
