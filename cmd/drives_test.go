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
								DisplayName string `json:"displayName"`
							} `json:"user"`
						}{User: struct {
							DisplayName string `json:"displayName"`
						}{DisplayName: "Test User"}},
						Quota: struct {
							Total     int64  `json:"total"`
							Used      int64  `json:"used"`
							Remaining int64  `json:"remaining"`
							State     string `json:"state"`
						}{Total: 2000000000, Used: 1000000000},
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

	assert.Contains(t, output, "Personal Drive")
	assert.Contains(t, output, "personal")
	assert.Contains(t, output, "Test User")
	assert.Contains(t, output, "953.7 MiB / 1.9 GiB")
	assert.True(t, strings.HasPrefix(output, "Drive Name"))
}
