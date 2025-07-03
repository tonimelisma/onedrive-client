// Package ui (display.go) provides functions for formatting and printing
// various OneDrive data structures (like DriveItems, Drives, Quota, User info, etc.)
// to the console in a user-friendly way. It also includes helpers for progress bars
// and standardized success/error messages.
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

// StdLogger implements the onedrive.Logger interface using the standard log package.
// This allows the SDK to output debug messages through the application's logger
// when debug mode is enabled.
type StdLogger struct{}

// Debug prints debug messages using the standard logger, prefixed with "DEBUG:".
// This method satisfies the new structured logger.Logger interface.
func (l StdLogger) Debug(msg string, args ...any) {
	if len(args) > 0 {
		log.Printf("DEBUG: "+msg, args...)
	} else {
		log.Println("DEBUG: " + msg)
	}
}

// Debugf prints formatted debug messages using the standard logger.
func (l StdLogger) Debugf(format string, args ...any) {
	log.Printf("DEBUG: "+format, args...)
}

// Info prints info messages using the standard logger.
func (l StdLogger) Info(msg string, args ...any) {
	if len(args) > 0 {
		log.Printf("INFO: "+msg, args...)
	} else {
		log.Println("INFO: " + msg)
	}
}

// Infof prints formatted info messages using the standard logger.
func (l StdLogger) Infof(format string, args ...any) {
	log.Printf("INFO: "+format, args...)
}

// Warn prints warning messages using the standard logger.
func (l StdLogger) Warn(msg string, args ...any) {
	if len(args) > 0 {
		log.Printf("WARN: "+msg, args...)
	} else {
		log.Println("WARN: " + msg)
	}
}

// Warnf prints formatted warning messages using the standard logger.
func (l StdLogger) Warnf(format string, args ...any) {
	log.Printf("WARN: "+format, args...)
}

// Error prints error messages using the standard logger.
func (l StdLogger) Error(msg string, args ...any) {
	if len(args) > 0 {
		log.Printf("ERROR: "+msg, args...)
	} else {
		log.Println("ERROR: " + msg)
	}
}

// Errorf prints formatted error messages using the standard logger.
func (l StdLogger) Errorf(format string, args ...any) {
	log.Printf("ERROR: "+format, args...)
}

// Success prints a simple success message to standard output.
// It can be expanded later for more styled output (e.g., colors).
func Success(msg string) {
	fmt.Println(msg)
}

// PrintSuccess prints a formatted success message to the console using the standard logger.
// It's intended for positive feedback to the user after successful operations.
func PrintSuccess(msg string, args ...interface{}) {
	// Using log.Printf ensures consistent output formatting with other log messages.
	log.Printf("SUCCESS: "+msg, args...)
}

// PrintError prints a formatted error message to the console using the standard logger.
// It's used for reporting errors encountered during command execution.
func PrintError(err error) {
	log.Printf("ERROR: %v", err)
}

// DisplayDriveItems prints a table of DriveItem resources, showing name, size, and type.
// It indicates whether each item is a file or a folder.
// The title parameter allows customizing the header message.
func DisplayDriveItems(items onedrive.DriveItemList) {
	DisplayDriveItemsWithTitle(items, "Items found:")
}

// DisplayDriveItemsWithTitle prints a table of DriveItem resources with a custom title.
// This resolves the issue where the title might be misleading when listing subfolders.
func DisplayDriveItemsWithTitle(items onedrive.DriveItemList, title string) {
	if len(items.Value) == 0 {
		fmt.Println("No items found in this location.")
		return
	}

	fmt.Println(title)
	fmt.Printf("%-50s %12s %s\n", "Name", "Size", "Type")
	fmt.Println(strings.Repeat("-", 70)) // Adjust line length based on columns
	for _, item := range items.Value {
		itemType := "File"
		if item.Folder != nil { // Check if the Folder facet is present.
			itemType = "Folder"
		}
		// Use formatBytes for human-readable size.
		fmt.Printf("%-50.50s %12s %s\n", item.Name, formatBytes(item.Size), itemType)
	}
}

