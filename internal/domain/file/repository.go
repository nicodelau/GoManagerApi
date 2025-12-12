package file

import "mime/multipart"

// Repository defines the contract for file storage operations
type Repository interface {
	List(path string) ([]FileInfo, error)
	GetFilePath(relativePath string) (string, error)
	Save(path string, files []*multipart.FileHeader) ([]string, error)
	CreateDirectory(path string) error
	Delete(path string) error
	Exists(path string) (bool, error)
	IsDirectory(path string) (bool, error)
	GetStats(excludePaths []string) (*StorageStats, error)
}
