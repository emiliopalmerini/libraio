package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"libraio/internal/application/commands"
)

var listCmd = &cobra.Command{
	Use:   "list [scopes|areas|categories|items] [parent-id]",
	Short: "List entities in the vault",
	Long: `List scopes, areas, categories, or items in the vault.

Examples:
  libraio-cli list scopes
  libraio-cli list areas S01
  libraio-cli list categories S01.10-19
  libraio-cli list items S01.11`,
}

var listScopesCmd = &cobra.Command{
	Use:   "scopes",
	Short: "List all scopes",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		listCmd := commands.NewListScopesCommand(GetRepo())
		scopes, err := listCmd.Execute(ctx)
		if err != nil {
			return err
		}

		for _, s := range scopes {
			fmt.Printf("%s %s\n", s.ID, s.Name)
		}
		return nil
	},
}

var listAreasCmd = &cobra.Command{
	Use:   "areas <scope-id>",
	Short: "List areas in a scope",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		listCmd := commands.NewListAreasCommand(GetRepo(), args[0])
		areas, err := listCmd.Execute(ctx)
		if err != nil {
			return err
		}

		for _, a := range areas {
			fmt.Printf("%s %s\n", a.ID, a.Name)
		}
		return nil
	},
}

var listCategoriesCmd = &cobra.Command{
	Use:   "categories <area-id>",
	Short: "List categories in an area",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		listCmd := commands.NewListCategoriesCommand(GetRepo(), args[0])
		categories, err := listCmd.Execute(ctx)
		if err != nil {
			return err
		}

		for _, c := range categories {
			fmt.Printf("%s %s\n", c.ID, c.Name)
		}
		return nil
	},
}

var listItemsCmd = &cobra.Command{
	Use:   "items <category-id>",
	Short: "List items in a category",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		listCmd := commands.NewListItemsCommand(GetRepo(), args[0])
		items, err := listCmd.Execute(ctx)
		if err != nil {
			return err
		}

		for _, i := range items {
			fmt.Printf("%s %s\n", i.ID, i.Name)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.AddCommand(listScopesCmd)
	listCmd.AddCommand(listAreasCmd)
	listCmd.AddCommand(listCategoriesCmd)
	listCmd.AddCommand(listItemsCmd)
}
