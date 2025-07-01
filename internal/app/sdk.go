// Package app (sdk.go) defines the SDK interface used by the application core.
// This interface abstracts the concrete implementation of the OneDrive SDK (from pkg/onedrive),
// primarily to facilitate mocking and testing of the application logic without
// making actual network calls to the Microsoft Graph API.
//
// The SDK interface lists all methods from the pkg/onedrive.Client that are
// utilized by the application's command layer (cmd/) via the App struct.
package app

import (
	"context"
	"io"

	"github.com/tonimelisma/onedrive-client/pkg/onedrive"
)

// SDK defines the interface for interacting with the OneDrive API.
// It mirrors the methods exposed by `pkg/onedrive.Client`, allowing the `App` struct
// to use this interface type. This abstraction is key for:
//   - Decoupling: The `cmd` and `internal/app` layers depend on this interface,
//     not the concrete `pkg/onedrive.Client`.
//   - Testability: Mock implementations of this interface can be injected during
//     unit tests of commands and application logic, simulating various API responses
//     and error conditions without actual network calls.
type SDK interface {
	// User and Drive Information
	GetMe(ctx context.Context) (onedrive.User, error)
	GetDrives(ctx context.Context) (onedrive.DriveList, error)
	GetDefaultDrive(ctx context.Context) (onedrive.Drive, error)
	GetDriveByID(ctx context.Context, driveID string) (onedrive.Drive, error)

	// Item Metadata and Listing
	GetDriveItemByPath(ctx context.Context, path string) (onedrive.DriveItem, error)
	GetDriveItemChildrenByPath(ctx context.Context, path string) (onedrive.DriveItemList, error)
	GetRootDriveItems(ctx context.Context) (onedrive.DriveItemList, error) // Lists children of the default drive's root.

	// File and Folder Management (CRUD)
	CreateFolder(ctx context.Context, parentPath string, folderName string) (onedrive.DriveItem, error)
	DeleteDriveItem(ctx context.Context, path string) error
	CopyDriveItem(ctx context.Context, sourcePath, destinationParentPath, newName string) (string, error) // Returns monitor URL.
	MoveDriveItem(ctx context.Context, sourcePath, destinationParentPath string) (onedrive.DriveItem, error)
	UpdateDriveItem(ctx context.Context, path, newName string) (onedrive.DriveItem, error) // Primarily for renaming.
	MonitorCopyOperation(ctx context.Context, monitorURL string) (onedrive.CopyOperationStatus, error)

	// Upload Operations
	UploadFile(ctx context.Context, localPath, remotePath string) (onedrive.DriveItem, error) // Simple upload for small files.
	CreateUploadSession(ctx context.Context, remotePath string) (onedrive.UploadSession, error)
	UploadChunk(ctx context.Context, uploadURL string, startByte, endByte, totalSize int64, chunkData io.Reader) (onedrive.UploadSession, error)
	GetUploadSessionStatus(ctx context.Context, uploadURL string) (onedrive.UploadSession, error)
	CancelUploadSession(ctx context.Context, uploadURL string) error

	// Download Operations
	DownloadFile(ctx context.Context, remotePath, localPath string) error
	DownloadFileAsFormat(ctx context.Context, remotePath, localPath, format string) error
	DownloadFileChunk(ctx context.Context, url string, startByte, endByte int64) (io.ReadCloser, error)

	// Search
	SearchDriveItems(ctx context.Context, query string) (onedrive.DriveItemList, error) // Deprecated in favor of paginated version.
	SearchDriveItemsWithPaging(ctx context.Context, query string, paging onedrive.Paging) (onedrive.DriveItemList, string, error)
	SearchDriveItemsInFolder(ctx context.Context, folderPath, query string, paging onedrive.Paging) (onedrive.DriveItemList, string, error)

	// Activity and Versioning
	GetDriveActivities(ctx context.Context, paging onedrive.Paging) (onedrive.ActivityList, string, error) // Drive-level activities.
	GetItemActivities(ctx context.Context, remotePath string, paging onedrive.Paging) (onedrive.ActivityList, string, error) // Item-specific activities.
	GetFileVersions(ctx context.Context, filePath string) (onedrive.DriveItemVersionList, error)
	GetDelta(ctx context.Context, deltaToken string) (onedrive.DeltaResponse, error) // For tracking changes.

	// Sharing and Special Views
	GetSharedWithMe(ctx context.Context) (onedrive.DriveItemList, error)
	GetRecentItems(ctx context.Context) (onedrive.DriveItemList, error)
	GetSpecialFolder(ctx context.Context, folderName string) (onedrive.DriveItem, error)

	// Sharing Links and Permissions
	CreateSharingLink(ctx context.Context, path, linkType, scope string) (onedrive.SharingLink, error)
	InviteUsers(ctx context.Context, remotePath string, request onedrive.InviteRequest) (onedrive.InviteResponse, error)
	ListPermissions(ctx context.Context, remotePath string) (onedrive.PermissionList, error)
	GetPermission(ctx context.Context, remotePath, permissionID string) (onedrive.Permission, error)
	UpdatePermission(ctx context.Context, remotePath, permissionID string, request onedrive.UpdatePermissionRequest) (onedrive.Permission, error)
	DeletePermission(ctx context.Context, remotePath, permissionID string) error

	// Thumbnails and Previews
	GetThumbnails(ctx context.Context, remotePath string) (onedrive.ThumbnailSetList, error)
	GetThumbnailBySize(ctx context.Context, remotePath, thumbID, size string) (onedrive.Thumbnail, error)
	PreviewItem(ctx context.Context, remotePath string, request onedrive.PreviewRequest) (onedrive.PreviewResponse, error)
}
