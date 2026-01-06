package show

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Styles
var (
	// Panel styles
	listPanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240"))

	detailPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240"))

	statusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("252")).
			Padding(0, 1)

	// Selection styles
	selectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("255"))

	// Tree indent
	indentStr = "  "

	// Expansion indicators
	expandedIndicator   = "▼"
	collapsedIndicator  = "▶"
	nonExpandablePrefix = " "
)

// model is the Bubble Tea model for the TUI
type model struct {
	tree         *Tree
	visible      []Node
	cursor       int
	listOffset   int
	detailOffset int
	width        int
	height       int
	commitSpec   string
	quitting     bool
	err          error
}

// NewModel creates a new TUI model
func NewModel(commitSpec string, full bool) (tea.Model, error) {
	tree, err := LoadTree(commitSpec, full)
	if err != nil {
		return nil, err
	}

	m := model{
		tree:       tree,
		visible:    tree.FlattenVisible(),
		cursor:     0,
		commitSpec: commitSpec,
	}

	return m, nil
}

// Init implements tea.Model
func (m model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		// Navigation
		case "j", "down":
			if m.cursor < len(m.visible)-1 {
				m.cursor++
				m.detailOffset = 0
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
				m.detailOffset = 0
			}
		case "g", "home":
			m.cursor = 0
			m.detailOffset = 0
		case "G", "end":
			m.cursor = len(m.visible) - 1
			m.detailOffset = 0
		case "ctrl+d":
			m.cursor = min(m.cursor+m.listHeight()/2, len(m.visible)-1)
			m.detailOffset = 0
		case "ctrl+u":
			m.cursor = max(m.cursor-m.listHeight()/2, 0)
			m.detailOffset = 0

		// Detail pane scrolling
		case "J", "shift+down":
			m.detailOffset++
		case "K", "shift+up":
			if m.detailOffset > 0 {
				m.detailOffset--
			}

		// Expand/Collapse
		case "e", "enter", "l", "right":
			m.tree.Expand(m.visible, m.cursor)
			m.visible = m.tree.FlattenVisible()
		case "c", "h", "left":
			m.tree.Collapse(m.visible, m.cursor)
			m.visible = m.tree.FlattenVisible()
		case "E":
			m.tree.ExpandAll()
			m.visible = m.tree.FlattenVisible()
		case "C":
			m.tree.CollapseAll()
			m.visible = m.tree.FlattenVisible()
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	// Ensure cursor stays in bounds
	if m.cursor >= len(m.visible) {
		m.cursor = max(0, len(m.visible)-1)
	}

	// Adjust list scroll to keep cursor visible
	m.adjustListScroll()

	return m, nil
}

// View implements tea.Model
func (m model) View() string {
	if m.quitting {
		return ""
	}

	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	if len(m.visible) == 0 {
		return "No entries to display\n"
	}

	// Wait for terminal dimensions
	if m.width < 20 || m.height < 10 {
		return "Loading..."
	}

	// Calculate panel dimensions
	// Leave room for status bar (1 line) and borders (2 lines each panel)
	contentHeight := max(m.height-3, 5)
	listWidth := max(m.width*2/5, 10)
	detailWidth := max(m.width-listWidth-1, 10)

	// Render panels
	listPanel := m.renderList(max(listWidth-2, 5), max(contentHeight-2, 3))
	detailPanel := m.renderDetail(max(detailWidth-2, 5), max(contentHeight-2, 3))

	// Style panels
	listPanel = listPanelStyle.
		Width(max(listWidth-2, 5)).
		Height(max(contentHeight-2, 3)).
		Render(listPanel)

	detailPanel = detailPanelStyle.
		Width(max(detailWidth-2, 5)).
		Height(max(contentHeight-2, 3)).
		Render(detailPanel)

	// Join panels horizontally
	content := lipgloss.JoinHorizontal(lipgloss.Top, listPanel, detailPanel)

	// Status bar
	statusBar := m.renderStatusBar()

	return lipgloss.JoinVertical(lipgloss.Left, content, statusBar)
}

