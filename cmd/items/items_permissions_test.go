package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/tonimelisma/onedrive-client/internal/app"
	"github.com/tonimelisma/onedrive-client/pkg/onedrive"
)

func TestFilesPermissionsListLogic(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		mockSetup func() *MockSDK
		wantErr   bool
	}{
		{
			name: "list permissions success",
			args: []string{"/test-file.txt"},
			mockSetup: func() *MockSDK {
				return &MockSDK{
					ListPermissionsFunc: func(remotePath string) (onedrive.PermissionList, error) {
						assert.Equal(t, "/test-file.txt", remotePath)
						return onedrive.PermissionList{
							Value: []onedrive.Permission{
								{ID: "perm1", Roles: []string{"read"}},
								{ID: "perm2", Roles: []string{"write"}},
							},
						}, nil
					},
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSDK := tt.mockSetup()
			a := &app.App{SDK: mockSDK}

			err := filesPermissionsListLogic(a, &cobra.Command{}, tt.args)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFilesShareLogic(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("type", "view", "")
	cmd.Flags().String("scope", "anonymous", "")

	tests := []struct {
		name      string
		args      []string
		linkType  string
		scope     string
		mockSetup func() *MockSDK
		wantErr   bool
	}{
		{
			name:     "create sharing link success",
			args:     []string{"/test-file.txt", "view", "anonymous"},
			linkType: "view",
			scope:    "anonymous",
			mockSetup: func() *MockSDK {
				return &MockSDK{
					CreateSharingLinkFunc: func(path, linkType, scope string) (onedrive.SharingLink, error) {
						assert.Equal(t, "/test-file.txt", path)
						assert.Equal(t, "view", linkType)
						assert.Equal(t, "anonymous", scope)
						return onedrive.SharingLink{
							ID: "link1",
						}, nil
					},
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd.Flags().Set("type", tt.linkType)
			cmd.Flags().Set("scope", tt.scope)

			mockSDK := tt.mockSetup()
			a := &app.App{SDK: mockSDK}

			err := filesShareLogic(a, cmd, tt.args)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
