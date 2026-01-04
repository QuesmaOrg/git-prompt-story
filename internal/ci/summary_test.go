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

	// Verify Prompts section exists (new format: merged with details)
	if !strings.Contains(result, "user prompts</strong>") {
		t.Error("Missing user prompts section")
	}

	// Verify All Steps section exists (renamed from Full Transcript)
	if !strings.Contains(result, "All 5 steps</strong>") {
		t.Error("Missing 'All N steps' section")
	}

	// Verify prompts section is collapsible with count
	if !strings.Contains(result, "2 user prompts</strong>") {
		t.Error("Missing collapsible user prompts section")
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

func TestRenderMarkdown_MultipleCommitsDifferentEntries(t *testing.T) {
	// This test verifies that when multiple commits are rendered:
	// 1. Each commit shows its own counts (not duplicated from first commit)
	// 2. Timeline entries are not duplicated
	// 3. Commit markers appear between commits

	// Commit 1: 4 entries (2 prompts) at 09:15-09:30
	time1 := time.Date(2025, 1, 15, 9, 15, 0, 0, time.Local)
	time2 := time.Date(2025, 1, 15, 9, 20, 0, 0, time.Local)
	time3 := time.Date(2025, 1, 15, 9, 25, 0, 0, time.Local)
	time4 := time.Date(2025, 1, 15, 9, 30, 0, 0, time.Local)

	// Commit 2: 3 entries (2 prompts) at 10:15-10:25
	time5 := time.Date(2025, 1, 15, 10, 15, 0, 0, time.Local)
	time6 := time.Date(2025, 1, 15, 10, 20, 0, 0, time.Local)
	time7 := time.Date(2025, 1, 15, 10, 25, 0, 0, time.Local)

	summary := &Summary{
		CommitsWithNotes: 2,
		TotalUserPrompts: 4, // 2 + 2
		TotalSteps:       7, // 4 + 3
		Commits: []CommitSummary{
			{
				ShortSHA: "abc1234",
				Subject:  "First commit",
				Sessions: []SessionSummary{
					{
						Tool: "claude-code",
						Prompts: []PromptEntry{
							{Type: "PROMPT", Text: "First prompt", Time: time1},
							{Type: "ASSISTANT", Text: "Response 1", Time: time2},
							{Type: "TOOL_USE", Text: "Bash", ToolName: "Bash", ToolInput: "ls", Time: time3},
							{Type: "PROMPT", Text: "Second prompt", Time: time4},
						},
					},
				},
			},
			{
				ShortSHA: "def5678",
				Subject:  "Second commit",
				Sessions: []SessionSummary{
					{
						Tool: "claude-code",
						Prompts: []PromptEntry{
							{Type: "PROMPT", Text: "Third prompt", Time: time5},
							{Type: "ASSISTANT", Text: "Response 2", Time: time6},
							{Type: "PROMPT", Text: "Fourth prompt", Time: time7},
						},
					},
				},
			},
		},
	}

	result := RenderMarkdown(summary, "")

	// Verify table has two rows with different counts
	// Commit 1: 2 user prompts, 4 steps
	if !strings.Contains(result, "| abc1234 | First commit | Claude Code | 2 | 4 |") {
		t.Error("First commit row should show 2 user prompts and 4 steps")
	}

	// Commit 2: 2 user prompts, 3 steps
	if !strings.Contains(result, "| def5678 | Second commit | Claude Code | 2 | 3 |") {
		t.Error("Second commit row should show 2 user prompts and 3 steps")
	}

	// Verify commit marker exists between commits
	if !strings.Contains(result, "--- Commit def5678: Second commit ---") {
		t.Error("Should have commit marker for second commit")
	}

	// Verify total steps count in All Steps section
	if !strings.Contains(result, "All 7 steps</strong>") {
		t.Error("Should show total of 7 steps in All Steps section")
	}

	// Verify total user prompts count
	if !strings.Contains(result, "4 user prompts</strong>") {
		t.Error("Should show total of 4 user prompts")
	}

	// Verify no duplicates - count occurrences of unique prompts
	firstPromptCount := strings.Count(result, "First prompt")
	if firstPromptCount != 2 { // Once in Prompts section, once in Full Transcript
		t.Errorf("'First prompt' should appear exactly 2 times (got %d)", firstPromptCount)
	}

	thirdPromptCount := strings.Count(result, "Third prompt")
	if thirdPromptCount != 2 { // Once in Prompts section, once in Full Transcript
		t.Errorf("'Third prompt' should appear exactly 2 times (got %d)", thirdPromptCount)
	}
}

func TestFormatMarkdownEntryCollapsible_ShortText(t *testing.T) {
	entry := PromptEntry{
		Type: "PROMPT",
		Text: "Short prompt text",
		Time: time.Date(2025, 1, 15, 9, 30, 0, 0, time.Local),
	}

	result := formatMarkdownEntryCollapsible(entry)

	// Short prompts should use <details open>
	if !strings.Contains(result, "<details open>") {
		t.Error("Short prompts should use <details open>")
	}

	// Should contain the full text in summary
	if !strings.Contains(result, "Short prompt text") {
		t.Error("Should contain full text")
	}
}

func TestFormatMarkdownEntryCollapsible_LongText(t *testing.T) {
	// Create entry with 300+ char text where CONTINUATION is past 247 chars
	// 260 'a's + "CONTINUATION" + 50 'b's = 322 chars total
	longText := strings.Repeat("a", 260) + "CONTINUATION" + strings.Repeat("b", 50)
	entry := PromptEntry{
		Type: "PROMPT",
		Text: longText,
		Time: time.Date(2025, 1, 15, 9, 30, 0, 0, time.Local),
	}

	result := formatMarkdownEntryCollapsible(entry)

	// Long prompts should use <details> (not open)
	if !strings.Contains(result, "<details><summary>") {
		t.Error("Long prompts should use <details> (collapsed)")
	}

	// Summary should end with "..."
	if !strings.Contains(result, "...") {
		t.Error("Summary should be truncated with ...")
	}

	// Continuation inside details should have the rest of the text
	if !strings.Contains(result, "CONTINUATION") {
		t.Error("Should contain rest of text in details")
	}

	// The continuation should start with "..."
	if !strings.Contains(result, "\n\n...") {
		t.Error("Continuation should start with '...'")
	}
}

func TestFormatMarkdownEntryCollapsible_EscapesHTML(t *testing.T) {
	// Prompt text containing HTML tags that should be escaped
	entry := PromptEntry{
		Type: "PROMPT",
		Text: "Use <details> and <summary> tags for collapsible content",
		Time: time.Date(2025, 1, 15, 9, 30, 0, 0, time.Local),
	}

	result := formatMarkdownEntryCollapsible(entry)

	// The literal <details> in the prompt should be escaped to &lt;details&gt;
	// Otherwise it would break the outer <details> structure
	if strings.Contains(result, "Use <details>") {
		t.Error("HTML tags in prompt text should be escaped, but found unescaped <details>")
	}

	// Should contain escaped version
	if !strings.Contains(result, "&lt;details&gt;") {
		t.Error("Should contain escaped HTML: &lt;details&gt;")
	}

	if !strings.Contains(result, "&lt;summary&gt;") {
		t.Error("Should contain escaped HTML: &lt;summary&gt;")
	}
}

func TestFormatMarkdownEntry_EscapesHTML(t *testing.T) {
	// Test that the non-collapsible format also escapes HTML
	entry := PromptEntry{
		Type: "ASSISTANT",
		Text: "Here is some <script>alert('xss')</script> content",
		Time: time.Date(2025, 1, 15, 9, 30, 0, 0, time.Local),
	}

	result := formatMarkdownEntry(entry)

	// Should not contain unescaped script tag
	if strings.Contains(result, "<script>") {
		t.Error("HTML tags should be escaped")
	}

	// Should contain escaped version
	if !strings.Contains(result, "&lt;script&gt;") {
		t.Error("Should contain escaped HTML")
	}
}

func TestIsAgentSession(t *testing.T) {
	tests := []struct {
		sessionID string
		expected  bool
	}{
		{"agent-aa5fd63", true},
		{"agent-123abc", true},
		{"agent-", true},
		{"fb813892-a738-4fc4-bcf8-b6f175a27a93", false},
		{"7b383e66-9fd6-4c9e-b17e-839042a6cd81", false},
		{"main-session", false},
		{"", false},
		{"AGENT-uppercase", false}, // Case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.sessionID, func(t *testing.T) {
			result := IsAgentSession(tt.sessionID)
			if result != tt.expected {
				t.Errorf("IsAgentSession(%q) = %v, want %v", tt.sessionID, result, tt.expected)
			}
		})
	}
}

