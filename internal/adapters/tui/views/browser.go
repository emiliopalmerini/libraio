package views

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"libraio/internal/adapters/tui/styles"
	"libraio/internal/application"
	"libraio/internal/application/commands"
	"libraio/internal/domain"
	"libraio/internal/ports"
)

// BrowserKeyMap defines key bindings for the browser view
type BrowserKeyMap struct {
	Up           key.Binding
	Down         key.Binding
	Enter        key.Binding
	Obsidian     key.Binding
	Yank         key.Binding
	New          key.Binding
	Move         key.Binding
	Archive      key.Binding
	Delete       key.Binding
	SmartCatalog key.Binding
	Search       key.Binding
	Help         key.Binding
	Quit         key.Binding
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
	Obsidian: key.NewBinding(
		key.WithKeys("o"),
		key.WithHelp("o", "obsidian"),
	),
	Yank: key.NewBinding(
		key.WithKeys("y"),
		key.WithHelp("y", "copy ID"),
	),
	New: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "new"),
	),
	Move: key.NewBinding(
		key.WithKeys("m"),
		key.WithHelp("m", "move"),
	),
	Archive: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "archive"),
	),
	Delete: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "delete"),
	),
	SmartCatalog: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "smart catalog"),
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
	ViewState
	repo      ports.VaultRepository
	root      *application.TreeNode
	flatNodes []*application.TreeNode
	cursor    int
	viewport  int // viewport offset (first visible line)

	// Search mode
	searchMode    bool
	searchInput   textinput.Model
	searchMatches []application.SearchResult // matched results from repo
	searchIndex   int                        // current match index

	// For restoring state after reload
	restoreCursor int
	expandedIDs   map[string]bool
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
	root *application.TreeNode
}

type errMsg struct {
	err error
}

type childrenLoadedMsg struct {
	node *application.TreeNode
}

type successMsg struct {
	message string
}

