package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gofrs/flock"
)

const authSessionFile = "auth_session.json"

// AuthState represents the state of a pending device code authentication.
type AuthState struct {
	DeviceCode      string `json:"device_code"`
	VerificationURI string `json:"verification_uri"`
	UserCode        string `json:"user_code"`
	Interval        int    `json:"interval"`
}

func (m *Manager) getAuthSessionFilePath() string {
	sessionDir := m.getSessionDir()
	return filepath.Join(sessionDir, authSessionFile)
}

// SaveAuthState persists the pending authentication state to a file.
func SaveAuthState(state *AuthState) error {
	mgr, err := NewManager()
	if err != nil {
		return fmt.Errorf("creating session manager: %w", err)
	}
	return mgr.SaveAuthState(state)
}

// SaveAuthState persists the pending authentication state to a file using the manager.
func (m *Manager) SaveAuthState(state *AuthState) error {
	sessionDir := m.getSessionDir()
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		return fmt.Errorf("could not create session directory: %w", err)
	}

	filePath := m.getAuthSessionFilePath()

	fileLock := flock.New(filePath + ".lock")
	locked, err := fileLock.TryLock()
	if err != nil {
		return fmt.Errorf("could not acquire file lock: %w", err)
	}
	if !locked {
		return errors.New("could not acquire file lock, another instance may be running")
	}
	defer fileLock.Unlock()

	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("could not marshal auth session state: %w", err)
	}

	return os.WriteFile(filePath, data, 0600)
}

// LoadAuthState retrieves the pending authentication state from a file.
func LoadAuthState() (*AuthState, error) {
	mgr, err := NewManager()
	if err != nil {
		return nil, fmt.Errorf("creating session manager: %w", err)
	}
	return mgr.LoadAuthState()
}

// LoadAuthState retrieves the pending authentication state from a file using the manager.
func (m *Manager) LoadAuthState() (*AuthState, error) {
	filePath := m.getAuthSessionFilePath()

	// Ensure the session directory exists before trying to create lock files
	sessionDir := m.getSessionDir()
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		return nil, fmt.Errorf("could not create session directory: %w", err)
	}

	fileLock := flock.New(filePath + ".lock")
	locked, err := fileLock.TryLock()
	if err != nil {
		return nil, fmt.Errorf("could not acquire file lock: %w", err)
	}
	if !locked {
		return nil, errors.New("could not acquire file lock, another instance may be running")
	}
	defer fileLock.Unlock()

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Return nil, nil if the session file doesn't exist
		}
		return nil, fmt.Errorf("could not read auth session file: %w", err)
	}

	var state AuthState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("could not unmarshal auth session state: %w", err)
	}

	return &state, nil
}

// DeleteAuthState removes the auth session state file.
func DeleteAuthState() error {
	mgr, err := NewManager()
	if err != nil {
		return fmt.Errorf("creating session manager: %w", err)
	}
	return mgr.DeleteAuthState()
}

// DeleteAuthState removes the auth session state file using the manager.
func (m *Manager) DeleteAuthState() error {
	filePath := m.getAuthSessionFilePath()

	// Ensure the session directory exists before trying to create lock files
	sessionDir := m.getSessionDir()
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		return fmt.Errorf("could not create session directory: %w", err)
	}

	fileLock := flock.New(filePath + ".lock")
	locked, err := fileLock.TryLock()
	if err != nil {
		return fmt.Errorf("could not acquire file lock: %w", err)
	}
	if !locked {
		return errors.New("could not acquire file lock, another instance may be running")
	}
	defer fileLock.Unlock()

	err = os.Remove(filePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("could not delete auth session file: %w", err)
	}
	return nil
}
