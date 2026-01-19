package views

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"libraio/internal/adapters/tui/styles"
	"libraio/internal/application"
	"libraio/internal/application/commands"
	"libraio/internal/ports"
)

// MoveKeyMap defines key bindings for the move view
type MoveKeyMap struct {
	Submit key.Binding
	Cancel key.Binding
}

var MoveKeys = MoveKeyMap{
	Submit: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "move"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
	),
}

// MoveModel is the model for the move view
type MoveModel struct {
	ViewState
	repo       ports.VaultRepository
	sourceNode *application.TreeNode
	destInput  textinput.Model
}

// NewMoveModel creates a new move view model
func NewMoveModel(repo ports.VaultRepository) *MoveModel {
	destInput := textinput.New()
	destInput.Placeholder = "S01.12 (category) or S01.20-29 (area)"
	destInput.CharLimit = 20

	return &MoveModel{
		repo:      repo,
		destInput: destInput,
	}
}

// SetSource sets the source node for the move operation
func (m *MoveModel) SetSource(node *application.TreeNode) {
	m.sourceNode = node
	m.Message = ""
	m.MessageErr = false
	m.destInput.SetValue("")
	m.destInput.Focus()
}

// Init initializes the move view
func (m *MoveModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages for the move view
func (m *MoveModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, MoveKeys.Cancel):
			return m, func() tea.Msg {
				return SwitchToBrowserMsg{}
			}

		case key.Matches(msg, MoveKeys.Submit):
			return m, m.move()
		}
	}

	// Update input
	var cmd tea.Cmd
	m.destInput, cmd = m.destInput.Update(msg)
	return m, cmd
}

func (m *MoveModel) move() tea.Cmd {
	return func() tea.Msg {
		if m.sourceNode == nil {
			return MoveErrMsg{Err: fmt.Errorf("no source selected")}
		}

		destID := strings.TrimSpace(m.destInput.Value())
		if destID == "" {
			return MoveErrMsg{Err: fmt.Errorf("destination is required")}
		}

		ctx := context.Background()

		switch m.sourceNode.Type {
		case application.IDTypeItem:
			cmd := commands.NewMoveItemCommand(m.repo, m.sourceNode.ID, destID)
			result, err := cmd.Execute(ctx)
			if err != nil {
				return MoveErrMsg{Err: err}
			}
			return MoveSuccessMsg{Message: result.Message}

		case application.IDTypeCategory:
			cmd := commands.NewMoveCategoryCommand(m.repo, m.sourceNode.ID, destID)
			result, err := cmd.Execute(ctx)
			if err != nil {
				return MoveErrMsg{Err: err}
			}
			return MoveSuccessMsg{Message: result.Message}

		default:
			return MoveErrMsg{Err: fmt.Errorf("can only move items or categories")}
		}
	}
}

// MoveSuccessMsg indicates successful move
type MoveSuccessMsg struct {
	Message string
}

// MoveErrMsg indicates an error during move
type MoveErrMsg struct {
	Err error
}

// View renders the move view
func (m *MoveModel) View() string {
	var b strings.Builder

	// Title
	title := "Move Item"
	if m.sourceNode != nil && m.sourceNode.Type == application.IDTypeCategory {
		title = "Move Category"
	}
	b.WriteString(styles.Title.Render(title))
	b.WriteString("\n\n")

	// Source info
	if m.sourceNode != nil {
		b.WriteString(styles.InputLabel.Render("Source:"))
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("  %s %s", m.sourceNode.ID, m.sourceNode.Name))
		b.WriteString("\n\n")

		// Instructions
		switch m.sourceNode.Type {
		case application.IDTypeItem:
			b.WriteString(styles.Subtitle.Render("Enter destination category ID (e.g., S01.12)"))
		case application.IDTypeCategory:
			b.WriteString(styles.Subtitle.Render("Enter destination area ID (e.g., S01.20-29)"))
		}
		b.WriteString("\n\n")
	}

	// Destination input
	b.WriteString(styles.InputLabel.Render("Destination:"))
	b.WriteString("\n")
	b.WriteString(styles.InputFocused.Render(m.destInput.View()))
	b.WriteString("\n\n")

	// Message
	if m.Message != "" {
		if m.MessageErr {
			b.WriteString(styles.ErrorMsg.Render(m.Message))
		} else {
			b.WriteString(styles.Success.Render(m.Message))
		}
		b.WriteString("\n\n")
	}

	// Help
	b.WriteString(fmt.Sprintf("%s %s  %s %s",
		styles.HelpKey.Render("enter"),
		styles.HelpDesc.Render("move"),
		styles.HelpKey.Render("esc"),
		styles.HelpDesc.Render("cancel"),
	))

	return styles.App.Render(b.String())
}

