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

func getAuthSessionFilePath() (string, error) {
	sessionDir, err := getSessionDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(sessionDir, authSessionFile), nil
}

// SaveAuthState persists the pending authentication state to a file.
func SaveAuthState(state *AuthState) error {
	sessionDir, err := getSessionDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		return fmt.Errorf("could not create session directory: %w", err)
	}

	filePath, err := getAuthSessionFilePath()
	if err != nil {
		return err
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

	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("could not marshal auth session state: %w", err)
	}

	return os.WriteFile(filePath, data, 0644)
}

// LoadAuthState retrieves the pending authentication state from a file.
func LoadAuthState() (*AuthState, error) {
	filePath, err := getAuthSessionFilePath()
	if err != nil {
		return nil, err
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
	filePath, err := getAuthSessionFilePath()
	if err != nil {
		return err
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
