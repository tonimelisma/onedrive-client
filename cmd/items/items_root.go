// Package items (items_root.go) defines the primary 'items' subcommand and
// initializes all its child commands related to file and folder operations.
// This file acts as the entry point for the 'items' command hierarchy.
package items

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tonimelisma/onedrive-client/internal/ui"
)

// ItemsCmd represents the base 'items' command.
// It groups all subcommands for managing DriveItems (files and folders).
var ItemsCmd = &cobra.Command{
	Use:   "items",
	Short: "Manage OneDrive files and folders (DriveItems)",
	Long: `Provides a comprehensive set of subcommands to interact with files and folders
in your OneDrive. This includes listing, getting metadata (stat), creating folders (mkdir),
uploading, downloading, deleting (rm), copying (cp), moving (mv), renaming, searching,
managing sharing links and permissions, viewing versions, activities, thumbnails, and previews.`,
	// Example: onedrive-client items list /Documents
	// Example: onedrive-client items upload ./localfile.txt /Backup
}

// InitItemsCommands registers all 'items' subcommands and their flags to the provided root command.
// This function is called from the root command's init() to build the command tree.
func InitItemsCommands(rootCmd *cobra.Command) {
	// Register the main 'items' command to the root command.
	rootCmd.AddCommand(ItemsCmd)

	// Add all item-related subcommands to the ItemsCmd.
	// These are defined in other files within this package (e.g., items_meta.go, items_upload.go).
	ItemsCmd.AddCommand(filesListCmd)     // items list
	ItemsCmd.AddCommand(filesStatCmd)     // items stat
	ItemsCmd.AddCommand(filesMkdirCmd)    // items mkdir
	ItemsCmd.AddCommand(filesUploadCmd)   // items upload
	ItemsCmd.AddCommand(filesDownloadCmd) // items download
	ItemsCmd.AddCommand(filesCancelUploadCmd)
	ItemsCmd.AddCommand(filesGetUploadStatusCmd)
	ItemsCmd.AddCommand(filesUploadSimpleCmd)
	ItemsCmd.AddCommand(filesListRootDeprecatedCmd)
	ItemsCmd.AddCommand(filesRmCmd)   // items rm
	ItemsCmd.AddCommand(filesCopyCmd) // items copy
	ItemsCmd.AddCommand(filesCopyStatusCmd)
	ItemsCmd.AddCommand(filesMvCmd)       // items mv
	ItemsCmd.AddCommand(filesRenameCmd)   // items rename
	ItemsCmd.AddCommand(filesSearchCmd)   // items search
	ItemsCmd.AddCommand(filesShareCmd)    // items share
	ItemsCmd.AddCommand(filesVersionsCmd) // items versions
	ItemsCmd.AddCommand(activitiesCmd)    // items activities (item-specific)
	ItemsCmd.AddCommand(filesThumbnailsCmd)
	ItemsCmd.AddCommand(filesPreviewCmd)
	ItemsCmd.AddCommand(filesInviteCmd)      // items invite
	ItemsCmd.AddCommand(filesPermissionsCmd) // items permissions (parent for list, get, update, delete)

	// Add subcommands to the 'items permissions' command.
	filesPermissionsCmd.AddCommand(filesPermissionsListCmd)
	filesPermissionsCmd.AddCommand(filesPermissionsGetCmd)
	filesPermissionsCmd.AddCommand(filesPermissionsUpdateCmd)
	filesPermissionsCmd.AddCommand(filesPermissionsDeleteCmd)

	// --- Define flags for specific 'items' subcommands ---

	// Flags for 'items copy':
	// --wait: Blocks the command until the copy operation completes, rather than returning immediately.
	filesCopyCmd.Flags().Bool("wait", false, "Wait for copy operation to complete instead of returning immediately")

	// Flags for 'items download':
	// --format: Allows specifying a format for downloading a file (e.g., "pdf" for a docx file).
	filesDownloadCmd.Flags().String("format", "", "Download file in a specific format (e.g., pdf, jpg)")

	// Flags for 'items search':
	// --in: Specifies the folder path to search within. This is mandatory for 'items search'.
	// For drive-wide search, 'drives search' should be used.
	filesSearchCmd.Flags().String("in", "", "Folder path to search within (required for 'items search')")
	if err := filesSearchCmd.MarkFlagRequired("in"); err != nil {
		// This would be a programmatic error if MarkFlagRequired fails.
		// Handle appropriately, perhaps by panic or logging a fatal error,
		// as it indicates a problem with command setup.
		panic(fmt.Sprintf("failed to mark '--in' flag as required for 'items search': %v", err))
	}

	// Flags for 'items preview':
	// --page: Specifies a page number or name for document previews.
	// --zoom: Sets the zoom level for the preview.
	filesPreviewCmd.Flags().String("page", "", "Page number or name to preview (for multi-page documents)")
	filesPreviewCmd.Flags().Float64("zoom", 1.0, "Zoom level for preview (e.g., 1.0 for 100%, 0.5 for 50%)")

	// Flags for 'items invite':
	// These flags configure the properties of the invitation sent to users.
	filesInviteCmd.Flags().String("message", "", "Optional custom message to include in the invitation email")
	filesInviteCmd.Flags().StringSlice("roles", []string{"read"}, "Permission roles to grant (e.g., 'read', 'write')")
	filesInviteCmd.Flags().Bool("require-signin", true, "Specify if the invited user must sign in to access the item")
	filesInviteCmd.Flags().Bool("send-invitation", true, "Specify if an email invitation should be sent to the recipients")

	// Flags for 'items permissions update':
	// These flags allow modification of an existing permission's properties.
	filesPermissionsUpdateCmd.Flags().StringSlice("roles", nil, "New roles to set for the permission (e.g., 'read', 'write')")
	filesPermissionsUpdateCmd.Flags().String("expiration", "", "New expiration date/time in ISO 8601 format (e.g., 'YYYY-MM-DDTHH:MM:SSZ')")
	filesPermissionsUpdateCmd.Flags().String("password", "", "Set or change the password for a link-based permission")

	// Add standard pagination flags (--top, --all, --next) to commands that support them.
	// This is handled by a utility function for consistency.
	ui.AddPagingFlags(activitiesCmd)  // For item activities
	ui.AddPagingFlags(filesSearchCmd) // For item search results
	// Other commands like 'items list' might also benefit from pagination if the SDK supports it for GetDriveItemChildrenByPath.
	// ui.AddPagingFlags(filesListCmd) // Example if 'items list' were to support pagination.
}
