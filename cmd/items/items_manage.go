package cmd

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/spf13/cobra"
	"github.com/tonimelisma/onedrive-client/internal/app"
	"github.com/tonimelisma/onedrive-client/internal/ui"
)

var filesRmCmd = &cobra.Command{
	Use:   "rm <remote-path>",
	Short: "Delete a file or folder",
	Long:  "Deletes a file or folder from your OneDrive. Items are moved to the recycle bin, not permanently deleted.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("error creating app: %w", err)
		}
		return filesRmLogic(a, cmd, args)
	},
}

var filesCopyCmd = &cobra.Command{
	Use:   "copy <source-path> <destination-path> [new-name]",
	Short: "Copy a file or folder",
	Long:  "Creates a copy of a file or folder in OneDrive. The copy operation is asynchronous by default. Use --wait to block until completion.",
	Args:  cobra.RangeArgs(2, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("error creating app: %w", err)
		}
		return filesCopyLogic(a, cmd, args)
	},
}

var filesCopyStatusCmd = &cobra.Command{
	Use:   "copy-status <monitor-url>",
	Short: "Check the status of a copy operation",
	Long:  "Monitors the progress and status of an asynchronous copy operation using its monitor URL.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("error creating app: %w", err)
		}
		return filesCopyStatusLogic(a, cmd, args)
	},
}

var filesMvCmd = &cobra.Command{
	Use:   "mv <source-path> <destination-path>",
	Short: "Move a file or folder",
	Long:  "Moves a file or folder from one location to another in your OneDrive.",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("error creating app: %w", err)
		}
		return filesMvLogic(a, cmd, args)
	},
}

var filesRenameCmd = &cobra.Command{
	Use:   "rename <remote-path> <new-name>",
	Short: "Rename a file or folder",
	Long:  "Renames a file or folder in your OneDrive.",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("error creating app: %w", err)
		}
		return filesRenameLogic(a, cmd, args)
	},
}

func filesRmLogic(a *app.App, cmd *cobra.Command, args []string) error {
	remotePath := args[0]
	if remotePath == "" {
		return fmt.Errorf("remote path cannot be empty")
	}

	err := a.SDK.DeleteDriveItem(cmd.Context(), remotePath)
	if err != nil {
		return err
	}
	ui.PrintSuccess("Item '%s' deleted successfully (moved to recycle bin).\n", remotePath)
	return nil
}

func filesCopyLogic(a *app.App, cmd *cobra.Command, args []string) error {
	sourcePath := args[0]
	destinationPath := args[1]
	var newName string
	if len(args) > 2 {
		newName = args[2]
	}

	if sourcePath == "" || destinationPath == "" {
		return fmt.Errorf("source and destination paths cannot be empty")
	}

	wait, _ := cmd.Flags().GetBool("wait")

	monitorURL, err := a.SDK.CopyDriveItem(cmd.Context(), sourcePath, destinationPath, newName)
	if err != nil {
		return fmt.Errorf("initiating copy operation: %w", err)
	}

	if wait {
		log.Printf("Copy operation started. Monitoring progress...")
		return monitorCopyToCompletion(a, cmd.Context(), monitorURL)
	} else {
		log.Printf("Copy operation started.")
		log.Printf("Monitor URL: %s", monitorURL)
		log.Printf("Use 'items copy-status %s' to check progress.", monitorURL)
		return nil
	}
}

func monitorCopyToCompletion(a *app.App, ctx context.Context, monitorURL string) error {
	for {
		status, err := a.SDK.MonitorCopyOperation(ctx, monitorURL)
		if err != nil {
			return fmt.Errorf("monitoring copy operation: %w", err)
		}

		switch status.Status {
		case "inProgress":
			log.Printf("Copy in progress... %s", status.StatusDescription)
		case "completed":
			log.Printf("Copy operation completed successfully!")
			if status.ResourceLocation != "" {
				log.Printf("New item location: %s", status.ResourceLocation)
			}
			return nil
		case "failed":
			return fmt.Errorf("copy operation failed: %s", status.StatusDescription)
		default:
			log.Printf("Copy status: %s - %s", status.Status, status.StatusDescription)
		}

		// Wait before next check
		time.Sleep(2 * time.Second)
	}
}

func filesCopyStatusLogic(a *app.App, cmd *cobra.Command, args []string) error {
	monitorURL := args[0]
	if monitorURL == "" {
		return fmt.Errorf("monitor URL cannot be empty")
	}

	status, err := a.SDK.MonitorCopyOperation(cmd.Context(), monitorURL)
	if err != nil {
		return fmt.Errorf("checking copy status: %w", err)
	}

	fmt.Printf("Copy Operation Status:\n")
	fmt.Printf("  Status: %s\n", status.Status)
	fmt.Printf("  Description: %s\n", status.StatusDescription)
	if status.PercentageComplete > 0 {
		fmt.Printf("  Progress: %d%%\n", status.PercentageComplete)
	}
	if status.ResourceLocation != "" {
		fmt.Printf("  Resource Location: %s\n", status.ResourceLocation)
	}

	return nil
}

func filesMvLogic(a *app.App, cmd *cobra.Command, args []string) error {
	sourcePath := args[0]
	destinationPath := args[1]

	if sourcePath == "" || destinationPath == "" {
		return fmt.Errorf("source and destination paths cannot be empty")
	}

	item, err := a.SDK.MoveDriveItem(cmd.Context(), sourcePath, destinationPath)
	if err != nil {
		return fmt.Errorf("moving item: %w", err)
	}

	ui.PrintSuccess("Item moved successfully. New ID: %s\n", item.ID)
	return nil
}

func filesRenameLogic(a *app.App, cmd *cobra.Command, args []string) error {
	remotePath := args[0]
	newName := args[1]

	if remotePath == "" || newName == "" {
		return fmt.Errorf("remote path and new name cannot be empty")
	}

	item, err := a.SDK.UpdateDriveItem(cmd.Context(), remotePath, newName)
	if err != nil {
		return fmt.Errorf("renaming item: %w", err)
	}

	ui.PrintSuccess("Item renamed successfully to '%s'. ID: %s\n", item.Name, item.ID)
	return nil
}
