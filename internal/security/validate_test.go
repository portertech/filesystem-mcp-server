package security

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidatePath(t *testing.T) {
	// Create temp directory structure for testing
	tmpDir := t.TempDir()
	allowedDir := filepath.Join(tmpDir, "allowed")
	disallowedDir := filepath.Join(tmpDir, "disallowed")

	if err := os.MkdirAll(allowedDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(disallowedDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a test file in allowed directory
	testFile := filepath.Join(allowedDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	allowedDirs := []string{allowedDir}

	tests := []struct {
		name        string
		path        string
		expectError error
	}{
		{"allowed file", testFile, nil},
		{"allowed dir", allowedDir, nil},
		{"allowed new file", filepath.Join(allowedDir, "new.txt"), nil},
		{"disallowed file", filepath.Join(disallowedDir, "test.txt"), ErrPathOutsideAllowed},
		{"parent traversal", filepath.Join(allowedDir, "..", "disallowed", "test.txt"), ErrPathOutsideAllowed},
		{"empty path", "", ErrEmptyPath},
		{"null byte", "test\x00.txt", ErrNullByte},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidatePath(tt.path, allowedDirs)
			if tt.expectError != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.expectError)
				} else if err != tt.expectError && err.Error() != tt.expectError.Error() {
					t.Errorf("expected error %v, got %v", tt.expectError, err)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidatePathSymlinkAttack(t *testing.T) {
	tmpDir := t.TempDir()
	allowedDir := filepath.Join(tmpDir, "allowed")
	secretDir := filepath.Join(tmpDir, "secret")

	if err := os.MkdirAll(allowedDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(secretDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a secret file
	secretFile := filepath.Join(secretDir, "secret.txt")
	if err := os.WriteFile(secretFile, []byte("secret"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a symlink in allowed directory pointing to secret file
	symlinkPath := filepath.Join(allowedDir, "link")
	if err := os.Symlink(secretFile, symlinkPath); err != nil {
		t.Skip("cannot create symlink")
	}

	allowedDirs := []string{allowedDir}

	// Symlink should be rejected because it points outside allowed directories
	_, err := ValidatePath(symlinkPath, allowedDirs)
	if err != ErrPathOutsideAllowed {
		t.Errorf("expected ErrPathOutsideAllowed, got %v", err)
	}
}

func TestIsPathWithinAllowedDirectories(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		allowedDirs []string
		expected    bool
	}{
		{"exact match", "/tmp/allowed", []string{"/tmp/allowed"}, true},
		{"subdirectory", "/tmp/allowed/sub/file.txt", []string{"/tmp/allowed"}, true},
		{"outside", "/tmp/other/file.txt", []string{"/tmp/allowed"}, false},
		{"prefix attack", "/tmp/allowedfoo/file.txt", []string{"/tmp/allowed"}, false},
		{"multiple allowed", "/tmp/dir2/file.txt", []string{"/tmp/dir1", "/tmp/dir2"}, true},
		{"empty allowed", "/tmp/file.txt", []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsPathWithinAllowedDirectories(tt.path, tt.allowedDirs)
			if result != tt.expected {
				t.Errorf("IsPathWithinAllowedDirectories(%q, %v) = %v, want %v",
					tt.path, tt.allowedDirs, result, tt.expected)
			}
		})
	}
}
