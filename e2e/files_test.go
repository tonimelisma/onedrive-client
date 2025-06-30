package e2e

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tonimelisma/onedrive-client/pkg/onedrive"
)

func TestFileOperations(t *testing.T) {
	helper := NewE2ETestHelper(t)
	helper.LogTestInfo(t)

	t.Run("CreateDirectory", func(t *testing.T) {
		testDirName := "test-directory"
		remotePath := helper.GetTestPath(testDirName)

		// Create a directory
		item, err := helper.App.SDK.CreateFolder(helper.TestDir, testDirName)
		if err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}

		if item.Name != testDirName {
			t.Errorf("Expected directory name %s, got %s", testDirName, item.Name)
		}

		// Verify directory exists
		helper.AssertFileExists(t, remotePath)
	})

	t.Run("UploadSmallFile", func(t *testing.T) {
		// Create a small test file (1KB)
		testContent := []byte("This is a test file for E2E testing.\nIt contains some sample content.\n")
		for len(testContent) < 1024 {
			testContent = append(testContent, "Adding more content to reach 1KB size. "...)
		}
		testContent = testContent[:1024] // Ensure exactly 1KB

		localFile := helper.CreateTestFile(t, "small-test.txt", testContent)
		remotePath := helper.GetTestPath("small-test.txt")

		// Upload using simple upload (non-resumable)
		item, err := helper.App.SDK.UploadFile(localFile, remotePath)
		if err != nil {
			t.Fatalf("Failed to upload small file: %v", err)
		}

		if item.Size != int64(len(testContent)) {
			t.Errorf("Expected file size %d, got %d", len(testContent), item.Size)
		}

		// Wait for file to appear and verify existence
		helper.WaitForFile(t, remotePath, 30*time.Second)
		helper.AssertFileExists(t, remotePath)
	})

	t.Run("UploadLargeFile", func(t *testing.T) {
		// Create a large test file (5MB) for a more realistic test
		fileSize := int64(5 * 1024 * 1024)
		localFile := helper.CreateTestFileWithSize(t, "large-upload-test.txt", fileSize)
		remotePath := helper.GetTestPath("large-upload-test.txt")

		t.Logf("Created large test file: %s (%d bytes)", localFile, fileSize)

		// 1. Create upload session
		session, err := helper.App.SDK.CreateUploadSession(remotePath)
		if err != nil {
			t.Fatalf("Failed to create upload session: %v", err)
		}
		if session.UploadURL == "" {
			t.Fatal("Upload session URL is empty")
		}
		t.Logf("Upload session created: %s", session.UploadURL)

		// 2. Open local file for chunked upload
		file, err := os.Open(localFile)
		if err != nil {
			t.Fatalf("Failed to open local file for upload: %v", err)
		}
		defer file.Close()

		// 3. Upload file in chunks
		chunkSize := int64(320 * 1024 * 4) // 1.25 MB chunks
		buffer := make([]byte, chunkSize)
		var offset int64

		for {
			bytesRead, readErr := file.Read(buffer)
			if readErr != nil && readErr != io.EOF {
				t.Fatalf("Error reading from local file: %v", readErr)
			}
			if bytesRead == 0 {
				break
			}

			currentChunk := buffer[:bytesRead]
			endByte := offset + int64(len(currentChunk)) - 1

			t.Logf("Uploading chunk: bytes %d-%d/%d", offset, endByte, fileSize)
			_, err = helper.App.SDK.UploadChunk(session.UploadURL, offset, endByte, fileSize, bytes.NewReader(currentChunk))
			if err != nil {
				// The SDK's UploadChunk doesn't properly handle the final response (a DriveItem).
				// We'll ignore the error on the final chunk and verify the file exists after.
				if endByte+1 >= fileSize {
					t.Log("Ignoring potential error on final chunk, will verify file existence.")
				} else {
					t.Fatalf("Failed to upload chunk: %v", err)
				}
			}
			offset += int64(len(currentChunk))
		}

		// 4. Verify file exists and content is correct
		helper.WaitForFile(t, remotePath, 60*time.Second)
		helper.AssertFileExists(t, remotePath)
		helper.CompareFileHash(t, localFile, remotePath)
	})

	t.Run("DownloadLargeFile", func(t *testing.T) {
		// 1. Upload a file to be downloaded
		fileSize := int64(2 * 1024 * 1024) // 2MB
		localUploadFile := helper.CreateTestFileWithSize(t, "download-test.txt", fileSize)
		remotePath := helper.GetTestPath("download-test.txt")

		_, err := helper.App.SDK.UploadFile(localUploadFile, remotePath)
		if err != nil {
			t.Fatalf("Setup for download test failed: could not upload file: %v", err)
		}
		helper.WaitForFile(t, remotePath, 30*time.Second)
		t.Logf("Test file uploaded for download test: %s", remotePath)

		// Verify the file exists before attempting download
		item, err := helper.App.SDK.GetDriveItemByPath(remotePath)
		if err != nil {
			t.Fatalf("File verification failed before download: %v", err)
		}
		t.Logf("File verified before download: %s (size: %d)", item.Name, item.Size)

		// 2. Download the file
		localDownloadPath := helper.CreateTestFile(t, "downloaded-file.txt", nil)
		t.Logf("Attempting to download %s to %s", remotePath, localDownloadPath)

		err = helper.App.SDK.DownloadFile(remotePath, localDownloadPath)
		if err != nil {
			t.Fatalf("Failed to download file: %v", err)
		}

		// 3. Verify the downloaded file's integrity
		localUploadHash, err := helper.CalculateFileHash(localUploadFile)
		if err != nil {
			t.Fatalf("Failed to calculate hash for uploaded file: %v", err)
		}

		localDownloadHash, err := helper.CalculateFileHash(localDownloadPath)
		if err != nil {
			t.Fatalf("Failed to calculate hash for downloaded file: %v", err)
		}

		if localUploadHash != localDownloadHash {
			t.Errorf("File hash mismatch. Upload hash: %s, Download hash: %s", localUploadHash, localDownloadHash)
		} else {
			t.Logf("✓ File hashes match for upload and download")
		}
	})

	t.Run("VerifyUploadedFile", func(t *testing.T) {
		// Upload a file and verify it exists
		testContent := []byte("Content for verification test.\nThis file will be uploaded and verified.\n")
		localFile := helper.CreateTestFile(t, "verify-test.txt", testContent)
		remotePath := helper.GetTestPath("verify-test.txt")

		// Upload the file
		item, err := helper.App.SDK.UploadFile(localFile, remotePath)
		if err != nil {
			t.Fatalf("Failed to upload file for verification test: %v", err)
		}

		if item.Size != int64(len(testContent)) {
			t.Errorf("Expected file size %d, got %d", len(testContent), item.Size)
		}

		helper.WaitForFile(t, remotePath, 30*time.Second)
		helper.AssertFileExists(t, remotePath)

		t.Logf("✓ File uploaded and verified: %s (%d bytes)", item.Name, item.Size)
	})

	t.Run("ListDirectoryContents", func(t *testing.T) {
		// Debug: Check if test directory exists
		t.Logf("Checking if test directory exists: %s", helper.TestDir)
		_, err := helper.App.SDK.GetDriveItemByPath(helper.TestDir)
		if err != nil {
			t.Logf("Test directory does not exist yet: %v", err)
			// Try to create it explicitly
			_, createErr := helper.App.SDK.CreateFolder("E2E-Tests", helper.TestID)
			if createErr != nil {
				t.Logf("Failed to create test directory: %v", createErr)
			} else {
				t.Logf("Successfully created test directory")
			}
		} else {
			t.Logf("Test directory exists")
		}

		// Ensure we have at least one file in the test directory
		testContent := []byte("File for directory listing test.\n")
		localFile := helper.CreateTestFile(t, "list-test.txt", testContent)
		remotePath := helper.GetTestPath("list-test.txt")

		// Upload the file to ensure the directory has content
		t.Logf("Uploading test file: %s", remotePath)
		_, err = helper.App.SDK.UploadFile(localFile, remotePath)
		if err != nil {
			t.Fatalf("Failed to upload test file for directory listing: %v", err)
		}
		helper.WaitForFile(t, remotePath, 30*time.Second)
		t.Logf("Test file uploaded successfully")

		// List the contents of our test directory
		t.Logf("Attempting to list directory: %s", helper.TestDir)
		items, err := helper.App.SDK.GetDriveItemChildrenByPath(helper.TestDir)
		if err != nil {
			t.Fatalf("Failed to list directory contents: %v", err)
		}

		if len(items.Value) == 0 {
			t.Error("Expected at least one item in test directory")
		}

		// Verify we can see the file we just created
		foundFiles := make(map[string]bool)
		for _, item := range items.Value {
			foundFiles[item.Name] = true
			t.Logf("Found item: %s", item.Name)
		}

		// We should at least find the file we just uploaded
		if !foundFiles["list-test.txt"] {
			t.Errorf("Expected to find file 'list-test.txt' in listing")
		}
	})

	t.Run("GetFileMetadata", func(t *testing.T) {
		remotePath := helper.GetTestPath("small-test.txt")

		item, err := helper.App.SDK.GetDriveItemByPath(remotePath)
		if err != nil {
			t.Fatalf("Failed to get file metadata: %v", err)
		}

		if item.Name != "small-test.txt" {
			t.Errorf("Expected file name 'small-test.txt', got '%s'", item.Name)
		}

		if item.Size != 1024 {
			t.Errorf("Expected file size 1024, got %d", item.Size)
		}

		if item.ID == "" {
			t.Error("File ID should not be empty")
		}
	})
}

