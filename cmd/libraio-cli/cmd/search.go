package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"libraio/internal/application/commands"
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search the vault",
	Long: `Search for items in the vault by ID, name, or content.

Results are ranked by relevance using fuzzy matching.

Examples:
  libraio-cli search theatre
  libraio-cli search S01.11`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := args[0]
		ctx := context.Background()

		searchCmd := commands.NewSearchCommand(GetRepo(), query)
		results, err := searchCmd.Execute(ctx)
		if err != nil {
			return err
		}

		if len(results) == 0 {
			fmt.Println("No results found")
			return nil
		}

		for _, r := range results {
			typeStr := strings.ToLower(r.Type.String())
			fmt.Printf("[%s] %s %s\n", typeStr, r.ID, r.Name)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)
}
