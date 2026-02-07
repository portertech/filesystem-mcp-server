package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestHandleWriteFile(t *testing.T) {
	reg, tmpDir := setupTestRegistry(t)

	tests := []struct {
		name     string
		args     map[string]any
		isError  bool
		validate func(t *testing.T)
		setup    func(t *testing.T)
	}{
		{
			name: "write new file",
			args: map[string]any{
				"path":    filepath.Join(tmpDir, "new.txt"),
				"content": "test content",
			},
			isError: false,
			validate: func(t *testing.T) {
				data, err := os.ReadFile(filepath.Join(tmpDir, "new.txt"))
				if err != nil {
					t.Fatalf("failed to read written file: %v", err)
				}
				if string(data) != "test content" {
					t.Errorf("content mismatch: got %q, want %q", string(data), "test content")
				}
			},
		},
		{
			name: "overwrite existing file",
			args: map[string]any{
				"path":    filepath.Join(tmpDir, "existing.txt"),
				"content": "new content",
			},
			isError: false,
			validate: func(t *testing.T) {
				data, err := os.ReadFile(filepath.Join(tmpDir, "existing.txt"))
				if err != nil {
					t.Fatalf("failed to read written file: %v", err)
				}
				if string(data) != "new content" {
					t.Errorf("content mismatch: got %q, want %q", string(data), "new content")
				}
			},
		},
		{
			name: "create parent directories",
			args: map[string]any{
				"path":    filepath.Join(tmpDir, "subdir", "nested", "file.txt"),
				"content": "nested content",
			},
			isError: false,
			validate: func(t *testing.T) {
				data, err := os.ReadFile(filepath.Join(tmpDir, "subdir", "nested", "file.txt"))
				if err != nil {
					t.Fatalf("failed to read written file: %v", err)
				}
				if string(data) != "nested content" {
					t.Errorf("content mismatch: got %q, want %q", string(data), "nested content")
				}
			},
		},
		{
			name: "path outside allowed",
			args: map[string]any{
				"path":    "/etc/test.txt",
				"content": "test",
			},
			isError: true,
		},
		{
			name: "write fails when destination is symlink",
			args: map[string]any{
				"path":    filepath.Join(tmpDir, "link.txt"),
				"content": "test",
			},
			isError: true,
			setup: func(t *testing.T) {
				target := filepath.Join(tmpDir, "target.txt")
				os.WriteFile(target, []byte("test"), 0644)
				link := filepath.Join(tmpDir, "link.txt")
				if err := os.Symlink(target, link); err != nil {
					t.Skip("cannot create symlink")
				}
			},
		},
		{
			name: "write fails when parent is symlink",
			args: map[string]any{
				"path":    filepath.Join(tmpDir, "linkdir", "file.txt"),
				"content": "test",
			},
			isError: true,
			setup: func(t *testing.T) {
				outside := t.TempDir()
				link := filepath.Join(tmpDir, "linkdir")
				if err := os.Symlink(outside, link); err != nil {
					t.Skip("cannot create symlink")
				}
			},
		},
	}

	// Create existing file for overwrite test
	if err := os.WriteFile(filepath.Join(tmpDir, "existing.txt"), []byte("old content"), 0644); err != nil {
		t.Fatal(err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup(t)
			}
			request := mcp.CallToolRequest{}
			request.Params.Arguments = tt.args

			result, err := HandleWriteFile(context.Background(), reg, request)
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
				if tt.validate != nil {
					tt.validate(t)
				}
			}
		})
	}
}

func TestAtomicWriteFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "atomic.txt")

	if err := atomicWriteFile(testFile, []byte("test"), 0644, []string{tmpDir}); err != nil {
		t.Fatalf("atomicWriteFile error: %v", err)
	}

	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "test" {
		t.Errorf("content mismatch: got %q, want %q", string(data), "test")
	}

	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0644 {
		t.Errorf("permissions mismatch: got %o, want %o", info.Mode().Perm(), 0644)
	}
}
