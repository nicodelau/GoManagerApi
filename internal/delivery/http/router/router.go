package router

import (
	"net/http"

	"gomanager/internal/application/auth"
	"gomanager/internal/delivery/http/handler"
	"gomanager/internal/delivery/http/middleware"
	"gomanager/internal/domain/user"
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
	mux := http.NewServeMux()

	// Middleware helpers
	cors := middleware.CORS
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
	// Auth routes (public)
	// ==================
	mux.HandleFunc("/api/auth/register", cors(handlers.Auth.Register))
	mux.HandleFunc("/api/auth/login", cors(handlers.Auth.Login))
	mux.HandleFunc("/api/auth/logout", chain(handlers.Auth.Logout, cors, authRequired))
	mux.HandleFunc("/api/auth/me", chain(handlers.Auth.Me, cors, authRequired))

	// ==================
	// Google OAuth routes (public)
	// ==================
	if handlers.OAuth != nil {
		mux.HandleFunc("/api/auth/google", cors(handlers.OAuth.GoogleLogin))
		mux.HandleFunc("/api/auth/google/callback", handlers.OAuth.GoogleCallback)
		mux.HandleFunc("/api/auth/google/status", cors(handlers.OAuth.GoogleStatus))
	}

	// ==================
	// File routes (protected)
	// ==================
	mux.HandleFunc("/api/files", chain(handlers.File.List, cors, authRequired))
	mux.HandleFunc("/api/stats", chain(handlers.File.Stats, cors, authRequired))
	mux.HandleFunc("/api/upload", chain(handlers.File.Upload, cors, authRequired, canUpload))
	mux.HandleFunc("/api/download/", chain(handlers.File.Download, cors, authRequired))
	mux.HandleFunc("/api/mkdir", chain(handlers.File.CreateFolder, cors, authRequired, canUpload))
	mux.HandleFunc("/api/delete", chain(handlers.File.Delete, cors, authRequired, canUpload))

	// ==================
	// Share routes
	// ==================
	mux.HandleFunc("/api/shares", chain(handlers.Share.HandleShares, cors, authRequired))
	mux.HandleFunc("/api/shares/", chain(handlers.Share.HandleShareByID, cors, authRequired))

	// Public share access (no auth required)
	mux.HandleFunc("/api/s/", chain(handlers.Share.AccessShare, cors, optionalAuth))

	// ==================
	// Admin routes
	// ==================
	_ = adminOnly // Will be used for user management endpoints

	// ==================
	// User profile routes (protected)
	// ==================
	if handlers.User != nil {
		mux.HandleFunc("/api/user/profile", chain(handlers.User.GetProfile, cors, authRequired))
		mux.HandleFunc("/api/user/profile/update", chain(handlers.User.UpdateProfile, cors, authRequired))
		mux.HandleFunc("/api/user/password", chain(handlers.User.UpdatePassword, cors, authRequired))
		mux.HandleFunc("/api/user/avatar", chain(handlers.User.UploadAvatar, cors, authRequired))
		mux.HandleFunc("/api/user/avatar/delete", chain(handlers.User.DeleteAvatar, cors, authRequired))
		mux.HandleFunc("/api/user/avatar/", cors(handlers.User.ServeAvatar)) // Public for serving images
	}

	// ==================
	// Google Services routes (protected)
	// ==================
	if handlers.GoogleServices != nil {
		mux.HandleFunc("/api/google/status", chain(handlers.GoogleServices.GoogleConnectionStatus, cors, authRequired))
		mux.HandleFunc("/api/google/calendars", chain(handlers.GoogleServices.ListCalendars, cors, authRequired))
		mux.HandleFunc("/api/google/calendar/events", chain(handlers.GoogleServices.ListEvents, cors, authRequired))
		mux.HandleFunc("/api/google/calendar/events/create", chain(handlers.GoogleServices.CreateEvent, cors, authRequired))
		mux.HandleFunc("/api/google/tasks/lists", chain(handlers.GoogleServices.ListTaskLists, cors, authRequired))
		mux.HandleFunc("/api/google/tasks", chain(handlers.GoogleServices.ListTasks, cors, authRequired))
		mux.HandleFunc("/api/google/tasks/create", chain(handlers.GoogleServices.CreateTask, cors, authRequired))
		mux.HandleFunc("/api/google/tasks/update", chain(handlers.GoogleServices.UpdateTask, cors, authRequired))
		mux.HandleFunc("/api/google/tasks/complete", chain(handlers.GoogleServices.CompleteTask, cors, authRequired))

		// Google Drive routes
		mux.HandleFunc("/api/google/drive/files", chain(handlers.GoogleServices.ListDriveFiles, cors, authRequired))
		mux.HandleFunc("/api/google/drive/folders", chain(handlers.GoogleServices.CreateDriveFolder, cors, authRequired))
		mux.HandleFunc("/api/google/drive/upload", chain(handlers.GoogleServices.UploadDriveFile, cors, authRequired))
		mux.HandleFunc("/api/google/drive/delete", chain(handlers.GoogleServices.DeleteDriveFile, cors, authRequired))
	}

	// ==================
	// Google Ads routes (protected)
	// ==================
	if handlers.GoogleAds != nil {
		mux.HandleFunc("/api/google/ads/status", chain(handlers.GoogleAds.GoogleAdsStatus, cors, authRequired))
		mux.HandleFunc("/api/google/ads/campaigns", chain(handlers.GoogleAds.ListCampaigns, cors, authRequired))
		mux.HandleFunc("/api/google/ads/campaigns/create", chain(handlers.GoogleAds.CreateCampaign, cors, authRequired))
		mux.HandleFunc("/api/google/ads/campaigns/performance", chain(handlers.GoogleAds.GetCampaignPerformance, cors, authRequired))
	}

	return mux
}
