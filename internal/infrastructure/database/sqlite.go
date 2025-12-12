package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

// DB holds the database connection
type DB struct {
	*sql.DB
}

// New creates a new database connection
func New(dbPath string) (*DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{db}, nil
}

// Migrate runs database migrations
func (db *DB) Migrate() error {
	// Core table creation
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			email TEXT UNIQUE NOT NULL,
			username TEXT UNIQUE NOT NULL,
			password TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'user',
			auth_provider TEXT DEFAULT 'local',
			google_id TEXT,
			google_token TEXT,
			avatar_url TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			token TEXT UNIQUE NOT NULL,
			expires_at DATETIME NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS shares (
			id TEXT PRIMARY KEY,
			token TEXT UNIQUE NOT NULL,
			path TEXT NOT NULL,
			created_by TEXT NOT NULL,
			share_type TEXT NOT NULL DEFAULT 'public',
			password TEXT,
			permission TEXT NOT NULL DEFAULT 'view',
			expires_at DATETIME,
			max_downloads INTEGER,
			downloads INTEGER DEFAULT 0,
			is_active BOOLEAN DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE CASCADE
		)`,
	}

	// Add columns if they don't exist (for existing databases)
	// These must run BEFORE index creation on these columns
	alterMigrations := []string{
		`ALTER TABLE users ADD COLUMN auth_provider TEXT DEFAULT 'local'`,
		`ALTER TABLE users ADD COLUMN google_id TEXT`,
		`ALTER TABLE users ADD COLUMN google_token TEXT`,
		`ALTER TABLE users ADD COLUMN avatar_url TEXT`,
	}

	// Index creation (must run after ALTER TABLE for google_id)
	indexMigrations := []string{
		`CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_shares_token ON shares(token)`,
		`CREATE INDEX IF NOT EXISTS idx_shares_created_by ON shares(created_by)`,
		`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)`,
		`CREATE INDEX IF NOT EXISTS idx_users_google_id ON users(google_id)`,
	}

	// 1. Create tables
	for _, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	// 2. Add columns (ignore errors if they already exist)
	for _, migration := range alterMigrations {
		db.Exec(migration) // Ignore errors - column may already exist
	}

	// 3. Create indexes (now that all columns exist)
	for _, migration := range indexMigrations {
		if _, err := db.Exec(migration); err != nil {
			return fmt.Errorf("index creation failed: %w", err)
		}
	}

	return nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.DB.Close()
}
