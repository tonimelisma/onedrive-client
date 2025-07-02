// Package items (items_permissions.go) defines Cobra commands for managing
// sharing links and permissions on OneDrive items (files and folders).
// This includes creating sharing links ('items share'), inviting users ('items invite'),
// and managing specific permissions ('items permissions list/get/update/delete').
package items

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tonimelisma/onedrive-client/internal/app"
	"github.com/tonimelisma/onedrive-client/internal/ui"
	"github.com/tonimelisma/onedrive-client/pkg/onedrive"
)

// filesShareCmd handles 'items share <remote-path> <link-type> <scope>'.
// It creates a new sharing link for a specified file or folder.
var filesShareCmd = &cobra.Command{
	Use:   "share <remote-path> <link-type> <scope>",
	Short: "Create a sharing link for a file or folder",
	Long: `Creates a new sharing link for a specified OneDrive file or folder.
Link types can be 'view' (read-only), 'edit' (read-write), or 'embed' (for embedding in web pages).
Scopes define who can use the link: 'anonymous' (anyone with the link) or 'organization' (only members of your organization).`,
	Example: `onedrive-client items share /Documents/Report.docx view anonymous
onedrive-client items share /Projects/TeamFolder edit organization`,
	Args: cobra.ExactArgs(3), // Requires remote-path, link-type, and scope.
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("initializing app for 'items share': %w", err)
		}
		return filesShareLogic(a, cmd, args)
	},
}

// filesInviteCmd handles 'items invite <remote-path> <email> [additional-emails...]'.
// It invites one or more users to access a file or folder with specified permissions.
var filesInviteCmd = &cobra.Command{
	Use:   "invite <remote-path> <email-address> [additional-email-addresses...]",
	Short: "Invite users to access a file or folder",
	Long: `Invites one or more users (via their email addresses) to access a specified OneDrive file or folder.
You can configure the permission roles (e.g., 'read', 'write'), whether an email invitation is sent,
if sign-in is required, and an optional custom message for the invitation.`,
	Example: `onedrive-client items invite /Shared/Doc.docx user1@example.com --roles write --message "Please review this document."
onedrive-client items invite /TeamSite/Folder user1@example.com user2@example.com --roles read --send-invitation=false`,
	Args: cobra.MinimumNArgs(2), // Requires remote-path and at least one email address.
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("initializing app for 'items invite': %w", err)
		}
		return filesInviteLogic(a, cmd, args)
	},
}

// filesPermissionsCmd is the parent command for managing specific permissions (list, get, update, delete).
// It doesn't have its own RunE function as it serves as a grouper.
var filesPermissionsCmd = &cobra.Command{
	Use:   "permissions",
	Short: "Manage granular permissions on files and folders",
	Long:  `Provides subcommands to list, get detailed information about, update, or delete specific permissions associated with a OneDrive file or folder.`,
}

// filesPermissionsListCmd handles 'items permissions list <remote-path>'.
// It lists all existing permissions for a file or folder.
var filesPermissionsListCmd = &cobra.Command{
	Use:     "list <remote-path>",
	Short:   "List all permissions on a file or folder",
	Long:    `Lists all existing permissions (both direct and link-based) for a specified OneDrive file or folder.`,
	Example: `onedrive-client items permissions list /Documents/Confidential.pdf`,
	Args:    cobra.ExactArgs(1), // Requires the remote path.
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("initializing app for 'items permissions list': %w", err)
		}
		return filesPermissionsListLogic(a, cmd, args)
	},
}

// filesPermissionsGetCmd handles 'items permissions get <remote-path> <permission-id>'.
// It retrieves detailed information about a single, specific permission.
var filesPermissionsGetCmd = &cobra.Command{
	Use:     "get <remote-path> <permission-id>",
	Short:   "Get detailed information about a specific permission",
	Long:    `Retrieves and displays detailed information about a single, specific permission on a OneDrive file or folder, identified by its unique permission ID.`,
	Example: `onedrive-client items permissions get /Documents/Report.docx "perm_id_string"`,
	Args:    cobra.ExactArgs(2), // Requires remote path and permission ID.
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("initializing app for 'items permissions get': %w", err)
		}
		return filesPermissionsGetLogic(a, cmd, args)
	},
}

