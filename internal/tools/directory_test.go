package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestHandleCreateDirectory(t *testing.T) {
	reg, tmpDir := setupTestRegistry(t)

	tests := []struct {
		name    string
		args    map[string]any
		isError bool
	}{
		{
			name:    "create simple directory",
			args:    map[string]any{"path": filepath.Join(tmpDir, "newdir")},
			isError: false,
		},
		{
			name:    "create nested directory",
			args:    map[string]any{"path": filepath.Join(tmpDir, "a", "b", "c")},
			isError: false,
		},
		{
			name:    "create existing directory (idempotent)",
			args:    map[string]any{"path": tmpDir},
			isError: false,
		},
		{
			name:    "create outside allowed",
			args:    map[string]any{"path": "/tmp/outsideallowed"},
			isError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := mcp.CallToolRequest{}
			request.Params.Arguments = tt.args

			result, err := HandleCreateDirectory(context.Background(), reg, request)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.isError {
				if !result.IsError {
					t.Error("expected error result")
				}
			} else {
				if result.IsError {
					t.Errorf("unexpected error: %v", result.Content)
				}
			}
		})
	}
}

func TestHandleListDirectory(t *testing.T) {
	reg, tmpDir := setupTestRegistry(t)

	// Create some files and directories
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("test"), 0644)
	os.MkdirAll(filepath.Join(tmpDir, "subdir"), 0755)

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]any{"path": tmpDir}

	result, err := HandleListDirectory(context.Background(), reg, request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsError {
		t.Errorf("unexpected error: %v", result.Content)
	}

	// Check output contains expected entries
	output := result.Content[0].(mcp.TextContent).Text
	if !strings.Contains(output, "[FILE] file1.txt") {
		t.Error("expected file1.txt in output")
	}
	if !strings.Contains(output, "[DIR] subdir") {
		t.Error("expected subdir in output")
	}
}

func TestHandleListDirectoryWithSizes(t *testing.T) {
	reg, tmpDir := setupTestRegistry(t)

	// Create files with known sizes
	os.WriteFile(filepath.Join(tmpDir, "small.txt"), []byte("small"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "large.txt"), []byte(strings.Repeat("x", 1000)), 0644)

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]any{"path": tmpDir, "sortBy": "size"}

	result, err := HandleListDirectoryWithSizes(context.Background(), reg, request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsError {
		t.Errorf("unexpected error: %v", result.Content)
	}

	output := result.Content[0].(mcp.TextContent).Text
	if !strings.Contains(output, "Summary:") {
		t.Error("expected Summary in output")
	}
}

func TestHandleDirectoryTree(t *testing.T) {
	reg, tmpDir := setupTestRegistry(t)

	// Create nested structure
	os.MkdirAll(filepath.Join(tmpDir, "a", "b"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "a", "file.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "a", "b", "nested.txt"), []byte("test"), 0644)
	os.MkdirAll(filepath.Join(tmpDir, "excluded"), 0755)

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]any{
		"path":            tmpDir,
		"excludePatterns": []interface{}{"excluded"},
	}

	result, err := HandleDirectoryTree(context.Background(), reg, request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsError {
		t.Errorf("unexpected error: %v", result.Content)
	}

	output := result.Content[0].(mcp.TextContent).Text
	if strings.Contains(output, "excluded") {
		t.Error("excluded directory should not be in output")
	}
	if !strings.Contains(output, "nested.txt") {
		t.Error("expected nested.txt in output")
	}
}
