package cmd

import (
	"context"
	"io"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/tonimelisma/onedrive-client/internal/app"
	"github.com/tonimelisma/onedrive-client/pkg/onedrive"
)

// MockSDK implements the SDK interface for testing with context support
type MockSDK struct {
	// Core item operations
	GetDriveItemByPathFunc         func(ctx context.Context, path string) (onedrive.DriveItem, error)
	GetDriveItemChildrenByPathFunc func(ctx context.Context, path string) (onedrive.DriveItemList, error)
	GetRootDriveItemsFunc          func(ctx context.Context) (onedrive.DriveItemList, error)

	// File operations
	CreateFolderFunc         func(ctx context.Context, parentPath, folderName string) (onedrive.DriveItem, error)
	DeleteDriveItemFunc      func(ctx context.Context, path string) error
	CopyDriveItemFunc        func(ctx context.Context, sourcePath, destinationParentPath, newName string) (string, error)
	MoveDriveItemFunc        func(ctx context.Context, sourcePath, destinationParentPath string) (onedrive.DriveItem, error)
	UpdateDriveItemFunc      func(ctx context.Context, path, newName string) (onedrive.DriveItem, error)
	MonitorCopyOperationFunc func(ctx context.Context, monitorURL string) (onedrive.CopyOperationStatus, error)

	// Search operations
	SearchDriveItemsInFolderFunc func(ctx context.Context, folderPath, query string, paging onedrive.Paging) (onedrive.DriveItemList, string, error)

	// Activity operations
	GetItemActivitiesFunc func(ctx context.Context, remotePath string, paging onedrive.Paging) (onedrive.ActivityList, string, error)

	// Version operations
	GetFileVersionsFunc func(ctx context.Context, filePath string) (onedrive.DriveItemVersionList, error)

	// Epic 7 operations
	GetThumbnailsFunc     func(ctx context.Context, remotePath string) (onedrive.ThumbnailSetList, error)
	PreviewItemFunc       func(ctx context.Context, remotePath string, request onedrive.PreviewRequest) (onedrive.PreviewResponse, error)
	ListPermissionsFunc   func(ctx context.Context, remotePath string) (onedrive.PermissionList, error)
	CreateSharingLinkFunc func(ctx context.Context, path, linkType, scope string) (onedrive.SharingLink, error)
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

func (m *MockSDK) GetRootDriveItems(ctx context.Context) (onedrive.DriveItemList, error) {
	if m.GetRootDriveItemsFunc != nil {
		return m.GetRootDriveItemsFunc(ctx)
	}
	return onedrive.DriveItemList{}, nil
}

func (m *MockSDK) CreateFolder(ctx context.Context, parentPath, folderName string) (onedrive.DriveItem, error) {
	if m.CreateFolderFunc != nil {
		return m.CreateFolderFunc(ctx, parentPath, folderName)
	}
	return onedrive.DriveItem{}, nil
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
	return "", nil
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
	return onedrive.DriveItem{}, nil
}

func (m *MockSDK) MonitorCopyOperation(ctx context.Context, monitorURL string) (onedrive.CopyOperationStatus, error) {
	if m.MonitorCopyOperationFunc != nil {
		return m.MonitorCopyOperationFunc(ctx, monitorURL)
	}
	return onedrive.CopyOperationStatus{}, nil
}

func (m *MockSDK) ListPermissions(ctx context.Context, remotePath string) (onedrive.PermissionList, error) {
	if m.ListPermissionsFunc != nil {
		return m.ListPermissionsFunc(ctx, remotePath)
	}
	return onedrive.PermissionList{}, nil
}

func (m *MockSDK) CreateSharingLink(ctx context.Context, path, linkType, scope string) (onedrive.SharingLink, error) {
	if m.CreateSharingLinkFunc != nil {
		return m.CreateSharingLinkFunc(ctx, path, linkType, scope)
	}
	return onedrive.SharingLink{}, nil
}

func (m *MockSDK) SearchDriveItemsInFolder(ctx context.Context, folderPath, query string, paging onedrive.Paging) (onedrive.DriveItemList, string, error) {
	if m.SearchDriveItemsInFolderFunc != nil {
		return m.SearchDriveItemsInFolderFunc(ctx, folderPath, query, paging)
	}
	return onedrive.DriveItemList{}, "", nil
}

func (m *MockSDK) GetItemActivities(ctx context.Context, remotePath string, paging onedrive.Paging) (onedrive.ActivityList, string, error) {
	if m.GetItemActivitiesFunc != nil {
		return m.GetItemActivitiesFunc(ctx, remotePath, paging)
	}
	return onedrive.ActivityList{}, "", nil
}

func (m *MockSDK) GetFileVersions(ctx context.Context, filePath string) (onedrive.DriveItemVersionList, error) {
	if m.GetFileVersionsFunc != nil {
		return m.GetFileVersionsFunc(ctx, filePath)
	}
	return onedrive.DriveItemVersionList{}, nil
}

func (m *MockSDK) GetThumbnails(ctx context.Context, remotePath string) (onedrive.ThumbnailSetList, error) {
	if m.GetThumbnailsFunc != nil {
		return m.GetThumbnailsFunc(ctx, remotePath)
	}
	return onedrive.ThumbnailSetList{}, nil
}

func (m *MockSDK) PreviewItem(ctx context.Context, remotePath string, request onedrive.PreviewRequest) (onedrive.PreviewResponse, error) {
	if m.PreviewItemFunc != nil {
		return m.PreviewItemFunc(ctx, remotePath, request)
	}
	return onedrive.PreviewResponse{}, nil
}

// Add all the other required methods with proper context signatures
func (m *MockSDK) GetDrives(ctx context.Context) (onedrive.DriveList, error) {
	return onedrive.DriveList{}, nil
}
func (m *MockSDK) GetDefaultDrive(ctx context.Context) (onedrive.Drive, error) {
	return onedrive.Drive{}, nil
}
func (m *MockSDK) GetDriveByID(ctx context.Context, driveID string) (onedrive.Drive, error) {
	return onedrive.Drive{}, nil
}
func (m *MockSDK) GetDriveActivities(ctx context.Context, paging onedrive.Paging) (onedrive.ActivityList, string, error) {
	return onedrive.ActivityList{}, "", nil
}
func (m *MockSDK) GetMe(ctx context.Context) (onedrive.User, error) { return onedrive.User{}, nil }
func (m *MockSDK) CreateUploadSession(ctx context.Context, remotePath string) (onedrive.UploadSession, error) {
	return onedrive.UploadSession{}, nil
}
func (m *MockSDK) UploadChunk(ctx context.Context, uploadURL string, startByte, endByte, totalSize int64, chunkData io.Reader) (onedrive.UploadSession, error) {
	return onedrive.UploadSession{}, nil
}
func (m *MockSDK) GetUploadSessionStatus(ctx context.Context, uploadURL string) (onedrive.UploadSession, error) {
	return onedrive.UploadSession{}, nil
}
func (m *MockSDK) CancelUploadSession(ctx context.Context, uploadURL string) error { return nil }
func (m *MockSDK) UploadFile(ctx context.Context, localPath, remotePath string) (onedrive.DriveItem, error) {
	return onedrive.DriveItem{}, nil
}
func (m *MockSDK) DownloadFile(ctx context.Context, remotePath, localPath string) error { return nil }
func (m *MockSDK) DownloadFileAsFormat(ctx context.Context, remotePath, localPath, format string) error {
	return nil
}
func (m *MockSDK) DownloadFileChunk(ctx context.Context, url string, startByte, endByte int64) (io.ReadCloser, error) {
	return nil, nil
}
func (m *MockSDK) SearchDriveItems(ctx context.Context, query string) (onedrive.DriveItemList, error) {
	return onedrive.DriveItemList{}, nil
}
func (m *MockSDK) SearchDriveItemsWithPaging(ctx context.Context, query string, paging onedrive.Paging) (onedrive.DriveItemList, string, error) {
	return onedrive.DriveItemList{}, "", nil
}
func (m *MockSDK) GetSharedWithMe(ctx context.Context) (onedrive.DriveItemList, error) {
	return onedrive.DriveItemList{}, nil
}
func (m *MockSDK) GetRecentItems(ctx context.Context) (onedrive.DriveItemList, error) {
	return onedrive.DriveItemList{}, nil
}
func (m *MockSDK) GetSpecialFolder(ctx context.Context, folderName string) (onedrive.DriveItem, error) {
	return onedrive.DriveItem{}, nil
}
func (m *MockSDK) GetDelta(ctx context.Context, deltaToken string) (onedrive.DeltaResponse, error) {
	return onedrive.DeltaResponse{}, nil
}
func (m *MockSDK) GetThumbnailBySize(ctx context.Context, remotePath, thumbID, size string) (onedrive.Thumbnail, error) {
	return onedrive.Thumbnail{}, nil
}
func (m *MockSDK) InviteUsers(ctx context.Context, remotePath string, request onedrive.InviteRequest) (onedrive.InviteResponse, error) {
	return onedrive.InviteResponse{}, nil
}
func (m *MockSDK) GetPermission(ctx context.Context, remotePath, permissionID string) (onedrive.Permission, error) {
	return onedrive.Permission{}, nil
}
func (m *MockSDK) UpdatePermission(ctx context.Context, remotePath, permissionID string, request onedrive.UpdatePermissionRequest) (onedrive.Permission, error) {
	return onedrive.Permission{}, nil
}
func (m *MockSDK) DeletePermission(ctx context.Context, remotePath, permissionID string) error {
	return nil
}

func newTestApp(mockSDK *MockSDK) *app.App {
	return &app.App{
		SDK: mockSDK,
	}
}

func TestFilesListLogic(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Int("top", 0, "")
	cmd.Flags().Bool("all", false, "")
	cmd.Flags().String("next", "", "")

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "list root",
			args:    []string{},
			wantErr: false,
		},
		{
			name:    "list specific path",
			args:    []string{"/Documents"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSDK := &MockSDK{
				GetRootDriveItemsFunc: func(ctx context.Context) (onedrive.DriveItemList, error) {
					return onedrive.DriveItemList{}, nil
				},
				GetDriveItemChildrenByPathFunc: func(ctx context.Context, path string) (onedrive.DriveItemList, error) {
					return onedrive.DriveItemList{}, nil
				},
			}
			a := newTestApp(mockSDK)

			err := filesListLogic(a, cmd, tt.args)
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
		GetDriveItemByPathFunc: func(ctx context.Context, path string) (onedrive.DriveItem, error) {
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
			name: "search without folder scope (should error)",
			args: []string{"test query"},
			mockFunc: func() (*MockSDK, error) {
				return &MockSDK{}, nil
			},
			wantErr: true,
		},
		{
			name:        "search in folder",
			args:        []string{"test query"},
			folderScope: "/Documents",
			mockFunc: func() (*MockSDK, error) {
				return &MockSDK{
					SearchDriveItemsInFolderFunc: func(ctx context.Context, folderPath, query string, paging onedrive.Paging) (onedrive.DriveItemList, string, error) {
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

// TestFilesRecentLogic moved to drives_test.go since recent items are now drive-level
// TestFilesSpecialLogic moved to drives_test.go since special folders are now drive-level

func TestFilesVersionsLogic(t *testing.T) {
	mockSDK := &MockSDK{
		GetFileVersionsFunc: func(ctx context.Context, filePath string) (onedrive.DriveItemVersionList, error) {
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

	err := filesVersionsLogic(a, &cobra.Command{}, "/test-file.txt")
	assert.NoError(t, err)
}

func TestActivitiesLogic(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Int("top", 0, "")
	cmd.Flags().Bool("all", false, "")
	cmd.Flags().String("next", "", "")

	mockSDK := &MockSDK{
		GetItemActivitiesFunc: func(ctx context.Context, remotePath string, paging onedrive.Paging) (onedrive.ActivityList, string, error) {
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
				GetThumbnailsFunc: func(ctx context.Context, remotePath string) (onedrive.ThumbnailSetList, error) {
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
				PreviewItemFunc: func(ctx context.Context, remotePath string, request onedrive.PreviewRequest) (onedrive.PreviewResponse, error) {
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
