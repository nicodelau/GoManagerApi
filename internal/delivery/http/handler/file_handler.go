package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	fileService "gomanager/internal/application/file"
	domain "gomanager/internal/domain/file"
)

type FileHandler struct {
	service     fileService.Service
	maxFileSize int64
}

func NewFileHandler(service fileService.Service, maxFileSize int64) *FileHandler {
	return &FileHandler{
		service:     service,
		maxFileSize: maxFileSize,
	}
}

// List handles GET /api/files?path=...
func (h *FileHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := r.URL.Query().Get("path")
	files, err := h.service.ListFiles(path)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			SendError(w, "Directory not found", http.StatusNotFound)
			return
		}
		SendError(w, "Failed to read directory", http.StatusInternalServerError)
		return
	}

	SendSuccess(w, "", files)
}

// Upload handles POST /api/upload?path=...
func (h *FileHandler) Upload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseMultipartForm(h.maxFileSize); err != nil {
		SendError(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	targetPath := r.URL.Query().Get("path")
	files := r.MultipartForm.File["files"]

	if len(files) == 0 {
		SendError(w, "No files provided", http.StatusBadRequest)
		return
	}

	uploaded, err := h.service.UploadFiles(targetPath, files)
	if err != nil {
		SendError(w, "Failed to upload files", http.StatusInternalServerError)
		return
	}

	SendSuccess(w, fmt.Sprintf("Uploaded %d file(s)", len(uploaded)), uploaded)
}

// Download handles GET /api/download/{path}
func (h *FileHandler) Download(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	filePath := strings.TrimPrefix(r.URL.Path, "/api/download/")
	fullPath, err := h.service.GetFileForDownload(filePath)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			SendError(w, "File not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, domain.ErrIsDirectory) {
			SendError(w, "Cannot download directory", http.StatusBadRequest)
			return
		}
		SendError(w, "Failed to access file", http.StatusInternalServerError)
		return
	}

	// Check if this is a preview request (inline display)
	isPreview := r.URL.Query().Get("preview") == "true"

	filename := filepath.Base(fullPath)

	// Set appropriate Content-Type based on file extension
	contentType := getContentType(filename)
	w.Header().Set("Content-Type", contentType)

	if isPreview {
		// For preview, use inline disposition so browser displays the file
		w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", filename))
	} else {
		// For download, use attachment disposition
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	}

	http.ServeFile(w, r, fullPath)
}

// getContentType returns the MIME type based on file extension
func getContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	// Images
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".svg":
		return "image/svg+xml"
	case ".bmp":
		return "image/bmp"
	case ".ico":
		return "image/x-icon"
	// Videos
	case ".mp4":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	case ".ogg":
		return "video/ogg"
	case ".mov":
		return "video/quicktime"
	// Audio
	case ".mp3":
		return "audio/mpeg"
	case ".wav":
		return "audio/wav"
	case ".flac":
		return "audio/flac"
	case ".aac":
		return "audio/aac"
	case ".m4a":
		return "audio/mp4"
	// Documents
	case ".pdf":
		return "application/pdf"
	case ".txt":
		return "text/plain"
	case ".html", ".htm":
		return "text/html"
	case ".css":
		return "text/css"
	case ".js":
		return "application/javascript"
	case ".json":
		return "application/json"
	case ".xml":
		return "application/xml"
	default:
		return "application/octet-stream"
	}
}

// CreateFolder handles POST /api/mkdir
func (h *FileHandler) CreateFolder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req domain.CreateFolderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Path == "" {
		SendError(w, "Path is required", http.StatusBadRequest)
		return
	}

	if err := h.service.CreateFolder(req.Path); err != nil {
		SendError(w, "Failed to create directory", http.StatusInternalServerError)
		return
	}

	SendSuccess(w, "Directory created", nil)
}

// Delete handles POST /api/delete
func (h *FileHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req domain.DeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Path == "" {
		SendError(w, "Path is required", http.StatusBadRequest)
		return
	}

	if err := h.service.Delete(req.Path); err != nil {
		if errors.Is(err, domain.ErrRootDeletion) {
			SendError(w, "Cannot delete root directory", http.StatusForbidden)
			return
		}
		SendError(w, "Failed to delete", http.StatusInternalServerError)
		return
	}

	SendSuccess(w, "Deleted successfully", nil)
}

// Stats handles GET /api/stats
func (h *FileHandler) Stats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats, err := h.service.GetStats()
	if err != nil {
		SendError(w, "Failed to get stats", http.StatusInternalServerError)
		return
	}

	SendSuccess(w, "", stats)
}
