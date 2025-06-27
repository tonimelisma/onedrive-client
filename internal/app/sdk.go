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
	CreateFolder(parentPath string, folderName string) (onedrive.DriveItem, error)
	DownloadFile(remotePath, localPath string) error
	CreateUploadSession(remotePath string) (onedrive.UploadSession, error)
	UploadChunk(uploadURL string, startByte, endByte, totalSize int64, chunkData io.Reader) (onedrive.UploadSession, error)
	GetUploadSessionStatus(uploadURL string) (onedrive.UploadSession, error)
	CancelUploadSession(uploadURL string) error
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

// CreateFolder calls the real onedrive.CreateFolder function.
func (s *OneDriveSDK) CreateFolder(parentPath string, folderName string) (onedrive.DriveItem, error) {
	return onedrive.CreateFolder(s.client, parentPath, folderName)
}

// DownloadFile calls the real onedrive.DownloadFile function.
func (s *OneDriveSDK) DownloadFile(remotePath, localPath string) error {
	return onedrive.DownloadFile(s.client, remotePath, localPath)
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
