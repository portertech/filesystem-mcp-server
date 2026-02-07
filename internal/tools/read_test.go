package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/portertech/filesystem-mcp-server/internal/registry"
)

func setupTestRegistry(t *testing.T) (*registry.Registry, string) {
	tmpDir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	reg := registry.New([]string{tmpDir}, logger)
	return reg, tmpDir
}

func TestHandleReadTextFile(t *testing.T) {
	reg, tmpDir := setupTestRegistry(t)

	testFile := filepath.Join(tmpDir, "test.txt")
	content := "line1\nline2\nline3\nline4\nline5"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		args    map[string]any
		isError bool
	}{
		{
			name:    "read full file",
			args:    map[string]any{"path": testFile},
			isError: false,
		},
		{
			name:    "read head",
			args:    map[string]any{"path": testFile, "head": 2},
			isError: false,
		},
		{
			name:    "read tail",
			args:    map[string]any{"path": testFile, "tail": 2},
			isError: false,
		},
		{
			name:    "file not found",
			args:    map[string]any{"path": filepath.Join(tmpDir, "notfound.txt")},
			isError: true,
		},
		{
			name:    "path outside allowed",
			args:    map[string]any{"path": "/etc/passwd"},
			isError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := mcp.CallToolRequest{}
			request.Params.Arguments = tt.args

			result, err := HandleReadTextFile(context.Background(), reg, request)
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

func TestHandleReadMultipleFiles(t *testing.T) {
	reg, tmpDir := setupTestRegistry(t)

	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")
	if err := os.WriteFile(file1, []byte("content1"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file2, []byte("content2"), 0644); err != nil {
		t.Fatal(err)
	}

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]any{
		"paths": []interface{}{file1, file2},
	}

	result, err := HandleReadMultipleFiles(context.Background(), reg, request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsError {
		t.Errorf("unexpected error: %v", result.Content)
	}
}

func TestHandleReadMultipleFilesJSON(t *testing.T) {
	reg, tmpDir := setupTestRegistry(t)

	file1 := filepath.Join(tmpDir, "file1.txt")
	if err := os.WriteFile(file1, []byte("content1"), 0644); err != nil {
		t.Fatal(err)
	}

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]any{
		"paths":  []interface{}{file1, filepath.Join(tmpDir, "missing.txt")},
		"format": "json",
	}

	result, err := HandleReadMultipleFiles(context.Background(), reg, request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsError {
		t.Errorf("unexpected error: %v", result.Content)
	}

	output := result.Content[0].(mcp.TextContent).Text
	var entries []map[string]any
	if err := json.Unmarshal([]byte(output), &entries); err != nil {
		t.Fatalf("expected valid json output: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries in json output")
	}
}

func TestHandleReadTextFileWithLineNumbers(t *testing.T) {
	reg, tmpDir := setupTestRegistry(t)

	testFile := filepath.Join(tmpDir, "test.txt")
	content := "line1\nline2\nline3\nline4\nline5"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		args     map[string]any
		expected string
	}{
		{
			name:     "full file with line numbers",
			args:     map[string]any{"path": testFile, "line_numbers": true},
			expected: "   1 | line1\n   2 | line2\n   3 | line3\n   4 | line4\n   5 | line5",
		},
		{
			name:     "head with line numbers",
			args:     map[string]any{"path": testFile, "head": 2, "line_numbers": true},
			expected: "   1 | line1\n   2 | line2",
		},
		{
			name:     "tail with line numbers",
			args:     map[string]any{"path": testFile, "tail": 2, "line_numbers": true},
			expected: "   4 | line4\n   5 | line5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := mcp.CallToolRequest{}
			request.Params.Arguments = tt.args

			result, err := HandleReadTextFile(context.Background(), reg, request)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.IsError {
				t.Errorf("unexpected error: %v", result.Content)
				return
			}

			// Extract text content from result
			if len(result.Content) == 0 {
				t.Fatal("expected content in result")
			}
			textContent, ok := result.Content[0].(mcp.TextContent)
			if !ok {
				t.Fatal("expected TextContent")
			}

			if textContent.Text != tt.expected {
				t.Errorf("expected:\n%q\ngot:\n%q", tt.expected, textContent.Text)
			}
		})
	}
}

func TestHandleReadTextFileWithLineNumbersEmptyFile(t *testing.T) {
	reg, tmpDir := setupTestRegistry(t)

	emptyFile := filepath.Join(tmpDir, "empty.txt")
	if err := os.WriteFile(emptyFile, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]any{"path": emptyFile, "line_numbers": true}

	result, err := HandleReadTextFile(context.Background(), reg, request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsError {
		t.Errorf("unexpected error: %v", result.Content)
		return
	}

	// Extract text content from result
	if len(result.Content) == 0 {
		t.Fatal("expected content in result")
	}
	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatal("expected TextContent")
	}

	if textContent.Text != "" {
		t.Errorf("expected empty string, got: %q", textContent.Text)
	}
}
