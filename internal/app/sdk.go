package app

import (
	"io"

	"github.com/tonimelisma/onedrive-client/pkg/onedrive"
)

// SDK defines the interface for interacting with the OneDrive API.
// This allows for mocking in tests.
type SDK interface {
	GetDriveItemByPath(path string) (onedrive.DriveItem, error)
	GetDriveItemChildrenByPath(path string) (onedrive.DriveItemList, error)
	GetDrives() (onedrive.DriveList, error)
	GetDefaultDrive() (onedrive.Drive, error)
	GetMe() (onedrive.User, error)
	CreateFolder(parentPath string, folderName string) (onedrive.DriveItem, error)
	DownloadFile(remotePath, localPath string) error
	DownloadFileAsFormat(remotePath, localPath, format string) error
	DownloadFileChunk(url string, startByte, endByte int64) (io.ReadCloser, error)
	CreateUploadSession(remotePath string) (onedrive.UploadSession, error)
	UploadChunk(uploadURL string, startByte, endByte, totalSize int64, chunkData io.Reader) (onedrive.UploadSession, error)
	GetUploadSessionStatus(uploadURL string) (onedrive.UploadSession, error)
	CancelUploadSession(uploadURL string) error
	UploadFile(localPath, remotePath string) (onedrive.DriveItem, error)
	GetRootDriveItems() (onedrive.DriveItemList, error)
	DeleteDriveItem(path string) error
	CopyDriveItem(sourcePath, destinationParentPath, newName string) (string, error)
	MoveDriveItem(sourcePath, destinationParentPath string) (onedrive.DriveItem, error)
	UpdateDriveItem(path, newName string) (onedrive.DriveItem, error)
	MonitorCopyOperation(monitorURL string) (onedrive.CopyOperationStatus, error)
	SearchDriveItems(query string) (onedrive.DriveItemList, error)
	SearchDriveItemsWithPaging(query string, paging onedrive.Paging) (onedrive.DriveItemList, string, error)
	SearchDriveItemsInFolder(folderPath, query string, paging onedrive.Paging) (onedrive.DriveItemList, string, error)
	GetDriveActivities(paging onedrive.Paging) (onedrive.ActivityList, string, error)
	GetItemActivities(remotePath string, paging onedrive.Paging) (onedrive.ActivityList, string, error)
	GetSharedWithMe() (onedrive.DriveItemList, error)
	GetRecentItems() (onedrive.DriveItemList, error)
	GetSpecialFolder(folderName string) (onedrive.DriveItem, error)
	CreateSharingLink(path, linkType, scope string) (onedrive.SharingLink, error)
	GetDelta(deltaToken string) (onedrive.DeltaResponse, error)
	GetDriveByID(driveID string) (onedrive.Drive, error)
	GetFileVersions(filePath string) (onedrive.DriveItemVersionList, error)
	// New Epic 7 methods for thumbnails, preview, and permissions
	GetThumbnails(remotePath string) (onedrive.ThumbnailSetList, error)
	GetThumbnailBySize(remotePath, thumbID, size string) (onedrive.Thumbnail, error)
	PreviewItem(remotePath string, request onedrive.PreviewRequest) (onedrive.PreviewResponse, error)
	InviteUsers(remotePath string, request onedrive.InviteRequest) (onedrive.InviteResponse, error)
	ListPermissions(remotePath string) (onedrive.PermissionList, error)
	GetPermission(remotePath, permissionID string) (onedrive.Permission, error)
	UpdatePermission(remotePath, permissionID string, request onedrive.UpdatePermissionRequest) (onedrive.Permission, error)
	DeletePermission(remotePath, permissionID string) error
}
