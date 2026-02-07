package tools

import (
	"bufio"
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

// NewReadTextFileTool creates the read_text_file tool.
func NewReadTextFileTool(reg *registry.Registry) mcp.Tool {
	return mcp.NewTool(
		"read_text_file",
		mcp.WithDescription("Read the contents of a text file. Supports head/tail for partial reads."),
		mcp.WithString("path", mcp.Description("Path to the file to read"), mcp.Required()),
		mcp.WithNumber("head", mcp.Description("Number of lines to read from the beginning")),
		mcp.WithNumber("tail", mcp.Description("Number of lines to read from the end")),
		mcp.WithBoolean("line_numbers", mcp.Description("Prefix each line with its line number")),
	)
}

// HandleReadTextFile handles the read_text_file tool.
func HandleReadTextFile(ctx context.Context, reg *registry.Registry, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path := cast.ToString(request.Params.Arguments["path"])
	head := cast.ToInt(request.Params.Arguments["head"])
	tail := cast.ToInt(request.Params.Arguments["tail"])
	lineNumbers := cast.ToBool(request.Params.Arguments["line_numbers"])

	resolvedPath, err := reg.Validate(path)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("path validation failed: %v", err)), nil
	}

	// Check if it's a directory
	info, err := os.Stat(resolvedPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to stat file: %v", err)), nil
	}
	if info.IsDir() {
		return mcp.NewToolResultError("path is a directory, not a file"), nil
	}

	var content string

	if lineNumbers {
		// Use ReadFileWithLineNumbers for line-numbered output
		if head > 0 {
			content, err = stream.ReadFileWithLineNumbers(resolvedPath, 1, head)
		} else if tail > 0 {
			// For tail with line numbers, we need to find the starting line
			totalLines, countErr := countFileLines(resolvedPath)
			if countErr != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to count lines: %v", countErr)), nil
			}
			startLine := totalLines - tail + 1
			if startLine < 1 {
				startLine = 1
			}
			content, err = stream.ReadFileWithLineNumbers(resolvedPath, startLine, 0)
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
		return mcp.NewToolResultError(fmt.Sprintf("failed to read file: %v", err)), nil
	}

	return mcp.NewToolResultText(content), nil
}

// countFileLines counts the total number of lines in a file.
func countFileLines(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	count := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		count++
	}
	return count, scanner.Err()
}

// NewReadFileTool creates the read_file tool (deprecated alias for read_text_file).
func NewReadFileTool(reg *registry.Registry) mcp.Tool {
	return mcp.NewTool(
		"read_file",
		mcp.WithDescription("Read the contents of a file. Deprecated: use read_text_file instead."),
		mcp.WithString("path", mcp.Description("Path to the file to read"), mcp.Required()),
	)
}

// HandleReadFile handles the read_file tool.
func HandleReadFile(ctx context.Context, reg *registry.Registry, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path := cast.ToString(request.Params.Arguments["path"])

	resolvedPath, err := reg.Validate(path)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("path validation failed: %v", err)), nil
	}

	info, err := os.Stat(resolvedPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to stat file: %v", err)), nil
	}
	if info.IsDir() {
		return mcp.NewToolResultError("path is a directory, not a file"), nil
	}

	data, err := os.ReadFile(resolvedPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to read file: %v", err)), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}

// NewReadMultipleFilesTool creates the read_multiple_files tool.
func NewReadMultipleFilesTool(reg *registry.Registry) mcp.Tool {
	return mcp.NewTool(
		"read_multiple_files",
		mcp.WithDescription("Read multiple files concurrently. Returns content with paths as references."),
		mcp.WithArray("paths", mcp.Description("Array of file paths to read"), mcp.Required()),
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
	sem := make(chan struct{}, 10) // Limit concurrency to 10

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
			return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
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
