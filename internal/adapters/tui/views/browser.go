package views

import (
	"fmt"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"libraio/internal/adapters/tui/styles"
	"libraio/internal/domain"
	"libraio/internal/ports"
)

// BrowserKeyMap defines key bindings for the browser view
type BrowserKeyMap struct {
	Up      key.Binding
	Down    key.Binding
	Enter   key.Binding
	Yank    key.Binding
	New     key.Binding
	Archive key.Binding
	Move    key.Binding
	Search  key.Binding
	Help    key.Binding
	Quit    key.Binding
}

var BrowserKeys = BrowserKeyMap{
	Up: key.NewBinding(
		key.WithKeys("k"),
		key.WithHelp("k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("j"),
		key.WithHelp("j", "down"),
	),
	Enter: key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("space", "open/toggle"),
	),
	Yank: key.NewBinding(
		key.WithKeys("y"),
		key.WithHelp("y", "copy ID"),
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

	// Search mode
	searchMode    bool
	searchInput   textinput.Model
	searchMatches []domain.SearchResult // matched results from repo
	searchIndex   int                   // current match index
}

// NewBrowserModel creates a new browser model
func NewBrowserModel(repo ports.VaultRepository) *BrowserModel {
	input := textinput.New()
	input.Placeholder = ""
	input.Prompt = "/"

	return &BrowserModel{
		repo:        repo,
		searchInput: input,
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

		// Search mode handling
		if m.searchMode {
			return m.updateSearchMode(msg)
		}

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

		case key.Matches(msg, BrowserKeys.Enter):
			if node := m.selectedNode(); node != nil {
				if node.Type == domain.IDTypeItem {
					// Open README in editor
					readmePath := node.Path + "/README.md"
					return m, func() tea.Msg {
						return OpenEditorMsg{Path: readmePath}
					}
				}
				// Toggle expand/collapse for non-items
				if !node.IsExpanded {
					node.Expand()
					return m, m.loadNodeChildren(node)
				} else {
					node.Collapse()
					m.refreshFlatNodes()
				}
			}
			return m, nil

		case key.Matches(msg, BrowserKeys.Yank):
			if node := m.selectedNode(); node != nil {
				clipboard.WriteAll(node.ID)
				m.message = fmt.Sprintf("Yanked: %s", node.ID)
				m.messageErr = false
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
			m.searchMode = true
			m.searchInput.SetValue("")
			m.searchInput.Focus()
			m.searchMatches = nil
			m.searchIndex = 0
			return m, textinput.Blink

		case key.Matches(msg, BrowserKeys.Help):
			return m, func() tea.Msg {
				return SwitchToHelpMsg{}
			}
		}
	}

	return m, nil
}

func (m *BrowserModel) updateSearchMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.searchMode = false
		m.searchMatches = nil
		m.searchInput.SetValue("")
		return m, nil

	case tea.KeyEnter:
		// Confirm selection and close search
		m.searchMode = false
		query := m.searchInput.Value()
		m.searchInput.SetValue("")
		if len(m.searchMatches) > 0 {
			m.message = fmt.Sprintf("/%s [%d/%d]", query, m.searchIndex+1, len(m.searchMatches))
			m.messageErr = false
		}
		return m, nil

	case tea.KeyCtrlN:
		// Next match
		if len(m.searchMatches) > 0 {
			m.searchIndex = (m.searchIndex + 1) % len(m.searchMatches)
			m.navigateToResult(m.searchMatches[m.searchIndex])
		}
		return m, nil

	case tea.KeyCtrlP:
		// Previous match
		if len(m.searchMatches) > 0 {
			m.searchIndex--
			if m.searchIndex < 0 {
				m.searchIndex = len(m.searchMatches) - 1
			}
			m.navigateToResult(m.searchMatches[m.searchIndex])
		}
		return m, nil
	}

	// Handle 'y' for yank in search mode
	if msg.String() == "y" && len(m.searchMatches) > 0 {
		result := m.searchMatches[m.searchIndex]
		clipboard.WriteAll(result.ID)
		m.message = fmt.Sprintf("Yanked: %s", result.ID)
		m.messageErr = false
		return m, nil
	}

	// Update input
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)

	// Search using repository (searches filesystem)
	query := m.searchInput.Value()
	if len(query) >= 2 {
		results, err := m.repo.Search(query)
		if err == nil {
			m.searchMatches = m.fuzzySort(results, query)
			if len(m.searchMatches) > 0 {
				m.searchIndex = 0
				m.navigateToResult(m.searchMatches[0])
			}
		}
	} else {
		m.searchMatches = nil
	}

	return m, cmd
}

// fuzzyScore calculates a score for how well target matches query
func fuzzyScore(target, query string) int {
	target = strings.ToLower(target)
	query = strings.ToLower(query)

	if len(query) == 0 {
		return 0
	}

	// Check for exact substring match first (highest priority)
	if strings.Contains(target, query) {
		score := 100
		// Bonus if it starts with query
		if strings.HasPrefix(target, query) {
			score += 50
		}
		return score
	}

	// Fuzzy match: check if chars appear in order
	score := 0
	queryIdx := 0
	prevMatchIdx := -1

	for i := 0; i < len(target) && queryIdx < len(query); i++ {
		if target[i] == query[queryIdx] {
			if prevMatchIdx == i-1 {
				score += 10 // consecutive
			}
			if i == 0 {
				score += 15 // start
			}
			if i > 0 && (target[i-1] == ' ' || target[i-1] == '.' || target[i-1] == '-') {
				score += 10 // after separator
			}
			score += 1
			prevMatchIdx = i
			queryIdx++
		}
	}

	if queryIdx == len(query) {
		return score
	}
	return 0
}

