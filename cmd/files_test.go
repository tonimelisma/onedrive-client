package cmd

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/tonimelisma/onedrive-client/internal/session"
	"github.com/tonimelisma/onedrive-client/pkg/onedrive"
)

func TestFilesListLogic(t *testing.T) {
	mockSDK := &MockSDK{
		GetDriveItemChildrenByPathFunc: func(path string) (onedrive.DriveItemList, error) {
			assert.Equal(t, "/test", path)
			return onedrive.DriveItemList{
				Value: []onedrive.DriveItem{
					{Name: "file1.txt", Size: 100},
					{Name: "subfolder", Folder: &struct {
						ChildCount int `json:"childCount"`
						View       struct {
							ViewType  string `json:"viewType"`
							SortBy    string `json:"sortBy"`
							SortOrder string `json:"sortOrder"`
						} `json:"view"`
					}{ChildCount: 1}},
				},
			}, nil
		},
	}
	a := newTestApp(mockSDK)

	output := captureOutput(t, func() {
		err := filesListLogic(a, &cobra.Command{}, []string{"/test"})
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "file1.txt")
	assert.Contains(t, output, "subfolder")
}

func TestFilesMkdirLogic(t *testing.T) {
	mockSDK := &MockSDK{
		CreateFolderFunc: func(parentPath, folderName string) (onedrive.DriveItem, error) {
			assert.Equal(t, "/test", parentPath)
			assert.Equal(t, "new-folder", folderName)
			return onedrive.DriveItem{Name: folderName}, nil
		},
	}
	a := newTestApp(mockSDK)

	output := captureOutput(t, func() {
		err := filesMkdirLogic(a, &cobra.Command{}, []string{"/test/new-folder"})
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "Folder 'new-folder' created successfully")
}

func TestFilesUploadLogic(t *testing.T) {
	// Create a dummy file for upload
	tmpFile, err := ioutil.TempFile("", "test-upload-*.txt")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	// Write more than one chunk
	content := strings.Repeat("a", int(chunkSize)+100)
	_, err = tmpFile.WriteString(content)
	assert.NoError(t, err)
	tmpFile.Close()

	// Setup mock
	createSessionCalled := false
	uploadChunkCalled := 0
	mockSDK := &MockSDK{
		CreateUploadSessionFunc: func(remotePath string) (onedrive.UploadSession, error) {
			createSessionCalled = true
			assert.True(t, strings.HasSuffix(remotePath, filepath.Base(tmpFile.Name())))
			return onedrive.UploadSession{
				UploadURL:          "http://mock-upload-url.com",
				ExpirationDateTime: time.Now().Add(1 * time.Hour).Format(time.RFC3339),
			}, nil
		},
		UploadChunkFunc: func(uploadURL string, startByte, endByte, totalSize int64, chunkData io.Reader) (onedrive.UploadSession, error) {
			uploadChunkCalled++
			assert.Equal(t, "http://mock-upload-url.com", uploadURL)
			// Read the chunk to verify content if needed, for now just drain it
			io.Copy(ioutil.Discard, chunkData)
			return onedrive.UploadSession{
				NextExpectedRanges: []string{
					"12345-",
				},
			}, nil
		},
	}
	a := newTestApp(mockSDK)

	// Override session functions to use a temp directory
	oldGetConfigDir := session.GetConfigDir
	tmpDir, err := ioutil.TempDir("", "test-session-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)
	session.GetConfigDir = func() (string, error) { return tmpDir, nil }
	defer func() { session.GetConfigDir = oldGetConfigDir }()

	output := captureOutput(t, func() {
		err := filesUploadLogic(a, &cobra.Command{}, []string{tmpFile.Name(), "/remote/dest"})
		assert.NoError(t, err)
	})

	assert.True(t, createSessionCalled)
	assert.Equal(t, 2, uploadChunkCalled, "Expected UploadChunk to be called twice for a file > 1 chunk size")
	assert.Contains(t, output, "File '"+tmpFile.Name()+"' uploaded successfully")

	// Verify session file was deleted
	remoteFilePath := filepath.Join("/remote/dest", filepath.Base(tmpFile.Name()))
	sessionFilePath, err := session.GetSessionFilePath(tmpFile.Name(), remoteFilePath)
	assert.NoError(t, err)
	_, err = os.Stat(sessionFilePath)
	assert.True(t, os.IsNotExist(err), "Expected session file to be deleted after successful upload")
}

func TestFilesDownloadResumableLogic(t *testing.T) {
	// Create a dummy file for download
	content := strings.Repeat("a", int(chunkSize)+100)

	// Setup mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rangeHeader := r.Header.Get("Range")
		assert.NotEmpty(t, rangeHeader)

		parts := strings.Split(strings.Split(rangeHeader, "=")[1], "-")
		start, _ := strconv.ParseInt(parts[0], 10, 64)
		end, _ := strconv.ParseInt(parts[1], 10, 64)

		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(content)))
		w.WriteHeader(http.StatusPartialContent)
		w.Write([]byte(content[start : end+1]))
	}))
	defer server.Close()

	// Setup mock SDK
	mockSDK := &MockSDK{
		GetDriveItemByPathFunc: func(path string) (onedrive.DriveItem, error) {
			return onedrive.DriveItem{Size: int64(len(content))}, nil
		},
		DownloadFileChunkFunc: func(url string, startByte, endByte int64) (io.ReadCloser, error) {
			return onedrive.DownloadFileChunk(http.DefaultClient, server.URL, startByte, endByte)
		},
	}
	a := newTestApp(mockSDK)

	// Override session functions to use a temp directory
	oldGetConfigDir := session.GetConfigDir
	tmpDir, err := ioutil.TempDir("", "test-session-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)
	session.GetConfigDir = func() (string, error) { return tmpDir, nil }
	defer func() { session.GetConfigDir = oldGetConfigDir }()

	// First download (interrupted)
	captureOutput(t, func() {
		// We can't truly interrupt, so we'll just run it once and check the session file
		filesDownloadLogic(a, &cobra.Command{}, []string{"/remote/source/file.txt", filepath.Join(tmpDir, "downloaded-file.txt")})
	})

	// Second download (resume)
	output := captureOutput(t, func() {
		err := filesDownloadLogic(a, &cobra.Command{}, []string{"/remote/source/file.txt", filepath.Join(tmpDir, "downloaded-file.txt")})
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "File '/remote/source/file.txt' downloaded successfully")
	downloadedContent, err := ioutil.ReadFile(filepath.Join(tmpDir, "downloaded-file.txt"))
	assert.NoError(t, err)
	assert.Equal(t, content, string(downloadedContent))

	// Verify session file was deleted
	sessionFilePath, err := session.GetSessionFilePath(filepath.Join(tmpDir, "downloaded-file.txt"), "/remote/source/file.txt")
	assert.NoError(t, err)
	_, err = os.Stat(sessionFilePath)
	assert.True(t, os.IsNotExist(err), "Expected session file to be deleted after successful download")
}

func TestFilesCancelUploadLogic(t *testing.T) {
	mockSDK := &MockSDK{
		CancelUploadSessionFunc: func(uploadURL string) error {
			assert.Equal(t, "http://mock-upload-url.com", uploadURL)
			return nil
		},
	}
	a := newTestApp(mockSDK)

	output := captureOutput(t, func() {
		err := filesCancelUploadLogic(a, &cobra.Command{}, []string{"http://mock-upload-url.com"})
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "Upload session cancelled successfully")
}

func TestFilesCancelUploadLogicEmptyURL(t *testing.T) {
	mockSDK := &MockSDK{}
	a := newTestApp(mockSDK)

	err := filesCancelUploadLogic(a, &cobra.Command{}, []string{""})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "upload URL cannot be empty")
}

func TestFilesGetUploadStatusLogic(t *testing.T) {
	mockSDK := &MockSDK{
		GetUploadSessionStatusFunc: func(uploadURL string) (onedrive.UploadSession, error) {
			assert.Equal(t, "http://mock-upload-url.com", uploadURL)
			return onedrive.UploadSession{
				UploadURL:          uploadURL,
				ExpirationDateTime: "2024-12-01T12:00:00Z",
				NextExpectedRanges: []string{"1024-2047", "2048-"},
			}, nil
		},
	}
	a := newTestApp(mockSDK)

	output := captureOutput(t, func() {
		err := filesGetUploadStatusLogic(a, &cobra.Command{}, []string{"http://mock-upload-url.com"})
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "Upload Session Status:")
	assert.Contains(t, output, "http://mock-upload-url.com")
	assert.Contains(t, output, "2024-12-01T12:00:00Z")
	assert.Contains(t, output, "1024-2047")
}

func TestFilesGetUploadStatusLogicCompleted(t *testing.T) {
	mockSDK := &MockSDK{
		GetUploadSessionStatusFunc: func(uploadURL string) (onedrive.UploadSession, error) {
			return onedrive.UploadSession{
				UploadURL:          uploadURL,
				ExpirationDateTime: "2024-12-01T12:00:00Z",
				NextExpectedRanges: []string{}, // Empty means completed
			}, nil
		},
	}
	a := newTestApp(mockSDK)

	output := captureOutput(t, func() {
		err := filesGetUploadStatusLogic(a, &cobra.Command{}, []string{"http://mock-upload-url.com"})
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "Status: Upload completed")
}

func TestFilesUploadSimpleLogic(t *testing.T) {
	// Create a dummy file for upload
	tmpFile, err := ioutil.TempFile("", "test-upload-simple-*.txt")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	_, err = tmpFile.WriteString("test content")
	assert.NoError(t, err)
	tmpFile.Close()

	mockSDK := &MockSDK{
		UploadFileFunc: func(localPath, remotePath string) (onedrive.DriveItem, error) {
			assert.Equal(t, tmpFile.Name(), localPath)
			assert.Equal(t, "/remote/path/test.txt", remotePath)
			return onedrive.DriveItem{
				ID:   "mock-item-id-123",
				Name: "test.txt",
				Size: 12,
			}, nil
		},
	}
	a := newTestApp(mockSDK)

	output := captureOutput(t, func() {
		err := filesUploadSimpleLogic(a, &cobra.Command{}, []string{tmpFile.Name(), "/remote/path/test.txt"})
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "uploaded successfully")
	assert.Contains(t, output, "mock-item-id-123")
}

func TestFilesUploadSimpleLogicFileNotExists(t *testing.T) {
	mockSDK := &MockSDK{}
	a := newTestApp(mockSDK)

	err := filesUploadSimpleLogic(a, &cobra.Command{}, []string{"/nonexistent/file.txt", "/remote/path/test.txt"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestFilesListRootDeprecatedLogic(t *testing.T) {
	mockSDK := &MockSDK{
		GetRootDriveItemsFunc: func() (onedrive.DriveItemList, error) {
			return onedrive.DriveItemList{
				Value: []onedrive.DriveItem{
					{Name: "deprecated-file.txt", Size: 100},
				},
			}, nil
		},
	}
	a := newTestApp(mockSDK)

	output := captureOutput(t, func() {
		err := filesListRootDeprecatedLogic(a, &cobra.Command{}, []string{})
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "deprecated-file.txt")
}

func TestFilesRmLogic(t *testing.T) {
	mockSDK := &MockSDK{
		DeleteDriveItemFunc: func(path string) error {
			assert.Equal(t, "/test/file.txt", path)
			return nil
		},
	}
	a := newTestApp(mockSDK)

	output := captureOutput(t, func() {
		err := filesRmLogic(a, &cobra.Command{}, []string{"/test/file.txt"})
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "deleted successfully")
}

func TestFilesCopyLogic(t *testing.T) {
	mockSDK := &MockSDK{
		CopyDriveItemFunc: func(sourcePath, destinationPath, newName string) (string, error) {
			assert.Equal(t, "/source/file.txt", sourcePath)
			assert.Equal(t, "/destination", destinationPath)
			assert.Equal(t, "new-name.txt", newName)
			return "mock-monitor-url", nil
		},
	}
	a := newTestApp(mockSDK)

	cmd := &cobra.Command{}
	cmd.Flags().Bool("wait", false, "Wait for copy operation to complete")

	output := captureOutput(t, func() {
		err := filesCopyLogic(a, cmd, []string{"/source/file.txt", "/destination", "new-name.txt"})
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "copy initiated successfully")
	assert.Contains(t, output, "mock-monitor-url")
	assert.Contains(t, output, "files copy-status")
}

func TestFilesCopyLogicWithWait(t *testing.T) {
	mockSDK := &MockSDK{
		CopyDriveItemFunc: func(sourcePath, destinationPath, newName string) (string, error) {
			return "mock-monitor-url", nil
		},
		MonitorCopyOperationFunc: func(monitorURL string) (onedrive.CopyOperationStatus, error) {
			assert.Equal(t, "mock-monitor-url", monitorURL)
			return onedrive.CopyOperationStatus{
				Status:             "completed",
				PercentageComplete: 100,
				StatusDescription:  "Copy completed successfully",
				ResourceLocation:   "/destination/new-name.txt",
			}, nil
		},
	}
	a := newTestApp(mockSDK)

	cmd := &cobra.Command{}
	cmd.Flags().Bool("wait", false, "Wait for copy operation to complete")
	cmd.Flags().Set("wait", "true")

	output := captureOutput(t, func() {
		err := filesCopyLogic(a, cmd, []string{"/source/file.txt", "/destination", "new-name.txt"})
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "Copy completed successfully")
}

func TestFilesCopyStatusLogic(t *testing.T) {
	mockSDK := &MockSDK{
		MonitorCopyOperationFunc: func(monitorURL string) (onedrive.CopyOperationStatus, error) {
			assert.Equal(t, "mock-monitor-url", monitorURL)
			return onedrive.CopyOperationStatus{
				Status:             "inProgress",
				PercentageComplete: 75,
				StatusDescription:  "Copy operation in progress",
			}, nil
		},
	}
	a := newTestApp(mockSDK)

	output := captureOutput(t, func() {
		err := filesCopyStatusLogic(a, &cobra.Command{}, []string{"mock-monitor-url"})
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "Copy Operation Status")
	assert.Contains(t, output, "mock-monitor-url")
	assert.Contains(t, output, "inProgress")
	assert.Contains(t, output, "75%")
}

func TestFilesMvLogic(t *testing.T) {
	mockSDK := &MockSDK{
		MoveDriveItemFunc: func(sourcePath, destinationPath string) (onedrive.DriveItem, error) {
			assert.Equal(t, "/source/file.txt", sourcePath)
			assert.Equal(t, "/destination", destinationPath)
			return onedrive.DriveItem{
				Name: "file.txt",
				ParentReference: struct {
					DriveID   string `json:"driveId"`
					DriveType string `json:"driveType"`
					ID        string `json:"id"`
					Path      string `json:"path"`
				}{Path: "/destination"},
			}, nil
		},
	}
	a := newTestApp(mockSDK)

	output := captureOutput(t, func() {
		err := filesMvLogic(a, &cobra.Command{}, []string{"/source/file.txt", "/destination"})
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "moved successfully")
}

func TestFilesRenameLogic(t *testing.T) {
	mockSDK := &MockSDK{
		UpdateDriveItemFunc: func(path, newName string) (onedrive.DriveItem, error) {
			assert.Equal(t, "/test/old-name.txt", path)
			assert.Equal(t, "new-name.txt", newName)
			return onedrive.DriveItem{Name: "new-name.txt"}, nil
		},
	}
	a := newTestApp(mockSDK)

	output := captureOutput(t, func() {
		err := filesRenameLogic(a, &cobra.Command{}, []string{"/test/old-name.txt", "new-name.txt"})
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "renamed successfully")
	assert.Contains(t, output, "new-name.txt")
}

func TestFilesSearchLogic(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		mockItems   onedrive.DriveItemList
		mockError   error
		expectError bool
	}{
		{
			name:  "successful search",
			query: "test",
			mockItems: onedrive.DriveItemList{
				Value: []onedrive.DriveItem{
					{Name: "test-file.txt", Size: 1024},
					{Name: "another-test.docx", Size: 2048},
				},
			},
			mockError:   nil,
			expectError: false,
		},
		{
			name:        "empty query",
			query:       "",
			mockItems:   onedrive.DriveItemList{},
			mockError:   nil,
			expectError: true,
		},
		{
			name:        "search error",
			query:       "test",
			mockItems:   onedrive.DriveItemList{},
			mockError:   errors.New("search failed"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSDK := &MockSDK{
				SearchDriveItemsFunc: func(query string) (onedrive.DriveItemList, error) {
					if query == tt.query {
						return tt.mockItems, tt.mockError
					}
					return onedrive.DriveItemList{}, errors.New("unexpected query")
				},
			}

			app := newTestApp(mockSDK)
			cmd := &cobra.Command{}

			err := filesSearchLogic(app, cmd, []string{tt.query})

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFilesRecentLogic(t *testing.T) {
	tests := []struct {
		name        string
		mockItems   onedrive.DriveItemList
		mockError   error
		expectError bool
	}{
		{
			name: "successful recent items",
			mockItems: onedrive.DriveItemList{
				Value: []onedrive.DriveItem{
					{Name: "recent-file.txt", Size: 1024, LastModifiedDateTime: time.Now()},
					{Name: "another-recent.docx", Size: 2048, LastModifiedDateTime: time.Now()},
				},
			},
			mockError:   nil,
			expectError: false,
		},
		{
			name:        "recent items error",
			mockItems:   onedrive.DriveItemList{},
			mockError:   errors.New("recent items failed"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSDK := &MockSDK{
				GetRecentItemsFunc: func() (onedrive.DriveItemList, error) {
					return tt.mockItems, tt.mockError
				},
			}

			app := newTestApp(mockSDK)
			cmd := &cobra.Command{}

			err := filesRecentLogic(app, cmd, []string{})

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFilesSpecialLogic(t *testing.T) {
	tests := []struct {
		name        string
		folderName  string
		mockItem    onedrive.DriveItem
		mockError   error
		expectError bool
	}{
		{
			name:       "successful special folder",
			folderName: "documents",
			mockItem: onedrive.DriveItem{
				Name: "Documents",
				ID:   "12345",
				SpecialFolder: &struct {
					Name string `json:"name"`
				}{Name: "documents"},
			},
			mockError:   nil,
			expectError: false,
		},
		{
			name:        "empty folder name",
			folderName:  "",
			mockItem:    onedrive.DriveItem{},
			mockError:   nil,
			expectError: true,
		},
		{
			name:        "special folder error",
			folderName:  "documents",
			mockItem:    onedrive.DriveItem{},
			mockError:   errors.New("special folder failed"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSDK := &MockSDK{
				GetSpecialFolderFunc: func(folderName string) (onedrive.DriveItem, error) {
					if folderName == tt.folderName {
						return tt.mockItem, tt.mockError
					}
					return onedrive.DriveItem{}, errors.New("unexpected folder name")
				},
			}

			app := newTestApp(mockSDK)
			cmd := &cobra.Command{}

			err := filesSpecialLogic(app, cmd, []string{tt.folderName})

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
