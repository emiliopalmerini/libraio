package ports

// FileInfo represents a file to be cataloged
type FileInfo struct {
	Name    string
	Path    string // Full path to the file
	Content string // May be empty for binary files
}

// CatalogSuggestion represents a suggested destination for cataloging a file
type CatalogSuggestion struct {
	FileName            string // Which file this suggestion is for
	DestinationItemID   string // Full item ID (e.g., S01.11.15)
	DestinationItemName string // Item name (e.g., Theatre)
	Reasoning           string

	// Alternative suggestion (used when user skips the primary)
	AltDestinationItemID   string
	AltDestinationItemName string
	AltReasoning           string
}

// SmartSearchResult represents a search result from AI-powered natural language search
type SmartSearchResult struct {
	Path      string  // Relative path to the item folder
	JDID      string  // e.g., "S01.11.15"
	Name      string  // e.g., "Theatre"
	Type      string  // "scope", "area", "category", "item"
	Score     float64 // Relevance 0-1
	Reasoning string  // Why this matches
}

// AIAssistant defines the interface for AI-powered assistance features
type AIAssistant interface {
	// SuggestCatalogDestinations analyzes multiple files and suggests destinations
	SuggestCatalogDestinations(
		files []FileInfo,
		vaultStructure string,
	) ([]CatalogSuggestion, error)

	// SmartSearch performs AI-powered natural language search against vault structure
	SmartSearch(query string, vaultStructure string) ([]SmartSearchResult, error)

	// IsAvailable returns true if the AI assistant (e.g., Claude CLI) is available
	IsAvailable() bool
}
