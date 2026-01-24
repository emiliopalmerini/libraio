package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"libraio/internal/adapters/tui/styles"
	"libraio/internal/ports"
)

// SmartSearchState represents the state of the smart search view
type SmartSearchState int

const (
	SmartSearchInput SmartSearchState = iota
	SmartSearchLoading
	SmartSearchResults
	SmartSearchError
)

// SmartSearchKeyMap defines key bindings for the smart search view
type SmartSearchKeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Select   key.Binding
	Cancel   key.Binding
	NextPage key.Binding
	PrevPage key.Binding
}

var SmartSearchKeys = SmartSearchKeyMap{
	Up: key.NewBinding(
		key.WithKeys("k", "up"),
		key.WithHelp("k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("j", "down"),
		key.WithHelp("j", "down"),
	),
	Select: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "open"),
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
}

// SmartSearchModel is the model for the smart search view
type SmartSearchModel struct {
	ViewState
	assistant      ports.AIAssistant
	vaultStructure string

	state       SmartSearchState
	searchInput textinput.Model
	spinner     spinner.Model
	paginator   *Paginator

	query   string
	results []ports.SmartSearchResult
	err     error
}

// NewSmartSearchModel creates a new smart search view model
func NewSmartSearchModel(assistant ports.AIAssistant, vaultStructure string) *SmartSearchModel {
	input := textinput.New()
	input.Placeholder = "Ask about your notes..."
	input.Prompt = "Search: "
	input.Focus()

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = styles.Spinner

	return &SmartSearchModel{
		assistant:      assistant,
		vaultStructure: vaultStructure,
		state:          SmartSearchInput,
		searchInput:    input,
		spinner:        s,
		paginator:      NewPaginator(10),
	}
}

// Init initializes the smart search view
func (m *SmartSearchModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages for the smart search view
func (m *SmartSearchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case spinner.TickMsg:
		if m.state == SmartSearchLoading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case SmartSearchResultsMsg:
		m.results = msg.Results
		m.paginator.SetTotal(len(m.results))
		m.state = SmartSearchResults
		return m, nil

	case SmartSearchErrorMsg:
		m.err = msg.Err
		m.state = SmartSearchError
		return m, nil

	case tea.KeyMsg:
		switch m.state {
		case SmartSearchInput:
			return m.updateInputMode(msg)
		case SmartSearchResults:
			return m.updateResultsMode(msg)
		case SmartSearchError:
			// Any key returns to browser
			return m, func() tea.Msg {
				return SwitchToBrowserMsg{}
			}
		case SmartSearchLoading:
			if key.Matches(msg, SmartSearchKeys.Cancel) {
				return m, func() tea.Msg {
					return SwitchToBrowserMsg{}
				}
			}
		}
	}
	return m, nil
}

func (m *SmartSearchModel) updateInputMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		query := strings.TrimSpace(m.searchInput.Value())
		if len(query) < 3 {
			return m, nil
		}
		m.query = query
		m.state = SmartSearchLoading
		return m, tea.Batch(
			m.spinner.Tick,
			m.performSearch(),
		)
	case tea.KeyEsc:
		return m, func() tea.Msg {
			return SwitchToBrowserMsg{}
		}
	}

	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	return m, cmd
}

func (m *SmartSearchModel) updateResultsMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, SmartSearchKeys.Cancel):
		return m, func() tea.Msg {
			return SwitchToBrowserMsg{}
		}
	case key.Matches(msg, SmartSearchKeys.Up):
		m.paginator.CursorUp()
		return m, nil
	case key.Matches(msg, SmartSearchKeys.Down):
		m.paginator.CursorDown()
		return m, nil
	case key.Matches(msg, SmartSearchKeys.NextPage):
		m.paginator.NextPage()
		return m, nil
	case key.Matches(msg, SmartSearchKeys.PrevPage):
		m.paginator.PrevPage()
		return m, nil
	case key.Matches(msg, SmartSearchKeys.Select):
		if len(m.results) > 0 {
			cursor := m.paginator.Cursor()
			if cursor < len(m.results) {
				return m, func() tea.Msg {
					return SmartSearchSelectMsg{JDID: m.results[cursor].JDID}
				}
			}
		}
		return m, nil
	}
	return m, nil
}

func (m *SmartSearchModel) performSearch() tea.Cmd {
	return func() tea.Msg {
		if m.assistant == nil {
			return SmartSearchErrorMsg{Err: fmt.Errorf("AI assistant not available")}
		}

		results, err := m.assistant.SmartSearch(m.query, m.vaultStructure)
		if err != nil {
			return SmartSearchErrorMsg{Err: err}
		}

		return SmartSearchResultsMsg{Results: results}
	}
}

