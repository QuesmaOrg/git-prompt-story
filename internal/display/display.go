// Package display provides shared display utilities for terminal and HTML output.
package display

// TypeEmoji maps entry types to their display emojis.
var TypeEmoji = map[string]string{
	"PROMPT":      "💬",
	"COMMAND":     "📋",
	"TOOL_REJECT": "❌",
	"DECISION":    "❓",
	"TOOL_USE":    "🔧",
	"ASSISTANT":   "🤖",
	"TOOL_RESULT": "📤",
}

// GetTypeEmoji returns an emoji for the given entry type.
// Returns "•" for unknown types.
func GetTypeEmoji(entryType string) string {
	if emoji, ok := TypeEmoji[entryType]; ok {
		return emoji
	}
	return "•"
}

// TruncateText truncates text to maxLen characters, replacing newlines with spaces.
// If truncated, adds "..." suffix.
func TruncateText(s string, maxLen int) string {
	// Replace newlines with spaces
	text := s
	for i := 0; i < len(text); i++ {
		if text[i] == '\n' || text[i] == '\r' {
			text = text[:i] + " " + text[i+1:]
		}
	}

	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen-3] + "..."
}
