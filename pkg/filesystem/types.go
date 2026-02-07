// Package filesystem provides types for filesystem operations in the MCP server.
// These types are used for serializing file metadata, directory trees, and edit
// operations between the MCP client and server.
package filesystem

import "fmt"

// FileInfo contains metadata about a file or directory.
// It is returned by the get_file_info tool and provides details such as
// size, timestamps, type (file/directory), and Unix permissions.
type FileInfo struct {
	Size        int64  `json:"size"`
	Created     string `json:"created,omitempty"`
	Modified    string `json:"modified"`
	Accessed    string `json:"accessed,omitempty"`
	IsDirectory bool   `json:"isDirectory"`
	IsFile      bool   `json:"isFile"`
	Permissions string `json:"permissions"`
}

func (f FileInfo) String() string {
	return fmt.Sprintf("FileInfo{Size:%d, Created:%q, Modified:%q, Accessed:%q, IsDirectory:%t, IsFile:%t, Permissions:%q}", f.Size, f.Created, f.Modified, f.Accessed, f.IsDirectory, f.IsFile, f.Permissions)
}

// TreeEntry represents a node in a directory tree structure.
// It is returned by the directory_tree tool and recursively contains
// child entries for directories. Type is either "file" or "directory".
type TreeEntry struct {
	Name     string       `json:"name"`
	Type     string       `json:"type"` // "file" or "directory"
	Children []*TreeEntry `json:"children,omitempty"`
}

func (t TreeEntry) String() string {
	return fmt.Sprintf("TreeEntry{Name:%q, Type:%q, Children:%d}", t.Name, t.Type, len(t.Children))
}

// EditOperation represents a single find/replace edit for the edit_file tool.
// OldText specifies the text to find, NewText specifies the replacement.
// If RequireUnique is true (default), the operation fails if OldText appears
// more than once. Occurrence can select a specific match (1-indexed) when
// multiple matches exist.
type EditOperation struct {
	OldText       string `json:"oldText"`
	NewText       string `json:"newText"`
	RequireUnique *bool  `json:"requireUnique,omitempty"`
	Occurrence    *int   `json:"occurrence,omitempty"`
}
