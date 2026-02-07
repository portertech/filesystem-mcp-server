package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestHandleCopyFile(t *testing.T) {
	reg, tmpDir := setupTestRegistry(t)

	srcFile := filepath.Join(tmpDir, "source.txt")
	if err := os.WriteFile(srcFile, []byte("source content"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		args    map[string]any
		setup   func(t *testing.T)
		isError bool
	}{
		{
			name: "copy to new file",
			args: map[string]any{
				"source":      srcFile,
				"destination": filepath.Join(tmpDir, "dest1.txt"),
			},
			isError: false,
		},
		{
			name: "copy fails on existing without overwrite",
			args: map[string]any{
				"source":      srcFile,
				"destination": filepath.Join(tmpDir, "existing.txt"),
			},
			setup: func(t *testing.T) {
				if err := os.WriteFile(filepath.Join(tmpDir, "existing.txt"), []byte("existing"), 0644); err != nil {
					t.Fatalf("failed to write existing file: %v", err)
				}
			},
			isError: true,
		},
		{
			name: "copy succeeds with overwrite",
			args: map[string]any{
				"source":      srcFile,
				"destination": filepath.Join(tmpDir, "overwrite.txt"),
				"overwrite":   true,
			},
			setup: func(t *testing.T) {
				if err := os.WriteFile(filepath.Join(tmpDir, "overwrite.txt"), []byte("old"), 0644); err != nil {
					t.Fatalf("failed to write overwrite file: %v", err)
				}
			},
			isError: false,
		},
		{
			name: "source not found",
			args: map[string]any{
				"source":      filepath.Join(tmpDir, "notfound.txt"),
				"destination": filepath.Join(tmpDir, "dest.txt"),
			},
			isError: true,
		},
		{
			name: "source outside allowed",
			args: map[string]any{
				"source":      "/etc/passwd",
				"destination": filepath.Join(tmpDir, "dest.txt"),
			},
			isError: true,
		},
		{
			name: "destination symlink fails",
			args: map[string]any{
				"source":      srcFile,
				"destination": filepath.Join(tmpDir, "symlinked.txt"),
			},
			setup: func(t *testing.T) {
				target := filepath.Join(tmpDir, "target.txt")
				if err := os.WriteFile(target, []byte("target"), 0644); err != nil {
					t.Fatalf("failed to write target: %v", err)
				}
				if err := os.Symlink(target, filepath.Join(tmpDir, "symlinked.txt")); err != nil {
					t.Skip("cannot create symlink")
				}
			},
			isError: true,
		},
		{
			name: "destination parent symlink fails",
			args: map[string]any{
				"source":      srcFile,
				"destination": filepath.Join(tmpDir, "linkdir", "dest.txt"),
			},
			setup: func(t *testing.T) {
				outside := t.TempDir()
				if err := os.Symlink(outside, filepath.Join(tmpDir, "linkdir")); err != nil {
					t.Skip("cannot create symlink")
				}
			},
			isError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup(t)
			}

			request := mcp.CallToolRequest{}
			request.Params.Arguments = tt.args

			result, err := HandleCopyFile(context.Background(), reg, request)
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

				// Verify content was copied
				dest := tt.args["destination"].(string)
				data, err := os.ReadFile(dest)
				if err != nil {
					t.Fatalf("failed to read destination: %v", err)
				}
				if string(data) != "source content" {
					t.Errorf("content mismatch: got %q, want %q", string(data), "source content")
				}
			}
		})
	}
}
