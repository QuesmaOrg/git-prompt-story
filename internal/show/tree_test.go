package show

import (
	"testing"
	"time"

	"github.com/QuesmaOrg/git-prompt-story/internal/ci"
)

// Helper to create a test session summary with prompts
func makeTestSession(id string, prompts []ci.PromptEntry) ci.SessionSummary {
	return ci.SessionSummary{
		Tool:    "claude-code",
		ID:      id,
		IsAgent: false,
		Start:   time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
		End:     time.Date(2025, 1, 15, 11, 0, 0, 0, time.UTC),
		Prompts: prompts,
	}
}

func TestBuildActionNodes_Empty(t *testing.T) {
	sess := makeTestSession("sess1", []ci.PromptEntry{})
	nodes := buildActionNodes(sess, "commit1", 0)

	if len(nodes) != 0 {
		t.Errorf("Expected 0 nodes, got %d", len(nodes))
	}
}

func TestBuildActionNodes_OnlyUserActions(t *testing.T) {
	prompts := []ci.PromptEntry{
		{Type: "PROMPT", Text: "First prompt", Time: time.Now()},
		{Type: "COMMAND", Text: "/compact", Time: time.Now()},
		{Type: "PROMPT", Text: "Second prompt", Time: time.Now()},
	}
	sess := makeTestSession("sess1", prompts)
	nodes := buildActionNodes(sess, "commit1", 0)

	if len(nodes) != 3 {
		t.Errorf("Expected 3 user action nodes, got %d", len(nodes))
	}

	for i, n := range nodes {
		ua, ok := n.(*UserActionNode)
		if !ok {
			t.Errorf("Node %d is not UserActionNode", i)
			continue
		}
		if len(ua.FollowingSteps) != 0 {
			t.Errorf("Node %d has %d following steps, expected 0", i, len(ua.FollowingSteps))
		}
	}
}

func TestBuildActionNodes_WithSteps(t *testing.T) {
	prompts := []ci.PromptEntry{
		{Type: "PROMPT", Text: "Do something", Time: time.Now()},
		{Type: "TOOL_USE", ToolName: "Bash", ToolInput: "ls", Time: time.Now()},
		{Type: "TOOL_USE", ToolName: "Read", ToolInput: "file.go", Time: time.Now()},
		{Type: "PROMPT", Text: "Next prompt", Time: time.Now()},
		{Type: "TOOL_USE", ToolName: "Edit", Time: time.Now()},
	}
	sess := makeTestSession("sess1", prompts)
	nodes := buildActionNodes(sess, "commit1", 0)

	if len(nodes) != 2 {
		t.Errorf("Expected 2 user action nodes, got %d", len(nodes))
	}

	// First user action should have 2 following steps
	ua1, ok := nodes[0].(*UserActionNode)
	if !ok {
		t.Fatal("First node is not UserActionNode")
	}
	if len(ua1.FollowingSteps) != 2 {
		t.Errorf("First action has %d following steps, expected 2", len(ua1.FollowingSteps))
	}

	// Second user action should have 1 following step
	ua2, ok := nodes[1].(*UserActionNode)
	if !ok {
		t.Fatal("Second node is not UserActionNode")
	}
	if len(ua2.FollowingSteps) != 1 {
		t.Errorf("Second action has %d following steps, expected 1", len(ua2.FollowingSteps))
	}
}

func TestBuildActionNodes_StepsBeforeFirstAction(t *testing.T) {
	// Steps before the first user action should be ignored
	prompts := []ci.PromptEntry{
		{Type: "TOOL_USE", ToolName: "Bash", Time: time.Now()}, // Should be ignored
		{Type: "ASSISTANT", Text: "Response", Time: time.Now()}, // Should be ignored
		{Type: "PROMPT", Text: "First prompt", Time: time.Now()},
		{Type: "TOOL_USE", ToolName: "Read", Time: time.Now()},
	}
	sess := makeTestSession("sess1", prompts)
	nodes := buildActionNodes(sess, "commit1", 0)

	if len(nodes) != 1 {
		t.Errorf("Expected 1 user action node, got %d", len(nodes))
	}

	ua, ok := nodes[0].(*UserActionNode)
	if !ok {
		t.Fatal("Node is not UserActionNode")
	}
	if len(ua.FollowingSteps) != 1 {
		t.Errorf("Action has %d following steps, expected 1", len(ua.FollowingSteps))
	}
}

