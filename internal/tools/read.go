package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/portertech/filesystem-mcp-server/internal/registry"
	"github.com/portertech/filesystem-mcp-server/internal/stream"
	"github.com/spf13/cast"
)

const maxConcurrentReads = 10

// NewReadTextFileTool creates the read_text_file tool.
func NewReadTextFileTool(reg *registry.Registry) mcp.Tool {
	return mcp.NewTool(
		"read_text_file",
		mcp.WithDescription("Read the contents of a text file. Supports head/tail or start_line/end_line for partial reads."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithString("path", mcp.Description("Path to the file to read"), mcp.Required()),
		mcp.WithNumber("head", mcp.Description("Number of lines to read from the beginning")),
		mcp.WithNumber("tail", mcp.Description("Number of lines to read from the end")),
		mcp.WithNumber("start_line", mcp.Description("Starting line number (1-based, inclusive)")),
		mcp.WithNumber("end_line", mcp.Description("Ending line number (1-based, inclusive)")),
		mcp.WithBoolean("line_numbers", mcp.Description("Prefix each line with its line number")),
	)
}

// HandleReadTextFile handles the read_text_file tool.
func HandleReadTextFile(ctx context.Context, reg *registry.Registry, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path := cast.ToString(request.Params.Arguments["path"])
	head := cast.ToInt(request.Params.Arguments["head"])
	tail := cast.ToInt(request.Params.Arguments["tail"])
	startLine := cast.ToInt(request.Params.Arguments["start_line"])
	endLine := cast.ToInt(request.Params.Arguments["end_line"])
	lineNumbers := cast.ToBool(request.Params.Arguments["line_numbers"])

	resolvedPath, err := reg.Validate(path)
	if err != nil {
		return mcp.NewToolResultError(fmt.Errorf("path validation failed: %w", err).Error()), nil
	}

	// Check if it's a directory
	info, err := os.Stat(resolvedPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Errorf("failed to stat file: %w", err).Error()), nil
	}
	if info.IsDir() {
		return mcp.NewToolResultError("path is a directory, not a file"), nil
	}

	// Validate parameter combinations
	if (head > 0 || tail > 0) && (startLine > 0 || endLine > 0) {
		return mcp.NewToolResultError("cannot use head/tail with start_line/end_line"), nil
	}

	var content string

	// Handle start_line/end_line range (most efficient for AI agents)
	if startLine > 0 || endLine > 0 {
		if startLine <= 0 {
			startLine = 1
		}
		content, err = stream.ReadFileWithLineNumbers(resolvedPath, startLine, endLine)
	} else if lineNumbers {
		// Use optimized functions for line-numbered output
		if head > 0 {
			content, err = stream.ReadFileWithLineNumbers(resolvedPath, 1, head)
		} else if tail > 0 {
			// Single-pass tail with line numbers
			content, err = stream.TailFileWithLineNumbers(resolvedPath, tail)
		} else {
			content, err = stream.ReadFileWithLineNumbers(resolvedPath, 0, 0)
		}
	} else {
		if head > 0 {
			content, err = stream.HeadFile(resolvedPath, head)
		} else if tail > 0 {
			content, err = stream.TailFile(resolvedPath, tail)
		} else {
			var data []byte
			data, err = os.ReadFile(resolvedPath)
			content = string(data)
		}
	}

	if err != nil {
		return mcp.NewToolResultError(fmt.Errorf("failed to read file: %w", err).Error()), nil
	}

	return mcp.NewToolResultText(content), nil
}

// NewReadFileTool creates the read_file tool (deprecated alias for read_text_file).
func NewReadFileTool(reg *registry.Registry) mcp.Tool {
	return mcp.NewTool(
		"read_file",
		mcp.WithDescription("Read the contents of a file. Deprecated: use read_text_file instead."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithString("path", mcp.Description("Path to the file to read"), mcp.Required()),
	)
}

// HandleReadFile handles the read_file tool.
func HandleReadFile(ctx context.Context, reg *registry.Registry, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path := cast.ToString(request.Params.Arguments["path"])

	resolvedPath, err := reg.Validate(path)
	if err != nil {
		return mcp.NewToolResultError(fmt.Errorf("path validation failed: %w", err).Error()), nil
	}

	info, err := os.Stat(resolvedPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Errorf("failed to stat file: %w", err).Error()), nil
	}
	if info.IsDir() {
		return mcp.NewToolResultError("path is a directory, not a file"), nil
	}

	data, err := os.ReadFile(resolvedPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Errorf("failed to read file: %w", err).Error()), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}

// NewReadMultipleFilesTool creates the read_multiple_files tool.
func NewReadMultipleFilesTool(reg *registry.Registry) mcp.Tool {
	return mcp.NewTool(
		"read_multiple_files",
		mcp.WithDescription("Read multiple files concurrently. Returns content with paths as references."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithArray("paths", mcp.Description("Array of file paths to read"), mcp.Required(), mcp.Items(map[string]any{"type": "string"})),
		mcp.WithString("format", mcp.Description("Output format: 'text' or 'json'")),
	)
}

// HandleReadMultipleFiles handles the read_multiple_files tool.
func HandleReadMultipleFiles(ctx context.Context, reg *registry.Registry, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	pathsArg := request.Params.Arguments["paths"]
	format := cast.ToString(request.Params.Arguments["format"])
	var paths []string
	if arr, ok := pathsArg.([]interface{}); ok {
		for _, v := range arr {
			paths = append(paths, cast.ToString(v))
		}
	}

	if len(paths) == 0 {
		return mcp.NewToolResultError("no paths provided"), nil
	}

	type fileResult struct {
		path    string
		content string
		err     error
	}

	results := make([]fileResult, len(paths))
	var wg sync.WaitGroup
	sem := make(chan struct{}, maxConcurrentReads) // Limit concurrency to 10

	for i, path := range paths {
		wg.Add(1)
		go func(idx int, p string) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire
			defer func() { <-sem }() // Release

			result := fileResult{path: p}

			resolvedPath, err := reg.Validate(p)
			if err != nil {
				result.err = err
				results[idx] = result
				return
			}

			info, err := os.Stat(resolvedPath)
			if err != nil {
				result.err = err
				results[idx] = result
				return
			}
			if info.IsDir() {
				result.err = fmt.Errorf("path is a directory")
				results[idx] = result
				return
			}

			data, err := os.ReadFile(resolvedPath)
			if err != nil {
				result.err = err
				results[idx] = result
				return
			}

			result.content = string(data)
			results[idx] = result
		}(i, path)
	}

	wg.Wait()

	if format == "json" {
		type fileEntry struct {
			Path    string `json:"path"`
			Content string `json:"content,omitempty"`
			Error   string `json:"error,omitempty"`
		}

		entries := make([]fileEntry, 0, len(results))
		for _, r := range results {
			entry := fileEntry{Path: r.path}
			if r.err != nil {
				entry.Error = r.err.Error()
			} else {
				entry.Content = r.content
			}
			entries = append(entries, entry)
		}

		jsonResult, err := json.MarshalIndent(entries, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Errorf("failed to marshal result: %w", err).Error()), nil
		}
		return mcp.NewToolResultText(string(jsonResult)), nil
	}

	// Build response
	var output string
	for _, r := range results {
		output += fmt.Sprintf("=== %s ===\n", r.path)
		if r.err != nil {
			output += fmt.Sprintf("Error: %v\n", r.err)
		} else {
			output += r.content
		}
		output += "\n\n"
	}

	return mcp.NewToolResultText(output), nil
}
