package pathutil

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestExpandHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot get home directory")
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"tilde only", "~", home},
		{"tilde with slash", "~/Documents", filepath.Join(home, "Documents")},
		{"no tilde", "/tmp/foo", "/tmp/foo"},
		{"tilde in middle", "/home/~user", "/home/~user"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandHome(tt.input)
			if result != tt.expected {
				t.Errorf("ExpandHome(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNormalizePath(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		input    string
		contains string // substring that should be in result
	}{
		{"relative path", "foo/bar", "foo"},
		{"dot path", "./foo", "foo"},
		{"parent path", "../foo", "foo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NormalizePath(tt.input)
			if err != nil {
				t.Fatalf("NormalizePath(%q) error: %v", tt.input, err)
			}
			if !strings.Contains(result, tt.contains) {
				t.Errorf("NormalizePath(%q) = %q, should contain %q", tt.input, result, tt.contains)
			}
			if !filepath.IsAbs(result) {
				t.Errorf("NormalizePath(%q) = %q, should be absolute", tt.input, result)
			}
		})
	}

	// Test that normalized paths are clean
	t.Run("clean path", func(t *testing.T) {
		result, err := NormalizePath(filepath.Join(cwd, "foo", "..", "bar"))
		if err != nil {
			t.Fatal(err)
		}
		expected := filepath.Join(cwd, "bar")
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})
}

func TestIsAbsolute(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"unix absolute", "/foo/bar", true},
		{"relative", "foo/bar", false},
		{"dot relative", "./foo", false},
		{"parent relative", "../foo", false},
	}

	// Add Windows-specific tests
	if runtime.GOOS == "windows" {
		tests = append(tests, []struct {
			name     string
			input    string
			expected bool
		}{
			{"windows drive", "C:\\foo", true},
			{"windows drive lowercase", "c:\\foo", true},
			{"windows drive forward slash", "C:/foo", true},
		}...)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAbsolute(tt.input)
			if result != tt.expected {
				t.Errorf("IsAbsolute(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}
