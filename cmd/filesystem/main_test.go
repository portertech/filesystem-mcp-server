package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	// Build the binary before running tests
	cmd := exec.Command("go", "build", "-o", "filesystem-test-binary", ".")
	cmd.Dir = "."
	if err := cmd.Run(); err != nil {
		panic("failed to build test binary: " + err.Error())
	}
	code := m.Run()
	os.Remove("filesystem-test-binary")
	os.Exit(code)
}

func binaryPath(t *testing.T) string {
	t.Helper()
	// Get the directory of the test file
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	return filepath.Join(wd, "filesystem-test-binary")
}

func TestVersionFlag(t *testing.T) {
	bin := binaryPath(t)
	cmd := exec.Command(bin, "-version")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// Should output version string (default is "dev")
	out := strings.TrimSpace(string(output))
	if out == "" {
		t.Error("expected non-empty version output")
	}
}

func TestListFlagWithDirectories(t *testing.T) {
	bin := binaryPath(t)
	tmpDir := t.TempDir()
	cmd := exec.Command(bin, "-list", tmpDir)
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	out := strings.TrimSpace(string(output))
	if out != tmpDir {
		t.Errorf("expected %q, got %q", tmpDir, out)
	}
}

func TestListFlagWithNoDirectories(t *testing.T) {
	bin := binaryPath(t)
	cmd := exec.Command(bin, "-list")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// Should output nothing when no directories specified
	out := strings.TrimSpace(string(output))
	if out != "" {
		t.Errorf("expected empty output, got %q", out)
	}
}

func TestListFlagWithInvalidDirectory(t *testing.T) {
	bin := binaryPath(t)
	cmd := exec.Command(bin, "-list", "/nonexistent/path/that/does/not/exist")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// Invalid directories are filtered out, so output should be empty
	out := strings.TrimSpace(string(output))
	if out != "" {
		t.Errorf("expected empty output for invalid directory, got %q", out)
	}
}

func TestListFlagWithMultipleDirectories(t *testing.T) {
	bin := binaryPath(t)
	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()
	cmd := exec.Command(bin, "-list", tmpDir1, tmpDir2)
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	out := strings.TrimSpace(string(output))
	lines := strings.Split(out, "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 directories, got %d: %v", len(lines), lines)
	}
	// Both directories should be in the output (order may vary based on normalization)
	outStr := string(output)
	if !strings.Contains(outStr, tmpDir1) || !strings.Contains(outStr, tmpDir2) {
		t.Errorf("expected both directories in output, got %q", out)
	}
}

func TestHelpFlag(t *testing.T) {
	bin := binaryPath(t)
	cmd := exec.Command(bin, "-help")
	// -help causes exit code 0 but outputs to stderr
	output, _ := cmd.CombinedOutput()
	out := string(output)
	// Should contain usage information
	if !strings.Contains(out, "-version") {
		t.Error("expected help output to contain -version flag")
	}
	if !strings.Contains(out, "-list") {
		t.Error("expected help output to contain -list flag")
	}
	if !strings.Contains(out, "-verbose") {
		t.Error("expected help output to contain -verbose flag")
	}
}

func TestInvalidFlag(t *testing.T) {
	bin := binaryPath(t)
	cmd := exec.Command(bin, "-invalidflag")
	err := cmd.Run()
	if err == nil {
		t.Error("expected error for invalid flag")
	}
	// Check that it's an exit error with non-zero status
	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() == 0 {
			t.Error("expected non-zero exit code for invalid flag")
		}
	}
}
