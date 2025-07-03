// Package onedrive (item.go) provides methods for managing DriveItems (files and folders)
// within a OneDrive drive. This includes operations like retrieving metadata,
// listing children, creating, uploading, deleting, copying, moving, and renaming items.
package onedrive

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// GetDriveItemByPath retrieves the metadata for a single drive item (file or folder)
// by its path relative to the drive root.
//
// Example:
//
//	item, err := client.GetDriveItemByPath(context.Background(), "/Documents/MyReport.docx")
//	if err != nil { log.Fatal(err) }
//	fmt.Printf("Item Name: %s, ID: %s, Size: %d\n", item.Name, item.ID, item.Size)
func (c *Client) GetDriveItemByPath(ctx context.Context, path string) (DriveItem, error) {
	c.logger.Debug("GetDriveItemByPath called for path: ", path)
	var item DriveItem

	// BuildPathURL handles correct formatting for root or nested paths.
	url := BuildPathURL(path)
	res, err := c.apiCall(ctx, "GET", url, "", nil)
	if err != nil {
		return item, err
	}
	defer closeBodySafely(res.Body, c.logger, "get drive item by path")

	if err := json.NewDecoder(res.Body).Decode(&item); err != nil {
		return item, fmt.Errorf("%w: decoding item metadata for path '%s': %w", ErrDecodingFailed, path, err)
	}

	return item, nil
}

// GetDriveItemChildrenByPath retrieves a list of drive items (children) within a
// specific folder, identified by its path.
// If path is "" or "/", it lists children of the drive's root folder.
//
// Example:
//
//	children, err := client.GetDriveItemChildrenByPath(context.Background(), "/Documents")
//	if err != nil { log.Fatal(err) }
//	for _, child := range children.Value {
//	    fmt.Printf("Child Name: %s\n", child.Name)
//	}
func (c *Client) GetDriveItemChildrenByPath(ctx context.Context, path string) (DriveItemList, error) {
	c.logger.Debug("GetDriveItemChildrenByPath called for path: ", path)
	var items DriveItemList

	var url string
	// Determine the correct endpoint based on whether the path is root or a subfolder.
	if path == "" || path == "/" {
		// To list children of the root, target "me/drive/root/children".
		url = customRootURL + "me/drive/root/children"
	} else {
		// For subfolders, append ":/children" to the folder's path URL.
		url = BuildPathURL(path) + ":/children"
	}

	res, err := c.apiCall(ctx, "GET", url, "", nil)
	if err != nil {
		return items, err
	}
	defer closeBodySafely(res.Body, c.logger, "get drive item children")

	if err := json.NewDecoder(res.Body).Decode(&items); err != nil {
		return items, fmt.Errorf("%w: decoding children for path '%s': %w", ErrDecodingFailed, path, err)
	}

	return items, nil
}

// CreateFolder creates a new folder within a specified parent path.
// `parentPath` is the path to the directory where the new folder will be created.
// `folderName` is the name of the new folder.
//
// Example:
//
//	newFolder, err := client.CreateFolder(context.Background(), "/Documents", "New Project Folder")
//	if err != nil { log.Fatal(err) }
//	fmt.Printf("Created folder '%s' with ID: %s\n", newFolder.Name, newFolder.ID)
func (c *Client) CreateFolder(ctx context.Context, parentPath string, folderName string) (DriveItem, error) {
	c.logger.Debugf("CreateFolder called for parentPath: '%s', folderName: '%s'", parentPath, folderName)
	var item DriveItem

	// Prepare the request body for creating a folder.
	// It requires a name and an empty "folder" facet.
	createFolderRequest := struct {
		Name   string   `json:"name"`
		Folder struct{} `json:"folder"`
		// Optionally, conflict behavior can be specified here, e.g.,
		// ConflictBehavior string `json:"@microsoft.graph.conflictBehavior,omitempty"` // "rename", "replace", or "fail"
	}{
		Name:   folderName,
		Folder: struct{}{}, // Indicates that this is a folder.
	}

	data, err := json.Marshal(createFolderRequest)
	if err != nil {
		return item, fmt.Errorf("marshaling create folder request for '%s': %w", folderName, err)
	}

	var url string
	// Determine the target URL for creating a child item.
	if parentPath == "" || parentPath == "/" {
		// If parent is root, target "me/drive/root/children".
		url = customRootURL + "me/drive/root/children"
	} else {
		// If parent is a subfolder, target "<parent_path_url>:/children".
		url = BuildPathURL(parentPath) + ":/children"
	}

	res, err := c.apiCall(ctx, "POST", url, "application/json", bytes.NewReader(data))
	if err != nil {
		return item, err
	}
	defer closeBodySafely(res.Body, c.logger, "create folder")

	if err := json.NewDecoder(res.Body).Decode(&item); err != nil {
		return item, fmt.Errorf("%w: decoding created folder response for '%s': %w", ErrDecodingFailed, folderName, err)
	}

	return item, nil
}

