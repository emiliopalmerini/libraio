package views

import (
	"fmt"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"librarian/internal/adapters/tui/styles"
	"librarian/internal/domain"
	"librarian/internal/ports"
)

// SearchKeyMap defines key bindings for the search view
type SearchKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Select key.Binding
	Cancel key.Binding
}

var SearchKeys = SearchKeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "ctrl+p"),
		key.WithHelp("↑", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "ctrl+n"),
		key.WithHelp("↓", "down"),
	),
	Select: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
	),
}

// SearchModel is the model for the search view
type SearchModel struct {
	repo      ports.VaultRepository
	input     textinput.Model
	results   []domain.SearchResult
	cursor    int
	searching bool
	width     int
	height    int
}

// NewSearchModel creates a new search view model
func NewSearchModel(repo ports.VaultRepository) *SearchModel {
	input := textinput.New()
	input.Placeholder = "Search..."
	input.Focus()

	return &SearchModel{
		repo:  repo,
		input: input,
	}
}

// Init initializes the search view
func (m *SearchModel) Init() tea.Cmd {
	return textinput.Blink
}

// Reset resets the search view
func (m *SearchModel) Reset() {
	m.input.SetValue("")
	m.results = nil
	m.cursor = 0
	m.input.Focus()
}

// Update handles messages for the search view
func (m *SearchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case searchResultsMsg:
		m.results = msg.results
		m.cursor = 0
		m.searching = false
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, SearchKeys.Cancel):
			return m, func() tea.Msg {
				return SwitchToBrowserMsg{}
			}

		case key.Matches(msg, SearchKeys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil

		case key.Matches(msg, SearchKeys.Down):
			if m.cursor < len(m.results)-1 {
				m.cursor++
			}
			return m, nil

		case key.Matches(msg, SearchKeys.Select):
			if m.cursor >= 0 && m.cursor < len(m.results) {
				result := m.results[m.cursor]
				// Copy ID to clipboard
				clipboard.WriteAll(result.ID)
				return m, func() tea.Msg {
					return SearchSelectMsg{Result: result}
				}
			}
			return m, nil
		}
	}

	// Update input
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)

	// Trigger search on input change
	query := m.input.Value()
	if len(query) >= 2 {
		return m, tea.Batch(cmd, m.search(query))
	} else if len(query) == 0 {
		m.results = nil
	}

	return m, cmd
}

func (m *SearchModel) search(query string) tea.Cmd {
	return func() tea.Msg {
		results, err := m.repo.Search(query)
		if err != nil {
			return searchResultsMsg{results: nil}
		}
		return searchResultsMsg{results: results}
	}
}

type searchResultsMsg struct {
	results []domain.SearchResult
}

// SearchSelectMsg is sent when a search result is selected
type SearchSelectMsg struct {
	Result domain.SearchResult
}

// View renders the search view
func (m *SearchModel) View() string {
	var b strings.Builder

	// Title
	b.WriteString(styles.Title.Render("Search"))
	b.WriteString("\n\n")

	// Search input
	b.WriteString(styles.InputFocused.Render(m.input.View()))
	b.WriteString("\n\n")

	// Results
	if len(m.results) == 0 {
		if len(m.input.Value()) >= 2 {
			b.WriteString(styles.MutedText.Render("No results found"))
		} else {
			b.WriteString(styles.MutedText.Render("Type at least 2 characters to search"))
		}
	} else {
		b.WriteString(styles.Subtitle.Render(fmt.Sprintf("%d results", len(m.results))))
		b.WriteString("\n\n")

		// Show max 10 results
		maxResults := 10
		if len(m.results) < maxResults {
			maxResults = len(m.results)
		}

		for i := 0; i < maxResults; i++ {
			result := m.results[i]
			line := m.renderResult(result, i == m.cursor)
			b.WriteString(line)
			b.WriteString("\n")
		}

		if len(m.results) > 10 {
			b.WriteString(styles.MutedText.Render(fmt.Sprintf("... and %d more", len(m.results)-10)))
		}
	}

	b.WriteString("\n\n")

	// Help
	b.WriteString(fmt.Sprintf("%s %s  %s %s  %s %s",
		styles.HelpKey.Render("↑/↓"),
		styles.HelpDesc.Render("navigate"),
		styles.HelpKey.Render("enter"),
		styles.HelpDesc.Render("copy ID"),
		styles.HelpKey.Render("esc"),
		styles.HelpDesc.Render("cancel"),
	))

	return styles.App.Render(b.String())
}

func (m *SearchModel) renderResult(result domain.SearchResult, selected bool) string {
	// Type indicator
	var typeStr string
	switch result.Type {
	case domain.IDTypeScope:
		typeStr = "[SCOPE]"
	case domain.IDTypeArea:
		typeStr = "[AREA]"
	case domain.IDTypeCategory:
		typeStr = "[CAT]"
	case domain.IDTypeItem:
		typeStr = "[ITEM]"
	}

	text := fmt.Sprintf("%s %s %s", typeStr, result.ID, result.Name)

	if selected {
		return styles.NodeSelected.Render(text)
	}

	return text
}

// SetSize updates the view dimensions
func (m *SearchModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}
