package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithString("path", mcp.Description("Starting directory for the search"), mcp.Required()),
		mcp.WithString("pattern", mcp.Description("Glob pattern to match file names"), mcp.Required()),
		mcp.WithArray("excludePatterns", mcp.Description("Glob patterns to exclude"), mcp.Items(map[string]any{"type": "string"})),
		mcp.WithString("format", mcp.Description("Output format: 'text' or 'json'")),
	)
}

// HandleSearchFiles handles the search_files tool.
func HandleSearchFiles(ctx context.Context, reg *registry.Registry, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path := cast.ToString(request.Params.Arguments["path"])
	pattern := cast.ToString(request.Params.Arguments["pattern"])
	format := cast.ToString(request.Params.Arguments["format"])

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

	// Compile match pattern
	matchGlobs, err := compileGlobs(pattern)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid pattern %q: %v", pattern, err)), nil
	}

	// Compile exclude patterns
	var excludeGlobs []glob.Glob
	for _, p := range excludePatterns {
		globs, err := compileGlobs(p)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid exclude pattern %q: %v", p, err)), nil
		}
		excludeGlobs = append(excludeGlobs, globs...)
	}

	var matches []string

	err = filepath.Walk(resolvedPath, func(walkPath string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue on errors
		}

		relPath, relErr := filepath.Rel(resolvedPath, walkPath)
		if relErr != nil {
			return nil
		}
		relPath = filepath.ToSlash(relPath)

		// Check exclusions
		for _, g := range excludeGlobs {
			if g.Match(relPath) {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		// Check match
		if relPath != "." && matchesAny(matchGlobs, relPath) {
			matches = append(matches, walkPath)
		}

		return nil
	})

	if err != nil {
		return mcp.NewToolResultError(fmt.Errorf("search failed: %w", err).Error()), nil
	}

	if format == "json" {
		jsonResult, err := json.MarshalIndent(matches, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Errorf("failed to marshal result: %w", err).Error()), nil
		}
		return mcp.NewToolResultText(string(jsonResult)), nil
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

func compileGlobs(pattern string) ([]glob.Glob, error) {
	patterns := []string{pattern}
	if strings.Contains(pattern, "**/") {
		patterns = append(patterns, strings.ReplaceAll(pattern, "**/", ""))
	}
	if strings.HasPrefix(pattern, "**") {
		patterns = append(patterns, strings.TrimPrefix(pattern, "**"))
	}
	if strings.HasPrefix(pattern, "**/") {
		patterns = append(patterns, strings.TrimPrefix(pattern, "**/"))
	}

	globs := make([]glob.Glob, 0, len(patterns))
	for _, p := range patterns {
		g, err := glob.Compile(p)
		if err != nil {
			return nil, err
		}
		globs = append(globs, g)
	}
	return globs, nil
}

func matchesAny(globs []glob.Glob, path string) bool {
	for _, g := range globs {
		if g.Match(path) {
			return true
		}
	}
	return false
}
