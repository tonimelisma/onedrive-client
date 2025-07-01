package onedrive

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// SearchDriveItems searches for items across the entire drive.
func (c *Client) SearchDriveItems(ctx context.Context, query string) (DriveItemList, error) {
	var items DriveItemList

	url := customRootURL + "me/drive/root/search(q='" + url.QueryEscape(query) + "')"
	res, err := c.apiCall(ctx, "GET", url, "", nil)
	if err != nil {
		return items, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&items); err != nil {
		return items, fmt.Errorf("decoding search results failed: %v", err)
	}

	return items, nil
}

// SearchDriveItemsInFolder searches for items within a specific folder.
func (c *Client) SearchDriveItemsInFolder(ctx context.Context, folderPath, query string, paging Paging) (DriveItemList, string, error) {
	var items DriveItemList

	// First get the folder item to get its ID
	folderItem, err := c.GetDriveItemByPath(ctx, folderPath)
	if err != nil {
		return items, "", fmt.Errorf("getting folder item: %w", err)
	}

	// Build the search URL using the folder's item ID
	initialURL := customRootURL + "me/drive/items/" + folderItem.ID + "/search(q='" + url.QueryEscape(query) + "')"
	if paging.Top > 0 {
		initialURL += fmt.Sprintf("?$top=%d", paging.Top)
	}

	rawItems, nextLink, err := c.collectAllPages(ctx, initialURL, paging)
	if err != nil {
		return items, "", fmt.Errorf("decoding item: %w", err)
	}

	for _, rawItem := range rawItems {
		var item DriveItem
		if err := json.Unmarshal(rawItem, &item); err != nil {
			return items, "", fmt.Errorf("unmarshaling item: %w", err)
		}
		items.Value = append(items.Value, item)
	}

	return items, nextLink, nil
}

// SearchDriveItemsWithPaging searches for items across the entire drive with pagination support.
func (c *Client) SearchDriveItemsWithPaging(ctx context.Context, query string, paging Paging) (DriveItemList, string, error) {
	var items DriveItemList

	initialURL := customRootURL + "me/drive/root/search(q='" + url.QueryEscape(query) + "')"
	if paging.Top > 0 {
		initialURL += fmt.Sprintf("?$top=%d", paging.Top)
	}

	rawItems, nextLink, err := c.collectAllPages(ctx, initialURL, paging)
	if err != nil {
		return items, "", fmt.Errorf("decoding item: %w", err)
	}

	for _, rawItem := range rawItems {
		var item DriveItem
		if err := json.Unmarshal(rawItem, &item); err != nil {
			return items, "", fmt.Errorf("unmarshaling item: %w", err)
		}
		items.Value = append(items.Value, item)
	}

	return items, nextLink, nil
}
