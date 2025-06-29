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

// DisplayFileVersions displays file version information in a formatted table
func DisplayFileVersions(versions onedrive.DriveItemVersionList, filePath string) {
	if len(versions.Value) == 0 {
		fmt.Printf("No versions found for file: %s\n", filePath)
		return
	}

	fmt.Printf("File versions for: %s\n", filePath)
	fmt.Printf("Found %d version(s)\n\n", len(versions.Value))

	fmt.Printf("%-40s %12s %20s\n", "Version ID", "Size", "Last Modified")
	fmt.Println(strings.Repeat("-", 75))

	for _, version := range versions.Value {
		size := formatBytes(version.Size)
		lastModified := version.LastModifiedDateTime.Format("2006-01-02 15:04:05")

		versionID := version.ID
		if len(versionID) > 38 {
			versionID = versionID[:35] + "..."
		}

		fmt.Printf("%-40s %12s %20s\n", versionID, size, lastModified)
	}
}

// DisplayActivities displays activity information in a formatted table
func DisplayActivities(activities onedrive.ActivityList, title string) {
	if len(activities.Value) == 0 {
		fmt.Printf("No activities found for %s\n", title)
		return
	}

	fmt.Printf("Activities for %s\n", title)
	fmt.Printf("Found %d activit(ies)\n\n", len(activities.Value))

	fmt.Printf("%-20s %-15s %-15s %s\n", "Date", "Actor", "Action", "Item")
	fmt.Println(strings.Repeat("-", 80))

	for _, activity := range activities.Value {
		date := activity.Times.RecordedTime.Format("2006-01-02 15:04")
		actor := activity.Actor.User.DisplayName
		if actor == "" {
			actor = "Unknown"
		}
		if len(actor) > 13 {
			actor = actor[:10] + "..."
		}

		// Determine action type
		action := "Unknown"
		if activity.Action.Create != nil {
			action = "Create"
		} else if activity.Action.Edit != nil {
			action = "Edit"
		} else if activity.Action.Delete != nil {
			action = "Delete"
		} else if activity.Action.Move != nil {
			action = "Move"
		} else if activity.Action.Rename != nil {
			action = "Rename"
		} else if activity.Action.Share != nil {
			action = "Share"
		} else if activity.Action.Comment != nil {
			action = "Comment"
		} else if activity.Action.Version != nil {
			action = "Version"
		} else if activity.Action.Restore != nil {
			action = "Restore"
		} else if activity.Action.Mention != nil {
			action = "Mention"
		}

		if len(action) > 13 {
			action = action[:10] + "..."
		}

		// Get item name if available
		itemName := ""
		if activity.DriveItem != nil {
			itemName = activity.DriveItem.Name
		}

		fmt.Printf("%-20s %-15s %-15s %s\n", date, actor, action, itemName)
	}
}

// New Epic 7 display functions for thumbnails, preview, and permissions

// DisplayThumbnails displays thumbnail information for a file
func DisplayThumbnails(thumbnails onedrive.ThumbnailSetList, remotePath string) {
	if len(thumbnails.Value) == 0 {
		fmt.Printf("No thumbnails found for: %s\n", remotePath)
		return
	}

	fmt.Printf("Thumbnails for: %s\n", remotePath)
	fmt.Printf("Found %d thumbnail set(s)\n\n", len(thumbnails.Value))

	for i, thumbSet := range thumbnails.Value {
		fmt.Printf("Thumbnail Set %d (ID: %s):\n", i+1, thumbSet.ID)

		if thumbSet.Small != nil {
			fmt.Printf("  Small:  %dx%d - %s\n", thumbSet.Small.Width, thumbSet.Small.Height, thumbSet.Small.URL)
		}
		if thumbSet.Medium != nil {
			fmt.Printf("  Medium: %dx%d - %s\n", thumbSet.Medium.Width, thumbSet.Medium.Height, thumbSet.Medium.URL)
		}
		if thumbSet.Large != nil {
			fmt.Printf("  Large:  %dx%d - %s\n", thumbSet.Large.Width, thumbSet.Large.Height, thumbSet.Large.URL)
		}
		if thumbSet.Source != nil {
			fmt.Printf("  Source: %dx%d - %s\n", thumbSet.Source.Width, thumbSet.Source.Height, thumbSet.Source.URL)
		}
		fmt.Println()
	}
}

// DisplayPreview displays preview information for a file
func DisplayPreview(preview onedrive.PreviewResponse, remotePath string) {
	fmt.Printf("Preview generated for: %s\n", remotePath)

	if preview.GetURL != "" {
		fmt.Printf("  GET URL: %s\n", preview.GetURL)
	}
	if preview.PostURL != "" {
		fmt.Printf("  POST URL: %s\n", preview.PostURL)
	}
	if preview.PostParameters != "" {
		fmt.Printf("  POST Parameters: %s\n", preview.PostParameters)
	}

	if preview.GetURL == "" && preview.PostURL == "" {
		fmt.Println("  No preview URLs available for this file")
	}
}

// DisplayInviteResponse displays the result of inviting users to access a file
func DisplayInviteResponse(response onedrive.InviteResponse, remotePath string) {
	fmt.Printf("Invitation sent successfully for: %s\n", remotePath)

	if len(response.Value) == 0 {
		fmt.Println("No permissions were created")
		return
	}

	fmt.Printf("Created %d permission(s):\n\n", len(response.Value))

	for i, permission := range response.Value {
		fmt.Printf("Permission %d:\n", i+1)
		displayPermissionDetails(permission)
		fmt.Println()
	}
}

