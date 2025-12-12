package share

import "errors"

var (
	ErrShareNotFound    = errors.New("share not found")
	ErrShareExpired     = errors.New("share has expired")
	ErrShareInactive    = errors.New("share is no longer active")
	ErrMaxDownloads     = errors.New("maximum downloads reached")
	ErrInvalidPassword  = errors.New("invalid password")
	ErrPasswordRequired = errors.New("password required")
	ErrInvalidPath      = errors.New("invalid path")
	ErrPermissionDenied = errors.New("permission denied")
)
