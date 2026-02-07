package server

import (
	"log/slog"
	"os"
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
