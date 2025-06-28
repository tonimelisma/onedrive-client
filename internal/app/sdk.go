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
	GetSharedWithMe() (onedrive.DriveItemList, error)
	GetRecentItems() (onedrive.DriveItemList, error)
	GetSpecialFolder(folderName string) (onedrive.DriveItem, error)
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
