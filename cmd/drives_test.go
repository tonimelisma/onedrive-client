package cmd

import (
	"context"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/tonimelisma/onedrive-client/pkg/onedrive"
)

func TestDrivesListLogic(t *testing.T) {
	mockSDK := &MockSDK{
		GetDrivesFunc: func(ctx context.Context) (onedrive.DriveList, error) {
			return onedrive.DriveList{
				Value: []onedrive.Drive{
					{
						Name:      "Personal Drive",
						DriveType: "personal",
						Owner: struct {
							User struct {
								DisplayName string "json:\"displayName\""
							} "json:\"user\""
						}{User: struct {
							DisplayName string "json:\"displayName\""
						}{DisplayName: "Test User"}},
					},
				},
			}, nil
		},
	}

	app := newTestApp(mockSDK)
	cmd := &cobra.Command{}

	output := captureOutput(t, func() {
		err := drivesListLogic(app, cmd)
		assert.NoError(t, err)
	})

	assert.True(t, strings.HasPrefix(output, "Drive Name"), "Output should have a header")
	assert.Contains(t, output, "Personal Drive")
	assert.Contains(t, output, "personal")
	assert.Contains(t, output, "Test User")
	assert.NotContains(t, output, "GiB", "Output should not contain quota information for list")
}

func TestDrivesQuotaLogic(t *testing.T) {
	mockSDK := &MockSDK{
		GetDefaultDriveFunc: func(ctx context.Context) (onedrive.Drive, error) {
			return onedrive.Drive{
				Quota: struct {
					Total     int64  `json:"total"`
					Used      int64  `json:"used"`
					Remaining int64  `json:"remaining"`
					State     string `json:"state"`
				}{Total: 2000000000, Used: 1000000000, Remaining: 1000000000, State: "normal"},
			}, nil
		},
	}

	app := newTestApp(mockSDK)
	cmd := &cobra.Command{}

	output := captureOutput(t, func() {
		err := drivesQuotaLogic(app, cmd)
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "Drive Quota Information")
	assert.Contains(t, output, "Total:")
	assert.Contains(t, output, "1.9 GiB")
	assert.Contains(t, output, "Used:")
	assert.Contains(t, output, "953.7 MiB")
	assert.Contains(t, output, "Remaining:")
	assert.Contains(t, output, "State:     normal")
}

func TestDrivesGetLogic(t *testing.T) {
	tests := []struct {
		name         string
		driveID      string
		getDriveFunc func(ctx context.Context, driveID string) (onedrive.Drive, error)
		expectError  bool
	}{
		{
			name:    "Get drive by ID successfully",
			driveID: "test-drive-id",
			getDriveFunc: func(ctx context.Context, driveID string) (onedrive.Drive, error) {
				if driveID != "test-drive-id" {
					t.Errorf("Expected driveID 'test-drive-id', got %s", driveID)
				}
				return onedrive.Drive{
					ID:        "test-drive-id",
					Name:      "Test Drive",
					DriveType: "personal",
					Owner: struct {
						User struct {
							DisplayName string `json:"displayName"`
						} `json:"user"`
					}{
						User: struct {
							DisplayName string `json:"displayName"`
						}{
							DisplayName: "Test User",
						},
					},
					Quota: struct {
						Total     int64  `json:"total"`
						Used      int64  `json:"used"`
						Remaining int64  `json:"remaining"`
						State     string `json:"state"`
					}{
						Total:     1000000000,
						Used:      250000000,
						Remaining: 750000000,
						State:     "normal",
					},
				}, nil
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureOutput(t, func() {
				mockSDK := &MockSDK{
					GetDriveByIDFunc: tt.getDriveFunc,
				}
				testApp := newTestApp(mockSDK)
				cmd := &cobra.Command{}

				err := drivesGetLogic(testApp, cmd, tt.driveID)
				if (err != nil) != tt.expectError {
					t.Errorf("drivesGetLogic() error = %v, expectError %v", err, tt.expectError)
				}
			})

			if !tt.expectError && output == "" {
				t.Error("Expected output but got none")
			}
		})
	}
}

func TestDrivesDeltaLogic(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		deltaFunc   func(ctx context.Context, deltaToken string) (onedrive.DeltaResponse, error)
		expectError bool
	}{
		{
			name: "Delta without token",
			args: []string{},
			deltaFunc: func(ctx context.Context, deltaToken string) (onedrive.DeltaResponse, error) {
				if deltaToken != "" {
					t.Errorf("Expected empty token, got %s", deltaToken)
				}
				return onedrive.DeltaResponse{
					Value: []onedrive.DriveItem{
						{
							Name: "test-file.txt",
							Size: 1024,
						},
					},
					DeltaLink: "https://graph.microsoft.com/v1.0/me/drive/root/delta?token=abc123",
				}, nil
			},
			expectError: false,
		},
		{
			name: "Delta with token",
			args: []string{"abc123"},
			deltaFunc: func(ctx context.Context, deltaToken string) (onedrive.DeltaResponse, error) {
				if deltaToken != "abc123" {
					t.Errorf("Expected token 'abc123', got %s", deltaToken)
				}
				return onedrive.DeltaResponse{
					Value: []onedrive.DriveItem{
						{
							Name: "modified-file.txt",
							Size: 2048,
						},
					},
					DeltaLink: "https://graph.microsoft.com/v1.0/me/drive/root/delta?token=def456",
				}, nil
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureOutput(t, func() {
				mockSDK := &MockSDK{
					GetDeltaFunc: tt.deltaFunc,
				}
				testApp := newTestApp(mockSDK)
				cmd := &cobra.Command{}

				err := drivesDeltaLogic(testApp, cmd, tt.args)
				if (err != nil) != tt.expectError {
					t.Errorf("drivesDeltaLogic() error = %v, expectError %v", err, tt.expectError)
				}
			})

			if !tt.expectError && output == "" {
				t.Error("Expected output but got none")
			}
		})
	}
}

func TestDrivesSpecialLogic(t *testing.T) {
	mockSDK := &MockSDK{
		GetSpecialFolderFunc: func(ctx context.Context, folderName string) (onedrive.DriveItem, error) {
			assert.Equal(t, "Documents", folderName)
			return onedrive.DriveItem{
				Name: "Documents",
				ID:   "special-folder-id",
			}, nil
		},
	}
	a := newTestApp(mockSDK)
	cmd := &cobra.Command{}

	err := drivesSpecialLogic(a, cmd, []string{"Documents"})
	assert.NoError(t, err)
}

func TestDrivesRecentLogic(t *testing.T) {
	mockSDK := &MockSDK{
		GetRecentItemsFunc: func(ctx context.Context) (onedrive.DriveItemList, error) {
			return onedrive.DriveItemList{
				Value: []onedrive.DriveItem{
					{Name: "recent-file.txt", Size: 500},
				},
			}, nil
		},
	}
	a := newTestApp(mockSDK)
	cmd := &cobra.Command{}

	err := drivesRecentLogic(a, cmd)
	assert.NoError(t, err)
}

func TestDrivesRootLogic(t *testing.T) {
	mockSDK := &MockSDK{
		GetRootDriveItemsFunc: func(ctx context.Context) (onedrive.DriveItemList, error) {
			return onedrive.DriveItemList{
				Value: []onedrive.DriveItem{
					{Name: "Documents", Size: 0},
					{Name: "Pictures", Size: 0},
				},
			}, nil
		},
	}
	a := newTestApp(mockSDK)
	cmd := &cobra.Command{}

	err := drivesRootLogic(a, cmd)
	assert.NoError(t, err)
}
