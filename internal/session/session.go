package session

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// State represents the state of a resumable upload session.
type State struct {
	UploadURL          string    `json:"uploadUrl"`
	ExpirationDateTime time.Time `json:"expirationDateTime"`
	LocalPath          string    `json:"localPath"`
	RemotePath         string    `json:"remotePath"`
}

// GetConfigDir returns the base configuration directory for the application.
// It is a variable to allow for overriding in tests.
var GetConfigDir = func() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("could not get user config directory: %w", err)
	}
	return filepath.Join(configDir, "onedrive-client"), nil
}

func getSessionDir() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "sessions"), nil
}

// GetSessionFilePath returns the full path for a session file.
// It is a variable to allow for overriding in tests.
var GetSessionFilePath = func(localPath, remotePath string) (string, error) {
	sessionDir, err := getSessionDir()
	if err != nil {
		return "", err
	}

	// Create a unique and deterministic name for the session file
	hash := sha256.New()
	hash.Write([]byte(localPath + ":" + remotePath))
	filename := hex.EncodeToString(hash.Sum(nil)) + ".json"

	return filepath.Join(sessionDir, filename), nil
}

// Save persists the upload session state to a file.
func Save(state *State) error {
	sessionDir, err := getSessionDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		return fmt.Errorf("could not create session directory: %w", err)
	}

	filePath, err := GetSessionFilePath(state.LocalPath, state.RemotePath)
	if err != nil {
		return err
	}

	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("could not marshal session state: %w", err)
	}

	return os.WriteFile(filePath, data, 0644)
}

// Load retrieves the upload session state from a file.
func Load(localPath, remotePath string) (*State, error) {
	filePath, err := GetSessionFilePath(localPath, remotePath)
	if err != nil {
		return nil, err
	}

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
		_ = Delete(localPath, remotePath) // Attempt to clean up expired session
		return nil, nil
	}

	return &state, nil
}

// Delete removes the session state file.
func Delete(localPath, remotePath string) error {
	filePath, err := GetSessionFilePath(localPath, remotePath)
	if err != nil {
		return err
	}

	err = os.Remove(filePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("could not delete session file: %w", err)
	}
	return nil
}
