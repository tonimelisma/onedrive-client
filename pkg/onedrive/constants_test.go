package onedrive

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHTTPStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant int
		expected int
	}{
		{"StatusOK", StatusOK, 200},
		{"StatusCreated", StatusCreated, 201},
		{"StatusAccepted", StatusAccepted, 202},
		{"StatusNoContent", StatusNoContent, 204},
		{"StatusBadRequest", StatusBadRequest, 400},
		{"StatusUnauthorized", StatusUnauthorized, 401},
		{"StatusForbidden", StatusForbidden, 403},
		{"StatusNotFound", StatusNotFound, 404},
		{"StatusConflict", StatusConflict, 409},
		{"StatusPayloadTooLarge", StatusPayloadTooLarge, 413},
		{"StatusTooManyRequests", StatusTooManyRequests, 429},
		{"StatusServiceUnavailable", StatusServiceUnavailable, 503},
		{"StatusInsufficientStorage", StatusInsufficientStorage, 507},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.constant)
		})
	}
}

func TestFilePermissionConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant int
		expected int
	}{
		{"PermUserOnly", PermUserOnly, 0o700},
		{"PermUserRead", PermUserRead, 0o400},
		{"PermUserWrite", PermUserWrite, 0o200},
		{"PermUserReadWrite", PermUserReadWrite, 0o600},
		{"PermUserReadWriteExec", PermUserReadWriteExec, 0o700},
		{"PermGroupRead", PermGroupRead, 0o040},
		{"PermGroupWrite", PermGroupWrite, 0o020},
		{"PermGroupReadWrite", PermGroupReadWrite, 0o060},
		{"PermOtherRead", PermOtherRead, 0o004},
		{"PermOtherWrite", PermOtherWrite, 0o002},
		{"PermOtherReadWrite", PermOtherReadWrite, 0o006},
		{"PermStandardFile", PermStandardFile, 0o644},
		{"PermExecutableFile", PermExecutableFile, 0o755},
		{"PermSecureFile", PermSecureFile, 0o600},
		{"PermSecureDir", PermSecureDir, 0o700},
		{"PermStandardDir", PermStandardDir, 0o755},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.constant)
		})
	}
}

func TestHTTPConfigConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant any
		expected any
	}{
		{"DefaultTimeout", DefaultTimeout, 30 * time.Second},
		{"DefaultRetryAttempts", DefaultRetryAttempts, 3},
		{"DefaultRetryDelay", DefaultRetryDelay, 1 * time.Second},
		{"DefaultMaxRetryDelay", DefaultMaxRetryDelay, 10 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.constant)
		})
	}
}

func TestPollingConfigConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant any
		expected any
	}{
		{"DefaultPollInterval", DefaultPollInterval, 2 * time.Second},
		{"DefaultMaxInterval", DefaultMaxInterval, 30 * time.Second},
		{"DefaultPollMultiplier", DefaultPollMultiplier, 1.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.constant)
		})
	}
}

func TestUIDisplayConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant int
		expected int
	}{
		{"DefaultLineLength", DefaultLineLength, 120},
		{"ShortLineLength", ShortLineLength, 70},
		{"MediumLineLength", MediumLineLength, 80},
		{"LongLineLength", LongLineLength, 100},
		{"ExtraLongLineLength", ExtraLongLineLength, 110},
		{"MaxLineLength", MaxLineLength, 120},
		{"ProgressBarWidth", ProgressBarWidth, 40},
		{"SpinnerType", SpinnerType, 14},
		{"MaxNameDisplayLength", MaxNameDisplayLength, 57},
		{"MaxShortNameLength", MaxShortNameLength, 47},
		{"MaxPermissionIDLength", MaxPermissionIDLength, 38},
		{"MaxRolesDisplayLength", MaxRolesDisplayLength, 23},
		{"MaxVersionIDLength", MaxVersionIDLength, 37},
		{"MaxActorNameLength", MaxActorNameLength, 18},
		{"MaxShortPathLength", MaxShortPathLength, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.constant)
		})
	}
}

func TestUIDisplayTimeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant any
		expected any
	}{
		{"ProgressBarThrottle", ProgressBarThrottle, 100 * time.Millisecond},
		{"DefaultProgressInterval", DefaultProgressInterval, 100 * time.Millisecond},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.constant)
		})
	}
}

func TestFileAndPathConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant int
		expected int
	}{
		{"MaxFileNameLength", MaxFileNameLength, 255},
		{"MaxPathLength", MaxPathLength, 400},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.constant)
		})
	}
}

func TestAuthenticationConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant int
		expected int
	}{
		{"DefaultDeviceCodeExpiry", DefaultDeviceCodeExpiry, 60},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.constant)
		})
	}
}

func TestBufferAndChunkConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant int
		expected int
	}{
		{"DefaultChunkSize", DefaultChunkSize, 320 * 1024 * 4},
		{"DefaultBufferSize", DefaultBufferSize, 1024},
		{"MinFileSize", MinFileSize, 1024},
		{"LargeFileThreshold", LargeFileThreshold, 4 * 1024 * 1024},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.constant)
		})
	}
}

func TestTimeFormatConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"StandardTimeFormat", StandardTimeFormat, "2006-01-02 15:04"},
		{"FullTimeFormat", FullTimeFormat, "2006-01-02 15:04:05"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.constant)
		})
	}
}

func TestTableDisplayConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant int
		expected int
	}{
		{"TableNameColumnWidth", TableNameColumnWidth, 50},
		{"TableSizeColumnWidth", TableSizeColumnWidth, 12},
		{"TableTypeColumnWidth", TableTypeColumnWidth, 10},
		{"TableDateColumnWidth", TableDateColumnWidth, 20},
		{"TablePermissionColumnWidth", TablePermissionColumnWidth, 40},
		{"TableActorColumnWidth", TableActorColumnWidth, 20},
		{"TableActionColumnWidth", TableActionColumnWidth, 15},
		{"TableVersionColumnWidth", TableVersionColumnWidth, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.constant)
		})
	}
}

func TestSeparatorConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant int
		expected int
	}{
		{"StandardSeparatorLength", StandardSeparatorLength, 70},
		{"MediumSeparatorLength", MediumSeparatorLength, 80},
		{"LongSeparatorLength", LongSeparatorLength, 95},
		{"ExtraLongSeparatorLength", ExtraLongSeparatorLength, 100},
		{"PermissionSeparatorLength", PermissionSeparatorLength, 120},
		{"ActivitySeparatorLength", ActivitySeparatorLength, 90},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.constant)
		})
	}
}

func TestContentFormattingConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant any
		expected any
	}{
		{"EllipsisMarker", EllipsisMarker, "..."},
		{"EllipsisLength", EllipsisLength, 3},
		{"DefaultTableIndent", DefaultTableIndent, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.constant)
		})
	}
}

func TestConstantTypes(t *testing.T) {
	// Verify constants have correct types
	assert.IsType(t, int(0), StatusOK)
	assert.IsType(t, int(0), PermUserOnly)
	assert.IsType(t, time.Duration(0), DefaultTimeout)
	assert.IsType(t, string(""), StandardTimeFormat)
	assert.IsType(t, float64(0), DefaultPollMultiplier)
}

func TestChunkSizeValidation(t *testing.T) {
	// Test that chunk size constants are valid
	assert.True(t, DefaultChunkSize > 0, "DefaultChunkSize should be positive")
	assert.True(t, DefaultBufferSize > 0, "DefaultBufferSize should be positive")
	assert.True(t, LargeFileThreshold > 0, "LargeFileThreshold should be positive")
	assert.True(t, MinFileSize > 0, "MinFileSize should be positive")
	assert.True(t, LargeFileThreshold > MinFileSize, "LargeFileThreshold should be > MinFileSize")
}

func TestTimeConstantValidation(t *testing.T) {
	// Test that time constants are reasonable
	assert.True(t, DefaultTimeout > 0, "DefaultTimeout should be positive")
	assert.True(t, DefaultRetryDelay > 0, "DefaultRetryDelay should be positive")
	assert.True(t, DefaultMaxRetryDelay > DefaultRetryDelay, "DefaultMaxRetryDelay should be > DefaultRetryDelay")
	assert.True(t, DefaultPollInterval > 0, "DefaultPollInterval should be positive")
	assert.True(t, DefaultMaxInterval > DefaultPollInterval, "DefaultMaxInterval should be > DefaultPollInterval")
}

func TestDisplayConstantValidation(t *testing.T) {
	// Test that display constants are reasonable
	assert.True(t, ProgressBarWidth > 0, "ProgressBarWidth should be positive")
	assert.True(t, MaxNameDisplayLength > 0, "MaxNameDisplayLength should be positive")
	assert.True(t, DefaultLineLength > 0, "DefaultLineLength should be positive")
	assert.True(t, MaxLineLength >= DefaultLineLength, "MaxLineLength should be >= DefaultLineLength")
	assert.True(t, StandardSeparatorLength > 0, "StandardSeparatorLength should be positive")
}

func TestPermissionConstantValidation(t *testing.T) {
	// Test that permission constants are valid octal values
	assert.Equal(t, 0o700, PermUserOnly)
	assert.Equal(t, 0o644, PermStandardFile)
	assert.Equal(t, 0o755, PermExecutableFile)
	assert.Equal(t, 0o755, PermStandardDir)
}

func TestDeviceCodeExpiryValidation(t *testing.T) {
	// Test that device code expiry is reasonable (in minutes)
	assert.True(t, DefaultDeviceCodeExpiry > 0, "DefaultDeviceCodeExpiry should be positive")
	assert.True(t, DefaultDeviceCodeExpiry >= 15, "DefaultDeviceCodeExpiry should be at least 15 minutes")
	assert.True(t, DefaultDeviceCodeExpiry <= 120, "DefaultDeviceCodeExpiry should be at most 120 minutes")
}