// DisplayDrives prints a table of Drive resources, showing name, type, and owner.
func DisplayDrives(drives onedrive.DriveList) {
	if len(drives.Value) == 0 {
		fmt.Println("No drives found for this account.")
		return
	}

	fmt.Printf("%-35s %-20s %s\n", "Drive Name", "Drive Type", "Owner Display Name")
	fmt.Println(strings.Repeat("-", 80))
	for _, drive := range drives.Value {
		ownerName := "N/A"
		// Check if owner and user information is available.
		if drive.Owner.User != nil && drive.Owner.User.DisplayName != "" {
			ownerName = drive.Owner.User.DisplayName
		}
		driveName := drive.Name
		// Provide a default name if a personal drive has no explicit name.
		if drive.DriveType == "personal" && driveName == "" {
			driveName = "Personal OneDrive"
		}
		fmt.Printf("%-35.35s %-20s %s\n", driveName, drive.DriveType, ownerName)
	}
}

// formatBytes converts a size in bytes (int64) to a human-readable string
// using IEC units (KiB, MiB, GiB, etc.).
func formatBytes(b int64) string {
	const unit = 1024 // Use 1024 for KiB, MiB, etc.
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	// %.1f formats to one decimal place. "KMGTPE" are the unit prefixes.
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

// DisplayQuota prints detailed information about a drive's storage quota.
func DisplayQuota(drive onedrive.Drive) {
	fmt.Println("Drive Quota Information:")
	fmt.Printf("  Total Space: %s\n", formatBytes(drive.Quota.Total))
	fmt.Printf("  Used Space:  %s\n", formatBytes(drive.Quota.Used))
	fmt.Printf("  Free Space:  %s\n", formatBytes(drive.Quota.Remaining))
	fmt.Printf("  Quota State: %s\n", drive.Quota.State) // e.g., "normal", "nearing", "critical"
}

// DisplayUser prints information about the authenticated user.
func DisplayUser(user onedrive.User) {
	fmt.Printf("Logged in as: %s (User Principal Name: %s, ID: %s)\n", user.DisplayName, user.UserPrincipalName, user.ID)
}

// DisplayDriveItem prints detailed metadata for a single DriveItem.
func DisplayDriveItem(item onedrive.DriveItem) {
	fmt.Println("Item Metadata:")
	fmt.Printf("  Name:             %s\n", item.Name)
	fmt.Printf("  ID:               %s\n", item.ID)
	fmt.Printf("  Size:             %s (%d bytes)\n", formatBytes(item.Size), item.Size)
	fmt.Printf("  Created:          %s\n", item.CreatedDateTime.Local().Format(time.RFC1123)) // Format for readability
	fmt.Printf("  Last Modified:    %s\n", item.LastModifiedDateTime.Local().Format(time.RFC1123))
	if item.WebURL != "" {
		fmt.Printf("  Web URL:          %s\n", item.WebURL)
	}

	if item.Folder != nil {
		fmt.Printf("  Type:             Folder\n")
		fmt.Printf("  Child Count:      %d\n", item.Folder.ChildCount)
	} else if item.File != nil { // Check for File facet
		fmt.Printf("  Type:             File\n")
		if item.File.MimeType != "" {
			fmt.Printf("  MIME Type:        %s\n", item.File.MimeType)
		}
	} else {
		fmt.Printf("  Type:             Unknown/Other\n")
	}
	// Add more fields as needed, e.g., item.CreatedBy.User.DisplayName
}

// NewProgressBar creates and returns a new progress bar configured for file transfers.
// `maxBytes` is the total size of the transfer in bytes.
// `description` is the text displayed next to the progress bar.
func NewProgressBar(maxBytes int, description string) *progressbar.ProgressBar {
	if description == "" {
		description = "Processing..." // Default description
	}
	return progressbar.NewOptions(
		maxBytes,
		progressbar.OptionSetDescription(description),    // Use provided description
		progressbar.OptionSetWriter(os.Stderr),           // Write to Stderr to not interfere with Stdout data
		progressbar.OptionShowBytes(true),                // Display progress in bytes (e.g., 1.2MB/5MB)
		progressbar.OptionSetWidth(40),                   // Width of the progress bar itself
		progressbar.OptionThrottle(100*time.Millisecond), // Update frequency
		progressbar.OptionShowCount(),                    // Show item count (useful if maxBytes is actually item count)
		progressbar.OptionOnCompletion(func() {
			fmt.Fprint(os.Stderr, "\n") // Newline after completion
		}),
		progressbar.OptionSpinnerType(14), // A common spinner type
		// progressbar.OptionFullWidth(), // Makes the bar take the full terminal width; consider if desirable
		progressbar.OptionClearOnFinish(), // Clears the progress bar on completion
	)
}

// DisplaySearchResults prints search results in a table format, including the original query.
func DisplaySearchResults(items onedrive.DriveItemList, query string) {
	if len(items.Value) == 0 {
		fmt.Printf("No items found matching query: \"%s\"\n", query)
		return
	}

	fmt.Printf("Search results for \"%s\" (%d item(s)):\n", query, len(items.Value))
	// Adjust column widths as needed for typical data.
	fmt.Printf("%-60.60s %12s %-10s %s\n", "Name", "Size", "Type", "Last Modified")
	fmt.Println(strings.Repeat("-", 100)) // Adjust separator length

	for _, item := range items.Value {
		itemType := "File"
		if item.Folder != nil {
			itemType = "Folder"
		}
		// Format timestamp for better readability.
		modifiedTime := item.LastModifiedDateTime.Local().Format("2006-01-02 15:04")

		name := item.Name
		// Truncate long names to fit the column width.
		if len(name) > 57 { // 60 (column width) - 3 (for "...")
			name = name[:57] + "..."
		}

		fmt.Printf("%-60.60s %12s %-10s %s\n", name, formatBytes(item.Size), itemType, modifiedTime)
	}
}

// DisplaySharedItems prints items shared with the user.
// It attempts to show the owner and approximate shared date.
func DisplaySharedItems(items onedrive.DriveItemList) {
	if len(items.Value) == 0 {
		fmt.Println("No items have been shared with you.")
		return
	}

	fmt.Printf("Items shared with you (%d item(s)):\n", len(items.Value))
	fmt.Printf("%-50.50s %12s %-10s %-20s %s\n", "Name", "Size", "Type", "Shared/Created Date", "Shared By/Owner")
	fmt.Println(strings.Repeat("-", 110))

	for _, item := range items.Value {
		itemType := "File"
		if item.Folder != nil {
			itemType = "Folder"
		}

		name := item.Name
		if len(name) > 47 {
			name = name[:47] + "..."
		}

		// For shared items, `item.CreatedDateTime` might reflect when the share was created or item created.
		// `item.RemoteItem.Shared.SharedDateTime` would be ideal but might not always be populated directly in DriveItem.
		// `item.Shared.SharedDateTime` if `item.Shared` facet is present.
		// Using CreatedDateTime as a general proxy.
		sharedDate := item.CreatedDateTime.Local().Format("2006-01-02 15:04")

		owner := "N/A"
		// Try to get owner information. For shared items, this might be in `item.Shared.Owner`
		// or `item.CreatedBy` or `item.RemoteItem.CreatedBy`.
		// Prioritize `item.Shared.Owner` if available, then `item.RemoteItem.CreatedBy` for remote items,
		// then `item.CreatedBy` as a fallback.
		if item.CreatedBy.User != nil && item.CreatedBy.User.DisplayName != "" {
			owner = item.CreatedBy.User.DisplayName
		}
		// Note: RemoteItem does not contain CreatedBy information in the current model
		// Using the primary item's CreatedBy information only

		fmt.Printf("%-50.50s %12s %-10s %-20s %s\n", name, formatBytes(item.Size), itemType, sharedDate, owner)
	}
}

// DisplayRecentItems prints recently accessed or modified items.
func DisplayRecentItems(items onedrive.DriveItemList) {
	if len(items.Value) == 0 {
		fmt.Println("No recent items found.")
		return
	}

	fmt.Printf("Recently accessed/modified items (%d item(s)):\n", len(items.Value))
	fmt.Printf("%-60.60s %12s %-10s %s\n", "Name", "Size", "Type", "Last Modified/Accessed")
	fmt.Println(strings.Repeat("-", 100))

	for _, item := range items.Value {
		itemType := "File"
		if item.Folder != nil {
			itemType = "Folder"
		}

		name := item.Name
		if len(name) > 57 {
			name = name[:57] + "..."
		}

		// Graph API's "recent" endpoint often sorts by LastModifiedDateTime.
		// FileSystemInfo.LastAccessedDateTime is not standardly available for DriveItems.
		// Using LastModifiedDateTime as the primary timestamp.
		accessTime := item.LastModifiedDateTime.Local().Format("2006-01-02 15:04")

		fmt.Printf("%-60.60s %12s %-10s %s\n", name, formatBytes(item.Size), itemType, accessTime)
	}
}

// DisplaySpecialFolder prints detailed information about a requested special folder.
func DisplaySpecialFolder(item onedrive.DriveItem, folderName string) {
	fmt.Printf("Details for Special Folder: '%s'\n", folderName)
	// Use DisplayDriveItem for consistent detailed output.
	DisplayDriveItem(item)
}

// DisplaySharingLink prints information about a newly created sharing link.
func DisplaySharingLink(link onedrive.SharingLink) {
	fmt.Println("Sharing Link Created Successfully!")
	fmt.Printf("  Link ID:          %s\n", link.ID) // This is the permission ID for the link.
	fmt.Printf("  Type:             %s\n", link.Link.Type)
	fmt.Printf("  Scope:            %s\n", link.Link.Scope)
	if len(link.Roles) > 0 {
		fmt.Printf("  Effective Roles:  %s\n", strings.Join(link.Roles, ", "))
	}
	fmt.Printf("  Share URL:        %s\n", link.Link.WebUrl)

	if link.HasPassword {
		fmt.Printf("  Password:         Protected (password not displayed)\n")
	}

	if link.ExpirationDateTime != "" {
		expTime, err := time.Parse(time.RFC3339Nano, link.ExpirationDateTime)
		if err == nil {
			fmt.Printf("  Expires:          %s\n", expTime.Local().Format(time.RFC1123))
		} else {
			fmt.Printf("  Expires:          %s (raw value)\n", link.ExpirationDateTime)
		}
	}

	if link.Link.WebHtml != "" { // For 'embed' type links.
		fmt.Printf("  Embed HTML:       %s\n", link.Link.WebHtml)
	}

	if link.Link.Application != nil && link.Link.Application.DisplayName != "" {
		fmt.Printf("  Creating App:     %s (ID: %s)\n", link.Link.Application.DisplayName, link.Link.Application.Id)
	}
}

// DisplayDelta prints information about items returned from a delta query.
// It also shows the next deltaLink or nextLink for pagination if available.
func DisplayDelta(delta onedrive.DeltaResponse) {
	if len(delta.Value) == 0 {
		fmt.Println("No changes found since the last delta token (or no items in initial sync).")
	} else {
		fmt.Printf("Delta tracking results (%d item(s) changed or added):\n", len(delta.Value))
		// Use a compact format for delta, focusing on name and if it was deleted.
		fmt.Printf("%-60.60s %s\n", "Name", "Status")
		fmt.Println(strings.Repeat("-", 80))

		for _, item := range delta.Value {
			name := item.Name
			if len(name) > 57 {
				name = name[:57] + "..."
			}
			status := "Changed/Added"
			if item.Deleted != nil { // Check if the item has a 'deleted' facet.
				status = "Deleted"
			}
			fmt.Printf("%-60.60s %s\n", name, status)
		}
	}

	if delta.NextLink != "" {
		// This indicates the current set of delta results is paged.
		fmt.Printf("\nMore changes in this delta set. Use --next '%s' with the same delta token to continue.\n", delta.NextLink)
	}
	if delta.DeltaLink != "" {
		// This is the link to use for the *next* delta query to get changes *after* this set.
		fmt.Printf("\nDelta Link for next sync: %s\n(Save this token to get future changes)\n", delta.DeltaLink)
	}
}

// DisplayDrive prints detailed information about a specific OneDrive Drive resource.
func DisplayDrive(drive onedrive.Drive) {
	fmt.Println("Drive Information:")
	fmt.Printf("  Name:        %s\n", drive.Name)
	fmt.Printf("  ID:          %s\n", drive.ID)
	fmt.Printf("  Drive Type:  %s\n", drive.DriveType)

	if drive.Owner.User != nil && drive.Owner.User.DisplayName != "" {
		fmt.Printf("  Owner:       %s (ID: %s)\n", drive.Owner.User.DisplayName, drive.Owner.User.ID)
	}

	fmt.Println("  Storage Quota:")
	DisplayQuota(drive) // Re-use DisplayQuota for consistent formatting.
}

// DisplayFileVersions displays file version information in a formatted table.
func DisplayFileVersions(versions onedrive.DriveItemVersionList, filePath string) {
	if len(versions.Value) == 0 {
		fmt.Printf("No versions found for file: %s\n", filePath)
		return
	}

	fmt.Printf("File versions for: %s (%d version(s) found)\n\n", filePath, len(versions.Value))
	fmt.Printf("%-10s %-40.40s %12s %s\n", "Index", "Version ID", "Size", "Last Modified")
	fmt.Println(strings.Repeat("-", 95)) // Adjusted separator length

	// Display versions from newest to oldest (assuming API returns them that way, common pattern).
	// If API returns oldest first, might need to reverse or indicate.
	for i, version := range versions.Value {
		size := formatBytes(version.Size)
		lastModified := version.LastModifiedDateTime.Local().Format("2006-01-02 15:04:05")

		versionID := version.ID
		// Truncate long version IDs for display.
		if len(versionID) > 37 {
			versionID = versionID[:37] + "..."
		}

		fmt.Printf("%-10d %-40.40s %12s %s\n", i+1, versionID, size, lastModified)
	}
}

// DisplayActivities displays activity information in a formatted table.
// `title` is a descriptive string for the context of the activities (e.g., item path or "drive").
func DisplayActivities(activities onedrive.ActivityList, title string) {
	if len(activities.Value) == 0 {
		fmt.Printf("No activities found for %s.\n", title)
		return
	}

	fmt.Printf("Activities for: %s (%d activities found)\n\n", title, len(activities.Value))
	fmt.Printf("%-20s %-20.20s %-15s %s\n", "Date/Time", "Actor", "Action Type", "Item Name (if applicable)")
	fmt.Println(strings.Repeat("-", 90)) // Adjusted separator length

	for _, activity := range activities.Value {
		dateTime := activity.Times.RecordedTime.Local().Format("2006-01-02 15:04:05")

		actor := "N/A"
		if activity.Actor.User != nil && activity.Actor.User.DisplayName != "" {
			actor = activity.Actor.User.DisplayName
		}
		// Truncate long actor names.
		if len(actor) > 18 {
			actor = actor[:18] + "..."
		}

		// Determine action type more robustly.
		actionType := "Unknown"
		if activity.Action.Create != nil {
			actionType = "Create"
		} else if activity.Action.Edit != nil {
			actionType = "Edit"
		} else if activity.Action.Delete != nil {
			actionType = "Delete"
		} else if activity.Action.Move != nil {
			actionType = "Move"
		} else if activity.Action.Rename != nil {
			actionType = "Rename"
		} else if activity.Action.Share != nil {
			actionType = "Share"
		} else if activity.Action.Comment != nil {
			actionType = "Comment"
		} else if activity.Action.Version != nil {
			actionType = "Version Create" // Distinguish from simple Edit
		} else if activity.Action.Restore != nil {
			actionType = "Restore"
		} else if activity.Action.Mention != nil {
			actionType = "Mention"
		}
		// Could add more details from action facets if needed, e.g., activity.Action.Rename.OldName

		// Get item name if available in the activity.
		itemName := "N/A"
		if activity.DriveItem != nil && activity.DriveItem.Name != "" {
			itemName = activity.DriveItem.Name
		}

		fmt.Printf("%-20s %-20.20s %-15s %s\n", dateTime, actor, actionType, itemName)
	}
}

// DisplayThumbnails displays thumbnail information for a file.
func DisplayThumbnails(thumbnails onedrive.ThumbnailSetList, remotePath string) {
	if len(thumbnails.Value) == 0 {
		fmt.Printf("No thumbnails found for: %s\n", remotePath)
		return
	}

	fmt.Printf("Thumbnails for: %s (%d thumbnail set(s) found)\n\n", remotePath, len(thumbnails.Value))
	for i, thumbSet := range thumbnails.Value {
		fmt.Printf("Thumbnail Set %d (ID: %s):\n", i+1, thumbSet.ID)
		if thumbSet.Small != nil {
			fmt.Printf("  Small:  %4dx%-4d URL: %s\n", thumbSet.Small.Width, thumbSet.Small.Height, thumbSet.Small.URL)
		}
		if thumbSet.Medium != nil {
			fmt.Printf("  Medium: %4dx%-4d URL: %s\n", thumbSet.Medium.Width, thumbSet.Medium.Height, thumbSet.Medium.URL)
		}
		if thumbSet.Large != nil {
			fmt.Printf("  Large:  %4dx%-4d URL: %s\n", thumbSet.Large.Width, thumbSet.Large.Height, thumbSet.Large.URL)
		}
		if thumbSet.Source != nil { // Source usually refers to the original image dimensions for some APIs
			fmt.Printf("  Source: %4dx%-4d URL: %s\n", thumbSet.Source.Width, thumbSet.Source.Height, thumbSet.Source.URL)
		}
		fmt.Println() // Separator between thumbnail sets if multiple exist (rare for one item).
	}
}

// DisplayPreview displays preview information (embed URLs) for a file.
func DisplayPreview(preview onedrive.PreviewResponse, remotePath string) {
	fmt.Printf("Preview information for: %s\n", remotePath)

	if preview.GetURL != "" {
		fmt.Printf("  GET URL (for direct content):        %s\n", preview.GetURL)
	}
	if preview.PostURL != "" {
		fmt.Printf("  POST URL (for embedding with params): %s\n", preview.PostURL)
	}
	if preview.PostParameters != "" {
		fmt.Printf("  POST Parameters (form-encoded):     %s\n", preview.PostParameters)
	}

	if preview.GetURL == "" && preview.PostURL == "" {
		fmt.Println("  No specific preview URLs available for this file type or item.")
		fmt.Println("  The item might be viewable directly via its WebURL if it's a common format.")
	}
}

// DisplayInviteResponse displays the result of inviting users to access an item.
func DisplayInviteResponse(response onedrive.InviteResponse, remotePath string) {
	fmt.Printf("Invitation results for: %s\n", remotePath)

	if len(response.Value) == 0 {
		// This might mean the invitation was sent but no new permissions were immediately created,
		// or the API doesn't return permissions in this specific call for some reason.
		fmt.Println("Invitation processed. No new explicit permissions returned in this response (check item permissions separately if needed).")
		return
	}

	fmt.Printf("Successfully created/updated %d permission(s) through invitation:\n\n", len(response.Value))
	for i, permission := range response.Value {
		fmt.Printf("Permission %d:\n", i+1)
		displayPermissionDetails(permission) // Use the helper for detailed output.
		fmt.Println()
	}
}

// DisplayPermissions displays a list of permissions for a file or folder.
func DisplayPermissions(permissions onedrive.PermissionList, remotePath string) {
	if len(permissions.Value) == 0 {
		fmt.Printf("No explicit permissions found for: %s (item may inherit permissions or have default access).\n", remotePath)
		return
	}

	fmt.Printf("Permissions for: %s (%d permission(s) found)\n\n", remotePath, len(permissions.Value))
	// Adjusting column widths for better readability
	fmt.Printf("%-40.40s %-12s %-25.25s %s\n", "Permission ID", "Type", "Roles", "Granted To")
	fmt.Println(strings.Repeat("-", 120)) // Adjusted separator length

	for _, permission := range permissions.Value {
		permID := permission.ID
		if len(permID) > 38 { // Truncate long IDs
			permID = permID[:37] + "..."
		}

		permType := "Direct" // Assume direct unless a link is present
		if permission.Link != nil {
			permType = "Link (" + permission.Link.Scope + ")"
		}

		roles := strings.Join(permission.Roles, ", ")
		if len(roles) > 23 { // Truncate long role lists
			roles = roles[:20] + "..."
		}

		grantedTo := "N/A"
		if permission.GrantedToV2 != nil {
			if permission.GrantedToV2.User != nil && permission.GrantedToV2.User.DisplayName != "" {
				grantedTo = permission.GrantedToV2.User.DisplayName
				if permission.GrantedToV2.User.ID != "" {
					grantedTo += " [ID: " + permission.GrantedToV2.User.ID + "]"
				}
			} else if permission.GrantedToV2.SiteUser != nil && permission.GrantedToV2.SiteUser.DisplayName != "" { // Example for other identity types
				grantedTo = "SiteUser: " + permission.GrantedToV2.SiteUser.DisplayName
			}
		} else if permission.Link != nil && permission.Link.Scope == "anonymous" {
			grantedTo = "Anonymous (anyone with the link)"
		} else if permission.Link != nil && permission.Link.Scope == "organization" {
			grantedTo = "Organization (anyone in the org)"
		}

		fmt.Printf("%-40.40s %-12s %-25.25s %s\n", permID, permType, roles, grantedTo)
	}
}

// DisplaySinglePermission displays detailed information about a single permission.
func DisplaySinglePermission(permission onedrive.Permission, remotePath, permissionID string) {
	fmt.Printf("Detailed permission information for item: %s\n", remotePath)
	fmt.Printf("Permission ID: %s\n\n", permissionID) // Use the passed permissionID for consistency
	displayPermissionDetails(permission)
}

// displayPermissionDetails is an unexported helper function to print the details of a single Permission object.
// This promotes consistency in how permission details are displayed.
func displayPermissionDetails(permission onedrive.Permission) {
	fmt.Printf("  ID:                 %s\n", permission.ID)
	fmt.Printf("  Roles:              %s\n", strings.Join(permission.Roles, ", "))

	if permission.ShareID != "" {
		fmt.Printf("  Share ID:           %s\n", permission.ShareID)
	}
	if permission.ExpirationDateTime != "" {
		expTime, err := time.Parse(time.RFC3339Nano, permission.ExpirationDateTime)
		if err == nil {
			fmt.Printf("  Expires:            %s\n", expTime.Local().Format(time.RFC1123))
		} else {
			fmt.Printf("  Expires (raw):      %s\n", permission.ExpirationDateTime)
		}
	}
	if permission.HasPassword {
		fmt.Printf("  Password Protected: Yes\n")
	}

	// Display link information if this permission is for a sharing link.
	if permission.Link != nil {
		fmt.Printf("  Link Details:\n")
		fmt.Printf("    Type:             %s\n", permission.Link.Type)
		fmt.Printf("    Scope:            %s\n", permission.Link.Scope)
		fmt.Printf("    Web URL:          %s\n", permission.Link.WebURL)
		if permission.Link.WebHTML != "" {
			fmt.Printf("    Embed HTML:       %s\n", permission.Link.WebHTML)
		}
		if permission.Link.PreventsDownload {
			fmt.Printf("    Prevents Download:Yes\n")
		}
		if permission.Link.Application != nil && permission.Link.Application.DisplayName != "" {
			fmt.Printf("    Creating App:     %s (ID: %s)\n", permission.Link.Application.DisplayName, permission.Link.Application.ID)
		}
	}

	// Display "Granted To" information for direct permissions.
	if len(permission.GrantedToIdentitiesV2) > 0 {
		fmt.Printf("  Granted To Identities (%d):\n", len(permission.GrantedToIdentitiesV2))
		for i, identity := range permission.GrantedToIdentitiesV2 {
			fmt.Printf("    %d. ", i+1)
			if identity.User != nil {
				fmt.Printf("User: %s", identity.User.DisplayName)
				fmt.Printf(" [ID: %s]\n", identity.User.ID)
			} else if identity.SiteUser != nil { // Example for SharePoint site users
				fmt.Printf("Site User: %s", identity.SiteUser.DisplayName)
				fmt.Printf(" [ID: %s]\n", identity.SiteUser.ID)
			} else {
				fmt.Println("Unknown identity type")
			}
		}
	} else if permission.GrantedToV2 != nil { // Fallback for single GrantedToV2 if GrantedToIdentitiesV2 is empty
		fmt.Printf("  Granted To (V2):\n")
		if permission.GrantedToV2.User != nil {
			fmt.Printf("    User: %s", permission.GrantedToV2.User.DisplayName)
			fmt.Printf(" [ID: %s]\n", permission.GrantedToV2.User.ID)
		}
		if permission.GrantedToV2.SiteUser != nil {
			fmt.Printf("    Site User: %s", permission.GrantedToV2.SiteUser.DisplayName)
			fmt.Printf(" [ID: %s]\n", permission.GrantedToV2.SiteUser.ID)
		}
	}

	// Display "Inherited From" information if the permission is inherited.
	if permission.InheritedFrom != nil {
		fmt.Printf("  Inherited From (Item ID: %s):\n", permission.InheritedFrom.ID)
		if permission.InheritedFrom.DriveID != "" {
			fmt.Printf("    Drive ID: %s\n", permission.InheritedFrom.DriveID)
		}
		if permission.InheritedFrom.Path != "" { // Path is often like "/drive/root:"
			fmt.Printf("    Path:     %s\n", permission.InheritedFrom.Path)
		}
	}

	// Display invitation details if present
	if permission.Invitation != nil {
		fmt.Printf("  Invitation Details:\n")
		if permission.Invitation.Email != "" {
			fmt.Printf("    Invited Email: %s\n", permission.Invitation.Email)
		}
		fmt.Printf("    Sign-in Required: %t\n", permission.Invitation.SignInRequired)
	}
}
