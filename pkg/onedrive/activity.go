package onedrive

import (
	"encoding/json"
	"fmt"
)

// GetItemActivities retrieves activities for a specific item.
func (c *Client) GetItemActivities(remotePath string, paging Paging) (ActivityList, string, error) {
	var activities ActivityList
	item, err := c.GetDriveItemByPath(remotePath)
	if err != nil {
		return activities, "", fmt.Errorf("getting item: %w", err)
	}
	initialURL := fmt.Sprintf("%sme/drive/items/%s/activities", customRootURL, item.ID)
	if paging.Top > 0 {
		initialURL += fmt.Sprintf("?$top=%d", paging.Top)
	}

	rawActivities, nextLink, err := c.collectAllPages(initialURL, paging)
	if err != nil {
		return activities, "", err
	}

	for _, raw := range rawActivities {
		var activity Activity
		if err := json.Unmarshal(raw, &activity); err != nil {
			return activities, "", fmt.Errorf("decoding activity: %w", err)
		}
		activities.Value = append(activities.Value, activity)
	}

	return activities, nextLink, nil
}
