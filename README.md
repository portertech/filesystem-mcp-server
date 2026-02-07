# Filesystem MCP Server

A Go implementation of a Model Context Protocol (MCP) server providing secure filesystem operations with directory restrictions.

## Features

- **17 filesystem tools** for comprehensive file operations
- **Secure by default**: Only operates within explicitly allowed directories
- **Symlink attack prevention**: Resolves symlinks and validates targets
- **Streaming support**: Memory-efficient handling of large files
- **Atomic writes**: Uses temp files with rename for safe file operations
- **Cross-platform**: Works on Linux, macOS, and Windows

## Installation

```bash
go install github.com/portertech/filesystem-mcp-server/cmd/filesystem@latest
```

Or build from source:

```bash
git clone https://github.com/portertech/filesystem-mcp-server.git
cd filesystem-mcp-server
make build
```

## Usage

```bash
# Start server with allowed directories
./filesystem /path/to/allowed/dir1 /path/to/allowed/dir2

# Enable verbose logging
./filesystem -verbose /path/to/dir

# Show version
./filesystem -version

# List allowed directories
./filesystem -list /path/to/dir
```

## Available Tools

### `read_text_file`

Read the contents of a text file with optional partial read and line numbers.

**Parameters**:

- `path` (required): Path to the file to read
- `head` (optional): Number of lines to read from the start
- `tail` (optional): Number of lines to read from the end
- `line_numbers` (optional): Include line numbers in output (default: false)

**Returns**: File contents as text, optionally with line numbers prefixed

### `read_file`

Read the complete contents of a file. *Deprecated: use `read_text_file` instead.*

**Parameters**:

- `path` (required): Path to the file to read

**Returns**: File contents as text

### `read_multiple_files`

Read multiple files concurrently. Failed reads for individual files won't stop the entire operation.

**Parameters**:

- `paths` (required): Array of file paths to read

**Returns**: Array of file contents with their paths

### `read_media_file`

Read a media file and return its contents as base64-encoded data.

**Parameters**:

- `path` (required): Path to the media file

**Returns**: Base64-encoded file data with MIME type

### `write_file`

Create or overwrite a file with new content using atomic writes (temp file + rename).

**Parameters**:

- `path` (required): Path to the file to write
- `content` (required): Content to write to the file

**Returns**: Success confirmation

### `edit_file`

Apply find/replace edits to a text file with git-style diff output.

**Parameters**:

- `path` (required): Path to the file to edit
- `edits` (required): Array of edit operations, each with:
  - `oldText`: Text to search for (must match exactly)
  - `newText`: Text to replace with
- `dryRun` (optional): Preview changes without applying (default: false)

**Returns**: Git-style diff showing changes made

### `copy_file`

Copy a file to a new location. Uses streaming for memory-efficient handling of large files.

**Parameters**:

- `source` (required): Path to the source file
- `destination` (required): Path to the destination file

**Returns**: Success confirmation

### `move_file`

Move or rename a file or directory.

**Parameters**:

- `source` (required): Current path
- `destination` (required): New path

**Returns**: Success confirmation

### `delete_file`

Delete a file.

**Parameters**:

- `path` (required): Path to the file to delete

**Returns**: Success confirmation

### `delete_directory`

Delete a directory. Cannot delete allowed root directories.

**Parameters**:

- `path` (required): Path to the directory to delete
- `recursive` (optional): Delete contents recursively (default: false)

**Returns**: Success confirmation

### `create_directory`

Create a directory, including any necessary parent directories.

**Parameters**:

- `path` (required): Path to the directory to create

**Returns**: Success confirmation

### `list_directory`

List the contents of a directory with `[FILE]` and `[DIR]` prefixes.

**Parameters**:

- `path` (required): Path to the directory to list

**Returns**: Array of directory entries with type indicators

### `list_directory_with_sizes`

List directory contents with file sizes and optional sorting.

**Parameters**:

- `path` (required): Path to the directory to list
- `sortBy` (optional): Sort field - `name`, `size`, or `modified` (default: name)
- `order` (optional): Sort order - `asc` or `desc` (default: asc)

**Returns**: Array of entries with name, type, size, and modification time

