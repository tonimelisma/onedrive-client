package cmd

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tonimelisma/onedrive-client/pkg/onedrive"
)

func TestDrivesListLogic(t *testing.T) {
	mockSDK := &MockSDK{
		GetDrivesFunc: func() (onedrive.DriveList, error) {
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

	output := captureOutput(t, func() {
		err := drivesListLogic(app)
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
		GetDefaultDriveFunc: func() (onedrive.Drive, error) {
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

	output := captureOutput(t, func() {
		err := drivesQuotaLogic(app)
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
		getDriveFunc func(driveID string) (onedrive.Drive, error)
		expectError  bool
	}{
		{
			name:    "Get drive by ID successfully",
			driveID: "test-drive-id",
			getDriveFunc: func(driveID string) (onedrive.Drive, error) {
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

				err := drivesGetLogic(testApp, tt.driveID)
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
