package ui

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/schollz/progressbar/v3"
	"github.com/tonimelisma/onedrive-client/pkg/onedrive"
)

// StdLogger implements the onedrive.Logger interface using standard log.
type StdLogger struct{}

func (l StdLogger) Debug(v ...interface{}) {
	log.Println(v...)
}

// Success prints a success message.
func Success(msg string) {
	// Simple wrapper for now, could be expanded later (e.g., with colors)
	fmt.Println(msg)
}

// PrintSuccess prints a success message to the console.
func PrintSuccess(msg string, args ...interface{}) {
	log.Printf("SUCCESS: "+msg+"\n", args...)
}

// PrintError prints an error message to the console.
func PrintError(err error) {
	log.Printf("ERROR: %v\n", err)
}

func DisplayDriveItems(items onedrive.DriveItemList) {
	fmt.Println("Items in your root folder:")
	for _, item := range items.Value {
		itemType := "File"
		if item.Folder != nil {
			itemType = "Folder"
		}
		fmt.Printf("- %-50s %10d %s\n", item.Name, item.Size, itemType)
	}
}

// DisplayDrives prints a table of Drive resources.
func DisplayDrives(drives onedrive.DriveList) {
	if len(drives.Value) == 0 {
		fmt.Println("No drives found.")
		return
	}

	fmt.Printf("%-35s %-15s %s\n", "Drive Name", "Drive Type", "Owner")
	fmt.Println(strings.Repeat("-", 80))
	for _, drive := range drives.Value {
		ownerName := "N/A"
		if drive.Owner.User.DisplayName != "" {
			ownerName = drive.Owner.User.DisplayName
		}
		driveName := drive.Name
		if drive.DriveType == "personal" && driveName == "" {
			driveName = "OneDrive"
		}
		fmt.Printf("%-35s %-15s %s\n", driveName, drive.DriveType, ownerName)
	}
}

// formatBytes converts bytes to a human-readable string.
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

// DisplayQuota prints detailed information about drive quota.
func DisplayQuota(drive onedrive.Drive) {
	fmt.Println("Drive Quota Information:")
	fmt.Printf("  Total:     %s\n", formatBytes(drive.Quota.Total))
	fmt.Printf("  Used:      %s\n", formatBytes(drive.Quota.Used))
	fmt.Printf("  Remaining: %s\n", formatBytes(drive.Quota.Remaining))
	fmt.Printf("  State:     %s\n", drive.Quota.State)
}

// DisplayUser prints information about a user.
func DisplayUser(user onedrive.User) {
	fmt.Printf("Logged in as: %s (%s)\n", user.DisplayName, user.UserPrincipalName)
}

// DisplayDriveItem prints detailed information about a single DriveItem.
func DisplayDriveItem(item onedrive.DriveItem) {
	fmt.Printf("           Name: %s\n", item.Name)
	fmt.Printf("             ID: %s\n", item.ID)
	fmt.Printf("           Size: %d bytes\n", item.Size)
	fmt.Printf("        Created: %s\n", item.CreatedDateTime)
	fmt.Printf(" Last Modified: %s\n", item.LastModifiedDateTime)
	fmt.Printf("        Web URL: %s\n", item.WebURL)

	if item.Folder != nil {
		fmt.Printf("           Type: Folder\n")
		fmt.Printf("   Child Count: %d\n", item.Folder.ChildCount)
	} else {
		fmt.Printf("           Type: File\n")
	}
}

// NewProgressBar creates and returns a new progress bar.
func NewProgressBar(maxBytes int) *progressbar.ProgressBar {
	return progressbar.NewOptions(
		maxBytes,
		progressbar.OptionSetDescription("Uploading..."),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(15),
		progressbar.OptionThrottle(65*time.Millisecond),
		progressbar.OptionShowCount(),
		progressbar.OptionOnCompletion(func() {
			fmt.Fprint(os.Stderr, "\n")
		}),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionFullWidth(),
	)
}

