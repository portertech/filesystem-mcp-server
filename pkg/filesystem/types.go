package filesystem

import "fmt"

// FileInfo contains metadata about a file or directory.
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

// TreeEntry represents a node in a directory tree.
type TreeEntry struct {
	Name     string       `json:"name"`
	Type     string       `json:"type"` // "file" or "directory"
	Children []*TreeEntry `json:"children,omitempty"`
}

func (t TreeEntry) String() string {
	return fmt.Sprintf("TreeEntry{Name:%q, Type:%q, Children:%d}", t.Name, t.Type, len(t.Children))
}

// EditOperation represents a single find/replace edit.
type EditOperation struct {
	OldText       string `json:"oldText"`
	NewText       string `json:"newText"`
	RequireUnique *bool  `json:"requireUnique,omitempty"`
	Occurrence    *int   `json:"occurrence,omitempty"`
}
