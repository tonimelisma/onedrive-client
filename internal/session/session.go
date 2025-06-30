package session

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
)

// State represents the state of a resumable session.
type State struct {
	DownloadURL        string    `json:"downloadUrl,omitempty"`
	UploadURL          string    `json:"uploadUrl,omitempty"`
	ExpirationDateTime time.Time `json:"expirationDateTime"`
	LocalPath          string    `json:"localPath"`
	RemotePath         string    `json:"remotePath"`
	CompletedBytes     int64     `json:"completedBytes"`
}

// Manager handles session file operations with configurable directory
type Manager struct {
	configDir string
}

// NewManager creates a new session manager with default config directory
func NewManager() (*Manager, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("could not get user config directory: %w", err)
	}
	return &Manager{
		configDir: filepath.Join(configDir, "onedrive-client"),
	}, nil
}

// NewManagerWithConfigDir creates a session manager with custom config directory
func NewManagerWithConfigDir(configDir string) *Manager {
	return &Manager{configDir: configDir}
}

func (m *Manager) getSessionDir() string {
	return filepath.Join(m.configDir, "sessions")
}

// GetSessionFilePath returns the full path for a session file.
func (m *Manager) GetSessionFilePath(localPath, remotePath string) string {
	sessionDir := m.getSessionDir()

	// Create a unique and deterministic name for the session file
	hash := sha256.New()
	hash.Write([]byte(localPath + ":" + remotePath))
	filename := hex.EncodeToString(hash.Sum(nil)) + ".json"

	return filepath.Join(sessionDir, filename)
}

// Save persists the upload session state to a file.
func (m *Manager) Save(state *State) error {
	sessionDir := m.getSessionDir()
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		return fmt.Errorf("could not create session directory: %w", err)
	}

	filePath := m.GetSessionFilePath(state.LocalPath, state.RemotePath)

	lock := flock.New(filePath + ".lock")
	locked, err := lock.TryLock()
	if err != nil {
		return fmt.Errorf("could not acquire file lock: %w", err)
	}
	if !locked {
		return fmt.Errorf("could not acquire file lock, another instance may be running")
	}
	defer lock.Unlock()

	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("could not marshal session state: %w", err)
	}

	return os.WriteFile(filePath, data, 0600)
}

// Load retrieves the upload session state from a file.
func (m *Manager) Load(localPath, remotePath string) (*State, error) {
	filePath := m.GetSessionFilePath(localPath, remotePath)

	lock := flock.New(filePath + ".lock")
	locked, err := lock.TryLock()
	if err != nil {
		return nil, fmt.Errorf("could not acquire file lock: %w", err)
	}
	if !locked {
		return nil, fmt.Errorf("could not acquire file lock, another instance may be running")
	}
	defer lock.Unlock()

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Return nil, nil if the session file doesn't exist
		}
		return nil, fmt.Errorf("could not read session file: %w", err)
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("could not unmarshal session state: %w", err)
	}

	// Check if the session has expired
	if time.Now().After(state.ExpirationDateTime) {
		_ = m.Delete(localPath, remotePath) // Attempt to clean up expired session
		return nil, nil
	}

	return &state, nil
}

// Delete removes the session state file.
func (m *Manager) Delete(localPath, remotePath string) error {
	filePath := m.GetSessionFilePath(localPath, remotePath)

	lock := flock.New(filePath + ".lock")
	locked, err := lock.TryLock()
	if err != nil {
		return fmt.Errorf("could not acquire file lock: %w", err)
	}
	if !locked {
		return fmt.Errorf("could not acquire file lock, another instance may be running")
	}
	defer lock.Unlock()

	err = os.Remove(filePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("could not delete session file: %w", err)
	}
	return nil
}
