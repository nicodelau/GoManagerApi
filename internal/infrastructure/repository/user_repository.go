package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"gomanager/internal/domain/user"
	"gomanager/internal/infrastructure/database"
)

type userRepository struct {
	db *database.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *database.DB) user.Repository {
	return &userRepository{db: db}
}

// getPlaceholderQuery converts a query template with %s placeholders to the correct database syntax
func (r *userRepository) getPlaceholderQuery(queryTemplate string, paramCount int) string {
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

func (r *userRepository) Create(u *user.User) error {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	if u.AuthProvider == "" {
		u.AuthProvider = user.AuthProviderLocal
	}
	u.CreatedAt = time.Now()
	u.UpdatedAt = time.Now()

	query := r.getPlaceholderQuery(
		`INSERT INTO users (id, email, username, password, role, auth_provider, google_id, google_token, avatar_url, created_at, updated_at) 
		 VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)`,
		11)

	_, err := r.db.Exec(query,
		u.ID, u.Email, u.Username, u.Password, u.Role, u.AuthProvider, u.GoogleID, u.GoogleToken, u.AvatarURL, u.CreatedAt, u.UpdatedAt,
	)
	if err != nil {
		return user.ErrUserAlreadyExists
	}
	return nil
}

func (r *userRepository) GetByID(id string) (*user.User, error) {
	u := &user.User{}
	var googleID, googleToken, avatarURL sql.NullString

	query := r.getPlaceholderQuery(
		`SELECT id, email, username, password, role, auth_provider, google_id, google_token, avatar_url, created_at, updated_at 
		 FROM users WHERE id = %s`, 1)

	err := r.db.QueryRow(query, id).Scan(
		&u.ID, &u.Email, &u.Username, &u.Password, &u.Role, &u.AuthProvider,
		&googleID, &googleToken, &avatarURL, &u.CreatedAt, &u.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, user.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	u.GoogleID = googleID.String
	u.GoogleToken = googleToken.String
	u.AvatarURL = avatarURL.String
	return u, nil
}

func (r *userRepository) GetByEmail(email string) (*user.User, error) {
	u := &user.User{}
	var googleID, googleToken, avatarURL sql.NullString

	query := r.getPlaceholderQuery(
		`SELECT id, email, username, password, role, auth_provider, google_id, google_token, avatar_url, created_at, updated_at 
		 FROM users WHERE email = %s`, 1)

	err := r.db.QueryRow(query, email).Scan(
		&u.ID, &u.Email, &u.Username, &u.Password, &u.Role, &u.AuthProvider,
		&googleID, &googleToken, &avatarURL, &u.CreatedAt, &u.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, user.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	u.GoogleID = googleID.String
	u.GoogleToken = googleToken.String
	u.AvatarURL = avatarURL.String
	return u, nil
}

func (r *userRepository) GetByUsername(username string) (*user.User, error) {
	u := &user.User{}
	var googleID, googleToken, avatarURL sql.NullString

	query := r.getPlaceholderQuery(
		`SELECT id, email, username, password, role, auth_provider, google_id, google_token, avatar_url, created_at, updated_at 
		 FROM users WHERE username = %s`, 1)

	err := r.db.QueryRow(query, username).Scan(
		&u.ID, &u.Email, &u.Username, &u.Password, &u.Role, &u.AuthProvider,
		&googleID, &googleToken, &avatarURL, &u.CreatedAt, &u.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, user.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	u.GoogleID = googleID.String
	u.GoogleToken = googleToken.String
	u.AvatarURL = avatarURL.String
	return u, nil
}

func (r *userRepository) GetByGoogleID(googleID string) (*user.User, error) {
	u := &user.User{}
	var gID, googleToken, avatarURL sql.NullString

	query := r.getPlaceholderQuery(
		`SELECT id, email, username, password, role, auth_provider, google_id, google_token, avatar_url, created_at, updated_at 
		 FROM users WHERE google_id = %s`, 1)

	err := r.db.QueryRow(query, googleID).Scan(
		&u.ID, &u.Email, &u.Username, &u.Password, &u.Role, &u.AuthProvider,
		&gID, &googleToken, &avatarURL, &u.CreatedAt, &u.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, user.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	u.GoogleID = gID.String
	u.GoogleToken = googleToken.String
	u.AvatarURL = avatarURL.String
	return u, nil
}

func (r *userRepository) Update(u *user.User) error {
	u.UpdatedAt = time.Now()

	query := r.getPlaceholderQuery(
		`UPDATE users SET email = %s, username = %s, password = %s, role = %s, auth_provider = %s, google_id = %s, google_token = %s, avatar_url = %s, updated_at = %s 
		 WHERE id = %s`, 10)

	result, err := r.db.Exec(query,
		u.Email, u.Username, u.Password, u.Role, u.AuthProvider, u.GoogleID, u.GoogleToken, u.AvatarURL, u.UpdatedAt, u.ID,
	)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return user.ErrUserNotFound
	}
	return nil
}

func (r *userRepository) Delete(id string) error {
	query := r.getPlaceholderQuery(`DELETE FROM users WHERE id = %s`, 1)
	result, err := r.db.Exec(query, id)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return user.ErrUserNotFound
	}
	return nil
}

func (r *userRepository) List() ([]user.User, error) {
	rows, err := r.db.Query(
		`SELECT id, email, username, password, role, auth_provider, google_id, google_token, avatar_url, created_at, updated_at 
		 FROM users ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []user.User
	for rows.Next() {
		var u user.User
		var googleID, googleToken, avatarURL sql.NullString
		if err := rows.Scan(&u.ID, &u.Email, &u.Username, &u.Password, &u.Role, &u.AuthProvider, &googleID, &googleToken, &avatarURL, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		u.GoogleID = googleID.String
		u.GoogleToken = googleToken.String
		u.AvatarURL = avatarURL.String
		users = append(users, u)
	}
	return users, nil
}

func (r *userRepository) Count() (int, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count)
	return count, err
}
