package filesystem

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

// TreeEntry represents a node in a directory tree.
type TreeEntry struct {
	Name     string       `json:"name"`
	Type     string       `json:"type"` // "file" or "directory"
	Children []*TreeEntry `json:"children,omitempty"`
}

// EditOperation represents a single find/replace edit.
type EditOperation struct {
	OldText string `json:"oldText"`
	NewText string `json:"newText"`
}
