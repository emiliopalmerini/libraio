package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"libraio/internal/application/commands"
	"libraio/internal/domain"
)

var archiveCmd = &cobra.Command{
	Use:   "archive <id>",
	Short: "Archive an item or category",
	Long: `Archive an item or all items in a category to the archive category.

Items are moved to the area's archive category (e.g., S01.19 Archive).
Archiving a category moves all its items to the archive.

Examples:
  libraio-cli archive S01.11.15    # Archive single item
  libraio-cli archive S01.11       # Archive all items in category`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		ctx := context.Background()

		idType := domain.ParseIDType(id)

		switch idType {
		case domain.IDTypeItem:
			archiveCmd := commands.NewArchiveItemCommand(GetRepo(), id)
			result, err := archiveCmd.Execute(ctx)
			if err != nil {
				return err
			}
			fmt.Println(result.Message)

		case domain.IDTypeCategory:
			archiveCmd := commands.NewArchiveCategoryCommand(GetRepo(), id)
			result, err := archiveCmd.Execute(ctx)
			if err != nil {
				return err
			}
			fmt.Println(result.Message)

		default:
			return fmt.Errorf("can only archive items or categories, got: %s", idType)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(archiveCmd)
}
