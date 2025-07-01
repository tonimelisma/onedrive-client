package onedrive

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

// DownloadFile downloads a file from OneDrive.
// It handles the 302 redirect that Microsoft Graph API returns for download requests.
func (c *Client) DownloadFile(ctx context.Context, remotePath, localPath string) error {
	url := BuildPathURL(remotePath) + ":/content"

	// Create request but don't follow redirects automatically
	// The http client from oauth2 follows redirects by default, so we need a new one for this.
	noRedirectClient := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		// Copy transport from authenticated client
		Transport: c.httpClient.Transport,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("creating download request: %w", err)
	}

	res, err := noRedirectClient.Do(req)
	if err != nil {
		return fmt.Errorf("initiating download: %w", err)
	}
	defer res.Body.Close()

	// Handle 302 redirect to pre-authenticated download URL
	if res.StatusCode == http.StatusFound {
		downloadURL := res.Header.Get("Location")
		if downloadURL == "" {
			return fmt.Errorf("no download location in redirect header")
		}

		// Download from the pre-authenticated URL (no auth headers needed)
		return c.downloadFromURL(ctx, downloadURL, localPath)
	}

	// If we get 401 Unauthorized, try the alternative method using item metadata
	if res.StatusCode == http.StatusUnauthorized {
		return c.DownloadFileByItem(ctx, remotePath, localPath)
	}

	// If we get 404 Not Found, try the alternative method using item metadata
	if res.StatusCode == http.StatusNotFound {
		return c.DownloadFileByItem(ctx, remotePath, localPath)
	}

	// If not a redirect, assume it's the file content and save it
	return saveResponseToFile(res, localPath)
}

// DownloadFileByItem downloads a file by first getting its metadata.
// This is an alternative method that gets the download URL from item metadata first.
func (c *Client) DownloadFileByItem(ctx context.Context, remotePath, localPath string) error {
	// First get the item metadata to get the download URL
	item, err := c.GetDriveItemByPath(ctx, remotePath)
	if err != nil {
		return fmt.Errorf("getting item metadata for download: %w", err)
	}

	if item.DownloadURL == "" {
		return fmt.Errorf("item has no download URL")
	}

	return c.downloadFromURL(ctx, item.DownloadURL, localPath)
}

// downloadFromURL downloads a file from a URL (typically pre-authenticated).
func (c *Client) downloadFromURL(ctx context.Context, url, localPath string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("creating download request: %w", err)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("downloading from URL: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %s", res.Status)
	}

	return saveResponseToFile(res, localPath)
}

// saveResponseToFile saves an HTTP response body to a local file.
func saveResponseToFile(res *http.Response, localPath string) error {
	file, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("creating local file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, res.Body)
	if err != nil {
		return fmt.Errorf("saving to local file: %w", err)
	}

	return nil
}

// DownloadFileChunk downloads a specific chunk of a file.
func (c *Client) DownloadFileChunk(ctx context.Context, downloadURL string, startByte, endByte int64) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating chunk download request: %w", err)
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", startByte, endByte))

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("downloading chunk: %w", err)
	}

	if res.StatusCode != http.StatusPartialContent {
		defer res.Body.Close()
		return nil, fmt.Errorf("unexpected status code for chunk download: %d", res.StatusCode)
	}

	return res.Body, nil
}

// DownloadFileAsFormat downloads a file from OneDrive in a specific format.
func (c *Client) DownloadFileAsFormat(ctx context.Context, remotePath, localPath, format string) error {
	url := BuildPathURL(remotePath) + ":/content?format=" + url.QueryEscape(format)

	noRedirectClient := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: c.httpClient.Transport,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("creating download request: %w", err)
	}

	res, err := noRedirectClient.Do(req)
	if err != nil {
		return fmt.Errorf("initiating download: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusFound {
		downloadURL := res.Header.Get("Location")
		if downloadURL == "" {
			return fmt.Errorf("no download location in redirect header")
		}
		return c.downloadFromURL(ctx, downloadURL, localPath)
	}

	return saveResponseToFile(res, localPath)
}
