package cmd

import (
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/tonimelisma/onedrive-client/internal/app"
	"github.com/tonimelisma/onedrive-client/pkg/onedrive"
)

func TestFilesMkdirLogic(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		mockSetup func() *MockSDK
		wantErr   bool
	}{
		{
			name: "create folder",
			args: []string{"/NewFolder"},
			mockSetup: func() *MockSDK {
				return &MockSDK{
					CreateFolderFunc: func(parentPath, folderName string) (onedrive.DriveItem, error) {
						assert.Equal(t, "/", parentPath)
						assert.Equal(t, "NewFolder", folderName)
						return onedrive.DriveItem{Name: "NewFolder", ID: "folder-id"}, nil
					},
				}
			},
			wantErr: false,
		},
		{
			name: "create nested folder",
			args: []string{"/Documents/Projects/MyProject"},
			mockSetup: func() *MockSDK {
				return &MockSDK{
					CreateFolderFunc: func(parentPath, folderName string) (onedrive.DriveItem, error) {
						assert.Equal(t, "/Documents/Projects", parentPath)
						assert.Equal(t, "MyProject", folderName)
						return onedrive.DriveItem{Name: "MyProject", ID: "folder-id"}, nil
					},
				}
			},
			wantErr: false,
		},
		{
			name: "no arguments error",
			args: []string{},
			mockSetup: func() *MockSDK {
				return &MockSDK{}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSDK := tt.mockSetup()
			a := &app.App{SDK: mockSDK}

			err := filesMkdirLogic(a, &cobra.Command{}, tt.args)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFilesCancelUploadLogic(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		mockSetup func() *MockSDK
		wantErr   bool
	}{
		{
			name: "cancel existing session",
			args: []string{"test-upload-url"},
			mockSetup: func() *MockSDK {
				return &MockSDK{
					CancelUploadSessionFunc: func(uploadURL string) error {
						assert.Equal(t, "test-upload-url", uploadURL)
						return nil
					},
				}
			},
			wantErr: false,
		},
		{
			name: "empty upload URL",
			args: []string{""},
			mockSetup: func() *MockSDK {
				return &MockSDK{}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSDK := tt.mockSetup()
			a := &app.App{SDK: mockSDK}

			err := filesCancelUploadLogic(a, &cobra.Command{}, tt.args)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFilesGetUploadStatusLogic(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		mockSetup func() *MockSDK
		wantErr   bool
	}{
		{
			name: "get status of existing session",
			args: []string{"test-upload-url"},
			mockSetup: func() *MockSDK {
				return &MockSDK{
					GetUploadSessionStatusFunc: func(uploadURL string) (onedrive.UploadSession, error) {
						assert.Equal(t, "test-upload-url", uploadURL)
						return onedrive.UploadSession{
							UploadURL:          "test-upload-url",
							NextExpectedRanges: []string{"0-1023"},
						}, nil
					},
				}
			},
			wantErr: false,
		},
		{
			name: "empty upload URL",
			args: []string{""},
			mockSetup: func() *MockSDK {
				return &MockSDK{}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSDK := tt.mockSetup()
			a := &app.App{SDK: mockSDK}

			err := filesGetUploadStatusLogic(a, &cobra.Command{}, tt.args)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFilesUploadSimpleLogic(t *testing.T) {
	// Create a temporary file for testing
	tmpfile, err := os.CreateTemp("", "test-upload-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name()) // clean up
	defer tmpfile.Close()

	// Write some test data
	if _, err := tmpfile.Write([]byte("test content")); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		args      []string
		mockSetup func() *MockSDK
		wantErr   bool
	}{
		{
			name: "simple upload success",
			args: []string{tmpfile.Name(), "/remote.txt"},
			mockSetup: func() *MockSDK {
				return &MockSDK{
					UploadFileFunc: func(localPath, remotePath string) (onedrive.DriveItem, error) {
						assert.Equal(t, tmpfile.Name(), localPath)
						assert.Equal(t, "/remote.txt", remotePath)
						return onedrive.DriveItem{Name: "remote.txt", ID: "test-id"}, nil
					},
				}
			},
			wantErr: false,
		},
		{
			name: "insufficient arguments",
			args: []string{"local.txt"},
			mockSetup: func() *MockSDK {
				return &MockSDK{}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSDK := tt.mockSetup()
			a := &app.App{SDK: mockSDK}

			err := filesUploadSimpleLogic(a, &cobra.Command{}, tt.args)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestJoinRemotePathHelper(t *testing.T) {
	tests := []struct {
		name     string
		arg1     string
		arg2     string
		expected string
	}{
		{
			name:     "simple join",
			arg1:     "/folder",
			arg2:     "file.txt",
			expected: "/folder/file.txt",
		},
		{
			name:     "root and file",
			arg1:     "/",
			arg2:     "file.txt",
			expected: "/file.txt",
		},
		{
			name:     "no leading slash",
			arg1:     "folder",
			arg2:     "file.txt",
			expected: "/folder/file.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := joinRemotePath(tt.arg1, tt.arg2)
			assert.Equal(t, tt.expected, result)
		})
	}
}
