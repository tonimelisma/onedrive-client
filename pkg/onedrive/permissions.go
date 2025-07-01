package onedrive

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// CreateSharingLink creates a sharing link for a drive item.
func (c *Client) CreateSharingLink(ctx context.Context, path, linkType, scope string) (SharingLink, error) {
	var link SharingLink

	// Build the request body
	requestBody := map[string]string{
		"type":  linkType,
		"scope": scope,
	}

	data, err := json.Marshal(requestBody)
	if err != nil {
		return link, fmt.Errorf("marshalling sharing link request: %w", err)
	}

	url := BuildPathURL(path) + ":/createLink"
	res, err := c.apiCall(ctx, "POST", url, "application/json", bytes.NewReader(data))
	if err != nil {
		return link, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&link); err != nil {
		return link, fmt.Errorf("decoding sharing link response: %v", err)
	}

	return link, nil
}

// InviteUsers invites users to access a drive item with specified permissions.
func (c *Client) InviteUsers(ctx context.Context, remotePath string, request InviteRequest) (InviteResponse, error) {
	var invite InviteResponse

	item, err := c.GetDriveItemByPath(ctx, remotePath)
	if err != nil {
		return invite, fmt.Errorf("getting drive item: %w", err)
	}

	data, err := json.Marshal(request)
	if err != nil {
		return invite, fmt.Errorf("marshaling invite request: %v", err)
	}

	url := customRootURL + "me/drive/items/" + url.PathEscape(item.ID) + "/invite"
	res, err := c.apiCall(ctx, "POST", url, "application/json", bytes.NewReader(data))
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
func (c *Client) ListPermissions(ctx context.Context, remotePath string) (PermissionList, error) {
	var permissions PermissionList
	item, err := c.GetDriveItemByPath(ctx, remotePath)
	if err != nil {
		return permissions, fmt.Errorf("getting drive item: %w", err)
	}

	url := customRootURL + "me/drive/items/" + url.PathEscape(item.ID) + "/permissions"
	res, err := c.apiCall(ctx, "GET", url, "", nil)
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
func (c *Client) GetPermission(ctx context.Context, remotePath, permissionID string) (Permission, error) {
	var permission Permission
	item, err := c.GetDriveItemByPath(ctx, remotePath)
	if err != nil {
		return permission, fmt.Errorf("getting drive item: %w", err)
	}

	url := customRootURL + "me/drive/items/" + url.PathEscape(item.ID) + "/permissions/" + url.PathEscape(permissionID)
	res, err := c.apiCall(ctx, "GET", url, "", nil)
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
func (c *Client) UpdatePermission(ctx context.Context, remotePath, permissionID string, request UpdatePermissionRequest) (Permission, error) {
	var permission Permission
	item, err := c.GetDriveItemByPath(ctx, remotePath)
	if err != nil {
		return permission, fmt.Errorf("getting drive item: %w", err)
	}

	url := customRootURL + "me/drive/items/" + url.PathEscape(item.ID) + "/permissions/" + url.PathEscape(permissionID)
	requestBody, err := json.Marshal(request)
	if err != nil {
		return permission, fmt.Errorf("marshaling update permission request: %v", err)
	}

	res, err := c.apiCall(ctx, "PATCH", url, "application/json", bytes.NewReader(requestBody))
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
func (c *Client) DeletePermission(ctx context.Context, remotePath, permissionID string) error {
	item, err := c.GetDriveItemByPath(ctx, remotePath)
	if err != nil {
		return fmt.Errorf("getting drive item: %w", err)
	}

	url := customRootURL + "me/drive/items/" + url.PathEscape(item.ID) + "/permissions/" + url.PathEscape(permissionID)
	res, err := c.apiCall(ctx, "DELETE", url, "", nil)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusNoContent && res.StatusCode != http.StatusOK {
		return fmt.Errorf("delete permission failed with status: %s", res.Status)
	}

	return nil
}
