package show

import (
	"fmt"
	"time"

	"github.com/QuesmaOrg/git-prompt-story/internal/ci"
)

// NodeType represents the type of node in the tree
type NodeType int

const (
	NodeTypeCommit NodeType = iota
	NodeTypeSession
	NodeTypeUserAction
	NodeTypeStepGroup
	NodeTypeStep
)

// Node represents a node in the tree hierarchy
type Node interface {
	Type() NodeType
	Label() string       // Short label for the tree view
	Depth() int          // Indentation level
	IsExpandable() bool  // Can this node be expanded?
	IsExpanded() bool    // Is this node currently expanded?
	SetExpanded(bool)    // Set expansion state
	Children() []Node    // Child nodes (nil if not expandable or leaf)
	Entry() *ci.PromptEntry // The underlying entry (nil for container nodes)
	Time() time.Time     // Time for sorting/display
}

// BaseNode provides common fields for all node types
type BaseNode struct {
	depth    int
	expanded bool
	children []Node
}

func (b *BaseNode) Depth() int           { return b.depth }
func (b *BaseNode) IsExpanded() bool     { return b.expanded }
func (b *BaseNode) SetExpanded(e bool)   { b.expanded = e }
func (b *BaseNode) Children() []Node     { return b.children }
func (b *BaseNode) Entry() *ci.PromptEntry { return nil }

// CommitNode represents a commit in the tree
type CommitNode struct {
	BaseNode
	SHA      string
	ShortSHA string
	Subject  string
	Sessions []*SessionNode
}

func NewCommitNode(cs ci.CommitSummary, depth int) *CommitNode {
	return &CommitNode{
		BaseNode: BaseNode{depth: depth, expanded: true},
		SHA:      cs.SHA,
		ShortSHA: cs.ShortSHA,
		Subject:  cs.Subject,
	}
}

func (c *CommitNode) Type() NodeType      { return NodeTypeCommit }
func (c *CommitNode) IsExpandable() bool  { return true }
func (c *CommitNode) Time() time.Time     { return time.Time{} }

func (c *CommitNode) Label() string {
	subject := c.Subject
	if len(subject) > 30 {
		subject = subject[:27] + "..."
	}
	return fmt.Sprintf("Commit: %s - %s", c.ShortSHA, subject)
}

// SessionNode represents a session within a commit
type SessionNode struct {
	BaseNode
	Tool      string
	ID        string
	ShortID   string
	IsAgent   bool
	Start     time.Time
	End       time.Time
	CommitSHA string // Parent commit
}

func NewSessionNode(ss ci.SessionSummary, commitSHA string, depth int) *SessionNode {
	shortID := ss.ID
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}
	return &SessionNode{
		BaseNode:  BaseNode{depth: depth, expanded: true},
		Tool:      ss.Tool,
		ID:        ss.ID,
		ShortID:   shortID,
		IsAgent:   ss.IsAgent,
		Start:     ss.Start,
		End:       ss.End,
		CommitSHA: commitSHA,
	}
}

func (s *SessionNode) Type() NodeType      { return NodeTypeSession }
func (s *SessionNode) IsExpandable() bool  { return true }
func (s *SessionNode) Time() time.Time     { return s.Start }

func (s *SessionNode) Label() string {
	toolName := formatToolName(s.Tool)
	return fmt.Sprintf("Session: %s (%s)", toolName, s.ShortID)
}

// UserActionNode represents a user action (PROMPT, COMMAND, TOOL_REJECT, DECISION)
type UserActionNode struct {
	BaseNode
	entry     ci.PromptEntry
	SessionID string
	CommitSHA string
}

func NewUserActionNode(entry ci.PromptEntry, sessionID, commitSHA string, depth int) *UserActionNode {
	return &UserActionNode{
		BaseNode:  BaseNode{depth: depth, expanded: false},
		entry:     entry,
		SessionID: sessionID,
		CommitSHA: commitSHA,
	}
}

