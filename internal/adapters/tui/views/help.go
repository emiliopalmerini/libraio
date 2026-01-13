package views

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"librarian/internal/adapters/tui/styles"
)

// HelpKeyMap defines key bindings for the help view
type HelpKeyMap struct {
	Close key.Binding
}

var HelpKeys = HelpKeyMap{
	Close: key.NewBinding(
		key.WithKeys("esc", "q", "?"),
		key.WithHelp("esc/q/?", "close"),
	),
}

// HelpModel is the model for the help view
type HelpModel struct {
	width  int
	height int
}

// NewHelpModel creates a new help view model
func NewHelpModel() *HelpModel {
	return &HelpModel{}
}

// Init initializes the help view
func (m *HelpModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the help view
func (m *HelpModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if key.Matches(msg, HelpKeys.Close) {
			return m, func() tea.Msg {
				return SwitchToBrowserMsg{}
			}
		}
	}

	return m, nil
}

// View renders the help view
func (m *HelpModel) View() string {
	var b strings.Builder

	b.WriteString(styles.Title.Render("Librarian Help"))
	b.WriteString("\n\n")

	b.WriteString(styles.Subtitle.Render("Johnny Decimal Vault Manager"))
	b.WriteString("\n\n")

	// Navigation section
	b.WriteString(styles.InputLabel.Render("Navigation"))
	b.WriteString("\n")
	b.WriteString(helpLine("j / k / ↑ / ↓", "Move up/down"))
	b.WriteString(helpLine("h / ←", "Collapse / go to parent"))
	b.WriteString(helpLine("l / → / Enter", "Expand / enter"))
	b.WriteString("\n")

	// Actions section
	b.WriteString(styles.InputLabel.Render("Actions"))
	b.WriteString("\n")
	b.WriteString(helpLine("n", "Create new item/category"))
	b.WriteString(helpLine("a", "Archive selected item"))
	b.WriteString(helpLine("m", "Move selected item"))
	b.WriteString(helpLine("/", "Search"))
	b.WriteString("\n")

	// General section
	b.WriteString(styles.InputLabel.Render("General"))
	b.WriteString("\n")
	b.WriteString(helpLine("?", "Toggle help"))
	b.WriteString(helpLine("q / Ctrl+C", "Quit"))
	b.WriteString("\n\n")

	// Johnny Decimal info
	b.WriteString(styles.InputLabel.Render("Johnny Decimal Structure"))
	b.WriteString("\n")
	b.WriteString(styles.MutedText.Render("  Scope    : S00, S01, S02, S03"))
	b.WriteString("\n")
	b.WriteString(styles.MutedText.Render("  Area     : S01.10-19"))
	b.WriteString("\n")
	b.WriteString(styles.MutedText.Render("  Category : S01.11"))
	b.WriteString("\n")
	b.WriteString(styles.MutedText.Render("  Item     : S01.11.11"))
	b.WriteString("\n\n")

	// Close hint
	b.WriteString(styles.HelpDesc.Render("Press "))
	b.WriteString(styles.HelpKey.Render("esc"))
	b.WriteString(styles.HelpDesc.Render(" or "))
	b.WriteString(styles.HelpKey.Render("?"))
	b.WriteString(styles.HelpDesc.Render(" to close"))

	return styles.App.Render(b.String())
}

func helpLine(key, desc string) string {
	return "  " + styles.HelpKey.Render(padRight(key, 20)) + styles.HelpDesc.Render(desc) + "\n"
}

func padRight(s string, length int) string {
	if len(s) >= length {
		return s
	}
	return s + strings.Repeat(" ", length-len(s))
}

// SetSize updates the view dimensions
func (m *HelpModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}
