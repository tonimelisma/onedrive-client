package cmd

import (
	"io"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/tonimelisma/onedrive-client/internal/app"
	"github.com/tonimelisma/onedrive-client/pkg/onedrive"
)

// MockSDK implements the full SDK interface for testing
type MockSDK struct {
	// Core item operations
	GetDriveItemByPathFunc         func(path string) (onedrive.DriveItem, error)
	GetDriveItemChildrenByPathFunc func(path string) (onedrive.DriveItemList, error)
	GetRootDriveItemsFunc          func() (onedrive.DriveItemList, error)

	// Drive operations
	GetDrivesFunc          func() (onedrive.DriveList, error)
	GetDefaultDriveFunc    func() (onedrive.Drive, error)
	GetDriveByIDFunc       func(driveID string) (onedrive.Drive, error)
	GetDriveActivitiesFunc func(paging onedrive.Paging) (onedrive.ActivityList, string, error)

	// User operations
	GetMeFunc func() (onedrive.User, error)

	// File operations
	CreateFolderFunc         func(parentPath, folderName string) (onedrive.DriveItem, error)
	DeleteDriveItemFunc      func(path string) error
	CopyDriveItemFunc        func(sourcePath, destinationParentPath, newName string) (string, error)
	MoveDriveItemFunc        func(sourcePath, destinationParentPath string) (onedrive.DriveItem, error)
	UpdateDriveItemFunc      func(path, newName string) (onedrive.DriveItem, error)
	MonitorCopyOperationFunc func(monitorURL string) (onedrive.CopyOperationStatus, error)

	// Upload operations
	CreateUploadSessionFunc    func(remotePath string) (onedrive.UploadSession, error)
	UploadChunkFunc            func(uploadURL string, startByte, endByte, totalSize int64, chunkData io.Reader) (onedrive.UploadSession, error)
	GetUploadSessionStatusFunc func(uploadURL string) (onedrive.UploadSession, error)
	CancelUploadSessionFunc    func(uploadURL string) error
	UploadFileFunc             func(localPath, remotePath string) (onedrive.DriveItem, error)

	// Download operations
	DownloadFileFunc         func(remotePath, localPath string) error
	DownloadFileAsFormatFunc func(remotePath, localPath, format string) error
	DownloadFileChunkFunc    func(url string, startByte, endByte int64) (io.ReadCloser, error)

	// Search operations
	SearchDriveItemsFunc           func(query string) (onedrive.DriveItemList, error)
	SearchDriveItemsWithPagingFunc func(query string, paging onedrive.Paging) (onedrive.DriveItemList, string, error)
	SearchDriveItemsInFolderFunc   func(folderPath, query string, paging onedrive.Paging) (onedrive.DriveItemList, string, error)

	// Activity operations
	GetItemActivitiesFunc func(remotePath string, paging onedrive.Paging) (onedrive.ActivityList, string, error)

	// Sharing operations
	GetSharedWithMeFunc   func() (onedrive.DriveItemList, error)
	CreateSharingLinkFunc func(path, linkType, scope string) (onedrive.SharingLink, error)

	// Special operations
	GetRecentItemsFunc   func() (onedrive.DriveItemList, error)
	GetSpecialFolderFunc func(folderName string) (onedrive.DriveItem, error)
	GetDeltaFunc         func(deltaToken string) (onedrive.DeltaResponse, error)

	// Version operations
	GetFileVersionsFunc func(filePath string) (onedrive.DriveItemVersionList, error)

	// Epic 7 operations
	GetThumbnailsFunc      func(remotePath string) (onedrive.ThumbnailSetList, error)
	GetThumbnailBySizeFunc func(remotePath, thumbID, size string) (onedrive.Thumbnail, error)
	PreviewItemFunc        func(remotePath string, request onedrive.PreviewRequest) (onedrive.PreviewResponse, error)
	InviteUsersFunc        func(remotePath string, request onedrive.InviteRequest) (onedrive.InviteResponse, error)
	ListPermissionsFunc    func(remotePath string) (onedrive.PermissionList, error)
	GetPermissionFunc      func(remotePath, permissionID string) (onedrive.Permission, error)
	UpdatePermissionFunc   func(remotePath, permissionID string, request onedrive.UpdatePermissionRequest) (onedrive.Permission, error)
	DeletePermissionFunc   func(remotePath, permissionID string) error
}

// Core item operations
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

func (m *MockSDK) GetRootDriveItems() (onedrive.DriveItemList, error) {
	if m.GetRootDriveItemsFunc != nil {
		return m.GetRootDriveItemsFunc()
	}
	return onedrive.DriveItemList{}, nil
}

// Drive operations
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

func (m *MockSDK) GetDriveByID(driveID string) (onedrive.Drive, error) {
	if m.GetDriveByIDFunc != nil {
		return m.GetDriveByIDFunc(driveID)
	}
	return onedrive.Drive{}, nil
}

