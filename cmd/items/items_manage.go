// Package items (items_manage.go) defines Cobra commands for managing and
// manipulating OneDrive items (files and folders). This includes operations like
// deleting (rm), copying (cp, copy-status), moving (mv), and renaming items.
package items

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/spf13/cobra"
	"github.com/tonimelisma/onedrive-client/internal/app"
	"github.com/tonimelisma/onedrive-client/internal/ui"
)

// filesRmCmd handles 'items rm <remote-path>'.
// It deletes a file or folder, moving it to the OneDrive recycle bin.
var filesRmCmd = &cobra.Command{
	Use:   "rm <remote-path>",
	Short: "Delete a file or folder (moves to recycle bin)",
	Long: `Deletes a specified file or folder from your OneDrive.
Items are moved to the OneDrive recycle bin and are not permanently deleted immediately.
To permanently delete, you would typically need to empty the recycle bin via the OneDrive web interface.`,
	Example: `onedrive-client items rm /Documents/OldReport.docx`,
	Args:    cobra.ExactArgs(1), // Requires exactly one argument: the remote path of the item to delete.
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("initializing app for 'items rm': %w", err)
		}
		return filesRmLogic(a, cmd, args)
	},
}

// filesCopyCmd handles 'items copy <source-path> <destination-parent-path> [new-name]'.
// It copies a file or folder to a new location. The operation is asynchronous by default.
var filesCopyCmd = &cobra.Command{
	Use:   "copy <source-path> <destination-parent-path> [new-name]",
	Short: "Copy a file or folder to a new location",
	Long: `Creates a copy of a specified file or folder in your OneDrive at the given destination parent path.
Optionally, a new name for the copied item can be provided.
The copy operation is asynchronous by default; the command returns a monitor URL.
Use the '--wait' flag to make the command block and poll until the copy operation completes or fails.
Use 'items copy-status <monitor-url>' to check the progress of an ongoing asynchronous copy.`,
	Example: `onedrive-client items copy /Photos/MyImage.jpg /Backup/Photos
onedrive-client items copy /Projects/Report.docx /Archive "Archived Report.docx"
onedrive-client items copy /LargeFolder /Backup/Folders --wait`,
	Args: cobra.RangeArgs(2, 3), // Requires source, destination parent, and optionally a new name.
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("initializing app for 'items copy': %w", err)
		}
		return filesCopyLogic(a, cmd, args)
	},
}

// filesCopyStatusCmd handles 'items copy-status <monitor-url>'.
// It checks and displays the status of an asynchronous copy operation using its monitor URL.
var filesCopyStatusCmd = &cobra.Command{
	Use:     "copy-status <monitor-url>",
	Short:   "Check the status of an asynchronous copy operation",
	Long:    `Monitors and displays the progress and status of an asynchronous copy operation using the monitor URL provided by the 'items copy' command (when not using --wait).`,
	Example: `onedrive-client items copy-status "https://graph.microsoft.com/v1.0/me/drive/operations/0123456789ABCDEF!1.0"`,
	Args:    cobra.ExactArgs(1), // Requires exactly one argument: the monitor URL.
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("initializing app for 'items copy-status': %w", err)
		}
		return filesCopyStatusLogic(a, cmd, args)
	},
}

// filesMvCmd handles 'items mv <source-path> <destination-parent-path>'.
// It moves a file or folder to a new location.
var filesMvCmd = &cobra.Command{
	Use:     "mv <source-path> <destination-parent-path>",
	Short:   "Move a file or folder to a new location",
	Long:    `Moves a specified file or folder from its current location to a new destination parent path within your OneDrive. If the item is moved to a different folder with the same name, it's effectively a move. If the name also changes as part of the destination path (not directly supported by this command's arguments but by the API), it's a move and rename.`,
	Example: `onedrive-client items mv /Temporary/File.txt /Documents/Archive`,
	Args:    cobra.ExactArgs(2), // Requires source path and destination parent path.
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("initializing app for 'items mv': %w", err)
		}
		return filesMvLogic(a, cmd, args)
	},
}

