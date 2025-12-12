package repository

import (
	"database/sql"
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

func (r *userRepository) Create(u *user.User) error {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	if u.AuthProvider == "" {
		u.AuthProvider = user.AuthProviderLocal
	}
	u.CreatedAt = time.Now()
	u.UpdatedAt = time.Now()

	_, err := r.db.Exec(
		`INSERT INTO users (id, email, username, password, role, auth_provider, google_id, google_token, avatar_url, created_at, updated_at) 
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
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
	err := r.db.QueryRow(
		`SELECT id, email, username, password, role, auth_provider, google_id, google_token, avatar_url, created_at, updated_at 
		 FROM users WHERE id = ?`, id,
	).Scan(&u.ID, &u.Email, &u.Username, &u.Password, &u.Role, &u.AuthProvider, &googleID, &googleToken, &avatarURL, &u.CreatedAt, &u.UpdatedAt)

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
	err := r.db.QueryRow(
		`SELECT id, email, username, password, role, auth_provider, google_id, google_token, avatar_url, created_at, updated_at 
		 FROM users WHERE email = ?`, email,
	).Scan(&u.ID, &u.Email, &u.Username, &u.Password, &u.Role, &u.AuthProvider, &googleID, &googleToken, &avatarURL, &u.CreatedAt, &u.UpdatedAt)

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
	err := r.db.QueryRow(
		`SELECT id, email, username, password, role, auth_provider, google_id, google_token, avatar_url, created_at, updated_at 
		 FROM users WHERE username = ?`, username,
	).Scan(&u.ID, &u.Email, &u.Username, &u.Password, &u.Role, &u.AuthProvider, &googleID, &googleToken, &avatarURL, &u.CreatedAt, &u.UpdatedAt)

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
	err := r.db.QueryRow(
		`SELECT id, email, username, password, role, auth_provider, google_id, google_token, avatar_url, created_at, updated_at 
		 FROM users WHERE google_id = ?`, googleID,
	).Scan(&u.ID, &u.Email, &u.Username, &u.Password, &u.Role, &u.AuthProvider, &gID, &googleToken, &avatarURL, &u.CreatedAt, &u.UpdatedAt)

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
	result, err := r.db.Exec(
		`UPDATE users SET email = ?, username = ?, password = ?, role = ?, auth_provider = ?, google_id = ?, google_token = ?, avatar_url = ?, updated_at = ? 
		 WHERE id = ?`,
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
	result, err := r.db.Exec(`DELETE FROM users WHERE id = ?`, id)
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