func (m *MockSDK) GetDriveActivities(paging onedrive.Paging) (onedrive.ActivityList, string, error) {
	if m.GetDriveActivitiesFunc != nil {
		return m.GetDriveActivitiesFunc(paging)
	}
	return onedrive.ActivityList{}, "", nil
}

// User operations
func (m *MockSDK) GetMe() (onedrive.User, error) {
	if m.GetMeFunc != nil {
		return m.GetMeFunc()
	}
	return onedrive.User{}, nil
}

// File operations
func (m *MockSDK) CreateFolder(parentPath, folderName string) (onedrive.DriveItem, error) {
	if m.CreateFolderFunc != nil {
		return m.CreateFolderFunc(parentPath, folderName)
	}
	return onedrive.DriveItem{}, nil
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
	return "", nil
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
	return onedrive.DriveItem{}, nil
}

func (m *MockSDK) MonitorCopyOperation(monitorURL string) (onedrive.CopyOperationStatus, error) {
	if m.MonitorCopyOperationFunc != nil {
		return m.MonitorCopyOperationFunc(monitorURL)
	}
	return onedrive.CopyOperationStatus{}, nil
}

// Upload operations
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

// Download operations
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
	return nil, nil
}

// Search operations
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

// Activity operations
func (m *MockSDK) GetItemActivities(remotePath string, paging onedrive.Paging) (onedrive.ActivityList, string, error) {
	if m.GetItemActivitiesFunc != nil {
		return m.GetItemActivitiesFunc(remotePath, paging)
	}
	return onedrive.ActivityList{}, "", nil
}

// Sharing operations
func (m *MockSDK) GetSharedWithMe() (onedrive.DriveItemList, error) {
	if m.GetSharedWithMeFunc != nil {
		return m.GetSharedWithMeFunc()
	}
	return onedrive.DriveItemList{}, nil
}

func (m *MockSDK) CreateSharingLink(path, linkType, scope string) (onedrive.SharingLink, error) {
	if m.CreateSharingLinkFunc != nil {
		return m.CreateSharingLinkFunc(path, linkType, scope)
	}
	return onedrive.SharingLink{}, nil
}

// Special operations
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

func (m *MockSDK) GetDelta(deltaToken string) (onedrive.DeltaResponse, error) {
	if m.GetDeltaFunc != nil {
		return m.GetDeltaFunc(deltaToken)
	}
	return onedrive.DeltaResponse{}, nil
}

// Version operations
func (m *MockSDK) GetFileVersions(filePath string) (onedrive.DriveItemVersionList, error) {
	if m.GetFileVersionsFunc != nil {
		return m.GetFileVersionsFunc(filePath)
	}
	return onedrive.DriveItemVersionList{}, nil
}

// Epic 7 operations
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

func newTestApp(mockSDK *MockSDK) *app.App {
	return &app.App{
		SDK: mockSDK,
	}
}

