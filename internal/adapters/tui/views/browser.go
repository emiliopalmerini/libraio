package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"librarian/internal/adapters/tui/styles"
	"librarian/internal/domain"
	"librarian/internal/ports"
)

// BrowserKeyMap defines key bindings for the browser view
type BrowserKeyMap struct {
	Up      key.Binding
	Down    key.Binding
	Left    key.Binding
	Right   key.Binding
	Enter   key.Binding
	New     key.Binding
	Archive key.Binding
	Move    key.Binding
	Search  key.Binding
	Help    key.Binding
	Quit    key.Binding
}

var BrowserKeys = BrowserKeyMap{
	Up: key.NewBinding(
		key.WithKeys("k", "up"),
		key.WithHelp("k/↑", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("j", "down"),
		key.WithHelp("j/↓", "down"),
	),
	Left: key.NewBinding(
		key.WithKeys("h", "left"),
		key.WithHelp("h/←", "collapse"),
	),
	Right: key.NewBinding(
		key.WithKeys("l", "right"),
		key.WithHelp("l/→", "expand"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "toggle/select"),
	),
	New: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "new"),
	),
	Archive: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "archive"),
	),
	Move: key.NewBinding(
		key.WithKeys("m"),
		key.WithHelp("m", "move"),
	),
	Search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}

// BrowserModel is the model for the tree browser view
type BrowserModel struct {
	repo       ports.VaultRepository
	root       *domain.TreeNode
	flatNodes  []*domain.TreeNode
	cursor     int
	width      int
	height     int
	message    string
	messageErr bool
}

// NewBrowserModel creates a new browser model
func NewBrowserModel(repo ports.VaultRepository) *BrowserModel {
	return &BrowserModel{
		repo: repo,
	}
}

// Init initializes the browser
func (m *BrowserModel) Init() tea.Cmd {
	return m.loadTree
}

func (m *BrowserModel) loadTree() tea.Msg {
	root, err := m.repo.BuildTree()
	if err != nil {
		return errMsg{err}
	}
	return treeLoadedMsg{root}
}

type treeLoadedMsg struct {
	root *domain.TreeNode
}

type errMsg struct {
	err error
}

type childrenLoadedMsg struct {
	node *domain.TreeNode
}

type successMsg struct {
	message string
}

// Update handles messages for the browser
func (m *BrowserModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case treeLoadedMsg:
		m.root = msg.root
		m.refreshFlatNodes()
		return m, nil

	case childrenLoadedMsg:
		m.refreshFlatNodes()
		return m, nil

	case errMsg:
		m.message = msg.err.Error()
		m.messageErr = true
		return m, nil

	case successMsg:
		m.message = msg.message
		m.messageErr = false
		return m, m.Reload()

	case tea.KeyMsg:
		m.message = "" // Clear message on key press

		switch {
		case key.Matches(msg, BrowserKeys.Quit):
			return m, tea.Quit

		case key.Matches(msg, BrowserKeys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil

		case key.Matches(msg, BrowserKeys.Down):
			if m.cursor < len(m.flatNodes)-1 {
				m.cursor++
			}
			return m, nil

		case key.Matches(msg, BrowserKeys.Left):
			if node := m.selectedNode(); node != nil {
				if node.IsExpanded {
					node.Collapse()
					m.refreshFlatNodes()
				} else if node.Parent != nil && node.Parent.Type != domain.IDTypeUnknown {
					// Move to parent
					for i, n := range m.flatNodes {
						if n == node.Parent {
							m.cursor = i
							break
						}
					}
				}
			}
			return m, nil

		case key.Matches(msg, BrowserKeys.Right), key.Matches(msg, BrowserKeys.Enter):
			if node := m.selectedNode(); node != nil {
				if node.Type == domain.IDTypeItem {
					// Items don't expand, could open in editor
					return m, nil
				}
				if !node.IsExpanded {
					node.Expand()
					return m, m.loadNodeChildren(node)
				} else if key.Matches(msg, BrowserKeys.Enter) {
					node.Collapse()
					m.refreshFlatNodes()
				}
			}
			return m, nil

		case key.Matches(msg, BrowserKeys.New):
			// Return command to switch to create view
			if node := m.selectedNode(); node != nil {
				return m, func() tea.Msg {
					return SwitchToCreateMsg{ParentNode: node}
				}
			}
			return m, nil

		case key.Matches(msg, BrowserKeys.Archive):
			if node := m.selectedNode(); node != nil {
				if node.Type == domain.IDTypeItem || node.Type == domain.IDTypeCategory {
					return m, m.archiveNode(node)
				}
			}
			return m, nil

		case key.Matches(msg, BrowserKeys.Move):
			if node := m.selectedNode(); node != nil && node.Type == domain.IDTypeItem {
				return m, func() tea.Msg {
					return SwitchToMoveMsg{SourceNode: node}
				}
			}
			return m, nil

		case key.Matches(msg, BrowserKeys.Search):
			return m, func() tea.Msg {
				return SwitchToSearchMsg{}
			}

		case key.Matches(msg, BrowserKeys.Help):
			return m, func() tea.Msg {
				return SwitchToHelpMsg{}
			}
		}
	}

	return m, nil
}

