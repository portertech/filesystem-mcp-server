package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

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
		return mcp.NewToolResultError(fmt.Errorf("path validation failed: %w", err).Error()), nil
	}

	if err := os.MkdirAll(resolvedPath, 0755); err != nil {
		return mcp.NewToolResultError(fmt.Errorf("failed to create directory: %w", err).Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Successfully created directory %s", resolvedPath)), nil
}

// NewListDirectoryTool creates the list_directory tool.
func NewListDirectoryTool(reg *registry.Registry) mcp.Tool {
	return mcp.NewTool(
		"list_directory",
		mcp.WithDescription("List contents of a directory with [FILE] and [DIR] prefixes."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithString("path", mcp.Description("Path to the directory to list"), mcp.Required()),
		mcp.WithString("format", mcp.Description("Output format: 'text' or 'json'")),
	)
}

// HandleListDirectory handles the list_directory tool.
func HandleListDirectory(ctx context.Context, reg *registry.Registry, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path := cast.ToString(request.Params.Arguments["path"])
	format := cast.ToString(request.Params.Arguments["format"])

	resolvedPath, err := reg.Validate(path)
	if err != nil {
		return mcp.NewToolResultError(fmt.Errorf("path validation failed: %w", err).Error()), nil
	}

	info, err := os.Stat(resolvedPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Errorf("failed to stat path: %w", err).Error()), nil
	}
	if !info.IsDir() {
		return mcp.NewToolResultError("path is not a directory"), nil
	}

	entries, err := os.ReadDir(resolvedPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Errorf("failed to read directory: %w", err).Error()), nil
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	if format == "json" {
		type listEntry struct {
			Name string `json:"name"`
			Type string `json:"type"`
		}

		listEntries := make([]listEntry, 0, len(entries))
		for _, entry := range entries {
			entryType := "file"
			if entry.IsDir() {
				entryType = "directory"
			}
			listEntries = append(listEntries, listEntry{
				Name: entry.Name(),
				Type: entryType,
			})
		}

		jsonResult, err := json.MarshalIndent(listEntries, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Errorf("failed to marshal result: %w", err).Error()), nil
		}

		return mcp.NewToolResultText(string(jsonResult)), nil
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
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithString("path", mcp.Description("Path to the directory to list"), mcp.Required()),
		mcp.WithString("sortBy", mcp.Description("Sort by 'name', 'size', or 'modified'")),
		mcp.WithString("order", mcp.Description("Sort order: 'asc' or 'desc'")),
		mcp.WithString("format", mcp.Description("Output format: 'text' or 'json'")),
	)
}

// HandleListDirectoryWithSizes handles the list_directory_with_sizes tool.
func HandleListDirectoryWithSizes(ctx context.Context, reg *registry.Registry, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path := cast.ToString(request.Params.Arguments["path"])
	sortBy := cast.ToString(request.Params.Arguments["sortBy"])
	order := cast.ToString(request.Params.Arguments["order"])
	format := cast.ToString(request.Params.Arguments["format"])

	resolvedPath, err := reg.Validate(path)
	if err != nil {
		return mcp.NewToolResultError(fmt.Errorf("path validation failed: %w", err).Error()), nil
	}

	info, err := os.Stat(resolvedPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Errorf("failed to stat path: %w", err).Error()), nil
	}
	if !info.IsDir() {
		return mcp.NewToolResultError("path is not a directory"), nil
	}

	entries, err := os.ReadDir(resolvedPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Errorf("failed to read directory: %w", err).Error()), nil
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	type fileEntry struct {
		name     string
		isDir    bool
		size     int64
		modified int64
	}

	var files []fileEntry
	var totalSize int64
	var fileCount, dirCount int

	for _, entry := range entries {
		fe := fileEntry{
			name:  entry.Name(),
			isDir: entry.IsDir(),
		}

		entryInfo, infoErr := entry.Info()
		if infoErr == nil {
			fe.modified = entryInfo.ModTime().UnixNano()
			if !entry.IsDir() {
				fe.size = entryInfo.Size()
				totalSize += fe.size
			}
		}

		if entry.IsDir() {
			dirCount++
		} else {
			fileCount++
		}

		files = append(files, fe)
	}

	if order == "" {
		order = "asc"
	}
	ascending := order != "desc"

	sort.Slice(files, func(i, j int) bool {
		if sortBy == "size" {
			if files[i].size == files[j].size {
				return files[i].name < files[j].name
			}
			if ascending {
				return files[i].size < files[j].size
			}
			return files[i].size > files[j].size
		}
		if sortBy == "modified" {
			if files[i].modified == files[j].modified {
				return files[i].name < files[j].name
			}
			if ascending {
				return files[i].modified < files[j].modified
			}
			return files[i].modified > files[j].modified
		}
		if ascending {
			return files[i].name < files[j].name
		}
		return files[i].name > files[j].name
	})

	if format == "json" {
		type listEntry struct {
			Name     string `json:"name"`
			Type     string `json:"type"`
			Size     int64  `json:"size"`
			Modified string `json:"modified"`
		}

		entries := make([]listEntry, 0, len(files))
		for _, f := range files {
			entryType := "file"
			if f.isDir {
				entryType = "directory"
			}
			entries = append(entries, listEntry{
				Name:     f.name,
				Type:     entryType,
				Size:     f.size,
				Modified: time.Unix(0, f.modified).UTC().Format(time.RFC3339),
			})
		}

		payload := map[string]any{
			"entries": entries,
			"summary": map[string]any{
				"files":         fileCount,
				"directories":   dirCount,
				"totalSize":     totalSize,
				"totalSizeText": stream.FormatSize(totalSize),
			},
		}

		jsonResult, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Errorf("failed to marshal result: %w", err).Error()), nil
		}

		return mcp.NewToolResultText(string(jsonResult)), nil
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
		mcp.WithReadOnlyHintAnnotation(true),
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
		return mcp.NewToolResultError(fmt.Errorf("path validation failed: %w", err).Error()), nil
	}

	info, err := os.Stat(resolvedPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Errorf("failed to stat path: %w", err).Error()), nil
	}
	if !info.IsDir() {
		return mcp.NewToolResultError("path is not a directory"), nil
	}

	// Compile exclude patterns
	var excludeGlobs []glob.Glob
	for _, pattern := range excludePatterns {
		g, err := glob.Compile(pattern)
		if err != nil {
			return mcp.NewToolResultError(fmt.Errorf("invalid exclude pattern %q: %w", pattern, err).Error()), nil
		}
		excludeGlobs = append(excludeGlobs, g)
	}

	tree, err := buildTree(resolvedPath, excludeGlobs)
	if err != nil {
		return mcp.NewToolResultError(fmt.Errorf("failed to build tree: %w", err).Error()), nil
	}

	jsonResult, err := json.MarshalIndent(tree, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Errorf("failed to marshal tree: %w", err).Error()), nil
	}

	return mcp.NewToolResultText(string(jsonResult)), nil
}

// buildTree recursively builds a directory tree.
func buildTree(path string, excludeGlobs []glob.Glob) (*filesystem.TreeEntry, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	name := filepath.Base(path)

	// Check exclusions
	for _, g := range excludeGlobs {
		if g.Match(name) {
			return nil, nil
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
			return nil, err
		}

		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Name() < entries[j].Name()
		})

		for _, e := range entries {
			childPath := filepath.Join(path, e.Name())
			child, err := buildTree(childPath, excludeGlobs)
			if err != nil {
				return nil, err
			}
			if child != nil {
				entry.Children = append(entry.Children, child)
			}
		}
	} else {
		entry.Type = "file"
	}

	return entry, nil
}