func TestFilesListLogic(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		mockFunc func(path string) (onedrive.DriveItemList, error)
		wantErr  bool
	}{
		{
			name: "list root directory",
			args: []string{},
			mockFunc: func(path string) (onedrive.DriveItemList, error) {
				assert.Equal(t, "/", path)
				return onedrive.DriveItemList{
					Value: []onedrive.DriveItem{
						{Name: "file1.txt", Size: 100},
						{Name: "folder1"},
					},
				}, nil
			},
			wantErr: false,
		},
		{
			name: "list specific directory",
			args: []string{"/Documents"},
			mockFunc: func(path string) (onedrive.DriveItemList, error) {
				assert.Equal(t, "/Documents", path)
				return onedrive.DriveItemList{
					Value: []onedrive.DriveItem{
						{Name: "document.docx", Size: 5000},
					},
				}, nil
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSDK := &MockSDK{
				GetDriveItemChildrenByPathFunc: tt.mockFunc,
			}
			a := newTestApp(mockSDK)

			err := filesListLogic(a, &cobra.Command{}, tt.args)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFilesStatLogic(t *testing.T) {
	mockSDK := &MockSDK{
		GetDriveItemByPathFunc: func(path string) (onedrive.DriveItem, error) {
			assert.Equal(t, "/test-file.txt", path)
			return onedrive.DriveItem{
				Name: "test-file.txt",
				Size: 1024,
				ID:   "test-id",
			}, nil
		},
	}
	a := newTestApp(mockSDK)

	err := filesStatLogic(a, &cobra.Command{}, []string{"/test-file.txt"})
	assert.NoError(t, err)
}

func TestFilesSearchLogic(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Int("top", 0, "")
	cmd.Flags().Bool("all", false, "")
	cmd.Flags().String("next", "", "")
	cmd.Flags().String("in", "", "")

	tests := []struct {
		name        string
		args        []string
		folderScope string
		mockFunc    func() (*MockSDK, error)
		wantErr     bool
	}{
		{
			name: "search globally",
			args: []string{"test query"},
			mockFunc: func() (*MockSDK, error) {
				return &MockSDK{
					SearchDriveItemsWithPagingFunc: func(query string, paging onedrive.Paging) (onedrive.DriveItemList, string, error) {
						assert.Equal(t, "test query", query)
						return onedrive.DriveItemList{}, "", nil
					},
				}, nil
			},
			wantErr: false,
		},
		{
			name:        "search in folder",
			args:        []string{"test query"},
			folderScope: "/Documents",
			mockFunc: func() (*MockSDK, error) {
				return &MockSDK{
					SearchDriveItemsInFolderFunc: func(folderPath, query string, paging onedrive.Paging) (onedrive.DriveItemList, string, error) {
						assert.Equal(t, "/Documents", folderPath)
						assert.Equal(t, "test query", query)
						return onedrive.DriveItemList{}, "", nil
					},
				}, nil
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.folderScope != "" {
				cmd.Flags().Set("in", tt.folderScope)
			}

			mockSDK, err := tt.mockFunc()
			assert.NoError(t, err)
			a := newTestApp(mockSDK)

			err = filesSearchLogic(a, cmd, tt.args)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFilesRecentLogic(t *testing.T) {
	mockSDK := &MockSDK{
		GetRecentItemsFunc: func() (onedrive.DriveItemList, error) {
			return onedrive.DriveItemList{
				Value: []onedrive.DriveItem{
					{Name: "recent-file.txt", Size: 500},
				},
			}, nil
		},
	}
	a := newTestApp(mockSDK)

	err := filesRecentLogic(a, &cobra.Command{}, []string{})
	assert.NoError(t, err)
}

func TestFilesSpecialLogic(t *testing.T) {
	mockSDK := &MockSDK{
		GetSpecialFolderFunc: func(folderName string) (onedrive.DriveItem, error) {
			assert.Equal(t, "Documents", folderName)
			return onedrive.DriveItem{
				Name: "Documents",
				ID:   "special-folder-id",
			}, nil
		},
	}
	a := newTestApp(mockSDK)

	err := filesSpecialLogic(a, &cobra.Command{}, []string{"Documents"})
	assert.NoError(t, err)
}

func TestFilesVersionsLogic(t *testing.T) {
	mockSDK := &MockSDK{
		GetFileVersionsFunc: func(filePath string) (onedrive.DriveItemVersionList, error) {
			assert.Equal(t, "/test-file.txt", filePath)
			return onedrive.DriveItemVersionList{
				Value: []onedrive.DriveItemVersion{
					{ID: "v1", Size: 1000},
					{ID: "v2", Size: 1100},
				},
			}, nil
		},
	}
	a := newTestApp(mockSDK)

	err := filesVersionsLogic(a, "/test-file.txt")
	assert.NoError(t, err)
}

func TestActivitiesLogic(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Int("top", 0, "")
	cmd.Flags().Bool("all", false, "")
	cmd.Flags().String("next", "", "")

	mockSDK := &MockSDK{
		GetItemActivitiesFunc: func(remotePath string, paging onedrive.Paging) (onedrive.ActivityList, string, error) {
			assert.Equal(t, "/test-file.txt", remotePath)
			return onedrive.ActivityList{
				Value: []onedrive.Activity{
					{ID: "activity1"},
				},
			}, "", nil
		},
	}
	a := newTestApp(mockSDK)

	err := activitiesLogic(a, cmd, []string{"/test-file.txt"})
	assert.NoError(t, err)
}

func TestFilesThumbnailsLogic(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "valid path",
			args:    []string{"/test-image.jpg"},
			wantErr: false,
		},
		{
			name:    "empty path",
			args:    []string{""},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSDK := &MockSDK{
				GetThumbnailsFunc: func(remotePath string) (onedrive.ThumbnailSetList, error) {
					return onedrive.ThumbnailSetList{}, nil
				},
			}
			a := newTestApp(mockSDK)

			err := filesThumbnailsLogic(a, &cobra.Command{}, tt.args)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFilesPreviewLogic(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("page", "", "")
	cmd.Flags().Float64("zoom", 1.0, "")

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "valid path",
			args:    []string{"/test-document.pdf"},
			wantErr: false,
		},
		{
			name:    "empty path",
			args:    []string{""},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSDK := &MockSDK{
				PreviewItemFunc: func(remotePath string, request onedrive.PreviewRequest) (onedrive.PreviewResponse, error) {
					return onedrive.PreviewResponse{}, nil
				},
			}
			a := newTestApp(mockSDK)

			err := filesPreviewLogic(a, cmd, tt.args)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
