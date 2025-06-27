package app

import (
	"net/http"

	"github.com/tonimelisma/onedrive-client/pkg/onedrive"
)

// SDK defines the interface for interacting with the OneDrive API.
// This allows for mocking in tests.
type SDK interface {
	GetDriveItemByPath(client *http.Client, path string) (onedrive.DriveItem, error)
	GetDriveItemChildrenByPath(client *http.Client, path string) (onedrive.DriveItemList, error)
	CreateFolder(client *http.Client, parentPath string, folderName string) (onedrive.DriveItem, error)
	UploadFile(client *http.Client, localPath, remotePath string) (onedrive.DriveItem, error)
	DownloadFile(client *http.Client, remotePath, localPath string) error
}

// LiveSDK is the concrete implementation of the SDK interface that makes real API calls.
type LiveSDK struct{}

// GetDriveItemByPath calls the real onedrive.GetDriveItemByPath function.
func (s *LiveSDK) GetDriveItemByPath(client *http.Client, path string) (onedrive.DriveItem, error) {
	return onedrive.GetDriveItemByPath(client, path)
}

// GetDriveItemChildrenByPath calls the real onedrive.GetDriveItemChildrenByPath function.
func (s *LiveSDK) GetDriveItemChildrenByPath(client *http.Client, path string) (onedrive.DriveItemList, error) {
	return onedrive.GetDriveItemChildrenByPath(client, path)
}

// CreateFolder calls the real onedrive.CreateFolder function.
func (s *LiveSDK) CreateFolder(client *http.Client, parentPath string, folderName string) (onedrive.DriveItem, error) {
	return onedrive.CreateFolder(client, parentPath, folderName)
}

// UploadFile calls the real onedrive.UploadFile function.
func (s *LiveSDK) UploadFile(client *http.Client, localPath, remotePath string) (onedrive.DriveItem, error) {
	return onedrive.UploadFile(client, localPath, remotePath)
}

// DownloadFile calls the real onedrive.DownloadFile function.
func (s *LiveSDK) DownloadFile(client *http.Client, remotePath, localPath string) error {
	return onedrive.DownloadFile(client, remotePath, localPath)
}
