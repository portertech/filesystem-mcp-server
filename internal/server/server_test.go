package server

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/portertech/filesystem-mcp-server/internal/registry"
)

func setupTestServer(t *testing.T) (*Server, string) {
	tmpDir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	reg := registry.New([]string{tmpDir}, logger)
	srv := New(reg, logger)
	return srv, tmpDir
}

func TestServerCreation(t *testing.T) {
	srv, _ := setupTestServer(t)

	if srv.mcpServer == nil {
		t.Error("MCP server should not be nil")
	}

	if srv.registry == nil {
		t.Error("Registry should not be nil")
	}

	if srv.logger == nil {
		t.Error("Logger should not be nil")
	}
}

func TestGetMCPServer(t *testing.T) {
	srv, _ := setupTestServer(t)

	mcpSrv := srv.GetMCPServer()
	if mcpSrv == nil {
		t.Error("GetMCPServer should return non-nil server")
	}
}

func TestGetRegistry(t *testing.T) {
	srv, tmpDir := setupTestServer(t)

	reg := srv.GetRegistry()
	if reg == nil {
		t.Error("GetRegistry should return non-nil registry")
	}

	dirs := reg.Get()
	if len(dirs) != 1 {
		t.Errorf("expected 1 directory, got %d", len(dirs))
	}

	// The directory might be resolved (e.g., /var -> /private/var on macOS)
	// so we check if the base name matches
	if filepath.Base(dirs[0]) != filepath.Base(tmpDir) {
		t.Errorf("unexpected directory: %s", dirs[0])
	}
}

func TestUpdateFromRoots(t *testing.T) {
	tmpDir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	reg := registry.New([]string{}, logger)

	// Test with file:// URI
	UpdateFromRoots(reg, []string{"file://" + tmpDir}, logger)

	dirs := reg.Get()
	if len(dirs) != 1 {
		t.Errorf("expected 1 directory, got %d", len(dirs))
	}
}

func TestUpdateFromRootsWithInvalidPath(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	reg := registry.New([]string{}, logger)

	// Test with non-existent path
	UpdateFromRoots(reg, []string{"/nonexistent/path/12345"}, logger)

	dirs := reg.Get()
	if len(dirs) != 0 {
		t.Errorf("expected 0 directories for invalid path, got %d", len(dirs))
	}
}