// filesRenameCmd handles 'items rename <current-path> <new-name>'.
// It renames a file or folder.
var filesRenameCmd = &cobra.Command{
	Use:     "rename <current-path> <new-name>",
	Short:   "Rename a file or folder",
	Long:    `Renames a specified file or folder in your OneDrive. Provide the current path to the item and its desired new name.`,
	Example: `onedrive-client items rename /Documents/OldReport.docx "Final Report Q1.docx"`,
	Args:    cobra.ExactArgs(2), // Requires current path and new name.
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("initializing app for 'items rename': %w", err)
		}
		return filesRenameLogic(a, cmd, args)
	},
}

// filesRmLogic contains the core logic for the 'items rm' command.
func filesRmLogic(a *app.App, cmd *cobra.Command, args []string) error {
	remotePath := args[0]
	if remotePath == "" { // Should be caught by Args validation.
		return fmt.Errorf("remote path for 'rm' cannot be empty")
	}

	err := a.SDK.DeleteDriveItem(cmd.Context(), remotePath)
	if err != nil {
		return fmt.Errorf("deleting item '%s': %w", remotePath, err)
	}
	ui.PrintSuccess("Item '%s' successfully moved to recycle bin.", remotePath)
	return nil
}

// filesCopyLogic contains the core logic for the 'items copy' command.
func filesCopyLogic(a *app.App, cmd *cobra.Command, args []string) error {
	sourcePath := args[0]
	destinationParentPath := args[1] // Corrected variable name for clarity
	var newName string
	if len(args) > 2 {
		newName = args[2]
	}

	if sourcePath == "" || destinationParentPath == "" { // Should be caught by Args validation.
		return fmt.Errorf("source and destination parent paths for 'copy' cannot be empty")
	}

	wait, _ := cmd.Flags().GetBool("wait")

	monitorURL, err := a.SDK.CopyDriveItem(cmd.Context(), sourcePath, destinationParentPath, newName)
	if err != nil {
		return fmt.Errorf("initiating copy of '%s' to '%s': %w", sourcePath, destinationParentPath, err)
	}

	if wait {
		log.Printf("Copy operation for '%s' started. Monitoring progress (this may take a while)...", sourcePath)
		return monitorCopyToCompletion(a, cmd.Context(), monitorURL, sourcePath)
	}
	// If not waiting, provide the monitor URL for manual status checking.
	log.Printf("Copy operation for '%s' started asynchronously.", sourcePath)
	log.Printf("  Monitor URL: %s", monitorURL)
	log.Printf("  Use 'onedrive-client items copy-status \"%s\"' to check progress.", monitorURL)
	return nil
}

// monitorCopyToCompletion polls the copy operation status until it completes or fails.
// `sourcePath` is used for more informative logging.
func monitorCopyToCompletion(a *app.App, ctx context.Context, monitorURL, sourcePath string) error {
	// Get polling configuration from app config
	pollingInterval := a.Config.Polling.InitialInterval
	maxInterval := a.Config.Polling.MaxInterval
	multiplier := a.Config.Polling.Multiplier

	// Polling loop for copy status.
	for {
		status, err := a.SDK.MonitorCopyOperation(ctx, monitorURL)
		if err != nil {
			return fmt.Errorf("monitoring copy operation for '%s' (URL: %s): %w", sourcePath, monitorURL, err)
		}

		switch status.Status {
		case "inProgress", "notStarted", "waiting": // Added notStarted and waiting as potential transient states
			log.Printf("Copy of '%s' in progress... Status: %s (%s, %d%%)", sourcePath, status.Status, status.StatusDescription, status.PercentageComplete)
		case "completed":
			log.Printf("Copy operation for '%s' completed successfully!", sourcePath)
			if status.ResourceID != "" { // Prefer ResourceID if available
				log.Printf("  New item ID: %s", status.ResourceID)
			} else if status.ResourceLocation != "" { // Fallback to deprecated ResourceLocation
				log.Printf("  New item location (URL): %s", status.ResourceLocation)
			}
			return nil
		case "failed":
			errMsg := status.StatusDescription
			if status.Error != nil {
				errMsg = fmt.Sprintf("Code: %s, Message: %s", status.Error.Code, status.Error.Message)
			}
			return fmt.Errorf("copy operation for '%s' failed: %s", sourcePath, errMsg)
		default:
			// Unknown status, log it and continue polling.
			log.Printf("Copy of '%s' has an unknown status: %s - %s (%d%%)", sourcePath, status.Status, status.StatusDescription, status.PercentageComplete)
		}

		// Wait for the current polling interval before checking again
		log.Printf("Waiting %v before next status check...", pollingInterval)
		time.Sleep(pollingInterval)

		// Increase polling interval for exponential backoff, but cap at maximum
		nextInterval := time.Duration(float64(pollingInterval) * multiplier)
		if nextInterval > maxInterval {
			pollingInterval = maxInterval
		} else {
			pollingInterval = nextInterval
		}
	}
}