// fuzzySort sorts search results by relevance to query
func (m *BrowserModel) fuzzySort(results []domain.SearchResult, query string) []domain.SearchResult {
	type scored struct {
		result domain.SearchResult
		score  int
	}

	var scoredResults []scored
	for _, r := range results {
		s1 := fuzzyScore(r.ID, query)
		s2 := fuzzyScore(r.Name, query)
		best := s1
		if s2 > best {
			best = s2
		}
		if best > 0 {
			scoredResults = append(scoredResults, scored{result: r, score: best})
		}
	}

	// Sort by score descending
	for i := 0; i < len(scoredResults)-1; i++ {
		for j := i + 1; j < len(scoredResults); j++ {
			if scoredResults[j].score > scoredResults[i].score {
				scoredResults[i], scoredResults[j] = scoredResults[j], scoredResults[i]
			}
		}
	}

	sorted := make([]domain.SearchResult, len(scoredResults))
	for i, s := range scoredResults {
		sorted[i] = s.result
	}
	return sorted
}

// navigateToResult expands the tree path and navigates to a search result
func (m *BrowserModel) navigateToResult(result domain.SearchResult) {
	// Parse the ID to get parent IDs
	// e.g., "S01.11.15" -> ["S01", "S01.10-19", "S01.11", "S01.11.15"]
	parts := m.getIDPath(result.ID)

	// Expand each level
	current := m.root
	for _, partID := range parts {
		// Load children if needed
		if len(current.Children) == 0 {
			m.repo.LoadChildren(current)
		}
		current.Expand()

		// Find the child with this ID
		for _, child := range current.Children {
			if child.ID == partID {
				current = child
				break
			}
		}
	}

	// Refresh and find cursor position
	m.refreshFlatNodes()
	for i, node := range m.flatNodes {
		if node.ID == result.ID {
			m.cursor = i
			break
		}
	}
}

// getIDPath returns the hierarchy of IDs leading to the given ID
func (m *BrowserModel) getIDPath(id string) []string {
	var path []string
	idType := domain.ParseIDType(id)

	switch idType {
	case domain.IDTypeScope:
		path = []string{id}
	case domain.IDTypeArea:
		// Area: S01.10-19 -> scope is S01
		if len(id) >= 3 {
			path = []string{id[:3], id}
		}
	case domain.IDTypeCategory:
		// Category: S01.11 -> scope is S01, area is S01.10-19
		if len(id) >= 3 {
			scope := id[:3]
			// Derive area from category (e.g., S01.11 -> S01.10-19)
			if len(id) >= 6 {
				areaNum := id[4:5] // First digit of category
				area := scope + "." + areaNum + "0-" + areaNum + "9"
				path = []string{scope, area, id}
			}
		}
	case domain.IDTypeItem:
		// Item: S01.11.15 -> scope, area, category, item
		if len(id) >= 3 {
			scope := id[:3]
			if len(id) >= 6 {
				areaNum := id[4:5]
				area := scope + "." + areaNum + "0-" + areaNum + "9"
				category := id[:6]
				path = []string{scope, area, category, id}
			}
		}
	}

	return path
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
	b.WriteString(styles.Title.Render("Libraio"))
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

	// Search mode - vim-style command line at bottom
	if m.searchMode {
		b.WriteString("\n")
		b.WriteString(m.searchInput.View())
		if len(m.searchMatches) > 0 {
			b.WriteString(styles.MutedText.Render(fmt.Sprintf(" [%d/%d]", m.searchIndex+1, len(m.searchMatches))))
		} else if len(m.searchInput.Value()) >= 2 {
			b.WriteString(styles.ErrorMsg.Render(" [no match]"))
		}
	} else {
		// Help line
		b.WriteString("\n")
		b.WriteString(m.renderHelpLine())
	}

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

// keyHelp extracts the help text from a key.Binding
func keyHelp(b key.Binding) string {
	help := b.Help()
	return fmt.Sprintf("%s %s",
		styles.HelpKey.Render(help.Key),
		styles.HelpDesc.Render(help.Desc),
	)
}

func (m *BrowserModel) renderHelpLine() string {
	node := m.selectedNode()

	// Common keys for all contexts
	var bindings []key.Binding

	// Navigation is always available
	bindings = append(bindings,
		key.NewBinding(key.WithKeys("j/k"), key.WithHelp("j/k", "navigate")),
	)

	// Context-specific bindings based on node type
	if node != nil {
		switch node.Type {
		case domain.IDTypeItem:
			// Items: open README, yank ID, archive, move
			bindings = append(bindings,
				key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "open")),
				BrowserKeys.Yank,
				BrowserKeys.Archive,
				BrowserKeys.Move,
			)
		case domain.IDTypeCategory:
			// Categories: toggle, new item, archive
			bindings = append(bindings,
				key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "toggle")),
				key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new item")),
				BrowserKeys.Archive,
			)
		case domain.IDTypeArea:
			// Areas: toggle, new category
			bindings = append(bindings,
				key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "toggle")),
				key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new category")),
			)
		case domain.IDTypeScope:
			// Scopes: toggle only
			bindings = append(bindings,
				key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "toggle")),
			)
		}
	} else {
		// Fallback when no node selected
		bindings = append(bindings,
			key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "toggle")),
		)
	}

	// Always show search, help, quit
	bindings = append(bindings,
		BrowserKeys.Search,
		BrowserKeys.Help,
		BrowserKeys.Quit,
	)

	var parts []string
	for _, b := range bindings {
		parts = append(parts, keyHelp(b))
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
