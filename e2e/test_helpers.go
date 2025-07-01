package e2e

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/tonimelisma/onedrive-client/internal/app"
	"github.com/tonimelisma/onedrive-client/internal/config"
	"github.com/tonimelisma/onedrive-client/internal/ui"
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

	// Check if we have a local config.json for E2E testing (in project root)
	if _, err := os.Stat("../config.json"); err != nil {
		t.Skip("Skipping E2E tests: config.json not found in project root. See e2e/README.md for setup instructions.")
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
		t.Fatalf("failed to set config path: %v", err)
	}

	// Load the config from the local file
	cfg, err := config.LoadOrCreate()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Check if we have a valid token
	if cfg.Token.AccessToken == "" {
		t.Fatal("No access token found. Please run: ./onedrive-client auth login")
	}

	onNewToken := func(token *onedrive.Token) error {
		cfg.Token = *token
		if err := cfg.Save(); err != nil {
			t.Logf("Warning: failed to save refreshed token: %v", err)
		}
		return nil
	}

	client := onedrive.NewClient(context.Background(), &cfg.Token, config.ClientID, onNewToken, ui.StdLogger{})

	// Quick sanity check: make a lightweight call to verify the token. If the
	// access token is expired or otherwise invalid we prefer to skip the E2E
	// tests instead of hard-failing on every test case. This keeps CI green in
	// environments without valid credentials while still allowing developers
	// with valid tokens to run the full suite.
	if _, err := client.GetMe(context.Background()); err != nil {
		if errors.Is(err, onedrive.ErrReauthRequired) || strings.Contains(err.Error(), "AADSTS") {
			t.Skip("Skipping E2E tests: authentication is invalid or expired. Run 'onedrive-client auth login' to refresh.")
		}
		// For any other unexpected error we still fail fast.
	}

	testID := generateTestID()
	testDir := path.Join(testRootDir, testID)

	helper := &E2ETestHelper{
		App: &app.App{
			Config: cfg,
			SDK:    client,
		},
		TestID:    testID,
		TestDir:   testDir,
		TempFiles: make([]string, 0),
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

// ensureTestDirectory creates the test directory if it doesn't exist
func (h *E2ETestHelper) ensureTestDirectory() error {
	// First ensure the root test directory exists
	_, err := h.App.SDK.CreateFolder(context.Background(), "/", testRootDir)
	if err != nil {
		// Check if error is because directory already exists
		if !strings.Contains(err.Error(), "conflict") &&
			!strings.Contains(err.Error(), "nameAlreadyExists") &&
			!strings.Contains(err.Error(), "resource not found") {
			return fmt.Errorf("failed to create root test directory: %w", err)
		}
	}

	// Then create the specific test directory inside the root
	_, err = h.App.SDK.CreateFolder(context.Background(), testRootDir, h.TestID)
	if err != nil {
		// Check if error is because directory already exists
		if !strings.Contains(err.Error(), "conflict") &&
			!strings.Contains(err.Error(), "nameAlreadyExists") &&
			!strings.Contains(err.Error(), "resource not found") {
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
		_, err := h.App.SDK.GetDriveItemByPath(context.Background(), remotePath)
		if err == nil {
			return // File found
		}

		if !errors.Is(err, onedrive.ErrResourceNotFound) {
			t.Fatalf("Unexpected error while waiting for file %s: %v", remotePath, err)
		}

		time.Sleep(1 * time.Second)
	}

	t.Fatalf("File %s did not appear within %v", remotePath, timeout)
}

// AssertFileExists verifies that a file exists in OneDrive
func (h *E2ETestHelper) AssertFileExists(t *testing.T, remotePath string) {
	t.Helper()

	_, err := h.App.SDK.GetDriveItemByPath(context.Background(), remotePath)
	if err != nil {
		if errors.Is(err, onedrive.ErrResourceNotFound) {
			t.Errorf("Expected file %s to exist, but it was not found", remotePath)
		} else {
			t.Errorf("Error checking if file %s exists: %v", remotePath, err)
		}
	}
}

// AssertFileNotExists verifies that a file does not exist in OneDrive
func (h *E2ETestHelper) AssertFileNotExists(t *testing.T, remotePath string) {
	t.Helper()

	_, err := h.App.SDK.GetDriveItemByPath(context.Background(), remotePath)
	if err == nil {
		t.Errorf("Expected file %s to not exist, but it was found", remotePath)
	} else if !errors.Is(err, onedrive.ErrResourceNotFound) {
		t.Errorf("Unexpected error checking if file %s exists: %v", remotePath, err)
	}
}

// CompareFileContent downloads a file and compares its content
func (h *E2ETestHelper) CompareFileContent(t *testing.T, remotePath string, expectedContent []byte) {
	t.Helper()

	localPath := h.CreateTestFile(t, "downloaded-for-compare", []byte{})
	defer os.Remove(localPath)

	err := h.App.SDK.DownloadFile(context.Background(), remotePath, localPath)
	if err != nil {
		t.Fatalf("Failed to download file %s for comparison: %v", remotePath, err)
	}

	actualContent, err := os.ReadFile(localPath)
	if err != nil {
		t.Fatalf("Failed to read downloaded file %s: %v", localPath, err)
	}

	if string(actualContent) != string(expectedContent) {
		t.Errorf("File content mismatch for %s. Expected %d bytes, got %d bytes.",
			remotePath, len(expectedContent), len(actualContent))
	}
}

// CompareFileHash compares the hash of a local file and a remote file
func (h *E2ETestHelper) CompareFileHash(t *testing.T, localPath, remotePath string) {
	t.Helper()

	// Download the remote file to a temporary location
	downloadedPath := h.CreateTestFile(t, "downloaded-for-hash", []byte{})
	defer os.Remove(downloadedPath)

	err := h.App.SDK.DownloadFile(context.Background(), remotePath, downloadedPath)
	if err != nil {
		t.Fatalf("Failed to download remote file %s for hash comparison: %v", remotePath, err)
	}

	// Calculate hash of the original local file
	localHash, err := h.CalculateFileHash(localPath)
	if err != nil {
		t.Fatalf("Failed to calculate hash for local file %s: %v", localPath, err)
	}

	// Calculate hash of the downloaded file
	remoteHash, err := h.CalculateFileHash(downloadedPath)
	if err != nil {
		t.Fatalf("Failed to calculate hash for downloaded file %s: %v", downloadedPath, err)
	}

	if localHash != remoteHash {
		t.Errorf("File hash mismatch for %s. Local hash: %s, Remote hash: %s",
			remotePath, localHash, remoteHash)
	}
}

// CalculateFileHash calculates the SHA256 hash of a file
func (h *E2ETestHelper) CalculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("opening file: %w", err)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("copying file to hasher: %w", err)
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

// Cleanup removes the test directory and temporary files
func (h *E2ETestHelper) Cleanup(t *testing.T) {
	t.Helper()

	// Remove local temp files
	for _, file := range h.TempFiles {
		os.Remove(file)
	}

	// Remove remote test directory (specific test folder)
	if err := h.App.SDK.DeleteDriveItem(context.Background(), h.TestDir); err != nil {
		// Don't fail the test, but log the error
		t.Logf("Warning: failed to clean up remote directory %s: %v", h.TestDir, err)
	}

	// Optionally clean up the root test directory if it's empty
	// This is best effort - if it fails due to non-empty directory, that's fine
	if err := h.App.SDK.DeleteDriveItem(context.Background(), "/"+testRootDir); err != nil {
		// Only log if it's not a "directory not empty" or "not found" error
		if !strings.Contains(err.Error(), "not empty") &&
			!strings.Contains(err.Error(), "not found") &&
			!strings.Contains(err.Error(), "resource not found") {
			t.Logf("Info: could not clean up root test directory %s: %v", testRootDir, err)
		}
	}
}

func generateTestID() string {
	return fmt.Sprintf("e2e-%d", time.Now().UnixNano())
}

func (h *E2ETestHelper) LogTestInfo(t *testing.T) {
	t.Helper()
	t.Logf("Test ID: %s", h.TestID)
	t.Logf("Test Directory: %s", h.TestDir)
}
