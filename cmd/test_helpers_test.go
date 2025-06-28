package cmd

import (
	"errors"
	"io"
	"log"
	"os"
	"testing"

	"github.com/tonimelisma/onedrive-client/internal/app"
	"github.com/tonimelisma/onedrive-client/pkg/onedrive"
)

// MockSDK is a mock implementation of the SDK interface for testing.
type MockSDK struct {
	GetDrivesFunc                  func() (onedrive.DriveList, error)
	GetDefaultDriveFunc            func() (onedrive.Drive, error)
	GetMeFunc                      func() (onedrive.User, error)
	InitiateDeviceCodeFlowFunc     func() (*onedrive.DeviceCodeResponse, error)
	VerifyDeviceCodeFunc           func(deviceCode string) (*onedrive.OAuthToken, error)
	CreateFolderFunc               func(parentPath string, folderName string) (onedrive.DriveItem, error)
	DownloadFileFunc               func(remotePath, localPath string) error
	DownloadFileChunkFunc          func(url string, startByte, endByte int64) (io.ReadCloser, error)
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

func (m *MockSDK) GetMe() (onedrive.User, error) {
	if m.GetMeFunc != nil {
		return m.GetMeFunc()
	}
	return onedrive.User{}, nil
}

func (m *MockSDK) InitiateDeviceCodeFlow() (*onedrive.DeviceCodeResponse, error) {
	if m.InitiateDeviceCodeFlowFunc != nil {
		return m.InitiateDeviceCodeFlowFunc()
	}
	return nil, errors.New("not implemented")
}

func (m *MockSDK) VerifyDeviceCode(deviceCode string) (*onedrive.OAuthToken, error) {
	if m.VerifyDeviceCodeFunc != nil {
		return m.VerifyDeviceCodeFunc(deviceCode)
	}
	return nil, errors.New("not implemented")
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

func (m *MockSDK) DownloadFileChunk(url string, startByte, endByte int64) (io.ReadCloser, error) {
	if m.DownloadFileChunkFunc != nil {
		return m.DownloadFileChunkFunc(url, startByte, endByte)
	}
	return nil, errors.New("not implemented")
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

// captureOutput captures stdout and stderr, returning them as a string.
// This version doesn't mutate global log state.
func captureOutput(t *testing.T, f func()) string {
	t.Helper()

	// Save original log output
	originalLogOutput := log.Writer()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Capture stderr and redirect log to it
	oldStderr := os.Stderr
	r2, w2, _ := os.Pipe()
	os.Stderr = w2
	log.SetOutput(w2)

	// Run the function
	f()

	// Restore everything
	w.Close()
	w2.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr
	log.SetOutput(originalLogOutput)

	// Read captured output
	stdout, _ := io.ReadAll(r)
	stderr, _ := io.ReadAll(r2)

	// Combine stdout and stderr
	output := string(stdout) + string(stderr)
	return output
}
