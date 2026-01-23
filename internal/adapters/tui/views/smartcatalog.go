package views

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"libraio/internal/adapters/tui/styles"
	"libraio/internal/application"
	"libraio/internal/domain"
	"libraio/internal/ports"
)

// SmartCatalogState represents the state of the smart catalog view
type SmartCatalogState int

const (
	SmartCatalogLoading SmartCatalogState = iota
	SmartCatalogShowList
	SmartCatalogError
)

// SmartCatalogKeyMap defines key bindings for the smart catalog view
type SmartCatalogKeyMap struct {
	Up            key.Binding
	Down          key.Binding
	Confirm       key.Binding
	Skip          key.Binding
	Cancel        key.Binding
	NextPage      key.Binding
	PrevPage      key.Binding
	ReviewSkipped key.Binding
}

var SmartCatalogKeys = SmartCatalogKeyMap{
	Up: key.NewBinding(
		key.WithKeys("k", "up"),
		key.WithHelp("k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("j", "down"),
		key.WithHelp("j", "down"),
	),
	Confirm: key.NewBinding(
		key.WithKeys("y"),
		key.WithHelp("y", "accept"),
	),
	Skip: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "skip"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
	),
	NextPage: key.NewBinding(
		key.WithKeys("ctrl+f", "pgdown"),
		key.WithHelp("ctrl+f", "next page"),
	),
	PrevPage: key.NewBinding(
		key.WithKeys("ctrl+b", "pgup"),
		key.WithHelp("ctrl+b", "prev page"),
	),
	ReviewSkipped: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "review skipped"),
	),
}

// FileSuggestion pairs a file with its suggestion
type FileSuggestion struct {
	File       ports.FileInfo
	Suggestion *ports.CatalogSuggestion
}

// SmartCatalogModel is the model for the smart catalog view
type SmartCatalogModel struct {
	ViewState
	repo        ports.VaultRepository
	assistant   ports.AIAssistant
	inboxNode   *application.TreeNode // The inbox item node
	suggestions []FileSuggestion      // All suggestions from Claude
	cursor      int                   // Currently selected suggestion (absolute index)
	moved       int                   // Count of files moved
	state       SmartCatalogState
	err         error
	spinner     spinner.Model

	// Pagination
	pageSize   int // Number of items per page (default 10)
	pageOffset int // Current page start index

	// Skipped items for retry
	skipped    []FileSuggestion // Skipped files
	reviewMode bool             // True when reviewing skipped items
}

// NewSmartCatalogModel creates a new smart catalog view model
func NewSmartCatalogModel(repo ports.VaultRepository, assistant ports.AIAssistant) *SmartCatalogModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = styles.Spinner

	return &SmartCatalogModel{
		repo:      repo,
		assistant: assistant,
		spinner:   s,
		state:     SmartCatalogLoading,
		pageSize:  10,
	}
}

// SetSource sets the inbox node for smart cataloging
func (m *SmartCatalogModel) SetSource(node *application.TreeNode) {
	m.inboxNode = node
	m.suggestions = nil
	m.cursor = 0
	m.moved = 0
	m.err = nil
	m.state = SmartCatalogLoading
	m.pageOffset = 0
	m.skipped = nil
	m.reviewMode = false
}

// Init initializes the smart catalog view
func (m *SmartCatalogModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.fetchSuggestions(),
	)
}