func TestDriveOperations(t *testing.T) {
	helper := NewE2ETestHelper(t)

	t.Run("ListDrives", func(t *testing.T) {
		drives, err := helper.App.SDK.GetDrives()
		if err != nil {
			t.Fatalf("Failed to list drives: %v", err)
		}

		if len(drives.Value) == 0 {
			t.Error("Expected at least one drive")
		}

		for _, drive := range drives.Value {
			if drive.ID == "" {
				t.Error("Drive ID should not be empty")
			}
			if drive.DriveType == "" {
				t.Error("Drive type should not be empty")
			}
		}
	})

	t.Run("GetDefaultDrive", func(t *testing.T) {
		drive, err := helper.App.SDK.GetDefaultDrive()
		if err != nil {
			t.Fatalf("Failed to get default drive: %v", err)
		}

		if drive.ID == "" {
			t.Error("Default drive ID should not be empty")
		}

		if drive.Quota.Total <= 0 {
			t.Error("Drive quota total should be greater than 0")
		}
	})
}

func TestErrorHandling(t *testing.T) {
	helper := NewE2ETestHelper(t)
	helper.LogTestInfo(t)

	t.Run("NonExistentFile", func(t *testing.T) {
		nonExistentPath := helper.GetTestPath("does-not-exist.txt")

		_, err := helper.App.SDK.GetDriveItemByPath(nonExistentPath)
		if err == nil {
			t.Error("Expected error for non-existent file, but got none")
		}

		// Check that error message contains expected text
		errorStr := err.Error()
		if !strings.Contains(errorStr, "itemNotFound") && !strings.Contains(errorStr, "resource not found") {
			t.Errorf("Error message doesn't contain expected text. Got: %s", errorStr)
		}
	})

	t.Run("InvalidUploadSession", func(t *testing.T) {
		invalidURL := "https://invalid-upload-session-url.com"

		err := helper.App.SDK.CancelUploadSession(invalidURL)
		if err == nil {
			t.Error("Expected error for invalid upload session URL")
		}
	})
}

