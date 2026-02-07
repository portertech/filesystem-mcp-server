package tools

import (
	"context"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/portertech/filesystem-mcp-server/internal/registry"
	"github.com/portertech/filesystem-mcp-server/internal/security"
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
		return mcp.NewToolResultError(fmt.Errorf("source path validation failed: %w", err).Error()), nil
	}

	// Check source exists and is a file
	srcInfo, err := os.Stat(resolvedSrc)
	if err != nil {
		return mcp.NewToolResultError(fmt.Errorf("failed to stat source: %w", err).Error()), nil
	}
	if srcInfo.IsDir() {
		return mcp.NewToolResultError("source is a directory, not a file"), nil
	}

	// Validate destination path
	resolvedDst, err := reg.ValidateForCreation(destination)
	if err != nil {
		return mcp.NewToolResultError(fmt.Errorf("destination path validation failed: %w", err).Error()), nil
	}

	if err := security.ValidateNoSymlinksInPath(destination, reg.Get()); err != nil {
		return mcp.NewToolResultError(fmt.Errorf("destination path validation failed: %w", err).Error()), nil
	}

	// Check if destination exists
	if _, err := os.Lstat(resolvedDst); err == nil {
		if !overwrite {
			return mcp.NewToolResultError("destination already exists, set overwrite=true to replace"), nil
		}
		if err := ensureNoSymlink(resolvedDst); err != nil {
			return mcp.NewToolResultError(fmt.Errorf("destination path validation failed: %w", err).Error()), nil
		}
	}

	// Copy the file
	if err := stream.CopyFileStreaming(resolvedSrc, resolvedDst); err != nil {
		return mcp.NewToolResultError(fmt.Errorf("failed to copy file: %w", err).Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Successfully copied %s to %s", resolvedSrc, resolvedDst)), nil
}
