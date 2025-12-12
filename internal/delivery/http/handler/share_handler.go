package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	fileService "gomanager/internal/application/file"
	domain "gomanager/internal/domain/share"
)

type ShareHandler struct {
	shareRepo   domain.Repository
	fileService fileService.Service
	baseURL     string
}

func NewShareHandler(shareRepo domain.Repository, fileService fileService.Service, baseURL string) *ShareHandler {
	return &ShareHandler{
		shareRepo:   shareRepo,
		fileService: fileService,
		baseURL:     baseURL,
	}
}

// CreateShare handles POST /api/shares
func (h *ShareHandler) CreateShare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	u := GetUserFromContext(r.Context())
	if u == nil {
		SendError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req domain.CreateShareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Path == "" {
		SendError(w, "Path is required", http.StatusBadRequest)
		return
	}

	// Validate the path exists
	_, err := h.fileService.GetFileForDownload(req.Path)
	if err != nil {
		// Check if it's a directory by trying to list it
		_, listErr := h.fileService.ListFiles(req.Path)
		if listErr != nil {
			SendError(w, "Path not found", http.StatusNotFound)
			return
		}
	}

	// Set defaults
	if req.ShareType == "" {
		req.ShareType = domain.ShareTypePublic
	}
	if req.Permission == "" {
		req.Permission = domain.PermissionDownload
	}

	// Validate password for password-protected shares
	if req.ShareType == domain.ShareTypePassword && req.Password == "" {
		SendError(w, "Password is required for password-protected shares", http.StatusBadRequest)
		return
	}

	// Create share entity
	share := &domain.Share{
		Path:         req.Path,
		CreatedBy:    u.ID,
		ShareType:    req.ShareType,
		Password:     req.Password, // Will be hashed by repository
		Permission:   req.Permission,
		ExpiresAt:    req.ExpiresAt,
		MaxDownloads: req.MaxDownloads,
		IsActive:     true,
	}

	if err := h.shareRepo.Create(share); err != nil {
		SendError(w, "Failed to create share", http.StatusInternalServerError)
		return
	}

	SendSuccess(w, "Share created successfully", share.ToResponse(h.baseURL))
}

