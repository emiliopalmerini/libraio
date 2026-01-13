package styles

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	Primary   = lipgloss.Color("#7C3AED") // Purple
	Secondary = lipgloss.Color("#10B981") // Green
	Muted     = lipgloss.Color("#6B7280") // Gray
	Warning   = lipgloss.Color("#F59E0B") // Amber
	Error     = lipgloss.Color("#EF4444") // Red
	White     = lipgloss.Color("#FFFFFF")
	Black     = lipgloss.Color("#000000")

	// Scope colors
	ScopeS00 = lipgloss.Color("#6366F1") // Indigo
	ScopeS01 = lipgloss.Color("#8B5CF6") // Violet
	ScopeS02 = lipgloss.Color("#EC4899") // Pink
	ScopeS03 = lipgloss.Color("#F97316") // Orange

	// Base styles
	App = lipgloss.NewStyle().
		Padding(1, 2)

	Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(Primary).
		MarginBottom(1)

	Subtitle = lipgloss.NewStyle().
			Foreground(Muted).
			Italic(true)

	// Tree node styles
	NodeScope = lipgloss.NewStyle().
			Bold(true)

	NodeArea = lipgloss.NewStyle().
			Foreground(Secondary)

	NodeCategory = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#60A5FA")) // Blue

	NodeItem = lipgloss.NewStyle()

	NodeSelected = lipgloss.NewStyle().
			Background(Primary).
			Foreground(White).
			Bold(true)

	NodeArchive = lipgloss.NewStyle().
			Foreground(Muted).
			Italic(true)

	// Tree indicators
	TreeBranch    = lipgloss.NewStyle().Foreground(Muted)
	TreeExpanded  = "▼ "
	TreeCollapsed = "▶ "
	TreeLeaf      = "  "

	// Status bar
	StatusBar = lipgloss.NewStyle().
			Background(lipgloss.Color("#1F2937")).
			Foreground(White).
			Padding(0, 1)

	StatusKey = lipgloss.NewStyle().
			Background(Primary).
			Foreground(White).
			Padding(0, 1).
			MarginRight(1)

	StatusText = lipgloss.NewStyle().
			Foreground(Muted)

	// Input styles
	InputLabel = lipgloss.NewStyle().
			Foreground(Secondary).
			Bold(true)

	InputField = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Primary).
			Padding(0, 1)

	InputFocused = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Secondary).
			Padding(0, 1)

	// Help styles
	HelpKey = lipgloss.NewStyle().
		Foreground(Primary).
		Bold(true)

	HelpDesc = lipgloss.NewStyle().
			Foreground(Muted)

	HelpSeparator = lipgloss.NewStyle().
			Foreground(Muted).
			SetString(" • ")

	// Message styles
	Success = lipgloss.NewStyle().
		Foreground(Secondary).
		Bold(true)

	ErrorMsg = lipgloss.NewStyle().
			Foreground(Error).
			Bold(true)

	// Search
	SearchMatch = lipgloss.NewStyle().
			Background(Warning).
			Foreground(Black)

	// Muted text style (for using Muted color as a style)
	MutedText = lipgloss.NewStyle().
			Foreground(Muted)
)

// ScopeColor returns the color for a scope ID
func ScopeColor(scopeID string) lipgloss.Color {
	switch scopeID {
	case "S00":
		return ScopeS00
	case "S01":
		return ScopeS01
	case "S02":
		return ScopeS02
	case "S03":
		return ScopeS03
	default:
		return Primary
	}
}
