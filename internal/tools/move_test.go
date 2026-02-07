package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestHandleMoveFile(t *testing.T) {
	reg, tmpDir := setupTestRegistry(t)

	tests := []struct {
		name    string
		setup   func() (string, string)
		isError bool
	}{
		{
			name: "move file",
			setup: func() (string, string) {
				src := filepath.Join(tmpDir, "movesrc.txt")
				dst := filepath.Join(tmpDir, "movedst.txt")
				os.WriteFile(src, []byte("test"), 0644)
				return src, dst
			},
			isError: false,
		},
		{
			name: "move directory",
			setup: func() (string, string) {
				src := filepath.Join(tmpDir, "movedir")
				dst := filepath.Join(tmpDir, "movedirrenamed")
				os.MkdirAll(src, 0755)
				return src, dst
			},
			isError: false,
		},
		{
			name: "move fails if destination exists",
			setup: func() (string, string) {
				src := filepath.Join(tmpDir, "src2.txt")
				dst := filepath.Join(tmpDir, "dst2.txt")
				os.WriteFile(src, []byte("test"), 0644)
				os.WriteFile(dst, []byte("existing"), 0644)
				return src, dst
			},
			isError: true,
		},
		{
			name: "move fails if source doesn't exist",
			setup: func() (string, string) {
				return filepath.Join(tmpDir, "notfound.txt"), filepath.Join(tmpDir, "dst3.txt")
			},
			isError: true,
		},
		{
			name: "move fails if destination is symlink",
			setup: func() (string, string) {
				src := filepath.Join(tmpDir, "src3.txt")
				dst := filepath.Join(tmpDir, "symlinkdst.txt")
				if err := os.WriteFile(src, []byte("test"), 0644); err != nil {
					t.Fatal(err)
				}
				target := filepath.Join(tmpDir, "target.txt")
				if err := os.WriteFile(target, []byte("target"), 0644); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, dst); err != nil {
					t.Skip("cannot create symlink")
				}
				return src, dst
			},
			isError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, dst := tt.setup()
			request := mcp.CallToolRequest{}
			request.Params.Arguments = map[string]any{
				"source":      src,
				"destination": dst,
			}

			result, err := HandleMoveFile(context.Background(), reg, request)
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
				// Verify move happened
				if _, err := os.Stat(src); !os.IsNotExist(err) {
					t.Error("source should no longer exist")
				}
				if _, err := os.Stat(dst); err != nil {
					t.Error("destination should exist")
				}
			}
		})
	}
}
