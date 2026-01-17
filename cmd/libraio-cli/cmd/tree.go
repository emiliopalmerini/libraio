package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"libraio/internal/application"
	"libraio/internal/application/commands"
)

var treeCmd = &cobra.Command{
	Use:   "tree",
	Short: "Display the vault tree structure",
	Long: `Display the complete tree structure of the vault.

Example:
  libraio-cli tree`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		buildCmd := commands.NewBuildTreeCommand(GetRepo())
		root, err := buildCmd.Execute(ctx)
		if err != nil {
			return err
		}

		// Load and print tree recursively
		printTree(root, 0)
		return nil
	},
}

func printTree(node *application.TreeNode, depth int) {
	if node == nil {
		return
	}

	// Load children if not loaded
	if len(node.Children) == 0 && node.Type != application.IDTypeItem {
		GetRepo().LoadChildren(node)
	}

	indent := strings.Repeat("  ", depth)
	fmt.Printf("%s%s %s\n", indent, node.ID, node.Name)

	for _, child := range node.Children {
		printTree(child, depth+1)
	}
}

func init() {
	rootCmd.AddCommand(treeCmd)
}
