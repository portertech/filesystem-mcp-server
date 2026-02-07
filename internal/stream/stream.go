package stream

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	// DefaultChunkSize is the default buffer size for streaming operations.
	DefaultChunkSize = 32 * 1024 // 32KB

	// TailChunkSize is the chunk size for tail operations.
	TailChunkSize = 1024 // 1KB
)

// TailFile reads the last n lines from a file without loading the entire file.
func TailFile(path string, n int) (string, error) {
	if n <= 0 {
		return "", nil
	}

	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return "", err
	}

	fileSize := stat.Size()
	if fileSize == 0 {
		return "", nil
	}

	// Read from the end in chunks
	var lines []string
	chunk := make([]byte, TailChunkSize)
	var leftover []byte
	offset := fileSize

	for len(lines) <= n && offset > 0 {
		readSize := int64(TailChunkSize)
		if offset < readSize {
			readSize = offset
		}
		offset -= readSize

		_, err := f.Seek(offset, io.SeekStart)
		if err != nil {
			return "", err
		}

		bytesRead, err := f.Read(chunk[:readSize])
		if err != nil && err != io.EOF {
			return "", err
		}

		// Prepend leftover from previous iteration
		data := append(chunk[:bytesRead], leftover...)

		// Split into lines
		var newLines []string
		start := len(data)
		for i := len(data) - 1; i >= 0; i-- {
			if data[i] == '\n' {
				if i < start-1 {
					newLines = append([]string{string(data[i+1 : start])}, newLines...)
				}
				start = i
			}
		}

		// Save leftover for next iteration
		if start > 0 {
			leftover = make([]byte, start)
			copy(leftover, data[:start])
		} else {
			leftover = nil
		}

		lines = append(newLines, lines...)
	}

	// Handle remaining leftover (first line of file)
	if len(leftover) > 0 && len(lines) < n {
		lines = append([]string{string(leftover)}, lines...)
	}

	// Take only the last n lines
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}

	// Join lines with newlines using strings.Builder
	var result strings.Builder
	for i, line := range lines {
		if i > 0 {
			result.WriteByte('\n')
		}
		result.WriteString(line)
	}

	return result.String(), nil
}

// HeadFile reads the first n lines from a file.
func HeadFile(path string, n int) (string, error) {
	if n <= 0 {
		return "", nil
	}

	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var result strings.Builder
	first := true
	count := 0

	for scanner.Scan() && count < n {
		if !first {
			result.WriteByte('\n')
		}
		first = false
		result.WriteString(scanner.Text())
		count++
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return result.String(), nil
}

// CopyFileStreaming copies a file using streaming with a temporary file for atomicity.
func CopyFileStreaming(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source: %w", err)
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat source: %w", err)
	}

	if srcInfo.IsDir() {
		return errors.New("source is a directory")
	}

	// Create temp file in same directory for atomic rename
	dstDir := filepath.Dir(dst)
	tmpFile, err := createTempFile(dstDir, ".tmp-")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Clean up temp file on error
	success := false
	defer func() {
		if !success {
			os.Remove(tmpPath)
		}
	}()

	// Copy with buffered writes
	buf := make([]byte, DefaultChunkSize)
	_, err = io.CopyBuffer(tmpFile, srcFile, buf)
	if err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to copy: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Preserve permissions
	if err := os.Chmod(tmpPath, srcInfo.Mode()); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, dst); err != nil {
		return fmt.Errorf("failed to rename: %w", err)
	}

	success = true
	return nil
}

