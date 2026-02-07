package registry

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestRegistry(t *testing.T) {
	tmpDir := t.TempDir()
	dir1 := filepath.Join(tmpDir, "dir1")
	dir2 := filepath.Join(tmpDir, "dir2")

	if err := os.MkdirAll(dir1, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir2, 0755); err != nil {
		t.Fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	t.Run("New with valid directories", func(t *testing.T) {
		r := New([]string{dir1, dir2}, logger)
		dirs := r.Get()
		if len(dirs) != 2 {
			t.Errorf("expected 2 directories, got %d", len(dirs))
		}
	})

	t.Run("New filters invalid directories", func(t *testing.T) {
		r := New([]string{dir1, "/nonexistent/path"}, logger)
		dirs := r.Get()
		if len(dirs) != 1 {
			t.Errorf("expected 1 directory, got %d", len(dirs))
		}
	})

	t.Run("Set replaces directories", func(t *testing.T) {
		r := New([]string{dir1}, logger)
		r.Set([]string{dir2})
		dirs := r.Get()
		if len(dirs) != 1 {
			t.Errorf("expected 1 directory, got %d", len(dirs))
		}
		// Check base name matches (resolved path may differ, e.g., /var -> /private/var)
		if filepath.Base(dirs[0]) != filepath.Base(dir2) {
			t.Errorf("expected dir2, got %v", dirs[0])
		}
	})

	t.Run("Validate allows valid paths", func(t *testing.T) {
		r := New([]string{dir1}, logger)
		testFile := filepath.Join(dir1, "test.txt")

		// Create the file first
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}

		resolved, err := r.Validate(testFile)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		// The resolved path should end with the same relative path
		// (it may be resolved, e.g., /var -> /private/var on macOS)
		if !strings.HasSuffix(resolved, filepath.Join("dir1", "test.txt")) {
			t.Errorf("expected path ending with dir1/test.txt, got %s", resolved)
		}
	})

	t.Run("Validate rejects paths outside allowed", func(t *testing.T) {
		r := New([]string{dir1}, logger)
		_, err := r.Validate(filepath.Join(dir2, "test.txt"))
		if err == nil {
			t.Error("expected error for path outside allowed directories")
		}
	})

	t.Run("IsEmpty", func(t *testing.T) {
		r := New([]string{}, logger)
		if !r.IsEmpty() {
			t.Error("expected empty registry")
		}

		r = New([]string{dir1}, logger)
		if r.IsEmpty() {
			t.Error("expected non-empty registry")
		}
	})
}

func TestRegistryConcurrency(t *testing.T) {
	tmpDir := t.TempDir()
	dir1 := filepath.Join(tmpDir, "dir1")
	dir2 := filepath.Join(tmpDir, "dir2")

	if err := os.MkdirAll(dir1, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir2, 0755); err != nil {
		t.Fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	r := New([]string{dir1}, logger)

	var wg sync.WaitGroup

	// Concurrent reads
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = r.Get()
		}()
	}

	// Concurrent writes
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if i%2 == 0 {
				r.Set([]string{dir1})
			} else {
				r.Set([]string{dir2})
			}
		}(i)
	}

	wg.Wait()
}
