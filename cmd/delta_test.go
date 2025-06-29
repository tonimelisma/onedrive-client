package cmd

import (
	"testing"
	"time"

	"github.com/tonimelisma/onedrive-client/pkg/onedrive"
)

func TestDeltaLogic(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		deltaFunc   func(deltaToken string) (onedrive.DeltaResponse, error)
		expectError bool
	}{
		{
			name: "Delta without token",
			args: []string{},
			deltaFunc: func(deltaToken string) (onedrive.DeltaResponse, error) {
				if deltaToken != "" {
					t.Errorf("Expected empty token, got %s", deltaToken)
				}
				return onedrive.DeltaResponse{
					Value: []onedrive.DriveItem{
						{
							Name:                 "test-file.txt",
							Size:                 1024,
							LastModifiedDateTime: time.Now(),
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
			deltaFunc: func(deltaToken string) (onedrive.DeltaResponse, error) {
				if deltaToken != "abc123" {
					t.Errorf("Expected token 'abc123', got %s", deltaToken)
				}
				return onedrive.DeltaResponse{
					Value: []onedrive.DriveItem{
						{
							Name:                 "modified-file.txt",
							Size:                 2048,
							LastModifiedDateTime: time.Now(),
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

				err := deltaLogic(testApp, tt.args)
				if (err != nil) != tt.expectError {
					t.Errorf("deltaLogic() error = %v, expectError %v", err, tt.expectError)
				}
			})

			if !tt.expectError && output == "" {
				t.Error("Expected output but got none")
			}
		})
	}
}
