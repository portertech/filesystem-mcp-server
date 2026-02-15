package tools

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/portertech/filesystem-mcp-server/internal/registry"
	"github.com/portertech/filesystem-mcp-server/internal/security"
	"github.com/portertech/filesystem-mcp-server/pkg/filesystem"
	"github.com/spf13/cast"
)

// NewEditFileTool creates the edit_file tool.
func NewEditFileTool(reg *registry.Registry) mcp.Tool {
	return mcp.NewTool(
		"edit_file",
		mcp.WithDescription("Apply find/replace edits to a file. Supports exact matching and whitespace-normalized line matching. Returns a unified diff."),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithString("path", mcp.Description("Required. Absolute or relative path to the file to edit."), mcp.Required()),
		mcp.WithArray("edits", mcp.Description("Array of edit operations with oldText and newText"), mcp.Required(), mcp.Items(map[string]any{"type": "object"})),
		mcp.WithBoolean("dryRun", mcp.Description("If true, preview changes without writing")),
	)
}

// HandleEditFile handles the edit_file tool.
func HandleEditFile(ctx context.Context, reg *registry.Registry, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path := cast.ToString(request.Params.Arguments["path"])
	if path == "" {
		return mcp.NewToolResultError("path parameter is required"), nil
	}
	dryRun := cast.ToBool(request.Params.Arguments["dryRun"])

	// Parse edits
	var edits []filesystem.EditOperation
	if editsArg, ok := request.Params.Arguments["edits"].([]interface{}); ok {
		for _, e := range editsArg {
			if editMap, ok := e.(map[string]interface{}); ok {
				var requireUnique *bool
				if val, ok := editMap["requireUnique"]; ok {
					parsed := cast.ToBool(val)
					requireUnique = &parsed
				}
				var occurrence *int
				if val, ok := editMap["occurrence"]; ok {
					parsed := cast.ToInt(val)
					occurrence = &parsed
				}
				edits = append(edits, filesystem.EditOperation{
					OldText:       cast.ToString(editMap["oldText"]),
					NewText:       cast.ToString(editMap["newText"]),
					RequireUnique: requireUnique,
					Occurrence:    occurrence,
				})
			}
		}
	}

	// Use ValidateFinalPath to reject symlinks - editing through symlinks is a security risk
	resolvedPath, err := security.ValidateFinalPath(path, reg.Get())
	if err != nil {
		return mcp.NewToolResultError(fmt.Errorf("path validation failed: %w", err).Error()), nil
	}

	// Read original content
	originalData, err := os.ReadFile(resolvedPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Errorf("failed to read file: %w", err).Error()), nil
	}

	originalContent := string(originalData)
	newContent := originalContent

	// Apply edits sequentially
	for i, edit := range edits {
		if edit.OldText == "" {
			return mcp.NewToolResultError(fmt.Sprintf("edit %d: oldText cannot be empty", i+1)), nil
		}

		requireUnique := true
		if edit.RequireUnique != nil {
			requireUnique = *edit.RequireUnique
		}

		if edit.Occurrence != nil && *edit.Occurrence < 1 {
			return mcp.NewToolResultError(fmt.Sprintf("edit %d: occurrence must be >= 1", i+1)), nil
		}

		matchInfo, matchErr := findMatch(newContent, edit.OldText, requireUnique)
		if matchErr != nil {
			return mcp.NewToolResultError(fmt.Errorf("edit %d: %w", i+1, matchErr).Error()), nil
		}

		occurrence := 1
		if edit.Occurrence != nil {
			occurrence = *edit.Occurrence
		}
		if occurrence > len(matchInfo.Matches) {
			return mcp.NewToolResultError(fmt.Sprintf("edit %d: occurrence %d out of range", i+1, occurrence)), nil
		}

		newContent = applyMatch(newContent, edit.OldText, edit.NewText, matchInfo, occurrence)
	}

	// Generate unified diff
	diff := generateUnifiedDiff(resolvedPath, originalContent, newContent)

	if dryRun {
		return mcp.NewToolResultText(fmt.Sprintf("Dry run - changes not applied:\n\n%s", diff)), nil
	}

	// Write the changes atomically
	info, err := os.Stat(resolvedPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Errorf("failed to stat file: %w", err).Error()), nil
	}

	if err := atomicWriteFile(resolvedPath, []byte(newContent), info.Mode().Perm(), reg.Get()); err != nil {
		return mcp.NewToolResultError(fmt.Errorf("failed to write file: %w", err).Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Successfully edited %s\n\n%s", resolvedPath, diff)), nil
}

// normalizeWhitespace normalizes whitespace in text for fuzzy matching.
func normalizeWhitespace(s string) string {
	lines := strings.Split(s, "\n")
	var normalized []string
	for _, line := range lines {
		normalized = append(normalized, strings.TrimSpace(line))
	}
	return strings.Join(normalized, "\n")
}

// replaceWithIndentPreservation replaces text while preserving indentation.
type matchInfo struct {
	Matches           []int
	NormalizedMatches []int
	OldTextNormalized string
}

func findMatch(content, oldText string, requireUnique bool) (matchInfo, error) {
	if oldText == "" {
		return matchInfo{}, fmt.Errorf("oldText cannot be empty")
	}

	info := matchInfo{
		OldTextNormalized: normalizeWhitespace(oldText),
	}

	for idx := 0; idx <= len(content)-len(oldText); idx++ {
		if strings.HasPrefix(content[idx:], oldText) {
			info.Matches = append(info.Matches, idx)
		}
	}

	normalizedContent := normalizeWhitespace(content)
	if info.OldTextNormalized != "" {
		for idx := 0; idx <= len(normalizedContent)-len(info.OldTextNormalized); idx++ {
			if strings.HasPrefix(normalizedContent[idx:], info.OldTextNormalized) {
				info.NormalizedMatches = append(info.NormalizedMatches, idx)
			}
		}
	}

	if len(info.Matches) == 0 && len(info.NormalizedMatches) == 0 {
		return matchInfo{}, fmt.Errorf("oldText not found in file")
	}
	if requireUnique {
		count := len(info.Matches)
		if count == 0 {
			count = len(info.NormalizedMatches)
		}
		if count > 1 {
			return matchInfo{}, fmt.Errorf("oldText matches multiple locations")
		}
	}

	return info, nil
}

func applyMatch(content, oldText, newText string, info matchInfo, occurrence int) string {
	if len(info.Matches) > 0 {
		index := info.Matches[occurrence-1]
		return content[:index] + newText + content[index+len(oldText):]
	}
	return replaceWithIndentPreservation(content, oldText, newText, occurrence)
}

func replaceWithIndentPreservation(content, oldText, newText string, occurrence int) string {
	lines := strings.Split(content, "\n")
	oldLines := strings.Split(oldText, "\n")
	newLines := strings.Split(newText, "\n")

	// Find the start of oldText in content
	matchIndex := 0
	for i := 0; i <= len(lines)-len(oldLines); i++ {
		match := true
		var indent string

		for j, oldLine := range oldLines {
			contentLine := lines[i+j]
			trimmedContent := strings.TrimSpace(contentLine)
			trimmedOld := strings.TrimSpace(oldLine)

			if trimmedContent != trimmedOld {
				match = false
				break
			}

			// Capture indentation from first line
			if j == 0 {
				indent = getIndent(contentLine)
			}
		}

		if match {
			matchIndex++
			if matchIndex != occurrence {
				continue
			}

			// Build replacement with preserved indentation
			var result []string
			result = append(result, lines[:i]...)

			for k, newLine := range newLines {
				if k == 0 {
					result = append(result, indent+strings.TrimSpace(newLine))
				} else {
					// Preserve relative indentation
					origIndent := getIndent(newLine)
					result = append(result, indent+origIndent+strings.TrimSpace(newLine))
				}
			}

			result = append(result, lines[i+len(oldLines):]...)
			return strings.Join(result, "\n")
		}
	}

	// If no match found, return original
	return content
}

// getIndent returns the leading whitespace of a string.
func getIndent(s string) string {
	for i, r := range s {
		if r != ' ' && r != '\t' {
			return s[:i]
		}
	}
	return s
}

// generateUnifiedDiff generates a unified diff between two texts.
func generateUnifiedDiff(path, oldText, newText string) string {
	edits := myers.ComputeEdits(span.URIFromPath(path), oldText, newText)
	if len(edits) == 0 {
		return "No changes"
	}

	unified := gotextdiff.ToUnified("a/"+path, "b/"+path, oldText, edits)
	return fmt.Sprintf("%v", unified)
}