func (m *SmartCatalogModel) fetchSuggestions() tea.Cmd {
	return func() tea.Msg {
		if m.inboxNode == nil {
			return SmartCatalogFetchErrMsg{Err: fmt.Errorf("no inbox selected")}
		}

		if m.assistant == nil {
			return SmartCatalogFetchErrMsg{Err: fmt.Errorf("AI assistant not available")}
		}

		// Get the inbox folder path
		inboxPath, err := m.repo.GetPath(m.inboxNode.ID)
		if err != nil {
			return SmartCatalogFetchErrMsg{Err: fmt.Errorf("failed to get inbox path: %w", err)}
		}

		// List files in the inbox folder
		entries, err := os.ReadDir(inboxPath)
		if err != nil {
			return SmartCatalogFetchErrMsg{Err: fmt.Errorf("failed to read inbox folder: %w", err)}
		}

		// Get the JDex filename to exclude it
		jdexName := domain.JDexFileName(filepath.Base(inboxPath))

		var files []ports.FileInfo
		for _, entry := range entries {
			// Skip directories and the JDex file
			if entry.IsDir() {
				continue
			}
			if entry.Name() == jdexName || entry.Name() == "README.md" {
				continue
			}
			// Skip hidden files
			if strings.HasPrefix(entry.Name(), ".") {
				continue
			}

			filePath := filepath.Join(inboxPath, entry.Name())

			// Read file content if it's text
			fileContent := ""
			if content, err := os.ReadFile(filePath); err == nil {
				if isTextContent(content) {
					fileContent = string(content)
					// Limit content size
					if len(fileContent) > 2000 {
						fileContent = fileContent[:2000] + "\n...(truncated)"
					}
				}
			}

			files = append(files, ports.FileInfo{
				Name:    entry.Name(),
				Path:    filePath,
				Content: fileContent,
			})
		}

		if len(files) == 0 {
			return SmartCatalogFetchErrMsg{Err: fmt.Errorf("no files found in inbox")}
		}

		// Build vault structure context based on inbox level
		parentID, level := domain.GetInboxParentID(m.inboxNode.ID)
		vaultStructure, err := buildVaultContextForLevel(m.repo, parentID, level)
		if err != nil {
			return SmartCatalogFetchErrMsg{Err: fmt.Errorf("failed to build vault context: %w", err)}
		}

		// Get suggestions from AI
		suggestions, err := m.assistant.SuggestCatalogDestinations(files, vaultStructure)
		if err != nil {
			return SmartCatalogFetchErrMsg{Err: err}
		}

		// Match suggestions to files
		fileSuggestions := matchSuggestionsToFiles(files, suggestions)

		return SmartCatalogSuggestionsMsg{Suggestions: fileSuggestions}
	}
}

// matchSuggestionsToFiles pairs files with their suggestions
func matchSuggestionsToFiles(files []ports.FileInfo, suggestions []ports.CatalogSuggestion) []FileSuggestion {
	suggestionMap := make(map[string]*ports.CatalogSuggestion)
	for i := range suggestions {
		suggestionMap[suggestions[i].FileName] = &suggestions[i]
	}

	var result []FileSuggestion
	for _, f := range files {
		fs := FileSuggestion{File: f}
		if s, ok := suggestionMap[f.Name]; ok {
			fs.Suggestion = s
		}
		result = append(result, fs)
	}
	return result
}

// isTextContent checks if content appears to be text (not binary)
func isTextContent(content []byte) bool {
	if len(content) == 0 {
		return true
	}
	// Check first 512 bytes for null bytes (common in binary files)
	checkLen := min(len(content), 512)
	for i := range checkLen {
		if content[i] == 0 {
			return false
		}
	}
	return true
}

// buildVaultContextForLevel creates vault structure context filtered by inbox level
func buildVaultContextForLevel(repo ports.VaultRepository, parentID string, level domain.InboxLevel) (string, error) {
	var b strings.Builder

	switch level {
	case domain.InboxLevelCategory:
		// Only items in this category
		items, err := repo.ListItems(parentID)
		if err != nil {
			return "", err
		}
		for _, item := range items {
			if num, _ := domain.ExtractNumber(item.ID); num <= domain.StandardZeroMax {
				continue
			}
			fmt.Fprintf(&b, "- %s %s\n", item.ID, item.Name)
		}

	case domain.InboxLevelArea:
		// All categories and items in this area
		categories, err := repo.ListCategories(parentID)
		if err != nil {
			return "", err
		}
		for _, cat := range categories {
			if domain.IsAreaManagementCategory(cat.ID) {
				continue
			}
			fmt.Fprintf(&b, "\n%s %s\n", cat.ID, cat.Name)
			items, _ := repo.ListItems(cat.ID)
			for _, item := range items {
				if num, _ := domain.ExtractNumber(item.ID); num <= domain.StandardZeroMax {
					continue
				}
				fmt.Fprintf(&b, "  - %s %s\n", item.ID, item.Name)
			}
		}

	case domain.InboxLevelScope:
		// All areas, categories, items in this scope
		areas, err := repo.ListAreas(parentID)
		if err != nil {
			return "", err
		}
		for _, area := range areas {
			categories, _ := repo.ListCategories(area.ID)
			for _, cat := range categories {
				if domain.IsAreaManagementCategory(cat.ID) {
					continue
				}
				fmt.Fprintf(&b, "\n%s %s\n", cat.ID, cat.Name)
				items, _ := repo.ListItems(cat.ID)
				count := 0
				for _, item := range items {
					if num, _ := domain.ExtractNumber(item.ID); num <= domain.StandardZeroMax {
						continue
					}
					if count >= 5 {
						b.WriteString("  ...\n")
						break
					}
					fmt.Fprintf(&b, "  - %s %s\n", item.ID, item.Name)
					count++
				}
			}
		}
	}

	return b.String(), nil
}

