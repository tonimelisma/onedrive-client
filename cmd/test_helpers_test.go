package cmd

import (
	"bytes"
	"io"
	"log"
	"os"

	"github.com/tonimelisma/onedrive-client/internal/app"
	"github.com/tonimelisma/onedrive-client/pkg/onedrive"
)

// MockSDK is a mock implementation of the SDK interface for testing.
type MockSDK struct {
	GetDrivesFunc                  func() (onedrive.DriveList, error)
	GetDefaultDriveFunc            func() (onedrive.Drive, error)
	CreateFolderFunc               func(parentPath string, folderName string) (onedrive.DriveItem, error)
	DownloadFileFunc               func(remotePath, localPath string) error
	GetDriveItemByPathFunc         func(path string) (onedrive.DriveItem, error)
	GetDriveItemChildrenByPathFunc func(path string) (onedrive.DriveItemList, error)
	CreateUploadSessionFunc        func(remotePath string) (onedrive.UploadSession, error)
	UploadChunkFunc                func(uploadURL string, startByte, endByte, totalSize int64, chunkData io.Reader) (onedrive.UploadSession, error)
	GetUploadSessionStatusFunc     func(uploadURL string) (onedrive.UploadSession, error)
	CancelUploadSessionFunc        func(uploadURL string) error
}

func (m *MockSDK) GetDrives() (onedrive.DriveList, error) {
	if m.GetDrivesFunc != nil {
		return m.GetDrivesFunc()
	}
	return onedrive.DriveList{}, nil
}

func (m *MockSDK) GetDefaultDrive() (onedrive.Drive, error) {
	if m.GetDefaultDriveFunc != nil {
		return m.GetDefaultDriveFunc()
	}
	return onedrive.Drive{}, nil
}

func (m *MockSDK) CreateFolder(parentPath string, folderName string) (onedrive.DriveItem, error) {
	if m.CreateFolderFunc != nil {
		return m.CreateFolderFunc(parentPath, folderName)
	}
	return onedrive.DriveItem{}, nil
}

func (m *MockSDK) DownloadFile(remotePath, localPath string) error {
	if m.DownloadFileFunc != nil {
		return m.DownloadFileFunc(remotePath, localPath)
	}
	return nil
}

func (m *MockSDK) GetDriveItemByPath(path string) (onedrive.DriveItem, error) {
	if m.GetDriveItemByPathFunc != nil {
		return m.GetDriveItemByPathFunc(path)
	}
	return onedrive.DriveItem{}, nil
}

func (m *MockSDK) GetDriveItemChildrenByPath(path string) (onedrive.DriveItemList, error) {
	if m.GetDriveItemChildrenByPathFunc != nil {
		return m.GetDriveItemChildrenByPathFunc(path)
	}
	return onedrive.DriveItemList{}, nil
}

func (m *MockSDK) CreateUploadSession(remotePath string) (onedrive.UploadSession, error) {
	if m.CreateUploadSessionFunc != nil {
		return m.CreateUploadSessionFunc(remotePath)
	}
	return onedrive.UploadSession{}, nil
}

func (m *MockSDK) UploadChunk(uploadURL string, startByte, endByte, totalSize int64, chunkData io.Reader) (onedrive.UploadSession, error) {
	if m.UploadChunkFunc != nil {
		return m.UploadChunkFunc(uploadURL, startByte, endByte, totalSize, chunkData)
	}
	return onedrive.UploadSession{}, nil
}

func (m *MockSDK) GetUploadSessionStatus(uploadURL string) (onedrive.UploadSession, error) {
	if m.GetUploadSessionStatusFunc != nil {
		return m.GetUploadSessionStatusFunc(uploadURL)
	}
	return onedrive.UploadSession{}, nil
}

func (m *MockSDK) CancelUploadSession(uploadURL string) error {
	if m.CancelUploadSessionFunc != nil {
		return m.CancelUploadSessionFunc(uploadURL)
	}
	return nil
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
