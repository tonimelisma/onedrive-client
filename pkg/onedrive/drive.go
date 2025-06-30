package onedrive

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// GetDrives retrieves the list of available drives for the user.
func (c *Client) GetDrives() (DriveList, error) {
	c.logger.Debug("GetDrives called")
	var drives DriveList

	url := customRootURL + "me/drives"
	res, err := c.apiCall("GET", url, "", nil)
	if err != nil {
		return drives, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&drives); err != nil {
		return drives, fmt.Errorf("decoding drives failed: %v", err)
	}

	return drives, nil
}

// GetDefaultDrive retrieves the default drive for the user, including quota information.
func (c *Client) GetDefaultDrive() (Drive, error) {
	c.logger.Debug("GetDefaultDrive called")
	var drive Drive

	url := customRootURL + "me/drive"
	res, err := c.apiCall("GET", url, "", nil)
	if err != nil {
		return drive, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&drive); err != nil {
		return drive, fmt.Errorf("decoding drive failed: %v", err)
	}

	return drive, nil
}

// GetDriveByID gets metadata for a specific drive by its ID
func (c *Client) GetDriveByID(driveID string) (Drive, error) {
	c.logger.Debug("GetDriveByID called with ID: ", driveID)
	var drive Drive

	url := customRootURL + "drives/" + url.PathEscape(driveID)
	res, err := c.apiCall("GET", url, "", nil)
	if err != nil {
		return drive, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&drive); err != nil {
		return drive, fmt.Errorf("decoding drive failed: %v", err)
	}

	return drive, nil
}

// GetDriveActivities returns recent drive activities with pagination.
func (c *Client) GetDriveActivities(paging Paging) (ActivityList, string, error) {
	c.logger.Debug("GetDriveActivities called")
	var activities ActivityList

	initialURL := customRootURL + "me/drive/activities"
	if paging.Top > 0 {
		initialURL += fmt.Sprintf("?$top=%d", paging.Top)
	}

	pages, nextLink, err := c.collectAllPages(initialURL, paging)
	if err != nil {
		return activities, "", err
	}

	// Combine pages into a single ActivityList
	for _, page := range pages {
		var partial ActivityList
		if err := json.Unmarshal(page, &partial); err != nil {
			return activities, "", fmt.Errorf("decoding activities: %w", err)
		}
		activities.Value = append(activities.Value, partial.Value...)
	}

	return activities, nextLink, nil
}

// GetRootDriveItems lists items in the root of the user's default drive.
// It is equivalent to GET /me/drive/root/children and remains a first-class helper
// for callers that want a simple root listing without building a path.
func (c *Client) GetRootDriveItems() (DriveItemList, error) {
	var items DriveItemList

	res, err := c.apiCall("GET", customRootURL+"me/drive/root/children", "", nil)
	if err != nil {
		return items, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&items); err != nil {
		return items, fmt.Errorf("decoding item list failed: %v", err)
	}

	return items, nil
}
