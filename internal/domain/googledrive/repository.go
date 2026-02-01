package googledrive

// Repository defines the interface for Google Drive operations
type Repository interface {
	// Folder Management
	CreateFolder(userID, name, parentID string) (*DriveFolder, error)
	GetFolder(userID, folderID string) (*DriveFolder, error)
	ListUserFolders(userID string) ([]*DriveFolder, error)
	UpdateFolder(userID, folderID string, updates map[string]interface{}) error
	DeleteFolder(userID, folderID string) error

	// File Operations
	UploadFile(userID string, request *UploadRequest) (*UploadResponse, error)
	GetFile(userID, fileID string) (*DriveFile, error)
	ListFolderContents(userID, folderID string, pageToken string) (*FolderContents, error)
	DownloadFile(userID, fileID string) ([]byte, error)
	DeleteFile(userID, fileID string) error
	MoveFile(userID, fileID, newParentID string) error
	CopyFile(userID, fileID, newParentID, newName string) (*DriveFile, error)

	// Permissions
	ShareFile(userID, fileID string, permission *FilePermission) error
	GetFilePermissions(userID, fileID string) ([]*FilePermission, error)
	RemoveFilePermission(userID, fileID, permissionID string) error

	// Search
	SearchFiles(userID, query string) ([]*DriveFile, error)
}
