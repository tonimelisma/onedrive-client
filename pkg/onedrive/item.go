package onedrive

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// GetDriveItemByPath retrieves the metadata for a single drive item by its path.
func (c *Client) GetDriveItemByPath(ctx context.Context, path string) (DriveItem, error) {
	var item DriveItem

	url := BuildPathURL(path)
	res, err := c.apiCall(ctx, "GET", url, "", nil)
	if err != nil {
		return item, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&item); err != nil {
		return item, fmt.Errorf("decoding item failed: %v", err)
	}

	return item, nil
}

// GetDriveItemChildrenByPath retrieves the items in a specific folder by its path.
func (c *Client) GetDriveItemChildrenByPath(ctx context.Context, path string) (DriveItemList, error) {
	var items DriveItemList

	var url string
	if path == "" || path == "/" {
		url = customRootURL + "me/drive/root/children"
	} else {
		url = BuildPathURL(path) + ":/children"
	}

	res, err := c.apiCall(ctx, "GET", url, "", nil)
	if err != nil {
		return items, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&items); err != nil {
		return items, fmt.Errorf("decoding item list failed: %v", err)
	}

	return items, nil
}

// CreateFolder creates a new folder in the specified parent path.
func (c *Client) CreateFolder(ctx context.Context, parentPath string, folderName string) (DriveItem, error) {
	var item DriveItem

	// Prepare the request body
	createFolderRequest := struct {
		Name   string   `json:"name"`
		Folder struct{} `json:"folder"`
	}{
		Name:   folderName,
		Folder: struct{}{},
	}

	data, err := json.Marshal(createFolderRequest)
	if err != nil {
		return item, fmt.Errorf("marshalling create folder request: %w", err)
	}

	var url string
	if parentPath == "" || parentPath == "/" {
		url = customRootURL + "me/drive/root/children"
	} else {
		url = BuildPathURL(parentPath) + ":/children"
	}
	res, err := c.apiCall(ctx, "POST", url, "application/json", bytes.NewReader(data))
	if err != nil {
		return item, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&item); err != nil {
		return item, fmt.Errorf("decoding item failed: %v", err)
	}

	return item, nil
}

// UploadFile uploads a local file to the specified remote path.
func (c *Client) UploadFile(ctx context.Context, localPath, remotePath string) (DriveItem, error) {
	var item DriveItem

	file, err := os.Open(localPath)
	if err != nil {
		return item, fmt.Errorf("opening local file: %w", err)
	}
	defer file.Close()

	url := BuildPathURL(remotePath) + ":/content"
	res, err := c.apiCall(ctx, "PUT", url, "application/octet-stream", file)
	if err != nil {
		return item, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&item); err != nil {
		return item, fmt.Errorf("decoding item failed: %v", err)
	}

	return item, nil
}

// DeleteDriveItem moves an item to the recycle bin.
func (c *Client) DeleteDriveItem(ctx context.Context, path string) error {
	url := BuildPathURL(path)
	res, err := c.apiCall(ctx, "DELETE", url, "", nil)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusNoContent && res.StatusCode != http.StatusOK {
		return fmt.Errorf("delete failed: %s", res.Status)
	}
	return nil
}

// CopyDriveItem copies an item asynchronously and returns the monitor URL.
func (c *Client) CopyDriveItem(ctx context.Context, sourcePath, destinationParentPath, newName string) (string, error) {
	item, err := c.GetDriveItemByPath(ctx, sourcePath)
	if err != nil {
		return "", err
	}

	// Build the request body
	copyRequest := struct {
		ParentReference struct {
			Path string `json:"path"`
		} `json:"parentReference"`
		Name string `json:"name,omitempty"`
	}{
		ParentReference: struct {
			Path string `json:"path"`
		}{
			Path: fmt.Sprintf("/drive/root:%s", strings.TrimSuffix(destinationParentPath, "/")),
		},
	}

	if newName != "" {
		copyRequest.Name = newName
	}

	bodyBytes, err := json.Marshal(copyRequest)
	if err != nil {
		return "", fmt.Errorf("marshal copy body: %w", err)
	}

	url := customRootURL + "me/drive/items/" + url.PathEscape(item.ID) + "/copy"
	res, err := c.apiCall(ctx, "POST", url, "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusAccepted {
		return "", fmt.Errorf("copy did not start: %s", res.Status)
	}

	// Return the monitor URL from the Location header
	return res.Header.Get("Location"), nil
}

// MonitorCopyOperation polls the monitor URL returned by CopyDriveItem.
func (c *Client) MonitorCopyOperation(ctx context.Context, monitorURL string) (CopyOperationStatus, error) {
	var status CopyOperationStatus

	// Use a simple HTTP client for the monitor URL since it's already authenticated
	req, err := http.NewRequestWithContext(ctx, "GET", monitorURL, nil)
	if err != nil {
		return status, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return status, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&status); err != nil {
		return status, fmt.Errorf("decoding copy status failed: %v", err)
	}

	return status, nil
}

// MoveDriveItem relocates an item to a new parent path.
func (c *Client) MoveDriveItem(ctx context.Context, sourcePath, destinationParentPath string) (DriveItem, error) {
	var item DriveItem
	srcItem, err := c.GetDriveItemByPath(ctx, sourcePath)
	if err != nil {
		return item, err
	}

	// Build the request body for PATCH
	moveRequest := struct {
		ParentReference struct {
			Path string `json:"path"`
		} `json:"parentReference"`
	}{
		ParentReference: struct {
			Path string `json:"path"`
		}{
			Path: fmt.Sprintf("/drive/root:%s", strings.TrimSuffix(destinationParentPath, "/")),
		},
	}

	bodyBytes, err := json.Marshal(moveRequest)
	if err != nil {
		return item, err
	}

	url := customRootURL + "me/drive/items/" + url.PathEscape(srcItem.ID)
	res, err := c.apiCall(ctx, "PATCH", url, "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return item, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&item); err != nil {
		return item, fmt.Errorf("decoding moved item: %v", err)
	}
	return item, nil
}

// UpdateDriveItem renames an item.
func (c *Client) UpdateDriveItem(ctx context.Context, path, newName string) (DriveItem, error) {
	var item DriveItem
	srcItem, err := c.GetDriveItemByPath(ctx, path)
	if err != nil {
		return item, err
	}

	// Build the request body for PATCH
	updateRequest := struct {
		Name string `json:"name"`
	}{
		Name: newName,
	}

	bodyBytes, err := json.Marshal(updateRequest)
	if err != nil {
		return item, err
	}

	url := customRootURL + "me/drive/items/" + url.PathEscape(srcItem.ID)
	res, err := c.apiCall(ctx, "PATCH", url, "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return item, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&item); err != nil {
		return item, fmt.Errorf("decoding updated item: %v", err)
	}
	return item, nil
}
