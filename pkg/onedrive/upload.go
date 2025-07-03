// Package onedrive (upload.go) provides methods for uploading files to OneDrive,
// particularly focusing on resumable uploads for large files using upload sessions.
// This allows large files to be uploaded in chunks, making the process more reliable
// over unstable connections.
package onedrive

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// CreateUploadSession initiates a resumable upload session for a large file.
// `remotePath` is the full path (including filename) where the file will be uploaded in OneDrive.
// This is the first step for uploading files larger than a few megabytes (typically > 4MB).
//
// Returns an UploadSession object containing the `uploadUrl` to which file chunks should be PUT,
// and the `expirationDateTime` for the session.
//
// Example:
//
//	session, err := client.CreateUploadSession(context.Background(), "/LargeFiles/MyBigVideo.mp4")
//	if err != nil { log.Fatal(err) }
//	fmt.Printf("Upload session created. URL: %s, Expires: %s\n", session.UploadURL, session.ExpirationDateTime)
//	// Use session.UploadURL with UploadChunk to upload file parts.
func (c *Client) CreateUploadSession(ctx context.Context, remotePath string) (UploadSession, error) {
	c.logger.Debugf("CreateUploadSession called for remotePath: '%s'", remotePath)
	var session UploadSession

	// The endpoint for creating an upload session is on the item's path with ":/createUploadSession".
	url := BuildPathURL(remotePath) + ":/createUploadSession"
	// The request body can be empty or contain item metadata for conflict resolution if needed.
	// For a simple session creation, an empty body (nil) is sufficient.
	res, err := c.apiCall(ctx, "POST", url, "application/json", nil)
	if err != nil {
		return session, err
	}
	defer closeBodySafely(res.Body, c.logger, "create upload session")

	if err := json.NewDecoder(res.Body).Decode(&session); err != nil {
		return session, fmt.Errorf("%w: decoding upload session response for '%s': %w", ErrDecodingFailed, remotePath, err)
	}

	return session, nil
}

// UploadChunk uploads a segment (chunk) of a large file to an existing upload session.
// `uploadURL` is the URL obtained from CreateUploadSession.
// `startByte` and `endByte` define the inclusive byte range of the chunk being uploaded.
// `totalSize` is the total size of the file being uploaded.
// `chunkData` is an io.Reader providing the data for this specific chunk.
//
// Returns an UploadSession object, which might contain updated `nextExpectedRanges`
// indicating the progress. On successful upload of the final chunk, the response
// will typically be an HTTP 201 Created or 200 OK with the DriveItem metadata.
//
// Note: This function uses a standard `http.Client` because the `uploadURL` provided by
// CreateUploadSession is a pre-authenticated URL that expects no Authorization header.
// The `Content-Range` and `Content-Length` headers are critical for this operation.
//
// Example:
//
//	// Assuming 'uploadSession' from CreateUploadSession and 'fileChunkReader' for a part of the file:
//	// fileChunkReader := bytes.NewReader(chunkBytes)
//	// currentChunkStartByte, currentChunkEndByte, totalFileSize defined appropriately.
//	status, err := client.UploadChunk(context.Background(), uploadSession.UploadURL, currentChunkStartByte, currentChunkEndByte, totalFileSize, fileChunkReader)
//	if err != nil { log.Fatal(err) }
//	if len(status.NextExpectedRanges) > 0 {
//	    fmt.Printf("Chunk uploaded. Next expected range starts at: %s\n", status.NextExpectedRanges[0])
//	} else {
//	    fmt.Println("Final chunk uploaded, file creation likely complete.")
//	    // The response body for the final chunk might contain the DriveItem metadata.
//	}
func (c *Client) UploadChunk(ctx context.Context, uploadURL string, startByte, endByte, totalSize int64, chunkData io.Reader) (UploadSession, error) {
	c.logger.Debugf("UploadChunk called for uploadURL: '%s', range: %d-%d, totalSize: %d", uploadURL, startByte, endByte, totalSize)
	var session UploadSession // To hold the response, which could be UploadSession or DriveItem on final chunk.

	// Use a configured HTTP client for consistent timeout behavior.
	// Upload URLs are pre-authenticated and don't require OAuth headers.
	httpClient := NewConfiguredHTTPClient(c.httpConfig)

	req, err := http.NewRequestWithContext(ctx, "PUT", uploadURL, chunkData)
	if err != nil {
		return session, fmt.Errorf("creating chunk upload request for URL '%s': %w", uploadURL, err)
	}

	// Set required headers for uploading a file chunk.
	req.Header.Set("Content-Length", fmt.Sprintf("%d", endByte-startByte+1))
	req.Header.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", startByte, endByte, totalSize))
	// No "Content-Type" is typically needed for chunk uploads to the session URL.

	res, err := httpClient.Do(req)
	if err != nil {
		return session, fmt.Errorf("uploading chunk to '%s' (range %d-%d): %w", uploadURL, startByte, endByte, err)
	}
	defer closeBodySafely(res.Body, c.logger, "upload chunk")

	// Successful chunk uploads return 202 Accepted (if more chunks expected) or
	// 201 Created / 200 OK (if this was the final chunk and file creation is complete).
	if res.StatusCode != http.StatusAccepted && res.StatusCode != http.StatusCreated && res.StatusCode != http.StatusOK {
		errorBody, _ := io.ReadAll(res.Body)
		return session, fmt.Errorf("uploading chunk to '%s' (range %d-%d) failed with status %s: %s", uploadURL, startByte, endByte, res.Status, string(errorBody))
	}

	// The response body for intermediate chunks contains UploadSession status.
	// For the final chunk, it contains the DriveItem metadata of the completed file.
	// We attempt to decode into UploadSession; if it's the final chunk, this might
	// partially fail or fields like NextExpectedRanges will be empty.
	// A more robust handling might try to decode into DriveItem if status is 200/201.
	if err := json.NewDecoder(res.Body).Decode(&session); err != nil {
		// If decoding fails but status was OK/Created, it might be the final DriveItem response.
		// For simplicity, we return the partially decoded session or error.
		// A more robust implementation might try decoding into DriveItem here.
		c.logger.Debugf("Error decoding UploadChunk response (URL: %s, status: %s): %v. This might be ok if it was the final chunk.", uploadURL, res.Status, err)
		// If it's the final chunk and a DriveItem is returned, NextExpectedRanges will be empty.
		// The caller should check this.
	}
	c.logger.Debugf("UploadChunk response for URL '%s': %+v", uploadURL, session)
	return session, nil
}

