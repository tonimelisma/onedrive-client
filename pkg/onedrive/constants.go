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
