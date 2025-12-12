package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"gomanager/internal/application/auth"
	domain "gomanager/internal/domain/auth"
	"gomanager/internal/domain/user"
)

type AuthHandler struct {
	service auth.Service
}

func NewAuthHandler(service auth.Service) *AuthHandler {
	return &AuthHandler{
		service: service,
	}
}

// Register handles POST /api/auth/register
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req domain.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Username == "" || req.Password == "" {
		SendError(w, "Email, username, and password are required", http.StatusBadRequest)
		return
	}

	newUser, err := h.service.Register(req)
	if err != nil {
		switch {
		case errors.Is(err, user.ErrUserAlreadyExists):
			SendError(w, "User already exists", http.StatusConflict)
		case errors.Is(err, user.ErrInvalidEmail):
			SendError(w, "Invalid email address", http.StatusBadRequest)
		case errors.Is(err, user.ErrInvalidUsername):
			SendError(w, "Username must be at least 3 characters", http.StatusBadRequest)
		case errors.Is(err, user.ErrInvalidPassword):
			SendError(w, "Password must be at least 6 characters", http.StatusBadRequest)
		default:
			SendError(w, "Failed to register user", http.StatusInternalServerError)
		}
		return
	}

	SendSuccess(w, "User registered successfully", newUser.ToResponse())
}

// Login handles POST /api/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req domain.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" {
		SendError(w, "Email and password are required", http.StatusBadRequest)
		return
	}

	resp, err := h.service.Login(req)
	if err != nil {
		if errors.Is(err, user.ErrInvalidCredentials) {
			SendError(w, "Invalid email or password", http.StatusUnauthorized)
			return
		}
		SendError(w, "Failed to login", http.StatusInternalServerError)
		return
	}

	SendSuccess(w, "Login successful", resp)
}

// Logout handles POST /api/auth/logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token := extractToken(r)
	if token == "" {
		SendError(w, "Authorization required", http.StatusUnauthorized)
		return
	}

	if err := h.service.Logout(token); err != nil {
		SendError(w, "Failed to logout", http.StatusInternalServerError)
		return
	}

	SendSuccess(w, "Logged out successfully", nil)
}

// Me handles GET /api/auth/me
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token := extractToken(r)
	if token == "" {
		SendError(w, "Authorization required", http.StatusUnauthorized)
		return
	}

	u, err := h.service.ValidateToken(token)
	if err != nil {
		SendError(w, "Invalid or expired token", http.StatusUnauthorized)
		return
	}

	SendSuccess(w, "", u.ToResponse())
}

func extractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}

	if token := r.URL.Query().Get("token"); token != "" {
		return token
	}

	return ""
}
