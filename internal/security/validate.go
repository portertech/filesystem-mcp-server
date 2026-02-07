package security

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/portertech/filesystem-mcp-server/internal/pathutil"
)

// Sentinel errors for path validation.
var (
	ErrAccessDenied           = errors.New("access denied")
	ErrPathOutsideAllowed     = errors.New("path is outside allowed directories")
	ErrSymlinkOutside         = errors.New("symlink target is outside allowed directories")
	ErrSymlinkOperationDenied = errors.New("operation denied: path is a symlink")
	ErrNullByte               = errors.New("path contains null byte")
	ErrEmptyPath              = errors.New("path is empty")
	ErrNoValidAncestor        = errors.New("no valid ancestor found within allowed directories")
)

// ValidatePath validates that a path is within allowed directories and safe to access.
// It returns the resolved absolute path if valid.
func ValidatePath(path string, allowedDirs []string) (string, error) {
	return ValidatePathWithResolved(path, allowedDirs, resolveAllowedDirs(allowedDirs))
}

// ValidatePathWithResolved validates a path using pre-resolved allowed directories.
// This avoids redundant symlink resolution when the caller has already resolved them.
func ValidatePathWithResolved(path string, allowedDirs []string, resolvedAllowed []string) (string, error) {
	if path == "" {
		return "", ErrEmptyPath
	}

	// Check for null bytes (path injection attack)
	if strings.ContainsRune(path, 0) {
		return "", ErrNullByte
	}

	// Normalize the path
	normalizedPath, err := pathutil.NormalizePath(path)
	if err != nil {
		return "", err
	}

	// For existing paths, resolve symlinks
	resolvedPath := normalizedPath
	if info, err := os.Lstat(normalizedPath); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			// It's a symlink, resolve it
			resolvedPath, err = filepath.EvalSymlinks(normalizedPath)
			if err != nil {
				return "", err
			}
		} else {
			// Not a symlink, but still resolve to handle parent symlinks
			resolvedPath, err = filepath.EvalSymlinks(normalizedPath)
			if err != nil {
				return "", err
			}
		}
	} else if os.IsNotExist(err) {
		// Path doesn't exist yet, validate the parent directory
		parentDir := filepath.Dir(normalizedPath)
		if parentInfo, parentErr := os.Stat(parentDir); parentErr == nil {
			if !parentInfo.IsDir() {
				return "", errors.New("parent path is not a directory")
			}
			// Resolve parent symlinks
			resolvedParent, evalErr := filepath.EvalSymlinks(parentDir)
			if evalErr != nil {
				return "", evalErr
			}
			resolvedPath = filepath.Join(resolvedParent, filepath.Base(normalizedPath))
		}
		// If parent doesn't exist, we'll validate the normalized path as-is
	}

	// Check if resolved path is within allowed directories
	if !IsPathWithinAllowedDirectories(resolvedPath, resolvedAllowed) {
		return "", ErrPathOutsideAllowed
	}

	return resolvedPath, nil
}

// ValidateFinalPath validates an existing path and rejects symlinks.
// It returns the resolved absolute path if valid.
func ValidateFinalPath(path string, allowedDirs []string) (string, error) {
	if path == "" {
		return "", ErrEmptyPath
	}

	if strings.ContainsRune(path, 0) {
		return "", ErrNullByte
	}

	normalizedPath, err := pathutil.NormalizePath(path)
	if err != nil {
		return "", err
	}

	info, err := os.Lstat(normalizedPath)
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", ErrSymlinkOperationDenied
	}

	resolvedPath, err := filepath.EvalSymlinks(normalizedPath)
	if err != nil {
		return "", err
	}

	resolvedAllowed := resolveAllowedDirs(allowedDirs)
	if !IsPathWithinAllowedDirectories(resolvedPath, resolvedAllowed) {
		return "", ErrPathOutsideAllowed
	}

	return resolvedPath, nil
}

// ValidateFinalPathForCreation validates a path for creation and rejects symlink destinations.
// It returns the resolved absolute path if valid.
func ValidateFinalPathForCreation(path string, allowedDirs []string) (string, error) {
	if path == "" {
		return "", ErrEmptyPath
	}

	if strings.ContainsRune(path, 0) {
		return "", ErrNullByte
	}

	normalizedPath, err := pathutil.NormalizePath(path)
	if err != nil {
		return "", err
	}

	if info, err := os.Lstat(normalizedPath); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return "", ErrSymlinkOperationDenied
		}
		resolvedPath, err := filepath.EvalSymlinks(normalizedPath)
		if err != nil {
			return "", err
		}

		resolvedAllowed := resolveAllowedDirs(allowedDirs)
		if !IsPathWithinAllowedDirectories(resolvedPath, resolvedAllowed) {
			return "", ErrPathOutsideAllowed
		}

		return resolvedPath, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}

	resolvedAllowed := resolveAllowedDirs(allowedDirs)

	parentDir := filepath.Dir(normalizedPath)
	resolvedPath := normalizedPath
	foundAncestor := false

	for parentDir != "/" && parentDir != "." {
		if _, err := os.Stat(parentDir); err == nil {
			resolvedParent, err := filepath.EvalSymlinks(parentDir)
			if err != nil {
				return "", err
			}
			relPath, err := filepath.Rel(parentDir, normalizedPath)
			if err != nil {
				return "", err
			}
			resolvedPath = filepath.Join(resolvedParent, relPath)
			foundAncestor = true
			break
		}
		parentDir = filepath.Dir(parentDir)
	}

	if !foundAncestor {
		return "", ErrNoValidAncestor
	}

	if !IsPathWithinAllowedDirectories(resolvedPath, resolvedAllowed) {
		return "", ErrPathOutsideAllowed
	}

	return resolvedPath, nil
}

