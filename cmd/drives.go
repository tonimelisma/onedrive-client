package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tonimelisma/onedrive-client/internal/app"
	"github.com/tonimelisma/onedrive-client/internal/ui"
)

var drivesCmd = &cobra.Command{
	Use:   "drives",
	Short: "Manage available drives",
	Long:  "Provides commands to list available OneDrive drives.",
}

var drivesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available drives",
	Long:  "Lists all available OneDrive drives for the current user.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("error creating app: %w", err)
		}
		return drivesListLogic(a)
	},
}

var drivesQuotaCmd = &cobra.Command{
	Use:   "quota",
	Short: "Get storage quota for the default drive",
	Long:  "Displays the total, used, and remaining storage quota for the default drive.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("error creating app: %w", err)
		}
		return drivesQuotaLogic(a)
	},
}

var drivesGetCmd = &cobra.Command{
	Use:   "get <drive-id>",
	Short: "Get metadata for a specific drive by ID",
	Long:  "Retrieves detailed metadata for a specific OneDrive drive using its ID.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("error creating app: %w", err)
		}
		return drivesGetLogic(a, args[0])
	},
}

var drivesActivitiesCmd = &cobra.Command{
	Use:   "activities",
	Short: "Show activity history for the entire drive",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		app, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("initializing app: %w", err)
		}

		// Parse paging options
		paging, err := ui.ParsePagingFlags(cmd)
		if err != nil {
			return fmt.Errorf("parsing pagination flags: %w", err)
		}

		activities, nextLink, err := app.SDK.GetDriveActivities(paging)
		if err != nil {
			return fmt.Errorf("getting drive activities: %w", err)
		}

		ui.DisplayActivities(activities, "drive")
		ui.HandleNextPageInfo(nextLink, paging.FetchAll)

		return nil
	},
}

func drivesListLogic(a *app.App) error {
	drives, err := a.SDK.GetDrives()
	if err != nil {
		return err
	}
	ui.DisplayDrives(drives)
	return nil
}

func drivesQuotaLogic(a *app.App) error {
	drive, err := a.SDK.GetDefaultDrive()
	if err != nil {
		return err
	}
	ui.DisplayQuota(drive)
	return nil
}

func drivesGetLogic(a *app.App, driveID string) error {
	drive, err := a.SDK.GetDriveByID(driveID)
	if err != nil {
		return err
	}
	ui.DisplayDrive(drive)
	return nil
}

func init() {
	rootCmd.AddCommand(drivesCmd)
	drivesCmd.AddCommand(drivesListCmd)
	drivesCmd.AddCommand(drivesQuotaCmd)
	drivesCmd.AddCommand(drivesGetCmd)
	drivesCmd.AddCommand(drivesActivitiesCmd)

	ui.AddPagingFlags(drivesActivitiesCmd)
}
