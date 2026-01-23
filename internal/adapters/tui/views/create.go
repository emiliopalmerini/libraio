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

// CreateKeyMap defines key bindings for the create view
type CreateKeyMap struct {
	Submit key.Binding
	Cancel key.Binding
	Tab    key.Binding
}

var CreateKeys = CreateKeyMap{
	Submit: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "create"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "next field"),
	),
}

// CreateMode indicates what type of item to create
type CreateMode int

const (
	CreateModeScope CreateMode = iota
	CreateModeArea
	CreateModeCategory
	CreateModeItem
)

// CreateModel is the model for the create view
type CreateModel struct {
	ViewState
	repo         ports.VaultRepository
	openInEditor bool
	parentNode   *application.TreeNode
	mode         CreateMode
	descInput    textinput.Model
	parentInput  textinput.Model
	focusedField int
}

// NewCreateModel creates a new create view model
func NewCreateModel(repo ports.VaultRepository, openInEditor bool) *CreateModel {
	parentInput := textinput.New()
	parentInput.Placeholder = "S01.11 or S01.10-19"
	parentInput.CharLimit = 20

	descInput := textinput.New()
	descInput.Placeholder = "Description"
	descInput.CharLimit = 100

	return &CreateModel{
		repo:         repo,
		openInEditor: openInEditor,
		parentInput:  parentInput,
		descInput:    descInput,
	}
}

// SetParent sets the parent node for creation
func (m *CreateModel) SetParent(node *application.TreeNode) {
	m.parentNode = node
	m.Message = ""
	m.MessageErr = false

	// Determine mode and prefill parent
	switch node.Type {
	case application.IDTypeUnknown: // Root - create scope
		m.mode = CreateModeScope
		m.parentInput.SetValue("")
	case application.IDTypeScope:
		m.mode = CreateModeArea
		m.parentInput.SetValue(node.ID)
	case application.IDTypeArea:
		m.mode = CreateModeCategory
		m.parentInput.SetValue(node.ID)
	case application.IDTypeCategory:
		m.mode = CreateModeItem
		m.parentInput.SetValue(node.ID)
	default:
		m.parentInput.SetValue("")
	}

	m.descInput.SetValue("")
	m.focusedField = 1 // Focus description by default
	m.descInput.Focus()
	m.parentInput.Blur()
}

// Init initializes the create view
func (m *CreateModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages for the create view
func (m *CreateModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, CreateKeys.Cancel):
			return m, func() tea.Msg {
				return SwitchToBrowserMsg{}
			}

		case key.Matches(msg, CreateKeys.Tab):
			m.focusedField = (m.focusedField + 1) % 2
			if m.focusedField == 0 {
				m.parentInput.Focus()
				m.descInput.Blur()
			} else {
				m.descInput.Focus()
				m.parentInput.Blur()
			}
			return m, nil

		case key.Matches(msg, CreateKeys.Submit):
			return m, m.create()
		}
	}

	// Update focused input
	var cmd tea.Cmd
	if m.focusedField == 0 {
		m.parentInput, cmd = m.parentInput.Update(msg)
	} else {
		m.descInput, cmd = m.descInput.Update(msg)
	}

	return m, cmd
}

func (m *CreateModel) create() tea.Cmd {
	return func() tea.Msg {
		parentID := strings.TrimSpace(m.parentInput.Value())
		description := strings.TrimSpace(m.descInput.Value())

		if description == "" {
			return CreateErrMsg{Err: fmt.Errorf("description is required")}
		}

		ctx := context.Background()

		// Handle scope creation (no parent needed)
		if m.mode == CreateModeScope {
			cmd := commands.NewCreateScopeCommand(m.repo, description)
			result, err := cmd.Execute(ctx)
			if err != nil {
				return CreateErrMsg{Err: err}
			}
			return CreateSuccessMsg{Message: result.Message}
		}

		// All other modes require a parent
		if parentID == "" {
			return CreateErrMsg{Err: fmt.Errorf("parent ID is required")}
		}

		parentType := application.ParseIDType(parentID)

		switch parentType {
		case application.IDTypeScope:
			cmd := commands.NewCreateAreaCommand(m.repo, parentID, description)
			result, err := cmd.Execute(ctx)
			if err != nil {
				return CreateErrMsg{Err: err}
			}
			return CreateSuccessMsg{Message: result.Message}

		case application.IDTypeArea:
			cmd := commands.NewCreateCategoryCommand(m.repo, parentID, description)
			result, err := cmd.Execute(ctx)
			if err != nil {
				return CreateErrMsg{Err: err}
			}
			return CreateSuccessMsg{Message: result.Message}

		case application.IDTypeCategory:
			cmd := commands.NewCreateItemCommand(m.repo, parentID, description)
			result, err := cmd.Execute(ctx)
			if err != nil {
				return CreateErrMsg{Err: err}
			}
			if m.openInEditor {
				return OpenEditorMsg{
					Path:    result.Item.JDexPath,
					Message: result.Message,
				}
			}
			return CreateSuccessMsg{Message: result.Message}

		default:
			return CreateErrMsg{Err: fmt.Errorf("invalid parent type: %s", parentType)}
		}
	}
}

// CreateSuccessMsg indicates successful creation
type CreateSuccessMsg struct {
	Message string
}

// CreateErrMsg indicates an error during creation
type CreateErrMsg struct {
	Err error
}

// OpenEditorMsg requests opening a file in editor
type OpenEditorMsg struct {
	Path    string
	Message string
}

// View renders the create view
func (m *CreateModel) View() string {
	var b strings.Builder

	// Title and instructions based on mode
	var title, subtitle, parentLabel string
	switch m.mode {
	case CreateModeScope:
		title = "Create New Scope"
		subtitle = "Creating a new scope in the vault."
		parentLabel = "" // No parent for scope
	case CreateModeArea:
		title = "Create New Area"
		subtitle = "Creating a new area in scope."
		parentLabel = "Parent (Scope ID):"
	case CreateModeCategory:
		title = "Create New Category"
		subtitle = "Creating category in area. Standard zeros will be created."
		parentLabel = "Parent (Area ID):"
	case CreateModeItem:
		title = "Create New Item"
		subtitle = "Creating item in category. A JDex file will be generated."
		parentLabel = "Parent (Category ID):"
	}

	b.WriteString(styles.Title.Render(title))
	b.WriteString("\n\n")
	b.WriteString(styles.Subtitle.Render(subtitle))
	b.WriteString("\n\n")

	// Parent ID field (not shown for scope creation)
	if m.mode != CreateModeScope {
		b.WriteString(styles.InputLabel.Render(parentLabel))
		b.WriteString("\n")
		if m.focusedField == 0 {
			b.WriteString(styles.InputFocused.Render(m.parentInput.View()))
		} else {
			b.WriteString(styles.InputField.Render(m.parentInput.View()))
		}
		b.WriteString("\n\n")
	}

	// Description field
	b.WriteString(styles.InputLabel.Render("Description:"))
	b.WriteString("\n")
	if m.focusedField == 1 || m.mode == CreateModeScope {
		b.WriteString(styles.InputFocused.Render(m.descInput.View()))
	} else {
		b.WriteString(styles.InputField.Render(m.descInput.View()))
	}
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
	b.WriteString(fmt.Sprintf("%s %s  %s %s  %s %s",
		styles.HelpKey.Render("tab"),
		styles.HelpDesc.Render("next field"),
		styles.HelpKey.Render("enter"),
		styles.HelpDesc.Render("create"),
		styles.HelpKey.Render("esc"),
		styles.HelpDesc.Render("cancel"),
	))

	return styles.App.Render(b.String())
}
