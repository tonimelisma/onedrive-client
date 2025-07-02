package ui

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tonimelisma/onedrive-client/pkg/onedrive"
)

func TestDisplayDriveItems(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Create test items
	items := onedrive.DriveItemList{
		Value: []onedrive.DriveItem{
			{
				Name: "test-file.txt",
				Size: 1024,
				File: &onedrive.FileFacet{}, // Indicates it's a file
			},
			{
				Name:   "test-folder",
				Size:   0,
				Folder: &onedrive.FolderFacet{}, // Indicates it's a folder
			},
		},
	}

	// Call the function
	DisplayDriveItems(items)

	// Restore stdout and capture output
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify default title is used
	assert.Contains(t, output, "Items found:")
	assert.Contains(t, output, "test-file.txt")
	assert.Contains(t, output, "test-folder")
	assert.Contains(t, output, "File")
	assert.Contains(t, output, "Folder")
}

func TestDisplayDriveItemsWithTitle(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Create test items
	items := onedrive.DriveItemList{
		Value: []onedrive.DriveItem{
			{
				Name: "document.pdf",
				Size: 2048,
				File: &onedrive.FileFacet{},
			},
		},
	}

	// Call the function with custom title
	customTitle := "Files in /Documents subfolder:"
	DisplayDriveItemsWithTitle(items, customTitle)

	// Restore stdout and capture output
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify custom title is used
	assert.Contains(t, output, customTitle)
	assert.Contains(t, output, "document.pdf")
	assert.Contains(t, output, "File")
}

func TestDisplayDriveItemsEmpty(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Create empty items list
	items := onedrive.DriveItemList{
		Value: []onedrive.DriveItem{},
	}

	// Call the function
	DisplayDriveItems(items)

	// Restore stdout and capture output
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify empty message is shown
	assert.Contains(t, output, "No items found in this location.")
}

func TestDisplayDriveItemsWithTitleEmpty(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Create empty items list
	items := onedrive.DriveItemList{
		Value: []onedrive.DriveItem{},
	}

	// Call the function with custom title
	customTitle := "Files in custom folder:"
	DisplayDriveItemsWithTitle(items, customTitle)

	// Restore stdout and capture output
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Should show empty message regardless of custom title
	assert.Contains(t, output, "No items found in this location.")
	// Should not contain the custom title since list is empty
	assert.NotContains(t, output, customTitle)
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
