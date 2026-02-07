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

func TestHandleSearchFiles(t *testing.T) {
	reg, tmpDir := setupTestRegistry(t)

	// Create test structure
	os.MkdirAll(filepath.Join(tmpDir, "a", "b"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "test.go"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "a", "nested.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "a", "b", "deep.txt"), []byte("test"), 0644)
	os.MkdirAll(filepath.Join(tmpDir, "excluded"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "excluded", "skip.txt"), []byte("test"), 0644)

	tests := []struct {
		name        string
		args        map[string]any
		contains    []string
		notContains []string
		isError     bool
	}{
		{
			name: "find all txt files",
			args: map[string]any{
				"path":    tmpDir,
				"pattern": "**/*.txt",
			},
			contains: []string{filepath.ToSlash(filepath.Join("test.txt")), filepath.ToSlash(filepath.Join("a", "nested.txt")), filepath.ToSlash(filepath.Join("a", "b", "deep.txt"))},
		},
		{
			name: "find go files",
			args: map[string]any{
				"path":    tmpDir,
				"pattern": "*.go",
			},
			contains: []string{"test.go"},
		},
		{
			name: "with exclude pattern",
			args: map[string]any{
				"path":            tmpDir,
				"pattern":         "**/*.txt",
				"excludePatterns": []interface{}{"excluded/**"},
			},
			contains:    []string{filepath.ToSlash(filepath.Join("test.txt"))},
			notContains: []string{filepath.ToSlash(filepath.Join("excluded", "skip.txt"))},
		},
		{
			name: "no matches",
			args: map[string]any{
				"path":    tmpDir,
				"pattern": "*.xyz",
			},
			contains: []string{"No matches found"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := mcp.CallToolRequest{}
			request.Params.Arguments = tt.args

			result, err := HandleSearchFiles(context.Background(), reg, request)
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

				output := result.Content[0].(mcp.TextContent).Text
				for _, expected := range tt.contains {
					if !strings.Contains(output, expected) {
						t.Errorf("expected %q in output: %s", expected, output)
					}
				}
				for _, notExpected := range tt.notContains {
					if strings.Contains(output, notExpected) {
						t.Errorf("did not expect %q in output: %s", notExpected, output)
					}
				}
			}
		})
	}
}

func TestHandleSearchFilesJSON(t *testing.T) {
	reg, tmpDir := setupTestRegistry(t)

	os.MkdirAll(filepath.Join(tmpDir, "a"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "a", "nested.txt"), []byte("test"), 0644)

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]any{"path": tmpDir, "pattern": "**/*.txt", "format": "json"}

	result, err := HandleSearchFiles(context.Background(), reg, request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsError {
		t.Errorf("unexpected error: %v", result.Content)
	}

	output := result.Content[0].(mcp.TextContent).Text
	var matches []string
	if err := json.Unmarshal([]byte(output), &matches); err != nil {
		t.Fatalf("expected valid json output: %v", err)
	}
	if len(matches) == 0 {
		t.Fatalf("expected matches in json output")
	}
}

func TestHandleSearchFilesSkipsSymlinks(t *testing.T) {
	reg, tmpDir := setupTestRegistry(t)

	os.MkdirAll(filepath.Join(tmpDir, "dir"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "dir", "nested.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("test"), 0644)

	linkFile := filepath.Join(tmpDir, "linkfile.txt")
	if err := os.Symlink(filepath.Join(tmpDir, "file.txt"), linkFile); err != nil {
		t.Skip("cannot create symlink")
	}

	linkDir := filepath.Join(tmpDir, "linkdir")
	if err := os.Symlink(filepath.Join(tmpDir, "dir"), linkDir); err != nil {
		t.Skip("cannot create symlink")
	}

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]any{"path": tmpDir, "pattern": "**/*.txt"}

	result, err := HandleSearchFiles(context.Background(), reg, request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsError {
		t.Errorf("unexpected error: %v", result.Content)
	}

	output := result.Content[0].(mcp.TextContent).Text
	if strings.Contains(output, "linkfile.txt") {
		t.Error("symlinked file should not be included in output")
	}
	if strings.Contains(output, filepath.ToSlash(filepath.Join("linkdir", "nested.txt"))) {
		t.Error("symlinked directory contents should not be included in output")
	}
}
