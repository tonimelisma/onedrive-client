// Package onedrive (activity.go) provides methods for retrieving activity feeds
// related to items within a OneDrive drive. This allows tracking changes and
// actions performed on specific files or folders.
package onedrive

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// GetItemActivities retrieves a list of activities performed on a specific drive item
// (file or folder), identified by its `remotePath`.
// This is distinct from GetDriveActivities, which retrieves activities for the entire drive.
// This method supports pagination via the `paging` parameter.
//
// Example:
//
//	// Get first page of activities for a specific file
//	activities, nextLink, err := client.GetItemActivities(context.Background(), "/Documents/Report.docx", onedrive.Paging{})
//	if err != nil { log.Fatal(err) }
//	for _, activity := range activities.Value {
//	    fmt.Printf("Activity on Report.docx: Action by %s at %s\n", activity.Actor.User.DisplayName, activity.Times.RecordedTime)
//	}
//	if nextLink != "" {
//	    fmt.Println("More activities available for this item, next link:", nextLink)
//	}
func (c *Client) GetItemActivities(ctx context.Context, remotePath string, paging Paging) (ActivityList, string, error) {
	c.logger.Debugf("GetItemActivities called for remotePath: '%s', paging: %+v", remotePath, paging)
	var activities ActivityList

	// First, resolve the remotePath to a DriveItem to get its ID.
	// The activities endpoint requires an item ID.
	item, err := c.GetDriveItemByPath(ctx, remotePath)
	if err != nil {
		return activities, "", fmt.Errorf("getting item ID for path '%s' to fetch activities: %w", remotePath, err)
	}

	// Construct the base URL for item activities.
	baseURL := customRootURL + "me/drive/items/" + item.ID + "/activities"
	var queryParams []string
	if paging.Top > 0 {
		queryParams = append(queryParams, fmt.Sprintf("$top=%d", paging.Top))
	}
	// Other potential query parameters for activities (e.g., $filter) could be added here if supported.

	initialURL := baseURL
	if len(queryParams) > 0 {
		initialURL += "?" + strings.Join(queryParams, "&")
	}
	c.logger.Debug("GetItemActivities initialURL: ", initialURL)

	// Use collectAllPages helper for pagination.
	rawItems, nextLink, err := c.collectAllPages(ctx, initialURL, paging)
	if err != nil {
		return activities, "", fmt.Errorf("collecting item activities pages for path '%s': %w", remotePath, err)
	}

	// Unmarshal each raw JSON message into an Activity struct.
	for _, rawItem := range rawItems {
		var activity Activity
		if err := json.Unmarshal(rawItem, &activity); err != nil {
			c.logger.Debugf("Error unmarshaling individual activity for item '%s': %v, raw: %s", remotePath, err, string(rawItem))
			return activities, "", fmt.Errorf("unmarshaling activity item for path '%s': %w", remotePath, err)
		}
		activities.Value = append(activities.Value, activity)
	}
	activities.NextLink = nextLink // Populate NextLink in the returned ActivityList

	return activities, nextLink, nil
}
