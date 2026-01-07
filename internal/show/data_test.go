package show

import (
	"testing"
	"time"

	"github.com/QuesmaOrg/git-prompt-story/internal/ci"
)

// Note: Tests for GetTypeEmoji and TruncateText are in internal/display/display_test.go
// Note: Tests for FormatToolName are in internal/note/metadata_test.go

func TestNewCommitNode(t *testing.T) {
	cs := ci.CommitSummary{
		SHA:      "abc123def456",
		ShortSHA: "abc123d",
		Subject:  "Fix bug in authentication",
	}

	node := NewCommitNode(cs, 0)

	if node.Type() != NodeTypeCommit {
		t.Errorf("Type() = %v, want NodeTypeCommit", node.Type())
	}
	if !node.IsExpandable() {
		t.Error("IsExpandable() = false, want true")
	}
	if !node.IsExpanded() {
		t.Error("IsExpanded() = false, want true (default expanded)")
	}
	if node.Depth() != 0 {
		t.Errorf("Depth() = %d, want 0", node.Depth())
	}
	if node.SHA != cs.SHA {
		t.Errorf("SHA = %q, want %q", node.SHA, cs.SHA)
	}
	if node.ShortSHA != cs.ShortSHA {
		t.Errorf("ShortSHA = %q, want %q", node.ShortSHA, cs.ShortSHA)
	}
}

func TestNewSessionNode(t *testing.T) {
	ss := ci.SessionSummary{
		Tool:    "claude-code",
		ID:      "abc12345-6789-0abc-def0-123456789abc",
		IsAgent: false,
		Start:   time.Now(),
		End:     time.Now().Add(time.Hour),
	}

	node := NewSessionNode(ss, "commit123", 1)

	if node.Type() != NodeTypeSession {
		t.Errorf("Type() = %v, want NodeTypeSession", node.Type())
	}
	if !node.IsExpandable() {
		t.Error("IsExpandable() = false, want true")
	}
	if node.Depth() != 1 {
		t.Errorf("Depth() = %d, want 1", node.Depth())
	}
	if node.ShortID != "abc12345" {
		t.Errorf("ShortID = %q, want %q", node.ShortID, "abc12345")
	}
	if node.CommitSHA != "commit123" {
		t.Errorf("CommitSHA = %q, want %q", node.CommitSHA, "commit123")
	}

	// Test label format
	label := node.Label()
	if label != "Session: Claude Code (abc12345)" {
		t.Errorf("Label() = %q, want %q", label, "Session: Claude Code (abc12345)")
	}
}

func TestNewUserActionNode(t *testing.T) {
	entry := ci.PromptEntry{
		Type: "PROMPT",
		Text: "Fix the bug please",
		Time: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	node := NewUserActionNode(entry, "session123", "commit123", 2)

	if node.Type() != NodeTypeUserAction {
		t.Errorf("Type() = %v, want NodeTypeUserAction", node.Type())
	}
	// No following steps yet, so not expandable
	if node.IsExpandable() {
		t.Error("IsExpandable() = true, want false (no steps)")
	}
	if node.Depth() != 2 {
		t.Errorf("Depth() = %d, want 2", node.Depth())
	}
	if node.Entry().Type != "PROMPT" {
		t.Errorf("Entry().Type = %q, want %q", node.Entry().Type, "PROMPT")
	}

	// Add following steps and check expandability
	stepEntry := ci.PromptEntry{Type: "TOOL_USE", ToolName: "Bash"}
	node.FollowingSteps = append(node.FollowingSteps, NewStepNode(stepEntry, "session123", "commit123", 3))

	if !node.IsExpandable() {
		t.Error("IsExpandable() = false after adding steps, want true")
	}
	if len(node.Children()) != 1 {
		t.Errorf("Children() length = %d, want 1", len(node.Children()))
	}
}

func TestNewStepNode(t *testing.T) {
	entry := ci.PromptEntry{
		Type:      "TOOL_USE",
		ToolName:  "Bash",
		ToolInput: "ls -la",
		Time:      time.Date(2025, 1, 15, 10, 31, 0, 0, time.UTC),
	}

	node := NewStepNode(entry, "session123", "commit123", 3)

	if node.Type() != NodeTypeStep {
		t.Errorf("Type() = %v, want NodeTypeStep", node.Type())
	}
	if node.IsExpandable() {
		t.Error("IsExpandable() = true, want false")
	}
	if node.Depth() != 3 {
		t.Errorf("Depth() = %d, want 3", node.Depth())
	}
	if node.Entry().ToolName != "Bash" {
		t.Errorf("Entry().ToolName = %q, want %q", node.Entry().ToolName, "Bash")
	}

	// Test label format for tool use
	label := node.Label()
	// Should contain emoji, time, tool name
	if label == "" {
		t.Error("Label() is empty")
	}
}

func TestCommitNodeLabel(t *testing.T) {
	tests := []struct {
		name     string
		subject  string
		expected string
	}{
		{
			name:     "short subject",
			subject:  "Fix bug",
			expected: "Commit: abc1234 - Fix bug",
		},
		{
			name:     "long subject truncated",
			subject:  "This is a very long commit message that exceeds the limit",
			expected: "Commit: abc1234 - This is a very long commit ...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := ci.CommitSummary{
				SHA:      "abc1234567890",
				ShortSHA: "abc1234",
				Subject:  tt.subject,
			}
			node := NewCommitNode(cs, 0)
			label := node.Label()
			if label != tt.expected {
				t.Errorf("Label() = %q, want %q", label, tt.expected)
			}
		})
	}
}

func TestNodeExpandCollapse(t *testing.T) {
	cs := ci.CommitSummary{ShortSHA: "abc1234", Subject: "Test"}
	node := NewCommitNode(cs, 0)

	// Default is expanded
	if !node.IsExpanded() {
		t.Error("New commit node should be expanded by default")
	}

	// Collapse
	node.SetExpanded(false)
	if node.IsExpanded() {
		t.Error("After SetExpanded(false), IsExpanded() should be false")
	}

	// Expand again
	node.SetExpanded(true)
	if !node.IsExpanded() {
		t.Error("After SetExpanded(true), IsExpanded() should be true")
	}
}
