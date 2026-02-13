package mcp

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"libraio/internal/domain"
	"libraio/internal/ports"
)

// RegisterReadTools adds all read-only vault tools to the MCP server.
func RegisterReadTools(s *server.MCPServer, repo ports.VaultRepository) {
	s.AddTool(listTool(), listHandler(repo))
	s.AddTool(searchTool(), searchHandler(repo))
	s.AddTool(treeTool(), treeHandler(repo))
	s.AddTool(readJDexTool(), readJDexHandler(repo))
	s.AddTool(resolvePathTool(), resolvePathHandler(repo))
}

// --- list ---

func listTool() mcp.Tool {
	return mcp.NewTool("list",
		mcp.WithDescription("List vault entities. Without arguments lists scopes. With a parent ID lists its children (scope→areas, area→categories, category→items)."),
		mcp.WithString("parent_id",
			mcp.Description("Parent JD ID to list children of (e.g. S01, S01.10-19, S01.11). Omit to list all scopes."),
		),
	)
}

func listHandler(repo ports.VaultRepository) server.ToolHandlerFunc {
	return func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		parentID := req.GetString("parent_id", "")

		if parentID == "" {
			scopes, err := repo.ListScopes()
			if err != nil {
				return toolError(err)
			}
			return formatEntities(scopes, formatScope)
		}

		switch domain.ParseIDType(parentID) {
		case domain.IDTypeScope:
			areas, err := repo.ListAreas(parentID)
			if err != nil {
				return toolError(err)
			}
			return formatEntities(areas, formatArea)

		case domain.IDTypeArea:
			categories, err := repo.ListCategories(parentID)
			if err != nil {
				return toolError(err)
			}
			return formatEntities(categories, formatCategory)

		case domain.IDTypeCategory:
			items, err := repo.ListItems(parentID)
			if err != nil {
				return toolError(err)
			}
			return formatEntities(items, formatItem)

		default:
			return toolError(fmt.Errorf("invalid parent ID: %s (expected scope, area, or category)", parentID))
		}
	}
}

// --- search ---

func searchTool() mcp.Tool {
	return mcp.NewTool("search",
		mcp.WithDescription("Search the vault by keyword. Returns matching entities with their JD IDs."),
		mcp.WithString("query",
			mcp.Description("Search query"),
			mcp.Required(),
		),
	)
}

func searchHandler(repo ports.VaultRepository) server.ToolHandlerFunc {
	return func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		query := req.GetString("query", "")
		if query == "" {
			return toolError(fmt.Errorf("query is required"))
		}

		results, err := repo.Search(query)
		if err != nil {
			return toolError(err)
		}

		if len(results) == 0 {
			return mcp.NewToolResultText("No results found."), nil
		}

		var sb strings.Builder
		for _, r := range results {
			fmt.Fprintf(&sb, "%s  %s  %s\n", r.ID, r.Name, r.MatchedText)
		}
		return mcp.NewToolResultText(sb.String()), nil
	}
}

// --- tree ---

func treeTool() mcp.Tool {
	return mcp.NewTool("tree",
		mcp.WithDescription("Display the vault structure as a tree."),
	)
}

func treeHandler(repo ports.VaultRepository) server.ToolHandlerFunc {
	return func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		root, err := repo.BuildTree()
		if err != nil {
			return toolError(err)
		}
		var sb strings.Builder
		renderTree(&sb, root, "")
		return mcp.NewToolResultText(sb.String()), nil
	}
}

func renderTree(sb *strings.Builder, node *domain.TreeNode, prefix string) {
	if node.ID != "root" {
		fmt.Fprintf(sb, "%s%s %s\n", prefix, node.ID, node.Name)
		prefix += "  "
	}
	for _, child := range node.Children {
		renderTree(sb, child, prefix)
	}
}

// --- read_jdex ---

func readJDexTool() mcp.Tool {
	return mcp.NewTool("read_jdex",
		mcp.WithDescription("Read the JDex (index) file content for an item by its JD ID."),
		mcp.WithString("id",
			mcp.Description("Item JD ID (e.g. S01.11.15)"),
			mcp.Required(),
		),
	)
}

func readJDexHandler(repo ports.VaultRepository) server.ToolHandlerFunc {
	return func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id := req.GetString("id", "")
		if id == "" {
			return toolError(fmt.Errorf("id is required"))
		}

		jdexPath, err := repo.GetJDexPath(id)
		if err != nil {
			return toolError(err)
		}

		content, err := os.ReadFile(jdexPath)
		if err != nil {
			return toolError(fmt.Errorf("reading JDex file: %w", err))
		}

		return mcp.NewToolResultText(string(content)), nil
	}
}

// --- resolve_path ---

func resolvePathTool() mcp.Tool {
	return mcp.NewTool("resolve_path",
		mcp.WithDescription("Get the filesystem path for a JD ID."),
		mcp.WithString("id",
			mcp.Description("JD ID (e.g. S01, S01.10-19, S01.11, S01.11.15)"),
			mcp.Required(),
		),
	)
}

func resolvePathHandler(repo ports.VaultRepository) server.ToolHandlerFunc {
	return func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id := req.GetString("id", "")
		if id == "" {
			return toolError(fmt.Errorf("id is required"))
		}

		path, err := repo.GetPath(id)
		if err != nil {
			return toolError(err)
		}

		return mcp.NewToolResultText(path), nil
	}
}

// --- helpers ---

func toolError(err error) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultError(err.Error()), nil
}

func formatEntities[T any](entities []T, format func(T) string) (*mcp.CallToolResult, error) {
	if len(entities) == 0 {
		return mcp.NewToolResultText("No results."), nil
	}
	var sb strings.Builder
	for _, e := range entities {
		sb.WriteString(format(e))
		sb.WriteByte('\n')
	}
	return mcp.NewToolResultText(sb.String()), nil
}

func formatScope(s domain.Scope) string {
	return fmt.Sprintf("%s  %s", s.ID, s.Name)
}

func formatArea(a domain.Area) string {
	return fmt.Sprintf("%s  %s", a.ID, a.Name)
}

func formatCategory(c domain.Category) string {
	return fmt.Sprintf("%s  %s", c.ID, c.Name)
}

func formatItem(i domain.Item) string {
	return fmt.Sprintf("%s  %s", i.ID, i.Name)
}
