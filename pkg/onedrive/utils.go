// Package onedrive provides utility functions for common operations and error handling.
package onedrive

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
)

// closeBodySafely closes an HTTP response body and logs any error.
// This is intended for use in defer statements where error handling is not critical.
func closeBodySafely(body io.Closer, logger Logger, operation string) {
	if err := body.Close(); err != nil {
		logger.Warnf("Failed to close %s body: %v", operation, err)
	}
}

// logOnError logs an error if it occurs, but doesn't return it.
// Useful for cleanup operations where we want to log but not fail the main operation.
func logOnError(err error, logger Logger, operation string) {
	if err != nil {
		logger.Warnf("Non-critical error in %s: %v", operation, err)
	}
}

// seekToStart resets a ReadSeeker to the beginning for retry operations.
// Returns any error from the seek operation for use with logOnError.
func seekToStart(body io.ReadSeeker) error {
	if body == nil {
		return nil
	}
	_, err := body.Seek(0, 0)
	return err
}

// getItemAndBuildURL is a helper function that retrieves a DriveItem by path
// and constructs a URL for the given endpoint. This reduces code duplication
// across permissions, thumbnails, and other item-based operations.
func (c *Client) getItemAndBuildURL(ctx context.Context, remotePath, endpoint string) (string, error) {
	// Get the DriveItem ID from the path.
	item, err := c.GetDriveItemByPath(ctx, remotePath)
	if err != nil {
		return "", fmt.Errorf("getting DriveItem ID for path '%s': %w", remotePath, err)
	}

	// Build the URL using the item ID and provided endpoint.
	itemURL := customRootURL + "me/drive/items/" + url.PathEscape(item.ID) + endpoint
	return itemURL, nil
}

// makeAPICallAndDecode performs an API call and decodes the JSON response into the provided destination.
// This reduces repetitive patterns of apiCall + defer + json.Decode across the SDK.
func (c *Client) makeAPICallAndDecode(ctx context.Context, method, apiURL, contentType string, body io.ReadSeeker, dest interface{}, operation string) error {
	res, err := c.apiCall(ctx, method, apiURL, contentType, body)
	if err != nil {
		return err
	}
	defer closeBodySafely(res.Body, c.logger, operation)

	if err := json.NewDecoder(res.Body).Decode(dest); err != nil {
		return fmt.Errorf("%w: decoding %s response: %w", ErrDecodingFailed, operation, err)
	}

	return nil
}

// buildSimpleURL constructs a URL from the custom root URL and endpoint, with optional ID escaping.
func buildSimpleURL(endpoint string, id string) string {
	if id != "" {
		return customRootURL + endpoint + "/" + url.PathEscape(id)
	}
	return customRootURL + endpoint
}

// readErrorBody reads and returns the error body from an HTTP response, with safe error handling.
func readErrorBody(body io.Reader) string {
	if body == nil {
		return ""
	}
	errorBody, _ := io.ReadAll(body)
	return string(errorBody)
}
