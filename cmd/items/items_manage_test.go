package cmd

import (
	"context"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/tonimelisma/onedrive-client/internal/app"
	"github.com/tonimelisma/onedrive-client/pkg/onedrive"
)

func TestFilesRmLogic(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		mockSetup func() *MockSDK
		wantErr   bool
	}{
		{
			name: "delete file success",
			args: []string{"/test-file.txt"},
			mockSetup: func() *MockSDK {
				return &MockSDK{
					DeleteDriveItemFunc: func(ctx context.Context, path string) error {
						assert.Equal(t, "/test-file.txt", path)
						return nil
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

			err := filesRmLogic(a, &cobra.Command{}, tt.args)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFilesCopyLogic(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("new-name", "", "")
	cmd.Flags().Bool("wait", false, "")

	tests := []struct {
		name      string
		args      []string
		newName   string
		wait      bool
		mockSetup func() *MockSDK
		wantErr   bool
	}{
		{
			name: "copy file without new name",
			args: []string{"/source.txt", "/destination/"},
			mockSetup: func() *MockSDK {
				return &MockSDK{
					CopyDriveItemFunc: func(ctx context.Context, sourcePath, destinationParentPath, newName string) (string, error) {
						assert.Equal(t, "/source.txt", sourcePath)
						assert.Equal(t, "/destination/", destinationParentPath)
						assert.Equal(t, "", newName)
						return "monitor-url", nil
					},
				}
			},
			wantErr: false,
		},
		{
			name:    "copy file with new name",
			args:    []string{"/source.txt", "/destination/", "renamed.txt"},
			newName: "renamed.txt",
			mockSetup: func() *MockSDK {
				return &MockSDK{
					CopyDriveItemFunc: func(ctx context.Context, sourcePath, destinationParentPath, newName string) (string, error) {
						assert.Equal(t, "/source.txt", sourcePath)
						assert.Equal(t, "/destination/", destinationParentPath)
						assert.Equal(t, "renamed.txt", newName)
						return "monitor-url", nil
					},
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd.Flags().Set("new-name", tt.newName)
			cmd.Flags().Set("wait", "false")
			if tt.wait {
				cmd.Flags().Set("wait", "true")
			}

			mockSDK := tt.mockSetup()
			a := &app.App{SDK: mockSDK}

			err := filesCopyLogic(a, cmd, tt.args)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFilesCopyStatusLogic(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		mockSetup func() *MockSDK
		wantErr   bool
	}{
		{
			name: "get copy status success",
			args: []string{"monitor-url"},
			mockSetup: func() *MockSDK {
				return &MockSDK{
					MonitorCopyOperationFunc: func(ctx context.Context, monitorURL string) (onedrive.CopyOperationStatus, error) {
						assert.Equal(t, "monitor-url", monitorURL)
						return onedrive.CopyOperationStatus{
							Status:            "completed",
							StatusDescription: "Copy operation completed successfully",
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

			err := filesCopyStatusLogic(a, &cobra.Command{}, tt.args)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFilesMvLogic(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		mockSetup func() *MockSDK
		wantErr   bool
	}{
		{
			name: "move file success",
			args: []string{"/source.txt", "/destination/"},
			mockSetup: func() *MockSDK {
				return &MockSDK{
					MoveDriveItemFunc: func(ctx context.Context, sourcePath, destinationParentPath string) (onedrive.DriveItem, error) {
						assert.Equal(t, "/source.txt", sourcePath)
						assert.Equal(t, "/destination/", destinationParentPath)
						return onedrive.DriveItem{Name: "source.txt", ID: "moved-item-id"}, nil
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

			err := filesMvLogic(a, &cobra.Command{}, tt.args)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFilesRenameLogic(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		mockSetup func() *MockSDK
		wantErr   bool
	}{
		{
			name: "rename file success",
			args: []string{"/oldname.txt", "newname.txt"},
			mockSetup: func() *MockSDK {
				return &MockSDK{
					UpdateDriveItemFunc: func(ctx context.Context, path, newName string) (onedrive.DriveItem, error) {
						assert.Equal(t, "/oldname.txt", path)
						assert.Equal(t, "newname.txt", newName)
						return onedrive.DriveItem{Name: "newname.txt", ID: "renamed-item-id"}, nil
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

			err := filesRenameLogic(a, &cobra.Command{}, tt.args)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
