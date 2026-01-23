package styles

import "github.com/charmbracelet/lipgloss"

var (
	// Colors - NASA inspired: white, black, orange, gray
	Primary   = lipgloss.Color("#FF6B35") // NASA Orange
	Secondary = lipgloss.Color("#FFFFFF") // White
	Muted     = lipgloss.Color("#71717A") // Gray
	Warning   = lipgloss.Color("#FB923C") // Light Orange
	Error     = lipgloss.Color("#DC2626") // Red
	White     = lipgloss.Color("#FFFFFF")
	Black     = lipgloss.Color("#000000")

	// Scope colors - variations of orange and gray
	ScopeS00 = lipgloss.Color("#FF6B35") // NASA Orange
	ScopeS01 = lipgloss.Color("#F97316") // Bright Orange
	ScopeS02 = lipgloss.Color("#EA580C") // Deep Orange
	ScopeS03 = lipgloss.Color("#9CA3AF") // Cool Gray

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
			Foreground(lipgloss.Color("#D4D4D8")) // Light Gray

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

	// Spinner style
	Spinner = lipgloss.NewStyle().
		Foreground(Primary)
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

// NodeType represents the type of a tree node for styling purposes
type NodeType int

const (
	NodeTypeUnknown NodeType = iota
	NodeTypeScope
	NodeTypeArea
	NodeTypeCategory
	NodeTypeCategoryArchive
	NodeTypeItem
	NodeTypeFile
)

// NodeStyler provides styles for different node types
type NodeStyler struct{}

// GetStyle returns the appropriate style for a node type
func (s *NodeStyler) GetStyle(nodeType NodeType, scopeID string) lipgloss.Style {
	switch nodeType {
	case NodeTypeScope:
		return NodeScope.Foreground(ScopeColor(scopeID))
	case NodeTypeArea:
		return NodeArea
	case NodeTypeCategory:
		return NodeCategory
	case NodeTypeCategoryArchive:
		return NodeArchive
	case NodeTypeItem:
		return NodeItem
	case NodeTypeFile:
		return MutedText
	default:
		return lipgloss.NewStyle()
	}
}

// DefaultNodeStyler is the default node styler instance
var DefaultNodeStyler = &NodeStyler{}
