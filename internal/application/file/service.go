package file

import (
	"mime/multipart"
	"strings"

	domain "gomanager/internal/domain/file"
)

// Hidden folders/files that should not be shown in listings
var hiddenPaths = []string{".avatars"}

// Service defines the business logic for file operations
type Service interface {
	ListFiles(path string) ([]domain.FileInfo, error)
	GetFileForDownload(path string) (string, error)
	UploadFiles(path string, files []*multipart.FileHeader) ([]string, error)
	CreateFolder(path string) error
	Delete(path string) error
	GetStats() (*domain.StorageStats, error)
}

type service struct {
	repo domain.Repository
}

// NewService creates a new file service
func NewService(repo domain.Repository) Service {
	return &service{repo: repo}
}

func (s *service) ListFiles(path string) ([]domain.FileInfo, error) {
	files, err := s.repo.List(path)
	if err != nil {
		return nil, err
	}

	// Filter out hidden files/folders at root level
	if path == "" || path == "/" {
		filtered := make([]domain.FileInfo, 0, len(files))
		for _, f := range files {
			if !isHidden(f.Name) {
				filtered = append(filtered, f)
			}
		}
		return filtered, nil
	}

	return files, nil
}

// isHidden checks if a file/folder name should be hidden
func isHidden(name string) bool {
	for _, hidden := range hiddenPaths {
		if strings.EqualFold(name, hidden) {
			return true
		}
	}
	return false
}

func (s *service) GetFileForDownload(path string) (string, error) {
	isDir, err := s.repo.IsDirectory(path)
	if err != nil {
		return "", domain.ErrNotFound
	}

	if isDir {
		return "", domain.ErrIsDirectory
	}

	return s.repo.GetFilePath(path)
}

func (s *service) UploadFiles(path string, files []*multipart.FileHeader) ([]string, error) {
	if err := s.repo.CreateDirectory(path); err != nil {
		return nil, domain.ErrCreateFailed
	}

	uploaded, err := s.repo.Save(path, files)
	if err != nil {
		return nil, domain.ErrUploadFailed
	}

	return uploaded, nil
}

func (s *service) CreateFolder(path string) error {
	if path == "" {
		return domain.ErrInvalidPath
	}
	return s.repo.CreateDirectory(path)
}

func (s *service) Delete(path string) error {
	if path == "" {
		return domain.ErrRootDeletion
	}
	return s.repo.Delete(path)
}

func (s *service) GetStats() (*domain.StorageStats, error) {
	return s.repo.GetStats(hiddenPaths)
}
