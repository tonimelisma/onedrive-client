package onedrive

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSanitizePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
		errType  error
	}{
		{
			name:     "Valid simple path",
			input:    "/Documents/file.txt",
			expected: "/Documents/file.txt",
			wantErr:  false,
		},
		{
			name:     "Valid root path",
			input:    "/",
			expected: "/",
			wantErr:  false,
		},
		{
			name:     "Valid nested path",
			input:    "/Documents/Folder/Subfolder/file.txt",
			expected: "/Documents/Folder/Subfolder/file.txt",
			wantErr:  false,
		},
		{
			name:     "Path traversal with ../",
			input:    "/Documents/../../../etc/passwd",
			expected: "",
			wantErr:  true,
			errType:  ErrPathTraversal,
		},
		{
			name:     "Path traversal in middle",
			input:    "/Documents/folder/../../../file.txt",
			expected: "",
			wantErr:  true,
			errType:  ErrPathTraversal,
		},
		{
			name:     "Invalid characters",
			input:    "/Documents/file<>|.txt",
			expected: "",
			wantErr:  true,
			errType:  ErrInvalidPath,
		},
		{
			name:     "Too long path",
			input:    "/" + strings.Repeat("a", 500),
			expected: "",
			wantErr:  true,
			errType:  ErrInvalidPath,
		},
		{
			name:     "Empty path",
			input:    "",
			expected: "",
			wantErr:  true,
			errType:  ErrInvalidPath,
		},
		{
			name:     "Relative path without leading slash",
			input:    "Documents/file.txt",
			expected: "",
			wantErr:  true,
			errType:  ErrInvalidPath,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SanitizePath(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("SanitizePath() expected error, got nil")
					return
				}
				if tt.errType != nil && !errors.Is(err, tt.errType) {
					t.Errorf("SanitizePath() expected error type %v, got %v", tt.errType, err)
				}
			} else {
				if err != nil {
					t.Errorf("SanitizePath() unexpected error: %v", err)
					return
				}
				if result != tt.expected {
					t.Errorf("SanitizePath() = %v, want %v", result, tt.expected)
				}
			}
		})
	}
}

func TestSanitizeLocalPath(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		errType error
	}{
		{
			name:    "Valid relative path",
			input:   "Documents/file.txt",
			wantErr: false,
		},
		{
			name:    "Valid absolute path",
			input:   "/home/user/Documents/file.txt",
			wantErr: false,
		},
		{
			name:    "Path traversal attack",
			input:   "../../../etc/passwd",
			wantErr: true,
			errType: ErrPathTraversal,
		},
		{
			name:    "Mixed path separators",
			input:   "Documents\\file.txt",
			wantErr: false, // Should be normalized
		},
		{
			name:    "Empty path",
			input:   "",
			wantErr: true,
			errType: ErrInvalidPath,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SanitizeLocalPath(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("SanitizeLocalPath() expected error, got nil")
					return
				}
				if tt.errType != nil && !errors.Is(err, tt.errType) {
					t.Errorf("SanitizeLocalPath() expected error type %v, got %v", tt.errType, err)
				}
			} else {
				if err != nil {
					t.Errorf("SanitizeLocalPath() unexpected error: %v", err)
					return
				}
				if result == "" {
					t.Errorf("SanitizeLocalPath() returned empty result")
				}
			}
		})
	}
}

func TestValidateDownloadPath(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	existingFile := filepath.Join(tempDir, "existing.txt")
	if err := os.WriteFile(existingFile, []byte("test"), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name    string
		path    string
		wantErr bool
		errType error
	}{
		{
			name:    "Valid new file path",
			path:    filepath.Join(tempDir, "new_file.txt"),
			wantErr: false,
		},
		{
			name:    "Existing file (should fail)",
			path:    existingFile,
			wantErr: true,
			errType: ErrFileExists,
		},
		{
			name:    "Path traversal",
			path:    "../../../etc/passwd",
			wantErr: true,
			errType: ErrPathTraversal,
		},
		{
			name:    "Empty path",
			path:    "",
			wantErr: true,
			errType: ErrInvalidPath,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDownloadPath(tt.path, false, 0o755) // allowOverwrite = false, dirPermissions = 0755

			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateDownloadPath() expected error, got nil")
					return
				}
				if tt.errType != nil && !errors.Is(err, tt.errType) {
					t.Errorf("ValidateDownloadPath() expected error type %v, got %v", tt.errType, err)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateDownloadPath() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestSecureCreateFile(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name    string
		path    string
		wantErr bool
		errType error
	}{
		{
			name:    "Create new file",
			path:    filepath.Join(tempDir, "new_secure_file.txt"),
			wantErr: false,
		},
		{
			name:    "Invalid path",
			path:    "",
			wantErr: true,
			errType: ErrInvalidPath,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := SecureCreateFile(tt.path, false, 0o644, 0o755) // allowOverwrite = false, filePermissions = 0644, dirPermissions = 0755

			if tt.wantErr {
				if err == nil {
					t.Errorf("SecureCreateFile() expected error, got nil")
					if file != nil {
						file.Close()
					}
					return
				}
				if tt.errType != nil && !errors.Is(err, tt.errType) {
					t.Errorf("SecureCreateFile() expected error type %v, got %v", tt.errType, err)
				}
			} else {
				if err != nil {
					t.Errorf("SecureCreateFile() unexpected error: %v", err)
					return
				}
				if file == nil {
					t.Errorf("SecureCreateFile() returned nil file")
					return
				}

				// Test file permissions
				info, err := file.Stat()
				if err != nil {
					t.Errorf("Failed to stat created file: %v", err)
				} else {
					mode := info.Mode()
					expected := os.FileMode(0o644) // From test parameters
					if mode.Perm() != expected {
						t.Errorf("SecureCreateFile() file permissions = %v, want %v", mode.Perm(), expected)
					}
				}

				file.Close()
			}
		})
	}
}

func TestValidateFileName(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantErr  bool
		errType  error
	}{
		{
			name:     "Valid filename",
			filename: "document.txt",
			wantErr:  false,
		},
		{
			name:     "Valid filename with spaces",
			filename: "my document.txt",
			wantErr:  false,
		},
		{
			name:     "Invalid characters",
			filename: "file<>name.txt",
			wantErr:  true,
			errType:  ErrInvalidPath,
		},
		{
			name:     "Too long filename",
			filename: strings.Repeat("a", 300) + ".txt",
			wantErr:  true,
			errType:  ErrInvalidPath,
		},
		{
			name:     "Empty filename",
			filename: "",
			wantErr:  true,
			errType:  ErrInvalidPath,
		},
		{
			name:     "Dot filename",
			filename: ".",
			wantErr:  true,
			errType:  ErrInvalidPath,
		},
		{
			name:     "Double dot filename",
			filename: "..",
			wantErr:  true,
			errType:  ErrInvalidPath,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFileName(tt.filename)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateFileName() expected error, got nil")
					return
				}
				if tt.errType != nil && !errors.Is(err, tt.errType) {
					t.Errorf("ValidateFileName() expected error type %v, got %v", tt.errType, err)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateFileName() unexpected error: %v", err)
				}
			}
		})
	}
}
