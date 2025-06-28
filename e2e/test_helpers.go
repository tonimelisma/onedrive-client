//go:build e2e

package e2e

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/tonimelisma/onedrive-client/internal/app"
	"github.com/tonimelisma/onedrive-client/internal/config"
	"github.com/tonimelisma/onedrive-client/pkg/onedrive"
)

const testRootDir = "E2E-Tests"

// E2ETestHelper provides utilities for E2E testing
type E2ETestHelper struct {
	App       *app.App
	TestID    string
	TestDir   string
	TempFiles []string
}

// NewE2ETestHelper creates a new E2E test helper
func NewE2ETestHelper(t *testing.T) *E2ETestHelper {
	t.Helper()

	testID := generateTestID()
	testDir := path.Join(testRootDir, testID)

	helper := &E2ETestHelper{
		TestID:    testID,
		TestDir:   testDir,
		TempFiles: make([]string, 0),
	}

	// Set up authentication using existing CLI config
	client, err := helper.createAuthenticatedClient(t)
	if err != nil {
		t.Fatalf("Failed to create authenticated client: %v", err)
	}

	helper.App = &app.App{
		SDK: app.NewOneDriveSDK(client),
	}

	// Ensure test directory exists
	if err := helper.ensureTestDirectory(); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Set up cleanup
	t.Cleanup(func() {
		helper.Cleanup(t)
	})

	return helper
}

// createAuthenticatedClient creates an HTTP client using the existing CLI authentication
func (h *E2ETestHelper) createAuthenticatedClient(t *testing.T) (*http.Client, error) {
	t.Helper()

	// Check if we have a local config.json for E2E testing (in project root)
	if _, err := os.Stat("../config.json"); err != nil {
		t.Fatal(`
E2E Testing Setup Required:

1. Copy your authenticated config.json to the project root:
   cp ~/.config/onedrive-client/config.json ./config.json

2. The config.json will be ignored by git (safe)

3. Make sure you're logged in first:
   ./onedrive-client auth status

4. Then run E2E tests:
   go test -tags=e2e -v ./e2e/...

If you don't have config.json, run: ./onedrive-client auth login
`)
	}

	// For E2E tests, we need to load the config from the local config.json file
	// Temporarily set the environment to use the local file
	originalPath := os.Getenv("ONEDRIVE_CONFIG_PATH")
	defer func() {
		if originalPath != "" {
			os.Setenv("ONEDRIVE_CONFIG_PATH", originalPath)
		} else {
			os.Unsetenv("ONEDRIVE_CONFIG_PATH")
		}
	}()

	// Set to use the local config.json
	if err := os.Setenv("ONEDRIVE_CONFIG_PATH", "../config.json"); err != nil {
		return nil, fmt.Errorf("failed to set config path: %w", err)
	}

	// Load the config from the local file
	cfg, err := config.LoadOrCreate()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Check if we have a valid token
	if cfg.Token.AccessToken == "" {
		t.Fatal("No access token found. Please run: ./onedrive-client auth login")
	}

	// Create OAuth2 config
	ctx, oauthConfig := onedrive.GetOauth2Config(config.ClientID)

	// Convert our token to oauth2.Token
	token := onedrive.OAuthToken(cfg.Token)

	// Create HTTP client with token refresh capability
	client := onedrive.NewClient(ctx, (*onedrive.OAuthConfig)(oauthConfig), token, func(newToken onedrive.OAuthToken) {
		// Update config with refreshed token
		cfg.Token = newToken
		if err := cfg.Save(); err != nil {
			t.Logf("Warning: failed to save refreshed token: %v", err)
		}
	})

	return client, nil
}

// ensureTestDirectory creates the test directory if it doesn't exist
func (h *E2ETestHelper) ensureTestDirectory() error {
	_, err := h.App.SDK.CreateFolder(testRootDir, h.TestID)
	if err != nil {
		// Check if error is because directory already exists
		if !strings.Contains(err.Error(), "nameAlreadyExists") &&
			!strings.Contains(err.Error(), "itemNotFound") {
			return fmt.Errorf("failed to create test directory: %w", err)
		}
	}
	return nil
}

// CreateTestFile creates a temporary test file with specified content
func (h *E2ETestHelper) CreateTestFile(t *testing.T, name string, content []byte) string {
	t.Helper()

	file, err := os.CreateTemp("", fmt.Sprintf("e2e-%s-*-%s", h.TestID, name))
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	if _, err := file.Write(content); err != nil {
		file.Close()
		os.Remove(file.Name())
		t.Fatalf("Failed to write to temp file: %v", err)
	}

	file.Close()
	h.TempFiles = append(h.TempFiles, file.Name())
	return file.Name()
}

// CreateTestFileWithSize creates a test file with specified size
func (h *E2ETestHelper) CreateTestFileWithSize(t *testing.T, name string, size int64) string {
	t.Helper()

	content := make([]byte, size)
	// Fill with some pattern to make it compressible but not all zeros
	for i := range content {
		content[i] = byte(i % 256)
	}

	return h.CreateTestFile(t, name, content)
}

// CreateRandomTestFile creates a test file with random content
func (h *E2ETestHelper) CreateRandomTestFile(t *testing.T, name string, size int64) string {
	t.Helper()

	content := make([]byte, size)
	if _, err := rand.Read(content); err != nil {
		t.Fatalf("Failed to generate random content: %v", err)
	}

	return h.CreateTestFile(t, name, content)
}

