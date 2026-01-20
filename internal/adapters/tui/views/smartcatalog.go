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
	Up      key.Binding
	Down    key.Binding
	Confirm key.Binding
	Skip    key.Binding
	Cancel  key.Binding
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
	cursor      int                   // Currently selected suggestion
	moved       int                   // Count of files moved
	state       SmartCatalogState
	err         error
	spinner     spinner.Model
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
	checkLen := len(content)
	if checkLen > 512 {
		checkLen = 512
	}
	for i := 0; i < checkLen; i++ {
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
			b.WriteString(fmt.Sprintf("- %s %s\n", item.ID, item.Name))
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
			b.WriteString(fmt.Sprintf("\n%s %s\n", cat.ID, cat.Name))
			items, _ := repo.ListItems(cat.ID)
			for _, item := range items {
				if num, _ := domain.ExtractNumber(item.ID); num <= domain.StandardZeroMax {
					continue
				}
				b.WriteString(fmt.Sprintf("  - %s %s\n", item.ID, item.Name))
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
				b.WriteString(fmt.Sprintf("\n%s %s\n", cat.ID, cat.Name))
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
					b.WriteString(fmt.Sprintf("  - %s %s\n", item.ID, item.Name))
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
				}
				return m, nil
			case key.Matches(msg, SmartCatalogKeys.Down):
				if m.cursor < len(m.suggestions)-1 {
					m.cursor++
				}
				return m, nil
			case key.Matches(msg, SmartCatalogKeys.Confirm):
				if len(m.suggestions) > 0 && m.cursor < len(m.suggestions) {
					return m, m.moveCurrentFile()
				}
				return m, nil
			case key.Matches(msg, SmartCatalogKeys.Skip):
				// Move to next, or finish if at end
				if m.cursor < len(m.suggestions)-1 {
					m.cursor++
				} else if len(m.suggestions) > 0 {
					// At last item, remove it and stay at valid index
					m.suggestions = append(m.suggestions[:m.cursor], m.suggestions[m.cursor+1:]...)
					if m.cursor >= len(m.suggestions) && m.cursor > 0 {
						m.cursor = len(m.suggestions) - 1
					}
					if len(m.suggestions) == 0 {
						return m, func() tea.Msg {
							return SmartCatalogDoneMsg{Moved: m.moved}
						}
					}
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
			b.WriteString(fmt.Sprintf(" Analyzing %d files with Claude...", fileCount))
		} else {
			b.WriteString(" Analyzing files with Claude...")
		}
		b.WriteString("\n\n")
		b.WriteString(styles.MutedText.Render("Press "))
		b.WriteString(styles.HelpKey.Render("esc"))
		b.WriteString(styles.MutedText.Render(" to cancel"))

	case SmartCatalogShowList:
		if len(m.suggestions) == 0 {
			b.WriteString(styles.MutedText.Render("No files to catalog"))
		} else {
			// File list
			for i, fs := range m.suggestions {
				if i == m.cursor {
					// Selected item with destination
					dest := "no suggestion"
					if fs.Suggestion != nil {
						dest = fmt.Sprintf("%s %s", fs.Suggestion.DestinationItemID, fs.Suggestion.DestinationItemName)
					}
					b.WriteString(styles.NodeSelected.Render(fmt.Sprintf(" > %s ", fs.File.Name)))
					b.WriteString(styles.MutedText.Render(" → "))
					b.WriteString(dest)
				} else {
					b.WriteString(fmt.Sprintf("   %s", fs.File.Name))
				}
				b.WriteString("\n")
			}

			// Details for selected item
			if m.cursor < len(m.suggestions) {
				fs := m.suggestions[m.cursor]
				b.WriteString("\n")
				if fs.Suggestion != nil {
					b.WriteString(styles.InputLabel.Render("Destination: "))
					b.WriteString(fmt.Sprintf("%s %s", fs.Suggestion.DestinationItemID, fs.Suggestion.DestinationItemName))
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
			b.WriteString(styles.HelpKey.Render("y"))
			b.WriteString(styles.HelpDesc.Render(" accept, "))
			b.WriteString(styles.HelpKey.Render("n"))
			b.WriteString(styles.HelpDesc.Render(" skip, "))
			b.WriteString(styles.HelpKey.Render("esc"))
			b.WriteString(styles.HelpDesc.Render(" cancel"))

			// Status
			b.WriteString(fmt.Sprintf("     %d remaining", len(m.suggestions)))
			if m.moved > 0 {
				b.WriteString(fmt.Sprintf(", %d moved", m.moved))
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
