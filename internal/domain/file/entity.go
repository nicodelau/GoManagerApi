package file

import "time"

// FileInfo represents a file or directory in the system
type FileInfo struct {
	Name    string    `json:"name"`
	Size    int64     `json:"size"`
	IsDir   bool      `json:"isDir"`
	ModTime time.Time `json:"modTime"`
	Path    string    `json:"path"`
}

// CreateFolderRequest represents a request to create a folder
type CreateFolderRequest struct {
	Path string `json:"path"`
}

// DeleteRequest represents a request to delete a file or folder
type DeleteRequest struct {
	Path string `json:"path"`
}

// StorageStats represents storage statistics
type StorageStats struct {
	TotalFiles   int64            `json:"totalFiles"`
	TotalFolders int64            `json:"totalFolders"`
	TotalSize    int64            `json:"totalSize"`
	FilesByType  map[string]int64 `json:"filesByType"`
	RecentFiles  []FileInfo       `json:"recentFiles"`
}
