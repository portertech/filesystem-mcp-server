package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/gobwas/glob"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/portertech/filesystem-mcp-server/internal/registry"
	"github.com/portertech/filesystem-mcp-server/internal/stream"
	"github.com/portertech/filesystem-mcp-server/pkg/filesystem"
	"github.com/spf13/cast"
)

// NewCreateDirectoryTool creates the create_directory tool.
func NewCreateDirectoryTool(reg *registry.Registry) mcp.Tool {
	return mcp.NewTool(
		"create_directory",
		mcp.WithDescription("Create a directory, including any necessary parent directories."),
		mcp.WithString("path", mcp.Description("Path to the directory to create"), mcp.Required()),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:          "Create Directory",
			IdempotentHint: boolPtr(true),
		}),
	)
}

// HandleCreateDirectory handles the create_directory tool.
func HandleCreateDirectory(ctx context.Context, reg *registry.Registry, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path := cast.ToString(request.Params.Arguments["path"])

	resolvedPath, err := reg.ValidateForCreation(path)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("path validation failed: %v", err)), nil
	}

	if err := os.MkdirAll(resolvedPath, 0755); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create directory: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Successfully created directory %s", resolvedPath)), nil
}

// NewListDirectoryTool creates the list_directory tool.
func NewListDirectoryTool(reg *registry.Registry) mcp.Tool {
	return mcp.NewTool(
		"list_directory",
		mcp.WithDescription("List contents of a directory with [FILE] and [DIR] prefixes."),
		mcp.WithString("path", mcp.Description("Path to the directory to list"), mcp.Required()),
	)
}

// HandleListDirectory handles the list_directory tool.
func HandleListDirectory(ctx context.Context, reg *registry.Registry, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path := cast.ToString(request.Params.Arguments["path"])

	resolvedPath, err := reg.Validate(path)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("path validation failed: %v", err)), nil
	}

	info, err := os.Stat(resolvedPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to stat path: %v", err)), nil
	}
	if !info.IsDir() {
		return mcp.NewToolResultError("path is not a directory"), nil
	}

	entries, err := os.ReadDir(resolvedPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to read directory: %v", err)), nil
	}

	var result string
	for _, entry := range entries {
		prefix := "[FILE]"
		if entry.IsDir() {
			prefix = "[DIR]"
		}
		result += fmt.Sprintf("%s %s\n", prefix, entry.Name())
	}

	return mcp.NewToolResultText(result), nil
}

// NewListDirectoryWithSizesTool creates the list_directory_with_sizes tool.
func NewListDirectoryWithSizesTool(reg *registry.Registry) mcp.Tool {
	return mcp.NewTool(
		"list_directory_with_sizes",
		mcp.WithDescription("List directory contents with file sizes in human-readable format."),
		mcp.WithString("path", mcp.Description("Path to the directory to list"), mcp.Required()),
		mcp.WithString("sortBy", mcp.Description("Sort by 'name' or 'size'")),
	)
}

// HandleListDirectoryWithSizes handles the list_directory_with_sizes tool.
func HandleListDirectoryWithSizes(ctx context.Context, reg *registry.Registry, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path := cast.ToString(request.Params.Arguments["path"])
	sortBy := cast.ToString(request.Params.Arguments["sortBy"])

	resolvedPath, err := reg.Validate(path)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("path validation failed: %v", err)), nil
	}

	info, err := os.Stat(resolvedPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to stat path: %v", err)), nil
	}
	if !info.IsDir() {
		return mcp.NewToolResultError("path is not a directory"), nil
	}

	entries, err := os.ReadDir(resolvedPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to read directory: %v", err)), nil
	}

	type fileEntry struct {
		name  string
		isDir bool
		size  int64
	}

	var files []fileEntry
	var totalSize int64
	var fileCount, dirCount int

	for _, entry := range entries {
		fe := fileEntry{
			name:  entry.Name(),
			isDir: entry.IsDir(),
		}

		if entry.IsDir() {
			dirCount++
		} else {
			fileCount++
			if entryInfo, err := entry.Info(); err == nil {
				fe.size = entryInfo.Size()
				totalSize += fe.size
			}
		}

		files = append(files, fe)
	}

	// Sort
	if sortBy == "size" {
		sort.Slice(files, func(i, j int) bool {
			return files[i].size > files[j].size
		})
	} else {
		sort.Slice(files, func(i, j int) bool {
			return files[i].name < files[j].name
		})
	}

	var result string
	for _, f := range files {
		if f.isDir {
			result += fmt.Sprintf("[DIR]  %s\n", f.name)
		} else {
			result += fmt.Sprintf("[FILE] %s (%s)\n", f.name, stream.FormatSize(f.size))
		}
	}

	result += fmt.Sprintf("\nSummary: %d files, %d directories, Total: %s\n",
		fileCount, dirCount, stream.FormatSize(totalSize))

	return mcp.NewToolResultText(result), nil
}

// NewDirectoryTreeTool creates the directory_tree tool.
func NewDirectoryTreeTool(reg *registry.Registry) mcp.Tool {
	return mcp.NewTool(
		"directory_tree",
		mcp.WithDescription("Get a recursive tree view of files and directories as JSON."),
		mcp.WithString("path", mcp.Description("Path to the root directory"), mcp.Required()),
		mcp.WithArray("excludePatterns", mcp.Description("Glob patterns to exclude")),
	)
}

// HandleDirectoryTree handles the directory_tree tool.
func HandleDirectoryTree(ctx context.Context, reg *registry.Registry, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path := cast.ToString(request.Params.Arguments["path"])

	var excludePatterns []string
	if patternsArg, ok := request.Params.Arguments["excludePatterns"].([]interface{}); ok {
		for _, p := range patternsArg {
			excludePatterns = append(excludePatterns, cast.ToString(p))
		}
	}

	resolvedPath, err := reg.Validate(path)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("path validation failed: %v", err)), nil
	}

	info, err := os.Stat(resolvedPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to stat path: %v", err)), nil
	}
	if !info.IsDir() {
		return mcp.NewToolResultError("path is not a directory"), nil
	}

	// Compile exclude patterns
	var excludeGlobs []glob.Glob
	for _, pattern := range excludePatterns {
		g, err := glob.Compile(pattern)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid exclude pattern %q: %v", pattern, err)), nil
		}
		excludeGlobs = append(excludeGlobs, g)
	}

	tree := buildTree(resolvedPath, excludeGlobs)

	jsonResult, err := json.MarshalIndent(tree, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal tree: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonResult)), nil
}

// buildTree recursively builds a directory tree.
func buildTree(path string, excludeGlobs []glob.Glob) *filesystem.TreeEntry {
	info, err := os.Stat(path)
	if err != nil {
		return nil
	}

	name := filepath.Base(path)

	// Check exclusions
	for _, g := range excludeGlobs {
		if g.Match(name) {
			return nil
		}
	}

	entry := &filesystem.TreeEntry{
		Name: name,
	}

	if info.IsDir() {
		entry.Type = "directory"
		entry.Children = []*filesystem.TreeEntry{}

		entries, err := os.ReadDir(path)
		if err != nil {
			return entry
		}

		for _, e := range entries {
			childPath := filepath.Join(path, e.Name())
			child := buildTree(childPath, excludeGlobs)
			if child != nil {
				entry.Children = append(entry.Children, child)
			}
		}
	} else {
		entry.Type = "file"
	}

	return entry
}