func TestFlattenVisible_CollapsedTree(t *testing.T) {
	// Create a tree with commit -> session -> actions
	tree := &Tree{}

	commit := NewCommitNode(ci.CommitSummary{ShortSHA: "abc1234", Subject: "Test"}, 0)
	session := NewSessionNode(ci.SessionSummary{Tool: "claude-code", ID: "sess1"}, "abc1234", 1)

	// User action with steps
	action := NewUserActionNode(ci.PromptEntry{Type: "PROMPT", Text: "Test"}, "sess1", "abc1234", 2)
	step := NewStepNode(ci.PromptEntry{Type: "TOOL_USE", ToolName: "Bash"}, "sess1", "abc1234", 3)
	action.FollowingSteps = []*StepNode{step}

	session.children = []Node{action}
	commit.children = []Node{session}
	tree.Roots = []Node{commit}

	// Default: commit and session expanded, action collapsed
	visible := tree.FlattenVisible()

	// Should see: commit, session, action (but not step)
	if len(visible) != 3 {
		t.Errorf("Expected 3 visible nodes, got %d", len(visible))
	}

	// Verify types
	if visible[0].Type() != NodeTypeCommit {
		t.Error("First visible node should be commit")
	}
	if visible[1].Type() != NodeTypeSession {
		t.Error("Second visible node should be session")
	}
	if visible[2].Type() != NodeTypeUserAction {
		t.Error("Third visible node should be user action")
	}
}

func TestFlattenVisible_ExpandedAction(t *testing.T) {
	tree := &Tree{}

	action := NewUserActionNode(ci.PromptEntry{Type: "PROMPT", Text: "Test"}, "sess1", "abc1234", 0)
	step1 := NewStepNode(ci.PromptEntry{Type: "TOOL_USE", ToolName: "Bash"}, "sess1", "abc1234", 1)
	step2 := NewStepNode(ci.PromptEntry{Type: "TOOL_USE", ToolName: "Read"}, "sess1", "abc1234", 1)
	action.FollowingSteps = []*StepNode{step1, step2}
	action.SetExpanded(true)

	tree.Roots = []Node{action}

	visible := tree.FlattenVisible()

	// Should see: action + 2 steps = 3 nodes
	if len(visible) != 3 {
		t.Errorf("Expected 3 visible nodes when expanded, got %d", len(visible))
	}
}

func TestTreeExpandCollapse(t *testing.T) {
	tree := &Tree{}

	action := NewUserActionNode(ci.PromptEntry{Type: "PROMPT"}, "sess1", "abc1234", 0)
	step := NewStepNode(ci.PromptEntry{Type: "TOOL_USE"}, "sess1", "abc1234", 1)
	action.FollowingSteps = []*StepNode{step}

	tree.Roots = []Node{action}

	// Initially collapsed
	visible := tree.FlattenVisible()
	if len(visible) != 1 {
		t.Errorf("Collapsed: expected 1 visible, got %d", len(visible))
	}

	// Expand
	tree.Expand(visible, 0)
	visible = tree.FlattenVisible()
	if len(visible) != 2 {
		t.Errorf("Expanded: expected 2 visible, got %d", len(visible))
	}

	// Collapse
	tree.Collapse(visible, 0)
	visible = tree.FlattenVisible()
	if len(visible) != 1 {
		t.Errorf("Collapsed again: expected 1 visible, got %d", len(visible))
	}
}

func TestTreeToggleExpand(t *testing.T) {
	tree := &Tree{}

	action := NewUserActionNode(ci.PromptEntry{Type: "PROMPT"}, "sess1", "abc1234", 0)
	step := NewStepNode(ci.PromptEntry{Type: "TOOL_USE"}, "sess1", "abc1234", 1)
	action.FollowingSteps = []*StepNode{step}

	tree.Roots = []Node{action}

	visible := tree.FlattenVisible()

	// Toggle (should expand)
	tree.ToggleExpand(visible, 0)
	if !action.IsExpanded() {
		t.Error("After toggle, action should be expanded")
	}

	// Toggle again (should collapse)
	tree.ToggleExpand(visible, 0)
	if action.IsExpanded() {
		t.Error("After second toggle, action should be collapsed")
	}
}

