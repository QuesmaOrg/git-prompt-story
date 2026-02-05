package cursor

import (
	"encoding/json"
	"time"

	"github.com/QuesmaOrg/git-prompt-story/internal/parser"
)

func init() {
	parser.Register(&Parser{})
}

// Parser implements the parser.Parser interface for Cursor
type Parser struct{}

// Name returns the tool identifier
func (p *Parser) Name() string {
	return "cursor"
}

// ComposerData represents the structure of Cursor's composerData entries
type ComposerData struct {
	ComposerID   string   `json:"composerId"`
	CreatedAt    int64    `json:"createdAt"`
	Conversation []Bubble `json:"conversation"`
	Bubbles      []Bubble `json:"_bubbles"` // Bubbles from separate keys (new format)
}

// Bubble represents a single message in a Cursor conversation
type Bubble struct {
	Type       int        `json:"type"` // 1=user, 2=AI
	BubbleID   string     `json:"bubbleId"`
	Text       string     `json:"text"`
	TimingInfo TimingInfo `json:"timingInfo,omitempty"`
}

// TimingInfo contains timing information for a bubble
type TimingInfo struct {
	ClientStartTime int64 `json:"clientStartTime"` // epoch ms
}

// Parse converts Cursor JSON content to unified entries
func (p *Parser) Parse(content []byte, startWork, endWork time.Time) ([]parser.UnifiedEntry, error) {
	var data ComposerData
	if err := json.Unmarshal(content, &data); err != nil {
		return nil, err
	}

	// Combine bubbles from both old and new format
	allBubbles := append(data.Conversation, data.Bubbles...)

	var result []parser.UnifiedEntry
	for _, bubble := range allBubbles {
		// Get timestamp
		var ts time.Time
		if bubble.TimingInfo.ClientStartTime > 0 {
			ts = time.UnixMilli(bubble.TimingInfo.ClientStartTime)
		} else if data.CreatedAt > 0 {
			ts = time.UnixMilli(data.CreatedAt)
		}

		// Filter by time
		if !ts.IsZero() {
			if ts.Before(startWork) || ts.After(endWork) {
				continue
			}
		}

		entry := parser.UnifiedEntry{
			Time: ts,
		}

		switch bubble.Type {
		case 1: // User
			entry.Type = parser.EntryUser
			entry.Role = "user"
			entry.Text = bubble.Text
		case 2: // AI
			entry.Type = parser.EntryAssistant
			entry.Role = "assistant"
			entry.Text = bubble.Text
		default:
			entry.Type = parser.EntrySystem
			entry.Role = "system"
			entry.Text = bubble.Text
		}

		result = append(result, entry)
	}

	return result, nil
}

// CountUserActions counts user actions in the content
func (p *Parser) CountUserActions(content []byte, startWork, endWork time.Time) int {
	var data ComposerData
	if err := json.Unmarshal(content, &data); err != nil {
		return 0
	}

	// Combine bubbles from both old and new format
	allBubbles := append(data.Conversation, data.Bubbles...)

	count := 0
	for _, bubble := range allBubbles {
		if bubble.Type != 1 { // Only count user messages
			continue
		}

		// Get timestamp
		var ts time.Time
		if bubble.TimingInfo.ClientStartTime > 0 {
			ts = time.UnixMilli(bubble.TimingInfo.ClientStartTime)
		}

		// Filter by time
		if !ts.IsZero() {
			if ts.Before(startWork) || ts.After(endWork) {
				continue
			}
		}

		if bubble.Text != "" {
			count++
		}
	}

	return count
}
