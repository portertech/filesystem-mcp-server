package stream

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestHeadFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	content := "line1\nline2\nline3\nline4\nline5"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		n        int
		expected string
	}{
		{"first 2 lines", 2, "line1\nline2"},
		{"first 3 lines", 3, "line1\nline2\nline3"},
		{"more than available", 10, "line1\nline2\nline3\nline4\nline5"},
		{"zero lines", 0, ""},
		{"negative lines", -1, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := HeadFile(testFile, tt.n)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("HeadFile(%d) = %q, want %q", tt.n, result, tt.expected)
			}
		})
	}
}

func TestTailFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	content := "line1\nline2\nline3\nline4\nline5"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		n        int
		expected string
	}{
		{"last 2 lines", 2, "line4\nline5"},
		{"last 3 lines", 3, "line3\nline4\nline5"},
		{"more than available", 10, "line1\nline2\nline3\nline4\nline5"},
		{"zero lines", 0, ""},
		{"negative lines", -1, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := TailFile(testFile, tt.n)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("TailFile(%d) = %q, want %q", tt.n, result, tt.expected)
			}
		})
	}
}

func TestTailFileLargeLine(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "large-line.txt")

	longLine := strings.Repeat("x", TailChunkSize*3)
	content := "first\n" + longLine + "\nlast\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := TailFile(testFile, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != longLine+"\nlast" {
		t.Fatalf("unexpected tail result length: got %d", len(result))
	}
}

func TestTailFileNoTrailingNewline(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "no-trailing.txt")

	content := "line1\nline2\nline3"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := TailFile(testFile, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "line2\nline3" {
		t.Fatalf("TailFile returned %q", result)
	}
}

func TestTailFileVeryLarge(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "very-large.txt")

	var builder strings.Builder
	lineCount := 5000
	for i := 1; i <= lineCount; i++ {
		builder.WriteString("line")
		builder.WriteString(strconv.Itoa(i))
		builder.WriteString("\n")
	}
	if err := os.WriteFile(testFile, []byte(builder.String()), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := TailFile(testFile, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "line4998\nline4999\nline5000" {
		t.Fatalf("TailFile returned %q", result)
	}
}

func TestCopyFileStreaming(t *testing.T) {
	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "source.txt")
	dstFile := filepath.Join(tmpDir, "dest.txt")

	content := strings.Repeat("test content\n", 1000)
	if err := os.WriteFile(srcFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	if err := CopyFileStreaming(srcFile, dstFile); err != nil {
		t.Fatalf("CopyFileStreaming error: %v", err)
	}

	// Verify content
	copied, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(copied) != content {
		t.Error("copied content does not match original")
	}

	// Verify permissions
	srcInfo, _ := os.Stat(srcFile)
	dstInfo, _ := os.Stat(dstFile)
	if srcInfo.Mode() != dstInfo.Mode() {
		t.Errorf("permissions mismatch: src=%v, dst=%v", srcInfo.Mode(), dstInfo.Mode())
	}
}

func TestStreamToBase64(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	content := "Hello, World!"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := StreamToBase64(testFile)
	if err != nil {
		t.Fatalf("StreamToBase64 error: %v", err)
	}

	expected := "SGVsbG8sIFdvcmxkIQ=="
	if result != expected {
		t.Errorf("StreamToBase64 = %q, want %q", result, expected)
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		size     int64
		expected string
	}{
		{0, "0 B"},
		{100, "100 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatSize(tt.size)
			if result != tt.expected {
				t.Errorf("FormatSize(%d) = %q, want %q", tt.size, result, tt.expected)
			}
		})
	}
}
