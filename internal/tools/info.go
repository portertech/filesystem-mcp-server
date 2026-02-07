package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/portertech/filesystem-mcp-server/internal/registry"
	"github.com/portertech/filesystem-mcp-server/pkg/filesystem"
	"github.com/spf13/cast"
)

// NewGetFileInfoTool creates the get_file_info tool.
func NewGetFileInfoTool(reg *registry.Registry) mcp.Tool {
	return mcp.NewTool(
		"get_file_info",
		mcp.WithDescription("Get detailed metadata about a file or directory."),
		mcp.WithString("path", mcp.Description("Path to the file or directory"), mcp.Required()),
	)
}

// HandleGetFileInfo handles the get_file_info tool.
func HandleGetFileInfo(ctx context.Context, reg *registry.Registry, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path := cast.ToString(request.Params.Arguments["path"])

	resolvedPath, err := reg.Validate(path)
	if err != nil {
		return mcp.NewToolResultError(fmt.Errorf("path validation failed: %w", err).Error()), nil
	}

	info, err := os.Stat(resolvedPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Errorf("failed to stat path: %w", err).Error()), nil
	}

	fileInfo := filesystem.FileInfo{
		Size:        info.Size(),
		Modified:    info.ModTime().Format(time.RFC3339),
		IsDirectory: info.IsDir(),
		IsFile:      !info.IsDir(),
		Permissions: fmt.Sprintf("%04o", info.Mode().Perm()),
	}

	jsonResult, err := json.MarshalIndent(fileInfo, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Errorf("failed to marshal result: %w", err).Error()), nil
	}

	return mcp.NewToolResultText(string(jsonResult)), nil
}

// NewListAllowedDirectoriesTool creates the list_allowed_directories tool.
func NewListAllowedDirectoriesTool(reg *registry.Registry) mcp.Tool {
	return mcp.NewTool(
		"list_allowed_directories",
		mcp.WithDescription("List all directories that are allowed to be accessed."),
	)
}

// HandleListAllowedDirectories handles the list_allowed_directories tool.
func HandleListAllowedDirectories(ctx context.Context, reg *registry.Registry, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dirs := reg.Get()

	if len(dirs) == 0 {
		return mcp.NewToolResultText("No allowed directories configured"), nil
	}

	result := "Allowed directories:\n"
	for _, d := range dirs {
		result += fmt.Sprintf("  %s\n", d)
	}

	return mcp.NewToolResultText(result), nil
}