// StreamToBase64 encodes a file to base64 using streaming to handle large files.
func StreamToBase64(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// For files under 10MB, read all at once for simplicity
	stat, err := f.Stat()
	if err != nil {
		return "", err
	}

	if stat.Size() < 10*1024*1024 {
		data, err := io.ReadAll(f)
		if err != nil {
			return "", err
		}
		return base64.StdEncoding.EncodeToString(data), nil
	}

	// For larger files, stream the encoding
	var result bytes.Buffer
	buf := make([]byte, DefaultChunkSize)
	encoder := base64.NewEncoder(base64.StdEncoding, &result)

	for {
		n, err := f.Read(buf)
		if n > 0 {
			if _, err := encoder.Write(buf[:n]); err != nil {
				return "", err
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
	}

	if err := encoder.Close(); err != nil {
		return "", err
	}

	return result.String(), nil
}

// createTempFile creates a temporary file with a cryptographically random suffix.
func createTempFile(dir, prefix string) (*os.File, error) {
	randBytes := make([]byte, 8)
	if _, err := rand.Read(randBytes); err != nil {
		return nil, err
	}

	name := filepath.Join(dir, prefix+hex.EncodeToString(randBytes))
	return os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
}

// FormatSize formats a file size in human-readable format.
func FormatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

// ReadFileWithLineNumbers reads lines from a file in the given range and prefixes
// each with its 1-based line number. Use 0 for startLine to indicate "from beginning"
// and 0 for endLine to indicate "to end".
func ReadFileWithLineNumbers(path string, startLine, endLine int) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// Normalize start line (0 means from beginning, which is line 1)
	if startLine <= 0 {
		startLine = 1
	}

	// First pass: count total lines to determine width (only if reading to end)
	// For bounded ranges, use the endLine for width calculation
	var maxLineNum int
	if endLine > 0 {
		maxLineNum = endLine
	} else {
		// Count total lines for dynamic width
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			maxLineNum++
		}
		if err := scanner.Err(); err != nil {
			return "", err
		}
		// Reset file position for second pass
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return "", err
		}
	}

	// Calculate width needed for line numbers
	width := lineNumberWidth(maxLineNum)
	format := fmt.Sprintf("%%%dd | %%s", width)

	scanner := bufio.NewScanner(f)
	var result strings.Builder
	lineNum := 0
	first := true

	for scanner.Scan() {
		lineNum++

		// Skip lines before startLine
		if lineNum < startLine {
			continue
		}

		// Stop if we've passed endLine (unless endLine is 0, meaning read to end)
		if endLine > 0 && lineNum > endLine {
			break
		}

		if !first {
			result.WriteByte('\n')
		}
		first = false
		fmt.Fprintf(&result, format, lineNum, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return result.String(), nil
}

// TailFileWithLineNumbers reads the last n lines from a file with line numbers in a single pass.
func TailFileWithLineNumbers(path string, n int) (string, error) {
	if n <= 0 {
		return "", nil
	}

	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// Use a ring buffer to keep track of last n lines with their line numbers
	type lineEntry struct {
		num  int
		text string
	}
	buffer := make([]lineEntry, n)
	bufIdx := 0
	lineCount := 0

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lineCount++
		buffer[bufIdx] = lineEntry{num: lineCount, text: scanner.Text()}
		bufIdx = (bufIdx + 1) % n
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	if lineCount == 0 {
		return "", nil
	}

	// Calculate width needed for line numbers
	width := lineNumberWidth(lineCount)
	format := fmt.Sprintf("%%%dd | %%s", width)

	// Determine how many lines to output
	outputCount := n
	if lineCount < n {
		outputCount = lineCount
		bufIdx = 0 // Start from beginning if we have fewer lines than requested
	}

	var result strings.Builder
	for i := 0; i < outputCount; i++ {
		if i > 0 {
			result.WriteByte('\n')
		}
		entry := buffer[(bufIdx+i)%n]
		fmt.Fprintf(&result, format, entry.num, entry.text)
	}

	return result.String(), nil
}

// lineNumberWidth calculates the character width needed to display a line number.
func lineNumberWidth(maxLine int) int {
	if maxLine <= 0 {
		return 1
	}
	width := 0
	for maxLine > 0 {
		width++
		maxLine /= 10
	}
	return width
}