// renderList renders the tree list panel
func (m model) renderList(width, height int) string {
	var lines []string

	// Calculate visible range
	visibleStart := m.listOffset
	visibleEnd := min(m.listOffset+height, len(m.visible))

	for i := visibleStart; i < visibleEnd; i++ {
		node := m.visible[i]
		line := m.renderTreeLine(node, width, i == m.cursor)
		lines = append(lines, line)
	}

	// Pad with empty lines if needed
	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", width))
	}

	return strings.Join(lines, "\n")
}

// renderTreeLine renders a single tree line
func (m model) renderTreeLine(node Node, width int, selected bool) string {
	// Build indentation
	indent := strings.Repeat(indentStr, node.Depth())

	// Build expansion indicator
	var indicator string
	if node.IsExpandable() {
		if node.IsExpanded() {
			indicator = expandedIndicator
		} else {
			indicator = collapsedIndicator
		}
	} else {
		indicator = nonExpandablePrefix
	}

	// Build the line
	label := node.Label()
	line := fmt.Sprintf("%s%s %s", indent, indicator, label)

	// Truncate if needed
	if len(line) > width {
		line = line[:width-3] + "..."
	}

	// Pad to width
	if len(line) < width {
		line = line + strings.Repeat(" ", width-len(line))
	}

	// Apply selection style
	if selected {
		line = selectedStyle.Render(line)
	}

	return line
}

// renderDetail renders the detail panel for the selected node
func (m model) renderDetail(width, height int) string {
	if m.cursor >= len(m.visible) {
		return "No selection"
	}

	node := m.visible[m.cursor]
	var sb strings.Builder

	// Render based on node type
	switch n := node.(type) {
	case *CommitNode:
		sb.WriteString(fmt.Sprintf("Commit: %s\n", n.ShortSHA))
		sb.WriteString(fmt.Sprintf("Subject: %s\n", n.Subject))
		sb.WriteString(fmt.Sprintf("Sessions: %d\n", len(n.Sessions)))

	case *SessionNode:
		sb.WriteString(fmt.Sprintf("Session: %s\n", formatToolName(n.Tool)))
		sb.WriteString(fmt.Sprintf("ID: %s\n", n.ID))
		if n.IsAgent {
			sb.WriteString("Type: Agent session\n")
		}
		if !n.Start.IsZero() {
			sb.WriteString(fmt.Sprintf("Start: %s\n", n.Start.Local().Format("2006-01-02 15:04:05")))
		}
		if !n.End.IsZero() {
			sb.WriteString(fmt.Sprintf("End: %s\n", n.End.Local().Format("2006-01-02 15:04:05")))
		}

	case *UserActionNode:
		entry := n.Entry()
		sb.WriteString(fmt.Sprintf("Type: %s %s\n", getTypeEmoji(entry.Type), entry.Type))
		sb.WriteString(fmt.Sprintf("Time: %s\n", entry.Time.Local().Format("2006-01-02 15:04:05")))
		sb.WriteString(fmt.Sprintf("Session: %s\n", n.SessionID[:min(8, len(n.SessionID))]))
		sb.WriteString("\n")

		// Content based on type
		switch entry.Type {
		case "DECISION":
			if entry.DecisionHeader != "" {
				sb.WriteString(fmt.Sprintf("Question: %s\n", entry.DecisionHeader))
			}
			sb.WriteString(fmt.Sprintf("Prompt: %s\n", entry.Text))
			if entry.DecisionAnswer != "" {
				sb.WriteString(fmt.Sprintf("Answer: %s\n", entry.DecisionAnswer))
			}
		default:
			sb.WriteString("Content:\n")
			sb.WriteString(wrapText(entry.Text, width-2))
		}

		// Show following steps in detail panel (when collapsed, as preview)
		if len(n.FollowingSteps) > 0 && !n.IsExpanded() {
			sb.WriteString("\n")
			sb.WriteString(strings.Repeat("─", min(width-2, 40)))
			sb.WriteString(fmt.Sprintf("\nFollowing steps (%d) - press 'e' to expand:\n", len(n.FollowingSteps)))
			for _, step := range n.FollowingSteps {
				stepEntry := step.Entry()
				emoji := getTypeEmoji(stepEntry.Type)
				timeStr := stepEntry.Time.Local().Format("15:04")
				if stepEntry.Type == "TOOL_USE" && stepEntry.ToolName != "" {
					input := truncateText(stepEntry.ToolInput, width-20)
					sb.WriteString(fmt.Sprintf("%s %s %s: %s\n", emoji, timeStr, stepEntry.ToolName, input))
				} else {
					text := truncateText(stepEntry.Text, width-12)
					sb.WriteString(fmt.Sprintf("%s %s %s\n", emoji, timeStr, text))
				}
			}
		} else if len(n.FollowingSteps) > 0 {
			sb.WriteString(fmt.Sprintf("\n\n%d steps expanded in tree", len(n.FollowingSteps)))
		}

	case *StepNode:
		entry := n.Entry()
		sb.WriteString(fmt.Sprintf("Type: %s %s\n", getTypeEmoji(entry.Type), entry.Type))
		sb.WriteString(fmt.Sprintf("Time: %s\n", entry.Time.Local().Format("2006-01-02 15:04:05")))
		sb.WriteString("\n")

		if entry.Type == "TOOL_USE" {
			sb.WriteString(fmt.Sprintf("Tool: %s\n", entry.ToolName))
			if entry.ToolInput != "" {
				sb.WriteString("\nInput:\n")
				sb.WriteString(wrapText(entry.ToolInput, width-2))
			}
			if entry.ToolOutput != "" {
				sb.WriteString("\n\nOutput:\n")
				sb.WriteString(wrapText(entry.ToolOutput, width-2))
			}
		} else {
			sb.WriteString("Content:\n")
			sb.WriteString(wrapText(entry.Text, width-2))
		}
	}

	content := sb.String()
	lines := strings.Split(content, "\n")

	// Apply scroll offset
	if m.detailOffset > 0 && m.detailOffset < len(lines) {
		lines = lines[m.detailOffset:]
	}

	// Truncate to height
	if len(lines) > height {
		lines = lines[:height]
	}

	return strings.Join(lines, "\n")
}