func TestRenderMarkdown_AgentSessionCounts(t *testing.T) {
	now := time.Now()
	summary := &Summary{
		CommitsWithNotes:   1,
		TotalUserPrompts:   2, // Main session only
		TotalAgentPrompts:  3, // Agent sessions
		TotalAgentSessions: 2,
		TotalSteps:         10,
		Commits: []CommitSummary{
			{
				ShortSHA: "abc1234",
				Subject:  "Test commit with agents",
				Sessions: []SessionSummary{
					{
						Tool:    "claude-code",
						ID:      "main-session-uuid",
						IsAgent: false,
						Prompts: []PromptEntry{
							{Type: "PROMPT", Text: "User prompt 1", Time: now},
							{Type: "ASSISTANT", Text: "Response", Time: now},
							{Type: "PROMPT", Text: "User prompt 2", Time: now},
						},
					},
					{
						Tool:    "claude-code",
						ID:      "agent-explore1",
						IsAgent: true,
						Prompts: []PromptEntry{
							{Type: "PROMPT", Text: "Agent prompt 1", Time: now},
							{Type: "ASSISTANT", Text: "Agent response", Time: now},
							{Type: "PROMPT", Text: "Agent prompt 2", Time: now},
						},
					},
					{
						Tool:    "claude-code",
						ID:      "agent-explore2",
						IsAgent: true,
						Prompts: []PromptEntry{
							{Type: "PROMPT", Text: "Agent prompt 3", Time: now},
							{Type: "TOOL_USE", Text: "Bash", Time: now},
						},
					},
				},
			},
		},
	}

	result := RenderMarkdown(summary, "")

	// Should show main session prompts only (no agent count)
	if !strings.Contains(result, "| 2 |") {
		t.Errorf("Should show '| 2 |' for main session prompts only, got:\n%s", result)
	}
}

func TestRenderMarkdown_NoAgentSessions(t *testing.T) {
	now := time.Now()
	summary := &Summary{
		CommitsWithNotes:  1,
		TotalUserPrompts:  2,
		TotalAgentPrompts: 0,
		Commits: []CommitSummary{
			{
				ShortSHA: "abc1234",
				Subject:  "Test commit no agents",
				Sessions: []SessionSummary{
					{
						Tool:    "claude-code",
						ID:      "main-session-uuid",
						IsAgent: false,
						Prompts: []PromptEntry{
							{Type: "PROMPT", Text: "User prompt 1", Time: now},
							{Type: "PROMPT", Text: "User prompt 2", Time: now},
						},
					},
				},
			},
		},
	}

	result := RenderMarkdown(summary, "")

	// Should NOT show agent count when there are no agents
	if strings.Contains(result, "(+") {
		t.Errorf("Should not show agent count when there are no agents, got:\n%s", result)
	}

	// Should show just the number
	if !strings.Contains(result, "| 2 |") {
		t.Error("Should show '| 2 |' for user prompts when no agents")
	}
}
