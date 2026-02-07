package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/portertech/filesystem-mcp-server/internal/registry"
	"github.com/portertech/filesystem-mcp-server/internal/stream"
	"github.com/spf13/cast"
)

// MIME type mappings
var mimeTypes = map[string]string{
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".gif":  "image/gif",
	".webp": "image/webp",
	".bmp":  "image/bmp",
	".svg":  "image/svg+xml",
	".mp3":  "audio/mpeg",
	".wav":  "audio/wav",
	".ogg":  "audio/ogg",
	".flac": "audio/flac",
}

// NewReadMediaFileTool creates the read_media_file tool.
func NewReadMediaFileTool(reg *registry.Registry) mcp.Tool {
	return mcp.NewTool(
		"read_media_file",
		mcp.WithDescription("Read a media file (image or audio) and return it as base64-encoded data."),
		mcp.WithString("path", mcp.Description("Path to the media file to read"), mcp.Required()),
	)
}

// HandleReadMediaFile handles the read_media_file tool.
func HandleReadMediaFile(ctx context.Context, reg *registry.Registry, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path := cast.ToString(request.Params.Arguments["path"])

	resolvedPath, err := reg.Validate(path)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("path validation failed: %v", err)), nil
	}

	// Get MIME type from extension
	ext := strings.ToLower(filepath.Ext(resolvedPath))
	mimeType, ok := mimeTypes[ext]
	if !ok {
		return mcp.NewToolResultError(fmt.Sprintf("unsupported media type: %s", ext)), nil
	}

	// Stream to base64
	base64Data, err := stream.StreamToBase64(resolvedPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to read media file: %v", err)), nil
	}

	// Determine content type category
	var contentType string
	if strings.HasPrefix(mimeType, "image/") {
		contentType = "image"
	} else if strings.HasPrefix(mimeType, "audio/") {
		contentType = "audio"
	} else {
		contentType = "blob"
	}

	// Return as embedded content
	result := map[string]interface{}{
		"type":     contentType,
		"mimeType": mimeType,
		"data":     base64Data,
	}

	jsonResult, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonResult)), nil
}
