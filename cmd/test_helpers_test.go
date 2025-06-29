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
	DownloadFileAsFormatFunc       func(remotePath, localPath, format string) error
	DownloadFileChunkFunc          func(url string, startByte, endByte int64) (io.ReadCloser, error)
	GetDriveItemByPathFunc         func(path string) (onedrive.DriveItem, error)
	GetDriveItemChildrenByPathFunc func(path string) (onedrive.DriveItemList, error)
	CreateUploadSessionFunc        func(remotePath string) (onedrive.UploadSession, error)
	UploadChunkFunc                func(uploadURL string, startByte, endByte, totalSize int64, chunkData io.Reader) (onedrive.UploadSession, error)
	GetUploadSessionStatusFunc     func(uploadURL string) (onedrive.UploadSession, error)
	CancelUploadSessionFunc        func(uploadURL string) error
	UploadFileFunc                 func(localPath, remotePath string) (onedrive.DriveItem, error)
	GetRootDriveItemsFunc          func() (onedrive.DriveItemList, error)
	DeleteDriveItemFunc            func(path string) error
	CopyDriveItemFunc              func(sourcePath, destinationParentPath, newName string) (string, error)
	MoveDriveItemFunc              func(sourcePath, destinationParentPath string) (onedrive.DriveItem, error)
	UpdateDriveItemFunc            func(path, newName string) (onedrive.DriveItem, error)
	MonitorCopyOperationFunc       func(monitorURL string) (onedrive.CopyOperationStatus, error)
	SearchDriveItemsFunc           func(query string) (onedrive.DriveItemList, error)
	SearchDriveItemsWithPagingFunc func(query string, paging onedrive.Paging) (onedrive.DriveItemList, string, error)
	SearchDriveItemsInFolderFunc   func(folderPath, query string, paging onedrive.Paging) (onedrive.DriveItemList, string, error)
	GetDriveActivitiesFunc         func(paging onedrive.Paging) (onedrive.ActivityList, string, error)
	GetItemActivitiesFunc          func(remotePath string, paging onedrive.Paging) (onedrive.ActivityList, string, error)
	GetSharedWithMeFunc            func() (onedrive.DriveItemList, error)
	GetRecentItemsFunc             func() (onedrive.DriveItemList, error)
	GetSpecialFolderFunc           func(folderName string) (onedrive.DriveItem, error)
	CreateSharingLinkFunc          func(path, linkType, scope string) (onedrive.SharingLink, error)
	GetDeltaFunc                   func(deltaToken string) (onedrive.DeltaResponse, error)
	GetDriveByIDFunc               func(driveID string) (onedrive.Drive, error)
	GetFileVersionsFunc            func(filePath string) (onedrive.DriveItemVersionList, error)
	// New Epic 7 function fields
	GetThumbnailsFunc      func(remotePath string) (onedrive.ThumbnailSetList, error)
	GetThumbnailBySizeFunc func(remotePath, thumbID, size string) (onedrive.Thumbnail, error)
	PreviewItemFunc        func(remotePath string, request onedrive.PreviewRequest) (onedrive.PreviewResponse, error)
	InviteUsersFunc        func(remotePath string, request onedrive.InviteRequest) (onedrive.InviteResponse, error)
	ListPermissionsFunc    func(remotePath string) (onedrive.PermissionList, error)
	GetPermissionFunc      func(remotePath, permissionID string) (onedrive.Permission, error)
	UpdatePermissionFunc   func(remotePath, permissionID string, request onedrive.UpdatePermissionRequest) (onedrive.Permission, error)
	DeletePermissionFunc   func(remotePath, permissionID string) error
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

