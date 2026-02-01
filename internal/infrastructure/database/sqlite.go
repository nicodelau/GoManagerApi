package database

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

// DB holds the database connection
type DB struct {
	*sql.DB
	dbType string
}

// DatabaseType represents the type of database
type DatabaseType string

const (
	SQLite     DatabaseType = "sqlite"
	PostgreSQL DatabaseType = "postgres"
)

// NewDatabase creates a new database connection based on the connection string
func NewDatabase(connectionString string) (*DB, error) {
	if strings.HasPrefix(connectionString, "postgresql://") || strings.HasPrefix(connectionString, "postgres://") {
		return NewPostgreSQL(connectionString)
	}
	// Default to SQLite
	return New(connectionString)
}

// NewPostgreSQL creates a new PostgreSQL database connection
func NewPostgreSQL(connectionString string) (*DB, error) {
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to open PostgreSQL database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping PostgreSQL database: %w", err)
	}

	// Set connection pool settings for production
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	return &DB{db, "postgres"}, nil
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

	return &DB{db, "sqlite"}, nil
}

// Migrate runs database migrations based on database type
func (db *DB) Migrate() error {
	if db.dbType == "postgres" {
		return db.MigratePostgreSQL()
	}
	return db.MigrateSQLite()
}

// MigrateSQLite runs SQLite database migrations
func (db *DB) MigrateSQLite() error {
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
		// New table for Google Drive integration
		`CREATE TABLE IF NOT EXISTS google_drive_folders (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			folder_id TEXT NOT NULL,
			folder_name TEXT NOT NULL,
			folder_path TEXT NOT NULL,
			is_active BOOLEAN DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		// New table for Google Ads campaigns management
		`CREATE TABLE IF NOT EXISTS google_ads_campaigns (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			customer_id TEXT NOT NULL,
			campaign_id TEXT NOT NULL,
			campaign_name TEXT NOT NULL,
			campaign_status TEXT NOT NULL,
			budget_amount REAL,
			target_cpa REAL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
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
		`CREATE INDEX IF NOT EXISTS idx_google_drive_folders_user_id ON google_drive_folders(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_google_drive_folders_folder_id ON google_drive_folders(folder_id)`,
		`CREATE INDEX IF NOT EXISTS idx_google_ads_campaigns_user_id ON google_ads_campaigns(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_google_ads_campaigns_customer_id ON google_ads_campaigns(customer_id)`,
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

// MigratePostgreSQL runs PostgreSQL database migrations
func (db *DB) MigratePostgreSQL() error {
	// Core table creation for PostgreSQL
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
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			token TEXT UNIQUE NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
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
			expires_at TIMESTAMP,
			max_downloads INTEGER,
			downloads INTEGER DEFAULT 0,
			is_active BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE CASCADE
		)`,
		// New table for Google Drive integration
		`CREATE TABLE IF NOT EXISTS google_drive_folders (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			folder_id TEXT NOT NULL,
			folder_name TEXT NOT NULL,
			folder_path TEXT NOT NULL,
			is_active BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		// New table for Google Ads campaigns management
		`CREATE TABLE IF NOT EXISTS google_ads_campaigns (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			customer_id TEXT NOT NULL,
			campaign_id TEXT NOT NULL,
			campaign_name TEXT NOT NULL,
			campaign_status TEXT NOT NULL,
			budget_amount DECIMAL(10,2),
			target_cpa DECIMAL(10,2),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
	}

	// Index creation
	indexMigrations := []string{
		`CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_shares_token ON shares(token)`,
		`CREATE INDEX IF NOT EXISTS idx_shares_created_by ON shares(created_by)`,
		`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)`,
		`CREATE INDEX IF NOT EXISTS idx_users_google_id ON users(google_id)`,
		`CREATE INDEX IF NOT EXISTS idx_google_drive_folders_user_id ON google_drive_folders(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_google_drive_folders_folder_id ON google_drive_folders(folder_id)`,
		`CREATE INDEX IF NOT EXISTS idx_google_ads_campaigns_user_id ON google_ads_campaigns(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_google_ads_campaigns_customer_id ON google_ads_campaigns(customer_id)`,
	}

	// 1. Create tables
	for _, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			return fmt.Errorf("PostgreSQL migration failed: %w", err)
		}
	}

	// 2. Create indexes
	for _, migration := range indexMigrations {
		if _, err := db.Exec(migration); err != nil {
			return fmt.Errorf("PostgreSQL index creation failed: %w", err)
		}
	}

	return nil
}

// ParsePostgreSQLConnection parses a PostgreSQL connection string and returns connection parameters
func ParsePostgreSQLConnection(connectionString string) (map[string]string, error) {
	// Parse the connection string
	parsedURL, err := url.Parse(connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	params := make(map[string]string)

	// Extract components
	if parsedURL.User != nil {
		params["user"] = parsedURL.User.Username()
		if password, ok := parsedURL.User.Password(); ok {
			params["password"] = password
		}
	}

	params["host"] = parsedURL.Hostname()
	params["port"] = parsedURL.Port()
	if params["port"] == "" {
		params["port"] = "5432"
	}

	params["dbname"] = strings.TrimPrefix(parsedURL.Path, "/")

	// Parse query parameters
	for key, values := range parsedURL.Query() {
		if len(values) > 0 {
			params[key] = values[0]
		}
	}

	return params, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.DB.Close()
}

// GetType returns the database type
func (db *DB) GetType() string {
	return db.dbType
}