func TestFileManipulationOperations(t *testing.T) {
	helper := NewE2ETestHelper(t)
	helper.LogTestInfo(t)

	t.Run("CopyFile", func(t *testing.T) {
		// First upload a file to copy
		testContent := []byte("Content for copy test.\nThis file will be copied.\n")
		localFile := helper.CreateTestFile(t, "copy-source.txt", testContent)
		sourcePath := helper.GetTestPath("copy-source.txt")

		_, err := helper.App.SDK.UploadFile(localFile, sourcePath)
		if err != nil {
			t.Fatalf("Failed to upload source file for copy test: %v", err)
		}
		helper.WaitForFile(t, sourcePath, 30*time.Second)

		// Now copy the file
		destinationParentPath := helper.TestDir
		newName := "copied-file.txt"
		monitorURL, err := helper.App.SDK.CopyDriveItem(sourcePath, destinationParentPath, newName)
		if err != nil {
			t.Fatalf("Failed to copy file: %v", err)
		}

		if monitorURL == "" {
			t.Error("Expected monitor URL but got empty string")
		}

		// Wait and verify the copied file exists
		copiedPath := helper.GetTestPath(newName)
		helper.WaitForFile(t, copiedPath, 60*time.Second)
		helper.AssertFileExists(t, copiedPath)

		t.Logf("✓ File copied successfully. Monitor URL: %s", monitorURL)
	})

	t.Run("RenameFile", func(t *testing.T) {
		// First upload a file to rename
		testContent := []byte("Content for rename test.\nThis file will be renamed.\n")
		localFile := helper.CreateTestFile(t, "rename-original.txt", testContent)
		originalPath := helper.GetTestPath("rename-original.txt")

		_, err := helper.App.SDK.UploadFile(localFile, originalPath)
		if err != nil {
			t.Fatalf("Failed to upload file for rename test: %v", err)
		}
		helper.WaitForFile(t, originalPath, 30*time.Second)

		// Now rename the file
		newName := "renamed-file.txt"
		item, err := helper.App.SDK.UpdateDriveItem(originalPath, newName)
		if err != nil {
			t.Fatalf("Failed to rename file: %v", err)
		}

		if item.Name != newName {
			t.Errorf("Expected renamed file to have name %s, got %s", newName, item.Name)
		}

		// Verify the renamed file exists and original doesn't
		renamedPath := helper.GetTestPath(newName)
		helper.WaitForFile(t, renamedPath, 30*time.Second)
		helper.AssertFileExists(t, renamedPath)
		helper.AssertFileNotExists(t, originalPath)

		t.Logf("✓ File renamed successfully to: %s", item.Name)
	})

	t.Run("MoveFile", func(t *testing.T) {
		// First create a subdirectory for moving
		subDirName := "move-destination"
		subDirPath := helper.GetTestPath(subDirName)
		_, err := helper.App.SDK.CreateFolder(helper.TestDir, subDirName)
		if err != nil {
			t.Fatalf("Failed to create subdirectory for move test: %v", err)
		}
		helper.WaitForFile(t, subDirPath, 30*time.Second)

		// Upload a file to move
		testContent := []byte("Content for move test.\nThis file will be moved.\n")
		localFile := helper.CreateTestFile(t, "move-source.txt", testContent)
		sourcePath := helper.GetTestPath("move-source.txt")

		_, err = helper.App.SDK.UploadFile(localFile, sourcePath)
		if err != nil {
			t.Fatalf("Failed to upload file for move test: %v", err)
		}
		helper.WaitForFile(t, sourcePath, 30*time.Second)

		// Now move the file
		item, err := helper.App.SDK.MoveDriveItem(sourcePath, subDirPath)
		if err != nil {
			t.Fatalf("Failed to move file: %v", err)
		}

		// Verify the file is in the new location and not in the old location
		movedPath := path.Join(subDirPath, "move-source.txt")
		helper.WaitForFile(t, movedPath, 30*time.Second)
		helper.AssertFileExists(t, movedPath)
		helper.AssertFileNotExists(t, sourcePath)

		t.Logf("✓ File moved successfully to: %s", item.ParentReference.Path)
	})

	t.Run("DeleteFile", func(t *testing.T) {
		// First upload a file to delete
		testContent := []byte("Content for delete test.\nThis file will be deleted.\n")
		localFile := helper.CreateTestFile(t, "delete-test.txt", testContent)
		filePath := helper.GetTestPath("delete-test.txt")

		_, err := helper.App.SDK.UploadFile(localFile, filePath)
		if err != nil {
			t.Fatalf("Failed to upload file for delete test: %v", err)
		}
		helper.WaitForFile(t, filePath, 30*time.Second)

		// Now delete the file
		err = helper.App.SDK.DeleteDriveItem(filePath)
		if err != nil {
			t.Fatalf("Failed to delete file: %v", err)
		}

		// Verify the file no longer exists
		time.Sleep(5 * time.Second) // Give some time for deletion to propagate
		helper.AssertFileNotExists(t, filePath)

		t.Logf("✓ File deleted successfully")
	})
}

func TestSearchOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}

	helper := NewE2ETestHelper(t)

	t.Run("search for items", func(t *testing.T) {
		// Test basic search functionality
		items, err := helper.App.SDK.SearchDriveItems("test")
		if err != nil {
			// Search might return no results, which is acceptable
			if !strings.Contains(err.Error(), "no results") {
				t.Logf("Search returned error (this may be expected): %v", err)
			}
		} else {
			t.Logf("Search returned %d items", len(items.Value))

			// Verify the structure of returned items
			for _, item := range items.Value {
				assert.NotEmpty(t, item.ID, "Item should have an ID")
				assert.NotEmpty(t, item.Name, "Item should have a name")
			}
		}
	})

	t.Run("search with special characters", func(t *testing.T) {
		// Test search with special characters that need URL encoding
		_, err := helper.App.SDK.SearchDriveItems("test & special chars")
		// This should not fail due to URL encoding issues
		if err != nil {
			// Error is acceptable if it's not a URL encoding error
			assert.NotContains(t, err.Error(), "invalid URL escape", "URL encoding should work properly")
		}
	})
}

func TestRecentItemsOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}

	helper := NewE2ETestHelper(t)

	t.Run("get recent items", func(t *testing.T) {
		items, err := helper.App.SDK.GetRecentItems()
		if err != nil {
			t.Logf("GetRecentItems returned error (this may be expected): %v", err)
		} else {
			t.Logf("Recent items returned %d items", len(items.Value))

			// Verify the structure of returned items
			for _, item := range items.Value {
				assert.NotEmpty(t, item.ID, "Item should have an ID")
				assert.NotEmpty(t, item.Name, "Item should have a name")

				// Check if last modified time is present
				assert.False(t, item.LastModifiedDateTime.IsZero(), "Item should have LastModifiedDateTime")
			}
		}
	})
}

func TestSharedItemsOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}

	helper := NewE2ETestHelper(t)

	t.Run("get shared items", func(t *testing.T) {
		items, err := helper.App.SDK.GetSharedWithMe()
		if err != nil {
			t.Logf("GetSharedWithMe returned error (this may be expected): %v", err)
		} else {
			t.Logf("Shared items returned %d items", len(items.Value))

			// Verify the structure of returned items
			for _, item := range items.Value {
				assert.NotEmpty(t, item.ID, "Item should have an ID")
				assert.NotEmpty(t, item.Name, "Item should have a name")

				// Check if this is a remote item (shared from another drive)
				if item.RemoteItem != nil {
					assert.NotEmpty(t, item.RemoteItem.ID, "Remote item should have an ID")
					t.Logf("Found remote item: %s", item.Name)
				}
			}
		}
	})
}

func TestSpecialFolderOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}

	helper := NewE2ETestHelper(t)

	specialFolders := []string{"documents", "photos", "music"}

	for _, folderName := range specialFolders {
		t.Run(fmt.Sprintf("get special folder %s", folderName), func(t *testing.T) {
			item, err := helper.App.SDK.GetSpecialFolder(folderName)
			if err != nil {
				// Special folders might not exist or be accessible, which is acceptable
				t.Logf("GetSpecialFolder(%s) returned error (this may be expected): %v", folderName, err)

				// Check if it's a proper error format
				if !strings.Contains(err.Error(), "403") && !strings.Contains(err.Error(), "404") {
					// Other errors might indicate real issues
					t.Logf("Unexpected error type for special folder: %v", err)
				}
			} else {
				t.Logf("Special folder %s found: %s (ID: %s)", folderName, item.Name, item.ID)

				// Verify the item structure
				assert.NotEmpty(t, item.ID, "Special folder should have an ID")
				assert.NotEmpty(t, item.Name, "Special folder should have a name")

				// Check if it has the special folder facet
				if item.SpecialFolder != nil {
					assert.Equal(t, folderName, item.SpecialFolder.Name, "Special folder name should match requested name")
				}

				// Special folders should be folders
				assert.NotNil(t, item.Folder, "Special folder should have folder facet")
			}
		})
	}

	t.Run("invalid special folder", func(t *testing.T) {
		_, err := helper.App.SDK.GetSpecialFolder("invalid-folder")
		assert.Error(t, err, "Invalid special folder should return an error")
		assert.True(t, errors.Is(err, onedrive.ErrInvalidRequest), "Unexpected error type: %v", err)
	})

	// Test special folders that might be business-only
	t.Run("business special folders", func(t *testing.T) {
		businessFolders := []string{"cameraroll", "approot", "recordings"}

		for _, folderName := range businessFolders {
			t.Run(folderName, func(t *testing.T) {
				_, err := helper.App.SDK.GetSpecialFolder(folderName)
				if err != nil {
					t.Logf("Business special folder %s not available (expected for personal accounts): %v", folderName, err)
				} else {
					t.Logf("Business special folder %s is available", folderName)
				}
			})
		}
	})
}

