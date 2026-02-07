package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/portertech/filesystem-mcp-server/internal/registry"
	"github.com/portertech/filesystem-mcp-server/internal/security"
	"github.com/spf13/cast"
)

// NewDeleteFileTool creates the delete_file tool.
func NewDeleteFileTool(reg *registry.Registry) mcp.Tool {
	return mcp.NewTool(
		"delete_file",
		mcp.WithDescription("Delete a file. Cannot delete directories (use delete_directory instead)."),
		mcp.WithString("path", mcp.Description("Path to the file to delete"), mcp.Required()),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:           "Delete File",
			ReadOnlyHint:    boolPtr(false),
			DestructiveHint: boolPtr(true),
			IdempotentHint:  boolPtr(false),
		}),
	)
}

// HandleDeleteFile handles the delete_file tool.
func HandleDeleteFile(ctx context.Context, reg *registry.Registry, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path := cast.ToString(request.Params.Arguments["path"])

	resolvedPath, err := reg.Validate(path)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("path validation failed: %v", err)), nil
	}

	// Check if path exists and is a file
	info, err := os.Stat(resolvedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return mcp.NewToolResultError("file does not exist"), nil
		}
		return mcp.NewToolResultError(fmt.Sprintf("failed to stat path: %v", err)), nil
	}

	if info.IsDir() {
		return mcp.NewToolResultError("path is a directory, use delete_directory instead"), nil
	}

	if err := os.Remove(resolvedPath); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to delete file: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Successfully deleted %s", resolvedPath)), nil
}

// NewDeleteDirectoryTool creates the delete_directory tool.
func NewDeleteDirectoryTool(reg *registry.Registry) mcp.Tool {
	return mcp.NewTool(
		"delete_directory",
		mcp.WithDescription("Delete a directory. Requires recursive=true for non-empty directories."),
		mcp.WithString("path", mcp.Description("Path to the directory to delete"), mcp.Required()),
		mcp.WithBoolean("recursive", mcp.Description("If true, delete directory and all contents")),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:           "Delete Directory",
			ReadOnlyHint:    boolPtr(false),
			DestructiveHint: boolPtr(true),
			IdempotentHint:  boolPtr(false),
		}),
	)
}

// HandleDeleteDirectory handles the delete_directory tool.
func HandleDeleteDirectory(ctx context.Context, reg *registry.Registry, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path := cast.ToString(request.Params.Arguments["path"])
	recursive := cast.ToBool(request.Params.Arguments["recursive"])

	resolvedPath, err := reg.Validate(path)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("path validation failed: %v", err)), nil
	}

	// Check if path exists and is a directory
	info, err := os.Stat(resolvedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return mcp.NewToolResultError("directory does not exist"), nil
		}
		return mcp.NewToolResultError(fmt.Sprintf("failed to stat path: %v", err)), nil
	}

	if !info.IsDir() {
		return mcp.NewToolResultError("path is not a directory, use delete_file instead"), nil
	}

	// Prevent deleting an allowed root directory
	// Resolve allowed dirs to handle symlinks (e.g., /var -> /private/var on macOS)
	allowedDirs := reg.Get()
	cleanPath := filepath.Clean(resolvedPath)
	for _, allowed := range allowedDirs {
		resolvedAllowed := allowed
		if r, err := filepath.EvalSymlinks(allowed); err == nil {
			resolvedAllowed = r
		}
		if filepath.Clean(resolvedAllowed) == cleanPath {
			return mcp.NewToolResultError("cannot delete an allowed root directory"), nil
		}
	}

	if recursive {
		// Extra safety check: ensure we're not recursively deleting anything that
		// contains an allowed directory
		for _, allowed := range allowedDirs {
			resolvedAllowed := allowed
			if r, err := filepath.EvalSymlinks(allowed); err == nil {
				resolvedAllowed = r
			}
			if security.IsPathWithinAllowedDirectories(resolvedAllowed, []string{resolvedPath}) {
				return mcp.NewToolResultError("cannot recursively delete a directory containing an allowed directory"), nil
			}
		}

		if err := os.RemoveAll(resolvedPath); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to delete directory: %v", err)), nil
		}
	} else {
		// Non-recursive: only works on empty directories
		if err := os.Remove(resolvedPath); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to delete directory (may not be empty, use recursive=true): %v", err)), nil
		}
	}

	return mcp.NewToolResultText(fmt.Sprintf("Successfully deleted %s", resolvedPath)), nil
}

// boolPtr returns a pointer to a bool.
func boolPtr(b bool) *bool {
	return &b
}
