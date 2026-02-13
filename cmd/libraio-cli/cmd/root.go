package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"libraio/internal/adapters/filesystem"
	"libraio/internal/config"
	"libraio/internal/ports"
)

var (
	vaultPath string
	repo      ports.VaultRepository
)

var rootCmd = &cobra.Command{
	Use:   "libraio-cli",
	Short: "CLI for managing Johnny Decimal vaults",
	Long: `libraio-cli is a command-line interface for managing Obsidian vaults
organized with the Johnny Decimal system.

It provides commands to list, create, move, archive, delete, and search
items within your vault.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip initialization for help commands
		if cmd.Name() == "help" || cmd.Name() == "completion" {
			return nil
		}
		repo = filesystem.NewRepository(vaultPath)
		return nil
	},
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&vaultPath, "vault", "v", config.VaultPath(), "path to the vault")
}

// GetRepo returns the initialized repository
func GetRepo() ports.VaultRepository {
	return repo
}