// Update handles messages for the browser
func (m *BrowserModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case treeLoadedMsg:
		m.root = msg.root
		m.refreshFlatNodes()

		// Restore expansion state and load children for expanded nodes
		if len(m.expandedIDs) > 0 {
			m.restoreExpansion(m.root)
			m.expandedIDs = nil
			m.refreshFlatNodes()
		}

		// Restore cursor position (clamped to valid range)
		if m.restoreCursor >= 0 {
			if m.restoreCursor < len(m.flatNodes) {
				m.cursor = m.restoreCursor
			} else if len(m.flatNodes) > 0 {
				m.cursor = len(m.flatNodes) - 1
			}
			m.restoreCursor = -1
		}

		return m, nil

	case childrenLoadedMsg:
		m.refreshFlatNodes()
		return m, nil

	case errMsg:
		m.Message = msg.err.Error()
		m.MessageErr = true
		return m, nil

	case successMsg:
		m.Message = msg.message
		m.MessageErr = false
		return m, m.Reload()

	case tea.KeyMsg:
		m.Message = "" // Clear message on key press

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
				m.ensureCursorVisible()
			}
			return m, nil

		case key.Matches(msg, BrowserKeys.Down):
			if m.cursor < len(m.flatNodes)-1 {
				m.cursor++
				m.ensureCursorVisible()
			}
			return m, nil

		case key.Matches(msg, BrowserKeys.Enter):
			if node := m.selectedNode(); node != nil {
				if node.Type == application.IDTypeItem {
					// Open JDex file in editor
					jdexPath := m.getJDexPath(node)
					return m, func() tea.Msg {
						return OpenEditorMsg{Path: jdexPath}
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

		case key.Matches(msg, BrowserKeys.Obsidian):
			if node := m.selectedNode(); node != nil {
				if node.Type == application.IDTypeItem {
					jdexPath := m.getJDexPath(node)
					return m, func() tea.Msg {
						return OpenObsidianMsg{Path: jdexPath}
					}
				}
			}
			return m, nil

		case key.Matches(msg, BrowserKeys.Yank):
			if node := m.selectedNode(); node != nil {
				clipboard.WriteAll(node.ID)
				m.Message = fmt.Sprintf("Yanked: %s", node.ID)
				m.MessageErr = false
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

		case key.Matches(msg, BrowserKeys.Move):
			// Return command to switch to move view (only for items and categories)
			if node := m.selectedNode(); node != nil {
				if node.Type == application.IDTypeItem || node.Type == application.IDTypeCategory {
					return m, func() tea.Msg {
						return SwitchToMoveMsg{SourceNode: node}
					}
				}
			}
			return m, nil

		case key.Matches(msg, BrowserKeys.Archive):
			// Return command to switch to archive view (only for items and non-archive categories)
			if node := m.selectedNode(); node != nil {
				eligibility := commands.CheckArchiveEligibility(node.ID, node.Type)
				if eligibility.CanArchive {
					return m, func() tea.Msg {
						return SwitchToArchiveMsg{TargetNode: node}
					}
				}
				m.Message = eligibility.Reason
				m.MessageErr = true
			}
			return m, nil

		case key.Matches(msg, BrowserKeys.Delete):
			// Return command to switch to delete view
			if node := m.selectedNode(); node != nil {
				return m, func() tea.Msg {
					return SwitchToDeleteMsg{TargetNode: node}
				}
			}
			return m, nil

		case key.Matches(msg, BrowserKeys.SmartCatalog):
			// Return command to switch to smart catalog view (only for inbox items)
			if node := m.selectedNode(); node != nil {
				if node.Type == application.IDTypeItem {
					// Only allow on inbox items (.01)
					if domain.IsInboxItem(node.ID) {
						return m, func() tea.Msg {
							return SwitchToSmartCatalogMsg{SourceNode: node}
						}
					}
					m.Message = "Smart catalog only works on inbox items"
					m.MessageErr = true
					return m, nil
				}
				m.Message = "Smart catalog only works on inbox items"
				m.MessageErr = true
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
			m.Message = fmt.Sprintf("/%s [%d/%d]", query, m.searchIndex+1, len(m.searchMatches))
			m.MessageErr = false
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
		m.Message = fmt.Sprintf("Yanked: %s", result.ID)
		m.MessageErr = false
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
func (m *BrowserModel) fuzzySort(results []application.SearchResult, query string) []application.SearchResult {
	type scored struct {
		result application.SearchResult
		score  int
	}

	var scoredResults []scored
	for _, r := range results {
		s1 := fuzzyScore(r.ID, query)
		s2 := fuzzyScore(r.Name, query)
		s3 := fuzzyScore(r.MatchedText, query)
		best := s1
		if s2 > best {
			best = s2
		}
		if s3 > best {
			best = s3
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

	sorted := make([]application.SearchResult, len(scoredResults))
	for i, s := range scoredResults {
		sorted[i] = s.result
	}
	return sorted
}

// navigateToResult expands the tree path and navigates to a search result
func (m *BrowserModel) navigateToResult(result application.SearchResult) {
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
			m.ensureCursorVisible()
			break
		}
	}
}

// getIDPath returns the hierarchy of IDs leading to the given ID
func (m *BrowserModel) getIDPath(id string) []string {
	return application.GetIDHierarchy(id)
}

func (m *BrowserModel) loadNodeChildren(node *application.TreeNode) tea.Cmd {
	return func() tea.Msg {
		if err := m.repo.LoadChildren(node); err != nil {
			return errMsg{err}
		}
		return childrenLoadedMsg{node}
	}
}

// restoreExpansion recursively restores expansion state and loads children
func (m *BrowserModel) restoreExpansion(node *application.TreeNode) {
	if m.expandedIDs[node.ID] {
		node.Expand()
		m.repo.LoadChildren(node)
	}

	for _, child := range node.Children {
		m.restoreExpansion(child)
	}
}

func (m *BrowserModel) selectedNode() *application.TreeNode {
	if m.cursor >= 0 && m.cursor < len(m.flatNodes) {
		return m.flatNodes[m.cursor]
	}
	return nil
}

// getJDexPath returns the JDex file path for a node, with fallback to legacy README.md
func (m *BrowserModel) getJDexPath(node *application.TreeNode) string {
	folderName := filepath.Base(node.Path)
	jdexPath := filepath.Join(node.Path, domain.JDexFileName(folderName))

	// Check if new-style JDex file exists
	if _, err := os.Stat(jdexPath); err == nil {
		return jdexPath
	}

	// Fallback to legacy README.md for backwards compatibility
	legacyPath := filepath.Join(node.Path, "README.md")
	if _, err := os.Stat(legacyPath); err == nil {
		return legacyPath
	}

	// Return new-style path as default (for new items)
	return jdexPath
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
	m.ensureCursorVisible()
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

	// Tree (only render visible portion)
	viewHeight := m.treeViewHeight()
	endIdx := m.viewport + viewHeight
	if endIdx > len(m.flatNodes) {
		endIdx = len(m.flatNodes)
	}

	for i := m.viewport; i < endIdx; i++ {
		node := m.flatNodes[i]
		line := m.renderNode(node, i == m.cursor)
		b.WriteString(line)
		b.WriteString("\n")
	}

	// Message
	if m.Message != "" {
		b.WriteString("\n")
		if m.MessageErr {
			b.WriteString(styles.ErrorMsg.Render(m.Message))
		} else {
			b.WriteString(styles.Success.Render(m.Message))
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

func (m *BrowserModel) renderNode(node *application.TreeNode, selected bool) string {
	depth := node.Depth()
	indent := strings.Repeat("  ", depth)

	// Prefix (expand indicator)
	var prefix string
	if node.Type == application.IDTypeItem {
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
	case application.IDTypeScope:
		scopeColor := styles.ScopeColor(node.ID)
		style = styles.NodeScope.Foreground(scopeColor)
	case application.IDTypeArea:
		style = styles.NodeArea
	case application.IDTypeCategory:
		if strings.HasSuffix(node.ID, "9") {
			style = styles.NodeArchive
		} else {
			style = styles.NodeCategory
		}
	case application.IDTypeItem:
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
		case application.IDTypeItem:
			// Items: open README, open in obsidian, yank ID, move, archive, delete
			bindings = append(bindings,
				key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "open")),
				BrowserKeys.Obsidian,
				BrowserKeys.Yank,
				BrowserKeys.Move,
				BrowserKeys.Archive,
				BrowserKeys.Delete,
			)
		case application.IDTypeCategory:
			// Categories: toggle, new item, move, archive, delete
			bindings = append(bindings,
				key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "toggle")),
				key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new item")),
				BrowserKeys.Move,
				BrowserKeys.Archive,
				BrowserKeys.Delete,
			)
		case application.IDTypeArea:
			// Areas: toggle, new category, delete
			bindings = append(bindings,
				key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "toggle")),
				key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new category")),
				BrowserKeys.Delete,
			)
		case application.IDTypeScope:
			// Scopes: toggle, delete
			bindings = append(bindings,
				key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "toggle")),
				BrowserKeys.Delete,
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
	m.Width = width
	m.Height = height
	m.ensureCursorVisible()
}


// treeViewHeight returns the number of lines available for the tree view
func (m *BrowserModel) treeViewHeight() int {
	// Subtract: title (1) + subtitle (1) + blank line (1) + footer area (3)
	available := m.Height - 6
	if available < 1 {
		return 1
	}
	return available
}

// ensureCursorVisible adjusts the viewport to keep the cursor visible
func (m *BrowserModel) ensureCursorVisible() {
	viewHeight := m.treeViewHeight()

	// Cursor above viewport
	if m.cursor < m.viewport {
		m.viewport = m.cursor
	}

	// Cursor below viewport
	if m.cursor >= m.viewport+viewHeight {
		m.viewport = m.cursor - viewHeight + 1
	}

	// Clamp viewport
	maxViewport := len(m.flatNodes) - viewHeight
	if maxViewport < 0 {
		maxViewport = 0
	}
	if m.viewport > maxViewport {
		m.viewport = maxViewport
	}
	if m.viewport < 0 {
		m.viewport = 0
	}
}

// Reload reloads the tree from disk while preserving cursor position and expansion state
func (m *BrowserModel) Reload() tea.Cmd {
	// Save current cursor position to restore after reload
	m.restoreCursor = m.cursor
	// Save expanded nodes
	m.expandedIDs = make(map[string]bool)
	for _, node := range m.flatNodes {
		if node.IsExpanded {
			m.expandedIDs[node.ID] = true
		}
	}
	m.root = nil
	m.flatNodes = nil
	m.cursor = 0
	return m.loadTree
}

// Messages for view switching
type SwitchToCreateMsg struct {
	ParentNode *application.TreeNode
}

type SwitchToMoveMsg struct {
	SourceNode *application.TreeNode
}

type SwitchToDeleteMsg struct {
	TargetNode *application.TreeNode
}

type SwitchToSearchMsg struct{}

type SwitchToHelpMsg struct{}

type SwitchToBrowserMsg struct{}

// OpenObsidianMsg requests opening a file in Obsidian
type OpenObsidianMsg struct {
	Path string
}
