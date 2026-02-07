package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestHandleDeleteFile(t *testing.T) {
	reg, tmpDir := setupTestRegistry(t)

	tests := []struct {
		name    string
		setup   func() string
		isError bool
	}{
		{
			name: "delete existing file",
			setup: func() string {
				f := filepath.Join(tmpDir, "todelete.txt")
				os.WriteFile(f, []byte("test"), 0644)
				return f
			},
			isError: false,
		},
		{
			name: "delete non-existent file",
			setup: func() string {
				return filepath.Join(tmpDir, "notfound.txt")
			},
			isError: true,
		},
		{
			name: "delete directory fails",
			setup: func() string {
				d := filepath.Join(tmpDir, "dir")
				os.MkdirAll(d, 0755)
				return d
			},
			isError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup()
			request := mcp.CallToolRequest{}
			request.Params.Arguments = map[string]any{"path": path}

			result, err := HandleDeleteFile(context.Background(), reg, request)
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
				// Verify file is deleted
				if _, err := os.Stat(path); !os.IsNotExist(err) {
					t.Error("file should have been deleted")
				}
			}
		})
	}
}

func TestHandleDeleteDirectory(t *testing.T) {
	reg, tmpDir := setupTestRegistry(t)

	tests := []struct {
		name    string
		setup   func() string
		args    map[string]any
		isError bool
	}{
		{
			name: "delete empty directory",
			setup: func() string {
				d := filepath.Join(tmpDir, "emptydir")
				os.MkdirAll(d, 0755)
				return d
			},
			args:    map[string]any{"recursive": false},
			isError: false,
		},
		{
			name: "delete non-empty directory without recursive fails",
			setup: func() string {
				d := filepath.Join(tmpDir, "nonempty")
				os.MkdirAll(d, 0755)
				os.WriteFile(filepath.Join(d, "file.txt"), []byte("test"), 0644)
				return d
			},
			args:    map[string]any{"recursive": false},
			isError: true,
		},
		{
			name: "delete non-empty directory with recursive succeeds",
			setup: func() string {
				d := filepath.Join(tmpDir, "nonempty2")
				os.MkdirAll(d, 0755)
				os.WriteFile(filepath.Join(d, "file.txt"), []byte("test"), 0644)
				return d
			},
			args:    map[string]any{"recursive": true},
			isError: false,
		},
		{
			name: "delete file as directory fails",
			setup: func() string {
				f := filepath.Join(tmpDir, "file.txt")
				os.WriteFile(f, []byte("test"), 0644)
				return f
			},
			args:    map[string]any{"recursive": false},
			isError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup()
			tt.args["path"] = path
			request := mcp.CallToolRequest{}
			request.Params.Arguments = tt.args

			result, err := HandleDeleteDirectory(context.Background(), reg, request)
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
				// Verify directory is deleted
				if _, err := os.Stat(path); !os.IsNotExist(err) {
					t.Error("directory should have been deleted")
				}
			}
		})
	}
}

func TestDeleteAllowedRoot(t *testing.T) {
	reg, tmpDir := setupTestRegistry(t)

	// Trying to delete the allowed root should fail
	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]any{
		"path":      tmpDir,
		"recursive": true,
	}

	result, err := HandleDeleteDirectory(context.Background(), reg, request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.IsError {
		t.Error("should not be able to delete allowed root directory")
	}
}
