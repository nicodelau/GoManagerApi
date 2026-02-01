package repository

import (
	"database/sql"
	"errors"
	"fmt"
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

// getPlaceholderQuery converts a query template with %s placeholders to the correct database syntax
func (r *sessionRepository) getPlaceholderQuery(queryTemplate string, paramCount int) string {
	// Check if we're using PostgreSQL
	if r.db.GetType() == "postgres" {
		// Use PostgreSQL numbered placeholders
		placeholders := make([]interface{}, paramCount)
		for i := 0; i < paramCount; i++ {
			placeholders[i] = fmt.Sprintf("$%d", i+1)
		}
		return fmt.Sprintf(queryTemplate, placeholders...)
	}
	// Use SQLite ? placeholders
	placeholders := make([]interface{}, paramCount)
	for i := 0; i < paramCount; i++ {
		placeholders[i] = "?"
	}
	return fmt.Sprintf(queryTemplate, placeholders...)
}

func (r *sessionRepository) Create(session *domain.Session) error {
	if session.ID == "" {
		session.ID = uuid.New().String()
	}
	session.CreatedAt = time.Now()

	query := r.getPlaceholderQuery(
		`INSERT INTO sessions (id, user_id, token, expires_at, created_at) 
		 VALUES (%s, %s, %s, %s, %s)`, 5)

	_, err := r.db.Exec(query,
		session.ID, session.UserID, session.Token, session.ExpiresAt, session.CreatedAt,
	)
	return err
}

func (r *sessionRepository) GetByToken(token string) (*domain.Session, error) {
	session := &domain.Session{}

	query := r.getPlaceholderQuery(
		`SELECT id, user_id, token, expires_at, created_at 
		 FROM sessions WHERE token = %s`, 1)

	err := r.db.QueryRow(query, token).Scan(
		&session.ID, &session.UserID, &session.Token, &session.ExpiresAt, &session.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, err
	}
	return session, nil
}

func (r *sessionRepository) Delete(token string) error {
	query := r.getPlaceholderQuery(`DELETE FROM sessions WHERE token = %s`, 1)
	result, err := r.db.Exec(query, token)
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
	query := r.getPlaceholderQuery(`DELETE FROM sessions WHERE user_id = %s`, 1)
	_, err := r.db.Exec(query, userID)
	return err
}