// Update handles messages for the smart catalog view
func (m *SmartCatalogModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case spinner.TickMsg:
		if m.state == SmartCatalogLoading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case SmartCatalogSuggestionsMsg:
		m.suggestions = msg.Suggestions
		if len(m.suggestions) == 0 {
			m.err = fmt.Errorf("no suggestions received")
			m.state = SmartCatalogError
		} else {
			m.state = SmartCatalogShowList
		}
		return m, nil

	case SmartCatalogFetchErrMsg:
		m.err = msg.Err
		m.state = SmartCatalogError
		return m, nil

	case tea.KeyMsg:
		switch m.state {
		case SmartCatalogShowList:
			switch {
			case key.Matches(msg, SmartCatalogKeys.Cancel):
				return m, func() tea.Msg {
					return SmartCatalogDoneMsg{Moved: m.moved}
				}
			case key.Matches(msg, SmartCatalogKeys.Up):
				if m.cursor > 0 {
					m.cursor--
					m.ensureCursorInPage()
				}
				return m, nil
			case key.Matches(msg, SmartCatalogKeys.Down):
				if m.cursor < len(m.suggestions)-1 {
					m.cursor++
					m.ensureCursorInPage()
				}
				return m, nil
			case key.Matches(msg, SmartCatalogKeys.NextPage):
				m.nextPage()
				return m, nil
			case key.Matches(msg, SmartCatalogKeys.PrevPage):
				m.prevPage()
				return m, nil
			case key.Matches(msg, SmartCatalogKeys.Confirm):
				if len(m.suggestions) > 0 && m.cursor < len(m.suggestions) {
					return m, m.moveCurrentFile()
				}
				return m, nil
			case key.Matches(msg, SmartCatalogKeys.Skip):
				if len(m.suggestions) == 0 {
					return m, nil
				}
				// Move current item to skipped list
				if m.cursor < len(m.suggestions) {
					m.skipped = append(m.skipped, m.suggestions[m.cursor])
					m.suggestions = append(m.suggestions[:m.cursor], m.suggestions[m.cursor+1:]...)
					// Adjust cursor if needed
					if m.cursor >= len(m.suggestions) && m.cursor > 0 {
						m.cursor = len(m.suggestions) - 1
					}
					m.ensureCursorInPage()
					// Check if we're done with main list
					if len(m.suggestions) == 0 {
						if len(m.skipped) > 0 {
							// Prompt to review skipped
							m.Message = fmt.Sprintf("All files processed. %d skipped - press 'r' to review", len(m.skipped))
						} else {
							return m, func() tea.Msg {
								return SmartCatalogDoneMsg{Moved: m.moved}
							}
						}
					}
				}
				return m, nil
			case key.Matches(msg, SmartCatalogKeys.ReviewSkipped):
				if len(m.skipped) > 0 && !m.reviewMode {
					// Switch to review mode
					m.reviewMode = true
					m.suggestions = m.skipped
					m.skipped = nil
					m.cursor = 0
					m.pageOffset = 0
					m.Message = ""
				}
				return m, nil
			}

		case SmartCatalogError:
			// Any key returns to browser on error
			return m, func() tea.Msg {
				return SwitchToBrowserMsg{}
			}

		case SmartCatalogLoading:
			// Allow canceling during loading
			if key.Matches(msg, SmartCatalogKeys.Cancel) {
				return m, func() tea.Msg {
					return SwitchToBrowserMsg{}
				}
			}
		}
	}

	return m, nil
}

// visibleSuggestions returns the suggestions for the current page
func (m *SmartCatalogModel) visibleSuggestions() []FileSuggestion {
	if len(m.suggestions) == 0 {
		return nil
	}
	end := min(m.pageOffset+m.pageSize, len(m.suggestions))
	return m.suggestions[m.pageOffset:end]
}

// cursorInPage returns the cursor position relative to the current page
func (m *SmartCatalogModel) cursorInPage() int {
	return m.cursor - m.pageOffset
}

// totalPages returns the total number of pages
func (m *SmartCatalogModel) totalPages() int {
	if len(m.suggestions) == 0 {
		return 1
	}
	return (len(m.suggestions) + m.pageSize - 1) / m.pageSize
}

