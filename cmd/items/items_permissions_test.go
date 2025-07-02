package items

import (
	"context"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/tonimelisma/onedrive-client/internal/app"
	"github.com/tonimelisma/onedrive-client/pkg/onedrive"
)

func TestPermissionsListLogic(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "list permissions success",
			args:    []string{"/test-file.txt"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSDK := &MockSDK{
				ListPermissionsFunc: func(ctx context.Context, remotePath string) (onedrive.PermissionList, error) {
					assert.Equal(t, "/test-file.txt", remotePath)
					return onedrive.PermissionList{
						Value: []onedrive.Permission{
							{
								ID: "permission1",
								Link: &struct {
									Type        string `json:"type"`
									Scope       string `json:"scope"`
									WebURL      string `json:"webUrl"`
									WebHTML     string `json:"webHtml,omitempty"`
									Application *struct {
										ID          string `json:"id"`
										DisplayName string `json:"displayName"`
									} `json:"application,omitempty"`
									PreventsDownload bool `json:"preventsDownload,omitempty"`
								}{
									Type:    "view",
									Scope:   "anonymous",
									WebURL:  "https://example.com/share1",
									WebHTML: "",
									Application: &struct {
										ID          string `json:"id"`
										DisplayName string `json:"displayName"`
									}{
										ID:          "app1",
										DisplayName: "Test App",
									},
									PreventsDownload: false,
								},
							},
						},
					}, nil
				},
			}
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

func TestPermissionsShareLogic(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "create sharing link success",
			args:    []string{"/test-file.txt", "view", "anonymous"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSDK := &MockSDK{
				CreateSharingLinkFunc: func(ctx context.Context, path, linkType, scope string) (onedrive.SharingLink, error) {
					assert.Equal(t, "/test-file.txt", path)
					assert.Equal(t, "view", linkType)
					assert.Equal(t, "anonymous", scope)
					return onedrive.SharingLink{
						ID: "share1",
						Link: struct {
							Type        string `json:"type"`
							Scope       string `json:"scope"`
							WebUrl      string `json:"webUrl"`
							WebHtml     string `json:"webHtml,omitempty"`
							Application *struct {
								Id          string `json:"id"`
								DisplayName string `json:"displayName"`
							} `json:"application,omitempty"`
						}{
							Type:   "view",
							Scope:  "anonymous",
							WebUrl: "https://example.com/share1",
						},
					}, nil
				},
			}
			a := &app.App{SDK: mockSDK}

			err := filesShareLogic(a, &cobra.Command{}, tt.args)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
