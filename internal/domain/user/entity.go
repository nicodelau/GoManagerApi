package user

import "time"

// Role represents user roles in the system
type Role string

const (
	RoleAdmin  Role = "admin"
	RoleUser   Role = "user"
	RoleViewer Role = "viewer"
)

// AuthProvider represents the authentication provider
type AuthProvider string

const (
	AuthProviderLocal  AuthProvider = "local"
	AuthProviderGoogle AuthProvider = "google"
)

// User represents a user in the system
type User struct {
	ID           string       `json:"id"`
	Email        string       `json:"email"`
	Username     string       `json:"username"`
	Password     string       `json:"-"` // Never expose password in JSON
	Role         Role         `json:"role"`
	AuthProvider AuthProvider `json:"authProvider"`
	GoogleID     string       `json:"-"`
	GoogleToken  string       `json:"-"` // Google OAuth refresh token for API access
	AvatarURL    string       `json:"avatarUrl,omitempty"`
	CreatedAt    time.Time    `json:"createdAt"`
	UpdatedAt    time.Time    `json:"updatedAt"`
}

// UserResponse is the safe user representation for API responses
type UserResponse struct {
	ID           string       `json:"id"`
	Email        string       `json:"email"`
	Username     string       `json:"username"`
	Role         Role         `json:"role"`
	AuthProvider AuthProvider `json:"authProvider"`
	AvatarURL    string       `json:"avatarUrl,omitempty"`
	CreatedAt    time.Time    `json:"createdAt"`
}

// ToResponse converts a User to UserResponse
func (u *User) ToResponse() UserResponse {
	return UserResponse{
		ID:           u.ID,
		Email:        u.Email,
		Username:     u.Username,
		Role:         u.Role,
		AuthProvider: u.AuthProvider,
		AvatarURL:    u.AvatarURL,
		CreatedAt:    u.CreatedAt,
	}
}

// CreateUserRequest represents the request to create a user
type CreateUserRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// UpdateUserRequest represents the request to update a user
type UpdateUserRequest struct {
	Email    string `json:"email,omitempty"`
	Username string `json:"username,omitempty"`
	Role     Role   `json:"role,omitempty"`
}

// CanManageUsers returns true if the role can manage other users
func (r Role) CanManageUsers() bool {
	return r == RoleAdmin
}

// CanUpload returns true if the role can upload files
func (r Role) CanUpload() bool {
	return r == RoleAdmin || r == RoleUser
}

// CanDelete returns true if the role can delete files
func (r Role) CanDelete() bool {
	return r == RoleAdmin || r == RoleUser
}

// CanShare returns true if the role can share files
func (r Role) CanShare() bool {
	return r == RoleAdmin || r == RoleUser
}
