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

// AIAssistant defines the interface for AI-powered assistance features
type AIAssistant interface {
	// SuggestCatalogDestinations analyzes multiple files and suggests destinations
	SuggestCatalogDestinations(
		files []FileInfo,
		vaultStructure string,
	) ([]CatalogSuggestion, error)
}
