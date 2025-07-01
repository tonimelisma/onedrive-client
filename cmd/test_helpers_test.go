package cmd

import (
	"context"
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
	GetDrivesFunc                  func(ctx context.Context) (onedrive.DriveList, error)
	GetDefaultDriveFunc            func(ctx context.Context) (onedrive.Drive, error)
	GetMeFunc                      func(ctx context.Context) (onedrive.User, error)
	CreateFolderFunc               func(ctx context.Context, parentPath string, folderName string) (onedrive.DriveItem, error)
	DownloadFileFunc               func(ctx context.Context, remotePath, localPath string) error
	DownloadFileAsFormatFunc       func(ctx context.Context, remotePath, localPath, format string) error
	DownloadFileChunkFunc          func(ctx context.Context, url string, startByte, endByte int64) (io.ReadCloser, error)
	GetDriveItemByPathFunc         func(ctx context.Context, path string) (onedrive.DriveItem, error)
	GetDriveItemChildrenByPathFunc func(ctx context.Context, path string) (onedrive.DriveItemList, error)
	CreateUploadSessionFunc        func(ctx context.Context, remotePath string) (onedrive.UploadSession, error)
	UploadChunkFunc                func(ctx context.Context, uploadURL string, startByte, endByte, totalSize int64, chunkData io.Reader) (onedrive.UploadSession, error)
	GetUploadSessionStatusFunc     func(ctx context.Context, uploadURL string) (onedrive.UploadSession, error)
	CancelUploadSessionFunc        func(ctx context.Context, uploadURL string) error
	UploadFileFunc                 func(ctx context.Context, localPath, remotePath string) (onedrive.DriveItem, error)
	GetRootDriveItemsFunc          func(ctx context.Context) (onedrive.DriveItemList, error)
	DeleteDriveItemFunc            func(ctx context.Context, path string) error
	CopyDriveItemFunc              func(ctx context.Context, sourcePath, destinationParentPath, newName string) (string, error)
	MoveDriveItemFunc              func(ctx context.Context, sourcePath, destinationParentPath string) (onedrive.DriveItem, error)
	UpdateDriveItemFunc            func(ctx context.Context, path, newName string) (onedrive.DriveItem, error)
	MonitorCopyOperationFunc       func(ctx context.Context, monitorURL string) (onedrive.CopyOperationStatus, error)
	SearchDriveItemsFunc           func(ctx context.Context, query string) (onedrive.DriveItemList, error)
	SearchDriveItemsWithPagingFunc func(ctx context.Context, query string, paging onedrive.Paging) (onedrive.DriveItemList, string, error)
	SearchDriveItemsInFolderFunc   func(ctx context.Context, folderPath, query string, paging onedrive.Paging) (onedrive.DriveItemList, string, error)
	GetDriveActivitiesFunc         func(ctx context.Context, paging onedrive.Paging) (onedrive.ActivityList, string, error)
	GetItemActivitiesFunc          func(ctx context.Context, remotePath string, paging onedrive.Paging) (onedrive.ActivityList, string, error)
	GetSharedWithMeFunc            func(ctx context.Context) (onedrive.DriveItemList, error)
	GetRecentItemsFunc             func(ctx context.Context) (onedrive.DriveItemList, error)
	GetSpecialFolderFunc           func(ctx context.Context, folderName string) (onedrive.DriveItem, error)
	CreateSharingLinkFunc          func(ctx context.Context, path, linkType, scope string) (onedrive.SharingLink, error)
	GetDeltaFunc                   func(ctx context.Context, deltaToken string) (onedrive.DeltaResponse, error)
	GetDriveByIDFunc               func(ctx context.Context, driveID string) (onedrive.Drive, error)
	GetFileVersionsFunc            func(ctx context.Context, filePath string) (onedrive.DriveItemVersionList, error)
	// New Epic 7 function fields
	GetThumbnailsFunc      func(ctx context.Context, remotePath string) (onedrive.ThumbnailSetList, error)
	GetThumbnailBySizeFunc func(ctx context.Context, remotePath, thumbID, size string) (onedrive.Thumbnail, error)
	PreviewItemFunc        func(ctx context.Context, remotePath string, request onedrive.PreviewRequest) (onedrive.PreviewResponse, error)
	InviteUsersFunc        func(ctx context.Context, remotePath string, request onedrive.InviteRequest) (onedrive.InviteResponse, error)
	ListPermissionsFunc    func(ctx context.Context, remotePath string) (onedrive.PermissionList, error)
	GetPermissionFunc      func(ctx context.Context, remotePath, permissionID string) (onedrive.Permission, error)
	UpdatePermissionFunc   func(ctx context.Context, remotePath, permissionID string, request onedrive.UpdatePermissionRequest) (onedrive.Permission, error)
	DeletePermissionFunc   func(ctx context.Context, remotePath, permissionID string) error
}

