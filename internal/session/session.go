// Package session (session.go) provides functionality for managing resumable
// operation states, such as for large file uploads or downloads. It defines
// the structure for session state, a Manager to handle session file operations
// (CRUD), and uses file locking to ensure data integrity when multiple CLI
// instances might access session files.
package session

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
)

// State represents the persisted state of a resumable operation (e.g., upload or download).
// It includes necessary URLs, paths, and progress information.
type State struct {
	DownloadURL        string    `json:"downloadUrl,omitempty"`        // URL for downloading (if a download session).
	UploadURL          string    `json:"uploadUrl,omitempty"`          // URL for uploading chunks (if an upload session).
	ExpirationDateTime time.Time `json:"expirationDateTime"`           // When the session URL (upload/download) expires.
	LocalPath          string    `json:"localPath"`                    // Path to the local file.
	RemotePath         string    `json:"remotePath"`                   // Path to the remote file on OneDrive.
	CompletedBytes     int64     `json:"completedBytes"`               // Number of bytes successfully transferred.
	// TotalSize can be added if needed for progress calculation, though often derived from local file info.
	// TotalSize          int64     `json:"totalSize,omitempty"`
}

// Manager handles CRUD operations for session state files.
// It allows configuring the base directory where session files are stored.
type Manager struct {
	configDir string // Base directory for configuration, sessions will be in a 'sessions' subdirectory.
}

// NewManager creates a new session Manager.
// It determines the session directory based on standard user config paths or
// the `ONEDRIVE_CONFIG_PATH` environment variable (to align with where the main
// config.json might be stored, especially for testing or custom setups).
func NewManager() (*Manager, error) {
	// If ONEDRIVE_CONFIG_PATH is set, use its directory as the base for the 'sessions' subdir.
	// This keeps session files co-located with a custom config file, which is good for isolation (e.g., tests).
	if customPath := os.Getenv("ONEDRIVE_CONFIG_PATH"); customPath != "" {
		return &Manager{
			configDir: filepath.Dir(customPath), // Use the directory of the custom config file path.
		}, nil
	}

	// Default to a 'onedrive-client' subdirectory within the user's OS-specific config directory.
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("getting user config directory for session manager: %w", err)
	}
	return &Manager{
		configDir: filepath.Join(userConfigDir, "onedrive-client"),
	}, nil
}

// NewManagerWithConfigDir creates a session Manager with a custom base configuration directory.
// This is primarily useful for testing, allowing tests to specify an isolated directory for session files.
func NewManagerWithConfigDir(configDir string) *Manager {
	return &Manager{configDir: configDir}
}

// getSessionDir returns the full path to the 'sessions' subdirectory where session files are stored.
func (m *Manager) getSessionDir() string {
	return filepath.Join(m.configDir, "sessions")
}

// GetSessionFilePath generates a unique and deterministic file path for a session state file.
// The filename is a SHA256 hash of the combined local and remote file paths, ensuring
// a unique file for each transfer operation and allowing easy retrieval if the operation
// needs to be resumed.
func (m *Manager) GetSessionFilePath(localPath, remotePath string) string {
	sessionDir := m.getSessionDir()

	// Create a unique filename from a hash of local and remote paths.
	// This ensures that the session file is specific to this particular file transfer pair
	// and is deterministic (same paths will always produce the same session filename).
	hash := sha256.New()
	hash.Write([]byte(localPath + ":" + remotePath)) // Simple concatenation for hashing.
	filename := hex.EncodeToString(hash.Sum(nil)) + ".json" // e.g., "abcdef123456....json"

	return filepath.Join(sessionDir, filename)
}