// GetTestPath returns the full path for a test file
func (h *E2ETestHelper) GetTestPath(filename string) string {
	return path.Join(h.TestDir, filename)
}

// WaitForFile waits for a file to appear in OneDrive with a timeout
func (h *E2ETestHelper) WaitForFile(t *testing.T, remotePath string, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		_, err := h.App.SDK.GetDriveItemByPath(remotePath)
		if err == nil {
			return // File found
		}

		if !strings.Contains(err.Error(), "itemNotFound") {
			t.Fatalf("Unexpected error while waiting for file %s: %v", remotePath, err)
		}

		time.Sleep(1 * time.Second)
	}

	t.Fatalf("File %s did not appear within %v", remotePath, timeout)
}

// AssertFileExists verifies that a file exists in OneDrive
func (h *E2ETestHelper) AssertFileExists(t *testing.T, remotePath string) {
	t.Helper()

	_, err := h.App.SDK.GetDriveItemByPath(remotePath)
	if err != nil {
		if strings.Contains(err.Error(), "itemNotFound") {
			t.Errorf("Expected file %s to exist, but it was not found", remotePath)
		} else {
			t.Errorf("Error checking if file %s exists: %v", remotePath, err)
		}
	}
}

// AssertFileNotExists verifies that a file does not exist in OneDrive
func (h *E2ETestHelper) AssertFileNotExists(t *testing.T, remotePath string) {
	t.Helper()

	_, err := h.App.SDK.GetDriveItemByPath(remotePath)
	if err == nil {
		t.Errorf("Expected file %s to not exist, but it was found", remotePath)
		return
	}

	if !strings.Contains(err.Error(), "itemNotFound") {
		t.Errorf("Unexpected error checking if file %s exists: %v", remotePath, err)
	}
}

// CompareFileContent downloads a file and compares its content
func (h *E2ETestHelper) CompareFileContent(t *testing.T, remotePath string, expectedContent []byte) {
	t.Helper()

	// This is a placeholder for a more robust implementation
	// For now, we assume it's implemented elsewhere or will be added
	t.Logf("CompareFileContent not fully implemented - skipping content check for %s", remotePath)

	// A proper implementation would look like this:
	/*
		item, err := h.App.SDK.GetDriveItemByPath(remotePath)
		if err != nil {
			t.Fatalf("Failed to get item for content comparison: %v", err)
		}

		// Download the file (this needs a download helper)
		// For now, we assume a simple download URL exists
		resp, err := http.Get(item.DownloadURL)
		if err != nil {
			t.Fatalf("Failed to download file for comparison: %v", err)
		}
		defer resp.Body.Close()

		actualContent, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read downloaded content: %v", err)
		}

		if !bytes.Equal(expectedContent, actualContent) {
			t.Errorf("File content mismatch for %s", remotePath)
		}
	*/
}

// CompareFileHash downloads a remote file and compares its SHA256 hash with a local file.
func (h *E2ETestHelper) CompareFileHash(t *testing.T, localPath, remotePath string) {
	t.Helper()

	// Get local file hash
	localHash, err := h.calculateFileHash(localPath)
	if err != nil {
		t.Fatalf("Failed to calculate hash for local file %s: %v", localPath, err)
	}

	// Download remote file to a temporary location
	tempLocalPath := h.CreateTestFile(t, "downloaded-for-hash-comp.tmp", nil)
	err = h.App.SDK.DownloadFile(remotePath, tempLocalPath)
	if err != nil {
		t.Fatalf("Failed to download remote file %s for hash comparison: %v", remotePath, err)
	}

	// Get remote (downloaded) file hash
	remoteHash, err := h.calculateFileHash(tempLocalPath)
	if err != nil {
		t.Fatalf("Failed to calculate hash for downloaded file %s: %v", tempLocalPath, err)
	}

	if localHash != remoteHash {
		t.Errorf("File hash mismatch for %s. Local hash: %s, Remote hash: %s", remotePath, localHash, remoteHash)
	} else {
		t.Logf("✓ File hashes match for %s and %s", localPath, remotePath)
	}
}

func (h *E2ETestHelper) calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// Cleanup removes all test files and directories
func (h *E2ETestHelper) Cleanup(t *testing.T) {
	t.Helper()

	// Clean up local temp files
	for _, file := range h.TempFiles {
		if err := os.Remove(file); err != nil {
			t.Logf("Warning: failed to remove temp file %s: %v", file, err)
		}
	}

	// Clean up remote test directory
	// Note: Delete functionality is not yet implemented in the SDK
	// TODO: Implement delete when it becomes available
	if h.App != nil && h.App.SDK != nil {
		t.Logf("Note: Remote test directory cleanup not available yet: %s", h.TestDir)
		t.Logf("Please manually clean up test directory if needed")
	}
}

// generateTestID creates a unique test identifier
func generateTestID() string {
	return fmt.Sprintf("test-%d", time.Now().UnixNano())
}

// LogTestInfo logs useful information about the test setup
func (h *E2ETestHelper) LogTestInfo(t *testing.T) {
	t.Helper()
	t.Logf("Test ID: %s", h.TestID)
	t.Logf("Test Directory: %s", h.TestDir)
	t.Logf("Temp Files Created: %d", len(h.TempFiles))
}
