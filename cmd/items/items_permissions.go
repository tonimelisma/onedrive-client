package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tonimelisma/onedrive-client/internal/app"
	"github.com/tonimelisma/onedrive-client/internal/ui"
	"github.com/tonimelisma/onedrive-client/pkg/onedrive"
)

var filesShareCmd = &cobra.Command{
	Use:   "share <remote-path> <link-type> <scope>",
	Short: "Create sharing links for files and folders",
	Long:  "Creates sharing links for OneDrive files and folders. Link types: view, edit, embed. Scopes: anonymous, organization.",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("error creating app: %w", err)
		}
		return filesShareLogic(a, cmd, args)
	},
}

var filesInviteCmd = &cobra.Command{
	Use:   "invite <remote-path> <email> [additional-emails...]",
	Short: "Invite users to access files and folders",
	Long:  "Invites users to access OneDrive files and folders with configurable permissions.",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("error creating app: %w", err)
		}
		return filesInviteLogic(a, cmd, args)
	},
}

var filesPermissionsCmd = &cobra.Command{
	Use:   "permissions",
	Short: "Manage permissions on files and folders",
	Long:  "Provides commands to list, get, update, and delete permissions on OneDrive items.",
}

var filesPermissionsListCmd = &cobra.Command{
	Use:   "list <remote-path>",
	Short: "List permissions on a file or folder",
	Long:  "Lists all permissions on a specific file or folder.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("error creating app: %w", err)
		}
		return filesPermissionsListLogic(a, cmd, args)
	},
}

var filesPermissionsGetCmd = &cobra.Command{
	Use:   "get <remote-path> <permission-id>",
	Short: "Get detailed information about a specific permission",
	Long:  "Retrieves detailed information about a specific permission by its ID.",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("error creating app: %w", err)
		}
		return filesPermissionsGetLogic(a, cmd, args)
	},
}

var filesPermissionsUpdateCmd = &cobra.Command{
	Use:   "update <remote-path> <permission-id>",
	Short: "Update permission roles, expiration, or password",
	Long:  "Updates an existing permission with new roles, expiration date, or password.",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("error creating app: %w", err)
		}
		return filesPermissionsUpdateLogic(a, cmd, args)
	},
}

var filesPermissionsDeleteCmd = &cobra.Command{
	Use:   "delete <remote-path> <permission-id>",
	Short: "Remove a specific permission",
	Long:  "Removes a specific permission from a file or folder.",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("error creating app: %w", err)
		}
		return filesPermissionsDeleteLogic(a, cmd, args)
	},
}

func filesShareLogic(a *app.App, cmd *cobra.Command, args []string) error {
	remotePath := args[0]
	linkType := args[1]
	scope := args[2]

	if remotePath == "" || linkType == "" || scope == "" {
		return fmt.Errorf("remote path, link type, and scope cannot be empty")
	}

	// Validate link type
	validLinkTypes := map[string]bool{"view": true, "edit": true, "embed": true}
	if !validLinkTypes[linkType] {
		return fmt.Errorf("invalid link type '%s'. Valid types: view, edit, embed", linkType)
	}

	// Validate scope
	validScopes := map[string]bool{"anonymous": true, "organization": true}
	if !validScopes[scope] {
		return fmt.Errorf("invalid scope '%s'. Valid scopes: anonymous, organization", scope)
	}

	link, err := a.SDK.CreateSharingLink(remotePath, linkType, scope)
	if err != nil {
		return fmt.Errorf("creating sharing link: %w", err)
	}

	ui.DisplaySharingLink(link)
	return nil
}

func filesInviteLogic(a *app.App, cmd *cobra.Command, args []string) error {
	remotePath := args[0]
	if remotePath == "" {
		return fmt.Errorf("remote path cannot be empty")
	}

	// Extract emails from args
	emails := args[1:]
	if len(emails) == 0 {
		return fmt.Errorf("at least one email address is required")
	}

	// Parse optional flags
	message, _ := cmd.Flags().GetString("message")
	roles, _ := cmd.Flags().GetStringSlice("roles")
	requireSignIn, _ := cmd.Flags().GetBool("require-signin")
	sendInvitation, _ := cmd.Flags().GetBool("send-invitation")

	// Default to read access if no roles specified
	if len(roles) == 0 {
		roles = []string{"read"}
	}

	// Build recipients
	recipients := make([]struct {
		Email string `json:"email"`
	}, len(emails))
	for i, email := range emails {
		recipients[i].Email = email
	}

	request := onedrive.InviteRequest{
		Recipients:     recipients,
		Message:        message,
		RequireSignIn:  requireSignIn,
		SendInvitation: sendInvitation,
		Roles:          roles,
	}

	response, err := a.SDK.InviteUsers(remotePath, request)
	if err != nil {
		return fmt.Errorf("inviting users: %w", err)
	}

	ui.DisplayInviteResponse(response, remotePath)
	return nil
}

func filesPermissionsListLogic(a *app.App, cmd *cobra.Command, args []string) error {
	remotePath := args[0]
	if remotePath == "" {
		return fmt.Errorf("remote path cannot be empty")
	}

	permissions, err := a.SDK.ListPermissions(remotePath)
	if err != nil {
		return fmt.Errorf("listing permissions: %w", err)
	}

	ui.DisplayPermissions(permissions, remotePath)
	return nil
}

func filesPermissionsGetLogic(a *app.App, cmd *cobra.Command, args []string) error {
	remotePath := args[0]
	permissionID := args[1]

	if remotePath == "" || permissionID == "" {
		return fmt.Errorf("remote path and permission ID cannot be empty")
	}

	permission, err := a.SDK.GetPermission(remotePath, permissionID)
	if err != nil {
		return fmt.Errorf("getting permission: %w", err)
	}

	ui.DisplaySinglePermission(permission, remotePath, permissionID)
	return nil
}

func filesPermissionsUpdateLogic(a *app.App, cmd *cobra.Command, args []string) error {
	remotePath := args[0]
	permissionID := args[1]

	if remotePath == "" || permissionID == "" {
		return fmt.Errorf("remote path and permission ID cannot be empty")
	}

	// Parse update flags
	roles, _ := cmd.Flags().GetStringSlice("roles")
	expiration, _ := cmd.Flags().GetString("expiration")
	password, _ := cmd.Flags().GetString("password")

	request := onedrive.UpdatePermissionRequest{
		Roles:              roles,
		ExpirationDateTime: expiration,
		Password:           password,
	}

	permission, err := a.SDK.UpdatePermission(remotePath, permissionID, request)
	if err != nil {
		return fmt.Errorf("updating permission: %w", err)
	}

	ui.DisplaySinglePermission(permission, remotePath, permissionID)
	ui.PrintSuccess("Permission updated successfully.\n")
	return nil
}

func filesPermissionsDeleteLogic(a *app.App, cmd *cobra.Command, args []string) error {
	remotePath := args[0]
	permissionID := args[1]

	if remotePath == "" || permissionID == "" {
		return fmt.Errorf("remote path and permission ID cannot be empty")
	}

	err := a.SDK.DeletePermission(remotePath, permissionID)
	if err != nil {
		return fmt.Errorf("deleting permission: %w", err)
	}

	ui.PrintSuccess("Permission '%s' deleted successfully from '%s'.\n", permissionID, remotePath)
	return nil
}
