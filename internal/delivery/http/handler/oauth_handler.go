package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"gomanager/internal/application/auth"
	authDomain "gomanager/internal/domain/auth"
	"gomanager/internal/domain/user"
	"gomanager/internal/infrastructure/config"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// GoogleUserInfo represents the user info returned by Google
type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
}

// OAuthHandler handles Google OAuth authentication
type OAuthHandler struct {
	oauthConfig *oauth2.Config
	authService auth.Service
	userRepo    user.Repository
	frontendURL string
}

// NewOAuthHandler creates a new OAuth handler
func NewOAuthHandler(cfg *config.Config, authService auth.Service, userRepo user.Repository) *OAuthHandler {
	oauthConfig := &oauth2.Config{
		ClientID:     cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		RedirectURL:  cfg.BaseURL + "/api/auth/google/callback",
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
			"https://www.googleapis.com/auth/calendar.readonly",
			"https://www.googleapis.com/auth/calendar.events",
		},
		Endpoint: google.Endpoint,
	}

	return &OAuthHandler{
		oauthConfig: oauthConfig,
		authService: authService,
		userRepo:    userRepo,
		frontendURL: cfg.FrontendURL,
	}
}

// GoogleLogin redirects to Google OAuth login page
func (h *OAuthHandler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	if h.oauthConfig.ClientID == "" {
		SendError(w, "Google OAuth not configured", http.StatusServiceUnavailable)
		return
	}

	// Generate state token
	state := uuid.New().String()

	// Store state in cookie for verification
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		Secure:   strings.HasPrefix(h.frontendURL, "https"),
		MaxAge:   600, // 10 minutes
		SameSite: http.SameSiteLaxMode,
	})

	// Request offline access to get refresh token
	url := h.oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// GoogleCallback handles the OAuth callback from Google
func (h *OAuthHandler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	// Verify state
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil {
		h.redirectWithError(w, r, "Invalid state")
		return
	}

	state := r.URL.Query().Get("state")
	if state != stateCookie.Value {
		h.redirectWithError(w, r, "State mismatch")
		return
	}

	// Clear state cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	// Check for error from Google
	if errMsg := r.URL.Query().Get("error"); errMsg != "" {
		h.redirectWithError(w, r, errMsg)
		return
	}

	// Exchange code for token
	code := r.URL.Query().Get("code")
	token, err := h.oauthConfig.Exchange(context.Background(), code)
	if err != nil {
		h.redirectWithError(w, r, "Failed to exchange token")
		return
	}

	// Get user info from Google
	googleUser, err := h.getGoogleUserInfo(token.AccessToken)
	if err != nil {
		h.redirectWithError(w, r, "Failed to get user info")
		return
	}

	// Find or create user
	u, err := h.findOrCreateGoogleUser(googleUser, token)
	if err != nil {
		h.redirectWithError(w, r, "Failed to create user")
		return
	}

	// Create session token
	sessionToken, err := h.authService.GenerateToken()
	if err != nil {
		h.redirectWithError(w, r, "Failed to generate token")
		return
	}

	session := &authDomain.Session{
		ID:        uuid.New().String(),
		UserID:    u.ID,
		Token:     sessionToken,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}

	if err := h.authService.CreateSession(session); err != nil {
		h.redirectWithError(w, r, "Failed to create session")
		return
	}

	// Redirect to frontend with token
	redirectURL := fmt.Sprintf("%s/auth/callback?token=%s", h.frontendURL, sessionToken)
	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}

// getGoogleUserInfo fetches user info from Google API
func (h *OAuthHandler) getGoogleUserInfo(accessToken string) (*GoogleUserInfo, error) {
	resp, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + accessToken)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var userInfo GoogleUserInfo
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, err
	}

	return &userInfo, nil
}

// findOrCreateGoogleUser finds an existing user or creates a new one
func (h *OAuthHandler) findOrCreateGoogleUser(googleUser *GoogleUserInfo, token *oauth2.Token) (*user.User, error) {
	// First, try to find by Google ID
	u, err := h.userRepo.GetByGoogleID(googleUser.ID)
	if err == nil {
		// Update Google token if we have a refresh token
		if token.RefreshToken != "" {
			u.GoogleToken = token.RefreshToken
			u.AvatarURL = googleUser.Picture
			h.userRepo.Update(u)
		}
		return u, nil
	}

	// Try to find by email
	u, err = h.userRepo.GetByEmail(googleUser.Email)
	if err == nil {
		// Link existing account to Google
		u.GoogleID = googleUser.ID
		u.AuthProvider = user.AuthProviderGoogle
		if token.RefreshToken != "" {
			u.GoogleToken = token.RefreshToken
		}
		u.AvatarURL = googleUser.Picture
		if err := h.userRepo.Update(u); err != nil {
			return nil, err
		}
		return u, nil
	}

	if !errors.Is(err, user.ErrUserNotFound) {
		return nil, err
	}

	// Create new user
	username := googleUser.GivenName
	if username == "" {
		username = strings.Split(googleUser.Email, "@")[0]
	}

	// Make username unique if needed
	baseUsername := username
	for i := 1; ; i++ {
		_, err := h.userRepo.GetByUsername(username)
		if errors.Is(err, user.ErrUserNotFound) {
			break
		}
		username = fmt.Sprintf("%s%d", baseUsername, i)
	}

	newUser := &user.User{
		ID:           uuid.New().String(),
		Email:        googleUser.Email,
		Username:     username,
		Password:     "", // No password for Google users
		Role:         user.RoleUser,
		AuthProvider: user.AuthProviderGoogle,
		GoogleID:     googleUser.ID,
		GoogleToken:  token.RefreshToken,
		AvatarURL:    googleUser.Picture,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := h.userRepo.Create(newUser); err != nil {
		return nil, err
	}

	return newUser, nil
}

// redirectWithError redirects to frontend with error message
func (h *OAuthHandler) redirectWithError(w http.ResponseWriter, r *http.Request, errMsg string) {
	redirectURL := fmt.Sprintf("%s/auth/callback?error=%s", h.frontendURL, errMsg)
	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}

// GoogleStatus returns whether Google OAuth is configured
func (h *OAuthHandler) GoogleStatus(w http.ResponseWriter, r *http.Request) {
	SendSuccess(w, "", map[string]interface{}{
		"enabled":  h.oauthConfig.ClientID != "",
		"calendar": h.oauthConfig.ClientID != "",
	})
}
