package display

import "testing"

func TestGetTypeEmoji(t *testing.T) {
	tests := []struct {
		entryType string
		expected  string
	}{
		{"PROMPT", "💬"},
		{"COMMAND", "📋"},
		{"TOOL_REJECT", "❌"},
		{"DECISION", "❓"},
		{"TOOL_USE", "🔧"},
		{"ASSISTANT", "🤖"},
		{"TOOL_RESULT", "📤"},
		{"UNKNOWN", "•"},
		{"", "•"},
	}

	for _, tt := range tests {
		t.Run(tt.entryType, func(t *testing.T) {
			result := GetTypeEmoji(tt.entryType)
			if result != tt.expected {
				t.Errorf("GetTypeEmoji(%q) = %q, want %q", tt.entryType, result, tt.expected)
			}
		})
	}
}

func TestTruncateText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "short text unchanged",
			input:    "hello",
			maxLen:   10,
			expected: "hello",
		},
		{
			name:     "exact length unchanged",
			input:    "hello",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "long text truncated",
			input:    "hello world",
			maxLen:   8,
			expected: "hello...",
		},
		{
			name:     "newline replaced with space",
			input:    "hello\nworld",
			maxLen:   20,
			expected: "hello world",
		},
		{
			name:     "carriage return replaced",
			input:    "hello\rworld",
			maxLen:   20,
			expected: "hello world",
		},
		{
			name:     "empty string",
			input:    "",
			maxLen:   10,
			expected: "",
		},
		{
			name:     "truncate with newlines",
			input:    "line1\nline2\nline3",
			maxLen:   10,
			expected: "line1 l...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncateText(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("TruncateText(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}
