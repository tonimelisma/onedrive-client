// Package onedrive (permissions.go) provides methods for managing sharing links
// and permissions on DriveItems within a OneDrive drive. This includes creating
// sharing links, inviting users, and listing, getting, updating, or deleting permissions.
package onedrive

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// CreateSharingLink creates a sharing link for a specified DriveItem.
// `path` is the path to the DriveItem.
// `linkType` determines the type of link, e.g., "view" (read-only), "edit" (read-write), or "embed".
// `scope` defines who can use the link, e.g., "anonymous" or "organization".
//
// Example:
//
//	link, err := client.CreateSharingLink(context.Background(), "/Documents/MyFile.docx", "view", "anonymous")
//	if err != nil { log.Fatal(err) }
//	fmt.Printf("Sharing link created: %s\n", link.Link.WebUrl)
func (c *Client) CreateSharingLink(ctx context.Context, path, linkType, scope string) (SharingLink, error) {
	c.logger.Debugf("CreateSharingLink called for path: '%s', type: '%s', scope: '%s'", path, linkType, scope)
	var link SharingLink

	// Prepare the request body for creating a sharing link.
	requestBody := CreateLinkRequest{
		Type:  linkType,
		Scope: scope,
		// Password and ExpirationDateTime can also be set here if needed.
	}

	data, err := json.Marshal(requestBody)
	if err != nil {
		return link, fmt.Errorf("marshaling CreateSharingLink request for path '%s': %w", path, err)
	}

	// The createLink action is performed on the DriveItem's path.
	url := BuildPathURL(path) + ":/createLink"
	res, err := c.apiCall(ctx, "POST", url, "application/json", bytes.NewReader(data))
	if err != nil {
		return link, err
	}
	defer closeBodySafely(res.Body, c.logger, "create sharing link")

	if err := json.NewDecoder(res.Body).Decode(&link); err != nil {
		return link, fmt.Errorf("%w: decoding sharing link response for path '%s': %w", ErrDecodingFailed, path, err)
	}

	return link, nil
}

// InviteUsers invites users to access a DriveItem with specified roles and options.
// `remotePath` is the path to the DriveItem.
// `request` is an InviteRequest struct containing recipient emails, roles (e.g., "read", "write"),
// and other invitation options like message, sign-in requirement, and whether to send an email.
//
// Returns an InviteResponse containing the permissions created for the invited users.
//
// Example:
//
//	inviteReq := onedrive.InviteRequest{
//	    Recipients: []struct{ Email string `json:"email"`}{ {Email: "user@example.com"} },
//	    Roles:      []string{"read"},
//	    SendInvitation: true,
//	    Message:    "Check out this document!",
//	}
//	inviteResp, err := client.InviteUsers(context.Background(), "/Documents/Collaboration.docx", inviteReq)
//	if err != nil { log.Fatal(err) }
//	for _, perm := range inviteResp.Value {
//	    fmt.Printf("Permission created for ID: %s with roles: %v\n", perm.ID, perm.Roles)
//	}
func (c *Client) InviteUsers(ctx context.Context, remotePath string, request InviteRequest) (InviteResponse, error) {
	c.logger.Debugf("InviteUsers called for remotePath: '%s', request: %+v", remotePath, request)
	var invite InviteResponse

	// First, get the DriveItem to resolve its ID from the path.
	// The invite action is performed on the item's ID.
	item, err := c.GetDriveItemByPath(ctx, remotePath)
	if err != nil {
		return invite, fmt.Errorf("getting DriveItem ID for path '%s' to invite users: %w", remotePath, err)
	}

	data, err := json.Marshal(request)
	if err != nil {
		return invite, fmt.Errorf("marshaling InviteRequest for path '%s': %w", remotePath, err)
	}

	// The invite action is performed on the item's ID.
	url := customRootURL + "me/drive/items/" + url.PathEscape(item.ID) + "/invite"
	res, err := c.apiCall(ctx, "POST", url, "application/json", bytes.NewReader(data))
	if err != nil {
		return invite, err
	}
	defer closeBodySafely(res.Body, c.logger, "invite users")

	if err := json.NewDecoder(res.Body).Decode(&invite); err != nil {
		return invite, fmt.Errorf("%w: decoding invite response for path '%s': %w", ErrDecodingFailed, remotePath, err)
	}

	return invite, nil
}

