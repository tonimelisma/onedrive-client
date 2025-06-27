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

	output := captureOutput(func() {
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

	output := captureOutput(func() {
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