func (m *BrowserModel) loadNodeChildren(node *domain.TreeNode) tea.Cmd {
	return func() tea.Msg {
		if err := m.repo.LoadChildren(node); err != nil {
			return errMsg{err}
		}
		return childrenLoadedMsg{node}
	}
}

func (m *BrowserModel) archiveNode(node *domain.TreeNode) tea.Cmd {
	return func() tea.Msg {
		if err := m.repo.Archive(node.ID); err != nil {
			return errMsg{err}
		}
		return successMsg{fmt.Sprintf("Archived %s", node.ID)}
	}
}

func (m *BrowserModel) selectedNode() *domain.TreeNode {
	if m.cursor >= 0 && m.cursor < len(m.flatNodes) {
		return m.flatNodes[m.cursor]
	}
	return nil
}

func (m *BrowserModel) refreshFlatNodes() {
	if m.root == nil {
		return
	}
	m.flatNodes = m.root.Flatten()
	// Skip root node in display
	if len(m.flatNodes) > 0 {
		m.flatNodes = m.flatNodes[1:]
	}
	// Clamp cursor
	if m.cursor >= len(m.flatNodes) {
		m.cursor = len(m.flatNodes) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

// View renders the browser
func (m *BrowserModel) View() string {
	if m.root == nil {
		return "Loading..."
	}

	var b strings.Builder

	// Title
	b.WriteString(styles.Title.Render("Librarian"))
	b.WriteString("\n")
	b.WriteString(styles.Subtitle.Render("Johnny Decimal Vault Manager"))
	b.WriteString("\n\n")

	// Tree
	for i, node := range m.flatNodes {
		line := m.renderNode(node, i == m.cursor)
		b.WriteString(line)
		b.WriteString("\n")
	}

	// Message
	if m.message != "" {
		b.WriteString("\n")
		if m.messageErr {
			b.WriteString(styles.ErrorMsg.Render(m.message))
		} else {
			b.WriteString(styles.Success.Render(m.message))
		}
	}

	// Help line
	b.WriteString("\n")
	b.WriteString(m.renderHelpLine())

	return styles.App.Render(b.String())
}

func (m *BrowserModel) renderNode(node *domain.TreeNode, selected bool) string {
	depth := node.Depth()
	indent := strings.Repeat("  ", depth)

	// Prefix (expand indicator)
	var prefix string
	if node.Type == domain.IDTypeItem {
		prefix = styles.TreeLeaf
	} else if node.IsExpanded {
		prefix = styles.TreeExpanded
	} else {
		prefix = styles.TreeCollapsed
	}

	// Format: "ID Description"
	text := fmt.Sprintf("%s %s", node.ID, node.Name)

	// Apply style based on type
	var style lipgloss.Style
	switch node.Type {
	case domain.IDTypeScope:
		scopeColor := styles.ScopeColor(node.ID)
		style = styles.NodeScope.Foreground(scopeColor)
	case domain.IDTypeArea:
		style = styles.NodeArea
	case domain.IDTypeCategory:
		if strings.HasSuffix(node.ID, "9") {
			style = styles.NodeArchive
		} else {
			style = styles.NodeCategory
		}
	case domain.IDTypeItem:
		style = styles.NodeItem
	}

	styledText := style.Render(text)

	if selected {
		styledText = styles.NodeSelected.Render(text)
	}

	return fmt.Sprintf("%s%s%s", indent, styles.TreeBranch.Render(prefix), styledText)
}

func (m *BrowserModel) renderHelpLine() string {
	keys := []struct {
		key  string
		desc string
	}{
		{"j/k", "navigate"},
		{"h/l", "collapse/expand"},
		{"n", "new"},
		{"a", "archive"},
		{"/", "search"},
		{"?", "help"},
		{"q", "quit"},
	}

	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s %s",
			styles.HelpKey.Render(k.key),
			styles.HelpDesc.Render(k.desc),
		))
	}

	return strings.Join(parts, styles.HelpSeparator.String())
}

// SetSize updates the view dimensions
func (m *BrowserModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Reload reloads the tree from disk
func (m *BrowserModel) Reload() tea.Cmd {
	m.root = nil
	m.flatNodes = nil
	m.cursor = 0
	return m.loadTree
}

// Messages for view switching
type SwitchToCreateMsg struct {
	ParentNode *domain.TreeNode
}

type SwitchToMoveMsg struct {
	SourceNode *domain.TreeNode
}

type SwitchToSearchMsg struct{}

type SwitchToHelpMsg struct{}

type SwitchToBrowserMsg struct{}