// Save persists the given session State to a file.
// It ensures the session directory exists and uses file locking for safe concurrent access.
func (m *Manager) Save(state *State) error {
	sessionDir := m.getSessionDir()
	if err := os.MkdirAll(sessionDir, 0755); err != nil { // 0755: rwxr-xr-x
		return fmt.Errorf("creating session directory '%s' for save: %w", sessionDir, err)
	}

	filePath := m.GetSessionFilePath(state.LocalPath, state.RemotePath)

	// Use a file lock to prevent race conditions during write.
	lock := flock.New(filePath + ".lock")
	locked, err := lock.TryLock()
	if err != nil {
		return fmt.Errorf("acquiring file lock for session '%s': %w", filePath, err)
	}
	if !locked {
		return fmt.Errorf("could not acquire file lock for session '%s', another instance may be active", filePath)
	}
	defer lock.Unlock()

	data, err := json.MarshalIndent(state, "", "  ") // Pretty-print JSON.
	if err != nil {
		return fmt.Errorf("marshalling session state for '%s': %w", filePath, err)
	}

	// Write with 0600 permissions (user read/write only).
	return os.WriteFile(filePath, data, 0600)
}

// Load retrieves a session State from its file, identified by local and remote paths.
// It returns (nil, nil) if the session file doesn't exist or if the session has expired.
// Expired sessions are automatically deleted. File locking is used.
func (m *Manager) Load(localPath, remotePath string) (*State, error) {
	filePath := m.GetSessionFilePath(localPath, remotePath)

	// Ensure session directory exists before attempting to lock/read.
	sessionDir := m.getSessionDir()
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		return nil, fmt.Errorf("creating session directory '%s' for load: %w", sessionDir, err)
	}

	lock := flock.New(filePath + ".lock")
	locked, err := lock.TryLock()
	if err != nil {
		return nil, fmt.Errorf("acquiring file lock for session '%s': %w", filePath, err)
	}
	if !locked {
		return nil, fmt.Errorf("could not acquire file lock for session '%s', another instance may be active", filePath)
	}
	defer lock.Unlock()

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// File not found means no existing session for these paths.
			return nil, nil
		}
		return nil, fmt.Errorf("reading session file '%s': %w", filePath, err)
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		// Corrupted session file.
		return nil, fmt.Errorf("unmarshalling session state from '%s': %w", filePath, err)
	}

	// Check if the loaded session has expired.
	// OneDrive upload session URLs typically expire after a set period.
	if !state.ExpirationDateTime.IsZero() && time.Now().After(state.ExpirationDateTime) {
		log.Printf("Session for local '%s', remote '%s' has expired (Expiry: %s). Deleting.", localPath, remotePath, state.ExpirationDateTime)
		// Attempt to clean up the expired session file. Errors during delete are logged but not fatal for Load.
		if delErr := m.deleteFileInternal(filePath); delErr != nil { // Use internal delete without re-locking
			log.Printf("Warning: Failed to delete expired session file '%s': %v", filePath, delErr)
		}
		return nil, nil // Treat expired session as if it doesn't exist.
	}

	return &state, nil
}

// Delete removes the session state file identified by local and remote paths.
// Uses file locking to coordinate.
func (m *Manager) Delete(localPath, remotePath string) error {
	filePath := m.GetSessionFilePath(localPath, remotePath)

	// Ensure session directory exists before attempting to lock/delete.
	sessionDir := m.getSessionDir()
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		return fmt.Errorf("creating session directory '%s' for delete: %w", sessionDir, err)
	}

	lock := flock.New(filePath + ".lock")
	locked, err := lock.TryLock()
	if err != nil {
		return fmt.Errorf("acquiring file lock for deleting session '%s': %w", filePath, err)
	}
	if !locked {
		return fmt.Errorf("could not acquire file lock for deleting session '%s', another instance may be active", filePath)
	}
	defer lock.Unlock()

	return m.deleteFileInternal(filePath)
}

// deleteFileInternal performs the actual os.Remove without acquiring a lock.
// Assumes the caller has already handled locking.
func (m *Manager) deleteFileInternal(filePath string) error {
	err := os.Remove(filePath)
	// If the file doesn't exist, it's not an error (idempotent delete).
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("deleting session file '%s': %w", filePath, err)
	}
	return nil
}
