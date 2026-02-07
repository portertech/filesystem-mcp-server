package tools

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/portertech/filesystem-mcp-server/internal/pathutil"
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
	dir := filepath.Dir(path)
	if err := safeMkdirAll(dir, 0755, reg.Get()); err != nil {
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

func safeMkdirAll(path string, perm os.FileMode, allowedDirs []string) error {
	normalized, err := pathutil.NormalizePath(path)
	if err != nil {
		return err
	}
	if _, err := security.ValidatePathForCreation(normalized, allowedDirs); err != nil {
		return err
	}

	// Validate no symlinks exist in the path before creating directories
	if err := security.ValidateNoSymlinksInPath(normalized, allowedDirs); err != nil {
		return err
	}

	allowedRoot := ""
	for _, dir := range allowedDirs {
		normalizedDir, err := pathutil.NormalizePath(dir)
		if err != nil {
			continue
		}
		if normalized == normalizedDir || strings.HasPrefix(normalized, normalizedDir+string(filepath.Separator)) {
			if len(normalizedDir) > len(allowedRoot) {
				allowedRoot = normalizedDir
			}
		}
	}
	if allowedRoot == "" {
		return security.ErrPathOutsideAllowed
	}

	relative, err := filepath.Rel(allowedRoot, normalized)
	if err != nil {
		return err
	}
	if relative == "." {
		return nil
	}

	current := allowedRoot
	for _, part := range strings.Split(relative, string(filepath.Separator)) {
		if part == "" {
			continue
		}
		current = filepath.Join(current, part)

		info, err := os.Lstat(current)
		if err == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				return security.ErrSymlinkOperationDenied
			}
			if !info.IsDir() {
				return fmt.Errorf("path segment is not a directory: %s", current)
			}
			continue
		}
		if !os.IsNotExist(err) {
			return err
		}

		if _, err := security.ValidatePathForCreation(current, allowedDirs); err != nil {
			return err
		}
		if err := os.Mkdir(current, perm); err != nil {
			return err
		}
	}

	return nil
}

func ensureNoSymlink(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return security.ErrSymlinkOperationDenied
	}
	return nil
}
