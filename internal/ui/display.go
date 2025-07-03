// Package ui (display.go) provides functions for formatting and printing
// various OneDrive data structures (like DriveItems, Drives, Quota, User info, etc.)
// to the console in a user-friendly way. It also includes helpers for progress bars
// and standardized success/error messages.
package ui

import (
	"fmt"
	"log"
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

// DisplayItems prints a table of DriveItem resources, showing name, size, and type.
// It indicates whether each item is a file or a folder.
// The title parameter allows customizing the header message.
func DisplayItems(items onedrive.DriveItemList) {
	if len(items.Value) == 0 {
		fmt.Println("No items found in the specified location.")
		return
	}

	fmt.Printf("Items in the specified location (%d item(s) found)\n\n", len(items.Value))
	fmt.Printf("%-60.60s %12s %-10s %s\n", "Name", "Size", "Type", "Last Modified")
	fmt.Println(strings.Repeat("-", onedrive.StandardSeparatorLength))
	for i := range items.Value {
		item := &items.Value[i] // Use pointer to avoid copying large struct
		itemType := "File"
		if item.Folder != nil {
			itemType = "Folder"
		}

		name := item.Name
		if len(name) > onedrive.MaxNameDisplayLength {
			name = name[:onedrive.MaxNameDisplayLength] + onedrive.EllipsisMarker
		}

		lastModified := item.LastModifiedDateTime.Local().Format(onedrive.StandardTimeFormat)

		fmt.Printf("%-60.60s %12s %-10s %s\n", name, formatBytes(item.Size), itemType, lastModified)
	}
}

// DisplayDrives displays a list of drives with their names, types, and quota information.
func DisplayDrives(drives onedrive.DriveList) {
	if len(drives.Value) == 0 {
		fmt.Println("No drives found for this account.")
		return
	}

	fmt.Printf("Drives for this account (%d drive(s) found)\n\n", len(drives.Value))
	fmt.Printf("%-40.40s %-15s %10s %10s %s\n", "Name", "Type", "Used", "Total", "Owner")
	fmt.Println(strings.Repeat("-", onedrive.StandardSeparatorLength))
	for i := range drives.Value {
		drive := &drives.Value[i] // Use pointer to avoid copying large struct
		owner := "N/A"
		if drive.Owner.User != nil && drive.Owner.User.DisplayName != "" {
			owner = drive.Owner.User.DisplayName
		}

		fmt.Printf("%-40.40s %-15s %10s %10s %s\n",
			drive.Name,
			drive.DriveType,
			formatBytes(drive.Quota.Used),
			formatBytes(drive.Quota.Total),
			owner)
	}
}

// formatBytes converts a size in bytes (int64) to a human-readable string
// using IEC units (KiB, MiB, GiB, etc.).
func formatBytes(b int64) string {
	const unit = onedrive.DefaultBufferSize // Use constant for consistency
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

// NewProgressBar creates and returns a new progress bar for file operations.
// It uses standard formatting options for consistency across the application.
func NewProgressBar(maxBytes int, description string) *progressbar.ProgressBar {
	return progressbar.NewOptions(maxBytes,
		progressbar.OptionSetDescription(description),
		progressbar.OptionShowIts(),                              // Show iteration rate (e.g., 1.2MB/s)
		progressbar.OptionShowCount(),                            // Show current/total count
		progressbar.OptionShowBytes(true),                        // Display progress in bytes (e.g., 1.2MB/5MB)
		progressbar.OptionSetWidth(onedrive.ProgressBarWidth),    // Use constant
		progressbar.OptionThrottle(onedrive.ProgressBarThrottle), // Use constant
	)
}

// NewSpinner creates a simple spinner for operations without progress tracking.
func NewSpinner() *progressbar.ProgressBar {
	return progressbar.NewOptions(-1,
		progressbar.OptionSpinnerType(onedrive.SpinnerType), // Use constant
	)
}

// DisplaySearchResults displays search results with highlighting or context.
func DisplaySearchResults(items onedrive.DriveItemList) {
	if len(items.Value) == 0 {
		fmt.Println("No items matched your search criteria.")
		return
	}

	fmt.Printf("Search results (%d item(s) found)\n\n", len(items.Value))
	fmt.Printf("%-60.60s %12s %-10s %-20s %s\n", "Name", "Size", "Type", "Last Modified", "Path")
	fmt.Println(strings.Repeat("-", onedrive.ExtraLongSeparatorLength))
	for i := range items.Value {
		item := &items.Value[i] // Use pointer to avoid copying large struct
		itemType := "File"
		if item.Folder != nil {
			itemType = "Folder"
		}

		name := item.Name
		if len(name) > onedrive.MaxShortNameLength {
			name = name[:onedrive.MaxShortNameLength] + onedrive.EllipsisMarker
		}

		path := item.ParentReference.Path
		if len(path) > onedrive.MaxShortPathLength {
			path = onedrive.EllipsisMarker + path[len(path)-onedrive.MaxShortPathLength+3:]
		}

		lastModified := item.LastModifiedDateTime.Local().Format(onedrive.StandardTimeFormat)

		fmt.Printf("%-60.60s %12s %-10s %-20s %s\n", name, formatBytes(item.Size), itemType, lastModified, path)
	}
}

// DisplaySharedItems displays items that have been shared with the user.
func DisplaySharedItems(items onedrive.DriveItemList) {
	if len(items.Value) == 0 {
		fmt.Println("No shared items found for this account.")
		return
	}

	fmt.Printf("Shared items for this account (%d item(s) found)\n\n", len(items.Value))
	fmt.Printf("%-50.50s %12s %-10s %-20s %s\n", "Name", "Size", "Type", "Shared/Created Date", "Shared By/Owner")
	fmt.Println(strings.Repeat("-", onedrive.ExtraLongLineLength))
	for i := range items.Value {
		item := &items.Value[i] // Use pointer to avoid copying large struct
		itemType := "File"
		if item.Folder != nil {
			itemType = "Folder"
		}

		name := item.Name
		if len(name) > onedrive.MaxShortNameLength {
			name = name[:onedrive.MaxShortNameLength] + onedrive.EllipsisMarker
		}

		owner := "N/A"
		if item.CreatedBy.User != nil && item.CreatedBy.User.DisplayName != "" {
			owner = item.CreatedBy.User.DisplayName
		}

		sharedDate := item.CreatedDateTime.Local().Format(onedrive.StandardTimeFormat)

		fmt.Printf("%-50.50s %12s %-10s %-20s %s\n", name, formatBytes(item.Size), itemType, sharedDate, owner)
	}
}

// DisplayRecentItems displays recently accessed items with timestamps.
func DisplayRecentItems(items onedrive.DriveItemList) {
	if len(items.Value) == 0 {
		fmt.Println("No recent items found for this account.")
		return
	}

	fmt.Printf("Recent items for this account (%d item(s) found)\n\n", len(items.Value))
	fmt.Printf("%-60.60s %12s %-10s %s\n", "Name", "Size", "Type", "Last Modified/Accessed")
	fmt.Println(strings.Repeat("-", onedrive.ExtraLongSeparatorLength))
	for i := range items.Value {
		item := &items.Value[i] // Use pointer to avoid copying large struct
		itemType := "File"
		if item.Folder != nil {
			itemType = "Folder"
		}

		name := item.Name
		if len(name) > onedrive.MaxNameDisplayLength {
			name = name[:onedrive.MaxNameDisplayLength] + onedrive.EllipsisMarker
		}

		accessTime := item.LastModifiedDateTime.Local().Format(onedrive.StandardTimeFormat)

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

// DisplayDeltaItems displays items from a delta response, indicating what has changed.
func DisplayDeltaItems(delta onedrive.DeltaResponse) {
	if len(delta.Value) == 0 {
		fmt.Println("No changes found since last sync.")
		return
	}

	fmt.Printf("Delta changes (%d item(s) found)\n\n", len(delta.Value))
	fmt.Printf("%-60.60s %12s %-10s %-20s %s\n", "Name", "Size", "Type", "Last Modified", "Status")
	fmt.Println(strings.Repeat("-", onedrive.ExtraLongSeparatorLength))
	for i := range delta.Value {
		item := &delta.Value[i] // Use pointer to avoid copying large struct
		itemType := "File"
		if item.Folder != nil {
			itemType = "Folder"
		}

		name := item.Name
		if len(name) > onedrive.MaxNameDisplayLength {
			name = name[:onedrive.MaxNameDisplayLength] + onedrive.EllipsisMarker
		}

		status := "Modified"
		if item.Deleted != nil {
			status = "Deleted"
		}

		lastModified := item.LastModifiedDateTime.Local().Format(onedrive.StandardTimeFormat)

		fmt.Printf("%-60.60s %12s %-10s %-20s %s\n", name, formatBytes(item.Size), itemType, lastModified, status)
	}

	if delta.DeltaLink != "" {
		fmt.Printf("\nDelta link for next sync: %s\n", delta.DeltaLink)
	}
	if delta.NextLink != "" {
		fmt.Printf("Next page available: %s\n", delta.NextLink)
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

// DisplayFileVersions displays file version history with formatting.
func DisplayFileVersions(versions onedrive.DriveItemVersionList, filePath string) {
	if len(versions.Value) == 0 {
		fmt.Printf("No versions found for file: %s\n", filePath)
		return
	}

	fmt.Printf("Version history for file: %s (%d version(s) found)\n\n", filePath, len(versions.Value))
	fmt.Printf("%-10s %-40.40s %12s %s\n", "Index", "Version ID", "Size", "Last Modified")
	fmt.Println(strings.Repeat("-", onedrive.LongSeparatorLength)) // Use constant
	for i, version := range versions.Value {
		size := "N/A"
		if version.Size > 0 {
			size = formatBytes(version.Size)
		}
		lastModified := version.LastModifiedDateTime.Local().Format(onedrive.FullTimeFormat) // Use constant

		versionID := version.ID
		if len(versionID) > onedrive.MaxVersionIDLength {
			versionID = versionID[:onedrive.MaxVersionIDLength] + onedrive.EllipsisMarker
		}

		fmt.Printf("%-10d %-40.40s %12s %s\n", i+1, versionID, size, lastModified)
	}
}

// DisplayActivities displays a list of activities with timestamps and actors.
func DisplayActivities(activities onedrive.ActivityList) {
	if len(activities.Value) == 0 {
		fmt.Println("No activities found.")
		return
	}

	fmt.Printf("Activities (%d found)\n\n", len(activities.Value))
	fmt.Printf("%-20s %-18s %-30s %s\n", "Time", "Actor", "Action", "Item Name")
	fmt.Println(strings.Repeat("-", onedrive.MediumSeparatorLength))
	for i := range activities.Value {
		activity := &activities.Value[i] // Use pointer to avoid copying large struct
		actorName := "Unknown"
		if activity.Actor.User != nil && activity.Actor.User.DisplayName != "" {
			actorName = activity.Actor.User.DisplayName
			if len(actorName) > onedrive.MaxActorNameLength {
				actorName = actorName[:onedrive.MaxActorNameLength] + onedrive.EllipsisMarker
			}
		} else if activity.Actor.Application != nil && activity.Actor.Application.DisplayName != "" {
			actorName = activity.Actor.Application.DisplayName
			if len(actorName) > onedrive.MaxActorNameLength {
				actorName = actorName[:onedrive.MaxActorNameLength] + onedrive.EllipsisMarker
			}
		}

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
		} else if activity.Action.Mention != nil {
			actionType = "Mention"
		} else if activity.Action.Restore != nil {
			actionType = "Restore"
		} else if activity.Action.Version != nil {
			actionType = "Version"
		}

		itemName := "N/A"
		if activity.DriveItem != nil && activity.DriveItem.Name != "" {
			itemName = activity.DriveItem.Name
		}

		timeFormatted := activity.Times.RecordedTime.Local().Format(onedrive.StandardTimeFormat)

		fmt.Printf("%-20s %-18s %-30s %s\n", timeFormatted, actorName, actionType, itemName)
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

// DisplayPermissions displays a list of permissions for a DriveItem with details.
func DisplayPermissions(permissions onedrive.PermissionList) {
	if len(permissions.Value) == 0 {
		fmt.Println("No permissions found for this item.")
		return
	}

	fmt.Printf("Permissions for this item (%d found)\n\n", len(permissions.Value))
	for i := range permissions.Value {
		permission := &permissions.Value[i] // Use pointer to avoid copying large struct
		displayPermissionDetails(*permission)
		fmt.Println(strings.Repeat("-", onedrive.StandardSeparatorLength))
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
	displayBasicPermissionInfo(permission)
	displayPermissionLink(permission)
	displayGrantedToInfo(permission)
	displayInheritedFromInfo(permission)
	displayInvitationInfo(permission)
}

// displayBasicPermissionInfo displays basic permission information
func displayBasicPermissionInfo(permission onedrive.Permission) {
	fmt.Printf("  ID:                 %s\n", permission.ID)
	fmt.Printf("  Roles:              %s\n", strings.Join(permission.Roles, ", "))

	if permission.ShareID != "" {
		fmt.Printf("  Share ID:           %s\n", permission.ShareID)
	}

	displayExpirationTime(permission.ExpirationDateTime)

	if permission.HasPassword {
		fmt.Printf("  Password Protected: Yes\n")
	}
}

// displayExpirationTime formats and displays permission expiration time
func displayExpirationTime(expirationDateTime string) {
	if expirationDateTime == "" {
		return
	}

	expTime, err := time.Parse(time.RFC3339Nano, expirationDateTime)
	if err == nil {
		fmt.Printf("  Expires:            %s\n", expTime.Local().Format(time.RFC1123))
	} else {
		fmt.Printf("  Expires (raw):      %s\n", expirationDateTime)
	}
}

// displayPermissionLink displays sharing link information if present
func displayPermissionLink(permission onedrive.Permission) {
	if permission.Link == nil {
		return
	}

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

// displayGrantedToInfo displays information about who the permission is granted to
func displayGrantedToInfo(permission onedrive.Permission) {
	if len(permission.GrantedToIdentitiesV2) > 0 {
		displayGrantedToIdentities(permission.GrantedToIdentitiesV2)
	} else if permission.GrantedToV2 != nil {
		displayGrantedToV2(permission.GrantedToV2)
	}
}

// displayGrantedToIdentities displays multiple granted identities
func displayGrantedToIdentities(identities []struct {
	User     *onedrive.Identity `json:"user,omitempty"`
	SiteUser *onedrive.Identity `json:"siteUser,omitempty"`
}) {
	fmt.Printf("  Granted To Identities (%d):\n", len(identities))
	for i, identity := range identities {
		fmt.Printf("    %d. ", i+1)
		displayIdentityInfo(identity.User, identity.SiteUser)
	}
}

// displayGrantedToV2 displays single granted identity (V2 format)
func displayGrantedToV2(grantedTo *struct {
	User     *onedrive.Identity `json:"user,omitempty"`
	SiteUser *onedrive.Identity `json:"siteUser,omitempty"`
}) {
	fmt.Printf("  Granted To (V2):\n")
	displayIdentityInfo(grantedTo.User, grantedTo.SiteUser)
}

// displayIdentityInfo displays user or site user identity information
func displayIdentityInfo(user, siteUser *onedrive.Identity) {
	if user != nil {
		fmt.Printf("User: %s", user.DisplayName)
		fmt.Printf(" [ID: %s]\n", user.ID)
	} else if siteUser != nil {
		fmt.Printf("Site User: %s", siteUser.DisplayName)
		fmt.Printf(" [ID: %s]\n", siteUser.ID)
	} else {
		fmt.Println("Unknown identity type")
	}
}

// displayInheritedFromInfo displays inheritance information if present
func displayInheritedFromInfo(permission onedrive.Permission) {
	if permission.InheritedFrom == nil {
		return
	}

	fmt.Printf("  Inherited From (Item ID: %s):\n", permission.InheritedFrom.ID)
	if permission.InheritedFrom.DriveID != "" {
		fmt.Printf("    Drive ID: %s\n", permission.InheritedFrom.DriveID)
	}
	if permission.InheritedFrom.Path != "" {
		fmt.Printf("    Path:     %s\n", permission.InheritedFrom.Path)
	}
}

// displayInvitationInfo displays invitation details if present
func displayInvitationInfo(permission onedrive.Permission) {
	if permission.Invitation == nil {
		return
	}

	fmt.Printf("  Invitation Details:\n")
	if permission.Invitation.Email != "" {
		fmt.Printf("    Invited Email: %s\n", permission.Invitation.Email)
	}
	fmt.Printf("    Sign-in Required: %t\n", permission.Invitation.SignInRequired)
}
