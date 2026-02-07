package registry

import (
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/portertech/filesystem-mcp-server/internal/pathutil"
	"github.com/portertech/filesystem-mcp-server/internal/security"
)

// Validator defines path validation methods used by the registry.
type Validator interface {
	Validate(path string) (string, error)
	ValidateForCreation(path string) (string, error)
}

var _ Validator = (*Registry)(nil)

// Registry manages the list of allowed directories.
type Registry struct {
	mu       sync.RWMutex
	dirs     []string
	resolved []string // symlink-resolved versions of dirs, computed once at init
	logger   *slog.Logger
}

// New creates a new Registry with the given directories.
func New(dirs []string, logger *slog.Logger) *Registry {
	r := &Registry{
		logger: logger,
	}

	validDirs := make([]string, 0, len(dirs))
	resolvedDirs := make([]string, 0, len(dirs))
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

		// Pre-resolve symlinks for the allowed directory
		resolved, err := filepath.EvalSymlinks(normalized)
		if err != nil {
			// Fall back to normalized if resolution fails
			resolved = normalized
		}
		resolvedDirs = append(resolvedDirs, resolved)
	}

	r.dirs = validDirs
	r.resolved = resolvedDirs
	return r
}

// Set replaces the allowed directories with a new set.
func (r *Registry) Set(dirs []string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	validDirs := make([]string, 0, len(dirs))
	resolvedDirs := make([]string, 0, len(dirs))
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

		// Pre-resolve symlinks
		resolved, err := filepath.EvalSymlinks(normalized)
		if err != nil {
			resolved = normalized
		}
		resolvedDirs = append(resolvedDirs, resolved)
	}

	r.dirs = validDirs
	r.resolved = resolvedDirs
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

// GetResolved returns a copy of the symlink-resolved allowed directories.
// These are computed once at initialization or when Set() is called.
func (r *Registry) GetResolved() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]string, len(r.resolved))
	copy(result, r.resolved)
	return result
}

// Validate checks if a path is within allowed directories.
// Returns the resolved path if valid, or an error if not.
func (r *Registry) Validate(path string) (string, error) {
	r.mu.RLock()
	dirs := make([]string, len(r.dirs))
	copy(dirs, r.dirs)
	resolved := make([]string, len(r.resolved))
	copy(resolved, r.resolved)
	r.mu.RUnlock()

	return security.ValidatePathWithResolved(path, dirs, resolved)
}

// ValidateForCreation validates a path for file/directory creation.
func (r *Registry) ValidateForCreation(path string) (string, error) {
	r.mu.RLock()
	dirs := make([]string, len(r.dirs))
	copy(dirs, r.dirs)
	resolved := make([]string, len(r.resolved))
	copy(resolved, r.resolved)
	r.mu.RUnlock()

	return security.ValidatePathForCreationWithResolved(path, dirs, resolved)
}

// IsEmpty returns true if no directories are registered.
func (r *Registry) IsEmpty() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.dirs) == 0
}
