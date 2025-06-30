package onedrive

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
)

// GetDriveItemByPath retrieves the metadata for a single drive item by its path.
func (c *Client) GetDriveItemByPath(path string) (DriveItem, error) {
	var item DriveItem

	url := BuildPathURL(path)
	res, err := c.apiCall("GET", url, "", nil)
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
func (c *Client) GetDriveItemChildrenByPath(path string) (DriveItemList, error) {
	var items DriveItemList

	url := BuildPathURL(path)
	if url == customRootURL+"me/drive/root" {
		url += "/children"
	} else {
		url += ":/children"
	}

	res, err := c.apiCall("GET", url, "", nil)
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
func (c *Client) CreateFolder(parentPath string, folderName string) (DriveItem, error) {
	var item DriveItem

	url := BuildPathURL(parentPath)
	if url == customRootURL+"me/drive/root" {
		url += "/children"
	} else {
		url += ":/children"
	}

	requestBody := map[string]interface{}{
		"name":                              folderName,
		"folder":                            map[string]interface{}{},
		"@microsoft.graph.conflictBehavior": "rename",
	}
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return item, fmt.Errorf("marshalling create folder request: %w", err)
	}

	res, err := c.apiCall("POST", url, "application/json", strings.NewReader(string(jsonBody)))
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
func (c *Client) UploadFile(localPath, remotePath string) (DriveItem, error) {
	var item DriveItem

	file, err := os.Open(localPath)
	if err != nil {
		return item, fmt.Errorf("opening local file: %w", err)
	}
	defer file.Close()

	url := BuildPathURL(remotePath) + ":/content"
	res, err := c.apiCall("PUT", url, "application/octet-stream", file)
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
func (c *Client) DeleteDriveItem(path string) error {
	url := BuildPathURL(path)
	res, err := c.apiCall("DELETE", url, "", nil)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != 204 {
		return fmt.Errorf("delete failed: %s", res.Status)
	}
	return nil
}

// CopyDriveItem copies an item asynchronously and returns the monitor URL.
func (c *Client) CopyDriveItem(sourcePath, destinationParentPath, newName string) (string, error) {
	item, err := c.GetDriveItemByPath(sourcePath)
	if err != nil {
		return "", err
	}

	reqBody := map[string]interface{}{
		"parentReference": map[string]string{
			"path": BuildPathURL(destinationParentPath),
		},
	}
	if newName != "" {
		reqBody["name"] = newName
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal copy body: %w", err)
	}

	url := customRootURL + "me/drive/items/" + url.PathEscape(item.ID) + "/copy"
	res, err := c.apiCall("POST", url, "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode != 202 {
		return "", fmt.Errorf("copy did not start: %s", res.Status)
	}

	return res.Header.Get("Location"), nil
}

// MonitorCopyOperation polls the monitor URL returned by CopyDriveItem.
func (c *Client) MonitorCopyOperation(monitorURL string) (CopyOperationStatus, error) {
	var status CopyOperationStatus

	res, err := c.apiCall("GET", monitorURL, "", nil)
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
func (c *Client) MoveDriveItem(sourcePath, destinationParentPath string) (DriveItem, error) {
	var item DriveItem
	srcItem, err := c.GetDriveItemByPath(sourcePath)
	if err != nil {
		return item, err
	}

	reqBody := map[string]interface{}{
		"parentReference": map[string]string{
			"path": BuildPathURL(destinationParentPath),
		},
	}
	bodyBytes, _ := json.Marshal(reqBody)

	url := customRootURL + "me/drive/items/" + url.PathEscape(srcItem.ID)
	res, err := c.apiCall("PATCH", url, "application/json", bytes.NewReader(bodyBytes))
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
func (c *Client) UpdateDriveItem(path, newName string) (DriveItem, error) {
	var item DriveItem
	srcItem, err := c.GetDriveItemByPath(path)
	if err != nil {
		return item, err
	}

	reqBody := map[string]string{"name": newName}
	bodyBytes, _ := json.Marshal(reqBody)

	url := customRootURL + "me/drive/items/" + url.PathEscape(srcItem.ID)
	res, err := c.apiCall("PATCH", url, "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return item, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&item); err != nil {
		return item, fmt.Errorf("decoding updated item: %v", err)
	}
	return item, nil
}
