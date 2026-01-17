package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"libraio/internal/application"
	"libraio/internal/application/commands"
)

var moveCmd = &cobra.Command{
	Use:   "move <source-id> <dest-id>",
	Short: "Move an item or category",
	Long: `Move an item to a different category, or a category to a different area.

Rules:
- Items can only be moved to categories
- Categories can only be moved to areas

Examples:
  libraio-cli move S01.11.15 S01.12      # Move item to category
  libraio-cli move S01.11 S01.20-29      # Move category to area`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		sourceID := args[0]
		destID := args[1]
		ctx := context.Background()

		sourceType := application.ParseIDType(sourceID)

		switch sourceType {
		case application.IDTypeItem:
			moveCmd := commands.NewMoveItemCommand(GetRepo(), sourceID, destID)
			result, err := moveCmd.Execute(ctx)
			if err != nil {
				return err
			}
			fmt.Println(result.Message)

		case application.IDTypeCategory:
			moveCmd := commands.NewMoveCategoryCommand(GetRepo(), sourceID, destID)
			result, err := moveCmd.Execute(ctx)
			if err != nil {
				return err
			}
			fmt.Println(result.Message)

		default:
			return fmt.Errorf("can only move items or categories, got: %s", sourceType)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(moveCmd)
}
