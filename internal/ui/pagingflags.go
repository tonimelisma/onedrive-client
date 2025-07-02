// Package ui (pagingflags.go) provides utility functions for consistently
// adding and parsing standard pagination flags (like --top, --all, --next)
// for Cobra commands that interact with paginated Microsoft Graph API endpoints.
// It also includes a helper to inform users about available next pages.
package ui

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tonimelisma/onedrive-client/pkg/onedrive"
)

const (
	// FlagTop is the name for the pagination flag limiting items per page.
	FlagTop = "top"
	// FlagAll is the name for the pagination flag to fetch all items across pages.
	FlagAll = "all"
	// FlagNext is the name for the pagination flag to resume from a specific nextLink URL.
	FlagNext = "next"
)

// AddPagingFlags adds a standardized set of pagination flags to a Cobra command.
// These flags are:
//   --top <int>:  Maximum number of items to return per page (0 means API default).
//   --all <bool>: If true, fetch all items across all available pages, ignoring --top.
//   --next <string>: A specific @odata.nextLink URL to resume pagination from.
//
// This ensures consistency in how users interact with pagination across different commands.
func AddPagingFlags(cmd *cobra.Command) {
	cmd.Flags().Int(FlagTop, 0, "Maximum number of items to return per page (0 for API default)")
	cmd.Flags().Bool(FlagAll, false, "Fetch all items across all pages (overrides --top)")
	cmd.Flags().String(FlagNext, "", "Resume pagination from this specific @odata.nextLink URL")
}

// ParsePagingFlags extracts pagination settings from the flags of a given Cobra command.
// It reads the values of flags added by AddPagingFlags and populates an
// onedrive.Paging struct, which can then be passed to SDK methods.
//
// Returns an onedrive.Paging struct and any error encountered while parsing flags.
func ParsePagingFlags(cmd *cobra.Command) (onedrive.Paging, error) {
	top, err := cmd.Flags().GetInt(FlagTop)
	if err != nil {
		return onedrive.Paging{}, fmt.Errorf("parsing '--%s' flag: %w", FlagTop, err)
	}

	all, err := cmd.Flags().GetBool(FlagAll)
	if err != nil {
		return onedrive.Paging{}, fmt.Errorf("parsing '--%s' flag: %w", FlagAll, err)
	}

	next, err := cmd.Flags().GetString(FlagNext)
	if err != nil {
		return onedrive.Paging{}, fmt.Errorf("parsing '--%s' flag: %w", FlagNext, err)
	}

	// Construct the Paging struct for the SDK.
	// The SDK's collectAllPages function will interpret these values:
	// - If FetchAll is true, it will retrieve all pages.
	// - If NextLink is provided, it starts from there.
	// - Top is used if neither FetchAll nor NextLink (for specific resume) are dominant.
	return onedrive.Paging{
		Top:      top,
		FetchAll: all,
		NextLink: next,
	}, nil
}

// HandleNextPageInfo displays a message to the user if a next page of results
// is available and not all items were fetched (i.e., --all was not used).
// `nextLink` is the @odata.nextLink URL from the API response.
// `fetchAll` indicates if the --all flag was active for the current command.
//
// This provides a user-friendly way to continue pagination.
func HandleNextPageInfo(nextLink string, fetchAll bool) {
	// Only show next page info if there is a nextLink and the user wasn't trying to fetch all pages.
	// If fetchAll was true, the SDK's collectAllPages should have already retrieved everything.
	if nextLink != "" && !fetchAll {
		fmt.Printf("\nMore items available on the next page.\nTo continue, use the '--%s' flag with the following URL:\n  --%s \"%s\"\n", FlagNext, FlagNext, nextLink)
	}
}
