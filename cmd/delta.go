package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	cmdItems "github.com/tonimelisma/onedrive-client/cmd/items"
	"github.com/tonimelisma/onedrive-client/internal/app"
	"github.com/tonimelisma/onedrive-client/internal/ui"
)

var deltaCmd = &cobra.Command{
	Use:   "delta [delta-token]",
	Short: "Track changes in the drive using delta queries",
	Long:  "Track changes to items in your OneDrive using delta queries. Provides efficient synchronization by only returning changed items.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("error creating app: %w", err)
		}
		return deltaLogic(a, args)
	},
}

func deltaLogic(a *app.App, args []string) error {
	var deltaToken string
	if len(args) > 0 {
		deltaToken = args[0]
	}

	delta, err := a.SDK.GetDelta(deltaToken)
	if err != nil {
		return err
	}

	ui.DisplayDelta(delta)
	return nil
}

func init() {
	cmdItems.ItemsCmd.AddCommand(deltaCmd)
}
