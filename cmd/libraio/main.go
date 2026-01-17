package main

import (
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"libraio/internal/adapters/claudecli"
	"libraio/internal/adapters/editor"
	"libraio/internal/adapters/filesystem"
	"libraio/internal/adapters/obsidian"
	"libraio/internal/adapters/sqlite"
	"libraio/internal/adapters/tui"
)

const vaultPath = "~/Documents/bag_of_holding"

func main() {
	// Initialize SQLite index for caching
	index := sqlite.NewIndex()
	if err := index.Open(vaultPath); err != nil {
		log.Printf("Warning: failed to open index, caching disabled: %v", err)
		index = nil
	} else {
		defer index.Close()

		// Sync index on startup
		if index.NeedsFullRebuild() {
			stats, err := index.SyncFull()
			if err != nil {
				log.Printf("Warning: full sync failed: %v", err)
			} else {
				log.Printf("Index rebuilt: %d nodes, %d edges in %v",
					stats.NodesAdded, stats.EdgesAdded, stats.Duration)
			}
		} else {
			stats, err := index.SyncIncremental()
			if err != nil {
				log.Printf("Warning: incremental sync failed: %v", err)
			} else if stats.NodesAdded > 0 || stats.NodesUpdated > 0 || stats.NodesDeleted > 0 {
				log.Printf("Index updated: +%d/~%d/-%d nodes in %v",
					stats.NodesAdded, stats.NodesUpdated, stats.NodesDeleted, stats.Duration)
			}
		}
	}

	// Initialize adapters
	var repo *filesystem.Repository
	if index != nil {
		repo = filesystem.NewRepository(vaultPath, filesystem.WithIndex(index))
	} else {
		repo = filesystem.NewRepository(vaultPath)
	}
	editorOpener := editor.NewOpener()
	obsidianOpener := obsidian.NewOpener(repo.VaultPath())
	aiAssistant := claudecli.NewAssistant()

	// Create and run TUI app
	app := tui.NewApp(repo, editorOpener, obsidianOpener, aiAssistant)

	p := tea.NewProgram(app, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
