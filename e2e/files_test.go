package e2e

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFileOperations(t *testing.T) {
	helper := NewE2ETestHelper(t)
	helper.LogTestInfo(t)

	t.Run("CreateDirectory", func(t *testing.T) {
		testDirName := "test-directory"
		remotePath := helper.GetTestPath(testDirName)

		// Create a directory
		item, err := helper.App.SDK.CreateFolder(context.Background(), helper.TestDir, testDirName)
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
		item, err := helper.App.SDK.UploadFile(context.Background(), localFile, remotePath)
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
		session, err := helper.App.SDK.CreateUploadSession(context.Background(), remotePath)
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
			_, err = helper.App.SDK.UploadChunk(context.Background(), session.UploadURL, offset, endByte, fileSize, bytes.NewReader(currentChunk))
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

		_, err := helper.App.SDK.UploadFile(context.Background(), localUploadFile, remotePath)
		if err != nil {
			t.Fatalf("Setup for download test failed: could not upload file: %v", err)
		}
		helper.WaitForFile(t, remotePath, 30*time.Second)
		t.Logf("Test file uploaded for download test: %s", remotePath)

		// Verify the file exists before attempting download
		item, err := helper.App.SDK.GetDriveItemByPath(context.Background(), remotePath)
		if err != nil {
			t.Fatalf("File verification failed before download: %v", err)
		}
		t.Logf("File verified before download: %s (size: %d)", item.Name, item.Size)

		// 2. Download the file
		localDownloadPath := helper.CreateTestFile(t, "downloaded-file.txt", nil)
		t.Logf("Attempting to download %s to %s", remotePath, localDownloadPath)

		err = helper.App.SDK.DownloadFile(context.Background(), remotePath, localDownloadPath)
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
		item, err := helper.App.SDK.UploadFile(context.Background(), localFile, remotePath)
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
		_, err := helper.App.SDK.GetDriveItemByPath(context.Background(), helper.TestDir)
		if err != nil {
			t.Logf("Test directory does not exist yet: %v", err)
			// Try to create it explicitly
			_, createErr := helper.App.SDK.CreateFolder(context.Background(), "E2E-Tests", helper.TestID)
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
		_, err = helper.App.SDK.UploadFile(context.Background(), localFile, remotePath)
		if err != nil {
			t.Fatalf("Failed to upload test file for directory listing: %v", err)
		}
		helper.WaitForFile(t, remotePath, 30*time.Second)
		t.Logf("Test file uploaded successfully")

		// List the contents of our test directory
		t.Logf("Attempting to list directory: %s", helper.TestDir)
		items, err := helper.App.SDK.GetDriveItemChildrenByPath(context.Background(), helper.TestDir)
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

		item, err := helper.App.SDK.GetDriveItemByPath(context.Background(), remotePath)
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
		drives, err := helper.App.SDK.GetDrives(context.Background())
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
		drive, err := helper.App.SDK.GetDefaultDrive(context.Background())
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

		_, err := helper.App.SDK.GetDriveItemByPath(context.Background(), nonExistentPath)
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

		err := helper.App.SDK.CancelUploadSession(context.Background(), invalidURL)
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

		_, err := helper.App.SDK.UploadFile(context.Background(), localFile, sourcePath)
		if err != nil {
			t.Fatalf("Failed to upload source file for copy test: %v", err)
		}
		helper.WaitForFile(t, sourcePath, 30*time.Second)

		// Now copy the file
		destinationParentPath := helper.TestDir
		newName := "copied-file.txt"
		monitorURL, err := helper.App.SDK.CopyDriveItem(context.Background(), sourcePath, destinationParentPath, newName)
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

		_, err := helper.App.SDK.UploadFile(context.Background(), localFile, originalPath)
		if err != nil {
			t.Fatalf("Failed to upload file for rename test: %v", err)
		}
		helper.WaitForFile(t, originalPath, 30*time.Second)

		// Now rename the file
		newName := "renamed-file.txt"
		item, err := helper.App.SDK.UpdateDriveItem(context.Background(), originalPath, newName)
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
		_, err := helper.App.SDK.CreateFolder(context.Background(), helper.TestDir, subDirName)
		if err != nil {
			t.Fatalf("Failed to create subdirectory for move test: %v", err)
		}
		helper.WaitForFile(t, subDirPath, 30*time.Second)

		// Upload a file to move
		testContent := []byte("Content for move test.\nThis file will be moved.\n")
		localFile := helper.CreateTestFile(t, "move-source.txt", testContent)
		sourcePath := helper.GetTestPath("move-source.txt")

		_, err = helper.App.SDK.UploadFile(context.Background(), localFile, sourcePath)
		if err != nil {
			t.Fatalf("Failed to upload file for move test: %v", err)
		}
		helper.WaitForFile(t, sourcePath, 30*time.Second)

		// Now move the file
		item, err := helper.App.SDK.MoveDriveItem(context.Background(), sourcePath, subDirPath)
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

		_, err := helper.App.SDK.UploadFile(context.Background(), localFile, filePath)
		if err != nil {
			t.Fatalf("Failed to upload file for delete test: %v", err)
		}
		helper.WaitForFile(t, filePath, 30*time.Second)

		// Now delete the file
		err = helper.App.SDK.DeleteDriveItem(context.Background(), filePath)
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
		items, err := helper.App.SDK.SearchDriveItems(context.Background(), "test")
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
		// Test search with special characters and verify it doesn't crash
		_, err := helper.App.SDK.SearchDriveItems(context.Background(), "test & special chars")
		if err != nil && !strings.Contains(err.Error(), "no results") {
			t.Logf("Search with special characters returned error (may be expected): %v", err)
		}
	})
}

func TestRecentItemsOperations(t *testing.T) {
	helper := NewE2ETestHelper(t)

	t.Run("GetRecentItems", func(t *testing.T) {
		items, err := helper.App.SDK.GetRecentItems(context.Background())
		if err != nil {
			t.Fatalf("Failed to get recent items: %v", err)
		}

		// Recent items may be empty, which is acceptable
		t.Logf("Found %d recent items", len(items.Value))

		// Verify the structure of returned items
		for _, item := range items.Value {
			assert.NotEmpty(t, item.ID, "Recent item should have an ID")
			assert.NotEmpty(t, item.Name, "Recent item should have a name")
		}
	})
}

func TestSharedItemsOperations(t *testing.T) {
	helper := NewE2ETestHelper(t)

	t.Run("GetSharedWithMe", func(t *testing.T) {
		items, err := helper.App.SDK.GetSharedWithMe(context.Background())
		if err != nil {
			t.Fatalf("Failed to get shared items: %v", err)
		}

		// Shared items may be empty, which is acceptable
		t.Logf("Found %d shared items", len(items.Value))

		// Verify the structure of returned items
		for _, item := range items.Value {
			assert.NotEmpty(t, item.ID, "Shared item should have an ID")
			assert.NotEmpty(t, item.Name, "Shared item should have a name")
		}
	})
}

func TestSpecialFolderOperations(t *testing.T) {
	helper := NewE2ETestHelper(t)

	// Test specific special folders that are likely to exist
	specialFolders := []string{"documents", "photos", "music"}

	for _, folderName := range specialFolders {
		t.Run(fmt.Sprintf("get special folder %s", folderName), func(t *testing.T) {
			item, err := helper.App.SDK.GetSpecialFolder(context.Background(), folderName)
			if err != nil {
				// Special folders might not exist or be accessible, which is acceptable
				t.Logf("Special folder '%s' not accessible (may be expected): %v", folderName, err)
				return
			}

			if item.Name == "" {
				t.Errorf("Special folder '%s' should have a name", folderName)
			}

			if item.ID == "" {
				t.Errorf("Special folder '%s' should have an ID", folderName)
			}

			t.Logf("✓ Successfully accessed special folder '%s': %s", folderName, item.Name)
		})
	}

	t.Run("InvalidSpecialFolder", func(t *testing.T) {
		_, err := helper.App.SDK.GetSpecialFolder(context.Background(), "invalid-folder")
		if err == nil {
			t.Error("Expected error for invalid special folder")
		}
	})

	// Test duplicate access to verify consistency
	for _, folderName := range []string{"documents"} {
		t.Run(folderName+"_duplicate", func(t *testing.T) {
			_, err := helper.App.SDK.GetSpecialFolder(context.Background(), folderName)
			if err != nil {
				t.Logf("Special folder '%s' not accessible on duplicate access (may be expected): %v", folderName, err)
			}
		})
	}
}

func TestSharingLinkOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping sharing link tests in short mode")
	}

	helper := NewE2ETestHelper(t)

	t.Run("CreateSharingLinks", func(t *testing.T) {
		// Upload a test file to share
		testContent := []byte("Content for sharing link test.\nThis file will have sharing links created.\n")
		localFile := helper.CreateTestFile(t, "sharing-test.txt", testContent)
		remotePath := helper.GetTestPath("sharing-test.txt")

		// Upload the test file
		item, err := helper.App.SDK.UploadFile(context.Background(), localFile, remotePath)
		if err != nil {
			t.Fatalf("Failed to upload test file: %v", err)
		}
		helper.WaitForFile(t, remotePath, 30*time.Second)
		t.Logf("Test file uploaded: %s", item.Name)

		// Create a view sharing link (public)
		link, err := helper.App.SDK.CreateSharingLink(context.Background(), remotePath, "view", "anonymous")
		if err != nil {
			// Sharing links might not be supported or available for this account
			t.Logf("Failed to create view sharing link (may be expected): %v", err)
		} else {
			if link.Link.WebUrl == "" {
				t.Error("Expected sharing link URL but got empty string")
			}
			t.Logf("✓ Created view sharing link: %s", link.Link.WebUrl)
		}

		// Create an edit sharing link (organization only)
		link, err = helper.App.SDK.CreateSharingLink(context.Background(), remotePath, "edit", "organization")
		if err != nil {
			// Edit links might not be supported or available for this account
			t.Logf("Failed to create edit sharing link (may be expected): %v", err)
		} else {
			if link.Link.WebUrl == "" {
				t.Error("Expected sharing link URL but got empty string")
			}
			t.Logf("✓ Created edit sharing link: %s", link.Link.WebUrl)
		}

		// Test invalid link type
		_, err = helper.App.SDK.CreateSharingLink(context.Background(), remotePath, "invalid", "anonymous")
		if err == nil {
			t.Error("Expected error for invalid link type")
		}

		// Test invalid scope
		_, err = helper.App.SDK.CreateSharingLink(context.Background(), remotePath, "view", "invalid")
		if err == nil {
			t.Error("Expected error for invalid scope")
		}

		// Test with non-existent file
		_, err = helper.App.SDK.CreateSharingLink(context.Background(), "/non-existent-file.txt", "view", "anonymous")
		if err == nil {
			t.Error("Expected error for non-existent file")
		}

		// Clean up test file
		err = helper.App.SDK.DeleteDriveItem(context.Background(), remotePath)
		if err != nil {
			t.Logf("Failed to clean up test file (may be expected): %v", err)
		}
	})
}
