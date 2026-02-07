package server

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/portertech/filesystem-mcp-server/internal/pathutil"
	"github.com/portertech/filesystem-mcp-server/internal/registry"
)

// UpdateFromRoots updates the allowed directories from MCP roots.
func UpdateFromRoots(reg *registry.Registry, roots []string, logger *slog.Logger) {
	var validDirs []string

	for _, root := range roots {
		// Handle file:// URIs
		path := root
		if strings.HasPrefix(root, "file://") {
			path = strings.TrimPrefix(root, "file://")
		}

		// Normalize the path
		normalized, err := pathutil.NormalizePath(path)
		if err != nil {
			logger.Warn("failed to normalize root path", "path", path, "error", err)
			continue
		}

		// Verify it exists and is a directory
		info, err := os.Stat(normalized)
		if err != nil {
			logger.Warn("root path not accessible", "path", normalized, "error", err)
			continue
		}

		if !info.IsDir() {
			// If it's a file, use its parent directory
			normalized = filepath.Dir(normalized)
		}

		validDirs = append(validDirs, normalized)
		logger.Debug("added root directory", "path", normalized)
	}

	if len(validDirs) > 0 {
		reg.Set(validDirs)
		logger.Info("updated allowed directories from roots", "count", len(validDirs))
	}
}