func (m *MockSDK) GetDrives(ctx context.Context) (onedrive.DriveList, error) {
	if m.GetDrivesFunc != nil {
		return m.GetDrivesFunc(ctx)
	}
	return onedrive.DriveList{}, nil
}

func (m *MockSDK) GetDefaultDrive(ctx context.Context) (onedrive.Drive, error) {
	if m.GetDefaultDriveFunc != nil {
		return m.GetDefaultDriveFunc(ctx)
	}
	return onedrive.Drive{}, nil
}

func (m *MockSDK) GetMe(ctx context.Context) (onedrive.User, error) {
	if m.GetMeFunc != nil {
		return m.GetMeFunc(ctx)
	}
	return onedrive.User{}, nil
}

func (m *MockSDK) CreateFolder(ctx context.Context, parentPath string, folderName string) (onedrive.DriveItem, error) {
	if m.CreateFolderFunc != nil {
		return m.CreateFolderFunc(ctx, parentPath, folderName)
	}
	return onedrive.DriveItem{}, nil
}

func (m *MockSDK) DownloadFile(ctx context.Context, remotePath, localPath string) error {
	if m.DownloadFileFunc != nil {
		return m.DownloadFileFunc(ctx, remotePath, localPath)
	}
	return nil
}

func (m *MockSDK) DownloadFileAsFormat(ctx context.Context, remotePath, localPath, format string) error {
	if m.DownloadFileAsFormatFunc != nil {
		return m.DownloadFileAsFormatFunc(ctx, remotePath, localPath, format)
	}
	return nil
}

func (m *MockSDK) DownloadFileChunk(ctx context.Context, url string, startByte, endByte int64) (io.ReadCloser, error) {
	if m.DownloadFileChunkFunc != nil {
		return m.DownloadFileChunkFunc(ctx, url, startByte, endByte)
	}
	return nil, errors.New("not implemented")
}

func (m *MockSDK) GetDriveItemByPath(ctx context.Context, path string) (onedrive.DriveItem, error) {
	if m.GetDriveItemByPathFunc != nil {
		return m.GetDriveItemByPathFunc(ctx, path)
	}
	return onedrive.DriveItem{}, nil
}

func (m *MockSDK) GetDriveItemChildrenByPath(ctx context.Context, path string) (onedrive.DriveItemList, error) {
	if m.GetDriveItemChildrenByPathFunc != nil {
		return m.GetDriveItemChildrenByPathFunc(ctx, path)
	}
	return onedrive.DriveItemList{}, nil
}

func (m *MockSDK) CreateUploadSession(ctx context.Context, remotePath string) (onedrive.UploadSession, error) {
	if m.CreateUploadSessionFunc != nil {
		return m.CreateUploadSessionFunc(ctx, remotePath)
	}
	return onedrive.UploadSession{}, nil
}

func (m *MockSDK) UploadChunk(ctx context.Context, uploadURL string, startByte, endByte, totalSize int64, chunkData io.Reader) (onedrive.UploadSession, error) {
	if m.UploadChunkFunc != nil {
		return m.UploadChunkFunc(ctx, uploadURL, startByte, endByte, totalSize, chunkData)
	}
	return onedrive.UploadSession{}, nil
}

func (m *MockSDK) GetUploadSessionStatus(ctx context.Context, uploadURL string) (onedrive.UploadSession, error) {
	if m.GetUploadSessionStatusFunc != nil {
		return m.GetUploadSessionStatusFunc(ctx, uploadURL)
	}
	return onedrive.UploadSession{}, nil
}

func (m *MockSDK) CancelUploadSession(ctx context.Context, uploadURL string) error {
	if m.CancelUploadSessionFunc != nil {
		return m.CancelUploadSessionFunc(ctx, uploadURL)
	}
	return nil
}

