package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"libraio/internal/adapters/tui/styles"
	"libraio/internal/application"
	"libraio/internal/ports"
)

// DeleteModel is the model for the delete confirmation view
type DeleteModel struct {
	ConfirmationModel
	repo ports.VaultRepository
}

// NewDeleteModel creates a new delete view model
func NewDeleteModel(repo ports.VaultRepository) *DeleteModel {
	return &DeleteModel{
		ConfirmationModel: NewConfirmationModel(),
		repo:              repo,
	}
}

// Init initializes the delete view
func (m *DeleteModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the delete view
func (m *DeleteModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case tea.KeyMsg:
		handled, cmd := m.HandleKeyMsg(msg,
			func() tea.Msg { return m.doDelete() },
			func() tea.Msg { return SwitchToBrowserMsg{} },
		)
		if handled {
			return m, cmd
		}
	}

	return m, nil
}

func (m *DeleteModel) doDelete() tea.Msg {
	if m.TargetNode == nil {
		return DeleteErrMsg{Err: fmt.Errorf("no target selected")}
	}

	if err := m.repo.Delete(m.TargetNode.ID); err != nil {
		return DeleteErrMsg{Err: err}
	}

	return DeleteSuccessMsg{
		Message: fmt.Sprintf("Deleted %s %s", m.TargetNode.ID, m.TargetNode.Name),
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
	b.WriteString(RenderTargetInfo(m.TargetNode, "Delete"))
	b.WriteString("\n\n")

	// Additional warning for containers
	if m.TargetNode != nil && m.TargetNode.Type != application.IDTypeItem {
		b.WriteString(styles.MutedText.Render("  All contents will be permanently deleted."))
		b.WriteString("\n\n")
	}

	// Confirmation prompt
	b.WriteString(RenderConfirmPrompt("Are you sure?"))

	return styles.App.Render(b.String())
}
