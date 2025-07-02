// Package cmd (items_helpers.go) contains utility functions shared by various
// subcommands within the 'items' command group. These helpers assist with
// common tasks like path manipulation for remote OneDrive paths.
package cmd

import (
	"strings"
)

// joinRemotePath constructs a remote path for OneDrive by joining directory and file components.
// It ensures that paths are correctly formatted with forward slashes, suitable for URL construction,
// and handles edge cases like empty or root directory paths.
//
// This is different from `filepath.Join` which is OS-dependent and uses backslashes on Windows.
// OneDrive paths in the Graph API always use forward slashes.
//
// Behavior:
//   - If `dir` is empty or "/", it returns `file` prefixed with "/".
//   - It trims trailing slashes from `dir` and leading slashes from `file` before joining.
//   - The final result is always ensured to start with a single forward slash.
//
// Examples:
//   joinRemotePath("/Documents", "MyFile.txt") -> "/Documents/MyFile.txt"
//   joinRemotePath("/", "MyFile.txt")         -> "/MyFile.txt"
//   joinRemotePath("", "MyFile.txt")          -> "/MyFile.txt"
//   joinRemotePath("/Folder1/", "/SubFolder/file.doc") -> "/Folder1/SubFolder/file.doc"
func joinRemotePath(dir, file string) string {
	// If the directory is the root, the path is just the file prefixed with a slash.
	if dir == "" || dir == "/" {
		return "/" + strings.TrimPrefix(file, "/")
	}

	// Normalize directory and file parts:
	// - Ensure directory does not end with a slash.
	// - Ensure file does not start with a slash.
	// This prevents double slashes when joining.
	dir = strings.TrimSuffix(dir, "/")
	file = strings.TrimPrefix(file, "/")

	// Join the parts.
	result := dir + "/" + file

	// The Microsoft Graph API expects paths relative to the drive root to start with a colon,
	// e.g., "root:/Documents/file.txt". However, this function is for constructing the
	// user-facing path component *after* "root:".
	// The BuildPathURL function in the SDK (pkg/onedrive/client.go) handles the "root:" prefix.
	// This helper ensures the path *segment* is clean.
	// The primary role here is to ensure a clean, forward-slash-separated path.
	//
	// The requirement from ARCHITECTURE.md: "joinRemotePath() for URL-safe path joining"
	// and the current implementation's output (e.g., "/Documents/MyFile.txt") is consistent
	// with how paths are typically represented *before* being passed to BuildPathURL.
	// BuildPathURL itself expects paths like "/Documents/MyFile.txt".

	// The original implementation ensured the result starts with a forward slash if not already.
	// This is generally good practice for canonical path representation.
	if !strings.HasPrefix(result, "/") {
		result = "/" + result
	}
	return result
}
