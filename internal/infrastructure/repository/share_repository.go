package repository

import (
	"database/sql"
	"time"

	"github.com/google/uuid"

	"gomanager/internal/domain/share"
	"gomanager/internal/infrastructure/database"
)

type shareRepository struct {
	db *database.DB
}

// NewShareRepository creates a new share repository
func NewShareRepository(db *database.DB) share.Repository {
	return &shareRepository{db: db}
}

func (r *shareRepository) Create(s *share.Share) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	s.CreatedAt = time.Now()

	_, err := r.db.Exec(
		`INSERT INTO shares (id, token, path, created_by, share_type, password, permission, expires_at, max_downloads, downloads, is_active, created_at) 
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.Token, s.Path, s.CreatedBy, s.ShareType, s.Password, s.Permission, s.ExpiresAt, s.MaxDownloads, s.Downloads, s.IsActive, s.CreatedAt,
	)
	return err
}

func (r *shareRepository) GetByID(id string) (*share.Share, error) {
	s := &share.Share{}
	var expiresAt sql.NullTime
	var maxDownloads sql.NullInt64

	err := r.db.QueryRow(
		`SELECT id, token, path, created_by, share_type, password, permission, expires_at, max_downloads, downloads, is_active, created_at 
		 FROM shares WHERE id = ?`, id,
	).Scan(&s.ID, &s.Token, &s.Path, &s.CreatedBy, &s.ShareType, &s.Password, &s.Permission, &expiresAt, &maxDownloads, &s.Downloads, &s.IsActive, &s.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, share.ErrShareNotFound
	}
	if err != nil {
		return nil, err
	}

	if expiresAt.Valid {
		s.ExpiresAt = &expiresAt.Time
	}
	if maxDownloads.Valid {
		md := int(maxDownloads.Int64)
		s.MaxDownloads = &md
	}

	return s, nil
}

func (r *shareRepository) GetByToken(token string) (*share.Share, error) {
	s := &share.Share{}
	var expiresAt sql.NullTime
	var maxDownloads sql.NullInt64

	err := r.db.QueryRow(
		`SELECT id, token, path, created_by, share_type, password, permission, expires_at, max_downloads, downloads, is_active, created_at 
		 FROM shares WHERE token = ?`, token,
	).Scan(&s.ID, &s.Token, &s.Path, &s.CreatedBy, &s.ShareType, &s.Password, &s.Permission, &expiresAt, &maxDownloads, &s.Downloads, &s.IsActive, &s.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, share.ErrShareNotFound
	}
	if err != nil {
		return nil, err
	}

	if expiresAt.Valid {
		s.ExpiresAt = &expiresAt.Time
	}
	if maxDownloads.Valid {
		md := int(maxDownloads.Int64)
		s.MaxDownloads = &md
	}

	return s, nil
}

func (r *shareRepository) GetByUser(userID string) ([]share.Share, error) {
	rows, err := r.db.Query(
		`SELECT id, token, path, created_by, share_type, password, permission, expires_at, max_downloads, downloads, is_active, created_at 
		 FROM shares WHERE created_by = ? ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shares []share.Share
	for rows.Next() {
		var s share.Share
		var expiresAt sql.NullTime
		var maxDownloads sql.NullInt64

		if err := rows.Scan(&s.ID, &s.Token, &s.Path, &s.CreatedBy, &s.ShareType, &s.Password, &s.Permission, &expiresAt, &maxDownloads, &s.Downloads, &s.IsActive, &s.CreatedAt); err != nil {
			return nil, err
		}

		if expiresAt.Valid {
			s.ExpiresAt = &expiresAt.Time
		}
		if maxDownloads.Valid {
			md := int(maxDownloads.Int64)
			s.MaxDownloads = &md
		}

		shares = append(shares, s)
	}

	return shares, nil
}

func (r *shareRepository) GetByPath(path string) ([]share.Share, error) {
	rows, err := r.db.Query(
		`SELECT id, token, path, created_by, share_type, password, permission, expires_at, max_downloads, downloads, is_active, created_at 
		 FROM shares WHERE path = ? ORDER BY created_at DESC`, path,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shares []share.Share
	for rows.Next() {
		var s share.Share
		var expiresAt sql.NullTime
		var maxDownloads sql.NullInt64

		if err := rows.Scan(&s.ID, &s.Token, &s.Path, &s.CreatedBy, &s.ShareType, &s.Password, &s.Permission, &expiresAt, &maxDownloads, &s.Downloads, &s.IsActive, &s.CreatedAt); err != nil {
			return nil, err
		}

		if expiresAt.Valid {
			s.ExpiresAt = &expiresAt.Time
		}
		if maxDownloads.Valid {
			md := int(maxDownloads.Int64)
			s.MaxDownloads = &md
		}

		shares = append(shares, s)
	}

	return shares, nil
}

func (r *shareRepository) Update(s *share.Share) error {
	result, err := r.db.Exec(
		`UPDATE shares SET token = ?, path = ?, share_type = ?, password = ?, permission = ?, expires_at = ?, max_downloads = ?, downloads = ?, is_active = ? 
		 WHERE id = ?`,
		s.Token, s.Path, s.ShareType, s.Password, s.Permission, s.ExpiresAt, s.MaxDownloads, s.Downloads, s.IsActive, s.ID,
	)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return share.ErrShareNotFound
	}
	return nil
}

func (r *shareRepository) Delete(id string) error {
	result, err := r.db.Exec(`DELETE FROM shares WHERE id = ?`, id)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return share.ErrShareNotFound
	}
	return nil
}

func (r *shareRepository) IncrementDownloads(id string) error {
	result, err := r.db.Exec(`UPDATE shares SET downloads = downloads + 1 WHERE id = ?`, id)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return share.ErrShareNotFound
	}
	return nil
}