// UploadFile uploads a local file to the specified remote path in OneDrive.
// This method is suitable for smaller files (typically under 4MB) as it performs a simple
// PUT upload. For larger files, use the resumable upload methods (CreateUploadSession, UploadChunk).
// `localPath` is the path to the file on the local filesystem.
// `remotePath` is the full path (including filename) where the file will be stored in OneDrive.
//
// Example:
//
//	uploadedItem, err := client.UploadFile(context.Background(), "./localfile.txt", "/Documents/remoteFileName.txt")
//	if err != nil { log.Fatal(err) }
//	fmt.Printf("Uploaded file '%s' with ID: %s\n", uploadedItem.Name, uploadedItem.ID)
func (c *Client) UploadFile(ctx context.Context, localPath, remotePath string) (DriveItem, error) {
	c.logger.Debugf("UploadFile called for localPath: '%s', remotePath: '%s'", localPath, remotePath)
	var item DriveItem

	file, err := os.Open(localPath)
	if err != nil {
		return item, fmt.Errorf("opening local file '%s': %w", localPath, err)
	}
	defer file.Close()

	// The target URL for content upload is "<item_path_url>:/content".
	url := BuildPathURL(remotePath) + ":/content"
	// Content-Type for raw file upload.
	res, err := c.apiCall(ctx, "PUT", url, "application/octet-stream", file)
	if err != nil {
		return item, err
	}
	defer closeBodySafely(res.Body, c.logger, "upload file")

	if err := json.NewDecoder(res.Body).Decode(&item); err != nil {
		return item, fmt.Errorf("%w: decoding uploaded file response for '%s': %w", ErrDecodingFailed, remotePath, err)
	}

	return item, nil
}

// DeleteDriveItem moves a drive item (file or folder) to the OneDrive recycle bin.
// It does not permanently delete the item.
//
// Example:
//
//	err := client.DeleteDriveItem(context.Background(), "/Documents/OldFile.txt")
//	if err != nil { log.Fatal(err) }
//	fmt.Println("File moved to recycle bin.")
func (c *Client) DeleteDriveItem(ctx context.Context, path string) error {
	c.logger.Debug("DeleteDriveItem called for path: ", path)
	url := BuildPathURL(path) // URL of the item to delete.
	res, err := c.apiCall(ctx, "DELETE", url, "", nil)
	if err != nil {
		return err
	}
	defer closeBodySafely(res.Body, c.logger, "delete drive item")

	// Successful deletion typically returns HTTP 204 No Content.
	// Some APIs might also return 200 OK or 202 Accepted.
	if res.StatusCode != http.StatusNoContent && res.StatusCode != http.StatusOK && res.StatusCode != http.StatusAccepted {
		return fmt.Errorf("%w: delete failed for path '%s' with status: %s", ErrOperationFailed, path, res.Status)
	}
	return nil
}

