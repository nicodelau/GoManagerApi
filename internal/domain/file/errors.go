package file

import "errors"

var (
	ErrNotFound     = errors.New("file or directory not found")
	ErrInvalidPath  = errors.New("invalid path")
	ErrIsDirectory  = errors.New("cannot download a directory")
	ErrRootDeletion = errors.New("cannot delete root directory")
	ErrUploadFailed = errors.New("failed to upload files")
	ErrCreateFailed = errors.New("failed to create directory")
	ErrDeleteFailed = errors.New("failed to delete")
	ErrReadFailed   = errors.New("failed to read directory")
)