func TestTreeExpandAll(t *testing.T) {
	tree := &Tree{}

	commit := NewCommitNode(ci.CommitSummary{ShortSHA: "abc"}, 0)
	action := NewUserActionNode(ci.PromptEntry{Type: "PROMPT"}, "sess1", "abc", 1)
	step := NewStepNode(ci.PromptEntry{Type: "TOOL_USE"}, "sess1", "abc", 2)
	action.FollowingSteps = []*StepNode{step}
	commit.children = []Node{action}
	tree.Roots = []Node{commit}

	// Collapse action first
	action.SetExpanded(false)

	tree.ExpandAll()

	if !action.IsExpanded() {
		t.Error("After ExpandAll, action should be expanded")
	}

	visible := tree.FlattenVisible()
	// commit + action + step = 3
	if len(visible) != 3 {
		t.Errorf("After ExpandAll, expected 3 visible, got %d", len(visible))
	}
}

func TestTreeCollapseAll(t *testing.T) {
	tree := &Tree{}

	commit := NewCommitNode(ci.CommitSummary{ShortSHA: "abc"}, 0)
	action := NewUserActionNode(ci.PromptEntry{Type: "PROMPT"}, "sess1", "abc", 1)
	step := NewStepNode(ci.PromptEntry{Type: "TOOL_USE"}, "sess1", "abc", 2)
	action.FollowingSteps = []*StepNode{step}
	action.SetExpanded(true)
	commit.children = []Node{action}
	tree.Roots = []Node{commit}

	tree.CollapseAll()

	// Commit stays expanded (it's structural), but action collapses
	if !commit.IsExpanded() {
		t.Error("After CollapseAll, commit should still be expanded")
	}
	if action.IsExpanded() {
		t.Error("After CollapseAll, action should be collapsed")
	}
}

func TestCountUserActions(t *testing.T) {
	commit := NewCommitNode(ci.CommitSummary{ShortSHA: "abc"}, 0)
	session := NewSessionNode(ci.SessionSummary{ID: "sess1"}, "abc", 1)

	action1 := NewUserActionNode(ci.PromptEntry{Type: "PROMPT"}, "sess1", "abc", 2)
	action2 := NewUserActionNode(ci.PromptEntry{Type: "COMMAND"}, "sess1", "abc", 2)

	session.children = []Node{action1, action2}
	commit.children = []Node{session}

	count := countUserActions(commit)
	if count != 2 {
		t.Errorf("Expected 2 user actions, got %d", count)
	}
}

func TestCountAllSteps(t *testing.T) {
	action := NewUserActionNode(ci.PromptEntry{Type: "PROMPT"}, "sess1", "abc", 0)
	step1 := NewStepNode(ci.PromptEntry{Type: "TOOL_USE"}, "sess1", "abc", 1)
	step2 := NewStepNode(ci.PromptEntry{Type: "TOOL_USE"}, "sess1", "abc", 1)
	action.FollowingSteps = []*StepNode{step1, step2}

	// Action counts as 1, plus 2 steps = 3 total
	count := countAllSteps(action)
	if count != 3 {
		t.Errorf("Expected 3 total steps, got %d", count)
	}
}

func TestExpandCollapseOutOfBounds(t *testing.T) {
	tree := &Tree{}
	action := NewUserActionNode(ci.PromptEntry{Type: "PROMPT"}, "sess1", "abc", 0)
	tree.Roots = []Node{action}

	visible := tree.FlattenVisible()

	// These should not panic
	tree.Expand(visible, -1)
	tree.Expand(visible, 100)
	tree.Collapse(visible, -1)
	tree.Collapse(visible, 100)
	tree.ToggleExpand(visible, -1)
	tree.ToggleExpand(visible, 100)
}
