package ui

import (
	"fmt"
	"log"
	"os"
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
