package repository

import (
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"

	"gomanager/internal/application/auth"
	domain "gomanager/internal/domain/auth"
	"gomanager/internal/infrastructure/database"
)

var (
	ErrSessionNotFound = errors.New("session not found")
)

type sessionRepository struct {
	db *database.DB
}

// NewSessionRepository creates a new session repository
func NewSessionRepository(db *database.DB) auth.SessionRepository {
	return &sessionRepository{db: db}
}

func (r *sessionRepository) Create(session *domain.Session) error {
	if session.ID == "" {
		session.ID = uuid.New().String()
	}
	session.CreatedAt = time.Now()

	_, err := r.db.Exec(
		`INSERT INTO sessions (id, user_id, token, expires_at, created_at) 
		 VALUES (?, ?, ?, ?, ?)`,
		session.ID, session.UserID, session.Token, session.ExpiresAt, session.CreatedAt,
	)
	return err
}

func (r *sessionRepository) GetByToken(token string) (*domain.Session, error) {
	session := &domain.Session{}
	err := r.db.QueryRow(
		`SELECT id, user_id, token, expires_at, created_at 
		 FROM sessions WHERE token = ?`, token,
	).Scan(&session.ID, &session.UserID, &session.Token, &session.ExpiresAt, &session.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, err
	}
	return session, nil
}

func (r *sessionRepository) Delete(token string) error {
	result, err := r.db.Exec(`DELETE FROM sessions WHERE token = ?`, token)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrSessionNotFound
	}
	return nil
}

func (r *sessionRepository) DeleteByUserID(userID string) error {
	_, err := r.db.Exec(`DELETE FROM sessions WHERE user_id = ?`, userID)
	return err
}
