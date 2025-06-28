package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/tonimelisma/onedrive-client/internal/config"
	"github.com/tonimelisma/onedrive-client/pkg/onedrive"
)

const authSessionFile = "auth_session.json"

// AuthState represents the state of a pending device code authentication.
type AuthState struct {
	DeviceCode string    `json:"device_code"`
	Expires    time.Time `json:"expires"`
	Interval   int       `json:"interval"`
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

	// Check if the session has expired
	if time.Now().After(state.Expires) {
		_ = DeleteAuthState() // Attempt to clean up expired session
		return nil, nil
	}

	return &state, nil
}

// DeleteAuthState removes the auth session state file.
func DeleteAuthState() error {
	filePath, err := getAuthSessionFilePath()
	if err != nil {
		return err
	}

	err = os.Remove(filePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("could not delete auth session file: %w", err)
	}
	return nil
}

// Login handles the entire device code authentication flow.
func Login(cfg *config.Configuration) error {
	// Step 1: Initiate device code flow
	deviceCodeResp, err := onedrive.InitiateDeviceCodeFlow(config.ClientID, cfg.Debug)
	if err != nil {
		return fmt.Errorf("initiating device code flow: %w", err)
	}

	fmt.Println(deviceCodeResp.Message)

	// Step 2: Poll for the token
	interval := time.Duration(deviceCodeResp.Interval) * time.Second
	expires := time.Now().Add(time.Duration(deviceCodeResp.ExpiresIn) * time.Second)

	for {
		if time.Now().After(expires) {
			return fmt.Errorf("authentication timed out")
		}

		time.Sleep(interval)

		token, err := onedrive.VerifyDeviceCode(config.ClientID, deviceCodeResp.DeviceCode, cfg.Debug)
		if err != nil {
			if errors.Is(err, onedrive.ErrAuthorizationPending) {
				// This is expected, continue polling
				continue
			}
			return fmt.Errorf("verifying device code: %w", err)
		}

		// Step 3: Save the token
		cfg.Token = *token
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("saving token: %w", err)
		}
		return nil // Success
	}
}
