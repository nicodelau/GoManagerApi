package googledrive

// DriveFile represents a file in Google Drive
type DriveFile struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	MimeType     string            `json:"mime_type"`
	Size         int64             `json:"size"`
	Parents      []string          `json:"parents"`
	CreatedTime  string            `json:"created_time"`
	ModifiedTime string            `json:"modified_time"`
	WebViewLink  string            `json:"web_view_link"`
	Permissions  []FilePermission  `json:"permissions"`
	Properties   map[string]string `json:"properties"`
}

// DriveFolder represents a folder in Google Drive
type DriveFolder struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	FolderID  string `json:"folder_id"`
	Name      string `json:"name"`
	Path      string `json:"path"`
	IsActive  bool   `json:"is_active"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// FilePermission represents sharing permissions for a file
type FilePermission struct {
	ID           string `json:"id"`
	Type         string `json:"type"` // user, group, domain, anyone
	Role         string `json:"role"` // owner, organizer, fileOrganizer, writer, commenter, reader
	EmailAddress string `json:"email_address,omitempty"`
	Domain       string `json:"domain,omitempty"`
}

// UploadRequest represents a file upload request
type UploadRequest struct {
	Name        string            `json:"name"`
	ParentID    string            `json:"parent_id"`
	MimeType    string            `json:"mime_type"`
	Content     []byte            `json:"-"`
	Properties  map[string]string `json:"properties,omitempty"`
	Description string            `json:"description,omitempty"`
}

// UploadResponse represents the response from a file upload
type UploadResponse struct {
	File    *DriveFile `json:"file"`
	Success bool       `json:"success"`
	Message string     `json:"message,omitempty"`
}

// FolderContents represents the contents of a folder
type FolderContents struct {
	Files         []*DriveFile `json:"files"`
	Folders       []*DriveFile `json:"folders"`
	NextPageToken string       `json:"next_page_token,omitempty"`
	TotalItems    int          `json:"total_items"`
}
