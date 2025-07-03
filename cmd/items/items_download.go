// Package items (items_download.go) defines Cobra commands related to
// downloading files from OneDrive. This includes the primary 'items download'
// command and any associated helper logic or deprecated download commands.
package items

import (
	"fmt"
	"log"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tonimelisma/onedrive-client/internal/app"
	"github.com/tonimelisma/onedrive-client/internal/ui"
)

// filesDownloadCmd handles 'items download <remote-path> [local-path]'.
// It downloads a file from OneDrive to the local filesystem.
// If local-path is omitted, it defaults to the filename in the current directory.
// Supports downloading in a specific format via the --format flag.
var filesDownloadCmd = &cobra.Command{
	Use:   "download <remote-path> [local-path]",
	Short: "Download a file from OneDrive to the local filesystem",
	Long: `Downloads a specified file from your OneDrive.
You must provide the remote path to the file.
Optionally, you can specify a local path where the file should be saved.
If the local path is omitted, the file will be saved in the current directory
with its original name.

Use the '--format' flag to download the file converted to a different format
(e.g., downloading a .docx file as .pdf). Supported formats depend on the
Microsoft Graph API capabilities.`,
	Example: `onedrive-client items download /Documents/MyReport.docx ./MyReport_local.docx
onedrive-client items download /Images/Photo.jpg
onedrive-client items download /Presentations/Deck.pptx --format pdf`,
	Args: cobra.RangeArgs(1, 2), // Requires 1 (remote-path) or 2 (remote-path, local-path) arguments.
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd) // Renamed 'app' to 'a'
		if err != nil {
			return fmt.Errorf("initializing app for 'items download': %w", err)
		}

		remotePath := args[0]

		// Determine the local path for saving the downloaded file.
		localPath := ""
		if len(args) > 1 {
			localPath = args[1] // Use provided local path.
		} else {
			// If no local path is provided, extract the filename from the remote path
			// and use it in the current directory.
			parts := strings.Split(remotePath, "/")
			if len(parts) > 0 {
				localPath = parts[len(parts)-1]
			} else {
				// Should not happen if remotePath is valid, but as a fallback.
				localPath = "downloaded_file"
			}
		}

		// Check if the --format flag is specified for format conversion.
		format, _ := cmd.Flags().GetString("format")
		if format != "" {
			// Download with format conversion.
			err = a.SDK.DownloadFileAsFormat(cmd.Context(), remotePath, localPath, format)
			if err != nil {
				return fmt.Errorf("downloading file '%s' as format '%s' to '%s': %w", remotePath, format, localPath, err)
			}
			log.Printf("Successfully downloaded '%s' as format '%s' to '%s'", remotePath, format, localPath)
		} else {
			// Standard download without format conversion.
			err = a.SDK.DownloadFile(cmd.Context(), remotePath, localPath)
			if err != nil {
				return fmt.Errorf("downloading file '%s' to '%s': %w", remotePath, localPath, err)
			}
			log.Printf("Successfully downloaded '%s' to '%s'", remotePath, localPath)
		}
		return nil
	},
}

// filesListRootDeprecatedCmd handles 'items list-root-deprecated'.
// This command is kept for backward compatibility but users are encouraged
// to use 'items list /' instead.
var filesListRootDeprecatedCmd = &cobra.Command{
	Use:   "list-root-deprecated",
	Short: "DEPRECATED: List items in the root of the default drive",
	Long:  `DEPRECATED: Lists items in the root directory of your default OneDrive drive. This command uses an older SDK method. Please use 'items list /' or 'items list' (with no path) instead for the same functionality with updated internal logic.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("initializing app for 'list-root-deprecated': %w", err)
		}
		// Explicitly warn the user about deprecation.
		log.Println("Warning: 'items list-root-deprecated' is deprecated. Please use 'items list /' or 'items list' instead.")
		return filesListRootDeprecatedLogic(a, cmd, args)
	},
}

// filesListRootDeprecatedLogic contains the core logic for the deprecated root listing command.
func filesListRootDeprecatedLogic(a *app.App, cmd *cobra.Command, args []string) error {
	items, err := a.SDK.GetRootDriveItems(cmd.Context())
	if err != nil {
		return fmt.Errorf("fetching root drive items (deprecated method): %w", err)
	}
	ui.DisplayItems(items)
	return nil
}
