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

	fmt.Printf("%-25s %-15s %-25s %s\n", "Drive Name", "Drive Type", "Owner", "Quota")
	fmt.Println(strings.Repeat("-", 80))
	for _, drive := range drives.Value {
		ownerName := "N/A"
		if drive.Owner.User.DisplayName != "" {
			ownerName = drive.Owner.User.DisplayName
		}
		quotaStr := fmt.Sprintf("%s / %s", formatBytes(drive.Quota.Used), formatBytes(drive.Quota.Total))
		fmt.Printf("%-25s %-15s %-25s %s\n", drive.Name, drive.DriveType, ownerName, quotaStr)
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