// renderStatusBar renders the status bar
func (m model) renderStatusBar() string {
	// Position info
	position := fmt.Sprintf("%d/%d", m.cursor+1, len(m.visible))

	// Context info
	var context string
	if m.tree.TotalCommits > 1 {
		context = fmt.Sprintf("%d commits", m.tree.TotalCommits)
	} else {
		context = fmt.Sprintf("%d actions", m.tree.TotalActions)
	}

	// Keybindings help
	help := "j/k:nav  e:expand  c:collapse  E/C:all  J/K:scroll  q:quit"

	// Build status bar
	status := fmt.Sprintf(" %s | %s | %s", position, context, help)

	return statusBarStyle.Width(m.width).Render(status)
}

// Helper functions

func (m model) listHeight() int {
	return max(m.height-5, 1) // Account for borders and status bar
}

func (m *model) adjustListScroll() {
	visibleHeight := m.listHeight()

	// Scroll up if cursor is above visible area
	if m.cursor < m.listOffset {
		m.listOffset = m.cursor
	}

	// Scroll down if cursor is below visible area
	if m.cursor >= m.listOffset+visibleHeight {
		m.listOffset = m.cursor - visibleHeight + 1
	}
}

func wrapText(s string, width int) string {
	if width < 1 {
		width = 1
	}

	var result strings.Builder
	for _, line := range strings.Split(s, "\n") {
		for len(line) > width {
			result.WriteString(line[:width])
			result.WriteString("\n")
			line = line[width:]
		}
		result.WriteString(line)
		result.WriteString("\n")
	}
	return strings.TrimSuffix(result.String(), "\n")
}


// RunTUI starts the interactive TUI
func RunTUI(commitSpec string, full bool) error {
	m, err := NewModel(commitSpec, full)
	if err != nil {
		return err
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
}
