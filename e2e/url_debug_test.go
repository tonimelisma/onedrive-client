package e2e

import (
	"testing"

	"github.com/tonimelisma/onedrive-client/pkg/onedrive"
)

func TestURLConstruction(t *testing.T) {
	testPaths := []string{
		"",
		"/",
		"test-file.txt",
		"E2E-Tests/test-file.txt",
		"Documents/test-file.txt",
	}

	for _, path := range testPaths {
		url := onedrive.BuildPathURL(path)
		t.Logf("Path: '%s' -> URL: '%s'", path, url)
	}

	// Test upload URL construction
	uploadURL := onedrive.BuildPathURL("E2E-Tests/test-file.txt") + ":/content"
	t.Logf("Upload URL: %s", uploadURL)
}
