package cmd

import (
	"strings"
)

// joinRemotePath joins directory and file path components for remote paths.
// Unlike filepath.Join, this is for URL paths and always uses forward slashes.
func joinRemotePath(dir, file string) string {
	if dir == "" || dir == "/" {
		return "/" + strings.TrimPrefix(file, "/")
	}
	dir = strings.TrimSuffix(dir, "/")
	file = strings.TrimPrefix(file, "/")

	// Ensure the result always starts with a forward slash
	result := dir + "/" + file
	if !strings.HasPrefix(result, "/") {
		result = "/" + result
	}
	return result
}
