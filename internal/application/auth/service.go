package auth

import (
	"crypto/rand"
	"encoding/hex"
	"regexp"
	"time"

	"golang.org/x/crypto/bcrypt"

	domain "gomanager/internal/domain/auth"
	"gomanager/internal/domain/user"
)

// Service defines the authentication service interface
type Service interface {
	Register(req domain.RegisterRequest) (*user.User, error)
	Login(req domain.LoginRequest) (*domain.LoginResponse, error)
	LoginWithUser(req domain.LoginRequest) (*domain.LoginResponse, *user.User, error)
	ValidateToken(token string) (*user.User, error)
	Logout(token string) error
	HashPassword(password string) (string, error)
	CheckPassword(hashedPassword, password string) bool
	CreateSession(session *domain.Session) error
	GenerateToken() (string, error)
}

type service struct {
	userRepo    user.Repository
	sessionRepo SessionRepository
	tokenExpiry time.Duration
}

// SessionRepository defines the session storage interface
type SessionRepository interface {
	Create(session *domain.Session) error
	GetByToken(token string) (*domain.Session, error)
	Delete(token string) error
	DeleteByUserID(userID string) error
}

// NewService creates a new auth service
func NewService(userRepo user.Repository, sessionRepo SessionRepository, tokenExpiry time.Duration) Service {
	return &service{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		tokenExpiry: tokenExpiry,
	}
}

func (s *service) Register(req domain.RegisterRequest) (*user.User, error) {
	// Validate email
	if !isValidEmail(req.Email) {
		return nil, user.ErrInvalidEmail
	}

	// Validate username
	if len(req.Username) < 3 {
		return nil, user.ErrInvalidUsername
	}

	// Validate password
	if len(req.Password) < 6 {
		return nil, user.ErrInvalidPassword
	}

	// Check if user already exists
	if _, err := s.userRepo.GetByEmail(req.Email); err == nil {
		return nil, user.ErrUserAlreadyExists
	}

	if _, err := s.userRepo.GetByUsername(req.Username); err == nil {
		return nil, user.ErrUserAlreadyExists
	}

	// Hash password
	hashedPassword, err := s.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	// Determine role (first user is admin)
	role := user.RoleUser
	count, _ := s.userRepo.Count()
	if count == 0 {
		role = user.RoleAdmin
	}

	// Create user
	newUser := &user.User{
		Email:    req.Email,
		Username: req.Username,
		Password: hashedPassword,
		Role:     role,
	}

	if err := s.userRepo.Create(newUser); err != nil {
		return nil, err
	}

	return newUser, nil
}

func (s *service) Login(req domain.LoginRequest) (*domain.LoginResponse, error) {
	resp, _, err := s.LoginWithUser(req)
	return resp, err
}

func (s *service) LoginWithUser(req domain.LoginRequest) (*domain.LoginResponse, *user.User, error) {
	// Find user by email
	u, err := s.userRepo.GetByEmail(req.Email)
	if err != nil {
		return nil, nil, user.ErrInvalidCredentials
	}

	// Check password (skip for Google users)
	if u.AuthProvider == user.AuthProviderLocal && !s.CheckPassword(u.Password, req.Password) {
		return nil, nil, user.ErrInvalidCredentials
	}

	// Generate token
	token, err := generateToken()
	if err != nil {
		return nil, nil, err
	}

	// Create session
	expiresAt := time.Now().Add(s.tokenExpiry)
	session := &domain.Session{
		UserID:    u.ID,
		Token:     token,
		ExpiresAt: expiresAt,
	}

	if err := s.sessionRepo.Create(session); err != nil {
		return nil, nil, err
	}

	return &domain.LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt.Unix(),
	}, u, nil
}

func (s *service) ValidateToken(token string) (*user.User, error) {
	session, err := s.sessionRepo.GetByToken(token)
	if err != nil {
		return nil, user.ErrUnauthorized
	}

	if time.Now().After(session.ExpiresAt) {
		s.sessionRepo.Delete(token)
		return nil, user.ErrUnauthorized
	}

	return s.userRepo.GetByID(session.UserID)
}

func (s *service) Logout(token string) error {
	return s.sessionRepo.Delete(token)
}

func (s *service) CreateSession(session *domain.Session) error {
	return s.sessionRepo.Create(session)
}

func (s *service) GenerateToken() (string, error) {
	return generateToken()
}

func (s *service) HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func (s *service) CheckPassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

func generateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func isValidEmail(email string) bool {
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return re.MatchString(email)
}
