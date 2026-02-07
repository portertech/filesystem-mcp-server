package pathutil

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// NormalizePath converts a path to an absolute, clean path.
func NormalizePath(path string) (string, error) {
	path = ExpandHome(path)

	// Handle Windows-specific paths
	if runtime.GOOS == "windows" {
		path = normalizeWindowsPath(path)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	return filepath.Clean(absPath), nil
}

// ExpandHome expands ~ to the user's home directory.
func ExpandHome(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	if path == "~" {
		return home
	}

	if strings.HasPrefix(path, "~/") || strings.HasPrefix(path, "~\\") {
		return filepath.Join(home, path[2:])
	}

	return path
}

// normalizeWindowsPath handles Windows-specific path formats.
func normalizeWindowsPath(path string) string {
	// Handle WSL /mnt/ paths
	if strings.HasPrefix(path, "/mnt/") && len(path) > 5 {
		driveLetter := path[5]
		if (driveLetter >= 'A' && driveLetter <= 'Z') || (driveLetter >= 'a' && driveLetter <= 'z') {
			drive := strings.ToUpper(string(driveLetter))
			rest := ""
			if len(path) > 6 {
				rest = path[6:]
			}
			return drive + ":" + rest
		}
	}

	// Handle UNC paths
	if strings.HasPrefix(path, "//") || strings.HasPrefix(path, "\\\\") {
		return path
	}

	// Convert forward slashes to backslashes for consistency
	return strings.ReplaceAll(path, "/", "\\")
}

// IsAbsolute checks if a path is absolute, handling cross-platform differences.
func IsAbsolute(path string) bool {
	if filepath.IsAbs(path) {
		return true
	}

	// Windows drive letter check
	if runtime.GOOS == "windows" && len(path) >= 2 {
		if path[1] == ':' && ((path[0] >= 'A' && path[0] <= 'Z') || (path[0] >= 'a' && path[0] <= 'z')) {
			return true
		}
	}

	return false
}
