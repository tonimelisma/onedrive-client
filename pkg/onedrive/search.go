// Package onedrive (search.go) provides methods for searching for DriveItems
// within a OneDrive drive. This includes searching across the entire drive or
// scoping searches to specific folders, with support for pagination.
package onedrive

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// SearchDriveItems searches for items across the entire user's default drive
// based on a query string. This method is deprecated in favor of SearchDriveItemsWithPaging
// as it does not support pagination and might return incomplete results for large datasets.
//
// Deprecated: Use SearchDriveItemsWithPaging for pagination support.
//
// Example:
//
//	results, err := client.SearchDriveItems(context.Background(), "Annual Report")
//	if err != nil { log.Fatal(err) }
//	for _, item := range results.Value {
//	    fmt.Printf("Found item: %s\n", item.Name)
//	}
func (c *Client) SearchDriveItems(ctx context.Context, query string) (DriveItemList, error) {
	c.logger.Debugf("SearchDriveItems (deprecated) called with query: '%s'", query)
	var items DriveItemList

	// Endpoint for searching the root of the drive. Query string needs URL escaping.
	searchURL := customRootURL + "me/drive/root/search(q='" + url.QueryEscape(query) + "')"
	res, err := c.apiCall(ctx, "GET", searchURL, "", nil)
	if err != nil {
		return items, err
	}
	defer closeBodySafely(res.Body, c.logger, "search drive items")

	if err := json.NewDecoder(res.Body).Decode(&items); err != nil {
		return items, fmt.Errorf("%w: decoding search results for query '%s': %w", ErrDecodingFailed, query, err)
	}
	// Note: This does not handle NextLink from the response, so results might be truncated.
	return items, nil
}

// SearchDriveItemsInFolder searches for items matching a query string within a specific folder.
// `folderPath` is the path to the folder to search within.
// `query` is the search query string.
// `paging` allows for controlling pagination of the results.
//
// Returns a DriveItemList containing the found items and a `nextLink` string for pagination.
//
// Example:
//
//	results, nextLink, err := client.SearchDriveItemsInFolder(context.Background(), "/Projects", "Q3 Financials", onedrive.Paging{Top: 10})
//	if err != nil { log.Fatal(err) }
//	for _, item := range results.Value {
//	    fmt.Printf("Found in Projects: %s\n", item.Name)
//	}
//	if nextLink != "" { fmt.Println("More results available via nextLink.") }
func (c *Client) SearchDriveItemsInFolder(ctx context.Context, folderPath, query string, paging Paging) (DriveItemList, string, error) {
	c.logger.Debugf("SearchDriveItemsInFolder called for folderPath: '%s', query: '%s', paging: %+v", folderPath, query, paging)
	var items DriveItemList

	// First, get the DriveItem for the folder to obtain its ID.
	// The search-in-folder endpoint requires the folder's item ID.
	folderItem, err := c.GetDriveItemByPath(ctx, folderPath)
	if err != nil {
		return items, "", fmt.Errorf("getting folder item ID for path '%s' to search: %w", folderPath, err)
	}

	// Construct the base URL for searching within a folder.
	baseURL := customRootURL + "me/drive/items/" + folderItem.ID + "/search(q='" + url.QueryEscape(query) + "')"
	var queryParams []string
	if paging.Top > 0 {
		// The $top parameter for search results is typically applied outside the search() parentheses.
		queryParams = append(queryParams, fmt.Sprintf("$top=%d", paging.Top))
	}
	// Other OData query options like $select, $expand could be added here if needed.

	initialURL := baseURL
	if len(queryParams) > 0 {
		initialURL += "?" + strings.Join(queryParams, "&")
	}
	c.logger.Debug("SearchDriveItemsInFolder initialURL: ", initialURL)

	// Use collectAllPages helper for pagination.
	rawItems, nextLink, err := c.collectAllPages(ctx, initialURL, paging)
	if err != nil {
		return items, "", fmt.Errorf("collecting search results pages for folder '%s', query '%s': %w", folderPath, query, err)
	}

	// Unmarshal each raw JSON message into a DriveItem struct.
	for _, rawItem := range rawItems {
		var item DriveItem
		if err := json.Unmarshal(rawItem, &item); err != nil {
			c.logger.Debugf("Error unmarshaling search result item for folder '%s', query '%s': %v, raw: %s", folderPath, query, err, string(rawItem))
			return items, "", fmt.Errorf("unmarshaling search result item: %w", err)
		}
		items.Value = append(items.Value, item)
	}
	items.NextLink = nextLink // Populate NextLink in the returned DriveItemList

	return items, nextLink, nil
}

// SearchDriveItemsWithPaging searches for items across the entire user's default drive
// based on a query string, with support for pagination.
// `query` is the search query string.
// `paging` allows for controlling pagination of the results.
//
// Returns a DriveItemList containing the found items and a `nextLink` string for pagination.
// This is the recommended method for drive-wide searches.
//
// Example:
//
//	results, nextLink, err := client.SearchDriveItemsWithPaging(context.Background(), "Invoice", onedrive.Paging{Top: 20})
//	if err != nil { log.Fatal(err) }
//	for _, item := range results.Value {
//	    fmt.Printf("Found drive-wide: %s\n", item.Name)
//	}
//	if nextLink != "" { fmt.Println("More results available via nextLink.") }
func (c *Client) SearchDriveItemsWithPaging(ctx context.Context, query string, paging Paging) (DriveItemList, string, error) {
	c.logger.Debugf("SearchDriveItemsWithPaging called for query: '%s', paging: %+v", query, paging)
	var items DriveItemList

	// Construct the base URL for searching the root of the drive.
	baseURL := customRootURL + "me/drive/root/search(q='" + url.QueryEscape(query) + "')"
	var queryParams []string
	if paging.Top > 0 {
		// The $top parameter for search results is applied outside the search() parentheses.
		queryParams = append(queryParams, fmt.Sprintf("$top=%d", paging.Top))
	}

	initialURL := baseURL
	if len(queryParams) > 0 {
		initialURL += "?" + strings.Join(queryParams, "&")
	}
	c.logger.Debug("SearchDriveItemsWithPaging initialURL: ", initialURL)

	// Use collectAllPages helper for pagination.
	rawItems, nextLink, err := c.collectAllPages(ctx, initialURL, paging)
	if err != nil {
		return items, "", fmt.Errorf("collecting drive-wide search results pages for query '%s': %w", query, err)
	}

	// Unmarshal each raw JSON message into a DriveItem struct.
	for _, rawItem := range rawItems {
		var item DriveItem
		if err := json.Unmarshal(rawItem, &item); err != nil {
			c.logger.Debugf("Error unmarshaling drive-wide search result item for query '%s': %v, raw: %s", query, err, string(rawItem))
			return items, "", fmt.Errorf("unmarshaling drive-wide search result item: %w", err)
		}
		items.Value = append(items.Value, item)
	}
	items.NextLink = nextLink // Populate NextLink in the returned DriveItemList

	return items, nextLink, nil
}
