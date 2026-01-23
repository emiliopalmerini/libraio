package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"

	"libraio/internal/adapters/tui/styles"
)

// RenderKeyHelp formats a key binding as help text (key + description)
func RenderKeyHelp(b key.Binding) string {
	help := b.Help()
	return fmt.Sprintf("%s %s",
		styles.HelpKey.Render(help.Key),
		styles.HelpDesc.Render(help.Desc),
	)
}

// RenderHelpLine renders multiple key bindings as a help line separated by bullets
func RenderHelpLine(bindings ...key.Binding) string {
	var parts []string
	for _, b := range bindings {
		parts = append(parts, RenderKeyHelp(b))
	}
	return strings.Join(parts, styles.HelpSeparator.String())
}

// RenderMessage renders a message with appropriate styling based on isError
func RenderMessage(message string, isError bool) string {
	if message == "" {
		return ""
	}
	if isError {
		return styles.ErrorMsg.Render(message)
	}
	return styles.Success.Render(message)
}

// RenderTitle renders a title with the standard title style
func RenderTitle(title string) string {
	return styles.Title.Render(title)
}

// RenderSubtitle renders a subtitle with the standard subtitle style
func RenderSubtitle(subtitle string) string {
	return styles.Subtitle.Render(subtitle)
}

// RenderMuted renders muted/secondary text
func RenderMuted(text string) string {
	return styles.MutedText.Render(text)
}

// RenderLabelValue renders a label: value pair
func RenderLabelValue(label, value string) string {
	return fmt.Sprintf("%s %s",
		styles.InputLabel.Render(label+":"),
		value,
	)
}

// ViewBuilder helps construct view output with consistent formatting
type ViewBuilder struct {
	b strings.Builder
}

// NewViewBuilder creates a new view builder
func NewViewBuilder() *ViewBuilder {
	return &ViewBuilder{}
}

// Title adds a title section
func (v *ViewBuilder) Title(title string) *ViewBuilder {
	v.b.WriteString(RenderTitle(title))
	v.b.WriteString("\n\n")
	return v
}

// Subtitle adds a subtitle section
func (v *ViewBuilder) Subtitle(subtitle string) *ViewBuilder {
	v.b.WriteString(RenderSubtitle(subtitle))
	v.b.WriteString("\n\n")
	return v
}

// Line adds a line of text
func (v *ViewBuilder) Line(text string) *ViewBuilder {
	v.b.WriteString(text)
	v.b.WriteString("\n")
	return v
}

// BlankLine adds a blank line
func (v *ViewBuilder) BlankLine() *ViewBuilder {
	v.b.WriteString("\n")
	return v
}

// Muted adds muted text followed by a newline
func (v *ViewBuilder) Muted(text string) *ViewBuilder {
	v.b.WriteString(RenderMuted(text))
	v.b.WriteString("\n")
	return v
}

// Message adds a message if non-empty, with appropriate error/success styling
func (v *ViewBuilder) Message(message string, isError bool) *ViewBuilder {
	if message == "" {
		return v
	}
	v.b.WriteString(RenderMessage(message, isError))
	v.b.WriteString("\n\n")
	return v
}

// Help adds a help line with key bindings
func (v *ViewBuilder) Help(bindings ...key.Binding) *ViewBuilder {
	v.b.WriteString(RenderHelpLine(bindings...))
	return v
}

// Raw adds raw text without any formatting
func (v *ViewBuilder) Raw(text string) *ViewBuilder {
	v.b.WriteString(text)
	return v
}

// String returns the built view string wrapped in the app style
func (v *ViewBuilder) String() string {
	return styles.App.Render(v.b.String())
}

// StringUnwrapped returns the built view string without app style wrapping
func (v *ViewBuilder) StringUnwrapped() string {
	return v.b.String()
}
