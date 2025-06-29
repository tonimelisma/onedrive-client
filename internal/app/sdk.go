package app

import (
	"io"
	"net/http"

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

// OneDriveSDK implements the SDK interface by calling the real OneDrive API.
type OneDriveSDK struct {
	client *http.Client
}

// NewOneDriveSDK creates a new LiveSDK instance.
func NewOneDriveSDK(client *http.Client) SDK {
	return &OneDriveSDK{client: client}
}

// GetDriveItemByPath calls the real onedrive.GetDriveItemByPath function.
func (s *OneDriveSDK) GetDriveItemByPath(path string) (onedrive.DriveItem, error) {
	return onedrive.GetDriveItemByPath(s.client, path)
}

// GetDriveItemChildrenByPath calls the real onedrive.GetDriveItemChildrenByPath function.
func (s *OneDriveSDK) GetDriveItemChildrenByPath(path string) (onedrive.DriveItemList, error) {
	return onedrive.GetDriveItemChildrenByPath(s.client, path)
}

// GetDrives calls the real onedrive.GetDrives function.
func (s *OneDriveSDK) GetDrives() (onedrive.DriveList, error) {
	return onedrive.GetDrives(s.client)
}

// GetDefaultDrive calls the real onedrive.GetDefaultDrive function.
func (s *OneDriveSDK) GetDefaultDrive() (onedrive.Drive, error) {
	return onedrive.GetDefaultDrive(s.client)
}

// GetMe calls the real onedrive.GetMe function.
func (s *OneDriveSDK) GetMe() (onedrive.User, error) {
	return onedrive.GetMe(s.client)
}

// CreateFolder calls the real onedrive.CreateFolder function.
func (s *OneDriveSDK) CreateFolder(parentPath string, folderName string) (onedrive.DriveItem, error) {
	return onedrive.CreateFolder(s.client, parentPath, folderName)
}

// DownloadFile calls the real onedrive.DownloadFile function.
func (s *OneDriveSDK) DownloadFile(remotePath, localPath string) error {
	return onedrive.DownloadFile(s.client, remotePath, localPath)
}

// DownloadFileAsFormat downloads a file in a specific format
func (sdk *OneDriveSDK) DownloadFileAsFormat(remotePath, localPath, format string) error {
	return onedrive.DownloadFileAsFormat(sdk.client, remotePath, localPath, format)
}

// DownloadFileChunk calls the real onedrive.DownloadFileChunk function.
func (s *OneDriveSDK) DownloadFileChunk(url string, startByte, endByte int64) (io.ReadCloser, error) {
	return onedrive.DownloadFileChunk(s.client, url, startByte, endByte)
}

// CreateUploadSession calls the real onedrive.CreateUploadSession function.
func (s *OneDriveSDK) CreateUploadSession(remotePath string) (onedrive.UploadSession, error) {
	return onedrive.CreateUploadSession(s.client, remotePath)
}

// UploadChunk calls the real onedrive.UploadChunk function.
func (s *OneDriveSDK) UploadChunk(uploadURL string, startByte, endByte, totalSize int64, chunkData io.Reader) (onedrive.UploadSession, error) {
	return onedrive.UploadChunk(uploadURL, startByte, endByte, totalSize, chunkData)
}

// GetUploadSessionStatus calls the real onedrive.GetUploadSessionStatus function.
func (s *OneDriveSDK) GetUploadSessionStatus(uploadURL string) (onedrive.UploadSession, error) {
	return onedrive.GetUploadSessionStatus(uploadURL)
}

// CancelUploadSession calls the real onedrive.CancelUploadSession function.
func (s *OneDriveSDK) CancelUploadSession(uploadURL string) error {
	return onedrive.CancelUploadSession(uploadURL)
}

// UploadFile calls the real onedrive.UploadFile function.
func (s *OneDriveSDK) UploadFile(localPath, remotePath string) (onedrive.DriveItem, error) {
	return onedrive.UploadFile(s.client, localPath, remotePath)
}

// GetRootDriveItems calls the real onedrive.GetRootDriveItems function.
func (s *OneDriveSDK) GetRootDriveItems() (onedrive.DriveItemList, error) {
	return onedrive.GetRootDriveItems(s.client)
}

// DeleteDriveItem calls the real onedrive.DeleteDriveItem function.
func (s *OneDriveSDK) DeleteDriveItem(path string) error {
	return onedrive.DeleteDriveItem(s.client, path)
}

// CopyDriveItem calls the real onedrive.CopyDriveItem function.
func (s *OneDriveSDK) CopyDriveItem(sourcePath, destinationParentPath, newName string) (string, error) {
	return onedrive.CopyDriveItem(s.client, sourcePath, destinationParentPath, newName)
}

// MoveDriveItem calls the real onedrive.MoveDriveItem function.
func (s *OneDriveSDK) MoveDriveItem(sourcePath, destinationParentPath string) (onedrive.DriveItem, error) {
	return onedrive.MoveDriveItem(s.client, sourcePath, destinationParentPath)
}

// UpdateDriveItem calls the real onedrive.UpdateDriveItem function.
func (s *OneDriveSDK) UpdateDriveItem(path, newName string) (onedrive.DriveItem, error) {
	return onedrive.UpdateDriveItem(s.client, path, newName)
}

// MonitorCopyOperation calls the real onedrive.MonitorCopyOperation function.
func (s *OneDriveSDK) MonitorCopyOperation(monitorURL string) (onedrive.CopyOperationStatus, error) {
	return onedrive.MonitorCopyOperation(s.client, monitorURL)
}

// SearchDriveItems calls the real onedrive.SearchDriveItems function.
func (s *OneDriveSDK) SearchDriveItems(query string) (onedrive.DriveItemList, error) {
	return onedrive.SearchDriveItems(s.client, query)
}

// SearchDriveItemsWithPaging searches for items with paging support
func (sdk *OneDriveSDK) SearchDriveItemsWithPaging(query string, paging onedrive.Paging) (onedrive.DriveItemList, string, error) {
	return onedrive.SearchDriveItemsWithPaging(sdk.client, query, paging)
}

// SearchDriveItemsInFolder searches for items within a specific folder
func (sdk *OneDriveSDK) SearchDriveItemsInFolder(folderPath, query string, paging onedrive.Paging) (onedrive.DriveItemList, string, error) {
	return onedrive.SearchDriveItemsInFolder(sdk.client, folderPath, query, paging)
}

// GetDriveActivities retrieves activities for the entire drive
func (sdk *OneDriveSDK) GetDriveActivities(paging onedrive.Paging) (onedrive.ActivityList, string, error) {
	return onedrive.GetDriveActivities(sdk.client, paging)
}

// GetItemActivities retrieves activities for a specific item
func (sdk *OneDriveSDK) GetItemActivities(remotePath string, paging onedrive.Paging) (onedrive.ActivityList, string, error) {
	return onedrive.GetItemActivities(sdk.client, remotePath, paging)
}

// GetSharedWithMe calls the real onedrive.GetSharedWithMe function.
func (s *OneDriveSDK) GetSharedWithMe() (onedrive.DriveItemList, error) {
	return onedrive.GetSharedWithMe(s.client)
}

// GetRecentItems calls the real onedrive.GetRecentItems function.
func (s *OneDriveSDK) GetRecentItems() (onedrive.DriveItemList, error) {
	return onedrive.GetRecentItems(s.client)
}

// GetSpecialFolder calls the real onedrive.GetSpecialFolder function.
func (s *OneDriveSDK) GetSpecialFolder(folderName string) (onedrive.DriveItem, error) {
	return onedrive.GetSpecialFolder(s.client, folderName)
}

// CreateSharingLink calls the real onedrive.CreateSharingLink function.
func (s *OneDriveSDK) CreateSharingLink(path, linkType, scope string) (onedrive.SharingLink, error) {
	return onedrive.CreateSharingLink(s.client, path, linkType, scope)
}

// GetDelta calls the real onedrive.GetDelta function.
func (s *OneDriveSDK) GetDelta(deltaToken string) (onedrive.DeltaResponse, error) {
	return onedrive.GetDelta(s.client, deltaToken)
}

// GetDriveByID calls the real onedrive.GetDriveByID function.
func (s *OneDriveSDK) GetDriveByID(driveID string) (onedrive.Drive, error) {
	return onedrive.GetDriveByID(s.client, driveID)
}

// GetFileVersions calls the real onedrive.GetFileVersions function.
func (s *OneDriveSDK) GetFileVersions(filePath string) (onedrive.DriveItemVersionList, error) {
	return onedrive.GetFileVersions(s.client, filePath)
}

// GetThumbnails calls the real onedrive.GetThumbnails function.
func (s *OneDriveSDK) GetThumbnails(remotePath string) (onedrive.ThumbnailSetList, error) {
	return onedrive.GetThumbnails(s.client, remotePath)
}

// GetThumbnailBySize calls the real onedrive.GetThumbnailBySize function.
func (s *OneDriveSDK) GetThumbnailBySize(remotePath, thumbID, size string) (onedrive.Thumbnail, error) {
	return onedrive.GetThumbnailBySize(s.client, remotePath, thumbID, size)
}

// PreviewItem calls the real onedrive.PreviewItem function.
func (s *OneDriveSDK) PreviewItem(remotePath string, request onedrive.PreviewRequest) (onedrive.PreviewResponse, error) {
	return onedrive.PreviewItem(s.client, remotePath, request)
}

// InviteUsers calls the real onedrive.InviteUsers function.
func (s *OneDriveSDK) InviteUsers(remotePath string, request onedrive.InviteRequest) (onedrive.InviteResponse, error) {
	return onedrive.InviteUsers(s.client, remotePath, request)
}

// ListPermissions calls the real onedrive.ListPermissions function.
func (s *OneDriveSDK) ListPermissions(remotePath string) (onedrive.PermissionList, error) {
	return onedrive.ListPermissions(s.client, remotePath)
}

// GetPermission calls the real onedrive.GetPermission function.
func (s *OneDriveSDK) GetPermission(remotePath, permissionID string) (onedrive.Permission, error) {
	return onedrive.GetPermission(s.client, remotePath, permissionID)
}

// UpdatePermission calls the real onedrive.UpdatePermission function.
func (s *OneDriveSDK) UpdatePermission(remotePath, permissionID string, request onedrive.UpdatePermissionRequest) (onedrive.Permission, error) {
	return onedrive.UpdatePermission(s.client, remotePath, permissionID, request)
}

// DeletePermission calls the real onedrive.DeletePermission function.
func (s *OneDriveSDK) DeletePermission(remotePath, permissionID string) error {
	return onedrive.DeletePermission(s.client, remotePath, permissionID)
}