// resolveAllowedDirs resolves symlinks in allowed directories.
func resolveAllowedDirs(allowedDirs []string) []string {
	resolved := make([]string, 0, len(allowedDirs))
	for _, dir := range allowedDirs {
		if r, err := filepath.EvalSymlinks(dir); err == nil {
			resolved = append(resolved, r)
		} else {
			// If we can't resolve, use the original
			resolved = append(resolved, dir)
		}
	}
	return resolved
}

// IsPathWithinAllowedDirectories checks if a path is within any of the allowed directories.
func IsPathWithinAllowedDirectories(path string, allowedDirs []string) bool {
	if len(allowedDirs) == 0 {
		return false
	}

	// Clean the path for comparison
	cleanPath := filepath.Clean(path)

	for _, allowed := range allowedDirs {
		cleanAllowed := filepath.Clean(allowed)

		// Check if path equals allowed directory
		if cleanPath == cleanAllowed {
			return true
		}

		// Check if path is under allowed directory
		// Add separator to prevent /tmp matching /tmpfoo
		if strings.HasPrefix(cleanPath, cleanAllowed+string(filepath.Separator)) {
			return true
		}
	}

	return false
}

// ValidateNoSymlinksInPath walks each segment of the path starting from the
// allowed root and verifies that no component is a symlink. Returns nil if all
// components are regular directories (or don't exist). This prevents symlink
// TOCTOU attacks during directory creation or file operations.
func ValidateNoSymlinksInPath(path string, allowedDirs []string) error {
	normalized, err := pathutil.NormalizePath(path)
	if err != nil {
		return err
	}

	// Find the matching allowed root
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
		return ErrPathOutsideAllowed
	}

	relative, err := filepath.Rel(allowedRoot, normalized)
	if err != nil {
		return err
	}
	if relative == "." {
		return nil
	}

	// Walk each path segment checking for symlinks
	current := allowedRoot
	for _, part := range strings.Split(relative, string(filepath.Separator)) {
		if part == "" {
			continue
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			if os.IsNotExist(err) {
				// Path segment doesn't exist yet, stop checking
				return nil
			}
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return ErrSymlinkOperationDenied
		}
	}

	return nil
}

// ValidatePathForCreation validates a path for file/directory creation.
// It checks that the parent directory is within allowed directories.
func ValidatePathForCreation(path string, allowedDirs []string) (string, error) {
	return ValidatePathForCreationWithResolved(path, allowedDirs, resolveAllowedDirs(allowedDirs))
}

// ValidatePathForCreationWithResolved validates a path for creation using pre-resolved allowed directories.
// This avoids redundant symlink resolution when the caller has already resolved them.
func ValidatePathForCreationWithResolved(path string, allowedDirs []string, resolvedAllowed []string) (string, error) {
	if path == "" {
		return "", ErrEmptyPath
	}

	if strings.ContainsRune(path, 0) {
		return "", ErrNullByte
	}

	normalizedPath, err := pathutil.NormalizePath(path)
	if err != nil {
		return "", err
	}

	// For new files, we need to validate based on where they would be created
	// Walk up the path to find the first existing parent
	parentDir := filepath.Dir(normalizedPath)
	resolvedPath := normalizedPath

	for parentDir != "/" && parentDir != "." {
		if _, err := os.Stat(parentDir); err == nil {
			// Parent exists, resolve it
			resolvedParent, err := filepath.EvalSymlinks(parentDir)
			if err != nil {
				return "", err
			}
			// Build the path relative to the resolved parent
			relPath, err := filepath.Rel(parentDir, normalizedPath)
			if err != nil {
				return "", err
			}
			resolvedPath = filepath.Join(resolvedParent, relPath)
			break
		}
		// Parent doesn't exist, try the grandparent
		parentDir = filepath.Dir(parentDir)
	}

	if !IsPathWithinAllowedDirectories(resolvedPath, resolvedAllowed) {
		return "", ErrPathOutsideAllowed
	}

	return resolvedPath, nil
}
