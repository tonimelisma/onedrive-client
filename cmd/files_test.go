package cmd

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/tonimelisma/onedrive-client/internal/app"
	"github.com/tonimelisma/onedrive-client/pkg/onedrive"
)

// MockSDK is a mock implementation of the SDK interface for testing.
type MockSDK struct {
	CreateFolderFunc               func(client *http.Client, parentPath string, folderName string) (onedrive.DriveItem, error)
	UploadFileFunc                 func(client *http.Client, localPath, remotePath string) (onedrive.DriveItem, error)
	DownloadFileFunc               func(client *http.Client, remotePath, localPath string) error
	GetDriveItemByPathFunc         func(client *http.Client, path string) (onedrive.DriveItem, error)
	GetDriveItemChildrenByPathFunc func(client *http.Client, path string) (onedrive.DriveItemList, error)
}

func (m *MockSDK) CreateFolder(client *http.Client, parentPath string, folderName string) (onedrive.DriveItem, error) {
	if m.CreateFolderFunc != nil {
		return m.CreateFolderFunc(client, parentPath, folderName)
	}
	return onedrive.DriveItem{}, nil
}

func (m *MockSDK) UploadFile(client *http.Client, localPath, remotePath string) (onedrive.DriveItem, error) {
	if m.UploadFileFunc != nil {
		return m.UploadFileFunc(client, localPath, remotePath)
	}
	return onedrive.DriveItem{}, nil
}

func (m *MockSDK) DownloadFile(client *http.Client, remotePath, localPath string) error {
	if m.DownloadFileFunc != nil {
		return m.DownloadFileFunc(client, remotePath, localPath)
	}
	return nil
}

func (m *MockSDK) GetDriveItemByPath(client *http.Client, path string) (onedrive.DriveItem, error) {
	if m.GetDriveItemByPathFunc != nil {
		return m.GetDriveItemByPathFunc(client, path)
	}
	return onedrive.DriveItem{}, nil
}

func (m *MockSDK) GetDriveItemChildrenByPath(client *http.Client, path string) (onedrive.DriveItemList, error) {
	if m.GetDriveItemChildrenByPathFunc != nil {
		return m.GetDriveItemChildrenByPathFunc(client, path)
	}
	return onedrive.DriveItemList{}, nil
}

// newTestApp creates a new app instance with a mock SDK for testing.
func newTestApp(sdk app.SDK) *app.App {
	return &app.App{
		SDK: sdk,
	}
}

// captureOutput captures stdout and log output, returning it as a string.
func captureOutput(f func()) string {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	log.SetOutput(os.Stderr)
	w.Close()
	os.Stdout = oldStdout
	buf.ReadFrom(r)
	return buf.String()
}

func TestFilesListLogic(t *testing.T) {
	mockSDK := &MockSDK{
		GetDriveItemChildrenByPathFunc: func(client *http.Client, path string) (onedrive.DriveItemList, error) {
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
					}{}},
				},
			}, nil
		},
	}
	a := newTestApp(mockSDK)

	output := captureOutput(func() {
		err := filesListLogic(a, &cobra.Command{}, []string{"/test"})
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "file1.txt")
	assert.Contains(t, output, "subfolder")
}

func TestFilesMkdirLogic(t *testing.T) {
	mockSDK := &MockSDK{
		CreateFolderFunc: func(client *http.Client, parentPath, folderName string) (onedrive.DriveItem, error) {
			assert.Equal(t, "/test", parentPath)
			assert.Equal(t, "new-folder", folderName)
			return onedrive.DriveItem{Name: folderName}, nil
		},
	}
	a := newTestApp(mockSDK)

	output := captureOutput(func() {
		err := filesMkdirLogic(a, &cobra.Command{}, []string{"/test/new-folder"})
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "Folder 'new-folder' created successfully")
}

func TestFilesUploadLogic(t *testing.T) {
	tmpFile, err := ioutil.TempFile("", "test-upload-*.txt")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString("hello world")
	tmpFile.Close()

	mockSDK := &MockSDK{
		UploadFileFunc: func(client *http.Client, localPath, remotePath string) (onedrive.DriveItem, error) {
			assert.Equal(t, tmpFile.Name(), localPath)
			assert.True(t, strings.HasSuffix(remotePath, filepath.Base(tmpFile.Name())))
			return onedrive.DriveItem{Name: filepath.Base(remotePath)}, nil
		},
	}
	a := newTestApp(mockSDK)

	output := captureOutput(func() {
		err := filesUploadLogic(a, &cobra.Command{}, []string{tmpFile.Name(), "/remote/dest"})
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "File '"+tmpFile.Name()+"' uploaded successfully")
}

func TestFilesDownloadLogic(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "test-download-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)
	localPath := filepath.Join(tmpDir, "downloaded-file.txt")

	mockSDK := &MockSDK{
		DownloadFileFunc: func(client *http.Client, remotePath, localDestPath string) error {
			assert.Equal(t, "/remote/source/file.txt", remotePath)
			assert.Equal(t, localPath, localDestPath)
			return ioutil.WriteFile(localDestPath, []byte("downloaded content"), 0644)
		},
	}
	a := newTestApp(mockSDK)

	output := captureOutput(func() {
		err := filesDownloadLogic(a, &cobra.Command{}, []string{"/remote/source/file.txt", localPath})
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "File '/remote/source/file.txt' downloaded successfully")
	content, err := ioutil.ReadFile(localPath)
	assert.NoError(t, err)
	assert.Equal(t, "downloaded content", string(content))
}
