package onedrive

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// CreateUploadSession creates a new upload session for a large file.
func (c *Client) CreateUploadSession(ctx context.Context, remotePath string) (UploadSession, error) {
	var session UploadSession

	url := BuildPathURL(remotePath) + ":/createUploadSession"
	res, err := c.apiCall(ctx, "POST", url, "application/json", nil)
	if err != nil {
		return session, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&session); err != nil {
		return session, fmt.Errorf("decoding upload session failed: %v", err)
	}

	return session, nil
}

// UploadChunk uploads a chunk of a large file using an upload session.
// Note: this uses a standard http client because it's a pre-authenticated URL
// and the Graph API expects no Authorization header on this request.
func (c *Client) UploadChunk(ctx context.Context, uploadURL string, startByte, endByte, totalSize int64, chunkData io.Reader) (UploadSession, error) {
	var session UploadSession
	client := &http.Client{} // Use a standard client

	req, err := http.NewRequestWithContext(ctx, "PUT", uploadURL, chunkData)
	if err != nil {
		return session, fmt.Errorf("creating chunk upload request: %w", err)
	}
	req.Header.Set("Content-Length", fmt.Sprintf("%d", endByte-startByte+1))
	req.Header.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", startByte, endByte, totalSize))

	res, err := client.Do(req)
	if err != nil {
		return session, fmt.Errorf("uploading chunk: %w", err)
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&session); err != nil {
		return session, fmt.Errorf("decoding upload session response: %v", err)
	}

	return session, nil
}

// GetUploadSessionStatus retrieves the status of an upload session.
func (c *Client) GetUploadSessionStatus(ctx context.Context, uploadURL string) (UploadSession, error) {
	var session UploadSession
	client := &http.Client{}

	req, err := http.NewRequestWithContext(ctx, "GET", uploadURL, nil)
	if err != nil {
		return session, fmt.Errorf("creating get status request: %w", err)
	}

	res, err := client.Do(req)
	if err != nil {
		return session, fmt.Errorf("getting upload session status: %w", err)
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&session); err != nil {
		return session, fmt.Errorf("decoding upload session status: %v", err)
	}

	return session, nil
}

// CancelUploadSession cancels an upload session.
func (c *Client) CancelUploadSession(ctx context.Context, uploadURL string) error {
	client := &http.Client{}

	req, err := http.NewRequestWithContext(ctx, "DELETE", uploadURL, nil)
	if err != nil {
		return fmt.Errorf("creating cancel request: %w", err)
	}

	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("canceling upload session: %w", err)
	}
	defer res.Body.Close()

	return nil
}
