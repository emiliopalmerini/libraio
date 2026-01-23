package views

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"libraio/internal/adapters/tui/styles"
)

// InputFormKeyMap defines key bindings for input forms
type InputFormKeyMap struct {
	Submit key.Binding
	Cancel key.Binding
	Tab    key.Binding
}

// DefaultInputFormKeys returns the default input form key bindings
var DefaultInputFormKeys = InputFormKeyMap{
	Submit: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "submit"),
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

// InputField represents a single input field with label and textinput
type InputField struct {
	Label string
	Input textinput.Model
}

// InputForm manages multiple text input fields with focus handling
type InputForm struct {
	Fields       []InputField
	FocusedField int
	Keys         InputFormKeyMap
}

// NewInputForm creates a new input form with the given fields
func NewInputForm(fields ...InputField) *InputForm {
	form := &InputForm{
		Fields:       fields,
		FocusedField: 0,
		Keys:         DefaultInputFormKeys,
	}
	// Focus the first field
	if len(fields) > 0 {
		form.Fields[0].Input.Focus()
	}
	return form
}

// NewInputField creates a new input field with the given label and placeholder
func NewInputField(label, placeholder string, charLimit int) InputField {
	input := textinput.New()
	input.Placeholder = placeholder
	if charLimit > 0 {
		input.CharLimit = charLimit
	}
	return InputField{
		Label: label,
		Input: input,
	}
}

// Init returns the blink command for the focused input
func (f *InputForm) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages for the input form.
// Returns (handled, cmd) where handled is true if the key was processed.
func (f *InputForm) Update(msg tea.Msg) (bool, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, f.Keys.Tab):
			f.NextField()
			return true, nil
		}
	}

	// Update the focused input
	var cmd tea.Cmd
	if f.FocusedField >= 0 && f.FocusedField < len(f.Fields) {
		f.Fields[f.FocusedField].Input, cmd = f.Fields[f.FocusedField].Input.Update(msg)
	}
	return false, cmd
}

// NextField moves focus to the next field
func (f *InputForm) NextField() {
	if len(f.Fields) <= 1 {
		return
	}

	// Blur current field
	f.Fields[f.FocusedField].Input.Blur()

	// Move to next field
	f.FocusedField = (f.FocusedField + 1) % len(f.Fields)

	// Focus new field
	f.Fields[f.FocusedField].Input.Focus()
}

// SetFocus sets focus to a specific field
func (f *InputForm) SetFocus(index int) {
	if index < 0 || index >= len(f.Fields) {
		return
	}

	// Blur current field
	if f.FocusedField >= 0 && f.FocusedField < len(f.Fields) {
		f.Fields[f.FocusedField].Input.Blur()
	}

	// Focus new field
	f.FocusedField = index
	f.Fields[f.FocusedField].Input.Focus()
}

// Value returns the value of a field by index
func (f *InputForm) Value(index int) string {
	if index < 0 || index >= len(f.Fields) {
		return ""
	}
	return strings.TrimSpace(f.Fields[index].Input.Value())
}

// SetValue sets the value of a field by index
func (f *InputForm) SetValue(index int, value string) {
	if index < 0 || index >= len(f.Fields) {
		return
	}
	f.Fields[index].Input.SetValue(value)
}

// Reset clears all field values and resets focus to the first field
func (f *InputForm) Reset() {
	for i := range f.Fields {
		f.Fields[i].Input.SetValue("")
		f.Fields[i].Input.Blur()
	}
	f.FocusedField = 0
	if len(f.Fields) > 0 {
		f.Fields[0].Input.Focus()
	}
}

// RenderField renders a single field with appropriate styling
func (f *InputForm) RenderField(index int) string {
	if index < 0 || index >= len(f.Fields) {
		return ""
	}

	field := f.Fields[index]
	var b strings.Builder

	b.WriteString(styles.InputLabel.Render(field.Label))
	b.WriteString("\n")

	if index == f.FocusedField {
		b.WriteString(styles.InputFocused.Render(field.Input.View()))
	} else {
		b.WriteString(styles.InputField.Render(field.Input.View()))
	}

	return b.String()
}

// RenderHelp renders the help text for the form
func (f *InputForm) RenderHelp(submitText string) string {
	var parts []string

	if len(f.Fields) > 1 {
		parts = append(parts, styles.HelpKey.Render("tab")+" "+styles.HelpDesc.Render("next field"))
	}
	parts = append(parts, styles.HelpKey.Render("enter")+" "+styles.HelpDesc.Render(submitText))
	parts = append(parts, styles.HelpKey.Render("esc")+" "+styles.HelpDesc.Render("cancel"))

	return strings.Join(parts, "  ")
}
