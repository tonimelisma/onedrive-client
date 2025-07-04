// Package onedrive (thumbnails.go) provides methods for retrieving thumbnail images
// and generating previews for DriveItems within a OneDrive drive. This is useful for
// displaying visual representations of files without downloading the entire file content.
package onedrive

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
)

// GetThumbnails retrieves a collection of thumbnail images for a specific DriveItem.
// OneDrive can generate thumbnails in various sizes (e.g., small, medium, large) for common
// file types like images, videos, and Office documents.
// `remotePath` is the path to the DriveItem.
//
// Returns a ThumbnailSetList which contains different available thumbnail sizes and their URLs.
//
// Example:
//
//	thumbnails, err := client.GetThumbnails(context.Background(), "/Pictures/MyPhoto.jpg")
//	if err != nil { log.Fatal(err) }
//	if len(thumbnails.Value) > 0 && thumbnails.Value[0].Medium != nil {
//	    fmt.Printf("Medium thumbnail URL: %s\n", thumbnails.Value[0].Medium.URL)
//	}
func (c *Client) GetThumbnails(ctx context.Context, remotePath string) (ThumbnailSetList, error) {
	c.logger.Debugf("GetThumbnails called for remotePath: '%s'", remotePath)
	var thumbnails ThumbnailSetList

	// Use helper to get item and build URL, reducing duplication.
	apiURL, err := c.getItemAndBuildURL(ctx, remotePath, "/thumbnails")
	if err != nil {
		return thumbnails, fmt.Errorf("building thumbnails URL for path '%s': %w", remotePath, err)
	}

	// Use helper for API call and decode, with proper error handling.
	err = c.makeAPICallAndDecode(ctx, "GET", apiURL, "", nil, &thumbnails, "get thumbnails")
	if err != nil {
		return thumbnails, err
	}

	return thumbnails, nil
}

// GetThumbnailBySize retrieves a specific thumbnail for a DriveItem by its set ID and size.
// Typically, a DriveItem has one thumbnail set with ID "0". Common sizes include "small",
// "medium", "large", or custom sizes like "c200x200".
// `remotePath` is the path to the DriveItem.
// `thumbID` is usually "0" for the default thumbnail set.
// `size` is the desired thumbnail size (e.g., "medium").
//
// Returns a single Thumbnail object containing the URL and dimensions for the requested size.
//
// Example:
//
//	mediumThumb, err := client.GetThumbnailBySize(context.Background(), "/Videos/MyMovie.mp4", "0", "medium")
//	if err != nil { log.Fatal(err) }
//	fmt.Printf("Medium thumbnail URL: %s, Width: %d, Height: %d\n", mediumThumb.URL, mediumThumb.Width, mediumThumb.Height)
func (c *Client) GetThumbnailBySize(ctx context.Context, remotePath, thumbID, size string) (Thumbnail, error) {
	c.logger.Debugf("GetThumbnailBySize called for remotePath: '%s', thumbID: '%s', size: '%s'", remotePath, thumbID, size)
	var thumbnail Thumbnail

	// Use helper to get item and build URL, reducing duplication.
	apiURL, err := c.getItemAndBuildURL(ctx, remotePath, "/thumbnails/"+url.PathEscape(thumbID)+"/"+url.PathEscape(size))
	if err != nil {
		return thumbnail, fmt.Errorf("building thumbnail URL for path '%s': %w", remotePath, err)
	}

	// Use helper for API call and decode, with proper error handling.
	err = c.makeAPICallAndDecode(ctx, "GET", apiURL, "", nil, &thumbnail, "get thumbnail by size")
	if err != nil {
		return thumbnail, err
	}

	return thumbnail, nil
}

// PreviewItem generates a short-lived embeddable preview URL for a DriveItem.
// This is useful for displaying previews of Office documents, PDFs, images, etc.,
// without requiring the user to download the file.
// `remotePath` is the path to the DriveItem.
// `request` is a PreviewRequest struct allowing optional parameters like page number or zoom level.
//
// Returns a PreviewResponse containing URLs (e.g., GetURL, PostURL) that can be used
// to render the preview.
//
// Example:
//
//	previewReq := onedrive.PreviewRequest{Page: "1", Zoom: 0.8} // Preview page 1 at 80% zoom
//	previewInfo, err := client.PreviewItem(context.Background(), "/Documents/Report.docx", previewReq)
//	if err != nil { log.Fatal(err) }
//	if previewInfo.GetURL != "" {
//	    fmt.Printf("Preview GetURL: %s\n", previewInfo.GetURL)
//	} else if previewInfo.PostURL != "" {
//	    fmt.Printf("Preview PostURL: %s, Parameters: %s\n", previewInfo.PostURL, previewInfo.PostParameters)
//	}
func (c *Client) PreviewItem(ctx context.Context, remotePath string, request PreviewRequest) (PreviewResponse, error) {
	c.logger.Debugf("PreviewItem called for remotePath: '%s', request: %+v", remotePath, request)
	var preview PreviewResponse

	// Use helper to get item and build URL, reducing duplication.
	apiURL, err := c.getItemAndBuildURL(ctx, remotePath, "/preview")
	if err != nil {
		return preview, fmt.Errorf("building preview URL for path '%s': %w", remotePath, err)
	}

	var bodyReader io.ReadSeeker

	// If page or zoom parameters are provided, include them in the POST request body.
	// An empty request body is also valid if no specific parameters are needed.
	if request.Page != "" || request.Zoom != 0 { // Check if zoom is non-default (0 is often default for "not set")
		requestBody, err := json.Marshal(request)
		if err != nil {
			return preview, fmt.Errorf("marshaling PreviewRequest for path '%s': %w", remotePath, err)
		}
		bodyReader = bytes.NewReader(requestBody)
		c.logger.Debugf("PreviewItem request body: %s", string(requestBody))
	} else {
		c.logger.Debug("PreviewItem called with no specific page or zoom parameters.")
	}

	// The preview action is a POST request.
	res, err := c.apiCall(ctx, "POST", apiURL, "application/json", bodyReader)
	if err != nil {
		return preview, err
	}
	defer closeBodySafely(res.Body, c.logger, "preview item")

	if err := json.NewDecoder(res.Body).Decode(&preview); err != nil {
		return preview, fmt.Errorf("%w: decoding preview response for path '%s': %w", ErrDecodingFailed, remotePath, err)
	}

	return preview, nil
}
