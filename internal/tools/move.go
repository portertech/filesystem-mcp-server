package tools

import (
	"context"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/portertech/filesystem-mcp-server/internal/registry"
	"github.com/spf13/cast"
)

// NewMoveFileTool creates the move_file tool.
func NewMoveFileTool(reg *registry.Registry) mcp.Tool {
	return mcp.NewTool(
		"move_file",
		mcp.WithDescription("Move or rename a file or directory. Fails if destination exists."),
		mcp.WithString("source", mcp.Description("Path to the source file or directory"), mcp.Required()),
		mcp.WithString("destination", mcp.Description("Path to the destination"), mcp.Required()),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:           "Move File",
			DestructiveHint: boolPtr(true),
		}),
	)
}

// HandleMoveFile handles the move_file tool.
func HandleMoveFile(ctx context.Context, reg *registry.Registry, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	source := cast.ToString(request.Params.Arguments["source"])
	destination := cast.ToString(request.Params.Arguments["destination"])

	// Validate source path
	resolvedSrc, err := reg.Validate(source)
	if err != nil {
		return mcp.NewToolResultError(fmt.Errorf("source path validation failed: %w", err).Error()), nil
	}

	// Check source exists
	if _, err := os.Stat(resolvedSrc); err != nil {
		return mcp.NewToolResultError(fmt.Errorf("source does not exist: %w", err).Error()), nil
	}

	// Validate destination path
	resolvedDst, err := reg.ValidateForCreation(destination)
	if err != nil {
		return mcp.NewToolResultError(fmt.Errorf("destination path validation failed: %w", err).Error()), nil
	}

	// Check if destination exists
	if _, err := os.Stat(resolvedDst); err == nil {
		return mcp.NewToolResultError("destination already exists"), nil
	}

	// Move the file
	if err := os.Rename(resolvedSrc, resolvedDst); err != nil {
		return mcp.NewToolResultError(fmt.Errorf("failed to move: %w", err).Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Successfully moved %s to %s", resolvedSrc, resolvedDst)), nil
}