### `directory_tree`

Get a recursive tree view of files and directories as JSON.

**Parameters**:

- `path` (required): Path to the root directory

**Returns**: JSON structure with `name`, `type`, and `children` for each entry

### `search_files`

Recursively search for files matching a glob pattern.

**Parameters**:

- `path` (required): Starting directory for the search
- `pattern` (required): Glob pattern to match (e.g., `*.go`, `**/*.json`)
- `excludePatterns` (optional): Array of patterns to exclude

**Returns**: Array of matching file paths

### `get_file_info`

Get detailed metadata about a file or directory.

**Parameters**:

- `path` (required): Path to the file or directory

**Returns**: Metadata including:

- `size`: Size in bytes
- `created`: Creation timestamp
- `modified`: Last modification timestamp
- `accessed`: Last access timestamp
- `isDirectory`: Whether path is a directory
- `isFile`: Whether path is a file
- `permissions`: Unix permission string

### `list_allowed_directories`

List all directories the server is allowed to access.

**Parameters**: None

**Returns**: Array of allowed directory paths

## Tool Annotations

This server sets [MCP Tool Annotations](https://modelcontextprotocol.io/specification/2025-03-26/server/tools#toolannotations) on each tool so clients can:

- Distinguish **read-only** tools from write-capable tools
- Understand which write operations are **idempotent** (safe to retry with the same arguments)
- Highlight operations that may be **destructive** (overwriting or deleting data)

| Tool                        | readOnlyHint | idempotentHint | destructiveHint | Notes                                       |
|-----------------------------|--------------|----------------|-----------------|---------------------------------------------|
| `read_text_file`            | `true`       | –              | –               | Pure read                                   |
| `read_file`                 | `true`       | –              | –               | Pure read (deprecated)                      |
| `read_multiple_files`       | `true`       | –              | –               | Pure read                                   |
| `read_media_file`           | `true`       | –              | –               | Pure read                                   |
| `list_directory`            | `true`       | –              | –               | Pure read                                   |
| `list_directory_with_sizes` | `true`       | –              | –               | Pure read                                   |
| `directory_tree`            | `true`       | –              | –               | Pure read                                   |
| `search_files`              | `true`       | –              | –               | Pure read                                   |
| `get_file_info`             | `true`       | –              | –               | Pure read                                   |
| `list_allowed_directories`  | `true`       | –              | –               | Pure read                                   |
| `create_directory`          | `false`      | `true`         | `false`         | Re-creating the same dir is a no-op         |
| `write_file`                | `false`      | `true`         | `true`          | Overwrites existing files                   |
| `edit_file`                 | `false`      | `false`        | `true`          | Re-applying edits can fail or double-apply  |
| `copy_file`                 | `false`      | `true`         | `true`          | Overwrites destination if exists            |
| `move_file`                 | `false`      | `false`        | `false`         | Move/rename only; repeat usually errors     |
| `delete_file`               | `false`      | `true`         | `true`          | Deleting already-deleted file is a no-op    |
| `delete_directory`          | `false`      | `true`         | `true`          | Deleting already-deleted dir is a no-op     |

> **Note**: `idempotentHint` and `destructiveHint` are meaningful only when `readOnlyHint` is `false`, as defined by the MCP spec.

## Configuration

### Claude Desktop

Add to `~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "filesystem": {
      "command": "/path/to/filesystem",
      "args": [
        "/Users/username/Documents",
        "/Users/username/Projects"
      ]
    }
  }
}
```

## Docker

```bash
# Build
docker build -t filesystem-mcp-server .

# Run with mounted directories
docker run -v /host/path:/data filesystem-mcp-server /data
```

## Security

- **Path validation**: All paths are validated against allowed directories
- **Symlink resolution**: Symlinks are resolved and validated
- **Null byte rejection**: Paths with null bytes are rejected
- **Parent traversal prevention**: `..` sequences cannot escape allowed directories
- **Atomic writes**: File writes use temp files to prevent corruption
- **Delete protection**: Cannot delete allowed root directories

## Development

```bash
# Run tests
make test

# Run linter
make lint

# Format code
make fmt

# Build binary
make build
```

## License

MIT