// GetUploadSessionStatus retrieves the current status of an active resumable upload session.
// This can be used to find out which byte ranges have been successfully uploaded,
// which is useful for resuming an interrupted upload.
// `uploadURL` is the URL of the upload session.
//
// Note: This function also uses a standard `http.Client` as the `uploadURL` is pre-authenticated.
//
// Example:
//
//	status, err := client.GetUploadSessionStatus(context.Background(), existingUploadURL)
//	if err != nil { log.Fatal(err) }
//	if len(status.NextExpectedRanges) > 0 {
//	    fmt.Printf("Upload session active. Next expected range: %s\n", status.NextExpectedRanges[0])
//	    // Resume upload from the start of NextExpectedRanges[0]
//	} else {
//	    fmt.Println("Upload session seems complete or invalid.")
//	}
func (c *Client) GetUploadSessionStatus(ctx context.Context, uploadURL string) (UploadSession, error) {
	c.logger.Debugf("GetUploadSessionStatus called for uploadURL: '%s'", uploadURL)
	var session UploadSession

	// Use a configured HTTP client for consistent timeout behavior.
	// Session URLs are pre-authenticated and don't require OAuth headers.
	httpClient := NewConfiguredHTTPClient(c.httpConfig)

	req, err := http.NewRequestWithContext(ctx, "GET", uploadURL, nil)
	if err != nil {
		return session, fmt.Errorf("creating get upload session status request for URL '%s': %w", uploadURL, err)
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return session, fmt.Errorf("getting upload session status from '%s': %w", uploadURL, err)
	}
	defer closeBodySafely(res.Body, c.logger, "get upload session status")

	if res.StatusCode != http.StatusOK {
		errorBody, _ := io.ReadAll(res.Body)
		return session, fmt.Errorf("getting upload session status from '%s' failed with status %s: %s", uploadURL, res.Status, string(errorBody))
	}

	if err := json.NewDecoder(res.Body).Decode(&session); err != nil {
		return session, fmt.Errorf("%w: decoding upload session status from '%s': %w", ErrDecodingFailed, uploadURL, err)
	}

	return session, nil
}

// CancelUploadSession cancels an active resumable upload session.
// This is useful if an upload is no longer needed, to free up any server-side resources.
// `uploadURL` is the URL of the upload session to cancel.
//
// Note: Uses a standard `http.Client` for the pre-authenticated `uploadURL`.
// A successful cancellation typically returns an HTTP 204 No Content response.
//
// Example:
//
//	err := client.CancelUploadSession(context.Background(), uploadURLToCancel)
//	if err != nil { log.Fatal(err) }
//	fmt.Println("Upload session canceled.")
func (c *Client) CancelUploadSession(ctx context.Context, uploadURL string) error {
	c.logger.Debugf("CancelUploadSession called for uploadURL: '%s'", uploadURL)
	// Use a configured HTTP client for consistent timeout behavior.
	httpClient := NewConfiguredHTTPClient(c.httpConfig)

	req, err := http.NewRequestWithContext(ctx, "DELETE", uploadURL, nil)
	if err != nil {
		return fmt.Errorf("creating cancel upload session request for URL '%s': %w", uploadURL, err)
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("canceling upload session at '%s': %w", uploadURL, err)
	}
	defer closeBodySafely(res.Body, c.logger, "cancel upload session")

	// Successful cancellation should return HTTP 204 No Content.
	if res.StatusCode != http.StatusNoContent {
		errorBody, _ := io.ReadAll(res.Body)
		return fmt.Errorf("canceling upload session at '%s' failed with status %s: %s", uploadURL, res.Status, string(errorBody))
	}

	return nil
}
