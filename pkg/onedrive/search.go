package onedrive

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// SearchDriveItems searches for items in the drive by query string.
func (c *Client) SearchDriveItems(query string) (DriveItemList, error) {
	var items DriveItemList

	url := customRootURL + "me/drive/root/search(q='" + url.QueryEscape(query) + "')"
	res, err := c.apiCall("GET", url, "", nil)
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
func (c *Client) SearchDriveItemsInFolder(folderPath, query string, paging Paging) (DriveItemList, string, error) {
	var items DriveItemList

	folder, err := c.GetDriveItemByPath(folderPath)
	if err != nil {
		return items, "", fmt.Errorf("getting folder item: %w", err)
	}

	initialURL := fmt.Sprintf("%sme/drives/%s/items/%s/search(q='%s')", customRootURL, folder.ParentReference.DriveID, folder.ID, url.QueryEscape(query))
	if paging.Top > 0 {
		initialURL += fmt.Sprintf("&$top=%d", paging.Top)
	}

	rawItems, nextLink, err := c.collectAllPages(initialURL, paging)
	if err != nil {
		return items, "", err
	}

	for _, raw := range rawItems {
		var item DriveItem
		if err := json.Unmarshal(raw, &item); err != nil {
			return items, "", fmt.Errorf("decoding item: %w", err)
		}
		items.Value = append(items.Value, item)
	}

	return items, nextLink, nil
}

// SearchDriveItemsWithPaging searches for items in the drive with paging support.
func (c *Client) SearchDriveItemsWithPaging(query string, paging Paging) (DriveItemList, string, error) {
	var items DriveItemList
	initialURL := fmt.Sprintf("%sme/drive/root/search(q='%s')", customRootURL, url.QueryEscape(query))
	if paging.Top > 0 {
		initialURL += fmt.Sprintf("&$top=%d", paging.Top)
	}

	rawItems, nextLink, err := c.collectAllPages(initialURL, paging)
	if err != nil {
		return items, "", err
	}

	for _, raw := range rawItems {
		var item DriveItem
		if err := json.Unmarshal(raw, &item); err != nil {
			return items, "", fmt.Errorf("decoding item: %w", err)
		}
		items.Value = append(items.Value, item)
	}

	return items, nextLink, nil
}