func (m *MockSDK) DownloadFileAsFormat(remotePath, localPath, format string) error {
	if m.DownloadFileAsFormatFunc != nil {
		return m.DownloadFileAsFormatFunc(remotePath, localPath, format)
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

func (m *MockSDK) UploadFile(localPath, remotePath string) (onedrive.DriveItem, error) {
	if m.UploadFileFunc != nil {
		return m.UploadFileFunc(localPath, remotePath)
	}
	return onedrive.DriveItem{}, nil
}

func (m *MockSDK) GetRootDriveItems() (onedrive.DriveItemList, error) {
	if m.GetRootDriveItemsFunc != nil {
		return m.GetRootDriveItemsFunc()
	}
	return onedrive.DriveItemList{}, nil
}

func (m *MockSDK) DeleteDriveItem(path string) error {
	if m.DeleteDriveItemFunc != nil {
		return m.DeleteDriveItemFunc(path)
	}
	return nil
}

func (m *MockSDK) CopyDriveItem(sourcePath, destinationParentPath, newName string) (string, error) {
	if m.CopyDriveItemFunc != nil {
		return m.CopyDriveItemFunc(sourcePath, destinationParentPath, newName)
	}
	return "mock-monitor-url", nil
}

func (m *MockSDK) MoveDriveItem(sourcePath, destinationParentPath string) (onedrive.DriveItem, error) {
	if m.MoveDriveItemFunc != nil {
		return m.MoveDriveItemFunc(sourcePath, destinationParentPath)
	}
	return onedrive.DriveItem{}, nil
}

func (m *MockSDK) UpdateDriveItem(path, newName string) (onedrive.DriveItem, error) {
	if m.UpdateDriveItemFunc != nil {
		return m.UpdateDriveItemFunc(path, newName)
	}
	return onedrive.DriveItem{Name: newName}, nil
}

func (m *MockSDK) MonitorCopyOperation(monitorURL string) (onedrive.CopyOperationStatus, error) {
	if m.MonitorCopyOperationFunc != nil {
		return m.MonitorCopyOperationFunc(monitorURL)
	}
	return onedrive.CopyOperationStatus{
		Status:             "completed",
		PercentageComplete: 100,
		StatusDescription:  "Copy completed successfully",
	}, nil
}

func (m *MockSDK) SearchDriveItems(query string) (onedrive.DriveItemList, error) {
	if m.SearchDriveItemsFunc != nil {
		return m.SearchDriveItemsFunc(query)
	}
	return onedrive.DriveItemList{}, nil
}

func (m *MockSDK) SearchDriveItemsWithPaging(query string, paging onedrive.Paging) (onedrive.DriveItemList, string, error) {
	if m.SearchDriveItemsWithPagingFunc != nil {
		return m.SearchDriveItemsWithPagingFunc(query, paging)
	}
	return onedrive.DriveItemList{}, "", nil
}

func (m *MockSDK) SearchDriveItemsInFolder(folderPath, query string, paging onedrive.Paging) (onedrive.DriveItemList, string, error) {
	if m.SearchDriveItemsInFolderFunc != nil {
		return m.SearchDriveItemsInFolderFunc(folderPath, query, paging)
	}
	return onedrive.DriveItemList{}, "", nil
}

func (m *MockSDK) GetDriveActivities(paging onedrive.Paging) (onedrive.ActivityList, string, error) {
	if m.GetDriveActivitiesFunc != nil {
		return m.GetDriveActivitiesFunc(paging)
	}
	return onedrive.ActivityList{}, "", nil
}

func (m *MockSDK) GetItemActivities(remotePath string, paging onedrive.Paging) (onedrive.ActivityList, string, error) {
	if m.GetItemActivitiesFunc != nil {
		return m.GetItemActivitiesFunc(remotePath, paging)
	}
	return onedrive.ActivityList{}, "", nil
}

func (m *MockSDK) GetSharedWithMe() (onedrive.DriveItemList, error) {
	if m.GetSharedWithMeFunc != nil {
		return m.GetSharedWithMeFunc()
	}
	return onedrive.DriveItemList{}, nil
}

func (m *MockSDK) GetRecentItems() (onedrive.DriveItemList, error) {
	if m.GetRecentItemsFunc != nil {
		return m.GetRecentItemsFunc()
	}
	return onedrive.DriveItemList{}, nil
}

func (m *MockSDK) GetSpecialFolder(folderName string) (onedrive.DriveItem, error) {
	if m.GetSpecialFolderFunc != nil {
		return m.GetSpecialFolderFunc(folderName)
	}
	return onedrive.DriveItem{}, nil
}

func (m *MockSDK) CreateSharingLink(path, linkType, scope string) (onedrive.SharingLink, error) {
	if m.CreateSharingLinkFunc != nil {
		return m.CreateSharingLinkFunc(path, linkType, scope)
	}
	return onedrive.SharingLink{}, nil
}

func (m *MockSDK) GetDelta(deltaToken string) (onedrive.DeltaResponse, error) {
	if m.GetDeltaFunc != nil {
		return m.GetDeltaFunc(deltaToken)
	}
	return onedrive.DeltaResponse{}, nil
}

func (m *MockSDK) GetDriveByID(driveID string) (onedrive.Drive, error) {
	if m.GetDriveByIDFunc != nil {
		return m.GetDriveByIDFunc(driveID)
	}
	return onedrive.Drive{}, nil
}

func (m *MockSDK) GetFileVersions(filePath string) (onedrive.DriveItemVersionList, error) {
	if m.GetFileVersionsFunc != nil {
		return m.GetFileVersionsFunc(filePath)
	}
	return onedrive.DriveItemVersionList{}, nil
}

// New Epic 7 mock method implementations

func (m *MockSDK) GetThumbnails(remotePath string) (onedrive.ThumbnailSetList, error) {
	if m.GetThumbnailsFunc != nil {
		return m.GetThumbnailsFunc(remotePath)
	}
	return onedrive.ThumbnailSetList{}, nil
}

func (m *MockSDK) GetThumbnailBySize(remotePath, thumbID, size string) (onedrive.Thumbnail, error) {
	if m.GetThumbnailBySizeFunc != nil {
		return m.GetThumbnailBySizeFunc(remotePath, thumbID, size)
	}
	return onedrive.Thumbnail{}, nil
}

func (m *MockSDK) PreviewItem(remotePath string, request onedrive.PreviewRequest) (onedrive.PreviewResponse, error) {
	if m.PreviewItemFunc != nil {
		return m.PreviewItemFunc(remotePath, request)
	}
	return onedrive.PreviewResponse{}, nil
}

func (m *MockSDK) InviteUsers(remotePath string, request onedrive.InviteRequest) (onedrive.InviteResponse, error) {
	if m.InviteUsersFunc != nil {
		return m.InviteUsersFunc(remotePath, request)
	}
	return onedrive.InviteResponse{}, nil
}

func (m *MockSDK) ListPermissions(remotePath string) (onedrive.PermissionList, error) {
	if m.ListPermissionsFunc != nil {
		return m.ListPermissionsFunc(remotePath)
	}
	return onedrive.PermissionList{}, nil
}

func (m *MockSDK) GetPermission(remotePath, permissionID string) (onedrive.Permission, error) {
	if m.GetPermissionFunc != nil {
		return m.GetPermissionFunc(remotePath, permissionID)
	}
	return onedrive.Permission{}, nil
}

func (m *MockSDK) UpdatePermission(remotePath, permissionID string, request onedrive.UpdatePermissionRequest) (onedrive.Permission, error) {
	if m.UpdatePermissionFunc != nil {
		return m.UpdatePermissionFunc(remotePath, permissionID, request)
	}
	return onedrive.Permission{}, nil
}

func (m *MockSDK) DeletePermission(remotePath, permissionID string) error {
	if m.DeletePermissionFunc != nil {
		return m.DeletePermissionFunc(remotePath, permissionID)
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
