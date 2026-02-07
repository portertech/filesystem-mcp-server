package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gobwas/glob"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/portertech/filesystem-mcp-server/internal/registry"
	"github.com/spf13/cast"
)

// NewSearchFilesTool creates the search_files tool.
func NewSearchFilesTool(reg *registry.Registry) mcp.Tool {
	return mcp.NewTool(
		"search_files",
		mcp.WithDescription("Recursively search for files matching a glob pattern."),
		mcp.WithString("path", mcp.Description("Starting directory for the search"), mcp.Required()),
		mcp.WithString("pattern", mcp.Description("Glob pattern to match file names"), mcp.Required()),
		mcp.WithArray("excludePatterns", mcp.Description("Glob patterns to exclude")),
	)
}

// HandleSearchFiles handles the search_files tool.
func HandleSearchFiles(ctx context.Context, reg *registry.Registry, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path := cast.ToString(request.Params.Arguments["path"])
	pattern := cast.ToString(request.Params.Arguments["pattern"])

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

	// Compile match pattern
	matchGlob, err := glob.Compile(pattern)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid pattern %q: %v", pattern, err)), nil
	}

	// Compile exclude patterns
	var excludeGlobs []glob.Glob
	for _, p := range excludePatterns {
		g, err := glob.Compile(p)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid exclude pattern %q: %v", p, err)), nil
		}
		excludeGlobs = append(excludeGlobs, g)
	}

	var matches []string

	err = filepath.Walk(resolvedPath, func(walkPath string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue on errors
		}

		name := info.Name()

		// Check exclusions
		for _, g := range excludeGlobs {
			if g.Match(name) {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		// Check match
		if matchGlob.Match(name) {
			matches = append(matches, walkPath)
		}

		return nil
	})

	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
	}

	if len(matches) == 0 {
		return mcp.NewToolResultText("No matches found"), nil
	}

	result := ""
	for _, m := range matches {
		result += m + "\n"
	}

	return mcp.NewToolResultText(result), nil
}
