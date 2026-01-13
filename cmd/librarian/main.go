package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"librarian/internal/adapters/editor"
	"librarian/internal/adapters/filesystem"
	"librarian/internal/adapters/tui"
)

const vaultPath = "~/Documents/bag_of_holding"

func main() {
	// Initialize adapters
	repo := filesystem.NewRepository(vaultPath)
	editorOpener := editor.NewOpener()

	// Create and run TUI app
	app := tui.NewApp(repo, editorOpener)

	p := tea.NewProgram(app, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
