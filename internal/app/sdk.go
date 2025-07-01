package app

import (
	"context"
	"io"

	"github.com/tonimelisma/onedrive-client/pkg/onedrive"
)

// SDK defines the interface for interacting with the OneDrive API.
// This allows for mocking in tests.
type SDK interface {
	GetDriveItemByPath(ctx context.Context, path string) (onedrive.DriveItem, error)
	GetDriveItemChildrenByPath(ctx context.Context, path string) (onedrive.DriveItemList, error)
	GetDrives(ctx context.Context) (onedrive.DriveList, error)
	GetDefaultDrive(ctx context.Context) (onedrive.Drive, error)
	GetMe(ctx context.Context) (onedrive.User, error)
	CreateFolder(ctx context.Context, parentPath string, folderName string) (onedrive.DriveItem, error)
	DownloadFile(ctx context.Context, remotePath, localPath string) error
	DownloadFileAsFormat(ctx context.Context, remotePath, localPath, format string) error
	DownloadFileChunk(ctx context.Context, url string, startByte, endByte int64) (io.ReadCloser, error)
	CreateUploadSession(ctx context.Context, remotePath string) (onedrive.UploadSession, error)
	UploadChunk(ctx context.Context, uploadURL string, startByte, endByte, totalSize int64, chunkData io.Reader) (onedrive.UploadSession, error)
	GetUploadSessionStatus(ctx context.Context, uploadURL string) (onedrive.UploadSession, error)
	CancelUploadSession(ctx context.Context, uploadURL string) error
	UploadFile(ctx context.Context, localPath, remotePath string) (onedrive.DriveItem, error)
	GetRootDriveItems(ctx context.Context) (onedrive.DriveItemList, error)
	DeleteDriveItem(ctx context.Context, path string) error
	CopyDriveItem(ctx context.Context, sourcePath, destinationParentPath, newName string) (string, error)
	MoveDriveItem(ctx context.Context, sourcePath, destinationParentPath string) (onedrive.DriveItem, error)
	UpdateDriveItem(ctx context.Context, path, newName string) (onedrive.DriveItem, error)
	MonitorCopyOperation(ctx context.Context, monitorURL string) (onedrive.CopyOperationStatus, error)
	SearchDriveItems(ctx context.Context, query string) (onedrive.DriveItemList, error)
	SearchDriveItemsWithPaging(ctx context.Context, query string, paging onedrive.Paging) (onedrive.DriveItemList, string, error)
	SearchDriveItemsInFolder(ctx context.Context, folderPath, query string, paging onedrive.Paging) (onedrive.DriveItemList, string, error)
	GetDriveActivities(ctx context.Context, paging onedrive.Paging) (onedrive.ActivityList, string, error)
	GetItemActivities(ctx context.Context, remotePath string, paging onedrive.Paging) (onedrive.ActivityList, string, error)
	GetSharedWithMe(ctx context.Context) (onedrive.DriveItemList, error)
	GetRecentItems(ctx context.Context) (onedrive.DriveItemList, error)
	GetSpecialFolder(ctx context.Context, folderName string) (onedrive.DriveItem, error)
	CreateSharingLink(ctx context.Context, path, linkType, scope string) (onedrive.SharingLink, error)
	GetDelta(ctx context.Context, deltaToken string) (onedrive.DeltaResponse, error)
	GetDriveByID(ctx context.Context, driveID string) (onedrive.Drive, error)
	GetFileVersions(ctx context.Context, filePath string) (onedrive.DriveItemVersionList, error)
	// New Epic 7 methods for thumbnails, preview, and permissions
	GetThumbnails(ctx context.Context, remotePath string) (onedrive.ThumbnailSetList, error)
	GetThumbnailBySize(ctx context.Context, remotePath, thumbID, size string) (onedrive.Thumbnail, error)
	PreviewItem(ctx context.Context, remotePath string, request onedrive.PreviewRequest) (onedrive.PreviewResponse, error)
	InviteUsers(ctx context.Context, remotePath string, request onedrive.InviteRequest) (onedrive.InviteResponse, error)
	ListPermissions(ctx context.Context, remotePath string) (onedrive.PermissionList, error)
	GetPermission(ctx context.Context, remotePath, permissionID string) (onedrive.Permission, error)
	UpdatePermission(ctx context.Context, remotePath, permissionID string, request onedrive.UpdatePermissionRequest) (onedrive.Permission, error)
	DeletePermission(ctx context.Context, remotePath, permissionID string) error
}
