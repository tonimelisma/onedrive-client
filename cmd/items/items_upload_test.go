package items

import (
	"context"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/tonimelisma/onedrive-client/internal/app"
	"github.com/tonimelisma/onedrive-client/pkg/onedrive"
)

func TestFilesMkdirLogic(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "create folder success",
			args:    []string{"/test-folder"},
			wantErr: false,
		},
		{
			name:    "create folder in parent",
			args:    []string{"/parent", "child"},
			wantErr: false,
		},
		{
			name:    "empty arguments",
			args:    []string{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSDK := &MockSDK{
				CreateFolderFunc: func(ctx context.Context, parentPath string, folderName string) (onedrive.DriveItem, error) {
					return onedrive.DriveItem{Name: "test-folder", ID: "test-id"}, nil
				},
			}
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