// ListPermissions retrieves all permissions currently set on a DriveItem.
// `remotePath` is the path to the DriveItem.
//
// Example:
//
//	permissions, err := client.ListPermissions(context.Background(), "/SharedFolder")
//	if err != nil { log.Fatal(err) }
//	for _, p := range permissions.Value {
//	    fmt.Printf("Permission ID: %s, Roles: %v\n", p.ID, p.Roles)
//	    if p.Link != nil { fmt.Printf("  Link URL: %s\n", p.Link.WebURL) }
//	    if p.GrantedToV2 != nil && p.GrantedToV2.User != nil { fmt.Printf("  Granted to User: %s\n", p.GrantedToV2.User.DisplayName) }
//	}
func (c *Client) ListPermissions(ctx context.Context, remotePath string) (PermissionList, error) {
	c.logger.Debugf("ListPermissions called for remotePath: '%s'", remotePath)
	var permissions PermissionList

	// Use helper to get item and build URL, reducing duplication.
	apiURL, err := c.getItemAndBuildURL(ctx, remotePath, "/permissions")
	if err != nil {
		return permissions, fmt.Errorf("building permissions URL for path '%s': %w", remotePath, err)
	}

	// Use helper for API call and decode, with proper error handling.
	err = c.makeAPICallAndDecode(ctx, "GET", apiURL, "", nil, &permissions, "list permissions")
	if err != nil {
		return permissions, err
	}

	return permissions, nil
}

// GetPermission retrieves details for a specific permission on a DriveItem by its ID.
// `remotePath` is the path to the DriveItem.
// `permissionID` is the ID of the permission to retrieve.
//
// Example:
//
//	permID := "aRandomPermissionIdString"
//	permission, err := client.GetPermission(context.Background(), "/Documents/File.docx", permID)
//	if err != nil { log.Fatal(err) }
//	fmt.Printf("Permission details for ID %s: Roles %v\n", permission.ID, permission.Roles)
func (c *Client) GetPermission(ctx context.Context, remotePath, permissionID string) (Permission, error) {
	c.logger.Debugf("GetPermission called for remotePath: '%s', permissionID: '%s'", remotePath, permissionID)
	var permission Permission

	// Get DriveItem ID.
	item, err := c.GetDriveItemByPath(ctx, remotePath)
	if err != nil {
		return permission, fmt.Errorf("getting DriveItem ID for path '%s' to get permission '%s': %w", remotePath, permissionID, err)
	}

	// Endpoint for a specific permission.
	url := customRootURL + "me/drive/items/" + url.PathEscape(item.ID) + "/permissions/" + url.PathEscape(permissionID)
	res, err := c.apiCall(ctx, "GET", url, "", nil)
	if err != nil {
		return permission, err
	}
	defer closeBodySafely(res.Body, c.logger, "get permission")

	if err := json.NewDecoder(res.Body).Decode(&permission); err != nil {
		return permission, fmt.Errorf("%w: decoding permission details for ID '%s' on path '%s': %w", ErrDecodingFailed, permissionID, remotePath, err)
	}

	return permission, nil
}

