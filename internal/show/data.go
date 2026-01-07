package show

import (
	"fmt"
	"time"

	"github.com/QuesmaOrg/git-prompt-story/internal/ci"
	"github.com/QuesmaOrg/git-prompt-story/internal/display"
	"github.com/QuesmaOrg/git-prompt-story/internal/note"
)

// NodeType represents the type of node in the tree
type NodeType int

const (
	NodeTypeCommit NodeType = iota
	NodeTypeSession
	NodeTypeUserAction
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
	toolName := note.FormatToolName(s.Tool)
	return fmt.Sprintf("Session: %s (%s)", toolName, s.ShortID)
}

// UserActionNode represents a user action (PROMPT, COMMAND, TOOL_REJECT, DECISION)
type UserActionNode struct {
	BaseNode
	entry          ci.PromptEntry
	Tool           string
	SessionID      string
	CommitSHA      string
	FollowingSteps []*StepNode // Steps that follow this user action (shown in detail panel)
}

func NewUserActionNode(entry ci.PromptEntry, tool, sessionID, commitSHA string, depth int) *UserActionNode {
	return &UserActionNode{
		BaseNode:  BaseNode{depth: depth, expanded: false},
		entry:     entry,
		Tool:      tool,
		SessionID: sessionID,
		CommitSHA: commitSHA,
	}
}

func (u *UserActionNode) Type() NodeType         { return NodeTypeUserAction }
func (u *UserActionNode) IsExpandable() bool     { return len(u.FollowingSteps) > 0 }
func (u *UserActionNode) Entry() *ci.PromptEntry { return &u.entry }
func (u *UserActionNode) Time() time.Time        { return u.entry.Time }

// Children returns the following steps as child nodes (for tree expansion)
func (u *UserActionNode) Children() []Node {
	nodes := make([]Node, len(u.FollowingSteps))
	for i, s := range u.FollowingSteps {
		nodes[i] = s
	}
	return nodes
}

func (u *UserActionNode) Label() string {
	emoji := display.GetTypeEmoji(u.entry.Type)
	timeStr := u.entry.Time.Local().Format("15:04")
	text := display.TruncateText(u.entry.Text, 25)
	return fmt.Sprintf("%s %s %s", emoji, timeStr, text)
}

// StepNode represents an individual step (TOOL_USE, ASSISTANT, etc.)
type StepNode struct {
	BaseNode
	entry     ci.PromptEntry
	Tool      string
	SessionID string
	CommitSHA string
}

func NewStepNode(entry ci.PromptEntry, tool, sessionID, commitSHA string, depth int) *StepNode {
	return &StepNode{
		BaseNode:  BaseNode{depth: depth, expanded: false},
		entry:     entry,
		Tool:      tool,
		SessionID: sessionID,
		CommitSHA: commitSHA,
	}
}

func (s *StepNode) Type() NodeType        { return NodeTypeStep }
func (s *StepNode) IsExpandable() bool    { return false }
func (s *StepNode) Entry() *ci.PromptEntry { return &s.entry }
func (s *StepNode) Time() time.Time       { return s.entry.Time }

func (s *StepNode) Label() string {
	emoji := display.GetTypeEmoji(s.entry.Type)
	timeStr := s.entry.Time.Local().Format("15:04")

	// For tool uses, show tool name and truncated input
	if s.entry.Type == "TOOL_USE" && s.entry.ToolName != "" {
		input := display.TruncateText(s.entry.ToolInput, 20)
		return fmt.Sprintf("%s %s %s: %s", emoji, timeStr, s.entry.ToolName, input)
	}

	text := display.TruncateText(s.entry.Text, 25)
	return fmt.Sprintf("%s %s %s", emoji, timeStr, text)
}