// CopyDriveItem asynchronously copies a drive item (file or folder) to a new destination.
// `sourcePath` is the path of the item to copy.
// `destinationParentPath` is the path of the folder where the item will be copied.
// `newName` (optional) specifies a new name for the copied item; if empty, the original name is used.
//
// Returns a `monitorURL` which can be polled using `MonitorCopyOperation` to track the copy progress.
// The copy operation happens server-side and might take time for large items.
//
// Example:
//
//	monitorURL, err := client.CopyDriveItem(context.Background(), "/Photos/MyImage.jpg", "/Backup/Photos", "MyImage_Copy.jpg")
//	if err != nil { log.Fatal(err) }
//	fmt.Printf("Copy operation started. Monitor URL: %s\n", monitorURL)
//	// Later, poll with client.MonitorCopyOperation(ctx, monitorURL)
func (c *Client) CopyDriveItem(ctx context.Context, sourcePath, destinationParentPath, newName string) (string, error) {
	c.logger.Debugf("CopyDriveItem called for source: '%s', destParent: '%s', newName: '%s'", sourcePath, destinationParentPath, newName)
	// First, get the item ID of the source item.
	item, err := c.GetDriveItemByPath(ctx, sourcePath)
	if err != nil {
		return "", fmt.Errorf("getting source item '%s' for copy: %w", sourcePath, err)
	}

	// Prepare the request body for the copy operation.
	// It requires a ParentReference pointing to the destination folder.
	copyRequest := struct {
		ParentReference struct {
			Path string `json:"path"` // Graph API path for the parent, e.g., "/drive/root:/Documents"
		} `json:"parentReference"`
		Name string `json:"name,omitempty"` // Optional new name for the copy.
	}{
		ParentReference: struct {
			Path string `json:"path"`
		}{
			// The ParentReference path needs to be in the format "/drive/root:<path_to_parent_folder_from_root>"
			// or by providing driveId and itemId if copying across drives.
			// Assuming same drive for now.
			Path: fmt.Sprintf("/drive/root:%s", strings.TrimSuffix(destinationParentPath, "/")),
		},
	}

	if newName != "" {
		copyRequest.Name = newName
	}

	bodyBytes, err := json.Marshal(copyRequest)
	if err != nil {
		return "", fmt.Errorf("marshaling copy request for '%s': %w", sourcePath, err)
	}

	// The copy endpoint is on the source item: "/items/{source-item-id}/copy".
	url := customRootURL + "me/drive/items/" + url.PathEscape(item.ID) + "/copy"
	res, err := c.apiCall(ctx, "POST", url, "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	defer closeBodySafely(res.Body, c.logger, "copy drive item")

	// A successful initiation of an async copy returns HTTP 202 Accepted.
	if res.StatusCode != http.StatusAccepted {
		// Attempt to read error body for more details
		errorBody, _ := io.ReadAll(res.Body)
		return "", fmt.Errorf("copy operation for '%s' did not start: status %s, body: %s", sourcePath, res.Status, string(errorBody))
	}

	// The monitor URL is returned in the Location header.
	monitorURL := res.Header.Get("Location")
	if monitorURL == "" {
		return "", fmt.Errorf("copy operation for '%s' started but no monitor URL was returned", sourcePath)
	}
	return monitorURL, nil
}

// MonitorCopyOperation polls a monitor URL (obtained from `CopyDriveItem`) to get
// the status of an asynchronous copy operation.
//
// Example:
//
//	status, err := client.MonitorCopyOperation(context.Background(), monitorURL)
//	if err != nil { log.Fatal(err) }
//	fmt.Printf("Copy status: %s, Progress: %d%%\n", status.Status, status.PercentageComplete)
//	if status.Status == "completed" { fmt.Println("Copy finished!") }
func (c *Client) MonitorCopyOperation(ctx context.Context, monitorURL string) (CopyOperationStatus, error) {
	c.logger.Debug("MonitorCopyOperation called for URL: ", monitorURL)
	var status CopyOperationStatus

	// The monitor URL is a pre-authenticated URL and should be called directly
	// without the SDK's standard auth headers or retry logic.
	// Thus, use a simple http.Client.
	req, err := http.NewRequestWithContext(ctx, "GET", monitorURL, nil)
	if err != nil {
		return status, fmt.Errorf("creating request for monitor URL '%s': %w", monitorURL, err)
	}

	// It's important to use a client that does not automatically add Authorization headers,
	// as monitor URLs are typically pre-signed and expect no additional auth.
	// http.DefaultClient is suitable here.
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return status, fmt.Errorf("calling monitor URL '%s': %w", monitorURL, err)
	}
	defer closeBodySafely(res.Body, c.logger, "monitor copy operation")

	if res.StatusCode != http.StatusOK {
		errorBody, _ := io.ReadAll(res.Body)
		return status, fmt.Errorf("monitoring copy operation at '%s' failed with status %s: %s", monitorURL, res.Status, string(errorBody))
	}

	if err := json.NewDecoder(res.Body).Decode(&status); err != nil {
		return status, fmt.Errorf("%w: decoding copy status from '%s': %w", ErrDecodingFailed, monitorURL, err)
	}

	return status, nil
}

