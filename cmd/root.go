// Package cmd (root.go) defines the root command for the onedrive-client CLI.
// It sets up global flags, persistent pre-run checks for authentication,
// and initializes subcommands.
package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	cmdItems "github.com/tonimelisma/onedrive-client/cmd/items"
	"github.com/tonimelisma/onedrive-client/internal/app"
	"github.com/tonimelisma/onedrive-client/pkg/onedrive"
)

// rootCmd represents the base command when called without any subcommands.
// It provides overall application description and handles global setup.
var rootCmd = &cobra.Command{
	Use:   "onedrive-client",
	Short: "A CLI client for Microsoft OneDrive",
	Long: `onedrive-client is a command-line interface tool to interact with Microsoft OneDrive.
It allows you to manage files, folders, and drives, perform uploads and downloads,
and manage authentication with your OneDrive account.

Current capabilities include:
  - Authentication management (login, logout, status)
  - Listing drives and checking quota
  - File and folder operations (list, stat, mkdir, upload, download, rm, mv, cp, rename)
  - Sharing and permissions management
  - Activity tracking and version history
  - Resumable uploads and downloads

This tool aims to provide a comprehensive set of command-line primitives for
manual and scripted interactions with OneDrive.`,
	// PersistentPreRunE is executed before any subcommand's RunE.
	// It's used here to ensure that most commands require authentication,
	// while exempting the 'auth' command group.
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Exempt 'auth' command and its subcommands (like 'auth login') from auth checks,
		// as these commands are used to establish authentication.
		if cmd.Parent() != nil && cmd.Parent().Name() == "auth" {
			return nil
		}

		// For all other commands, attempt to initialize the app.
		// app.NewApp() handles checking for existing tokens and completing pending logins.
		// This is a lightweight check; it doesn't mean every command needs a fully valid token
		// to *start* (e.g. 'items list' might proceed to tell you to log in), but it ensures
		// the auth state is evaluated.
		_, err := app.NewApp(cmd)
		if err != nil {
			// If a login is pending (e.g., user ran 'auth login' but hasn't visited the URL),
			// app.NewApp() returns ErrLoginPending with a user-friendly message.
			// We print this message and return the sentinel error.
			if errors.Is(err, app.ErrLoginPending) {
				fmt.Println(err.Error()) // The error itself contains the message like "Please go to..."
				// Return the specific sentinel error so Execute() can recognize it and
				// avoid printing a generic error message again.
				return app.ErrLoginPending
			}
			// If re-authentication is required (no valid token), this is not necessarily an error
			// at this global pre-run stage. Individual commands will handle this by prompting
			// the user to log in if they try an operation requiring auth.
			if errors.Is(err, onedrive.ErrReauthRequired) {
				return nil // Allow command to proceed; it will handle the reauth requirement.
			}
			// For other unexpected errors during app initialization, propagate them.
			return err
		}
		return nil
	},
	// Run is executed if `onedrive-client` is called without any subcommands.
	// It defaults to showing the help message.
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
// It handles errors from command execution, including special handling for ErrLoginPending.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		// If the error is ErrLoginPending, the PersistentPreRunE already printed
		// the user-friendly message. We suppress a duplicate generic error message here.
		if !errors.Is(err, app.ErrLoginPending) {
			// For all other errors, print them to stderr.
			// Cobra itself often prints errors, so this might be redundant in some cases,
			// but it ensures an error is always shown.
			fmt.Fprintln(os.Stderr, "Error:", err)
		}
		os.Exit(1)
	}
}

// init is called by Go when the package is initialized.
// It sets up global flags and registers subcommands.
func init() {
	// Define global persistent flags applicable to all commands.
	// Example: rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.onedrive-client.yaml)")
	rootCmd.PersistentFlags().Bool("debug", false, "Enable debug logging for SDK and internal operations")

	// Initialize and register the 'items' subcommand and its children.
	// This modular approach keeps subcommand definitions organized.
	cmdItems.InitItemsCommands(rootCmd)

	// Other top-level commands (like 'auth', 'drives') are added directly in their respective files' init functions.
}
