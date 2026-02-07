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

func TestValidateFinalPath(t *testing.T) {
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

	// Create a test file in disallowed directory
	disallowedFile := filepath.Join(disallowedDir, "secret.txt")
	if err := os.WriteFile(disallowedFile, []byte("secret"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a symlink in allowed directory pointing to allowed file
	symlinkToAllowed := filepath.Join(allowedDir, "link_to_allowed")
	if err := os.Symlink(testFile, symlinkToAllowed); err != nil {
		t.Skip("cannot create symlink")
	}

	// Create a symlink in allowed directory pointing to disallowed file
	symlinkToDisallowed := filepath.Join(allowedDir, "link_to_disallowed")
	if err := os.Symlink(disallowedFile, symlinkToDisallowed); err != nil {
		t.Skip("cannot create symlink")
	}

	allowedDirs := []string{allowedDir}

	tests := []struct {
		name        string
		path        string
		expectError error
	}{
		{"existing file", testFile, nil},
		{"existing symlink (rejected)", symlinkToAllowed, ErrSymlinkOperationDenied},
		{"symlink to disallowed (rejected)", symlinkToDisallowed, ErrSymlinkOperationDenied},
		{"non-existent file", filepath.Join(allowedDir, "nonexistent.txt"), os.ErrNotExist},
		{"null bytes", "test\x00.txt", ErrNullByte},
		{"empty path", "", ErrEmptyPath},
		{"path outside allowed dirs", disallowedFile, ErrPathOutsideAllowed},
		{"parent traversal", filepath.Join(allowedDir, "..", "disallowed", "secret.txt"), ErrPathOutsideAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateFinalPath(tt.path, allowedDirs)
			if tt.expectError != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.expectError)
				} else if tt.expectError == os.ErrNotExist {
					if !os.IsNotExist(err) {
						t.Errorf("expected os.IsNotExist error, got %v", err)
					}
				} else if err != tt.expectError && err.Error() != tt.expectError.Error() {
					t.Errorf("expected error %v, got %v", tt.expectError, err)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateFinalPathForCreation(t *testing.T) {
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
	testFile := filepath.Join(allowedDir, "existing.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a symlink in allowed directory
	symlinkPath := filepath.Join(allowedDir, "link")
	if err := os.Symlink(testFile, symlinkPath); err != nil {
		t.Skip("cannot create symlink")
	}

	// Create a nested directory structure
	nestedDir := filepath.Join(allowedDir, "nested", "deep")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a symlink directory in allowed
	symlinkDir := filepath.Join(allowedDir, "symlink_dir")
	if err := os.Symlink(disallowedDir, symlinkDir); err != nil {
		t.Skip("cannot create symlink")
	}

	allowedDirs := []string{allowedDir}

	tests := []struct {
		name        string
		path        string
		expectError error
	}{
		{"existing file", testFile, nil},
		{"existing symlink (rejected)", symlinkPath, ErrSymlinkOperationDenied},
		{"non-existent file in allowed dir", filepath.Join(allowedDir, "new.txt"), nil},
		{"non-existent nested file", filepath.Join(nestedDir, "new.txt"), nil},
		{"non-existent with non-existent parent", filepath.Join(allowedDir, "newdir", "new.txt"), nil},
		{"null bytes", "test\x00.txt", ErrNullByte},
		{"empty path", "", ErrEmptyPath},
		{"path outside allowed dirs", filepath.Join(disallowedDir, "new.txt"), ErrPathOutsideAllowed},
		{"parent traversal", filepath.Join(allowedDir, "..", "disallowed", "new.txt"), ErrPathOutsideAllowed},
		{"nested parent symlink", filepath.Join(symlinkDir, "new.txt"), ErrPathOutsideAllowed},
		{"no valid ancestor", "/nonexistent/deeply/nested/path/file.txt", ErrNoValidAncestor},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateFinalPathForCreation(tt.path, allowedDirs)
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
