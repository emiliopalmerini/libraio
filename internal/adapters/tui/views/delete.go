package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"libraio/internal/adapters/tui/styles"
	"libraio/internal/domain"
	"libraio/internal/ports"
)

// DeleteKeyMap defines key bindings for the delete view
type DeleteKeyMap struct {
	Confirm key.Binding
	Cancel  key.Binding
}

var DeleteKeys = DeleteKeyMap{
	Confirm: key.NewBinding(
		key.WithKeys("y"),
		key.WithHelp("y", "confirm"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("n", "esc"),
		key.WithHelp("n/esc", "cancel"),
	),
}

// DeleteModel is the model for the delete confirmation view
type DeleteModel struct {
	repo       ports.VaultRepository
	targetNode *domain.TreeNode
	width      int
	height     int
}

// NewDeleteModel creates a new delete view model
func NewDeleteModel(repo ports.VaultRepository) *DeleteModel {
	return &DeleteModel{
		repo: repo,
	}
}

// SetTarget sets the target node for deletion
func (m *DeleteModel) SetTarget(node *domain.TreeNode) {
	m.targetNode = node
}

// Init initializes the delete view
func (m *DeleteModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the delete view
func (m *DeleteModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, DeleteKeys.Cancel):
			return m, func() tea.Msg {
				return SwitchToBrowserMsg{}
			}

		case key.Matches(msg, DeleteKeys.Confirm):
			return m, m.delete()
		}
	}

	return m, nil
}

func (m *DeleteModel) delete() tea.Cmd {
	return func() tea.Msg {
		if m.targetNode == nil {
			return DeleteErrMsg{Err: fmt.Errorf("no target selected")}
		}

		if err := m.repo.Delete(m.targetNode.ID); err != nil {
			return DeleteErrMsg{Err: err}
		}

		return DeleteSuccessMsg{
			Message: fmt.Sprintf("Deleted %s %s", m.targetNode.ID, m.targetNode.Name),
		}
	}
}

// DeleteSuccessMsg indicates successful deletion
type DeleteSuccessMsg struct {
	Message string
}

// DeleteErrMsg indicates an error during deletion
type DeleteErrMsg struct {
	Err error
}

// View renders the delete confirmation view
func (m *DeleteModel) View() string {
	var b strings.Builder

	// Title
	b.WriteString(styles.Title.Render("Delete Confirmation"))
	b.WriteString("\n\n")

	// Warning
	b.WriteString(styles.ErrorMsg.Render("This action cannot be undone!"))
	b.WriteString("\n\n")

	// Target info
	if m.targetNode != nil {
		typeStr := ""
		switch m.targetNode.Type {
		case domain.IDTypeItem:
			typeStr = "Item"
		case domain.IDTypeCategory:
			typeStr = "Category"
		case domain.IDTypeArea:
			typeStr = "Area"
		case domain.IDTypeScope:
			typeStr = "Scope"
		}

		b.WriteString(styles.InputLabel.Render(fmt.Sprintf("Delete %s:", typeStr)))
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("  %s %s", m.targetNode.ID, m.targetNode.Name))
		b.WriteString("\n\n")

		// Additional warning for containers
		if m.targetNode.Type != domain.IDTypeItem {
			b.WriteString(styles.MutedText.Render("  All contents will be permanently deleted."))
			b.WriteString("\n\n")
		}
	}

	// Confirmation prompt
	b.WriteString("Are you sure? ")
	b.WriteString(styles.HelpKey.Render("y"))
	b.WriteString(styles.HelpDesc.Render(" to confirm, "))
	b.WriteString(styles.HelpKey.Render("n"))
	b.WriteString(styles.HelpDesc.Render(" to cancel"))

	return styles.App.Render(b.String())
}

// SetSize updates the view dimensions
func (m *DeleteModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}
