package show

import (
	"github.com/QuesmaOrg/git-prompt-story/internal/ci"
)

// Tree represents the hierarchical tree of nodes
type Tree struct {
	Roots        []Node // Top-level nodes (commits or sessions)
	TotalCommits int
	TotalActions int // User actions only
	TotalSteps   int // All steps
}

// LoadTree builds a tree from the given commit spec
func LoadTree(commitSpec string, full bool) (*Tree, error) {
	summary, err := ci.GenerateSummary(commitSpec, full)
	if err != nil {
		return nil, err
	}

	tree := &Tree{
		TotalCommits: len(summary.Commits),
	}

	// Determine if we need commit-level nodes
	// Show commits if there are multiple commits
	showCommits := len(summary.Commits) > 1

	for _, commit := range summary.Commits {
		if showCommits {
			commitNode := NewCommitNode(commit, 0)
			tree.Roots = append(tree.Roots, commitNode)

			// Build sessions under commit
			for _, sess := range commit.Sessions {
				sessNode := buildSessionNode(sess, commit.ShortSHA, 1)
				commitNode.children = append(commitNode.children, sessNode)
				tree.TotalActions += countUserActions(sessNode)
				tree.TotalSteps += countAllSteps(sessNode)
			}
		} else {
			// Single commit - show sessions at root level
			// Only show session headers if there are multiple sessions
			showSessions := len(commit.Sessions) > 1

			for _, sess := range commit.Sessions {
				if showSessions {
					sessNode := buildSessionNode(sess, commit.ShortSHA, 0)
					tree.Roots = append(tree.Roots, sessNode)
					tree.TotalActions += countUserActions(sessNode)
					tree.TotalSteps += countAllSteps(sessNode)
				} else {
					// Single session - show actions at root level
					nodes := buildActionNodes(sess, commit.ShortSHA, 0)
					tree.Roots = append(tree.Roots, nodes...)
					for _, n := range nodes {
						if ua, ok := n.(*UserActionNode); ok {
							tree.TotalActions++
							tree.TotalSteps += 1 + len(ua.children)
						}
					}
				}
			}
		}
	}

	return tree, nil
}

// buildSessionNode creates a session node with its action children
func buildSessionNode(sess ci.SessionSummary, commitSHA string, depth int) *SessionNode {
	sessNode := NewSessionNode(sess, commitSHA, depth)
	actionNodes := buildActionNodes(sess, commitSHA, depth+1)
	sessNode.children = actionNodes
	return sessNode
}

// buildActionNodes creates user action nodes with step groups between them
func buildActionNodes(sess ci.SessionSummary, commitSHA string, depth int) []Node {
	var nodes []Node
	var pendingSteps []*StepNode

	for _, entry := range sess.Prompts {
		if ci.IsUserAction(entry.Type) {
			// Create a user action node
			actionNode := NewUserActionNode(entry, sess.ID, commitSHA, depth)

			// If there are pending steps, attach them to the previous action
			if len(nodes) > 0 && len(pendingSteps) > 0 {
				prevAction := findLastUserAction(nodes)
				if prevAction != nil {
					stepGroup := NewStepGroupNode(pendingSteps, sess.ID, commitSHA, depth+1)
					prevAction.children = append(prevAction.children, stepGroup)
				}
				pendingSteps = nil
			}

			nodes = append(nodes, actionNode)
		} else {
			// This is a step (TOOL_USE, ASSISTANT, etc.)
			stepNode := NewStepNode(entry, sess.ID, commitSHA, depth+2)
			pendingSteps = append(pendingSteps, stepNode)
		}
	}

	// Attach remaining steps to the last action
	if len(nodes) > 0 && len(pendingSteps) > 0 {
		lastAction := findLastUserAction(nodes)
		if lastAction != nil {
			stepGroup := NewStepGroupNode(pendingSteps, sess.ID, commitSHA, depth+1)
			lastAction.children = append(lastAction.children, stepGroup)
		}
	}

	return nodes
}

// findLastUserAction finds the last UserActionNode in the slice
func findLastUserAction(nodes []Node) *UserActionNode {
	for i := len(nodes) - 1; i >= 0; i-- {
		if ua, ok := nodes[i].(*UserActionNode); ok {
			return ua
		}
	}
	return nil
}

// FlattenVisible returns all currently visible nodes in display order
func (t *Tree) FlattenVisible() []Node {
	var result []Node
	for _, root := range t.Roots {
		result = flattenNode(root, result)
	}
	return result
}

func flattenNode(n Node, result []Node) []Node {
	result = append(result, n)

	if n.IsExpandable() && n.IsExpanded() {
		for _, child := range n.Children() {
			result = flattenNode(child, result)
		}
	}

	return result
}

// ToggleExpand toggles the expansion state of the node at the given index
func (t *Tree) ToggleExpand(visible []Node, index int) {
	if index < 0 || index >= len(visible) {
		return
	}
	n := visible[index]
	if n.IsExpandable() {
		n.SetExpanded(!n.IsExpanded())
	}
}

// Expand expands the node at the given index
func (t *Tree) Expand(visible []Node, index int) {
	if index < 0 || index >= len(visible) {
		return
	}
	n := visible[index]
	if n.IsExpandable() && !n.IsExpanded() {
		n.SetExpanded(true)
	}
}

// Collapse collapses the node at the given index
func (t *Tree) Collapse(visible []Node, index int) {
	if index < 0 || index >= len(visible) {
		return
	}
	n := visible[index]
	if n.IsExpandable() && n.IsExpanded() {
		n.SetExpanded(false)
	}
}

// ExpandAll expands all expandable nodes
func (t *Tree) ExpandAll() {
	for _, root := range t.Roots {
		expandAllRecursive(root)
	}
}

func expandAllRecursive(n Node) {
	if n.IsExpandable() {
		n.SetExpanded(true)
		for _, child := range n.Children() {
			expandAllRecursive(child)
		}
	}
}

// CollapseAll collapses all expandable nodes except commits/sessions
func (t *Tree) CollapseAll() {
	for _, root := range t.Roots {
		collapseAllRecursive(root)
	}
}

func collapseAllRecursive(n Node) {
	// Keep commits and sessions expanded, collapse everything else
	switch n.Type() {
	case NodeTypeCommit, NodeTypeSession:
		n.SetExpanded(true)
		for _, child := range n.Children() {
			collapseAllRecursive(child)
		}
	default:
		n.SetExpanded(false)
	}
}

// Helper functions for counting

func countUserActions(n Node) int {
	count := 0
	if n.Type() == NodeTypeUserAction {
		count = 1
	}
	for _, child := range n.Children() {
		count += countUserActions(child)
	}
	return count
}

func countAllSteps(n Node) int {
	count := 0
	switch n.Type() {
	case NodeTypeUserAction, NodeTypeStep:
		count = 1
	}
	for _, child := range n.Children() {
		count += countAllSteps(child)
	}
	return count
}
