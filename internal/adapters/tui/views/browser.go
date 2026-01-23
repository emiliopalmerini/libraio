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
	Left         key.Binding
	Right        key.Binding
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
		key.WithKeys("k", "up"),
		key.WithHelp("k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("j", "down"),
		key.WithHelp("j", "down"),
	),
	Left: key.NewBinding(
		key.WithKeys("h", "left"),
		key.WithHelp("h", "collapse"),
	),
	Right: key.NewBinding(
		key.WithKeys("l", "right"),
		key.WithHelp("l", "expand"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter", " "),
		key.WithHelp("enter/space", "open/toggle"),
	),
	Obsidian: key.NewBinding(
		key.WithKeys("o"),
		key.WithHelp("o", "obsidian"),
	),
	Yank: key.NewBinding(
		key.WithKeys("y"),
		key.WithHelp("y", "copy wikilink"),
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
		key.WithKeys("esc", "q", "ctrl+c"),
		key.WithHelp("esc/q", "quit"),
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
	searchScorer  *SearchScorer              // fuzzy search scorer

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
		repo:         repo,
		searchInput:  input,
		searchScorer: NewSearchScorer(),
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

		case key.Matches(msg, BrowserKeys.Left):
			return m.handleCollapse()

		case key.Matches(msg, BrowserKeys.Right):
			return m.handleExpand()

		case key.Matches(msg, BrowserKeys.Enter):
			return m.handleEnter()

		case key.Matches(msg, BrowserKeys.Obsidian):
			if node := m.selectedNode(); node != nil {
				var pathToOpen string

				switch node.Type {
				case application.IDTypeFile:
					// Check if Obsidian can open this file type
					if canObsidianOpen(node.Path) {
						pathToOpen = node.Path
					} else if node.Parent != nil {
						// Fall back to parent item's JDex
						pathToOpen = m.getJDexPath(node.Parent)
					}
				case application.IDTypeItem:
					pathToOpen = m.getJDexPath(node)
				}

				if pathToOpen != "" {
					return m, func() tea.Msg {
						return OpenObsidianMsg{Path: pathToOpen}
					}
				}
			}
			return m, nil

		case key.Matches(msg, BrowserKeys.Yank):
			if node := m.selectedNode(); node != nil {
				var wikilink string
				if node.Type == application.IDTypeFile {
					// For files: [[filename]]
					wikilink = fmt.Sprintf("[[%s]]", node.Name)
				} else if node.ID != "" {
					// For JD entities: [[ID Name]]
					wikilink = fmt.Sprintf("[[%s %s]]", node.ID, node.Name)
				}
				if wikilink != "" {
					clipboard.WriteAll(wikilink)
					m.Message = fmt.Sprintf("Yanked: %s", wikilink)
					m.MessageErr = false
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
			return m.handleArchive()

		case key.Matches(msg, BrowserKeys.Delete):
			// Return command to switch to delete view
			if node := m.selectedNode(); node != nil {
				return m, func() tea.Msg {
					return SwitchToDeleteMsg{TargetNode: node}
				}
			}
			return m, nil

		case key.Matches(msg, BrowserKeys.SmartCatalog):
			return m.handleSmartCatalog()

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

// handleEnter handles the enter/space key to open files or toggle expansion
func (m *BrowserModel) handleEnter() (tea.Model, tea.Cmd) {
	node := m.selectedNode()
	if node == nil {
		return m, nil
	}

	// Files open in editor
	if node.Type == application.IDTypeFile {
		return m, func() tea.Msg {
			return OpenEditorMsg{Path: node.Path}
		}
	}

	// Toggle expand/collapse for all other types (including items)
	if !node.IsExpanded {
		node.Expand()
		return m, m.loadNodeChildren(node)
	}
	node.Collapse()
	m.refreshFlatNodes()
	return m, nil
}

// handleExpand handles the l/right key to expand a node
func (m *BrowserModel) handleExpand() (tea.Model, tea.Cmd) {
	node := m.selectedNode()
	if node == nil {
		return m, nil
	}

	// Files can't be expanded
	if node.Type == application.IDTypeFile {
		return m, nil
	}

	// Expand if not already expanded
	if !node.IsExpanded {
		node.Expand()
		return m, m.loadNodeChildren(node)
	}
	return m, nil
}

// handleCollapse handles the h/left key to collapse a node or go to parent
func (m *BrowserModel) handleCollapse() (tea.Model, tea.Cmd) {
	node := m.selectedNode()
	if node == nil {
		return m, nil
	}

	// If expanded, collapse it
	if node.IsExpanded {
		node.Collapse()
		m.refreshFlatNodes()
		return m, nil
	}

	// If not expanded, go to parent
	if node.Parent != nil && node.Parent.Parent != nil { // Skip root
		for i, n := range m.flatNodes {
			if n == node.Parent {
				m.cursor = i
				m.ensureCursorVisible()
				break
			}
		}
	}
	return m, nil
}

// handleArchive handles the archive key
func (m *BrowserModel) handleArchive() (tea.Model, tea.Cmd) {
	node := m.selectedNode()
	if node == nil {
		return m, nil
	}

	eligibility := commands.CheckArchiveEligibility(node.ID, node.Type)
	if eligibility.CanArchive {
		return m, func() tea.Msg {
			return SwitchToArchiveMsg{TargetNode: node}
		}
	}
	m.Message = eligibility.Reason
	m.MessageErr = true
	return m, nil
}

// handleSmartCatalog handles the smart catalog key
func (m *BrowserModel) handleSmartCatalog() (tea.Model, tea.Cmd) {
	node := m.selectedNode()
	if node == nil {
		return m, nil
	}

	if node.Type != application.IDTypeItem {
		m.Message = "Smart catalog only works on inbox items"
		m.MessageErr = true
		return m, nil
	}

	if !domain.IsInboxItem(node.ID) {
		m.Message = "Smart catalog only works on inbox items"
		m.MessageErr = true
		return m, nil
	}

	return m, func() tea.Msg {
		return SwitchToSmartCatalogMsg{SourceNode: node}
	}
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
		var wikilink string
		if result.Type == application.IDTypeFile {
			// For files: [[filename]]
			wikilink = fmt.Sprintf("[[%s]]", result.Name)
		} else {
			// For JD entities: [[ID Name]] - MatchedText already has "ID Name" format
			wikilink = fmt.Sprintf("[[%s]]", result.MatchedText)
		}
		clipboard.WriteAll(wikilink)
		m.Message = fmt.Sprintf("Yanked: %s", wikilink)
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
			m.searchMatches = m.searchScorer.SortResults(results, query)
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

// navigateToResult expands the tree path and navigates to a search result (file)
func (m *BrowserModel) navigateToResult(result application.SearchResult) {
	if result.ID == "" {
		return // File has no parent item ID
	}

	// Parse the ID to get parent IDs
	// e.g., "S01.11.15" -> ["S01", "S01.10-19", "S01.11", "S01.11.15"]
	parts := m.getIDPath(result.ID)

	// Expand each level to reach the item
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

	// Now current is the item - expand it to show files
	if len(current.Children) == 0 {
		m.repo.LoadChildren(current)
	}
	current.Expand()

	// Refresh and find cursor position for the file
	m.refreshFlatNodes()
	for i, node := range m.flatNodes {
		if node.Path == result.Path {
			m.cursor = i
			m.ensureCursorVisible()
			return
		}
	}

	// Fallback: select the item if file not found
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

// canObsidianOpen checks if Obsidian can natively open a file type
func canObsidianOpen(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	// Markdown
	case ".md":
		return true
	// Images
	case ".png", ".jpg", ".jpeg", ".gif", ".bmp", ".svg", ".webp":
		return true
	// PDF
	case ".pdf":
		return true
	// Audio
	case ".mp3", ".wav", ".m4a", ".ogg", ".3gp", ".flac", ".webm":
		return true
	// Video
	case ".mp4", ".ogv", ".mov", ".mkv":
		return true
	// Canvas
	case ".canvas":
		return true
	default:
		return false
	}
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
	endIdx := min(m.viewport+viewHeight, len(m.flatNodes))

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
	if node.Type == application.IDTypeFile {
		prefix = styles.TreeLeaf
	} else if node.IsExpanded {
		prefix = styles.TreeExpanded
	} else {
		prefix = styles.TreeCollapsed
	}

	// Format text based on type
	var text string
	if node.Type == application.IDTypeFile {
		text = node.Name // Files just show filename
	} else {
		text = fmt.Sprintf("%s %s", node.ID, node.Name) // "ID Description"
	}

	// Map application IDType to styles NodeType
	nodeType := mapToNodeType(node)

	// Get style from NodeStyler
	style := styles.DefaultNodeStyler.GetStyle(nodeType, node.ID)
	styledText := style.Render(text)

	if selected {
		styledText = styles.NodeSelected.Render(text)
	}

	return fmt.Sprintf("%s%s%s", indent, styles.TreeBranch.Render(prefix), styledText)
}

// mapToNodeType converts application.IDType to styles.NodeType
func mapToNodeType(node *application.TreeNode) styles.NodeType {
	switch node.Type {
	case application.IDTypeScope:
		return styles.NodeTypeScope
	case application.IDTypeArea:
		return styles.NodeTypeArea
	case application.IDTypeCategory:
		if strings.HasSuffix(node.ID, "9") {
			return styles.NodeTypeCategoryArchive
		}
		return styles.NodeTypeCategory
	case application.IDTypeItem:
		return styles.NodeTypeItem
	case application.IDTypeFile:
		return styles.NodeTypeFile
	default:
		return styles.NodeTypeUnknown
	}
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

	// Context-specific bindings based on node type
	var bindings []key.Binding

	if node != nil {
		// Check smart catalog eligibility (only inbox items)
		canCatalog := node.Type == application.IDTypeItem && domain.IsInboxItem(node.ID)

		switch node.Type {
		case application.IDTypeItem:
			if canCatalog {
				bindings = append(bindings, BrowserKeys.SmartCatalog)
			}
		case application.IDTypeCategory:
			bindings = append(bindings,
				key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new item")),
			)
		case application.IDTypeArea:
			bindings = append(bindings,
				key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new category")),
			)
		case application.IDTypeScope:
			bindings = append(bindings,
				key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new area")),
			)
		}
	}

	// Always show search and help
	bindings = append(bindings,
		BrowserKeys.Search,
		BrowserKeys.Help,
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
	maxViewport := max(len(m.flatNodes)-viewHeight, 0)
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