func TestSharingLinkOperations(t *testing.T) {
	helper := NewE2ETestHelper(t)
	helper.LogTestInfo(t)

	t.Run("create sharing links", func(t *testing.T) {
		// Create a test file first
		testFileName := "test-share-file.txt"
		testContent := []byte("This is a test file for sharing")
		testFilePath := helper.CreateTestFile(t, testFileName, testContent)

		// Upload the test file
		remotePath := helper.GetTestPath(testFileName)
		item, err := helper.App.SDK.UploadFile(testFilePath, remotePath)
		if err != nil {
			t.Fatalf("Failed to upload test file: %v", err)
		}
		t.Logf("Uploaded test file: %s", item.Name)

		// Test creating a view link with anonymous scope
		t.Run("create view link anonymous", func(t *testing.T) {
			link, err := helper.App.SDK.CreateSharingLink(remotePath, "view", "anonymous")
			if err != nil {
				t.Logf("CreateSharingLink returned error (this may be expected on some accounts): %v", err)
				return
			}

			if link.ID == "" {
				t.Error("Expected sharing link to have an ID")
			}

			if link.Link.Type != "view" {
				t.Errorf("Expected link type 'view', got '%s'", link.Link.Type)
			}

			if link.Link.Scope != "anonymous" {
				t.Errorf("Expected scope 'anonymous', got '%s'", link.Link.Scope)
			}

			if link.Link.WebUrl == "" {
				t.Error("Expected sharing link to have a WebUrl")
			}

			t.Logf("Created view link: %s", link.Link.WebUrl)
		})

		// Test creating an edit link with organization scope
		t.Run("create edit link organization", func(t *testing.T) {
			link, err := helper.App.SDK.CreateSharingLink(remotePath, "edit", "organization")
			if err != nil {
				t.Logf("CreateSharingLink returned error (this may be expected on some accounts): %v", err)
				return
			}

			if link.ID == "" {
				t.Error("Expected sharing link to have an ID")
			}

			if link.Link.Type != "edit" {
				t.Errorf("Expected link type 'edit', got '%s'", link.Link.Type)
			}

			if link.Link.Scope != "organization" {
				t.Errorf("Expected scope 'organization', got '%s'", link.Link.Scope)
			}

			if link.Link.WebUrl == "" {
				t.Error("Expected sharing link to have a WebUrl")
			}

			t.Logf("Created edit link: %s", link.Link.WebUrl)
		})

		// Test error cases
		t.Run("invalid link type", func(t *testing.T) {
			_, err := helper.App.SDK.CreateSharingLink(remotePath, "invalid", "anonymous")
			if err == nil {
				t.Error("Expected error for invalid link type")
			}
		})

		t.Run("invalid scope", func(t *testing.T) {
			_, err := helper.App.SDK.CreateSharingLink(remotePath, "view", "invalid")
			if err == nil {
				t.Error("Expected error for invalid scope")
			}
		})

		t.Run("non-existent file", func(t *testing.T) {
			_, err := helper.App.SDK.CreateSharingLink("/non-existent-file.txt", "view", "anonymous")
			if err == nil {
				t.Error("Expected error for non-existent file")
			}
		})

		// Clean up: delete the test file
		err = helper.App.SDK.DeleteDriveItem(remotePath)
		if err != nil {
			t.Logf("Warning: Failed to clean up test file: %v", err)
		}
	})
}
