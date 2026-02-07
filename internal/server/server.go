package server

import (
	"context"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/portertech/filesystem-mcp-server/internal/registry"
	"github.com/portertech/filesystem-mcp-server/internal/tools"
)

// Server wraps the MCP server with filesystem tools.
type Server struct {
	mcpServer *server.MCPServer
	registry  *registry.Registry
	logger    *slog.Logger
}

// New creates a new filesystem MCP server.
func New(reg *registry.Registry, logger *slog.Logger) *Server {
	s := &Server{
		registry: reg,
		logger:   logger,
	}

	mcpServer := server.NewMCPServer(
		"filesystem-mcp-server",
		"1.0.0",
		server.WithLogging(),
	)

	s.mcpServer = mcpServer
	s.registerTools()

	return s
}

// registerTools registers all filesystem tools with the MCP server.
func (s *Server) registerTools() {
	// Read tools
	s.mcpServer.AddTool(
		tools.NewReadTextFileTool(s.registry),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return tools.HandleReadTextFile(ctx, s.registry, req)
		},
	)

	s.mcpServer.AddTool(
		tools.NewReadFileTool(s.registry),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return tools.HandleReadFile(ctx, s.registry, req)
		},
	)

	s.mcpServer.AddTool(
		tools.NewReadMultipleFilesTool(s.registry),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return tools.HandleReadMultipleFiles(ctx, s.registry, req)
		},
	)

	s.mcpServer.AddTool(
		tools.NewReadMediaFileTool(s.registry),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return tools.HandleReadMediaFile(ctx, s.registry, req)
		},
	)

	// Write tools
	s.mcpServer.AddTool(
		tools.NewWriteFileTool(s.registry),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return tools.HandleWriteFile(ctx, s.registry, req)
		},
	)

	s.mcpServer.AddTool(
		tools.NewEditFileTool(s.registry),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return tools.HandleEditFile(ctx, s.registry, req)
		},
	)

	// Copy tool
	s.mcpServer.AddTool(
		tools.NewCopyFileTool(s.registry),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return tools.HandleCopyFile(ctx, s.registry, req)
		},
	)

	// Delete tools
	s.mcpServer.AddTool(
		tools.NewDeleteFileTool(s.registry),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return tools.HandleDeleteFile(ctx, s.registry, req)
		},
	)

	s.mcpServer.AddTool(
		tools.NewDeleteDirectoryTool(s.registry),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return tools.HandleDeleteDirectory(ctx, s.registry, req)
		},
	)

	// Directory tools
	s.mcpServer.AddTool(
		tools.NewCreateDirectoryTool(s.registry),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return tools.HandleCreateDirectory(ctx, s.registry, req)
		},
	)

	s.mcpServer.AddTool(
		tools.NewListDirectoryTool(s.registry),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return tools.HandleListDirectory(ctx, s.registry, req)
		},
	)

	s.mcpServer.AddTool(
		tools.NewListDirectoryWithSizesTool(s.registry),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return tools.HandleListDirectoryWithSizes(ctx, s.registry, req)
		},
	)

	s.mcpServer.AddTool(
		tools.NewDirectoryTreeTool(s.registry),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return tools.HandleDirectoryTree(ctx, s.registry, req)
		},
	)

	// Move tool
	s.mcpServer.AddTool(
		tools.NewMoveFileTool(s.registry),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return tools.HandleMoveFile(ctx, s.registry, req)
		},
	)

	// Search tool
	s.mcpServer.AddTool(
		tools.NewSearchFilesTool(s.registry),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return tools.HandleSearchFiles(ctx, s.registry, req)
		},
	)

	// Info tools
	s.mcpServer.AddTool(
		tools.NewGetFileInfoTool(s.registry),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return tools.HandleGetFileInfo(ctx, s.registry, req)
		},
	)

	s.mcpServer.AddTool(
		tools.NewListAllowedDirectoriesTool(s.registry),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return tools.HandleListAllowedDirectories(ctx, s.registry, req)
		},
	)

	s.logger.Info("registered tools", "count", 17)
}

// Run starts the server with stdio transport.
func (s *Server) Run(ctx context.Context) error {
	s.logger.Info("starting filesystem MCP server")
	return server.ServeStdio(s.mcpServer)
}

// GetMCPServer returns the underlying MCP server for testing.
func (s *Server) GetMCPServer() *server.MCPServer {
	return s.mcpServer
}
