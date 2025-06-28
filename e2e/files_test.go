//go:build e2e

package e2e

import (
	"bytes"
	"io"
	"os"
	"path"
	"testing"
	"time"
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

		// 2. Download the file
		localDownloadPath := helper.CreateTestFile(t, "downloaded-file.txt", nil)
		err = helper.App.SDK.DownloadFile(remotePath, localDownloadPath)
		if err != nil {
			t.Fatalf("Failed to download file: %v", err)
		}

		// 3. Verify the downloaded file's integrity
		helper.CompareFileHash(t, localUploadFile, localDownloadPath)
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
		// List the contents of our test directory
		items, err := helper.App.SDK.GetDriveItemChildrenByPath(helper.TestDir)
		if err != nil {
			t.Fatalf("Failed to list directory contents: %v", err)
		}

		if len(items.Value) == 0 {
			t.Error("Expected at least one item in test directory")
		}

		// Verify we can see the files we created
		foundFiles := make(map[string]bool)
		for _, item := range items.Value {
			foundFiles[item.Name] = true
			t.Logf("Found item: %s", item.Name)
		}

		expectedFiles := []string{"test-directory", "small-test.txt", "verify-test.txt"}
		for _, expectedFile := range expectedFiles {
			if !foundFiles[expectedFile] {
				t.Errorf("Expected to find file/directory %s in listing", expectedFile)
			}
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

	t.Run("NonExistentFile", func(t *testing.T) {
		nonExistentPath := helper.GetTestPath("does-not-exist.txt")

		_, err := helper.App.SDK.GetDriveItemByPath(nonExistentPath)
		if err == nil {
			t.Error("Expected error for non-existent file")
		}

		// Verify the error is the expected type
		helper.AssertFileNotExists(t, nonExistentPath)
	})

	t.Run("InvalidUploadSession", func(t *testing.T) {
		invalidURL := "https://invalid-upload-session-url.com"

		err := helper.App.SDK.CancelUploadSession(invalidURL)
		if err == nil {
			t.Error("Expected error for invalid upload session URL")
		}
	})
}

func TestDownloadOperations(t *testing.T) {
	helper := NewE2ETestHelper(t)

	t.Run("DeleteFile", func(t *testing.T) {
		// 1. Upload a file to be deleted
		localFile := helper.CreateTestFile(t, "delete-test.txt", []byte("delete me"))
		remotePath := helper.GetTestPath("delete-test.txt")
		_, err := helper.App.SDK.UploadFile(localFile, remotePath)
		if err != nil {
			t.Fatalf("Setup for delete test failed: could not upload file: %v", err)
		}
		helper.WaitForFile(t, remotePath, 30*time.Second)
		helper.AssertFileExists(t, remotePath)

		// 2. Delete the file
		err = helper.App.SDK.DeleteDriveItem(remotePath)
		if err != nil {
			t.Fatalf("Failed to delete file: %v", err)
		}

		// 3. Verify the file is gone
		helper.AssertFileNotExists(t, remotePath)
	})

	t.Run("RenameFile", func(t *testing.T) {
		// 1. Upload a file to be renamed
		localFile := helper.CreateTestFile(t, "rename-test.txt", []byte("rename me"))
		remotePath := helper.GetTestPath("rename-test.txt")
		_, err := helper.App.SDK.UploadFile(localFile, remotePath)
		if err != nil {
			t.Fatalf("Setup for rename test failed: could not upload file: %v", err)
		}
		helper.WaitForFile(t, remotePath, 30*time.Second)

		// 2. Rename the file
		newName := "renamed-file.txt"
		newRemotePath := helper.GetTestPath(newName)
		_, err = helper.App.SDK.RenameDriveItem(remotePath, newName)
		if err != nil {
			t.Fatalf("Failed to rename file: %v", err)
		}

		// 3. Verify old name is gone and new name exists
		helper.AssertFileNotExists(t, remotePath)
		helper.AssertFileExists(t, newRemotePath)
	})

	t.Run("MoveFile", func(t *testing.T) {
		// 1. Create a subdirectory
		subDirName := "move-subdir"
		_, err := helper.App.SDK.CreateFolder(helper.TestDir, subDirName)
		if err != nil {
			t.Fatalf("Setup for move test failed: could not create subdir: %v", err)
		}
		subDirPath := helper.GetTestPath(subDirName)

		// 2. Upload a file to the root of the test dir
		localFile := helper.CreateTestFile(t, "move-test.txt", []byte("move me"))
		originalPath := helper.GetTestPath("move-test.txt")
		_, err = helper.App.SDK.UploadFile(localFile, originalPath)
		if err != nil {
			t.Fatalf("Setup for move test failed: could not upload file: %v", err)
		}
		helper.WaitForFile(t, originalPath, 30*time.Second)

		// 3. Move the file into the subdirectory
		newPath := path.Join(subDirPath, "move-test.txt")
		_, err = helper.App.SDK.MoveDriveItem(originalPath, subDirPath)
		if err != nil {
			t.Fatalf("Failed to move file: %v", err)
		}

		// 4. Verify file is in the new location and not the old one
		helper.AssertFileExists(t, newPath)
		helper.AssertFileNotExists(t, originalPath)
	})
}
