package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"gomanager/internal/application/auth"
	"gomanager/internal/domain/user"

	"github.com/google/uuid"
)

// UserHandler handles user profile operations
type UserHandler struct {
	authService auth.Service
	userRepo    user.Repository
	avatarPath  string
}

// NewUserHandler creates a new user handler
func NewUserHandler(authService auth.Service, userRepo user.Repository, storagePath string) *UserHandler {
	avatarPath := filepath.Join(storagePath, ".avatars")
	os.MkdirAll(avatarPath, 0755)

	return &UserHandler{
		authService: authService,
		userRepo:    userRepo,
		avatarPath:  avatarPath,
	}
}

// UpdateProfileRequest represents the request to update user profile
type UpdateProfileRequest struct {
	Username string `json:"username,omitempty"`
	Email    string `json:"email,omitempty"`
}

// UpdatePasswordRequest represents the request to change password
type UpdatePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

// GetProfile handles GET /api/user/profile
func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	u := GetUserFromContext(r.Context())
	if u == nil {
		SendError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	SendSuccess(w, "", u.ToResponse())
}

// UpdateProfile handles PUT /api/user/profile
func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		SendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	u := GetUserFromContext(r.Context())
	if u == nil {
		SendError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Update fields if provided
	if req.Username != "" && req.Username != u.Username {
		if len(req.Username) < 3 {
			SendError(w, "Username must be at least 3 characters", http.StatusBadRequest)
			return
		}
		// Check if username is taken
		existing, _ := h.userRepo.GetByUsername(req.Username)
		if existing != nil && existing.ID != u.ID {
			SendError(w, "Username already taken", http.StatusConflict)
			return
		}
		u.Username = req.Username
	}

	if req.Email != "" && req.Email != u.Email {
		// Check if email is taken
		existing, _ := h.userRepo.GetByEmail(req.Email)
		if existing != nil && existing.ID != u.ID {
			SendError(w, "Email already in use", http.StatusConflict)
			return
		}
		u.Email = req.Email
	}

	if err := h.userRepo.Update(u); err != nil {
		SendError(w, "Failed to update profile", http.StatusInternalServerError)
		return
	}

	SendSuccess(w, "Profile updated successfully", u.ToResponse())
}

// UpdatePassword handles PUT /api/user/password
func (h *UserHandler) UpdatePassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		SendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	u := GetUserFromContext(r.Context())
	if u == nil {
		SendError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Google users can't change password
	if u.AuthProvider == user.AuthProviderGoogle {
		SendError(w, "Google users cannot change password", http.StatusBadRequest)
		return
	}

	var req UpdatePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.CurrentPassword == "" || req.NewPassword == "" {
		SendError(w, "Current and new password are required", http.StatusBadRequest)
		return
	}

	if len(req.NewPassword) < 6 {
		SendError(w, "New password must be at least 6 characters", http.StatusBadRequest)
		return
	}

	// Verify current password
	if !h.authService.CheckPassword(u.Password, req.CurrentPassword) {
		SendError(w, "Current password is incorrect", http.StatusUnauthorized)
		return
	}

	// Hash new password
	hashedPassword, err := h.authService.HashPassword(req.NewPassword)
	if err != nil {
		SendError(w, "Failed to update password", http.StatusInternalServerError)
		return
	}

	u.Password = hashedPassword
	if err := h.userRepo.Update(u); err != nil {
		SendError(w, "Failed to update password", http.StatusInternalServerError)
		return
	}

	SendSuccess(w, "Password updated successfully", nil)
}

// UploadAvatar handles POST /api/user/avatar
func (h *UserHandler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	u := GetUserFromContext(r.Context())
	if u == nil {
		SendError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse multipart form (max 5MB for avatar)
	if err := r.ParseMultipartForm(5 << 20); err != nil {
		SendError(w, "File too large (max 5MB)", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("avatar")
	if err != nil {
		SendError(w, "No file provided", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Validate file type
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".gif" && ext != ".webp" {
		SendError(w, "Invalid file type. Allowed: jpg, jpeg, png, gif, webp", http.StatusBadRequest)
		return
	}

	// Delete old avatar if exists and is local
	if u.AvatarURL != "" && strings.HasPrefix(u.AvatarURL, "/api/user/avatar/") {
		oldPath := filepath.Join(h.avatarPath, filepath.Base(u.AvatarURL))
		os.Remove(oldPath)
	}

	// Generate unique filename
	filename := uuid.New().String() + ext
	filePath := filepath.Join(h.avatarPath, filename)

	// Save file
	dst, err := os.Create(filePath)
	if err != nil {
		SendError(w, "Failed to save avatar", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		SendError(w, "Failed to save avatar", http.StatusInternalServerError)
		return
	}

	// Update user avatar URL
	u.AvatarURL = "/api/user/avatar/" + filename
	if err := h.userRepo.Update(u); err != nil {
		SendError(w, "Failed to update avatar", http.StatusInternalServerError)
		return
	}

	SendSuccess(w, "Avatar uploaded successfully", map[string]string{
		"avatarUrl": u.AvatarURL,
	})
}

// ServeAvatar handles GET /api/user/avatar/{filename}
func (h *UserHandler) ServeAvatar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract filename from path
	filename := strings.TrimPrefix(r.URL.Path, "/api/user/avatar/")
	if filename == "" {
		SendError(w, "Avatar not found", http.StatusNotFound)
		return
	}

	// Prevent directory traversal
	filename = filepath.Base(filename)
	filePath := filepath.Join(h.avatarPath, filename)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		SendError(w, "Avatar not found", http.StatusNotFound)
		return
	}

	// Serve file
	http.ServeFile(w, r, filePath)
}

// DeleteAvatar handles DELETE /api/user/avatar
func (h *UserHandler) DeleteAvatar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		SendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	u := GetUserFromContext(r.Context())
	if u == nil {
		SendError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Delete file if it's a local avatar
	if u.AvatarURL != "" && strings.HasPrefix(u.AvatarURL, "/api/user/avatar/") {
		filePath := filepath.Join(h.avatarPath, filepath.Base(u.AvatarURL))
		os.Remove(filePath)
	}

	u.AvatarURL = ""
	if err := h.userRepo.Update(u); err != nil {
		SendError(w, "Failed to delete avatar", http.StatusInternalServerError)
		return
	}

	SendSuccess(w, "Avatar deleted successfully", nil)
}
