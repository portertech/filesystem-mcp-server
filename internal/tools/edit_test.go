package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestHandleEditFile(t *testing.T) {
	reg, tmpDir := setupTestRegistry(t)

	tests := []struct {
		name            string
		initialContent  string
		args            map[string]any
		expectedContent string
		isError         bool
	}{
		{
			name:           "simple replacement",
			initialContent: "hello world",
			args: map[string]any{
				"path": filepath.Join(tmpDir, "edit1.txt"),
				"edits": []interface{}{
					map[string]interface{}{"oldText": "world", "newText": "universe"},
				},
			},
			expectedContent: "hello universe",
		},
		{
			name:           "multiple edits",
			initialContent: "foo bar baz",
			args: map[string]any{
				"path": filepath.Join(tmpDir, "edit2.txt"),
				"edits": []interface{}{
					map[string]interface{}{"oldText": "foo", "newText": "FOO"},
					map[string]interface{}{"oldText": "baz", "newText": "BAZ"},
				},
			},
			expectedContent: "FOO bar BAZ",
		},
		{
			name:           "dry run",
			initialContent: "test content",
			args: map[string]any{
				"path": filepath.Join(tmpDir, "edit3.txt"),
				"edits": []interface{}{
					map[string]interface{}{"oldText": "test", "newText": "TEST"},
				},
				"dryRun": true,
			},
			expectedContent: "test content", // unchanged
		},
		{
			name:           "text not found",
			initialContent: "hello world",
			args: map[string]any{
				"path": filepath.Join(tmpDir, "edit4.txt"),
				"edits": []interface{}{
					map[string]interface{}{"oldText": "notfound", "newText": "replacement"},
				},
			},
			isError: true,
		},
		{
			name:           "multiple matches fail by default",
			initialContent: "repeat repeat",
			args: map[string]any{
				"path": filepath.Join(tmpDir, "edit5.txt"),
				"edits": []interface{}{
					map[string]interface{}{"oldText": "repeat", "newText": "done"},
				},
			},
			isError: true,
		},
		{
			name:           "multiple matches with occurrence",
			initialContent: "repeat repeat",
			args: map[string]any{
				"path": filepath.Join(tmpDir, "edit6.txt"),
				"edits": []interface{}{
					map[string]interface{}{"oldText": "repeat", "newText": "done", "requireUnique": false, "occurrence": 2},
				},
			},
			expectedContent: "repeat done",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.args["path"].(string)
			// Create initial file
			if err := os.WriteFile(path, []byte(tt.initialContent), 0644); err != nil {
				t.Fatal(err)
			}

			request := mcp.CallToolRequest{}
			request.Params.Arguments = tt.args

			result, err := HandleEditFile(context.Background(), reg, request)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.isError {
				if !result.IsError {
					t.Error("expected error result")
				}
			} else {
				if result.IsError {
					t.Errorf("unexpected error: %v", result.Content)
				}

				// Verify content
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("failed to read file: %v", err)
				}
				if string(data) != tt.expectedContent {
					t.Errorf("content mismatch: got %q, want %q", string(data), tt.expectedContent)
				}
			}
		})
	}
}

func TestGenerateUnifiedDiffPreservesSpecialCharacters(t *testing.T) {
	oldText := "url := `ws://127.0.0.1:1234`\nlist := []string{\"a|b\", \"c^d\"}\npath := `C:\\\\tmp\\\\file`\n"
	newText := "url := `ws://127.0.0.1:5678`\nlist := []string{\"a|b\", \"c^d\"}\npath := `C:\\\\tmp\\\\file`\nextra := map[string]string{\"key\": \"value\"}\n"
	diff := generateUnifiedDiff("test.txt", oldText, newText)

	expected := []string{
		"`ws://127.0.0.1:5678`",
		"[]string{\"a|b\", \"c^d\"}",
		"`C:\\\\tmp\\\\file`",
		"map[string]string{\"key\": \"value\"}",
	}
	for _, snippet := range expected {
		if !strings.Contains(diff, snippet) {
			t.Errorf("diff missing %q", snippet)
		}
	}

	unexpected := []string{"%60", "%5B", "%5D", "%7B", "%7D", "%7C", "%5C", "%5E", "%3A", "%2F"}
	for _, snippet := range unexpected {
		if strings.Contains(diff, snippet) {
			t.Errorf("diff contains encoded sequence %q", snippet)
		}
	}
}

func TestNormalizeWhitespace(t *testing.T) {
	input := "  hello  \n  world  "
	expected := "hello\nworld"
	result := normalizeWhitespace(input)
	if result != expected {
		t.Errorf("normalizeWhitespace = %q, want %q", result, expected)
	}
}

func TestGetIndent(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", ""},
		{"  hello", "  "},
		{"\thello", "\t"},
		{"    ", "    "},
	}

	for _, tt := range tests {
		result := getIndent(tt.input)
		if result != tt.expected {
			t.Errorf("getIndent(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestHandleEditFileRejectsSymlinks(t *testing.T) {
	reg, tmpDir := setupTestRegistry(t)

	// Create a real file
	realFile := filepath.Join(tmpDir, "real.txt")
	if err := os.WriteFile(realFile, []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a symlink to the real file
	linkFile := filepath.Join(tmpDir, "link.txt")
	if err := os.Symlink(realFile, linkFile); err != nil {
		t.Skip("cannot create symlink")
	}

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]any{
		"path": linkFile,
		"edits": []interface{}{
			map[string]interface{}{"oldText": "hello", "newText": "goodbye"},
		},
	}

	result, err := HandleEditFile(context.Background(), reg, request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.IsError {
		t.Error("expected error when editing through symlink")
	}

	// Verify original file is unchanged
	data, err := os.ReadFile(realFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(data) != "hello world" {
		t.Errorf("original file should be unchanged, got %q", string(data))
	}
}
