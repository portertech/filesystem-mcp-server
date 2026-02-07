package tools

import (
	"context"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/portertech/filesystem-mcp-server/internal/registry"
	"github.com/portertech/filesystem-mcp-server/internal/stream"
	"github.com/spf13/cast"
)

// NewCopyFileTool creates the copy_file tool.
func NewCopyFileTool(reg *registry.Registry) mcp.Tool {
	return mcp.NewTool(
		"copy_file",
		mcp.WithDescription("Copy a file to a new location. Uses streaming for memory-efficient copying of large files."),
		mcp.WithString("source", mcp.Description("Path to the source file"), mcp.Required()),
		mcp.WithString("destination", mcp.Description("Path to the destination file"), mcp.Required()),
		mcp.WithBoolean("overwrite", mcp.Description("If true, overwrite existing destination file")),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:           "Copy File",
			ReadOnlyHint:    boolPtr(false),
			IdempotentHint:  boolPtr(false),
			DestructiveHint: boolPtr(true),
		}),
	)
}

// HandleCopyFile handles the copy_file tool.
func HandleCopyFile(ctx context.Context, reg *registry.Registry, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	source := cast.ToString(request.Params.Arguments["source"])
	destination := cast.ToString(request.Params.Arguments["destination"])
	overwrite := cast.ToBool(request.Params.Arguments["overwrite"])

	// Validate source path
	resolvedSrc, err := reg.Validate(source)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("source path validation failed: %v", err)), nil
	}

	// Check source exists and is a file
	srcInfo, err := os.Stat(resolvedSrc)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to stat source: %v", err)), nil
	}
	if srcInfo.IsDir() {
		return mcp.NewToolResultError("source is a directory, not a file"), nil
	}

	// Validate destination path
	resolvedDst, err := reg.ValidateForCreation(destination)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("destination path validation failed: %v", err)), nil
	}

	// Check if destination exists
	if _, err := os.Stat(resolvedDst); err == nil {
		if !overwrite {
			return mcp.NewToolResultError("destination already exists, set overwrite=true to replace"), nil
		}
	}

	// Copy the file
	if err := stream.CopyFileStreaming(resolvedSrc, resolvedDst); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to copy file: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Successfully copied %s to %s", resolvedSrc, resolvedDst)), nil
}
