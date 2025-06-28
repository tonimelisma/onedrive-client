//go:build e2e

package e2e

import (
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
		// Create a large test file (10MB)
		fileSize := int64(10 * 1024 * 1024) // 10MB
		localFile := helper.CreateTestFileWithSize(t, "large-test.txt", fileSize)
		remotePath := helper.GetTestPath("large-test.txt")

		t.Logf("Created test file: %s (%d bytes)", localFile, fileSize)

		// Upload using resumable upload
		session, err := helper.App.SDK.CreateUploadSession(remotePath)
		if err != nil {
			t.Fatalf("Failed to create upload session: %v", err)
		}

		if session.UploadURL == "" {
			t.Fatal("Upload session URL is empty")
		}

		// For this test, we'll just verify the session was created
		// A full upload test would require implementing the chunked upload logic
		t.Logf("Upload session created successfully: %s", session.UploadURL)

		// Check upload session status
		status, err := helper.App.SDK.GetUploadSessionStatus(session.UploadURL)
		if err != nil {
			t.Fatalf("Failed to get upload session status: %v", err)
		}

		if status.UploadURL != session.UploadURL {
			t.Errorf("Expected upload URL %s, got %s", session.UploadURL, status.UploadURL)
		}

		// Cancel the upload session
		err = helper.App.SDK.CancelUploadSession(session.UploadURL)
		if err != nil {
			t.Fatalf("Failed to cancel upload session: %v", err)
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
