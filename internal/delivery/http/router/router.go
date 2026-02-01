package router

import (
	"net/http"
	"time"

	"gomanager/internal/application/auth"
	"gomanager/internal/delivery/http/handler"
	"gomanager/internal/delivery/http/middleware"
	"gomanager/internal/domain/user"
	"gomanager/internal/infrastructure/config"
)

// Handlers holds all HTTP handlers
type Handlers struct {
	File           *handler.FileHandler
	Auth           *handler.AuthHandler
	Share          *handler.ShareHandler
	OAuth          *handler.OAuthHandler
	User           *handler.UserHandler
	GoogleServices *handler.GoogleServicesHandler
	GoogleAds      *handler.GoogleAdsHandler
}

// Setup configures all routes for the application
func Setup(handlers Handlers, authService auth.Service) *http.ServeMux {
	return SetupWithConfig(handlers, authService, nil)
}

// SetupWithConfig configures all routes for the application with custom configuration
func SetupWithConfig(handlers Handlers, authService auth.Service, cfg *config.Config) *http.ServeMux {
	mux := http.NewServeMux()

	// Configure CORS - always include localhost for development
	allowedOrigins := []string{
		"http://localhost:5173",        // Development frontend
		"http://localhost:3000",        // Alternative development port
		"https://gomanager.render.app", // Production frontend
	}

	// Add configured frontend URL if present
	if cfg != nil && cfg.FrontendURL != "" {
		allowedOrigins = append(allowedOrigins, cfg.FrontendURL)
	}

	corsConfig := middleware.CORSConfig{
		AllowedOrigins: allowedOrigins,
	}
	corsMiddleware := func(next http.HandlerFunc) http.HandlerFunc {
		return middleware.CORSWithConfig(corsConfig, next)
	}

	// Middleware helpers
	authRequired := middleware.Auth(authService)
	optionalAuth := middleware.OptionalAuth(authService)
	adminOnly := middleware.RequireRole(user.RoleAdmin)
	canUpload := middleware.RequireRole(user.RoleAdmin, user.RoleUser)

	// Chain helper
	chain := func(h http.HandlerFunc, middlewares ...func(http.HandlerFunc) http.HandlerFunc) http.HandlerFunc {
		for i := len(middlewares) - 1; i >= 0; i-- {
			h = middlewares[i](h)
		}
		return h
	}

	// ==================
	// Health check route (public)
	// ==================
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","message":"GoManager API is running"}`))
	})
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy","timestamp":"` + time.Now().Format(time.RFC3339) + `"}`))
	})

	// ==================
	// Auth routes (public)
	// ==================
	mux.HandleFunc("/api/auth/register", corsMiddleware(handlers.Auth.Register))
	mux.HandleFunc("/api/auth/login", corsMiddleware(handlers.Auth.Login))
	mux.HandleFunc("/api/auth/logout", chain(handlers.Auth.Logout, corsMiddleware, authRequired))
	mux.HandleFunc("/api/auth/me", chain(handlers.Auth.Me, corsMiddleware, authRequired))

	// ==================
	// Google OAuth routes (public)
	// ==================
	if handlers.OAuth != nil {
		mux.HandleFunc("/api/auth/google", corsMiddleware(handlers.OAuth.GoogleLogin))
		mux.HandleFunc("/api/auth/google/callback", handlers.OAuth.GoogleCallback)
		mux.HandleFunc("/api/auth/google/status", corsMiddleware(handlers.OAuth.GoogleStatus))
	}

	// ==================
	// File routes (protected)
	// ==================
	mux.HandleFunc("/api/files", chain(handlers.File.List, corsMiddleware, authRequired))
	mux.HandleFunc("/api/stats", chain(handlers.File.Stats, corsMiddleware, authRequired))
	mux.HandleFunc("/api/upload", chain(handlers.File.Upload, corsMiddleware, authRequired, canUpload))
	mux.HandleFunc("/api/download/", chain(handlers.File.Download, corsMiddleware, authRequired))
	mux.HandleFunc("/api/mkdir", chain(handlers.File.CreateFolder, corsMiddleware, authRequired, canUpload))
	mux.HandleFunc("/api/delete", chain(handlers.File.Delete, corsMiddleware, authRequired, canUpload))

	// ==================
	// Share routes
	// ==================
	mux.HandleFunc("/api/shares", chain(handlers.Share.HandleShares, corsMiddleware, authRequired))
	mux.HandleFunc("/api/shares/", chain(handlers.Share.HandleShareByID, corsMiddleware, authRequired))

	// Public share access (no auth required)
	mux.HandleFunc("/api/s/", chain(handlers.Share.AccessShare, corsMiddleware, optionalAuth))

	// ==================
	// Admin routes
	// ==================
	_ = adminOnly // Will be used for user management endpoints

	// ==================
	// User profile routes (protected)
	// ==================
	if handlers.User != nil {
		mux.HandleFunc("/api/user/profile", chain(handlers.User.GetProfile, corsMiddleware, authRequired))
		mux.HandleFunc("/api/user/profile/update", chain(handlers.User.UpdateProfile, corsMiddleware, authRequired))
		mux.HandleFunc("/api/user/password", chain(handlers.User.UpdatePassword, corsMiddleware, authRequired))
		mux.HandleFunc("/api/user/avatar", chain(handlers.User.UploadAvatar, corsMiddleware, authRequired))
		mux.HandleFunc("/api/user/avatar/delete", chain(handlers.User.DeleteAvatar, corsMiddleware, authRequired))
		mux.HandleFunc("/api/user/avatar/", corsMiddleware(handlers.User.ServeAvatar)) // Public for serving images
	}

	// ==================
	// Google Services routes (protected)
	// ==================
	if handlers.GoogleServices != nil {
		mux.HandleFunc("/api/google/status", chain(handlers.GoogleServices.GoogleConnectionStatus, corsMiddleware, authRequired))
		mux.HandleFunc("/api/google/calendars", chain(handlers.GoogleServices.ListCalendars, corsMiddleware, authRequired))
		mux.HandleFunc("/api/google/calendar/events", chain(handlers.GoogleServices.ListEvents, corsMiddleware, authRequired))
		mux.HandleFunc("/api/google/calendar/events/create", chain(handlers.GoogleServices.CreateEvent, corsMiddleware, authRequired))
		mux.HandleFunc("/api/google/tasks/lists", chain(handlers.GoogleServices.ListTaskLists, corsMiddleware, authRequired))
		mux.HandleFunc("/api/google/tasks", chain(handlers.GoogleServices.ListTasks, corsMiddleware, authRequired))
		mux.HandleFunc("/api/google/tasks/create", chain(handlers.GoogleServices.CreateTask, corsMiddleware, authRequired))
		mux.HandleFunc("/api/google/tasks/update", chain(handlers.GoogleServices.UpdateTask, corsMiddleware, authRequired))
		mux.HandleFunc("/api/google/tasks/complete", chain(handlers.GoogleServices.CompleteTask, corsMiddleware, authRequired))

		// Google Drive routes
		mux.HandleFunc("/api/google/drive/files", chain(handlers.GoogleServices.ListDriveFiles, corsMiddleware, authRequired))
		mux.HandleFunc("/api/google/drive/folders", chain(handlers.GoogleServices.CreateDriveFolder, corsMiddleware, authRequired))
		mux.HandleFunc("/api/google/drive/upload", chain(handlers.GoogleServices.UploadDriveFile, corsMiddleware, authRequired))
		mux.HandleFunc("/api/google/drive/delete", chain(handlers.GoogleServices.DeleteDriveFile, corsMiddleware, authRequired))
	}

	// ==================
	// Google Ads routes (protected)
	// ==================
	if handlers.GoogleAds != nil {
		mux.HandleFunc("/api/google/ads/status", chain(handlers.GoogleAds.GoogleAdsStatus, corsMiddleware, authRequired))
		mux.HandleFunc("/api/google/ads/campaigns", chain(handlers.GoogleAds.ListCampaigns, corsMiddleware, authRequired))
		mux.HandleFunc("/api/google/ads/campaigns/create", chain(handlers.GoogleAds.CreateCampaign, corsMiddleware, authRequired))
		mux.HandleFunc("/api/google/ads/campaigns/performance", chain(handlers.GoogleAds.GetCampaignPerformance, corsMiddleware, authRequired))
	}

	return mux
}