func (u *UserActionNode) Type() NodeType        { return NodeTypeUserAction }
func (u *UserActionNode) IsExpandable() bool    { return len(u.children) > 0 }
func (u *UserActionNode) Entry() *ci.PromptEntry { return &u.entry }
func (u *UserActionNode) Time() time.Time       { return u.entry.Time }

func (u *UserActionNode) Label() string {
	emoji := getTypeEmoji(u.entry.Type)
	timeStr := u.entry.Time.Local().Format("15:04")
	text := truncateText(u.entry.Text, 25)
	return fmt.Sprintf("%s %s %s", emoji, timeStr, text)
}

// StepGroupNode represents a group of collapsed steps between user actions
type StepGroupNode struct {
	BaseNode
	Steps     []*StepNode
	SessionID string
	CommitSHA string
}

func NewStepGroupNode(steps []*StepNode, sessionID, commitSHA string, depth int) *StepGroupNode {
	sg := &StepGroupNode{
		BaseNode:  BaseNode{depth: depth, expanded: false},
		Steps:     steps,
		SessionID: sessionID,
		CommitSHA: commitSHA,
	}
	// Set children to the steps
	sg.children = make([]Node, len(steps))
	for i, s := range steps {
		sg.children[i] = s
	}
	return sg
}

func (sg *StepGroupNode) Type() NodeType     { return NodeTypeStepGroup }
func (sg *StepGroupNode) IsExpandable() bool { return true }
func (sg *StepGroupNode) Time() time.Time {
	if len(sg.Steps) > 0 {
		return sg.Steps[0].entry.Time
	}
	return time.Time{}
}

func (sg *StepGroupNode) Label() string {
	n := len(sg.Steps)
	if n == 1 {
		return "├─ 1 step"
	}
	return fmt.Sprintf("├─ %d steps", n)
}

// StepNode represents an individual step (TOOL_USE, ASSISTANT, etc.)
type StepNode struct {
	BaseNode
	entry     ci.PromptEntry
	SessionID string
	CommitSHA string
}

func NewStepNode(entry ci.PromptEntry, sessionID, commitSHA string, depth int) *StepNode {
	return &StepNode{
		BaseNode:  BaseNode{depth: depth, expanded: false},
		entry:     entry,
		SessionID: sessionID,
		CommitSHA: commitSHA,
	}
}

func (s *StepNode) Type() NodeType        { return NodeTypeStep }
func (s *StepNode) IsExpandable() bool    { return false }
func (s *StepNode) Entry() *ci.PromptEntry { return &s.entry }
func (s *StepNode) Time() time.Time       { return s.entry.Time }

func (s *StepNode) Label() string {
	emoji := getTypeEmoji(s.entry.Type)
	timeStr := s.entry.Time.Local().Format("15:04")

	// For tool uses, show tool name and truncated input
	if s.entry.Type == "TOOL_USE" && s.entry.ToolName != "" {
		input := truncateText(s.entry.ToolInput, 20)
		return fmt.Sprintf("%s %s %s: %s", emoji, timeStr, s.entry.ToolName, input)
	}

	text := truncateText(s.entry.Text, 25)
	return fmt.Sprintf("%s %s %s", emoji, timeStr, text)
}

// Helper functions

func getTypeEmoji(entryType string) string {
	switch entryType {
	case "PROMPT":
		return "💬"
	case "COMMAND":
		return "📋"
	case "TOOL_REJECT":
		return "❌"
	case "DECISION":
		return "❓"
	case "TOOL_USE":
		return "🔧"
	case "ASSISTANT":
		return "🤖"
	case "TOOL_RESULT":
		return "📤"
	default:
		return "•"
	}
}

func truncateText(s string, maxLen int) string {
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

func formatToolName(tool string) string {
	switch tool {
	case "claude-code":
		return "Claude Code"
	default:
		return tool
	}
}
