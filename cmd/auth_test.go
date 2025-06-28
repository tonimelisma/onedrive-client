package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tonimelisma/onedrive-client/internal/config"
	"github.com/tonimelisma/onedrive-client/internal/session"
	"github.com/tonimelisma/onedrive-client/pkg/onedrive"
)

// setupAuthTest overrides the config path to use a temporary directory.
func setupAuthTest(t *testing.T) (string, func()) {
	t.Helper()
	tempDir := t.TempDir()
	tempConfigFile := filepath.Join(tempDir, "config.json")

	t.Setenv("ONEDRIVE_CONFIG_PATH", tempConfigFile)

	// Override session package's GetConfigDir to point to same temp dir
	oldSessionDir := session.GetConfigDir
	session.GetConfigDir = func() (string, error) { return tempDir, nil }

	return tempDir, func() { session.GetConfigDir = oldSessionDir }
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

			// Handle Graph API requests (like /v1.0/me)
			if r.Method == "GET" && r.URL.Path == "/v1.0/me" {
				json.NewEncoder(w).Encode(onedrive.User{DisplayName: "Test User", UserPrincipalName: "test@user.com"})
				return
			}

			// Parse the request body to determine the type of OAuth request
			if r.Method == "POST" {
				body, _ := io.ReadAll(r.Body)
				bodyStr := string(body)

				if strings.Contains(bodyStr, "client_id") && strings.Contains(bodyStr, "scope") {
					// This is a device code request
					resp := onedrive.DeviceCodeResponse{
						DeviceCode:      "device-code-complete",
						UserCode:        "USERCODE",
						VerificationURI: "http://verify.com",
						ExpiresIn:       900,
						Interval:        1,
						Message:         "Go to http://verify.com and enter code USERCODE",
					}
					json.NewEncoder(w).Encode(resp)
				} else if strings.Contains(bodyStr, "device_code") {
					// This is a token verification request
					if loginAttempts == 0 {
						loginAttempts++
						w.WriteHeader(http.StatusBadRequest)
						json.NewEncoder(w).Encode(map[string]string{"error": "authorization_pending"})
					} else {
						json.NewEncoder(w).Encode(onedrive.OAuthToken{AccessToken: "final-token"})
					}
				} else {
					w.WriteHeader(http.StatusNotFound)
				}
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()
		onedrive.SetCustomEndpoints(server.URL, server.URL, server.URL)
		onedrive.SetCustomGraphEndpoint(server.URL + "/v1.0/")

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

func TestAuthFileLocking(t *testing.T) {
	t.Run("should prevent concurrent login attempts", func(t *testing.T) {
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

		_, cleanup := setupAuthTest(t)
		defer cleanup()

		// Start first login
		captureOutput(t, func() {
			rootCmd.SetArgs([]string{"auth", "login"})
			rootCmd.Execute()
		})

		// Try second login - should detect existing session
		output := captureOutput(t, func() {
			rootCmd.SetArgs([]string{"auth", "login"})
			rootCmd.Execute()
		})

		assert.Contains(t, output, "A login is already pending")
	})
}

func TestAuthErrorHandling(t *testing.T) {
	t.Run("should handle device code request failure", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
		}))
		defer server.Close()
		onedrive.SetCustomEndpoints(server.URL, server.URL, server.URL)

		_, cleanup := setupAuthTest(t)
		defer cleanup()

		// Test the command directly without going through Execute()
		rootCmd.SetArgs([]string{"auth", "login"})
		err := rootCmd.Execute()

		// Error should be returned when there's a server error
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "login failed")
	})

	t.Run("should handle token verification failure", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			body, _ := io.ReadAll(r.Body)
			bodyStr := string(body)

			if strings.Contains(bodyStr, "client_id") && strings.Contains(bodyStr, "scope") {
				resp := onedrive.DeviceCodeResponse{
					DeviceCode:      "test-device-code",
					UserCode:        "TESTCODE",
					VerificationURI: "https://test.com/verify",
					ExpiresIn:       900,
					Interval:        1,
					Message:         "Go to https://test.com/verify and enter code TESTCODE",
				}
				json.NewEncoder(w).Encode(resp)
			} else if strings.Contains(bodyStr, "device_code") {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": "access_denied"})
			}
		}))
		defer server.Close()
		onedrive.SetCustomEndpoints(server.URL, server.URL, server.URL)

		_, cleanup := setupAuthTest(t)
		defer cleanup()

		// Start login
		captureOutput(t, func() {
			rootCmd.SetArgs([]string{"auth", "login"})
			rootCmd.Execute()
		})

		// Check status - should detect auth failure and return error
		var statusErr error
		captureOutput(t, func() {
			rootCmd.SetArgs([]string{"auth", "status"})
			statusErr = rootCmd.Execute()
		})

		// Should return an error when auth fails
		assert.Error(t, statusErr)
		assert.Contains(t, statusErr.Error(), "authentication failed")

		// Session file should be cleaned up after failure
		configDir, err := config.GetConfigDir()
		require.NoError(t, err)
		sessionFilePath := filepath.Join(configDir, "sessions", "auth_session.json")
		assert.NoFileExists(t, sessionFilePath)
	})
}
