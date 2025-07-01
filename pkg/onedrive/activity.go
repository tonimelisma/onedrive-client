package onedrive

import (
	"context"
	"encoding/json"
	"fmt"
)

// GetItemActivities retrieves activities for a specific item with pagination support.
func (c *Client) GetItemActivities(ctx context.Context, remotePath string, paging Paging) (ActivityList, string, error) {
	var activities ActivityList

	item, err := c.GetDriveItemByPath(ctx, remotePath)
	if err != nil {
		return activities, "", fmt.Errorf("getting item: %w", err)
	}

	initialURL := customRootURL + "me/drive/items/" + item.ID + "/activities"
	if paging.Top > 0 {
		initialURL += fmt.Sprintf("?$top=%d", paging.Top)
	}

	rawItems, nextLink, err := c.collectAllPages(ctx, initialURL, paging)
	if err != nil {
		return activities, "", fmt.Errorf("decoding activity: %w", err)
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