// DisplaySearchResults prints search results with query context.
func DisplaySearchResults(items onedrive.DriveItemList, query string) {
	if len(items.Value) == 0 {
		fmt.Printf("No items found for query: %s\n", query)
		return
	}

	fmt.Printf("Search results for '%s' (%d items):\n", query, len(items.Value))
	fmt.Printf("%-50s %10s %15s %s\n", "Name", "Size", "Type", "Modified")
	fmt.Println(strings.Repeat("-", 90))

	for _, item := range items.Value {
		itemType := "File"
		if item.Folder != nil {
			itemType = "Folder"
		}
		modifiedTime := item.LastModifiedDateTime.Format("2006-01-02")

		name := item.Name
		if len(name) > 45 {
			name = name[:42] + "..."
		}

		fmt.Printf("%-50s %10s %15s %s\n", name, formatBytes(item.Size), itemType, modifiedTime)
	}
}

// DisplaySharedItems prints items shared with the user, highlighting remote items.
func DisplaySharedItems(items onedrive.DriveItemList) {
	if len(items.Value) == 0 {
		fmt.Println("No items have been shared with you.")
		return
	}

	fmt.Printf("Items shared with you (%d items):\n", len(items.Value))
	fmt.Printf("%-50s %10s %15s %20s %s\n", "Name", "Size", "Type", "Shared Date", "Owner")
	fmt.Println(strings.Repeat("-", 110))

	for _, item := range items.Value {
		itemType := "File"
		if item.Folder != nil {
			itemType = "Folder"
		}

		name := item.Name
		if len(name) > 45 {
			name = name[:42] + "..."
		}

		sharedDate := "N/A"
		owner := "N/A"

		// Check if this is a remote item (shared from another drive)
		if item.RemoteItem != nil {
			// Use the remote item's creation date as shared date approximation
			sharedDate = item.CreatedDateTime.Format("2006-01-02")
		}

		// Try to get owner from CreatedBy
		if item.CreatedBy.User.DisplayName != "" {
			owner = item.CreatedBy.User.DisplayName
		}

		fmt.Printf("%-50s %10s %15s %20s %s\n", name, formatBytes(item.Size), itemType, sharedDate, owner)
	}
}

// DisplayRecentItems prints recently accessed items with last access time.
func DisplayRecentItems(items onedrive.DriveItemList) {
	if len(items.Value) == 0 {
		fmt.Println("No recent items found.")
		return
	}

	fmt.Printf("Recently accessed items (%d items):\n", len(items.Value))
	fmt.Printf("%-50s %10s %15s %20s\n", "Name", "Size", "Type", "Last Accessed")
	fmt.Println(strings.Repeat("-", 100))

	for _, item := range items.Value {
		itemType := "File"
		if item.Folder != nil {
			itemType = "Folder"
		}

		name := item.Name
		if len(name) > 45 {
			name = name[:42] + "..."
		}

		// Use LastModifiedDateTime as proxy for last access if fileSystemInfo not available
		accessTime := item.LastModifiedDateTime.Format("2006-01-02 15:04")
		if !item.FileSystemInfo.LastModifiedDateTime.IsZero() {
			accessTime = item.FileSystemInfo.LastModifiedDateTime.Format("2006-01-02 15:04")
		}

		fmt.Printf("%-50s %10s %15s %20s\n", name, formatBytes(item.Size), itemType, accessTime)
	}
}

// DisplaySpecialFolder prints detailed information about a special folder.
func DisplaySpecialFolder(item onedrive.DriveItem, folderName string) {
	fmt.Printf("Special Folder: %s\n", folderName)
	fmt.Printf("           Name: %s\n", item.Name)
	fmt.Printf("             ID: %s\n", item.ID)
	fmt.Printf("           Size: %d bytes\n", item.Size)
	fmt.Printf("        Created: %s\n", item.CreatedDateTime)
	fmt.Printf(" Last Modified: %s\n", item.LastModifiedDateTime)
	fmt.Printf("        Web URL: %s\n", item.WebURL)

	if item.Folder != nil {
		fmt.Printf("           Type: Folder\n")
		fmt.Printf("   Child Count: %d\n", item.Folder.ChildCount)
	} else {
		fmt.Printf("           Type: File\n")
	}
}

