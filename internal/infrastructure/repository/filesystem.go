package repository

import (
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"sort"
	"strings"

	domain "gomanager/internal/domain/file"
)

type filesystemRepository struct {
	basePath string
}

// NewFilesystemRepository creates a new filesystem-based repository
func NewFilesystemRepository(basePath string) domain.Repository {
	// Ensure base path exists
	os.MkdirAll(basePath, 0755)
	return &filesystemRepository{basePath: basePath}
}

// sanitizePath prevents directory traversal attacks
func (r *filesystemRepository) sanitizePath(path string) string {
	cleaned := filepath.Clean(path)
	cleaned = strings.TrimPrefix(cleaned, "/")
	if strings.HasPrefix(cleaned, "..") {
		return ""
	}
	return cleaned
}

func (r *filesystemRepository) getFullPath(relativePath string) string {
	sanitized := r.sanitizePath(relativePath)
	return filepath.Join(r.basePath, sanitized)
}

func (r *filesystemRepository) List(path string) ([]domain.FileInfo, error) {
	fullPath := r.getFullPath(path)

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, domain.ErrNotFound
		}
		return nil, domain.ErrReadFailed
	}

	files := make([]domain.FileInfo, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		filePath := path
		if filePath != "" {
			filePath = filepath.Join(filePath, entry.Name())
		} else {
			filePath = entry.Name()
		}

		files = append(files, domain.FileInfo{
			Name:    entry.Name(),
			Size:    info.Size(),
			IsDir:   entry.IsDir(),
			ModTime: info.ModTime(),
			Path:    filePath,
		})
	}

	// Sort: directories first, then by name
	sort.Slice(files, func(i, j int) bool {
		if files[i].IsDir != files[j].IsDir {
			return files[i].IsDir
		}
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})

	return files, nil
}

func (r *filesystemRepository) GetFilePath(relativePath string) (string, error) {
	fullPath := r.getFullPath(relativePath)

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return "", domain.ErrNotFound
	}

	return fullPath, nil
}

func (r *filesystemRepository) Save(path string, files []*multipart.FileHeader) ([]string, error) {
	fullPath := r.getFullPath(path)
	uploadedFiles := make([]string, 0, len(files))

	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			continue
		}

		filename := filepath.Base(fileHeader.Filename)
		destPath := filepath.Join(fullPath, filename)

		dst, err := os.Create(destPath)
		if err != nil {
			file.Close()
			continue
		}

		if _, err := io.Copy(dst, file); err != nil {
			file.Close()
			dst.Close()
			continue
		}

		file.Close()
		dst.Close()
		uploadedFiles = append(uploadedFiles, filename)
	}

	if len(uploadedFiles) == 0 {
		return nil, domain.ErrUploadFailed
	}

	return uploadedFiles, nil
}

func (r *filesystemRepository) CreateDirectory(path string) error {
	fullPath := r.getFullPath(path)
	if err := os.MkdirAll(fullPath, 0755); err != nil {
		return domain.ErrCreateFailed
	}
	return nil
}

func (r *filesystemRepository) Delete(path string) error {
	if path == "" {
		return domain.ErrRootDeletion
	}

	fullPath := r.getFullPath(path)

	// Prevent deleting the base storage directory
	absBase, _ := filepath.Abs(r.basePath)
	absFull, _ := filepath.Abs(fullPath)
	if absBase == absFull {
		return domain.ErrRootDeletion
	}

	if err := os.RemoveAll(fullPath); err != nil {
		return domain.ErrDeleteFailed
	}

	return nil
}

func (r *filesystemRepository) Exists(path string) (bool, error) {
	fullPath := r.getFullPath(path)
	_, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *filesystemRepository) IsDirectory(path string) (bool, error) {
	fullPath := r.getFullPath(path)
	info, err := os.Stat(fullPath)
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}

func (r *filesystemRepository) GetStats(excludePaths []string) (*domain.StorageStats, error) {
	stats := &domain.StorageStats{
		FilesByType: make(map[string]int64),
		RecentFiles: make([]domain.FileInfo, 0),
	}

	var allFiles []domain.FileInfo

	err := filepath.Walk(r.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		// Get relative path
		relPath, _ := filepath.Rel(r.basePath, path)
		if relPath == "." {
			return nil
		}

		// Check if path should be excluded
		for _, exclude := range excludePaths {
			if strings.HasPrefix(relPath, exclude) || relPath == exclude {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		if info.IsDir() {
			stats.TotalFolders++
		} else {
			stats.TotalFiles++
			stats.TotalSize += info.Size()

			// Count by file extension
			ext := strings.ToLower(filepath.Ext(info.Name()))
			if ext == "" {
				ext = "no extension"
			}
			stats.FilesByType[ext]++

			// Collect for recent files
			allFiles = append(allFiles, domain.FileInfo{
				Name:    info.Name(),
				Size:    info.Size(),
				IsDir:   false,
				ModTime: info.ModTime(),
				Path:    relPath,
			})
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort by modification time (newest first) and take top 10
	sort.Slice(allFiles, func(i, j int) bool {
		return allFiles[i].ModTime.After(allFiles[j].ModTime)
	})

	if len(allFiles) > 10 {
		stats.RecentFiles = allFiles[:10]
	} else {
		stats.RecentFiles = allFiles
	}

	return stats, nil
}
