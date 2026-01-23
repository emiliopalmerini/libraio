package views

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"libraio/internal/adapters/tui/styles"
	"libraio/internal/application"
)

// ConfirmKeyMap defines key bindings for confirmation views
type ConfirmKeyMap struct {
	Confirm key.Binding
	Cancel  key.Binding
}

// DefaultConfirmKeys returns the default confirmation key bindings
var DefaultConfirmKeys = ConfirmKeyMap{
	Confirm: key.NewBinding(
		key.WithKeys("y"),
		key.WithHelp("y", "confirm"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("n", "esc"),
		key.WithHelp("n/esc", "cancel"),
	),
}

// ConfirmationModel provides a base for confirmation-style views (delete, archive, etc.)
type ConfirmationModel struct {
	ViewState
	TargetNode *application.TreeNode
	Keys       ConfirmKeyMap
}

// NewConfirmationModel creates a new confirmation model with default keys
func NewConfirmationModel() ConfirmationModel {
	return ConfirmationModel{
		Keys: DefaultConfirmKeys,
	}
}

// SetTarget sets the target node for the confirmation
func (m *ConfirmationModel) SetTarget(node *application.TreeNode) {
	m.TargetNode = node
}

// HandleKeyMsg processes key messages for confirmation views.
// Returns (handled, cmd) where handled is true if the key was processed.
func (m *ConfirmationModel) HandleKeyMsg(msg tea.KeyMsg, onConfirm, onCancel func() tea.Msg) (bool, tea.Cmd) {
	switch {
	case key.Matches(msg, m.Keys.Cancel):
		return true, func() tea.Msg { return onCancel() }
	case key.Matches(msg, m.Keys.Confirm):
		return true, func() tea.Msg { return onConfirm() }
	}
	return false, nil
}

// RenderConfirmPrompt renders the standard confirmation prompt
func RenderConfirmPrompt(question string) string {
	var b strings.Builder
	b.WriteString(question)
	b.WriteString(" ")
	b.WriteString(styles.HelpKey.Render("y"))
	b.WriteString(styles.HelpDesc.Render(" to confirm, "))
	b.WriteString(styles.HelpKey.Render("n"))
	b.WriteString(styles.HelpDesc.Render(" to cancel"))
	return b.String()
}

// RenderTargetInfo renders information about the target node
func RenderTargetInfo(node *application.TreeNode, action string) string {
	if node == nil {
		return ""
	}

	var b strings.Builder
	typeStr := nodeTypeString(node.Type)

	b.WriteString(styles.InputLabel.Render(action + " " + typeStr + ":"))
	b.WriteString("\n")
	b.WriteString("  ")
	b.WriteString(node.ID)
	b.WriteString(" ")
	b.WriteString(node.Name)

	return b.String()
}

// nodeTypeString returns a human-readable string for the node type
func nodeTypeString(t application.IDType) string {
	switch t {
	case application.IDTypeItem:
		return "Item"
	case application.IDTypeCategory:
		return "Category"
	case application.IDTypeArea:
		return "Area"
	case application.IDTypeScope:
		return "Scope"
	default:
		return t.String()
	}
}
