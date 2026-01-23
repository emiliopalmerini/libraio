package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"libraio/internal/application"
	"libraio/internal/application/commands"
)

var createCmd = &cobra.Command{
	Use:   "create <parent-id> <description>",
	Short: "Create a new item or category",
	Long: `Create a new item or category in the vault.

The type of entity created depends on the parent:
- Scope parent (e.g., S01) creates an area
- Area parent (e.g., S01.10-19) creates a category
- Category parent (e.g., S01.11) creates an item

Examples:
  libraio-cli create S01 "New Area"
  libraio-cli create S01.10-19 "New Category"
  libraio-cli create S01.11 "New Item"`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		parentID := args[0]
		description := args[1]
		ctx := context.Background()

		parentType := application.ParseIDType(parentID)

		switch parentType {
		case application.IDTypeScope:
			createCmd := commands.NewCreateAreaCommand(GetRepo(), parentID, description)
			result, err := createCmd.Execute(ctx)
			if err != nil {
				return err
			}
			fmt.Println(result.Message)

		case application.IDTypeArea:
			createCmd := commands.NewCreateCategoryCommand(GetRepo(), parentID, description)
			result, err := createCmd.Execute(ctx)
			if err != nil {
				return err
			}
			fmt.Println(result.Message)

		case application.IDTypeCategory:
			createCmd := commands.NewCreateItemCommand(GetRepo(), parentID, description)
			result, err := createCmd.Execute(ctx)
			if err != nil {
				return err
			}
			fmt.Println(result.Message)

		default:
			return fmt.Errorf("invalid parent type: %s (expected scope, area, or category)", parentType)
		}

		return nil
	},
}

var createScopeCmd = &cobra.Command{
	Use:   "scope <description>",
	Short: "Create a new scope",
	Long: `Create a new scope in the vault.

Examples:
  libraio-cli create scope "Me"
  libraio-cli create scope "Work"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		description := args[0]
		ctx := context.Background()

		createCmd := commands.NewCreateScopeCommand(GetRepo(), description)
		result, err := createCmd.Execute(ctx)
		if err != nil {
			return err
		}
		fmt.Println(result.Message)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(createCmd)
	createCmd.AddCommand(createScopeCmd)
}
