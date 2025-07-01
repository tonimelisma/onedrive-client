package onedrive

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
)

// GetThumbnails retrieves thumbnail images for a drive item.
func (c *Client) GetThumbnails(ctx context.Context, remotePath string) (ThumbnailSetList, error) {
	var thumbnails ThumbnailSetList
	item, err := c.GetDriveItemByPath(ctx, remotePath)
	if err != nil {
		return thumbnails, fmt.Errorf("getting drive item: %w", err)
	}

	url := customRootURL + "me/drive/items/" + url.PathEscape(item.ID) + "/thumbnails"
	res, err := c.apiCall(ctx, "GET", url, "", nil)
	if err != nil {
		return thumbnails, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&thumbnails); err != nil {
		return thumbnails, fmt.Errorf("decoding thumbnails response: %v", err)
	}

	return thumbnails, nil
}

// GetThumbnailBySize retrieves a specific size thumbnail for a drive item.
func (c *Client) GetThumbnailBySize(ctx context.Context, remotePath, thumbID, size string) (Thumbnail, error) {
	var thumbnail Thumbnail
	item, err := c.GetDriveItemByPath(ctx, remotePath)
	if err != nil {
		return thumbnail, fmt.Errorf("getting drive item: %w", err)
	}

	url := customRootURL + "me/drive/items/" + url.PathEscape(item.ID) + "/thumbnails/" + url.PathEscape(thumbID) + "/" + url.PathEscape(size)
	res, err := c.apiCall(ctx, "GET", url, "", nil)
	if err != nil {
		return thumbnail, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&thumbnail); err != nil {
		return thumbnail, fmt.Errorf("decoding thumbnail response: %v", err)
	}

	return thumbnail, nil
}

// PreviewItem creates a preview for a drive item.
func (c *Client) PreviewItem(ctx context.Context, remotePath string, request PreviewRequest) (PreviewResponse, error) {
	var preview PreviewResponse
	item, err := c.GetDriveItemByPath(ctx, remotePath)
	if err != nil {
		return preview, fmt.Errorf("getting drive item: %w", err)
	}

	url := customRootURL + "me/drive/items/" + url.PathEscape(item.ID) + "/preview"
	var bodyReader io.ReadSeeker
	if request.Page != "" || request.Zoom != 0 {
		requestBody, err := json.Marshal(request)
		if err != nil {
			return preview, fmt.Errorf("marshaling preview request: %v", err)
		}
		bodyReader = bytes.NewReader(requestBody)
	}

	res, err := c.apiCall(ctx, "POST", url, "application/json", bodyReader)
	if err != nil {
		return preview, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&preview); err != nil {
		return preview, fmt.Errorf("decoding preview response: %v", err)
	}

	return preview, nil
}
