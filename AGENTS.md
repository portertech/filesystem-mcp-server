# AGENTS

Instructions for AI agents working on this codebase.

## Project Overview

This is a Go implementation of a Model Context Protocol (MCP) server providing secure filesystem operations. The server restricts access to explicitly allowed directories and implements security measures against symlink attacks, path traversal, and other filesystem vulnerabilities.

## Project Structure

```
cmd/filesystem/     # Main entry point
internal/
  pathutil/         # Path validation and security utilities
  registry/         # Tool registry for MCP tools
  security/         # Security validation logic
  server/           # MCP server implementation
  stream/           # Streaming utilities for large files
  tools/            # Individual filesystem tool implementations
pkg/filesystem/     # Public filesystem package
```

## Useful Commands

| Command           | Purpose                                    |
| ----------------- | ------------------------------------------ |
| `make build`      | Build the binary                           |
| `make test`       | Run all tests                              |
| `make test-v`     | Run tests with verbose output              |
| `make test-cover` | Run tests with coverage report             |
| `make lint`       | Run all linting checks (fmt-check + vet)   |
| `make fmt`        | Format code with gofmt                     |
| `make vet`        | Run go vet                                 |
| `make clean`      | Clean build artifacts                      |
| `make ci`         | Run all CI checks (lint + test)            |

## Code Conventions

- Follow standard Go idioms and conventions
- All paths must be validated against allowed directories before use
- Use the `pathutil` package for path validation and security checks
- File writes should be atomic (temp file + rename pattern)
- Symlinks must be resolved and validated before operations
- Tools are registered in the `registry` package with proper MCP annotations

## Security Considerations

When modifying filesystem operations:

1. Always validate paths using `pathutil.ValidatePath()` or equivalent
2. Resolve symlinks before checking if path is within allowed directories
3. Reject paths containing null bytes
4. Prevent parent traversal (`..`) from escaping allowed directories
5. Never delete allowed root directories
6. Use atomic writes to prevent file corruption

## Testing

- Run `make test` after any code changes
- Run `make lint` before committing
- Add tests for new tools in corresponding `_test.go` files