func (m *MockSDK) UploadFile(ctx context.Context, localPath, remotePath string) (onedrive.DriveItem, error) {
	if m.UploadFileFunc != nil {
		return m.UploadFileFunc(ctx, localPath, remotePath)
	}
	return onedrive.DriveItem{}, nil
}

func (m *MockSDK) GetRootDriveItems(ctx context.Context) (onedrive.DriveItemList, error) {
	if m.GetRootDriveItemsFunc != nil {
		return m.GetRootDriveItemsFunc(ctx)
	}
	return onedrive.DriveItemList{}, nil
}

func (m *MockSDK) DeleteDriveItem(ctx context.Context, path string) error {
	if m.DeleteDriveItemFunc != nil {
		return m.DeleteDriveItemFunc(ctx, path)
	}
	return nil
}

func (m *MockSDK) CopyDriveItem(ctx context.Context, sourcePath, destinationParentPath, newName string) (string, error) {
	if m.CopyDriveItemFunc != nil {
		return m.CopyDriveItemFunc(ctx, sourcePath, destinationParentPath, newName)
	}
	return "mock-monitor-url", nil
}

func (m *MockSDK) MoveDriveItem(ctx context.Context, sourcePath, destinationParentPath string) (onedrive.DriveItem, error) {
	if m.MoveDriveItemFunc != nil {
		return m.MoveDriveItemFunc(ctx, sourcePath, destinationParentPath)
	}
	return onedrive.DriveItem{}, nil
}

func (m *MockSDK) UpdateDriveItem(ctx context.Context, path, newName string) (onedrive.DriveItem, error) {
	if m.UpdateDriveItemFunc != nil {
		return m.UpdateDriveItemFunc(ctx, path, newName)
	}
	return onedrive.DriveItem{Name: newName}, nil
}

func (m *MockSDK) MonitorCopyOperation(ctx context.Context, monitorURL string) (onedrive.CopyOperationStatus, error) {
	if m.MonitorCopyOperationFunc != nil {
		return m.MonitorCopyOperationFunc(ctx, monitorURL)
	}
	return onedrive.CopyOperationStatus{
		Status:             "completed",
		PercentageComplete: 100,
		StatusDescription:  "Copy completed successfully",
	}, nil
}

func (m *MockSDK) SearchDriveItems(ctx context.Context, query string) (onedrive.DriveItemList, error) {
	if m.SearchDriveItemsFunc != nil {
		return m.SearchDriveItemsFunc(ctx, query)
	}
	return onedrive.DriveItemList{}, nil
}

func (m *MockSDK) SearchDriveItemsWithPaging(ctx context.Context, query string, paging onedrive.Paging) (onedrive.DriveItemList, string, error) {
	if m.SearchDriveItemsWithPagingFunc != nil {
		return m.SearchDriveItemsWithPagingFunc(ctx, query, paging)
	}
	return onedrive.DriveItemList{}, "", nil
}

func (m *MockSDK) SearchDriveItemsInFolder(ctx context.Context, folderPath, query string, paging onedrive.Paging) (onedrive.DriveItemList, string, error) {
	if m.SearchDriveItemsInFolderFunc != nil {
		return m.SearchDriveItemsInFolderFunc(ctx, folderPath, query, paging)
	}
	return onedrive.DriveItemList{}, "", nil
}

func (m *MockSDK) GetDriveActivities(ctx context.Context, paging onedrive.Paging) (onedrive.ActivityList, string, error) {
	if m.GetDriveActivitiesFunc != nil {
		return m.GetDriveActivitiesFunc(ctx, paging)
	}
	return onedrive.ActivityList{}, "", nil
}

func (m *MockSDK) GetItemActivities(ctx context.Context, remotePath string, paging onedrive.Paging) (onedrive.ActivityList, string, error) {
	if m.GetItemActivitiesFunc != nil {
		return m.GetItemActivitiesFunc(ctx, remotePath, paging)
	}
	return onedrive.ActivityList{}, "", nil
}

func (m *MockSDK) GetSharedWithMe(ctx context.Context) (onedrive.DriveItemList, error) {
	if m.GetSharedWithMeFunc != nil {
		return m.GetSharedWithMeFunc(ctx)
	}
	return onedrive.DriveItemList{}, nil
}

