package ui

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tonimelisma/onedrive-client/pkg/onedrive"
)

func TestDisplayItems(t *testing.T) {
	// Test normal case with items
	items := onedrive.DriveItemList{
		Value: []onedrive.DriveItem{
			{
				Name:   "Test File 1.txt",
				Size:   1024,
				Folder: nil, // This is a file
			},
			{
				Name:   "Test Folder",
				Size:   0,
				Folder: &onedrive.FolderFacet{ChildCount: 5}, // This is a folder
			},
		},
	}

	// Capture stdout for testing
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	DisplayItems(items)

	w.Close()
	os.Stdout = oldStdout

	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Check if output contains expected elements
	if !strings.Contains(output, "Test File 1.txt") {
		t.Error("Output should contain test file name")
	}
	if !strings.Contains(output, "Test Folder") {
		t.Error("Output should contain test folder name")
	}
	if !strings.Contains(output, "File") {
		t.Error("Output should contain 'File' type")
	}
	if !strings.Contains(output, "Folder") {
		t.Error("Output should contain 'Folder' type")
	}
}

func TestDisplayItemsEmpty(t *testing.T) {
	// Test with empty list
	items := onedrive.DriveItemList{
		Value: []onedrive.DriveItem{},
	}

	// Capture stdout for testing
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	DisplayItems(items)

	w.Close()
	os.Stdout = oldStdout

	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Check if output contains expected message for empty list
	if !strings.Contains(output, "No items found") {
		t.Error("Output should contain 'No items found' message")
	}
}

func TestDisplayDrives(t *testing.T) {
	// Test normal case with drives
	drives := onedrive.DriveList{
		Value: []onedrive.Drive{
			{
				ID:        "drive1",
				Name:      "OneDrive - Personal",
				DriveType: "personal",
				Owner: struct {
					User *onedrive.Identity `json:"user,omitempty"`
				}{
					User: &onedrive.Identity{
						DisplayName: "Test User",
						ID:          "user123",
					},
				},
				Quota: struct {
					Total     int64  `json:"total"`
					Used      int64  `json:"used"`
					Remaining int64  `json:"remaining"`
					State     string `json:"state"`
				}{
					Total:     1000000000,
					Used:      500000000,
					Remaining: 500000000,
					State:     "normal",
				},
			},
		},
	}

	// Capture stdout for testing
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	DisplayDrives(drives)

	w.Close()
	os.Stdout = oldStdout

	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Check if output contains expected elements
	if !strings.Contains(output, "OneDrive - Personal") {
		t.Error("Output should contain drive name")
	}
	if !strings.Contains(output, "personal") {
		t.Error("Output should contain drive type")
	}
	if !strings.Contains(output, "Test User") {
		t.Error("Output should contain owner name")
	}
}

func TestFormatBytes(t *testing.T) {
	// Test the formatBytes helper function
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KiB"},
		{1536, "1.5 KiB"},
		{1048576, "1.0 MiB"},
		{1073741824, "1.0 GiB"},
		{1099511627776, "1.0 TiB"},
	}

	for _, test := range tests {
		result := formatBytes(test.input)
		assert.Equal(t, test.expected, result, "formatBytes(%d) should return %s, got %s", test.input, test.expected, result)
	}
}
