package views

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"libraio/internal/adapters/tui/styles"
	"libraio/internal/application"
	"libraio/internal/application/commands"
	"libraio/internal/ports"
)

// ArchiveKeyMap defines key bindings for the archive view
type ArchiveKeyMap struct {
	Confirm key.Binding
	Cancel  key.Binding
}

var ArchiveKeys = ArchiveKeyMap{
	Confirm: key.NewBinding(
		key.WithKeys("y"),
		key.WithHelp("y", "confirm"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("n", "esc"),
		key.WithHelp("n/esc", "cancel"),
	),
}

// ArchiveModel is the model for the archive confirmation view
type ArchiveModel struct {
	repo       ports.VaultRepository
	targetNode *application.TreeNode
	width      int
	height     int
}

// NewArchiveModel creates a new archive view model
func NewArchiveModel(repo ports.VaultRepository) *ArchiveModel {
	return &ArchiveModel{
		repo: repo,
	}
}

// SetTarget sets the target node for archiving
func (m *ArchiveModel) SetTarget(node *application.TreeNode) {
	m.targetNode = node
}

// Init initializes the archive view
func (m *ArchiveModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the archive view
func (m *ArchiveModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, ArchiveKeys.Cancel):
			return m, func() tea.Msg {
				return SwitchToBrowserMsg{}
			}

		case key.Matches(msg, ArchiveKeys.Confirm):
			return m, m.archive()
		}
	}

	return m, nil
}

func (m *ArchiveModel) archive() tea.Cmd {
	return func() tea.Msg {
		if m.targetNode == nil {
			return ArchiveErrMsg{Err: fmt.Errorf("no target selected")}
		}

		ctx := context.Background()

		switch m.targetNode.Type {
		case application.IDTypeItem:
			cmd := commands.NewArchiveItemCommand(m.repo, m.targetNode.ID)
			result, err := cmd.Execute(ctx)
			if err != nil {
				return ArchiveErrMsg{Err: err}
			}
			return ArchiveSuccessMsg{
				Message: fmt.Sprintf("Archived %s %s → %s", m.targetNode.ID, m.targetNode.Name, result.ArchivedItem.ID),
			}

		case application.IDTypeCategory:
			cmd := commands.NewArchiveCategoryCommand(m.repo, m.targetNode.ID)
			result, err := cmd.Execute(ctx)
			if err != nil {
				return ArchiveErrMsg{Err: err}
			}
			return ArchiveSuccessMsg{
				Message: fmt.Sprintf("Archived %d items from %s %s", len(result.ArchivedItems), m.targetNode.ID, m.targetNode.Name),
			}

		default:
			return ArchiveErrMsg{Err: fmt.Errorf("cannot archive %s (only items and categories can be archived)", m.targetNode.Type)}
		}
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

	// Target info
	if m.targetNode != nil {
		typeStr := ""
		description := ""
		switch m.targetNode.Type {
		case application.IDTypeItem:
			typeStr = "Item"
			description = "This item will be moved to the archive category with a new ID."
		case application.IDTypeCategory:
			typeStr = "Category"
			description = "All items in this category will be moved to the archive category.\nThe category will be deleted after archiving."
		default:
			typeStr = m.targetNode.Type.String()
			description = "Only items and categories can be archived."
		}

		b.WriteString(styles.InputLabel.Render(fmt.Sprintf("Archive %s:", typeStr)))
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("  %s %s", m.targetNode.ID, m.targetNode.Name))
		b.WriteString("\n\n")

		b.WriteString(styles.MutedText.Render("  " + strings.ReplaceAll(description, "\n", "\n  ")))
		b.WriteString("\n\n")

		// Show archive destination
		if m.targetNode.Type == application.IDTypeItem || m.targetNode.Type == application.IDTypeCategory {
			areaID, err := application.ParseArea(m.targetNode.ID)
			if err == nil {
				archiveCatID, err := application.ArchiveCategory(areaID)
				if err == nil {
					b.WriteString(styles.MutedText.Render(fmt.Sprintf("  Destination: %s Archive", archiveCatID)))
					b.WriteString("\n\n")
				}
			}
		}
	}

	// Confirmation prompt
	b.WriteString("Proceed with archive? ")
	b.WriteString(styles.HelpKey.Render("y"))
	b.WriteString(styles.HelpDesc.Render(" to confirm, "))
	b.WriteString(styles.HelpKey.Render("n"))
	b.WriteString(styles.HelpDesc.Render(" to cancel"))

	return styles.App.Render(b.String())
}

// SetSize updates the view dimensions
func (m *ArchiveModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}
