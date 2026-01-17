package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"libraio/internal/application/commands"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete an entity",
	Long: `Delete an item, category, area, or scope from the vault.

Warning: This operation cannot be undone. Deleting a container
(scope, area, or category) will also delete all its contents.

Examples:
  libraio-cli delete S01.11.15    # Delete item
  libraio-cli delete S01.11       # Delete category and all items`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		ctx := context.Background()

		deleteCmd := commands.NewDeleteCommand(GetRepo(), id)
		result, err := deleteCmd.Execute(ctx)
		if err != nil {
			return err
		}
		fmt.Println(result.Message)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
