package cmd

import (
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
					{Name: "deprecated-file1.txt", Size: 200},
					{Name: "deprecated-folder", Folder: &struct {
						ChildCount int `json:"childCount"`
						View       struct {
							ViewType  string `json:"viewType"`
							SortBy    string `json:"sortBy"`
							SortOrder string `json:"sortOrder"`
						} `json:"view"`
					}{ChildCount: 2}},
				},
			}, nil
		},
	}
	a := newTestApp(mockSDK)

	output := captureOutput(t, func() {
		err := filesListRootDeprecatedLogic(a, &cobra.Command{}, []string{})
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "deprecated-file1.txt")
	assert.Contains(t, output, "deprecated-folder")
}