// currentPage returns the current page number (1-based)
func (m *SmartCatalogModel) currentPage() int {
	return m.pageOffset/m.pageSize + 1
}

// nextPage moves to the next page
func (m *SmartCatalogModel) nextPage() {
	if m.pageOffset+m.pageSize < len(m.suggestions) {
		m.pageOffset += m.pageSize
		m.cursor = m.pageOffset
	}
}

// prevPage moves to the previous page
func (m *SmartCatalogModel) prevPage() {
	m.pageOffset -= m.pageSize
	if m.pageOffset < 0 {
		m.pageOffset = 0
	}
	m.cursor = m.pageOffset
}

// ensureCursorInPage ensures cursor is within the current page
func (m *SmartCatalogModel) ensureCursorInPage() {
	if m.cursor < m.pageOffset {
		m.pageOffset = (m.cursor / m.pageSize) * m.pageSize
	} else if m.cursor >= m.pageOffset+m.pageSize {
		m.pageOffset = (m.cursor / m.pageSize) * m.pageSize
	}
}

func (m *SmartCatalogModel) moveCurrentFile() tea.Cmd {
	return func() tea.Msg {
		if m.cursor >= len(m.suggestions) {
			return SmartCatalogErrMsg{Err: fmt.Errorf("invalid selection")}
		}

		fs := m.suggestions[m.cursor]
		if fs.Suggestion == nil {
			return SmartCatalogErrMsg{Err: fmt.Errorf("no suggestion for file")}
		}

		// Get the destination item path
		destItemPath, err := m.repo.GetPath(fs.Suggestion.DestinationItemID)
		if err != nil {
			return SmartCatalogErrMsg{Err: fmt.Errorf("failed to get destination path: %w", err)}
		}

		// Move the file to the destination item folder
		destPath := filepath.Join(destItemPath, fs.File.Name)
		if err := os.Rename(fs.File.Path, destPath); err != nil {
			return SmartCatalogErrMsg{Err: fmt.Errorf("failed to move file: %w", err)}
		}

		return SmartCatalogFileMoved{
			FileName:    fs.File.Name,
			Destination: fs.Suggestion.DestinationItemID,
		}
	}
}