// MoveDriveItem relocates a drive item (file or folder) to a new parent path.
// This is equivalent to a "rename" if the new parent path is the same as the old one
// but the item's name changes as part of the ParentReference.Name field (not shown here).
// This implementation focuses on changing the parent.
//
// Example:
//
//	movedItem, err := client.MoveDriveItem(context.Background(), "/Temporary/File.txt", "/Documents/Archive")
//	if err != nil { log.Fatal(err) }
//	fmt.Printf("Moved item '%s' to new location. New ID: %s\n", movedItem.Name, movedItem.ID)
func (c *Client) MoveDriveItem(ctx context.Context, sourcePath, destinationParentPath string) (DriveItem, error) {
	c.logger.Debugf("MoveDriveItem called for source: '%s', destParent: '%s'", sourcePath, destinationParentPath)
	var item DriveItem
	// Get the ID of the source item.
	srcItem, err := c.GetDriveItemByPath(ctx, sourcePath)
	if err != nil {
		return item, fmt.Errorf("getting source item '%s' for move: %w", sourcePath, err)
	}

	// Prepare the request body for the PATCH operation to update the parentReference.
	moveRequest := struct {
		ParentReference struct {
			Path string `json:"path"` // Graph API path for the new parent.
		} `json:"parentReference"`
		// To also rename during move, add: Name string `json:"name,omitempty"`
	}{
		ParentReference: struct {
			Path string `json:"path"`
		}{
			// Path format: "/drive/root:<path_to_new_parent_from_root>"
			Path: fmt.Sprintf("/drive/root:%s", strings.TrimSuffix(destinationParentPath, "/")),
		},
	}

	bodyBytes, err := json.Marshal(moveRequest)
	if err != nil {
		return item, fmt.Errorf("marshaling move request for '%s': %w", sourcePath, err)
	}

	// The PATCH request is made to the source item's URL.
	url := customRootURL + "me/drive/items/" + url.PathEscape(srcItem.ID)
	res, err := c.apiCall(ctx, "PATCH", url, "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return item, err
	}
	defer closeBodySafely(res.Body, c.logger, "move drive item")

	if err := json.NewDecoder(res.Body).Decode(&item); err != nil {
		return item, fmt.Errorf("%w: decoding moved item response for '%s': %w", ErrDecodingFailed, sourcePath, err)
	}
	return item, nil
}

// UpdateDriveItem renames a drive item (file or folder).
// `path` is the current path of the item.
// `newName` is the desired new name for the item.
//
// Example:
//
//	updatedItem, err := client.UpdateDriveItem(context.Background(), "/Documents/OldName.txt", "NewName.txt")
//	if err != nil { log.Fatal(err) }
//	fmt.Printf("Renamed item to '%s'. ID: %s\n", updatedItem.Name, updatedItem.ID)
func (c *Client) UpdateDriveItem(ctx context.Context, path, newName string) (DriveItem, error) {
	c.logger.Debugf("UpdateDriveItem called for path: '%s', newName: '%s'", path, newName)
	var item DriveItem
	// Get the ID of the source item.
	srcItem, err := c.GetDriveItemByPath(ctx, path)
	if err != nil {
		return item, fmt.Errorf("getting item '%s' for rename: %w", path, err)
	}

	// Prepare the request body for PATCH to update the name.
	updateRequest := struct {
		Name string `json:"name"`
	}{
		Name: newName,
	}

	bodyBytes, err := json.Marshal(updateRequest)
	if err != nil {
		return item, fmt.Errorf("marshaling rename request for '%s': %w", path, err)
	}

	// The PATCH request is made to the item's URL.
	url := customRootURL + "me/drive/items/" + url.PathEscape(srcItem.ID)
	res, err := c.apiCall(ctx, "PATCH", url, "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return item, err
	}
	defer closeBodySafely(res.Body, c.logger, "update drive item")

	if err := json.NewDecoder(res.Body).Decode(&item); err != nil {
		return item, fmt.Errorf("%w: decoding renamed item response for '%s': %w", ErrDecodingFailed, path, err)
	}
	return item, nil
}