// ListUserShares handles GET /api/shares
func (h *ShareHandler) ListUserShares(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	u := GetUserFromContext(r.Context())
	if u == nil {
		SendError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	shares, err := h.shareRepo.GetByUser(u.ID)
	if err != nil {
		SendError(w, "Failed to retrieve shares", http.StatusInternalServerError)
		return
	}

	// Convert to responses
	responses := make([]domain.ShareResponse, len(shares))
	for i, share := range shares {
		responses[i] = share.ToResponse(h.baseURL)
	}

	SendSuccess(w, "", responses)
}

// DeleteShare handles DELETE /api/shares/{id}
func (h *ShareHandler) DeleteShare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		SendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	u := GetUserFromContext(r.Context())
	if u == nil {
		SendError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Extract share ID from path: /api/shares/{id}
	shareID := strings.TrimPrefix(r.URL.Path, "/api/shares/")
	if shareID == "" {
		SendError(w, "Share ID is required", http.StatusBadRequest)
		return
	}

	// Get the share to verify ownership
	share, err := h.shareRepo.GetByID(shareID)
	if err != nil {
		if errors.Is(err, domain.ErrShareNotFound) {
			SendError(w, "Share not found", http.StatusNotFound)
			return
		}
		SendError(w, "Failed to retrieve share", http.StatusInternalServerError)
		return
	}

	// Verify ownership
	if share.CreatedBy != u.ID {
		SendError(w, "Permission denied", http.StatusForbidden)
		return
	}

	if err := h.shareRepo.Delete(shareID); err != nil {
		SendError(w, "Failed to delete share", http.StatusInternalServerError)
		return
	}

	SendSuccess(w, "Share deleted successfully", nil)
}

// AccessShare handles GET /api/s/{token} - Public share access by token
func (h *ShareHandler) AccessShare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		SendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract token from path: /api/s/{token}
	token := strings.TrimPrefix(r.URL.Path, "/api/s/")
	if token == "" {
		SendError(w, "Share token is required", http.StatusBadRequest)
		return
	}

	share, err := h.shareRepo.GetByToken(token)
	if err != nil {
		if errors.Is(err, domain.ErrShareNotFound) {
			SendError(w, "Share not found", http.StatusNotFound)
			return
		}
		SendError(w, "Failed to retrieve share", http.StatusInternalServerError)
		return
	}

	// Check if share is still valid
	if !share.IsActive {
		SendError(w, "Share is no longer active", http.StatusGone)
		return
	}

	if share.IsExpired() {
		SendError(w, "Share has expired", http.StatusGone)
		return
	}

	if share.HasReachedMaxDownloads() {
		SendError(w, "Maximum downloads reached", http.StatusGone)
		return
	}

	// Handle password-protected shares
	if share.ShareType == domain.ShareTypePassword {
		if r.Method == http.MethodGet {
			// Return info that password is required
			SendJSON(w, http.StatusOK, Response{
				Success: true,
				Message: "Password required",
				Data: map[string]interface{}{
					"requiresPassword": true,
					"path":             share.Path,
				},
			})
			return
		}

		// POST - validate password
		var req domain.AccessShareRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			SendError(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Password validation should be done by comparing hashed passwords
		// This assumes the repository stores hashed passwords
		if req.Password != share.Password {
			SendError(w, "Invalid password", http.StatusUnauthorized)
			return
		}
	}

	// Get file/folder info
	files, err := h.fileService.ListFiles(share.Path)
	if err != nil {
		// It's a file, not a directory
		fullPath, fileErr := h.fileService.GetFileForDownload(share.Path)
		if fileErr != nil {
			SendError(w, "Shared content not found", http.StatusNotFound)
			return
		}

		// Increment download counter
		h.shareRepo.IncrementDownloads(share.ID)

		// For download permission, serve the file
		if share.Permission == domain.PermissionDownload {
			w.Header().Set("Content-Disposition", "attachment; filename=\""+strings.TrimPrefix(share.Path, "/")+"\"")
			w.Header().Set("Content-Type", "application/octet-stream")
			http.ServeFile(w, r, fullPath)
			return
		}
	}

	// For directories or view permission, return the file list
	SendSuccess(w, "", map[string]interface{}{
		"path":       share.Path,
		"permission": share.Permission,
		"files":      files,
	})
}

// GetShareInfo handles GET /api/shares/{id}/info
func (h *ShareHandler) GetShareInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	u := GetUserFromContext(r.Context())
	if u == nil {
		SendError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Extract share ID from path: /api/shares/{id}/info
	path := strings.TrimPrefix(r.URL.Path, "/api/shares/")
	shareID := strings.TrimSuffix(path, "/info")
	if shareID == "" {
		SendError(w, "Share ID is required", http.StatusBadRequest)
		return
	}

	share, err := h.shareRepo.GetByID(shareID)
	if err != nil {
		if errors.Is(err, domain.ErrShareNotFound) {
			SendError(w, "Share not found", http.StatusNotFound)
			return
		}
		SendError(w, "Failed to retrieve share", http.StatusInternalServerError)
		return
	}

	// Verify ownership
	if share.CreatedBy != u.ID {
		SendError(w, "Permission denied", http.StatusForbidden)
		return
	}

	SendSuccess(w, "", share.ToResponse(h.baseURL))
}

// HandleShares routes /api/shares based on method
func (h *ShareHandler) HandleShares(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.ListUserShares(w, r)
	case http.MethodPost:
		h.CreateShare(w, r)
	default:
		SendError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleShareByID routes /api/shares/{id} based on method
func (h *ShareHandler) HandleShareByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/shares/")

	// Check if it's /api/shares/{id}/info
	if strings.HasSuffix(path, "/info") {
		h.GetShareInfo(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.GetShareInfo(w, r)
	case http.MethodDelete:
		h.DeleteShare(w, r)
	default:
		SendError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