// filesPermissionsUpdateCmd handles 'items permissions update <remote-path> <permission-id>'.
// It updates properties of an existing permission (e.g., roles, expiration).
var filesPermissionsUpdateCmd = &cobra.Command{
	Use:   "update <remote-path> <permission-id>",
	Short: "Update an existing permission's roles, expiration, or password",
	Long: `Updates properties of an existing permission on a OneDrive file or folder.
You can modify roles (e.g., from 'read' to 'write'), set or change an expiration date/time,
or manage a password for link-based permissions using the available flags.`,
	Example: `onedrive-client items permissions update /Shared/Doc.docx "perm_id" --roles write
onedrive-client items permissions update /Collaboration/Link "link_id" --expiration "2024-12-31T23:59:59Z"`,
	Args: cobra.ExactArgs(2), // Requires remote path and permission ID.
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("initializing app for 'items permissions update': %w", err)
		}
		return filesPermissionsUpdateLogic(a, cmd, args)
	},
}

// filesPermissionsDeleteCmd handles 'items permissions delete <remote-path> <permission-id>'.
// It removes/revokes a specific permission from a file or folder.
var filesPermissionsDeleteCmd = &cobra.Command{
	Use:     "delete <remote-path> <permission-id>",
	Short:   "Remove (revoke) a specific permission from a file or folder",
	Long:    `Removes (revokes) a specific permission from a OneDrive file or folder, identified by its unique permission ID. This action can remove direct user access or invalidate a sharing link.`,
	Example: `onedrive-client items permissions delete /Documents/OldShare.docx "perm_id_to_revoke"`,
	Args:    cobra.ExactArgs(2), // Requires remote path and permission ID.
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("initializing app for 'items permissions delete': %w", err)
		}
		return filesPermissionsDeleteLogic(a, cmd, args)
	},
}

// filesShareLogic contains the core logic for the 'items share' command.
func filesShareLogic(a *app.App, cmd *cobra.Command, args []string) error {
	remotePath := args[0]
	linkType := args[1]
	scope := args[2]

	// Basic validation, though Cobra's Args should catch count issues.
	if remotePath == "" || linkType == "" || scope == "" {
		return fmt.Errorf("remote path, link type, and scope cannot be empty")
	}

	// Validate link type to provide user-friendly error before API call.
	validLinkTypes := map[string]bool{"view": true, "edit": true, "embed": true}
	if !validLinkTypes[linkType] {
		return fmt.Errorf("invalid link type '%s'. Valid types are: view, edit, embed", linkType)
	}

	// Validate scope.
	validScopes := map[string]bool{"anonymous": true, "organization": true}
	if !validScopes[scope] {
		return fmt.Errorf("invalid scope '%s'. Valid scopes are: anonymous, organization", scope)
	}

	link, err := a.SDK.CreateSharingLink(cmd.Context(), remotePath, linkType, scope)
	if err != nil {
		return fmt.Errorf("creating sharing link for '%s': %w", remotePath, err)
	}

	ui.DisplaySharingLink(link)
	return nil
}

// filesInviteLogic contains the core logic for the 'items invite' command.
func filesInviteLogic(a *app.App, cmd *cobra.Command, args []string) error {
	remotePath := args[0]
	if remotePath == "" { // Should be caught by Args validation.
		return fmt.Errorf("remote path for 'invite' cannot be empty")
	}

	// Email addresses start from the second argument.
	emails := args[1:]
	if len(emails) == 0 { // Should be caught by Args validation.
		return fmt.Errorf("at least one email address is required for 'invite'")
	}

	// Parse optional flags for invitation properties.
	message, _ := cmd.Flags().GetString("message")
	roles, _ := cmd.Flags().GetStringSlice("roles")             // Default is ["read"] set in items_root.go
	requireSignIn, _ := cmd.Flags().GetBool("require-signin")   // Default is true
	sendInvitation, _ := cmd.Flags().GetBool("send-invitation") // Default is true

	// Construct the list of recipients for the SDK request.
	recipients := make([]struct {
		Email    string `json:"email,omitempty"`
		ObjectID string `json:"objectId,omitempty"`
	}, len(emails))
	for i, email := range emails {
		// Assuming emails are provided. If object IDs were supported, logic would be needed here.
		recipients[i].Email = email
	}

	request := onedrive.InviteRequest{
		Recipients:     recipients,
		Message:        message,
		RequireSignIn:  requireSignIn,
		SendInvitation: sendInvitation,
		Roles:          roles,
		// ExpirationDateTime and Password could also be added as flags if desired.
	}

	response, err := a.SDK.InviteUsers(cmd.Context(), remotePath, request)
	if err != nil {
		return fmt.Errorf("inviting users to '%s': %w", remotePath, err)
	}

	ui.DisplayInviteResponse(response, remotePath)
	return nil
}

