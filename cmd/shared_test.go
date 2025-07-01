package cmd

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/tonimelisma/onedrive-client/pkg/onedrive"
)

func TestDrivesSharedLogic(t *testing.T) {
	tests := []struct {
		name        string
		mockItems   onedrive.DriveItemList
		mockError   error
		expectError bool
	}{
		{
			name: "successful shared items",
			mockItems: onedrive.DriveItemList{
				Value: []onedrive.DriveItem{
					{
						Name: "shared-file.txt",
						Size: 1024,
						RemoteItem: &struct {
							ID             string `json:"id"`
							Size           int64  `json:"size"`
							WebURL         string `json:"webUrl"`
							FileSystemInfo struct {
								CreatedDateTime      time.Time `json:"createdDateTime"`
								LastModifiedDateTime time.Time `json:"lastModifiedDateTime"`
							} `json:"fileSystemInfo"`
						}{
							ID:   "remote123",
							Size: 1024,
						},
					},
					{
						Name: "shared-folder",
						Folder: &struct {
							ChildCount int `json:"childCount"`
							View       struct {
								ViewType  string `json:"viewType"`
								SortBy    string `json:"sortBy"`
								SortOrder string `json:"sortOrder"`
							} `json:"view"`
						}{ChildCount: 5},
					},
				},
			},
			mockError:   nil,
			expectError: false,
		},
		{
			name:        "empty shared items",
			mockItems:   onedrive.DriveItemList{Value: []onedrive.DriveItem{}},
			mockError:   nil,
			expectError: false,
		},
		{
			name:        "shared items error",
			mockItems:   onedrive.DriveItemList{},
			mockError:   errors.New("shared items failed"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSDK := &MockSDK{
				GetSharedWithMeFunc: func(ctx context.Context) (onedrive.DriveItemList, error) {
					return tt.mockItems, tt.mockError
				},
			}

			app := newTestApp(mockSDK)
			cmd := &cobra.Command{}
			err := drivesSharedLogic(app, cmd)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
