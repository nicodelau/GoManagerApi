package share

import "time"

// ShareType represents the type of share
type ShareType string

const (
	ShareTypePublic   ShareType = "public"   // Anyone with link
	ShareTypePassword ShareType = "password" // Requires password
)

// Permission represents what the share allows
type Permission string

const (
	PermissionView     Permission = "view"
	PermissionDownload Permission = "download"
)

// Share represents a shared file or folder link
type Share struct {
	ID           string     `json:"id"`
	Token        string     `json:"token"` // Unique token for the share link
	Path         string     `json:"path"`  // Path to the shared file/folder
	CreatedBy    string     `json:"createdBy"`
	ShareType    ShareType  `json:"shareType"`
	Password     string     `json:"-"` // Hashed password for password-protected shares
	Permission   Permission `json:"permission"`
	ExpiresAt    *time.Time `json:"expiresAt,omitempty"`
	MaxDownloads *int       `json:"maxDownloads,omitempty"`
	Downloads    int        `json:"downloads"`
	CreatedAt    time.Time  `json:"createdAt"`
	IsActive     bool       `json:"isActive"`
}

// ShareResponse is the safe share representation for API responses
type ShareResponse struct {
	ID           string     `json:"id"`
	Token        string     `json:"token"`
	Path         string     `json:"path"`
	ShareType    ShareType  `json:"shareType"`
	Permission   Permission `json:"permission"`
	ExpiresAt    *time.Time `json:"expiresAt,omitempty"`
	MaxDownloads *int       `json:"maxDownloads,omitempty"`
	Downloads    int        `json:"downloads"`
	CreatedAt    time.Time  `json:"createdAt"`
	IsActive     bool       `json:"isActive"`
	URL          string     `json:"url"`
}

// CreateShareRequest represents a request to create a share
type CreateShareRequest struct {
	Path         string     `json:"path"`
	ShareType    ShareType  `json:"shareType"`
	Password     string     `json:"password,omitempty"`
	Permission   Permission `json:"permission"`
	ExpiresAt    *time.Time `json:"expiresAt,omitempty"`
	MaxDownloads *int       `json:"maxDownloads,omitempty"`
}

// AccessShareRequest represents a request to access a password-protected share
type AccessShareRequest struct {
	Password string `json:"password"`
}

// ToResponse converts a Share to ShareResponse
func (s *Share) ToResponse(baseURL string) ShareResponse {
	return ShareResponse{
		ID:           s.ID,
		Token:        s.Token,
		Path:         s.Path,
		ShareType:    s.ShareType,
		Permission:   s.Permission,
		ExpiresAt:    s.ExpiresAt,
		MaxDownloads: s.MaxDownloads,
		Downloads:    s.Downloads,
		CreatedAt:    s.CreatedAt,
		IsActive:     s.IsActive,
		URL:          baseURL + "/s/" + s.Token,
	}
}

// IsExpired returns true if the share has expired
func (s *Share) IsExpired() bool {
	if s.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*s.ExpiresAt)
}

// HasReachedMaxDownloads returns true if max downloads reached
func (s *Share) HasReachedMaxDownloads() bool {
	if s.MaxDownloads == nil {
		return false
	}
	return s.Downloads >= *s.MaxDownloads
}

// IsValid returns true if the share is still valid
func (s *Share) IsValid() bool {
	return s.IsActive && !s.IsExpired() && !s.HasReachedMaxDownloads()
}
