package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tonimelisma/onedrive-client/internal/config"
	"github.com/tonimelisma/onedrive-client/pkg/onedrive"
)

// setupAuthTest overrides the config path to use a temporary directory.
func setupAuthTest(t *testing.T) (string, func()) {
	t.Helper()
	tempDir := t.TempDir()
	tempConfigFile := filepath.Join(tempDir, "config.json")

	t.Setenv("ONEDRIVE_CONFIG_PATH", tempConfigFile)

	// Teardown is handled automatically by t.Setenv and t.TempDir
	return tempDir, func() {}
}

func TestAuthLogin(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := onedrive.DeviceCodeResponse{
			UserCode:        "TESTCODE",
			DeviceCode:      "test-device-code",
			VerificationURI: "https://test.com/verify",
			ExpiresIn:       900,
			Interval:        1,
			Message:         "Go to https://test.com/verify and enter code TESTCODE",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()
	onedrive.SetCustomEndpoints(server.URL, server.URL, server.URL)

	t.Run("should start login and create session file", func(t *testing.T) {
		_, cleanup := setupAuthTest(t)
		defer cleanup()

		run := func() {
			rootCmd.SetArgs([]string{"auth", "login"})
			rootCmd.Execute()
		}
		output := captureOutput(t, run)

		assert.Contains(t, output, "Go to https://test.com/verify and enter code TESTCODE")

		// We need to construct the path in the same way the application does.
		configDir, err := config.GetConfigDir()
		require.NoError(t, err)
		sessionFilePath := filepath.Join(configDir, "sessions", "auth_session.json")
		assert.FileExists(t, sessionFilePath)
	})

	t.Run("should show message when already logged in", func(t *testing.T) {
		_, cleanup := setupAuthTest(t)
		defer cleanup()

		cfg, _ := config.LoadOrCreate()
		cfg.Token.AccessToken = "fake-token"
		require.NoError(t, cfg.Save())

		run := func() {
			rootCmd.SetArgs([]string{"auth", "login"})
			rootCmd.Execute()
		}
		output := captureOutput(t, run)

		assert.Contains(t, output, "You are already logged in")
	})
}

func TestAuthStatus(t *testing.T) {
	t.Run("should report logged out", func(t *testing.T) {
		_, cleanup := setupAuthTest(t)
		defer cleanup()

		run := func() {
			rootCmd.SetArgs([]string{"auth", "status"})
			rootCmd.Execute()
		}
		output := captureOutput(t, run)
		assert.Contains(t, output, "You are not logged in")
	})

	t.Run("should complete login and report success", func(t *testing.T) {
		loginAttempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if strings.Contains(r.URL.Path, "devicecode") {
				resp := onedrive.DeviceCodeResponse{DeviceCode: "device-code-complete", UserCode: "USERCODE", VerificationURI: "http://verify.com"}
				json.NewEncoder(w).Encode(resp)
			} else if strings.Contains(r.URL.Path, "token") {
				if loginAttempts == 0 {
					loginAttempts++
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(map[string]string{"error": "authorization_pending"})
				} else {
					json.NewEncoder(w).Encode(onedrive.OAuthToken{AccessToken: "final-token"})
				}
			} else if strings.Contains(r.URL.Path, "me") {
				json.NewEncoder(w).Encode(onedrive.User{DisplayName: "Test User", UserPrincipalName: "test@user.com"})
			}
		}))
		defer server.Close()
		onedrive.SetCustomEndpoints(server.URL, server.URL, server.URL)

		_, cleanup := setupAuthTest(t)
		defer cleanup()

		// 1. Start the login
		captureOutput(t, func() {
			rootCmd.SetArgs([]string{"auth", "login"})
			rootCmd.Execute()
		})

		// 2. Check status (should still be pending)
		outputPending := captureOutput(t, func() {
			rootCmd.SetArgs([]string{"auth", "status"})
			rootCmd.Execute()
		})
		assert.Contains(t, outputPending, "USERCODE")

		// 3. Check status again (should complete)
		outputComplete := captureOutput(t, func() {
			rootCmd.SetArgs([]string{"auth", "status"})
			rootCmd.Execute()
		})
		assert.Contains(t, outputComplete, "Login successful!")
		assert.Contains(t, outputComplete, "You are logged in as: Test User (test@user.com)")
		configDir, err := config.GetConfigDir()
		require.NoError(t, err)
		sessionFilePath := filepath.Join(configDir, "sessions", "auth_session.json")
		assert.NoFileExists(t, sessionFilePath)
	})
}

func TestAuthLogout(t *testing.T) {
	t.Run("should clear token and session file", func(t *testing.T) {
		_, cleanup := setupAuthTest(t)
		defer cleanup()

		cfg, _ := config.LoadOrCreate()
		cfg.Token.AccessToken = "fake-token-for-logout"
		require.NoError(t, cfg.Save())

		configDir, err := config.GetConfigDir()
		require.NoError(t, err)
		sessionFilePath := filepath.Join(configDir, "sessions", "auth_session.json")
		os.MkdirAll(filepath.Dir(sessionFilePath), 0755)
		require.NoError(t, os.WriteFile(sessionFilePath, []byte("{}"), 0644))

		run := func() {
			rootCmd.SetArgs([]string{"auth", "logout"})
			rootCmd.Execute()
		}
		output := captureOutput(t, run)

		assert.Contains(t, output, "You have been logged out")

		cfg, _ = config.LoadOrCreate()
		assert.Empty(t, cfg.Token.AccessToken)
		assert.NoFileExists(t, sessionFilePath)
	})
}
