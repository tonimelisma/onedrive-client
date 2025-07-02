// Package session (auth.go) manages the temporary state file used during the
// OAuth 2.0 Device Code Flow. This state includes the device code, user code,
// and verification URI provided by Microsoft's identity platform.
// The package uses file locking to prevent race conditions when multiple
// instances of the CLI might attempt to read or write this session file.
package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gofrs/flock" // flock is used for file locking.
)

// authSessionFile is the name of the file used to store pending auth state.
const authSessionFile = "auth_session.json"

// AuthState represents the state of a pending device code authentication flow.
// This information is temporarily stored on disk while the user authenticates
// in their browser.
type AuthState struct {
	DeviceCode      string `json:"device_code"`      // The code the application uses to poll for the token.
	VerificationURI string `json:"verification_uri"` // The URL the user should visit.
	UserCode        string `json:"user_code"`        // The code the user enters at the verification URI.
	Interval        int    `json:"interval"`         // Recommended polling interval in seconds.
}

// getAuthSessionFilePath constructs the full path to the authentication session file
// using the session manager's configured directory.
func (m *Manager) getAuthSessionFilePath() string {
	sessionDir := m.getSessionDir() // getSessionDir is defined in session.go
	return filepath.Join(sessionDir, authSessionFile)
}

// SaveAuthState persists the pending authentication state to a file.
// It ensures the session directory exists and uses a file lock to prevent
// concurrent writes, which could corrupt the session file or lead to race conditions.
func (m *Manager) SaveAuthState(state *AuthState) error {
	sessionDir := m.getSessionDir()
	// Ensure the directory for session files exists.
	if err := os.MkdirAll(sessionDir, 0755); err != nil { // 0755: rwxr-xr-x
		return fmt.Errorf("creating session directory '%s' for auth state: %w", sessionDir, err)
	}

	filePath := m.getAuthSessionFilePath()

	// Use a file lock to ensure atomic write operations and prevent race conditions
	// if multiple CLI instances try to modify the auth session concurrently.
	// The lock file is typically named `auth_session.json.lock`.
	fileLock := flock.New(filePath + ".lock")
	locked, err := fileLock.TryLock()
	if err != nil {
		return fmt.Errorf("acquiring file lock for auth session '%s': %w", filePath, err)
	}
	if !locked {
		// This indicates another process holds the lock, possibly another CLI instance
		// trying to perform an auth operation.
		return errors.New("could not acquire file lock for auth session, another instance may be running or performing authentication")
	}
	defer fileLock.Unlock() // Ensure the lock is released when the function exits.

	data, err := json.MarshalIndent(state, "", "  ") // Pretty-print for readability if opened manually.
	if err != nil {
		return fmt.Errorf("marshalling auth session state: %w", err)
	}

	// Write with 0600 permissions (user read/write only) for security.
	return os.WriteFile(filePath, data, 0600)
}

// LoadAuthState retrieves the pending authentication state from a file.
// It returns (nil, nil) if the session file does not exist (no pending auth).
// File locking is used to ensure safe concurrent reads, although primarily aimed
// at coordinating with writes.
func (m *Manager) LoadAuthState() (*AuthState, error) {
	filePath := m.getAuthSessionFilePath()

	// Ensure the session directory exists before trying to create lock files,
	// as flock might attempt to create the lock file in a non-existent directory.
	sessionDir := m.getSessionDir()
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		return nil, fmt.Errorf("creating session directory '%s' for loading auth state: %w", sessionDir, err)
	}

	fileLock := flock.New(filePath + ".lock")
	// Using TryLock for reads is a bit more about consistency with write locks
	// than strict necessity for read-only, but helps avoid issues if a write is
	// just about to happen or if the lock mechanism has side effects on file presence.
	locked, err := fileLock.TryLock()
	if err != nil {
		return nil, fmt.Errorf("acquiring file lock for reading auth session '%s': %w", filePath, err)
	}
	if !locked {
		return nil, errors.New("could not acquire file lock for reading auth session, another instance may be active")
	}
	defer fileLock.Unlock()

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// If the session file doesn't exist, it means there's no pending auth state.
			// This is a normal condition, not an error.
			return nil, nil
		}
		return nil, fmt.Errorf("reading auth session file '%s': %w", filePath, err)
	}

	var state AuthState
	if err := json.Unmarshal(data, &state); err != nil {
		// If the file is corrupted or not valid JSON.
		return nil, fmt.Errorf("unmarshalling auth session state from '%s': %w", filePath, err)
	}

	return &state, nil
}

// DeleteAuthState removes the authentication session state file.
// This is typically called after authentication is successfully completed or if
// a pending authentication needs to be cancelled (e.g., during logout).
// File locking is used to coordinate with other potential operations.
func (m *Manager) DeleteAuthState() error {
	filePath := m.getAuthSessionFilePath()

	sessionDir := m.getSessionDir()
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		// If directory creation fails, we might not be able to acquire lock or delete.
		return fmt.Errorf("creating session directory '%s' for deleting auth state: %w", sessionDir, err)
	}

	fileLock := flock.New(filePath + ".lock")
	locked, err := fileLock.TryLock()
	if err != nil {
		return fmt.Errorf("acquiring file lock for deleting auth session '%s': %w", filePath, err)
	}
	if !locked {
		return errors.New("could not acquire file lock for deleting auth session, another instance may be active")
	}
	defer fileLock.Unlock()

	// Attempt to remove the session file.
	err = os.Remove(filePath)
	// If the file doesn't exist, it's not an error (idempotent delete).
	// Otherwise, propagate any actual error during removal.
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("deleting auth session file '%s': %w", filePath, err)
	}
	return nil
}