// DisplayPermissions displays a list of permissions for a file or folder
func DisplayPermissions(permissions onedrive.PermissionList, remotePath string) {
	if len(permissions.Value) == 0 {
		fmt.Printf("No permissions found for: %s\n", remotePath)
		return
	}

	fmt.Printf("Permissions for: %s\n", remotePath)
	fmt.Printf("Found %d permission(s)\n\n", len(permissions.Value))

	fmt.Printf("%-40s %-15s %-20s %s\n", "Permission ID", "Type", "Roles", "Granted To")
	fmt.Println(strings.Repeat("-", 90))

	for _, permission := range permissions.Value {
		permID := permission.ID
		if len(permID) > 38 {
			permID = permID[:35] + "..."
		}

		permType := "Direct"
		if permission.Link != nil {
			permType = "Link"
		}

		roles := strings.Join(permission.Roles, ", ")
		if len(roles) > 18 {
			roles = roles[:15] + "..."
		}

		grantedTo := "Anonymous"
		if permission.GrantedToV2 != nil {
			if permission.GrantedToV2.User != nil {
				grantedTo = permission.GrantedToV2.User.DisplayName
				if permission.GrantedToV2.User.Email != "" {
					grantedTo += " (" + permission.GrantedToV2.User.Email + ")"
				}
			} else if permission.GrantedToV2.SiteUser != nil {
				grantedTo = permission.GrantedToV2.SiteUser.DisplayName
				if permission.GrantedToV2.SiteUser.Email != "" {
					grantedTo += " (" + permission.GrantedToV2.SiteUser.Email + ")"
				}
			}
		}

		fmt.Printf("%-40s %-15s %-20s %s\n", permID, permType, roles, grantedTo)
	}
}

// DisplaySinglePermission displays detailed information about a single permission
func DisplaySinglePermission(permission onedrive.Permission, remotePath, permissionID string) {
	fmt.Printf("Permission details for: %s\n", remotePath)
	fmt.Printf("Permission ID: %s\n\n", permissionID)

	displayPermissionDetails(permission)
}

// displayPermissionDetails is a helper function to display permission details
func displayPermissionDetails(permission onedrive.Permission) {
	fmt.Printf("  ID: %s\n", permission.ID)
	fmt.Printf("  Roles: %s\n", strings.Join(permission.Roles, ", "))

	if permission.PermissionScope != "" {
		fmt.Printf("  Scope: %s\n", permission.PermissionScope)
	}

	if permission.ExpirationDateTime != "" {
		fmt.Printf("  Expires: %s\n", permission.ExpirationDateTime)
	}

	if permission.HasPassword {
		fmt.Printf("  Password Protected: Yes\n")
	}

	if permission.ShareID != "" {
		fmt.Printf("  Share ID: %s\n", permission.ShareID)
	}

	// Display link information if available
	if permission.Link != nil {
		fmt.Printf("  Link Type: %s\n", permission.Link.Type)
		fmt.Printf("  Link Scope: %s\n", permission.Link.Scope)
		fmt.Printf("  Link URL: %s\n", permission.Link.WebURL)

		if permission.Link.WebHTML != "" {
			fmt.Printf("  Embed HTML: %s\n", permission.Link.WebHTML)
		}

		if permission.Link.PreventsDownload {
			fmt.Printf("  Prevents Download: Yes\n")
		}
	}

	// Display granted to information
	if permission.GrantedToV2 != nil {
		fmt.Printf("  Granted To:\n")
		if permission.GrantedToV2.User != nil {
			fmt.Printf("    User: %s", permission.GrantedToV2.User.DisplayName)
			if permission.GrantedToV2.User.Email != "" {
				fmt.Printf(" (%s)", permission.GrantedToV2.User.Email)
			}
			fmt.Printf(" [ID: %s]\n", permission.GrantedToV2.User.ID)
		}
		if permission.GrantedToV2.SiteUser != nil {
			fmt.Printf("    Site User: %s", permission.GrantedToV2.SiteUser.DisplayName)
			if permission.GrantedToV2.SiteUser.Email != "" {
				fmt.Printf(" (%s)", permission.GrantedToV2.SiteUser.Email)
			}
			fmt.Printf(" [ID: %s]\n", permission.GrantedToV2.SiteUser.ID)
		}
	}

	// Display multiple granted identities if available
	if len(permission.GrantedToIdentitiesV2) > 0 {
		fmt.Printf("  Granted To Identities (%d):\n", len(permission.GrantedToIdentitiesV2))
		for i, identity := range permission.GrantedToIdentitiesV2 {
			fmt.Printf("    %d. ", i+1)
			if identity.User != nil {
				fmt.Printf("User: %s", identity.User.DisplayName)
				if identity.User.Email != "" {
					fmt.Printf(" (%s)", identity.User.Email)
				}
				fmt.Printf(" [ID: %s]\n", identity.User.ID)
			}
			if identity.SiteUser != nil {
				fmt.Printf("Site User: %s", identity.SiteUser.DisplayName)
				if identity.SiteUser.Email != "" {
					fmt.Printf(" (%s)", identity.SiteUser.Email)
				}
				fmt.Printf(" [ID: %s]\n", identity.SiteUser.ID)
			}
		}
	}

	// Display inherited from information
	if permission.InheritedFrom != nil {
		fmt.Printf("  Inherited From:\n")
		fmt.Printf("    Drive ID: %s\n", permission.InheritedFrom.DriveID)
		fmt.Printf("    Item ID: %s\n", permission.InheritedFrom.ID)
		if permission.InheritedFrom.Path != "" {
			fmt.Printf("    Path: %s\n", permission.InheritedFrom.Path)
		}
	}
}