// DisplaySharingLink prints information about a created sharing link.
func DisplaySharingLink(link onedrive.SharingLink) {
	fmt.Printf("Sharing Link Created Successfully!\n")
	fmt.Printf("           ID: %s\n", link.ID)
	fmt.Printf("         Type: %s\n", link.Link.Type)
	fmt.Printf("        Scope: %s\n", link.Link.Scope)
	fmt.Printf("        Roles: %s\n", strings.Join(link.Roles, ", "))
	fmt.Printf("          URL: %s\n", link.Link.WebUrl)

	if link.HasPassword {
		fmt.Printf("     Password: Protected\n")
	}

	if link.ExpirationDateTime != "" {
		fmt.Printf("      Expires: %s\n", link.ExpirationDateTime)
	}

	if link.Link.WebHtml != "" {
		fmt.Printf("    Embed HTML: %s\n", link.Link.WebHtml)
	}

	if link.Link.Application.DisplayName != "" {
		fmt.Printf("  Application: %s\n", link.Link.Application.DisplayName)
	}
}

// DisplayDelta prints information about delta tracking results.
func DisplayDelta(delta onedrive.DeltaResponse) {
	if len(delta.Value) == 0 {
		fmt.Println("No changes found.")
		return
	}

	fmt.Printf("Delta tracking results (%d items):\n", len(delta.Value))
	fmt.Printf("%-50s %10s %15s %20s\n", "Name", "Size", "Type", "Last Modified")
	fmt.Println(strings.Repeat("-", 100))

	for _, item := range delta.Value {
		itemType := "File"
		if item.Folder != nil {
			itemType = "Folder"
		}

		name := item.Name
		if len(name) > 45 {
			name = name[:42] + "..."
		}

		modifiedTime := item.LastModifiedDateTime.Format("2006-01-02 15:04")

		fmt.Printf("%-50s %10s %15s %20s\n", name, formatBytes(item.Size), itemType, modifiedTime)
	}

	if delta.DeltaLink != "" {
		fmt.Printf("\nDelta Link (save for next sync): %s\n", delta.DeltaLink)
	}
	if delta.NextLink != "" {
		fmt.Printf("Next Link (continue current sync): %s\n", delta.NextLink)
	}
}

// DisplayDrive prints detailed information about a specific drive.
func DisplayDrive(drive onedrive.Drive) {
	fmt.Printf("Drive Information:\n")
	fmt.Printf("           Name: %s\n", drive.Name)
	fmt.Printf("             ID: %s\n", drive.ID)
	fmt.Printf("           Type: %s\n", drive.DriveType)

	if drive.Owner.User.DisplayName != "" {
		fmt.Printf("          Owner: %s\n", drive.Owner.User.DisplayName)
	}

	fmt.Printf("  Storage Quota:\n")
	fmt.Printf("          Total: %s\n", formatBytes(drive.Quota.Total))
	fmt.Printf("           Used: %s\n", formatBytes(drive.Quota.Used))
	fmt.Printf("      Remaining: %s\n", formatBytes(drive.Quota.Remaining))
	fmt.Printf("          State: %s\n", drive.Quota.State)
}

// DisplayFileVersions prints all versions of a file with details.
func DisplayFileVersions(versions onedrive.DriveItemVersionList, filePath string) {
	if len(versions.Value) == 0 {
		fmt.Printf("No versions found for file: %s\n", filePath)
		return
	}

	fmt.Printf("File versions for '%s' (%d versions):\n", filePath, len(versions.Value))
	fmt.Printf("%-20s %10s %20s %s\n", "Version ID", "Size", "Last Modified", "Modified By")
	fmt.Println(strings.Repeat("-", 80))

	for _, version := range versions.Value {
		versionID := version.ID
		if len(versionID) > 18 {
			versionID = versionID[:15] + "..."
		}

		modifiedTime := version.LastModifiedDateTime.Format("2006-01-02 15:04")
		modifiedBy := "N/A"
		if version.LastModifiedBy.User.DisplayName != "" {
			modifiedBy = version.LastModifiedBy.User.DisplayName
		}

		fmt.Printf("%-20s %10s %20s %s\n", versionID, formatBytes(version.Size), modifiedTime, modifiedBy)
	}
}
