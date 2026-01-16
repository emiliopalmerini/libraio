package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"libraio/internal/adapters/editor"
	"libraio/internal/adapters/filesystem"
	"libraio/internal/adapters/obsidian"
	"libraio/internal/adapters/tui"
)

const vaultPath = "~/Documents/bag_of_holding"

func main() {
	// Initialize adapters
	repo := filesystem.NewRepository(vaultPath)
	editorOpener := editor.NewOpener()
	obsidianOpener := obsidian.NewOpener(repo.VaultPath())

	// Create and run TUI app
	app := tui.NewApp(repo, editorOpener, obsidianOpener)

	p := tea.NewProgram(app, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
