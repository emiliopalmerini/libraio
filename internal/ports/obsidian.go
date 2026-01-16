package ports

// ObsidianOpener defines the interface for opening files in Obsidian
type ObsidianOpener interface {
	// OpenFile opens the specified file in Obsidian using the obsidian:// URI scheme
	// The filePath should be an absolute path to a file within the vault
	OpenFile(filePath string) error
}
