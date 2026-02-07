package tools

import (
	"context"
	"encoding/json"
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

func TestHandleCreateDirectoryRejectsSymlinksInPath(t *testing.T) {
	reg, tmpDir := setupTestRegistry(t)

	// Create a real directory
	realDir := filepath.Join(tmpDir, "realdir")
	if err := os.MkdirAll(realDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a symlink to the real directory
	linkDir := filepath.Join(tmpDir, "linkdir")
	if err := os.Symlink(realDir, linkDir); err != nil {
		t.Skip("cannot create symlink")
	}

	// Try to create a directory through the symlink
	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]any{
		"path": filepath.Join(linkDir, "newsubdir"),
	}

	result, err := HandleCreateDirectory(context.Background(), reg, request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.IsError {
		t.Error("expected error when creating directory through symlink")
	}

	// Verify directory was not created through the symlink
	if _, err := os.Stat(filepath.Join(realDir, "newsubdir")); err == nil {
		t.Error("directory should not have been created through symlink")
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

	output := result.Content[0].(mcp.TextContent).Text
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	if lines[0] != "[FILE] file1.txt" {
		t.Errorf("expected first line to be file1.txt, got %q", lines[0])
	}
	if lines[1] != "[FILE] file2.txt" {
		t.Errorf("expected second line to be file2.txt, got %q", lines[1])
	}
	if lines[2] != "[DIR] subdir" {
		t.Errorf("expected third line to be subdir, got %q", lines[2])
	}
}

func TestHandleListDirectoryJSON(t *testing.T) {
	reg, tmpDir := setupTestRegistry(t)

	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("test"), 0644)
	os.MkdirAll(filepath.Join(tmpDir, "subdir"), 0755)

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]any{"path": tmpDir, "format": "json"}

	result, err := HandleListDirectory(context.Background(), reg, request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsError {
		t.Errorf("unexpected error: %v", result.Content)
	}

	output := result.Content[0].(mcp.TextContent).Text
	var entries []map[string]string
	if err := json.Unmarshal([]byte(output), &entries); err != nil {
		t.Fatalf("expected valid json output: %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("expected entries in json output")
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

func TestHandleListDirectoryWithSizesJSON(t *testing.T) {
	reg, tmpDir := setupTestRegistry(t)

	os.WriteFile(filepath.Join(tmpDir, "small.txt"), []byte("small"), 0644)
	os.MkdirAll(filepath.Join(tmpDir, "subdir"), 0755)

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]any{"path": tmpDir, "sortBy": "name", "format": "json"}

	result, err := HandleListDirectoryWithSizes(context.Background(), reg, request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsError {
		t.Errorf("unexpected error: %v", result.Content)
	}

	output := result.Content[0].(mcp.TextContent).Text
	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("expected valid json output: %v", err)
	}
	entries, ok := payload["entries"].([]interface{})
	if !ok || len(entries) == 0 {
		t.Fatalf("expected entries in json output")
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

	var tree struct {
		Name     string `json:"name"`
		Type     string `json:"type"`
		Children []struct {
			Name     string `json:"name"`
			Type     string `json:"type"`
			Children []struct {
				Name string `json:"name"`
				Type string `json:"type"`
			} `json:"children"`
		} `json:"children"`
	}
	if err := json.Unmarshal([]byte(output), &tree); err != nil {
		t.Fatalf("expected valid json output: %v", err)
	}
	if len(tree.Children) == 0 {
		t.Fatalf("expected children in tree output")
	}
	if tree.Children[0].Name != "a" {
		t.Fatalf("expected first child to be 'a', got %q", tree.Children[0].Name)
	}
}

func TestHandleDirectoryTreeSkipsSymlinkedDirectories(t *testing.T) {
	reg, tmpDir := setupTestRegistry(t)

	targetDir := t.TempDir()
	os.WriteFile(filepath.Join(targetDir, "secret.txt"), []byte("secret"), 0644)

	linkPath := filepath.Join(tmpDir, "link")
	if err := os.Symlink(targetDir, linkPath); err != nil {
		t.Skip("cannot create symlink")
	}

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]any{"path": tmpDir}

	result, err := HandleDirectoryTree(context.Background(), reg, request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsError {
		t.Errorf("unexpected error: %v", result.Content)
	}

	output := result.Content[0].(mcp.TextContent).Text
	if strings.Contains(output, "secret.txt") {
		t.Error("symlinked directory contents should not be in output")
	}
	if strings.Contains(output, "link") {
		t.Error("symlinked directory should not be in output")
	}
}
