// Package onedrive (security.go) provides security utilities for the OneDrive SDK.
// This includes path sanitization, download overwrite protection, and other
// security-related functionality to prevent common vulnerabilities.
package onedrive

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Security-related errors
var (
	ErrPathTraversal = errors.New("path traversal attack detected")
	ErrInvalidPath   = errors.New("invalid path")
	ErrFileExists    = errors.New("file already exists")
	ErrUnsafePath    = errors.New("unsafe path detected")
)

// SanitizePath cleans and validates a path to prevent path traversal attacks.
// It removes dangerous elements like "..", null bytes, and normalizes the path.
// Returns the sanitized path or an error if the path is unsafe.
func SanitizePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("%w: path cannot be empty", ErrInvalidPath)
	}

	// Remove null bytes which can be used for attacks
	if strings.Contains(path, "\x00") {
		return "", fmt.Errorf("%w: null bytes not allowed in path", ErrUnsafePath)
	}

	// Check for path traversal attempts before cleaning
	if strings.Contains(path, "..") {
		return "", fmt.Errorf("%w: path contains path traversal elements", ErrPathTraversal)
	}

	// Clean the path to remove redundant separators and resolve . and .. elements
	cleaned := filepath.Clean(path)

	// Ensure the path starts with / for consistency (OneDrive paths are absolute from drive root)
	if !strings.HasPrefix(cleaned, "/") && !strings.HasPrefix(path, "/") {
		return "", fmt.Errorf("%w: OneDrive paths must be absolute (start with /)", ErrInvalidPath)
	}
	if !strings.HasPrefix(cleaned, "/") {
		cleaned = "/" + cleaned
	}

	// Additional security checks
	if len(cleaned) > 400 { // OneDrive has path length limits
		return "", fmt.Errorf("%w: path too long (max 400 characters)", ErrInvalidPath)
	}

	// Check for invalid characters in OneDrive paths
	invalidChars := []string{"<", ">", ":", "\"", "|", "?", "*"}
	for _, char := range invalidChars {
		if strings.Contains(cleaned, char) {
			return "", fmt.Errorf("%w: path contains invalid character '%s'", ErrInvalidPath, char)
		}
	}

	return cleaned, nil
}

// SanitizeLocalPath cleans and validates a local file system path.
// This is used for local file operations to prevent directory traversal.
func SanitizeLocalPath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("%w: path cannot be empty", ErrInvalidPath)
	}

	// Remove null bytes
	if strings.Contains(path, "\x00") {
		return "", fmt.Errorf("%w: null bytes not allowed in path", ErrUnsafePath)
	}

	// Check for suspicious path traversal patterns
	if strings.Contains(path, "../") || strings.HasPrefix(path, "../") || strings.HasSuffix(path, "/..") || path == ".." {
		return "", fmt.Errorf("%w: path contains directory traversal elements", ErrPathTraversal)
	}

	// Clean the path
	cleaned := filepath.Clean(path)

	// Get absolute path to resolve any remaining relative components
	abs, err := filepath.Abs(cleaned)
	if err != nil {
		return "", fmt.Errorf("%w: unable to resolve absolute path: %v", ErrInvalidPath, err)
	}

	return abs, nil
}

// ValidateDownloadPath checks if a download destination is safe and handles overwrite protection.
// It returns an error if the file exists and overwrite is false, or if the path is unsafe.
func ValidateDownloadPath(localPath string, allowOverwrite bool) error {
	// Sanitize the local path
	sanitized, err := SanitizeLocalPath(localPath)
	if err != nil {
		return fmt.Errorf("path validation failed: %w", err)
	}

	// Check if file already exists
	if _, err := os.Stat(sanitized); err == nil {
		if !allowOverwrite {
			return fmt.Errorf("%w: %s", ErrFileExists, sanitized)
		}
	} else if !os.IsNotExist(err) {
		// Some other error occurred (permission denied, etc.)
		return fmt.Errorf("checking file existence: %w", err)
	}

	// Ensure the parent directory exists or can be created
	dir := filepath.Dir(sanitized)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating parent directory: %w", err)
	}

	return nil
}

// SecureCreateFile creates a file with secure permissions and overwrite protection.
// Returns the file handle or an error if creation fails or file exists.
func SecureCreateFile(localPath string, allowOverwrite bool) (*os.File, error) {
	// Validate the download path first
	if err := ValidateDownloadPath(localPath, allowOverwrite); err != nil {
		return nil, err
	}

	// Determine file creation flags
	flags := os.O_CREATE | os.O_WRONLY | os.O_TRUNC
	if !allowOverwrite {
		flags |= os.O_EXCL // Fail if file exists
	}

	// Create the file with secure permissions (rw-r--r--)
	file, err := os.OpenFile(localPath, flags, 0644)
	if err != nil {
		if os.IsExist(err) && !allowOverwrite {
			return nil, fmt.Errorf("%w: %s", ErrFileExists, localPath)
		}
		return nil, fmt.Errorf("creating file: %w", err)
	}

	return file, nil
}

// ValidateFileName checks if a filename is valid for OneDrive.
// Returns an error if the filename contains invalid characters or patterns.
func ValidateFileName(filename string) error {
	if filename == "" {
		return fmt.Errorf("%w: filename cannot be empty", ErrInvalidPath)
	}

	// Check length (OneDrive has a 255 character limit for filenames)
	if len(filename) > 255 {
		return fmt.Errorf("%w: filename too long (max 255 characters)", ErrInvalidPath)
	}

	// Check for invalid characters
	invalidChars := []string{"<", ">", ":", "\"", "/", "\\", "|", "?", "*"}
	for _, char := range invalidChars {
		if strings.Contains(filename, char) {
			return fmt.Errorf("%w: filename contains invalid character '%s'", ErrInvalidPath, char)
		}
	}

	// Check for reserved names (Windows-specific but good practice)
	reservedNames := []string{
		"CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9",
	}

	upperFilename := strings.ToUpper(filename)
	for _, reserved := range reservedNames {
		if upperFilename == reserved || strings.HasPrefix(upperFilename, reserved+".") {
			return fmt.Errorf("%w: filename '%s' is reserved", ErrInvalidPath, filename)
		}
	}

	// Check for names ending with period or space (problematic on Windows)
	if strings.HasSuffix(filename, ".") || strings.HasSuffix(filename, " ") {
		return fmt.Errorf("%w: filename cannot end with period or space", ErrInvalidPath)
	}

	return nil
}
