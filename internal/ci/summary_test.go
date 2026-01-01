package ci

import (
	"strings"
	"testing"
	"time"
)

func TestIsUserAction(t *testing.T) {
	tests := []struct {
		entryType string
		expected  bool
	}{
		{"PROMPT", true},
		{"COMMAND", true},
		{"TOOL_REJECT", true},
		{"ASSISTANT", false},
		{"TOOL_USE", false},
		{"TOOL_RESULT", false},
		{"OTHER", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.entryType, func(t *testing.T) {
			result := isUserAction(tt.entryType)
			if result != tt.expected {
				t.Errorf("isUserAction(%q) = %v, want %v", tt.entryType, result, tt.expected)
			}
		})
	}
}

func TestCountUserPrompts(t *testing.T) {
	tests := []struct {
		name     string
		prompts  []PromptEntry
		expected int
	}{
		{
			name:     "empty",
			prompts:  []PromptEntry{},
			expected: 0,
		},
		{
			name: "only user actions",
			prompts: []PromptEntry{
				{Type: "PROMPT"},
				{Type: "COMMAND"},
				{Type: "TOOL_REJECT"},
			},
			expected: 3,
		},
		{
			name: "mixed entries",
			prompts: []PromptEntry{
				{Type: "PROMPT"},
				{Type: "ASSISTANT"},
				{Type: "TOOL_USE"},
				{Type: "COMMAND"},
			},
			expected: 2,
		},
		{
			name: "no user actions",
			prompts: []PromptEntry{
				{Type: "ASSISTANT"},
				{Type: "TOOL_USE"},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := countUserPrompts(tt.prompts)
			if result != tt.expected {
				t.Errorf("countUserPrompts() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFormatToolDisplay(t *testing.T) {
	tests := []struct {
		name     string
		tools    map[string]bool
		expected string
	}{
		{
			name:     "single tool",
			tools:    map[string]bool{"Claude Code": true},
			expected: "Claude Code",
		},
		{
			name:     "two tools",
			tools:    map[string]bool{"Claude Code": true, "Cursor": true},
			expected: "tools (2)",
		},
		{
			name:     "three tools",
			tools:    map[string]bool{"Claude Code": true, "Cursor": true, "Codex": true},
			expected: "tools (3)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatToolDisplay(tt.tools)
			if result != tt.expected {
				t.Errorf("formatToolDisplay() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestRenderMarkdown_NoCommits(t *testing.T) {
	summary := &Summary{
		CommitsWithNotes: 0,
	}

	result := RenderMarkdown(summary, "")

	expected := "No prompt-story notes found in this PR.\n"
	if result != expected {
		t.Errorf("RenderMarkdown() = %q, want %q", result, expected)
	}
}

func TestRenderMarkdown_Structure(t *testing.T) {
	now := time.Now()
	summary := &Summary{
		CommitsWithNotes: 1,
		TotalUserPrompts: 2,
		TotalSteps:       5,
		Commits: []CommitSummary{
			{
				ShortSHA: "abc1234",
				Subject:  "Test commit",
				Sessions: []SessionSummary{
					{
						Tool: "claude-code",
						Prompts: []PromptEntry{
							{Type: "PROMPT", Text: "User prompt 1", Time: now},
							{Type: "ASSISTANT", Text: "Response", Time: now},
							{Type: "TOOL_USE", Text: "Bash", ToolName: "Bash", ToolInput: "ls -la", Time: now},
							{Type: "PROMPT", Text: "User prompt 2", Time: now},
							{Type: "ASSISTANT", Text: "Final response", Time: now},
						},
					},
				},
			},
		},
	}

	result := RenderMarkdown(summary, "")

	// Verify NO old header
	if strings.Contains(result, "## Prompt Story") {
		t.Error("Should not contain old header '## Prompt Story'")
	}

	// Verify new table header
	if !strings.Contains(result, "| Commit | Subject | Tool(s) | User Prompts | Steps |") {
		t.Error("Missing new table header")
	}

	// Verify Prompts section exists
	if !strings.Contains(result, "### Prompts") {
		t.Error("Missing '### Prompts' section")
	}

	// Verify Full Transcript section exists
	if !strings.Contains(result, "### Full Transcript") {
		t.Error("Missing '### Full Transcript' section")
	}

	// Verify nested details for transcript with count
	if !strings.Contains(result, "Show all 5 steps") {
		t.Error("Missing collapsible 'Show all N steps' for Full Transcript")
	}

	// Verify prompts section is collapsible with count
	if !strings.Contains(result, "Show 2 user prompts") {
		t.Error("Missing collapsible 'Show N user prompts' for Prompts section")
	}

	// Verify commit SHA in table
	if !strings.Contains(result, "| abc1234 |") {
		t.Error("Missing commit SHA in table")
	}

	// Verify Claude Code tool name
	if !strings.Contains(result, "Claude Code") {
		t.Error("Missing 'Claude Code' tool name")
	}

	// Verify TOOL_USE formatting
	if !strings.Contains(result, "TOOL_USE (Bash)") {
		t.Error("Missing 'TOOL_USE (Bash)' formatting")
	}
}

func TestRenderMarkdown_MultipleTools(t *testing.T) {
	now := time.Now()
	summary := &Summary{
		CommitsWithNotes: 1,
		Commits: []CommitSummary{
			{
				ShortSHA: "def5678",
				Subject:  "Multi-tool commit",
				Sessions: []SessionSummary{
					{
						Tool: "claude-code",
						Prompts: []PromptEntry{
							{Type: "PROMPT", Text: "Prompt 1", Time: now},
						},
					},
					{
						Tool: "cursor",
						Prompts: []PromptEntry{
							{Type: "PROMPT", Text: "Prompt 2", Time: now},
						},
					},
				},
			},
		},
	}

	result := RenderMarkdown(summary, "")

	// Should show "tools (2)" for multiple tools
	if !strings.Contains(result, "tools (2)") {
		t.Error("Should show 'tools (2)' for multiple tools")
	}
}

func TestRenderMarkdown_PagesURL(t *testing.T) {
	now := time.Now()
	summary := &Summary{
		CommitsWithNotes: 1,
		Commits: []CommitSummary{
			{
				ShortSHA: "abc1234",
				Subject:  "Test",
				Sessions: []SessionSummary{
					{
						Tool: "claude-code",
						Prompts: []PromptEntry{
							{Type: "PROMPT", Text: "Test", Time: now},
						},
					},
				},
			},
		},
	}

	result := RenderMarkdown(summary, "https://example.github.io/repo/pr-42/")

	if !strings.Contains(result, "[View full transcripts](https://example.github.io/repo/pr-42/)") {
		t.Error("Should contain pages URL link")
	}
}

func TestRenderMarkdown_NoUserPrompts(t *testing.T) {
	now := time.Now()
	summary := &Summary{
		CommitsWithNotes: 1,
		Commits: []CommitSummary{
			{
				ShortSHA: "abc1234",
				Subject:  "Test",
				Sessions: []SessionSummary{
					{
						Tool: "claude-code",
						Prompts: []PromptEntry{
							{Type: "ASSISTANT", Text: "Only assistant", Time: now},
							{Type: "TOOL_USE", Text: "Bash", Time: now},
						},
					},
				},
			},
		},
	}

	result := RenderMarkdown(summary, "")

	// Should show message when no user prompts
	if !strings.Contains(result, "*No user prompts in this PR*") {
		t.Error("Should show 'No user prompts' message when there are no user actions")
	}
}

func TestFormatMarkdownEntry(t *testing.T) {
	now := time.Date(2025, 1, 15, 9, 30, 0, 0, time.Local)

	tests := []struct {
		name     string
		entry    PromptEntry
		contains []string
	}{
		{
			name:     "prompt entry",
			entry:    PromptEntry{Type: "PROMPT", Text: "Hello world", Time: now},
			contains: []string{"09:30", "PROMPT", "Hello world"},
		},
		{
			name:     "tool use with name",
			entry:    PromptEntry{Type: "TOOL_USE", ToolName: "Bash", ToolInput: "ls -la", Time: now},
			contains: []string{"09:30", "TOOL_USE (Bash)", "ls -la"},
		},
		{
			name:     "long text truncation",
			entry:    PromptEntry{Type: "PROMPT", Text: strings.Repeat("a", 150), Time: now},
			contains: []string{"09:30", "PROMPT", "..."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatMarkdownEntry(tt.entry)
			for _, substr := range tt.contains {
				if !strings.Contains(result, substr) {
					t.Errorf("formatMarkdownEntry() = %q, should contain %q", result, substr)
				}
			}
		})
	}
}
