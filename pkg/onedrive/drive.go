package onedrive

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// GetDrives retrieves a list of all available drives for the user.
func (c *Client) GetDrives(ctx context.Context) (DriveList, error) {
	var drives DriveList

	url := customRootURL + "me/drives"
	res, err := c.apiCall(ctx, "GET", url, "", nil)
	if err != nil {
		return drives, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&drives); err != nil {
		return drives, fmt.Errorf("decoding drives failed: %v", err)
	}

	return drives, nil
}

// GetDefaultDrive retrieves information about the user's default drive.
func (c *Client) GetDefaultDrive(ctx context.Context) (Drive, error) {
	var drive Drive

	url := customRootURL + "me/drive"
	res, err := c.apiCall(ctx, "GET", url, "", nil)
	if err != nil {
		return drive, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&drive); err != nil {
		return drive, fmt.Errorf("decoding drive failed: %v", err)
	}

	return drive, nil
}

// GetDriveByID retrieves a specific drive by its ID.
func (c *Client) GetDriveByID(ctx context.Context, driveID string) (Drive, error) {
	var drive Drive

	url := customRootURL + "drives/" + url.PathEscape(driveID)
	res, err := c.apiCall(ctx, "GET", url, "", nil)
	if err != nil {
		return drive, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&drive); err != nil {
		return drive, fmt.Errorf("decoding drive failed: %v", err)
	}

	return drive, nil
}

// GetDriveActivities retrieves activities for the entire drive.
func (c *Client) GetDriveActivities(ctx context.Context, paging Paging) (ActivityList, string, error) {
	var activities ActivityList

	url := customRootURL + "me/drive/activities"
	if paging.Top > 0 {
		url += fmt.Sprintf("?$top=%d", paging.Top)
	}

	rawItems, nextLink, err := c.collectAllPages(ctx, url, paging)
	if err != nil {
		return activities, "", fmt.Errorf("decoding activities: %w", err)
	}

	for _, rawItem := range rawItems {
		var activity Activity
		if err := json.Unmarshal(rawItem, &activity); err != nil {
			return activities, "", fmt.Errorf("unmarshaling activity: %w", err)
		}
		activities.Value = append(activities.Value, activity)
	}

	return activities, nextLink, nil
}

// GetRootDriveItems retrieves items in the root of the default drive.
func (c *Client) GetRootDriveItems(ctx context.Context) (DriveItemList, error) {
	var items DriveItemList

	url := customRootURL + "me/drive/root/children"
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
