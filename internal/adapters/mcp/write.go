package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"libraio/internal/application/commands"
	"libraio/internal/domain"
	"libraio/internal/ports"
)

// RegisterWriteTools adds all write vault tools to the MCP server.
func RegisterWriteTools(s *server.MCPServer, repo ports.VaultRepository) {
	s.AddTool(createTool(), createHandler(repo))
	s.AddTool(moveTool(), moveHandler(repo))
	s.AddTool(renameTool(), renameHandler(repo))
	s.AddTool(archiveTool(), archiveHandler(repo))
	s.AddTool(unarchiveTool(), unarchiveHandler(repo))
	s.AddTool(deleteTool(), deleteHandler(repo))
}

// --- create ---

func createTool() mcp.Tool {
	return mcp.NewTool("create",
		mcp.WithDescription("Create a new entity in the vault. Auto-detects the type from parent: no parent→scope, scope→area, area→category, category→item."),
		mcp.WithString("parent_id",
			mcp.Description("Parent JD ID. Omit to create a scope."),
		),
		mcp.WithString("description",
			mcp.Description("Name/description for the new entity"),
			mcp.Required(),
		),
	)
}

func createHandler(repo ports.VaultRepository) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		parentID := req.GetString("parent_id", "")
		description := req.GetString("description", "")

		factory := commands.NewCreateCommandFactory(repo)
		result, err := factory.Execute(ctx, parentID, description)
		if err != nil {
			return toolError(err)
		}

		return mcp.NewToolResultText(result.Message), nil
	}
}

// --- move ---

func moveTool() mcp.Tool {
	return mcp.NewTool("move",
		mcp.WithDescription("Move an item to a different category, or a category to a different area."),
		mcp.WithString("source_id",
			mcp.Description("JD ID of the entity to move (item or category)"),
			mcp.Required(),
		),
		mcp.WithString("destination_id",
			mcp.Description("JD ID of the destination (category for items, area for categories)"),
			mcp.Required(),
		),
	)
}

func moveHandler(repo ports.VaultRepository) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		srcID := req.GetString("source_id", "")
		dstID := req.GetString("destination_id", "")

		srcType := domain.ParseIDType(srcID)
		switch srcType {
		case domain.IDTypeItem:
			cmd := commands.NewMoveItemCommand(repo, srcID, dstID)
			result, err := cmd.Execute(ctx)
			if err != nil {
				return toolError(err)
			}
			return mcp.NewToolResultText(result.Message), nil

		case domain.IDTypeCategory:
			cmd := commands.NewMoveCategoryCommand(repo, srcID, dstID)
			result, err := cmd.Execute(ctx)
			if err != nil {
				return toolError(err)
			}
			return mcp.NewToolResultText(result.Message), nil

		default:
			return toolError(fmt.Errorf("can only move items or categories, got: %s", srcType))
		}
	}
}

// --- rename ---

func renameTool() mcp.Tool {
	return mcp.NewTool("rename",
		mcp.WithDescription("Rename an item, category, or area."),
		mcp.WithString("id",
			mcp.Description("JD ID of the entity to rename"),
			mcp.Required(),
		),
		mcp.WithString("new_description",
			mcp.Description("New name/description"),
			mcp.Required(),
		),
	)
}

func renameHandler(repo ports.VaultRepository) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id := req.GetString("id", "")
		newDesc := req.GetString("new_description", "")

		cmd := commands.NewRenameCommand(repo, id, newDesc)
		result, err := cmd.Execute(ctx)
		if err != nil {
			return toolError(err)
		}

		return mcp.NewToolResultText(result.Message), nil
	}
}

// --- archive ---

func archiveTool() mcp.Tool {
	return mcp.NewTool("archive",
		mcp.WithDescription("Archive an item or category. Items are moved to the category's .09 archive. Categories are moved to the area's archive."),
		mcp.WithString("id",
			mcp.Description("JD ID of the item or category to archive"),
			mcp.Required(),
		),
	)
}

func archiveHandler(repo ports.VaultRepository) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id := req.GetString("id", "")

		idType := domain.ParseIDType(id)
		switch idType {
		case domain.IDTypeItem:
			cmd := commands.NewArchiveItemCommand(repo, id)
			result, err := cmd.Execute(ctx)
			if err != nil {
				return toolError(err)
			}
			return mcp.NewToolResultText(result.Message), nil

		case domain.IDTypeCategory:
			cmd := commands.NewArchiveCategoryCommand(repo, id)
			result, err := cmd.Execute(ctx)
			if err != nil {
				return toolError(err)
			}
			return mcp.NewToolResultText(result.Message), nil

		default:
			return toolError(fmt.Errorf("can only archive items or categories, got: %s", idType))
		}
	}
}

// --- unarchive ---

func unarchiveTool() mcp.Tool {
	return mcp.NewTool("unarchive",
		mcp.WithDescription("Restore items from an archive (.09) item back to its category."),
		mcp.WithString("id",
			mcp.Description("JD ID of the archive item (e.g. S01.11.09)"),
			mcp.Required(),
		),
	)
}

func unarchiveHandler(repo ports.VaultRepository) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id := req.GetString("id", "")

		cmd := commands.NewUnarchiveItemCommand(repo, id)
		result, err := cmd.Execute(ctx)
		if err != nil {
			return toolError(err)
		}

		return mcp.NewToolResultText(result.Message), nil
	}
}

// --- delete ---

func deleteTool() mcp.Tool {
	return mcp.NewTool("delete",
		mcp.WithDescription("Delete an entity from the vault by its JD ID."),
		mcp.WithString("id",
			mcp.Description("JD ID of the entity to delete"),
			mcp.Required(),
		),
	)
}

func deleteHandler(repo ports.VaultRepository) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id := req.GetString("id", "")

		cmd := commands.NewDeleteCommand(repo, id)
		result, err := cmd.Execute(ctx)
		if err != nil {
			return toolError(err)
		}

		return mcp.NewToolResultText(result.Message), nil
	}
}