func (m *MockSDK) GetRecentItems(ctx context.Context) (onedrive.DriveItemList, error) {
	if m.GetRecentItemsFunc != nil {
		return m.GetRecentItemsFunc(ctx)
	}
	return onedrive.DriveItemList{}, nil
}

func (m *MockSDK) GetSpecialFolder(ctx context.Context, folderName string) (onedrive.DriveItem, error) {
	if m.GetSpecialFolderFunc != nil {
		return m.GetSpecialFolderFunc(ctx, folderName)
	}
	return onedrive.DriveItem{}, nil
}

func (m *MockSDK) CreateSharingLink(ctx context.Context, path, linkType, scope string) (onedrive.SharingLink, error) {
	if m.CreateSharingLinkFunc != nil {
		return m.CreateSharingLinkFunc(ctx, path, linkType, scope)
	}
	return onedrive.SharingLink{}, nil
}

func (m *MockSDK) GetDelta(ctx context.Context, deltaToken string) (onedrive.DeltaResponse, error) {
	if m.GetDeltaFunc != nil {
		return m.GetDeltaFunc(ctx, deltaToken)
	}
	return onedrive.DeltaResponse{}, nil
}

func (m *MockSDK) GetDriveByID(ctx context.Context, driveID string) (onedrive.Drive, error) {
	if m.GetDriveByIDFunc != nil {
		return m.GetDriveByIDFunc(ctx, driveID)
	}
	return onedrive.Drive{}, nil
}

func (m *MockSDK) GetFileVersions(ctx context.Context, filePath string) (onedrive.DriveItemVersionList, error) {
	if m.GetFileVersionsFunc != nil {
		return m.GetFileVersionsFunc(ctx, filePath)
	}
	return onedrive.DriveItemVersionList{}, nil
}

// New Epic 7 mock method implementations

func (m *MockSDK) GetThumbnails(ctx context.Context, remotePath string) (onedrive.ThumbnailSetList, error) {
	if m.GetThumbnailsFunc != nil {
		return m.GetThumbnailsFunc(ctx, remotePath)
	}
	return onedrive.ThumbnailSetList{}, nil
}

func (m *MockSDK) GetThumbnailBySize(ctx context.Context, remotePath, thumbID, size string) (onedrive.Thumbnail, error) {
	if m.GetThumbnailBySizeFunc != nil {
		return m.GetThumbnailBySizeFunc(ctx, remotePath, thumbID, size)
	}
	return onedrive.Thumbnail{}, nil
}

func (m *MockSDK) PreviewItem(ctx context.Context, remotePath string, request onedrive.PreviewRequest) (onedrive.PreviewResponse, error) {
	if m.PreviewItemFunc != nil {
		return m.PreviewItemFunc(ctx, remotePath, request)
	}
	return onedrive.PreviewResponse{}, nil
}

func (m *MockSDK) InviteUsers(ctx context.Context, remotePath string, request onedrive.InviteRequest) (onedrive.InviteResponse, error) {
	if m.InviteUsersFunc != nil {
		return m.InviteUsersFunc(ctx, remotePath, request)
	}
	return onedrive.InviteResponse{}, nil
}

func (m *MockSDK) ListPermissions(ctx context.Context, remotePath string) (onedrive.PermissionList, error) {
	if m.ListPermissionsFunc != nil {
		return m.ListPermissionsFunc(ctx, remotePath)
	}
	return onedrive.PermissionList{}, nil
}

func (m *MockSDK) GetPermission(ctx context.Context, remotePath, permissionID string) (onedrive.Permission, error) {
	if m.GetPermissionFunc != nil {
		return m.GetPermissionFunc(ctx, remotePath, permissionID)
	}
	return onedrive.Permission{}, nil
}

func (m *MockSDK) UpdatePermission(ctx context.Context, remotePath, permissionID string, request onedrive.UpdatePermissionRequest) (onedrive.Permission, error) {
	if m.UpdatePermissionFunc != nil {
		return m.UpdatePermissionFunc(ctx, remotePath, permissionID, request)
	}
	return onedrive.Permission{}, nil
}

func (m *MockSDK) DeletePermission(ctx context.Context, remotePath, permissionID string) error {
	if m.DeletePermissionFunc != nil {
		return m.DeletePermissionFunc(ctx, remotePath, permissionID)
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
