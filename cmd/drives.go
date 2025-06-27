package cmd

import (
	"log"

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
	Run: func(cmd *cobra.Command, args []string) {
		a, err := app.NewApp()
		if err != nil {
			log.Fatalf("Error creating app: %v", err)
		}
		if err := drivesListLogic(a); err != nil {
			log.Fatalf("Error: %v", err)
		}
	},
}

var drivesQuotaCmd = &cobra.Command{
	Use:   "quota",
	Short: "Get storage quota for the default drive",
	Long:  "Displays the total, used, and remaining storage quota for the default drive.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		a, err := app.NewApp()
		if err != nil {
			log.Fatalf("Error creating app: %v", err)
		}
		if err := drivesQuotaLogic(a); err != nil {
			log.Fatalf("Error: %v", err)
		}
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

func init() {
	rootCmd.AddCommand(drivesCmd)
	drivesCmd.AddCommand(drivesListCmd)
	drivesCmd.AddCommand(drivesQuotaCmd)
}
