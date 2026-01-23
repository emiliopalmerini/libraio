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

// ArchiveModel is the model for the archive confirmation view
type ArchiveModel struct {
	ConfirmationModel
	repo ports.VaultRepository
}

// NewArchiveModel creates a new archive view model
func NewArchiveModel(repo ports.VaultRepository) *ArchiveModel {
	return &ArchiveModel{
		ConfirmationModel: NewConfirmationModel(),
		repo:              repo,
	}
}

// Init initializes the archive view
func (m *ArchiveModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the archive view
func (m *ArchiveModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case tea.KeyMsg:
		handled, cmd := m.HandleKeyMsg(msg,
			func() tea.Msg { return m.doArchive() },
			func() tea.Msg { return SwitchToBrowserMsg{} },
		)
		if handled {
			return m, cmd
		}
	}

	return m, nil
}

func (m *ArchiveModel) doArchive() tea.Msg {
	if m.TargetNode == nil {
		return ArchiveErrMsg{Err: fmt.Errorf("no target selected")}
	}

	ctx := context.Background()

	switch m.TargetNode.Type {
	case application.IDTypeItem:
		cmd := commands.NewArchiveItemCommand(m.repo, m.TargetNode.ID)
		result, err := cmd.Execute(ctx)
		if err != nil {
			return ArchiveErrMsg{Err: err}
		}
		return ArchiveSuccessMsg{
			Message: fmt.Sprintf("Archived %s %s -> %s", m.TargetNode.ID, m.TargetNode.Name, result.ArchivedItem.ID),
		}

	case application.IDTypeCategory:
		cmd := commands.NewArchiveCategoryCommand(m.repo, m.TargetNode.ID)
		result, err := cmd.Execute(ctx)
		if err != nil {
			return ArchiveErrMsg{Err: err}
		}
		return ArchiveSuccessMsg{
			Message: fmt.Sprintf("Archived %d items from %s %s", len(result.ArchivedItems), m.TargetNode.ID, m.TargetNode.Name),
		}

	default:
		return ArchiveErrMsg{Err: fmt.Errorf("cannot archive %s (only items and categories can be archived)", m.TargetNode.Type)}
	}
}

// ArchiveSuccessMsg indicates successful archiving
type ArchiveSuccessMsg struct {
	Message string
}

// ArchiveErrMsg indicates an error during archiving
type ArchiveErrMsg struct {
	Err error
}

// SwitchToArchiveMsg requests switching to archive view
type SwitchToArchiveMsg struct {
	TargetNode *application.TreeNode
}

// View renders the archive confirmation view
func (m *ArchiveModel) View() string {
	var b strings.Builder

	// Title
	b.WriteString(styles.Title.Render("Archive Confirmation"))
	b.WriteString("\n\n")

	// Info
	b.WriteString(styles.MutedText.Render("Items will be moved to the archive category with updated links."))
	b.WriteString("\n\n")

	// Target info with description
	if m.TargetNode != nil {
		b.WriteString(RenderTargetInfo(m.TargetNode, "Archive"))
		b.WriteString("\n\n")

		// Type-specific description
		description := m.getArchiveDescription()
		b.WriteString(styles.MutedText.Render("  " + strings.ReplaceAll(description, "\n", "\n  ")))
		b.WriteString("\n\n")

		// Show archive destination
		if dest := m.getArchiveDestination(); dest != "" {
			b.WriteString(styles.MutedText.Render(fmt.Sprintf("  Destination: %s Archive", dest)))
			b.WriteString("\n\n")
		}
	}

	// Confirmation prompt
	b.WriteString(RenderConfirmPrompt("Proceed with archive?"))

	return styles.App.Render(b.String())
}

func (m *ArchiveModel) getArchiveDescription() string {
	if m.TargetNode == nil {
		return ""
	}

	switch m.TargetNode.Type {
	case application.IDTypeItem:
		return "This item will be moved to the archive category with a new ID."
	case application.IDTypeCategory:
		return "All items in this category will be moved to the archive category.\nThe category will be deleted after archiving."
	default:
		return "Only items and categories can be archived."
	}
}

func (m *ArchiveModel) getArchiveDestination() string {
	if m.TargetNode == nil {
		return ""
	}

	switch m.TargetNode.Type {
	case application.IDTypeItem:
		categoryID, err := application.ParseCategory(m.TargetNode.ID)
		if err != nil {
			return ""
		}
		archiveItemID, err := application.ArchiveItemID(categoryID)
		if err != nil {
			return ""
		}
		return archiveItemID

	case application.IDTypeCategory:
		archiveItemID, err := application.ArchiveItemID(m.TargetNode.ID)
		if err != nil {
			return ""
		}
		return archiveItemID
	}

	return ""
}
