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
	CreateModeCategory CreateMode = iota
	CreateModeItem
)

// CreateModel is the model for the create view
type CreateModel struct {
	repo         ports.VaultRepository
	openInEditor bool
	parentNode   *application.TreeNode
	mode         CreateMode
	descInput    textinput.Model
	parentInput  textinput.Model
	focusedField int
	message      string
	messageErr   bool
	width        int
	height       int
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
	m.message = ""
	m.messageErr = false

	// Determine mode and prefill parent
	switch node.Type {
	case application.IDTypeArea:
		m.mode = CreateModeCategory
		m.parentInput.SetValue(node.ID)
	case application.IDTypeCategory:
		m.mode = CreateModeItem
		m.parentInput.SetValue(node.ID)
	case application.IDTypeScope:
		// Need to select area first
		m.mode = CreateModeCategory
		m.parentInput.SetValue("")
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
		m.width = msg.Width
		m.height = msg.Height
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

		if parentID == "" {
			return CreateErrMsg{Err: fmt.Errorf("parent ID is required")}
		}
		if description == "" {
			return CreateErrMsg{Err: fmt.Errorf("description is required")}
		}

		ctx := context.Background()
		parentType := application.ParseIDType(parentID)

		switch parentType {
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
			return CreateErrMsg{Err: fmt.Errorf("invalid parent type: %s (expected area or category)", parentType)}
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

	// Title
	title := "Create New Item"
	if m.mode == CreateModeCategory {
		title = "Create New Category"
	}
	b.WriteString(styles.Title.Render(title))
	b.WriteString("\n\n")

	// Instructions
	if m.mode == CreateModeItem {
		b.WriteString(styles.Subtitle.Render("Creating item in category. A README will be generated."))
	} else {
		b.WriteString(styles.Subtitle.Render("Creating category in area. No README will be created."))
	}
	b.WriteString("\n\n")

	// Parent ID field
	parentLabel := "Parent (Category ID):"
	if m.mode == CreateModeCategory {
		parentLabel = "Parent (Area ID):"
	}
	b.WriteString(styles.InputLabel.Render(parentLabel))
	b.WriteString("\n")
	if m.focusedField == 0 {
		b.WriteString(styles.InputFocused.Render(m.parentInput.View()))
	} else {
		b.WriteString(styles.InputField.Render(m.parentInput.View()))
	}
	b.WriteString("\n\n")

	// Description field
	b.WriteString(styles.InputLabel.Render("Description:"))
	b.WriteString("\n")
	if m.focusedField == 1 {
		b.WriteString(styles.InputFocused.Render(m.descInput.View()))
	} else {
		b.WriteString(styles.InputField.Render(m.descInput.View()))
	}
	b.WriteString("\n\n")

	// Message
	if m.message != "" {
		if m.messageErr {
			b.WriteString(styles.ErrorMsg.Render(m.message))
		} else {
			b.WriteString(styles.Success.Render(m.message))
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

// SetMessage sets a message to display
func (m *CreateModel) SetMessage(msg string, isErr bool) {
	m.message = msg
	m.messageErr = isErr
}

// SetSize updates the view dimensions
func (m *CreateModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}
