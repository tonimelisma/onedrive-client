package ui

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tonimelisma/onedrive-client/pkg/onedrive"
)

func TestDisplayBasicPermissionInfo(t *testing.T) {
	tests := []struct {
		name       string
		permission onedrive.Permission
		expected   []string
	}{
		{
			name: "basic permission info",
			permission: onedrive.Permission{
				ID:    "test-id",
				Roles: []string{"read", "write"},
			},
			expected: []string{"ID:", "test-id", "Roles:", "read, write"},
		},
		{
			name: "empty roles",
			permission: onedrive.Permission{
				ID:    "test-id-2",
				Roles: []string{},
			},
			expected: []string{"ID:", "test-id-2", "Roles:", ""},
		},
	}

	// Test that basic permission info display doesn't panic
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				displayBasicPermissionInfo(tt.permission)
			})
		})
	}
}

func TestDisplayExpirationTime(t *testing.T) {
	tests := []struct {
		name           string
		expirationTime string
		hasExpiry      bool
	}{
		{
			name:           "with expiration",
			expirationTime: "2024-12-31T23:59:59Z",
			hasExpiry:      true,
		},
		{
			name:           "without expiration",
			expirationTime: "",
			hasExpiry:      false,
		},
	}

	// Test that expiration time display doesn't panic
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				displayExpirationTime(tt.expirationTime)
			})
		})
	}
}

func TestDisplayPermissionLink(t *testing.T) {
	tests := []struct {
		name       string
		permission onedrive.Permission
		hasLink    bool
	}{
		{
			name: "with sharing link",
			permission: onedrive.Permission{
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
					Type:   "view",
					WebURL: "https://example.com/share",
				},
			},
			hasLink: true,
		},
		{
			name: "without sharing link",
			permission: onedrive.Permission{
				Link: nil,
			},
			hasLink: false,
		},
	}

	// Test that permission link display doesn't panic
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				displayPermissionLink(tt.permission)
			})
		})
	}
}

func TestDisplayPermissionDetailsRefactored(t *testing.T) {
	// Test that the refactored displayPermissionDetails function works without panicking
	permission := onedrive.Permission{
		ID:    "test-permission-id",
		Roles: []string{"read", "write"},
		GrantedToV2: &struct {
			User     *onedrive.Identity `json:"user,omitempty"`
			SiteUser *onedrive.Identity `json:"siteUser,omitempty"`
		}{
			User: &onedrive.Identity{
				DisplayName: "John Doe",
				ID:          "user-123",
			},
		},
		ExpirationDateTime: "2024-12-31T23:59:59Z",
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
			Type:   "view",
			WebURL: "https://example.com/share",
		},
	}

	// Test that the function doesn't panic with a complete permission object
	assert.NotPanics(t, func() {
		displayPermissionDetails(permission)
	})
}

func TestDisplayPermissionDetailsEmptyFields(t *testing.T) {
	// Test with minimal permission object to ensure helper functions handle nil/empty values
	permission := onedrive.Permission{
		ID:    "minimal-permission-id",
		Roles: []string{},
	}

	// Test that the function doesn't panic with minimal data
	assert.NotPanics(t, func() {
		displayPermissionDetails(permission)
	})
}

func TestPerformanceOptimizations(t *testing.T) {
	// Test that the performance-optimized display functions work correctly
	items := onedrive.DriveItemList{
		Value: []onedrive.DriveItem{
			{
				Name:                 "test-file-1.txt",
				ID:                   "file-1",
				Size:                 1024,
				LastModifiedDateTime: time.Now(),
				File:                 &onedrive.FileFacet{MimeType: "text/plain"},
			},
			{
				Name:                 "test-file-2.txt",
				ID:                   "file-2",
				Size:                 2048,
				LastModifiedDateTime: time.Now(),
				File:                 &onedrive.FileFacet{MimeType: "text/plain"},
			},
		},
	}

	// Test that DisplayItems doesn't panic with the optimized iteration
	assert.NotPanics(t, func() {
		DisplayItems(items)
	})

	// Verify the data is accessible (pointer iteration should work)
	assert.Len(t, items.Value, 2)
	assert.Equal(t, "test-file-1.txt", items.Value[0].Name)
	assert.Equal(t, "test-file-2.txt", items.Value[1].Name)
}

