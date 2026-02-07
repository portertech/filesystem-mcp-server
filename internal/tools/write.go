package tools

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/portertech/filesystem-mcp-server/internal/registry"
	"github.com/portertech/filesystem-mcp-server/internal/security"
	"github.com/spf13/cast"
)

// NewWriteFileTool creates the write_file tool.
func NewWriteFileTool(reg *registry.Registry) mcp.Tool {
	return mcp.NewTool(
		"write_file",
		mcp.WithDescription("Write content to a file. Creates parent directories if needed. Uses atomic write."),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithString("path", mcp.Description("Path to the file to write"), mcp.Required()),
		mcp.WithString("content", mcp.Description("Content to write to the file"), mcp.Required()),
	)
}

// HandleWriteFile handles the write_file tool.
func HandleWriteFile(ctx context.Context, reg *registry.Registry, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path := cast.ToString(request.Params.Arguments["path"])
	content := cast.ToString(request.Params.Arguments["content"])

	resolvedPath, err := reg.ValidateForCreation(path)
	if err != nil {
		return mcp.NewToolResultError(fmt.Errorf("path validation failed: %w", err).Error()), nil
	}

	// Create parent directories if needed
	dir := filepath.Dir(resolvedPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return mcp.NewToolResultError(fmt.Errorf("failed to create directories: %w", err).Error()), nil
	}

	// Atomic write using temp file
	if err := atomicWriteFile(resolvedPath, []byte(content), 0644, reg.Get()); err != nil {
		return mcp.NewToolResultError(fmt.Errorf("failed to write file: %w", err).Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Successfully wrote to %s", resolvedPath)), nil
}

// atomicWriteFile writes data to a file atomically using a temp file and rename.
func atomicWriteFile(path string, data []byte, perm os.FileMode, allowedDirs []string) error {
	// Validate destination path before any I/O
	if _, err := security.ValidateFinalPathForCreation(path, allowedDirs); err != nil {
		return fmt.Errorf("path validation failed: %w", err)
	}

	dir := filepath.Dir(path)

	// Generate random suffix for temp file
	randBytes := make([]byte, 8)
	if _, err := rand.Read(randBytes); err != nil {
		return fmt.Errorf("failed to generate random bytes: %w", err)
	}
	tmpName := filepath.Join(dir, ".tmp-"+hex.EncodeToString(randBytes))

	// Create temp file with O_EXCL to prevent symlink attacks on new files
	f, err := os.OpenFile(tmpName, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	success := false
	defer func() {
		if !success {
			os.Remove(tmpName)
		}
	}()

	if _, err := f.Write(data); err != nil {
		f.Close()
		return fmt.Errorf("failed to write data: %w", err)
	}

	if err := f.Sync(); err != nil {
		f.Close()
		return fmt.Errorf("failed to sync file: %w", err)
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("failed to close file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	success = true
	return nil
}
