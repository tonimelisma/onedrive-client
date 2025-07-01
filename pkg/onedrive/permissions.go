package onedrive

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// CreateSharingLink creates a sharing link for a file or folder.
func (c *Client) CreateSharingLink(path, linkType, scope string) (SharingLink, error) {
	var link SharingLink

	url := BuildPathURL(path) + ":/createLink"

	createLinkRequest := map[string]string{
		"type":  linkType,
		"scope": scope,
	}

	jsonBody, err := json.Marshal(createLinkRequest)
	if err != nil {
		return link, fmt.Errorf("marshalling sharing link request: %w", err)
	}

	res, err := c.apiCall("POST", url, "application/json", strings.NewReader(string(jsonBody)))
	if err != nil {
		return link, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&link); err != nil {
		return link, fmt.Errorf("decoding sharing link response: %v", err)
	}

	return link, nil
}

// InviteUsers invites users to access a drive item.
func (c *Client) InviteUsers(remotePath string, request InviteRequest) (InviteResponse, error) {
	var invite InviteResponse
	item, err := c.GetDriveItemByPath(remotePath)
	if err != nil {
		return invite, fmt.Errorf("getting drive item: %w", err)
	}

	url := customRootURL + "me/drive/items/" + url.PathEscape(item.ID) + "/invite"
	requestBody, err := json.Marshal(request)
	if err != nil {
		return invite, fmt.Errorf("marshaling invite request: %v", err)
	}

	res, err := c.apiCall("POST", url, "application/json", bytes.NewReader(requestBody))
	if err != nil {
		return invite, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&invite); err != nil {
		return invite, fmt.Errorf("decoding invite response: %v", err)
	}

	return invite, nil
}

// ListPermissions lists all permissions on a drive item.
func (c *Client) ListPermissions(remotePath string) (PermissionList, error) {
	var permissions PermissionList
	item, err := c.GetDriveItemByPath(remotePath)
	if err != nil {
		return permissions, fmt.Errorf("getting drive item: %w", err)
	}

	url := customRootURL + "me/drive/items/" + url.PathEscape(item.ID) + "/permissions"
	res, err := c.apiCall("GET", url, "", nil)
	if err != nil {
		return permissions, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&permissions); err != nil {
		return permissions, fmt.Errorf("decoding permissions response: %v", err)
	}

	return permissions, nil
}

// GetPermission gets a specific permission by ID.
func (c *Client) GetPermission(remotePath, permissionID string) (Permission, error) {
	var permission Permission
	item, err := c.GetDriveItemByPath(remotePath)
	if err != nil {
		return permission, fmt.Errorf("getting drive item: %w", err)
	}

	url := customRootURL + "me/drive/items/" + url.PathEscape(item.ID) + "/permissions/" + url.PathEscape(permissionID)
	res, err := c.apiCall("GET", url, "", nil)
	if err != nil {
		return permission, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&permission); err != nil {
		return permission, fmt.Errorf("decoding permission response: %v", err)
	}

	return permission, nil
}

// UpdatePermission updates a specific permission.
func (c *Client) UpdatePermission(remotePath, permissionID string, request UpdatePermissionRequest) (Permission, error) {
	var permission Permission
	item, err := c.GetDriveItemByPath(remotePath)
	if err != nil {
		return permission, fmt.Errorf("getting drive item: %w", err)
	}

	url := customRootURL + "me/drive/items/" + url.PathEscape(item.ID) + "/permissions/" + url.PathEscape(permissionID)
	requestBody, err := json.Marshal(request)
	if err != nil {
		return permission, fmt.Errorf("marshaling update permission request: %v", err)
	}

	res, err := c.apiCall("PATCH", url, "application/json", bytes.NewReader(requestBody))
	if err != nil {
		return permission, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&permission); err != nil {
		return permission, fmt.Errorf("decoding permission response: %v", err)
	}

	return permission, nil
}

// DeletePermission removes a specific permission.
func (c *Client) DeletePermission(remotePath, permissionID string) error {
	item, err := c.GetDriveItemByPath(remotePath)
	if err != nil {
		return fmt.Errorf("getting drive item: %w", err)
	}

	url := customRootURL + "me/drive/items/" + url.PathEscape(item.ID) + "/permissions/" + url.PathEscape(permissionID)
	res, err := c.apiCall("DELETE", url, "", nil)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != 204 {
		return fmt.Errorf("delete permission failed with status: %s", res.Status)
	}

	return nil
}