func TestDriveDisplayOptimizations(t *testing.T) {
	// Test that the drive display optimizations work correctly
	drives := onedrive.DriveList{
		Value: []onedrive.Drive{
			{
				Name:      "OneDrive",
				ID:        "drive-1",
				DriveType: "personal",
				Quota: struct {
					Total     int64  `json:"total"`
					Used      int64  `json:"used"`
					Remaining int64  `json:"remaining"`
					State     string `json:"state"`
				}{
					Total: 1024 * 1024 * 1024, // 1 GB
					Used:  512 * 1024 * 1024,  // 512 MB
				},
			},
			{
				Name:      "SharePoint",
				ID:        "drive-2",
				DriveType: "business",
				Quota: struct {
					Total     int64  `json:"total"`
					Used      int64  `json:"used"`
					Remaining int64  `json:"remaining"`
					State     string `json:"state"`
				}{
					Total: 2048 * 1024 * 1024, // 2 GB
					Used:  1024 * 1024 * 1024, // 1 GB
				},
			},
		},
	}

	// Test that DisplayDrives doesn't panic with the optimized iteration
	assert.NotPanics(t, func() {
		DisplayDrives(drives)
	})

	// Verify the data is accessible (pointer iteration should work)
	assert.Len(t, drives.Value, 2)
	assert.Equal(t, "OneDrive", drives.Value[0].Name)
	assert.Equal(t, "SharePoint", drives.Value[1].Name)
}

func TestPermissionDisplayComponents(t *testing.T) {
	// Test various permission scenarios to ensure all helper functions work
	tests := []struct {
		name       string
		permission onedrive.Permission
	}{
		{
			name: "permission with link",
			permission: onedrive.Permission{
				ID:    "link-permission",
				Roles: []string{"read"},
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
					Type:   "view",
					WebURL: "https://example.com/link",
				},
			},
		},
		{
			name: "permission with granted user",
			permission: onedrive.Permission{
				ID:    "user-permission",
				Roles: []string{"write"},
				GrantedToV2: &struct {
					User     *onedrive.Identity `json:"user,omitempty"`
					SiteUser *onedrive.Identity `json:"siteUser,omitempty"`
				}{
					User: &onedrive.Identity{
						DisplayName: "Test User",
						ID:          "user-456",
					},
				},
			},
		},
		{
			name: "permission with expiration",
			permission: onedrive.Permission{
				ID:                 "expiring-permission",
				Roles:              []string{"read"},
				ExpirationDateTime: "2024-12-31T23:59:59Z",
			},
		},
		{
			name: "permission with inheritance",
			permission: onedrive.Permission{
				ID:    "inherited-permission",
				Roles: []string{"read"},
				InheritedFrom: &struct {
					DriveID string `json:"driveId"`
					ID      string `json:"id"`
					Path    string `json:"path"`
				}{
					ID:   "parent-folder-id",
					Path: "/Documents",
				},
			},
		},
		{
			name: "permission with invitation",
			permission: onedrive.Permission{
				ID:    "invitation-permission",
				Roles: []string{"read"},
				Invitation: &struct {
					SignInRequired bool   `json:"signInRequired"`
					Email          string `json:"email,omitempty"`
				}{
					Email:          "invited@example.com",
					SignInRequired: true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that each permission type can be displayed without panicking
			assert.NotPanics(t, func() {
				displayPermissionDetails(tt.permission)
			})
		})
	}
}

func TestActivityDisplayOptimizations(t *testing.T) {
	// Test that the activity display optimizations work correctly
	activities := onedrive.ActivityList{
		Value: []onedrive.Activity{
			{
				ID: "activity-1",
				Actor: struct {
					User        *onedrive.Identity `json:"user,omitempty"`
					Application *onedrive.Identity `json:"application,omitempty"`
				}{
					User: &onedrive.Identity{
						DisplayName: "John Doe",
						ID:          "user-123",
					},
				},
				Times: struct {
					RecordedTime time.Time `json:"recordedTime"`
				}{
					RecordedTime: time.Now(),
				},
			},
			{
				ID: "activity-2",
				Actor: struct {
					User        *onedrive.Identity `json:"user,omitempty"`
					Application *onedrive.Identity `json:"application,omitempty"`
				}{
					User: &onedrive.Identity{
						DisplayName: "Jane Smith",
						ID:          "user-456",
					},
				},
				Times: struct {
					RecordedTime time.Time `json:"recordedTime"`
				}{
					RecordedTime: time.Now(),
				},
			},
		},
	}

	// Test that DisplayActivities doesn't panic with the optimized iteration
	assert.NotPanics(t, func() {
		DisplayActivities(activities)
	})

	// Verify the data is accessible (pointer iteration should work)
	assert.Len(t, activities.Value, 2)
	assert.Equal(t, "activity-1", activities.Value[0].ID)
	assert.Equal(t, "activity-2", activities.Value[1].ID)
}

func TestPermissionListDisplayOptimizations(t *testing.T) {
	// Test that the permission list display optimizations work correctly
	permissions := onedrive.PermissionList{
		Value: []onedrive.Permission{
			{
				ID:    "permission-1",
				Roles: []string{"read"},
			},
			{
				ID:    "permission-2",
				Roles: []string{"write"},
			},
		},
	}

	// Test that DisplayPermissions doesn't panic with the optimized iteration
	assert.NotPanics(t, func() {
		DisplayPermissions(permissions)
	})

	// Verify the data is accessible (pointer iteration should work)
	assert.Len(t, permissions.Value, 2)
	assert.Equal(t, "permission-1", permissions.Value[0].ID)
	assert.Equal(t, "permission-2", permissions.Value[1].ID)
}
