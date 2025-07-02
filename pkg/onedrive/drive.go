// Package onedrive (drive.go) provides methods for managing and inspecting OneDrive drives.
// This includes listing available drives, getting information about specific drives
// (like quota), and retrieving drive-level activities.
package onedrive

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// GetDrives retrieves a list of all Drive resources available to the signed-in user.
// This can include the user's personal OneDrive, OneDrive for Business drives,
// and SharePoint document libraries they have access to.
//
// Example:
//
//	drives, err := client.GetDrives(context.Background())
//	if err != nil { log.Fatal(err) }
//	for _, d := range drives.Value {
//	    fmt.Printf("Drive Name: %s, ID: %s, Type: %s\n", d.Name, d.ID, d.DriveType)
//	}
func (c *Client) GetDrives(ctx context.Context) (DriveList, error) {
	c.logger.Debug("GetDrives called")
	var drives DriveList

	// Endpoint to list all drives accessible by the user.
	url := customRootURL + "me/drives"
	res, err := c.apiCall(ctx, "GET", url, "", nil)
	if err != nil {
		return drives, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&drives); err != nil {
		return drives, fmt.Errorf("%w: decoding drives list: %w", ErrDecodingFailed, err)
	}

	return drives, nil
}

// GetDefaultDrive retrieves information about the user's default OneDrive drive.
// This is typically the user's personal OneDrive or their primary OneDrive for Business drive.
// The returned Drive object includes details like quota information.
//
// Example:
//
//	defaultDrive, err := client.GetDefaultDrive(context.Background())
//	if err != nil { log.Fatal(err) }
//	fmt.Printf("Default Drive ID: %s, Total Quota: %d bytes\n", defaultDrive.ID, defaultDrive.Quota.Total)
func (c *Client) GetDefaultDrive(ctx context.Context) (Drive, error) {
	c.logger.Debug("GetDefaultDrive called")
	var drive Drive

	// Endpoint for the user's default drive.
	url := customRootURL + "me/drive"
	res, err := c.apiCall(ctx, "GET", url, "", nil)
	if err != nil {
		return drive, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&drive); err != nil {
		return drive, fmt.Errorf("%w: decoding default drive info: %w", ErrDecodingFailed, err)
	}

	return drive, nil
}

// GetDriveByID retrieves metadata for a specific Drive resource using its unique ID.
// This is useful when you know the ID of a drive (e.g., from GetDrives or a shared link)
// and want to get more details about it.
//
// Example:
//
//	driveID := "b!abCDeFgHiJkLmNoPqRsTuVwXyZ" // Example Drive ID
//	specificDrive, err := client.GetDriveByID(context.Background(), driveID)
//	if err != nil { log.Fatal(err) }
//	fmt.Printf("Drive Name: %s, Owner: %s\n", specificDrive.Name, specificDrive.Owner.User.DisplayName)
func (c *Client) GetDriveByID(ctx context.Context, driveID string) (Drive, error) {
	c.logger.Debug("GetDriveByID called for ID: ", driveID)
	var drive Drive

	// Endpoint to get a drive by its ID. The driveID needs to be URL-escaped.
	url := customRootURL + "drives/" + url.PathEscape(driveID)
	res, err := c.apiCall(ctx, "GET", url, "", nil)
	if err != nil {
		return drive, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&drive); err != nil {
		return drive, fmt.Errorf("%w: decoding drive info for ID '%s': %w", ErrDecodingFailed, driveID, err)
	}

	return drive, nil
}

// GetDriveActivities retrieves a list of activities that have occurred on the user's default drive.
// Activities can include file creation, deletion, edits, sharing, etc.
// This method supports pagination via the `paging` parameter.
//
// Example:
//
//	// Get the first page of activities (Graph default limit, usually 200)
//	activities, nextLink, err := client.GetDriveActivities(context.Background(), onedrive.Paging{})
//	if err != nil { log.Fatal(err) }
//	for _, activity := range activities.Value {
//	    fmt.Printf("Activity Time: %s, Actor: %s\n", activity.Times.RecordedTime, activity.Actor.User.DisplayName)
//	}
//	if nextLink != "" {
//	    fmt.Println("More activities available, next link:", nextLink)
//	    // To get next page: client.GetDriveActivities(ctx, onedrive.Paging{NextLink: nextLink})
//	}
func (c *Client) GetDriveActivities(ctx context.Context, paging Paging) (ActivityList, string, error) {
	c.logger.Debug("GetDriveActivities called with paging: ", paging)
	var activities ActivityList

	// Base URL for drive activities.
	baseURL := customRootURL + "me/drive/activities"
	var queryParams []string
	if paging.Top > 0 {
		queryParams = append(queryParams, fmt.Sprintf("$top=%d", paging.Top))
	}
	// Note: The 'token' parameter for delta-like activity fetching is typically handled by NextLink.
	// If this endpoint supports a direct 'token' param for delta, it would be added here.

	initialURL := baseURL
	if len(queryParams) > 0 {
		initialURL += "?" + strings.Join(queryParams, "&")
	}
	c.logger.Debug("GetDriveActivities initialURL: ", initialURL)

	// Use collectAllPages helper for pagination.
	rawItems, nextLink, err := c.collectAllPages(ctx, initialURL, paging)
	if err != nil {
		return activities, "", fmt.Errorf("collecting drive activities pages: %w", err)
	}

	// Unmarshal each raw JSON message into an Activity struct.
	for _, rawItem := range rawItems {
		var activity Activity
		if err := json.Unmarshal(rawItem, &activity); err != nil {
			// Log or collect errors if individual items fail to unmarshal.
			// For simplicity, returning the first error encountered.
			c.logger.Debugf("Error unmarshaling individual activity: %v, raw: %s", err, string(rawItem))
			return activities, "", fmt.Errorf("unmarshaling activity item: %w", err)
		}
		activities.Value = append(activities.Value, activity)
	}
	activities.NextLink = nextLink // Populate NextLink in the returned ActivityList

	return activities, nextLink, nil
}

// GetRootDriveItems retrieves a list of drive items (files and folders) in the root
// of the user's default OneDrive drive.
// This is equivalent to calling GetDriveItemChildrenByPath with path "/".
//
// Example:
//
//	rootItems, err := client.GetRootDriveItems(context.Background())
//	if err != nil { log.Fatal(err) }
//	for _, item := range rootItems.Value {
//	    fmt.Printf("Root item: %s\n", item.Name)
//	}
func (c *Client) GetRootDriveItems(ctx context.Context) (DriveItemList, error) {
	c.logger.Debug("GetRootDriveItems called")
	var items DriveItemList

	// Endpoint for children of the default drive's root.
	url := customRootURL + "me/drive/root/children"
	res, err := c.apiCall(ctx, "GET", url, "", nil)
	if err != nil {
		return items, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&items); err != nil {
		return items, fmt.Errorf("%w: decoding root drive items list: %w", ErrDecodingFailed, err)
	}

	return items, nil
}