// visibleResults returns the results for the current page
func (m *SmartSearchModel) visibleResults() []ports.SmartSearchResult {
	if len(m.results) == 0 {
		return nil
	}
	start, end := m.paginator.VisibleRange()
	return m.results[start:end]
}

// View renders the smart search view
func (m *SmartSearchModel) View() string {
	var b strings.Builder

	// Title
	b.WriteString(styles.Title.Render("Smart Search"))
	b.WriteString("\n\n")

	switch m.state {
	case SmartSearchInput:
		b.WriteString(m.searchInput.View())
		b.WriteString("\n\n")
		b.WriteString(styles.MutedText.Render("Enter a natural language query (e.g., 'find my Go programming notes')"))
		b.WriteString("\n\n")
		b.WriteString(styles.HelpKey.Render("enter"))
		b.WriteString(styles.HelpDesc.Render(" search, "))
		b.WriteString(styles.HelpKey.Render("esc"))
		b.WriteString(styles.HelpDesc.Render(" cancel"))

	case SmartSearchLoading:
		b.WriteString(m.spinner.View())
		b.WriteString(" Searching with Claude...")
		b.WriteString("\n\n")
		b.WriteString(styles.MutedText.Render("Press "))
		b.WriteString(styles.HelpKey.Render("esc"))
		b.WriteString(styles.MutedText.Render(" to cancel"))

	case SmartSearchResults:
		if len(m.results) == 0 {
			b.WriteString(styles.MutedText.Render("No matching items found for: "))
			b.WriteString(m.query)
			b.WriteString("\n\n")
			b.WriteString(styles.HelpKey.Render("esc"))
			b.WriteString(styles.HelpDesc.Render(" return"))
		} else {
			fmt.Fprintf(&b, "Found %d results for: %s\n\n", len(m.results), m.query)

			// Results list (paginated)
			visible := m.visibleResults()
			pageOffset := m.paginator.PageOffset()
			cursor := m.paginator.Cursor()
			for i, r := range visible {
				absIndex := pageOffset + i
				if absIndex == cursor {
					b.WriteString(styles.NodeSelected.Render(fmt.Sprintf(" > %s %s ", r.JDID, r.Name)))
				} else {
					fmt.Fprintf(&b, "   %s %s", r.JDID, r.Name)
				}
				b.WriteString("\n")
			}

			// Page indicator (if more than one page)
			if m.paginator.TotalPages() > 1 {
				b.WriteString("\n")
				b.WriteString(styles.MutedText.Render(fmt.Sprintf("Page %d/%d", m.paginator.CurrentPage(), m.paginator.TotalPages())))
			}

			// Details for selected item
			if cursor < len(m.results) {
				r := m.results[cursor]
				b.WriteString("\n\n")
				b.WriteString(styles.InputLabel.Render("Type: "))
				b.WriteString(r.Type)
				b.WriteString("\n")
				b.WriteString(styles.InputLabel.Render("Why: "))
				b.WriteString(styles.MutedText.Render(r.Reasoning))
			}

			// Help
			b.WriteString("\n\n")
			b.WriteString(styles.HelpKey.Render("j/k"))
			b.WriteString(styles.HelpDesc.Render(" navigate, "))
			if m.paginator.TotalPages() > 1 {
				b.WriteString(styles.HelpKey.Render("ctrl+f/b"))
				b.WriteString(styles.HelpDesc.Render(" page, "))
			}
			b.WriteString(styles.HelpKey.Render("enter"))
			b.WriteString(styles.HelpDesc.Render(" open, "))
			b.WriteString(styles.HelpKey.Render("esc"))
			b.WriteString(styles.HelpDesc.Render(" cancel"))
		}

	case SmartSearchError:
		b.WriteString(styles.ErrorMsg.Render("Error: "))
		if m.err != nil {
			b.WriteString(m.err.Error())
		}
		b.WriteString("\n\n")
		b.WriteString(styles.MutedText.Render("Press any key to return"))
	}

	return styles.App.Render(b.String())
}

// Messages

// SmartSearchResultsMsg indicates search completed
type SmartSearchResultsMsg struct {
	Results []ports.SmartSearchResult
}

// SmartSearchErrorMsg indicates search failed
type SmartSearchErrorMsg struct {
	Err error
}