// UpdatePermission modifies an existing permission on a DriveItem.
// `remotePath` is the path to the DriveItem.
// `permissionID` is the ID of the permission to update.
// `request` is an UpdatePermissionRequest struct containing the fields to modify (e.g., roles).
//
// Example:
//
//	updateReq := onedrive.UpdatePermissionRequest{Roles: []string{"write"}}
//	updatedPerm, err := client.UpdatePermission(context.Background(), "/Documents/File.docx", existingPermID, updateReq)
//	if err != nil { log.Fatal(err) }
//	fmt.Printf("Permission %s updated. New roles: %v\n", updatedPerm.ID, updatedPerm.Roles)
func (c *Client) UpdatePermission(ctx context.Context, remotePath, permissionID string, request UpdatePermissionRequest) (Permission, error) {
	c.logger.Debugf("UpdatePermission called for remotePath: '%s', permissionID: '%s', request: %+v", remotePath, permissionID, request)
	var permission Permission

	// Get DriveItem ID.
	item, err := c.GetDriveItemByPath(ctx, remotePath)
	if err != nil {
		return permission, fmt.Errorf("getting DriveItem ID for path '%s' to update permission '%s': %w", remotePath, permissionID, err)
	}

	// Endpoint for updating a specific permission.
	url := customRootURL + "me/drive/items/" + url.PathEscape(item.ID) + "/permissions/" + url.PathEscape(permissionID)
	requestBody, err := json.Marshal(request)
	if err != nil {
		return permission, fmt.Errorf("marshaling UpdatePermissionRequest for permission '%s' on path '%s': %w", permissionID, remotePath, err)
	}

	res, err := c.apiCall(ctx, "PATCH", url, "application/json", bytes.NewReader(requestBody))
	if err != nil {
		return permission, err
	}
	defer closeBodySafely(res.Body, c.logger, "update permission")

	if err := json.NewDecoder(res.Body).Decode(&permission); err != nil {
		// It's possible that on success, some APIs return 204 No Content or a minimal body.
		// For PATCH, a 200 OK with the updated resource is common.
		if res.StatusCode == http.StatusOK || res.StatusCode == http.StatusNoContent { // StatusNoContent might mean it was accepted but no body.
			c.logger.Debugf("UpdatePermission for '%s' on path '%s' returned status %d with potentially empty body. Assuming success if no decode error or fetching again.", permissionID, remotePath, res.StatusCode)
			// Optionally, re-fetch the permission here if a full object is always desired and not returned.
			// For now, we rely on the decode or lack of error.
			if res.StatusCode == http.StatusNoContent {
				return c.GetPermission(ctx, remotePath, permissionID) // Re-fetch to get the updated state.
			}
		}
		return permission, fmt.Errorf("%w: decoding updated permission response for ID '%s' on path '%s': %w", ErrDecodingFailed, permissionID, remotePath, err)
	}

	return permission, nil
}

// DeletePermission removes a specific permission from a DriveItem.
// `remotePath` is the path to the DriveItem.
// `permissionID` is the ID of the permission to delete.
//
// Example:
//
//	err := client.DeletePermission(context.Background(), "/Documents/File.docx", permissionIDToDelete)
//	if err != nil { log.Fatal(err) }
//	fmt.Println("Permission deleted successfully.")
func (c *Client) DeletePermission(ctx context.Context, remotePath, permissionID string) error {
	c.logger.Debugf("DeletePermission called for remotePath: '%s', permissionID: '%s'", remotePath, permissionID)
	// Get DriveItem ID.
	item, err := c.GetDriveItemByPath(ctx, remotePath)
	if err != nil {
		return fmt.Errorf("getting DriveItem ID for path '%s' to delete permission '%s': %w", remotePath, permissionID, err)
	}

	// Endpoint for deleting a specific permission.
	url := customRootURL + "me/drive/items/" + url.PathEscape(item.ID) + "/permissions/" + url.PathEscape(permissionID)
	res, err := c.apiCall(ctx, "DELETE", url, "", nil)
	if err != nil {
		return err
	}
	defer closeBodySafely(res.Body, c.logger, "delete permission")

	// Successful deletion usually returns HTTP 204 No Content.
	// Some APIs might also return 200 OK (though less common for DELETE).
	if res.StatusCode != http.StatusNoContent && res.StatusCode != http.StatusOK {
		errorBody, _ := io.ReadAll(res.Body)
		return fmt.Errorf("delete permission '%s' for path '%s' failed with status %s: %s", permissionID, remotePath, res.Status, string(errorBody))
	}

	return nil
}