// filesCopyStatusLogic contains the core logic for the 'items copy-status' command.
func filesCopyStatusLogic(a *app.App, cmd *cobra.Command, args []string) error {
	monitorURL := args[0]
	if monitorURL == "" { // Should be caught by Args validation.
		return fmt.Errorf("monitor URL for 'copy-status' cannot be empty")
	}

	status, err := a.SDK.MonitorCopyOperation(cmd.Context(), monitorURL)
	if err != nil {
		return fmt.Errorf("checking copy status from URL '%s': %w", monitorURL, err)
	}

	fmt.Printf("Copy Operation Status (from %s):\n", monitorURL)
	fmt.Printf("  Status:             %s\n", status.Status)
	fmt.Printf("  Description:        %s\n", status.StatusDescription)
	if status.PercentageComplete > 0 || status.Status == "inProgress" { // Show progress if available or in progress
		fmt.Printf("  Progress:           %d%%\n", status.PercentageComplete)
	}
	if status.ResourceID != "" {
		fmt.Printf("  New Item ID:        %s\n", status.ResourceID)
	} else if status.ResourceLocation != "" {
		fmt.Printf("  New Item Location:  %s (Note: This field is deprecated by Graph API)\n", status.ResourceLocation)
	}
	if status.Error != nil {
		fmt.Printf("  Error Code:         %s\n", status.Error.Code)
		fmt.Printf("  Error Message:      %s\n", status.Error.Message)
	}

	return nil
}

// filesMvLogic contains the core logic for the 'items mv' command.
func filesMvLogic(a *app.App, cmd *cobra.Command, args []string) error {
	sourcePath := args[0]
	destinationParentPath := args[1] // Corrected variable name for clarity

	if sourcePath == "" || destinationParentPath == "" { // Should be caught by Args validation.
		return fmt.Errorf("source and destination parent paths for 'mv' cannot be empty")
	}

	item, err := a.SDK.MoveDriveItem(cmd.Context(), sourcePath, destinationParentPath)
	if err != nil {
		return fmt.Errorf("moving item '%s' to '%s': %w", sourcePath, destinationParentPath, err)
	}

	ui.PrintSuccess("Item '%s' moved successfully to '%s'. New Item ID: %s", sourcePath, destinationParentPath, item.ID)
	return nil
}

// filesRenameLogic contains the core logic for the 'items rename' command.
func filesRenameLogic(a *app.App, cmd *cobra.Command, args []string) error {
	currentPath := args[0] // Renamed for clarity
	newName := args[1]

	if currentPath == "" || newName == "" { // Should be caught by Args validation.
		return fmt.Errorf("current path and new name for 'rename' cannot be empty")
	}

	item, err := a.SDK.UpdateDriveItem(cmd.Context(), currentPath, newName)
	if err != nil {
		return fmt.Errorf("renaming item '%s' to '%s': %w", currentPath, newName, err)
	}

	ui.PrintSuccess("Item '%s' renamed successfully to '%s'. New Item ID: %s", currentPath, item.Name, item.ID)
	return nil
}
