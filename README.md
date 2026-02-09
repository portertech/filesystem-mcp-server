# Filesystem MCP Server

A secure, high-performance [Model Context Protocol](https://modelcontextprotocol.io/) (MCP) server that gives AI coding agents like Crush and OpenCode safe access to your local filesystem. Built in Go with directory sandboxing, symlink attack prevention, and atomic writes—so you can confidently let AI agents read, write, and manage files within boundaries you control.

## Features

- **17 filesystem tools** for comprehensive file operations
- **Secure by default**: Only operates within explicitly allowed directories
- **Symlink attack prevention**: Resolves symlinks and validates targets
- **Streaming support**: Memory-efficient handling of large files
- **Atomic writes**: Uses temp files with rename for safe file operations
- **Cross-platform**: Works on Linux, macOS, and Windows
- **AI agent optimized**: Line range support and dynamic formatting for efficient code navigation

## Installation

Pre-built binaries are available on the [GitHub Releases](https://github.com/portertech/filesystem-mcp-server/releases) page.

### Docker (Recommended)

The easiest way to run the server is with Docker:

```bash
docker pull portertech/filesystem-mcp-server
```

See the [Docker Usage](#docker-usage) section for configuration examples.

### Go Install

```bash
go install github.com/portertech/filesystem-mcp-server/cmd/filesystem@latest
```

### Build from Source

```bash
git clone https://github.com/portertech/filesystem-mcp-server.git
cd filesystem-mcp-server
make build
```

## Usage

```bash
# Start server with allowed directories
filesystem /path/to/allowed/dir1 /path/to/allowed/dir2

# Enable verbose logging
filesystem -verbose /path/to/dir

# Show version
filesystem -version

# List allowed directories
filesystem -list /path/to/dir
```

## Available Tools

### `read_text_file`

Read the contents of a text file with optional partial read and line numbers.

**Parameters**:

- `path` (required): Path to the file to read
- `head` (optional): Number of lines to read from the start
- `tail` (optional): Number of lines to read from the end
- `start_line` (optional): Starting line number (1-based, inclusive)
- `end_line` (optional): Ending line number (1-based, inclusive)
- `line_numbers` (optional): Include line numbers in output (default: false)

**Notes**:
- `start_line`/`end_line` cannot be combined with `head`/`tail`
- Using `start_line`/`end_line` always includes line numbers (optimized for AI agent use)
- Line number width dynamically adjusts based on total lines

**Examples**:

```json
// Read specific line range (lines 50-100)
{"path": "/path/to/file.go", "start_line": 50, "end_line": 100}

// Read from line 200 to end of file
{"path": "/path/to/file.go", "start_line": 200}

// Read first 20 lines with line numbers
{"path": "/path/to/file.go", "head": 20, "line_numbers": true}

// Read last 10 lines with line numbers
{"path": "/path/to/file.go", "tail": 10, "line_numbers": true}
```

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
- `format` (optional): Output format - `text` or `json` (default: text)

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

Apply find/replace edits to a text file with git-style diff output. Supports exact matching and whitespace-normalized line matching.

**Parameters**:

- `path` (required): Path to the file to edit
- `edits` (required): Array of edit operations, each with:
  - `oldText`: Text to search for (exact or whitespace-normalized match)
  - `newText`: Text to replace with
  - `requireUnique` (optional): Require exactly one match (default: true)
  - `occurrence` (optional): Which occurrence to replace when multiple exist (1-indexed)
- `dryRun` (optional): Preview changes without applying (default: false)

**Returns**: Git-style diff showing changes made

### `copy_file`

Copy a file to a new location. Uses streaming for memory-efficient handling of large files.

**Parameters**:

- `source` (required): Path to the source file
- `destination` (required): Path to the destination file
- `overwrite` (optional): Overwrite existing destination file (default: false)

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
- `format` (optional): Output format - `text` or `json` (default: text)

**Returns**: Array of directory entries with type indicators

### `list_directory_with_sizes`

List directory contents with file sizes and optional sorting.

**Parameters**:

- `path` (required): Path to the directory to list
- `sortBy` (optional): Sort field - `name`, `size`, or `modified` (default: name)
- `order` (optional): Sort order - `asc` or `desc` (default: asc)
- `format` (optional): Output format - `text` or `json` (default: text)

**Returns**: Array of entries with name, type, size, and modification time

### `directory_tree`

Get a recursive tree view of files and directories as JSON.

**Parameters**:

- `path` (required): Path to the root directory
- `excludePatterns` (optional): Array of glob patterns to exclude

**Returns**: JSON structure with `name`, `type`, and `children` for each entry

### `search_files`

Recursively search for files matching a glob pattern.

**Parameters**:

- `path` (required): Starting directory for the search
- `pattern` (required): Glob pattern to match (e.g., `*.go`, `**/*.json`)
- `excludePatterns` (optional): Array of patterns to exclude
- `format` (optional): Output format - `text` or `json` (default: text)

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

This server sets [MCP Tool Annotations](https://modelcontextprotocol.io/specification/2025-03-26/server/tools#toolannotations) on each tool to help clients understand tool behavior:

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
| `create_directory`          | –            | `true`         | –               | Re-creating existing dir is a no-op         |
| `write_file`                | –            | `true`         | `true`          | Overwrites existing files                   |
| `edit_file`                 | –            | –              | `true`          | Re-applying edits can fail or double-apply  |
| `copy_file`                 | –            | –              | `true`          | May overwrite destination                   |
| `move_file`                 | –            | –              | `true`          | Source is removed                           |
| `delete_file`               | –            | –              | `true`          | Permanently removes file                    |
| `delete_directory`          | –            | –              | `true`          | Permanently removes directory               |

> **Note**: `–` indicates the hint is not set (treated as unknown/unspecified by clients).

## Configuration

### Claude Desktop

Add to `~/Library/Application Support/Claude/claude_desktop_config.json`:

#### Using Docker (Recommended)

```json
{
  "mcpServers": {
    "filesystem": {
      "command": "docker",
      "args": [
        "run", "-i", "--rm",
        "-v", "/Users/username/Documents:/Users/username/Documents",
        "-v", "/Users/username/Projects:/Users/username/Projects",
        "portertech/filesystem-mcp-server",
        "/Users/username/Documents", "/Users/username/Projects"
      ]
    }
  }
}
```

#### Using Binary

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

## Docker Usage

Running the server in Docker provides an additional layer of security through container isolation, limiting the agent's access to only the mounted directories. Mirror host paths in your volume mounts (e.g., `-v /path:/path` instead of `-v /path:/data`) so the AI agent sees the same paths you reference in conversation.

```bash
# Run with mounted directories
docker run -i \
  -v /path/to/dir:/path/to/dir \
  portertech/filesystem-mcp-server /path/to/dir

# Multiple directories
docker run -i \
  -v /home/user/projects:/home/user/projects \
  -v /home/user/documents:/home/user/documents \
  portertech/filesystem-mcp-server /home/user/projects /home/user/documents

# Build locally (optional)
docker build -t filesystem-mcp-server .
```

## Security

- **Path validation**: All paths are validated against allowed directories
- **Symlink resolution**: Symlinks are resolved and validated
- **Null byte rejection**: Paths with null bytes are rejected
- **Parent traversal prevention**: `..` sequences cannot escape allowed directories
- **Atomic writes**: File writes use temp files to prevent corruption
- **Delete protection**: Cannot delete allowed root directories

## Symlink Handling

Symlinks are handled consistently across all tools to balance usability with security. The server supports symlinks when they resolve to paths within allowed directories, while protecting against symlink-based attacks.

### Behavior by Operation Type

| Operation | Symlink Behavior | Rationale |
|-----------|------------------|----------|
| **Read** | Follow if target is within allowed directories | Safe - target path is validated before access |
| **Write/Edit** | Reject symlinks | Prevents TOCTOU (time-of-check-time-of-use) attacks |
| **Delete** | Reject symlinks | Prevents unintended deletion of symlink targets |
| **Create directory** | Reject symlinks in path | Prevents creating directories through symlinked paths |
| **Traversal** (search, tree) | Skip symlinks during recursion | Prevents infinite loops and directory escape |

### Tool-Specific Behavior

| Tool | Root Path | During Traversal |
|------|-----------|------------------|
| `read_text_file` | Follows symlinks | N/A |
| `read_file` | Follows symlinks | N/A |
| `read_multiple_files` | Follows symlinks | N/A |
| `read_media_file` | Follows symlinks | N/A |
| `write_file` | Rejects symlinks | N/A |
| `edit_file` | Rejects symlinks | N/A |
| `copy_file` | Source: follows, Destination: rejects | N/A |
| `move_file` | Source: follows, Destination: rejects | N/A |
| `delete_file` | Rejects symlinks | N/A |
| `delete_directory` | Rejects symlinks | Rejects if directory contains symlinks |
| `create_directory` | Rejects symlinks in path | N/A |
| `list_directory` | Follows symlinks | Shows symlinks as entries |
| `list_directory_with_sizes` | Follows symlinks | Shows symlinks as entries |
| `directory_tree` | Follows symlinks | Skips symlinked entries |
| `search_files` | Follows symlinks | Skips symlinked files/directories |
| `get_file_info` | Follows symlinks | N/A |

### Security Considerations

1. **TOCTOU Prevention**: Write operations reject symlinks to prevent race conditions where a symlink could be swapped between validation and the actual write.

2. **Atomic Writes**: File writes use a temp file + rename pattern with `O_EXCL` flag to prevent symlink attacks on the temp file.

3. **Traversal Safety**: During recursive operations, symlinks are skipped to prevent:
   - Infinite loops from circular symlinks
   - Escaping allowed directories via symlinks to parent paths
   - Accessing files outside allowed directories

4. **Allowed Directory Resolution**: Allowed directories themselves are resolved via `filepath.EvalSymlinks` at startup, so symlinked allowed directories work correctly.

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
