package main

import (
	"context"
	"flag"
	"log"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"libraio/internal/adapters/filesystem"
	mcpadapter "libraio/internal/adapters/mcp"
	"libraio/internal/config"
)

func main() {
	vaultFlag := flag.String("vault", config.VaultPath(), "path to the vault")
	flag.Parse()

	repo := filesystem.NewRepository(*vaultFlag)

	mcpServer := server.NewMCPServer(
		"libraio-mcp",
		"0.1.0",
		server.WithToolCapabilities(true),
	)

	mcpServer.AddTool(
		mcp.NewTool("ping",
			mcp.WithDescription("Health check — returns pong"),
		),
		func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText("pong"), nil
		},
	)

	mcpadapter.RegisterReadTools(mcpServer, repo)
	mcpadapter.RegisterWriteTools(mcpServer, repo)

	if err := server.ServeStdio(mcpServer); err != nil {
		log.Fatalf("libraio-mcp: %v", err)
	}
}