// View renders the smart catalog view
func (m *SmartCatalogModel) View() string {
	var b strings.Builder

	// Title
	b.WriteString(styles.Title.Render("Smart Catalog"))
	b.WriteString("\n\n")

	// Show inbox info
	if m.inboxNode != nil {
		b.WriteString(styles.MutedText.Render(fmt.Sprintf("Inbox: %s %s", m.inboxNode.ID, m.inboxNode.Name)))
		b.WriteString("\n\n")
	}

	switch m.state {
	case SmartCatalogLoading:
		b.WriteString(m.spinner.View())
		fileCount := len(m.suggestions)
		if fileCount > 0 {
			fmt.Fprintf(&b, " Analyzing %d files with Claude...", fileCount)
		} else {
			b.WriteString(" Analyzing files with Claude...")
		}
		b.WriteString("\n\n")
		b.WriteString(styles.MutedText.Render("Press "))
		b.WriteString(styles.HelpKey.Render("esc"))
		b.WriteString(styles.MutedText.Render(" to cancel"))

	case SmartCatalogShowList:
		if len(m.suggestions) == 0 {
			if len(m.skipped) > 0 {
				b.WriteString(styles.MutedText.Render(fmt.Sprintf("No files remaining. %d skipped - press 'r' to review", len(m.skipped))))
			} else {
				b.WriteString(styles.MutedText.Render("No files to catalog"))
			}
		} else {
			// Review mode header
			if m.reviewMode {
				b.WriteString(styles.Success.Render("Reviewing skipped files"))
				b.WriteString("\n\n")
			}

			// File list (paginated)
			visible := m.visibleSuggestions()
			for i, fs := range visible {
				absIndex := m.pageOffset + i
				if absIndex == m.cursor {
					// Selected item with destination
					dest := "no suggestion"
					if fs.Suggestion != nil {
						dest = fmt.Sprintf("%s %s", fs.Suggestion.DestinationItemID, fs.Suggestion.DestinationItemName)
					}
					b.WriteString(styles.NodeSelected.Render(fmt.Sprintf(" > %s ", fs.File.Name)))
					b.WriteString(styles.MutedText.Render(" → "))
					b.WriteString(dest)
				} else {
					fmt.Fprintf(&b, "   %s", fs.File.Name)
				}
				b.WriteString("\n")
			}

			// Page indicator (if more than one page)
			if m.totalPages() > 1 {
				b.WriteString("\n")
				b.WriteString(styles.MutedText.Render(fmt.Sprintf("Page %d/%d", m.currentPage(), m.totalPages())))
			}

			// Details for selected item
			if m.cursor < len(m.suggestions) {
				fs := m.suggestions[m.cursor]
				b.WriteString("\n")
				if fs.Suggestion != nil {
					b.WriteString(styles.InputLabel.Render("Destination: "))
					fmt.Fprintf(&b, "%s %s", fs.Suggestion.DestinationItemID, fs.Suggestion.DestinationItemName)
					b.WriteString("\n")
					b.WriteString(styles.InputLabel.Render("Reasoning: "))
					b.WriteString(styles.MutedText.Render(fs.Suggestion.Reasoning))
				} else {
					b.WriteString(styles.ErrorMsg.Render("No suggestion available for this file"))
				}
				b.WriteString("\n")
			}

			// Help
			b.WriteString("\n")
			b.WriteString(styles.HelpKey.Render("j/k"))
			b.WriteString(styles.HelpDesc.Render(" navigate, "))
			if m.totalPages() > 1 {
				b.WriteString(styles.HelpKey.Render("ctrl+f/b"))
				b.WriteString(styles.HelpDesc.Render(" page, "))
			}
			b.WriteString(styles.HelpKey.Render("y"))
			b.WriteString(styles.HelpDesc.Render(" accept, "))
			b.WriteString(styles.HelpKey.Render("n"))
			b.WriteString(styles.HelpDesc.Render(" skip, "))
			if len(m.skipped) > 0 && !m.reviewMode {
				b.WriteString(styles.HelpKey.Render("r"))
				b.WriteString(styles.HelpDesc.Render(fmt.Sprintf(" review (%d), ", len(m.skipped))))
			}
			b.WriteString(styles.HelpKey.Render("esc"))
			b.WriteString(styles.HelpDesc.Render(" cancel"))

			// Status
			fmt.Fprintf(&b, "     %d remaining", len(m.suggestions))
			if m.moved > 0 {
				fmt.Fprintf(&b, ", %d moved", m.moved)
			}
			if len(m.skipped) > 0 {
				fmt.Fprintf(&b, ", %d skipped", len(m.skipped))
			}
		}

	case SmartCatalogError:
		b.WriteString(styles.ErrorMsg.Render("Error: "))
		if m.err != nil {
			b.WriteString(m.err.Error())
		}
		b.WriteString("\n\n")
		b.WriteString(styles.MutedText.Render("Press any key to return"))
	}

	return styles.App.Render(b.String())
}

// HandleFileMoved processes a successful file move
func (m *SmartCatalogModel) HandleFileMoved() {
	m.moved++
	// Remove current item from list
	if m.cursor < len(m.suggestions) {
		m.suggestions = append(m.suggestions[:m.cursor], m.suggestions[m.cursor+1:]...)
		// Adjust cursor if needed
		if m.cursor >= len(m.suggestions) && m.cursor > 0 {
			m.cursor = len(m.suggestions) - 1
		}
		m.ensureCursorInPage()
	}
}

// IsEmpty returns true if there are no more suggestions
func (m *SmartCatalogModel) IsEmpty() bool {
	return len(m.suggestions) == 0
}

// GetMovedCount returns the number of files moved
func (m *SmartCatalogModel) GetMovedCount() int {
	return m.moved
}

// Messages

// SwitchToSmartCatalogMsg requests switching to smart catalog view
type SwitchToSmartCatalogMsg struct {
	SourceNode *application.TreeNode
}

// SmartCatalogSuggestionsMsg indicates suggestions were received
type SmartCatalogSuggestionsMsg struct {
	Suggestions []FileSuggestion
}

// SmartCatalogFetchErrMsg indicates an error during suggestion fetch
type SmartCatalogFetchErrMsg struct {
	Err error
}

// SmartCatalogFileMoved indicates a file was successfully moved
type SmartCatalogFileMoved struct {
	FileName    string
	Destination string
}

// SmartCatalogDoneMsg indicates all files have been processed
type SmartCatalogDoneMsg struct {
	Moved int
}

// SmartCatalogErrMsg indicates an error during cataloging
type SmartCatalogErrMsg struct {
	Err error
}
