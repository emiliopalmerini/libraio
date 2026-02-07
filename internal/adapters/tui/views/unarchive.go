package views

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"libraio/internal/adapters/tui/styles"
	"libraio/internal/application"
	"libraio/internal/application/commands"
	"libraio/internal/ports"
)

// UnarchiveModel is the model for the unarchive confirmation view
type UnarchiveModel struct {
	ConfirmationModel
	repo ports.VaultRepository
}

// NewUnarchiveModel creates a new unarchive view model
func NewUnarchiveModel(repo ports.VaultRepository) *UnarchiveModel {
	return &UnarchiveModel{
		ConfirmationModel: NewConfirmationModel(),
		repo:              repo,
	}
}

// Init initializes the unarchive view
func (m *UnarchiveModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the unarchive view
func (m *UnarchiveModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case tea.KeyMsg:
		handled, cmd := m.HandleKeyMsg(msg,
			func() tea.Msg { return m.doUnarchive() },
			func() tea.Msg { return SwitchToBrowserMsg{} },
		)
		if handled {
			return m, cmd
		}
	}

	return m, nil
}

func (m *UnarchiveModel) doUnarchive() tea.Msg {
	if m.TargetNode == nil {
		return UnarchiveErrMsg{Err: fmt.Errorf("no target selected")}
	}

	ctx := context.Background()
	cmd := commands.NewUnarchiveItemCommand(m.repo, m.TargetNode.ID)
	result, err := cmd.Execute(ctx)
	if err != nil {
		return UnarchiveErrMsg{Err: err}
	}

	return UnarchiveSuccessMsg{
		Message: result.Message,
	}
}

// UnarchiveSuccessMsg indicates successful unarchiving
type UnarchiveSuccessMsg struct {
	Message string
}

// UnarchiveErrMsg indicates an error during unarchiving
type UnarchiveErrMsg struct {
	Err error
}

// View renders the unarchive confirmation view
func (m *UnarchiveModel) View() string {
	var b strings.Builder

	b.WriteString(styles.Title.Render("Unarchive Confirmation"))
	b.WriteString("\n\n")

	b.WriteString(styles.MutedText.Render("Archived items will be restored to their original category with new IDs."))
	b.WriteString("\n\n")

	if m.TargetNode != nil {
		b.WriteString(RenderTargetInfo(m.TargetNode, "Unarchive"))
		b.WriteString("\n\n")

		// Show destination
		dstCategoryID, err := application.ParseCategory(m.TargetNode.ID)
		if err == nil {
			b.WriteString(styles.MutedText.Render(fmt.Sprintf("  Destination: %s", dstCategoryID)))
			b.WriteString("\n\n")
		}
	}

	b.WriteString(RenderConfirmPrompt("Proceed with unarchive?"))

	return styles.App.Render(b.String())
}
