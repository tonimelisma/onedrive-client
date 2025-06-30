package cmd

import (
	"github.com/spf13/cobra"
	"github.com/tonimelisma/onedrive-client/internal/ui"
)

var ItemsCmd = &cobra.Command{
	Use:   "items",
	Short: "Manage items (files and folders)",
	Long:  "Provides commands to list, stat, upload, download, and manage OneDrive items (files and folders).",
}

func InitItemsCommands(rootCmd *cobra.Command) {
	// Register the main items command to root
	rootCmd.AddCommand(ItemsCmd)

	// Add all subcommands
	ItemsCmd.AddCommand(filesListCmd)
	ItemsCmd.AddCommand(filesStatCmd)
	ItemsCmd.AddCommand(filesMkdirCmd)
	ItemsCmd.AddCommand(filesUploadCmd)
	ItemsCmd.AddCommand(filesDownloadCmd)
	ItemsCmd.AddCommand(filesCancelUploadCmd)
	ItemsCmd.AddCommand(filesGetUploadStatusCmd)
	ItemsCmd.AddCommand(filesUploadSimpleCmd)
	ItemsCmd.AddCommand(filesListRootDeprecatedCmd)
	ItemsCmd.AddCommand(filesRmCmd)
	ItemsCmd.AddCommand(filesCopyCmd)
	ItemsCmd.AddCommand(filesCopyStatusCmd)
	ItemsCmd.AddCommand(filesMvCmd)
	ItemsCmd.AddCommand(filesRenameCmd)
	ItemsCmd.AddCommand(filesSearchCmd)
	ItemsCmd.AddCommand(filesRecentCmd)
	ItemsCmd.AddCommand(filesSpecialCmd)
	ItemsCmd.AddCommand(filesShareCmd)
	ItemsCmd.AddCommand(filesVersionsCmd)
	ItemsCmd.AddCommand(activitiesCmd)
	ItemsCmd.AddCommand(filesThumbnailsCmd)
	ItemsCmd.AddCommand(filesPreviewCmd)
	ItemsCmd.AddCommand(filesInviteCmd)
	ItemsCmd.AddCommand(filesPermissionsCmd)

	// Add permissions subcommands
	filesPermissionsCmd.AddCommand(filesPermissionsListCmd)
	filesPermissionsCmd.AddCommand(filesPermissionsGetCmd)
	filesPermissionsCmd.AddCommand(filesPermissionsUpdateCmd)
	filesPermissionsCmd.AddCommand(filesPermissionsDeleteCmd)

	// Add flags
	filesCopyCmd.Flags().Bool("wait", false, "Wait for copy operation to complete instead of returning immediately")

	// Add download format flag
	filesDownloadCmd.Flags().String("format", "", "Download file in specific format (e.g., pdf, docx)")

	// Add search flags
	filesSearchCmd.Flags().String("in", "", "Search within a specific folder")

	// Add flags for new Epic 7 commands
	filesPreviewCmd.Flags().String("page", "", "Page number or name to preview")
	filesPreviewCmd.Flags().Float64("zoom", 1.0, "Zoom level (1.0 = 100%)")

	filesInviteCmd.Flags().String("message", "", "Optional invitation message")
	filesInviteCmd.Flags().StringSlice("roles", []string{"read"}, "Roles to grant (read, write)")
	filesInviteCmd.Flags().Bool("require-signin", true, "Whether sign-in is required")
	filesInviteCmd.Flags().Bool("send-invitation", true, "Whether to send email invitation")

	filesPermissionsUpdateCmd.Flags().StringSlice("roles", nil, "Roles to set (read, write, owner)")
	filesPermissionsUpdateCmd.Flags().String("expiration", "", "Expiration date/time (ISO 8601 format)")
	filesPermissionsUpdateCmd.Flags().String("password", "", "Optional password")

	// Add pagination flags for commands that need them
	ui.AddPagingFlags(activitiesCmd)
	ui.AddPagingFlags(filesSearchCmd)
}