// filesPermissionsListLogic contains the core logic for 'items permissions list'.
func filesPermissionsListLogic(a *app.App, cmd *cobra.Command, args []string) error {
	remotePath := args[0]
	if remotePath == "" { // Should be caught by Args validation.
		return fmt.Errorf("remote path for 'permissions list' cannot be empty")
	}

	permissions, err := a.SDK.ListPermissions(cmd.Context(), remotePath)
	if err != nil {
		return fmt.Errorf("listing permissions for '%s': %w", remotePath, err)
	}

	ui.DisplayPermissions(permissions, remotePath)
	return nil
}

// filesPermissionsGetLogic contains the core logic for 'items permissions get'.
func filesPermissionsGetLogic(a *app.App, cmd *cobra.Command, args []string) error {
	remotePath := args[0]
	permissionID := args[1]

	if remotePath == "" || permissionID == "" { // Should be caught by Args validation.
		return fmt.Errorf("remote path and permission ID for 'permissions get' cannot be empty")
	}

	permission, err := a.SDK.GetPermission(cmd.Context(), remotePath, permissionID)
	if err != nil {
		return fmt.Errorf("getting permission ID '%s' for '%s': %w", permissionID, remotePath, err)
	}

	ui.DisplaySinglePermission(permission, remotePath, permissionID)
	return nil
}

// filesPermissionsUpdateLogic contains the core logic for 'items permissions update'.
func filesPermissionsUpdateLogic(a *app.App, cmd *cobra.Command, args []string) error {
	remotePath := args[0]
	permissionID := args[1]

	if remotePath == "" || permissionID == "" { // Should be caught by Args validation.
		return fmt.Errorf("remote path and permission ID for 'permissions update' cannot be empty")
	}

	// Parse flags for fields to update.
	roles, _ := cmd.Flags().GetStringSlice("roles")
	expiration, _ := cmd.Flags().GetString("expiration")
	password, _ := cmd.Flags().GetString("password")

	// Only include fields in the request if they were actually provided by the user
	// (or have defaults that should always be sent).
	// For StringSlice, Cobra returns nil if not set, empty slice if set but no values.
	// For String, Cobra returns empty string if not set.
	request := onedrive.UpdatePermissionRequest{}
	if cmd.Flags().Changed("roles") { // Check if the flag was explicitly set
		request.Roles = roles
	}
	if cmd.Flags().Changed("expiration") {
		request.ExpirationDateTime = expiration
	}
	if cmd.Flags().Changed("password") {
		request.Password = password
	}

	// Check if any updateable field was actually provided
	if request.Roles == nil && request.ExpirationDateTime == "" && request.Password == "" && !cmd.Flags().Changed("roles") {
		// If roles was set to empty deliberately, cmd.Flags().Changed("roles") would be true.
		// This condition means no relevant flags were specified for update.
		return fmt.Errorf("no update parameters (roles, expiration, password) provided for permission '%s' on '%s'", permissionID, remotePath)
	}

	permission, err := a.SDK.UpdatePermission(cmd.Context(), remotePath, permissionID, request)
	if err != nil {
		return fmt.Errorf("updating permission ID '%s' for '%s': %w", permissionID, remotePath, err)
	}

	ui.DisplaySinglePermission(permission, remotePath, permissionID)
	ui.PrintSuccess("Permission '%s' on '%s' updated successfully.", permissionID, remotePath)
	return nil
}

// filesPermissionsDeleteLogic contains the core logic for 'items permissions delete'.
func filesPermissionsDeleteLogic(a *app.App, cmd *cobra.Command, args []string) error {
	remotePath := args[0]
	permissionID := args[1]

	if remotePath == "" || permissionID == "" { // Should be caught by Args validation.
		return fmt.Errorf("remote path and permission ID for 'permissions delete' cannot be empty")
	}

	err := a.SDK.DeletePermission(cmd.Context(), remotePath, permissionID)
	if err != nil {
		return fmt.Errorf("deleting permission ID '%s' for '%s': %w", permissionID, remotePath, err)
	}

	ui.PrintSuccess("Permission '%s' successfully deleted from item '%s'.", permissionID, remotePath)
	return nil
}
