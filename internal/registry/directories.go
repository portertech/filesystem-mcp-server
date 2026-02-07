package registry

import (
	"log/slog"
	"os"
	"sync"

	"github.com/portertech/filesystem-mcp-server/internal/pathutil"
	"github.com/portertech/filesystem-mcp-server/internal/security"
)

// Registry manages the list of allowed directories.
type Registry struct {
	mu     sync.RWMutex
	dirs   []string
	logger *slog.Logger
}

// New creates a new Registry with the given directories.
func New(dirs []string, logger *slog.Logger) *Registry {
	r := &Registry{
		logger: logger,
	}

	validDirs := make([]string, 0, len(dirs))
	for _, d := range dirs {
		normalized, err := pathutil.NormalizePath(d)
		if err != nil {
			logger.Warn("failed to normalize directory", "dir", d, "error", err)
			continue
		}

		// Check if directory exists and is accessible
		info, err := os.Stat(normalized)
		if err != nil {
			logger.Warn("directory not accessible", "dir", normalized, "error", err)
			continue
		}
		if !info.IsDir() {
			logger.Warn("path is not a directory", "path", normalized)
			continue
		}

		validDirs = append(validDirs, normalized)
		logger.Debug("added allowed directory", "dir", normalized)
	}

	r.dirs = validDirs
	return r
}

// Set replaces the allowed directories with a new set.
func (r *Registry) Set(dirs []string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	validDirs := make([]string, 0, len(dirs))
	for _, d := range dirs {
		normalized, err := pathutil.NormalizePath(d)
		if err != nil {
			r.logger.Warn("failed to normalize directory", "dir", d, "error", err)
			continue
		}

		info, err := os.Stat(normalized)
		if err != nil {
			r.logger.Warn("directory not accessible", "dir", normalized, "error", err)
			continue
		}
		if !info.IsDir() {
			r.logger.Warn("path is not a directory", "path", normalized)
			continue
		}

		validDirs = append(validDirs, normalized)
	}

	r.dirs = validDirs
	r.logger.Info("updated allowed directories", "count", len(validDirs))
}

// Get returns a copy of the allowed directories.
func (r *Registry) Get() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]string, len(r.dirs))
	copy(result, r.dirs)
	return result
}

// Validate checks if a path is within allowed directories.
// Returns the resolved path if valid, or an error if not.
func (r *Registry) Validate(path string) (string, error) {
	r.mu.RLock()
	dirs := r.dirs
	r.mu.RUnlock()

	return security.ValidatePath(path, dirs)
}

// ValidateForCreation validates a path for file/directory creation.
func (r *Registry) ValidateForCreation(path string) (string, error) {
	r.mu.RLock()
	dirs := r.dirs
	r.mu.RUnlock()

	return security.ValidatePathForCreation(path, dirs)
}

// IsEmpty returns true if no directories are registered.
func (r *Registry) IsEmpty() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.dirs) == 0
}
