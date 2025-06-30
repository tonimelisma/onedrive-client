package ui

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tonimelisma/onedrive-client/pkg/onedrive"
)

// AddPagingFlags adds the standard pagination flags to a command.
func AddPagingFlags(cmd *cobra.Command) {
	cmd.Flags().Int("top", 0, "Maximum number of items to return")
	cmd.Flags().Bool("all", false, "Fetch all items across all pages")
	cmd.Flags().String("next", "", "Continue from this next link URL")
}

// ParsePagingFlags extracts pagination settings from command flags.
func ParsePagingFlags(cmd *cobra.Command) (onedrive.Paging, error) {
	top, err := cmd.Flags().GetInt("top")
	if err != nil {
		return onedrive.Paging{}, fmt.Errorf("error parsing top flag: %w", err)
	}

	all, err := cmd.Flags().GetBool("all")
	if err != nil {
		return onedrive.Paging{}, fmt.Errorf("error parsing all flag: %w", err)
	}

	next, err := cmd.Flags().GetString("next")
	if err != nil {
		return onedrive.Paging{}, fmt.Errorf("error parsing next flag: %w", err)
	}

	return onedrive.Paging{
		Top:      top,
		FetchAll: all,
		NextLink: next,
	}, nil
}

// HandleNextPageInfo displays next page information if available.
func HandleNextPageInfo(nextLink string, fetchAll bool) {
	if nextLink != "" && !fetchAll {
		fmt.Printf("\nNext page available. Use --next '%s' to continue.\n", nextLink)
	}
}
