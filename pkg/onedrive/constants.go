// Package onedrive provides constants used throughout the OneDrive SDK.
package onedrive

import "time"

// HTTP Status Code Constants
const (
	StatusOK                  = 200
	StatusCreated             = 201
	StatusAccepted            = 202
	StatusNoContent           = 204
	StatusBadRequest          = 400
	StatusUnauthorized        = 401
	StatusForbidden           = 403
	StatusNotFound            = 404
	StatusConflict            = 409
	StatusPayloadTooLarge     = 413
	StatusTooManyRequests     = 429
	StatusServiceUnavailable  = 503
	StatusInsufficientStorage = 507
)

// File Permission Constants
const (
	PermUserOnly          = 0o700 // rwx------ (user read/write/execute only)
	PermUserRead          = 0o400 // r-------- (user read only)
	PermUserWrite         = 0o200 // -w------- (user write only)
	PermUserReadWrite     = 0o600 // rw------- (user read/write only)
	PermUserReadWriteExec = 0o700 // rwx------ (user read/write/execute only)
	PermGroupRead         = 0o040 // ---r----- (group read only)
	PermGroupWrite        = 0o020 // ----w---- (group write only)
	PermGroupReadWrite    = 0o060 // ---rw---- (group read/write only)
	PermOtherRead         = 0o004 // ------r-- (other read only)
	PermOtherWrite        = 0o002 // -------w- (other write only)
	PermOtherReadWrite    = 0o006 // ------rw- (other read/write only)
	PermStandardFile      = 0o644 // rw-r--r-- (user read/write, group/other read)
	PermExecutableFile    = 0o755 // rwxr-xr-x (user read/write/execute, group/other read/execute)
	PermSecureFile        = 0o600 // rw------- (user read/write only, secure)
	PermSecureDir         = 0o700 // rwx------ (user read/write/execute only, secure)
	PermStandardDir       = 0o755 // rwxr-xr-x (user read/write/execute, group/other read/execute)
)

// Default HTTP Configuration Constants
const (
	DefaultTimeout       = 30 * time.Second
	DefaultRetryAttempts = 3
	DefaultRetryDelay    = 1 * time.Second
	DefaultMaxRetryDelay = 10 * time.Second
)

// Default Polling Configuration Constants
const (
	DefaultPollInterval   = 2 * time.Second
	DefaultMaxInterval    = 30 * time.Second
	DefaultPollMultiplier = 1.5
)

// UI Display Constants
const (
	DefaultLineLength   = 120
	ShortLineLength     = 70
	MediumLineLength    = 80
	LongLineLength      = 100
	ExtraLongLineLength = 110
	MaxLineLength       = 120

	ProgressBarWidth    = 40
	SpinnerType         = 14
	ProgressBarThrottle = 100 * time.Millisecond

	MaxNameDisplayLength  = 57
	MaxShortNameLength    = 47
	MaxPermissionIDLength = 38
	MaxRolesDisplayLength = 23
	MaxVersionIDLength    = 37
	MaxActorNameLength    = 18
	MaxShortPathLength    = 50
)

// File and Path Constants
const (
	MaxFileNameLength = 255
	MaxPathLength     = 400
)

// Authentication Constants
const (
	DefaultDeviceCodeExpiry = 60 // minutes
)

// Buffer and Chunk Size Constants
const (
	DefaultChunkSize        = 320 * 1024 * 4         // 1.25 MB chunks for uploads
	DefaultBufferSize       = 1024                   // 1KB default buffer
	MinFileSize             = 1024                   // 1KB minimum file size
	LargeFileThreshold      = 4 * 1024 * 1024        // 4MB threshold for large files
	DefaultProgressInterval = 100 * time.Millisecond // Progress update frequency
)

// Time Format Constants
const (
	StandardTimeFormat = "2006-01-02 15:04"
	FullTimeFormat     = "2006-01-02 15:04:05"
)

// Table Display Constants
const (
	TableNameColumnWidth       = 50
	TableSizeColumnWidth       = 12
	TableTypeColumnWidth       = 10
	TableDateColumnWidth       = 20
	TablePermissionColumnWidth = 40
	TableActorColumnWidth      = 20
	TableActionColumnWidth     = 15
	TableVersionColumnWidth    = 10
)

// Separator Line Constants
const (
	StandardSeparatorLength   = 70
	MediumSeparatorLength     = 80
	LongSeparatorLength       = 95
	ExtraLongSeparatorLength  = 100
	PermissionSeparatorLength = 120
	ActivitySeparatorLength   = 90
)

// Content Formatting Constants
const (
	EllipsisMarker     = "..."
	EllipsisLength     = 3
	DefaultTableIndent = 4
)
