package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

func init() {
	// Load .env file if it exists (ignores error if not found)
	godotenv.Load()
}

type Config struct {
	Port         string
	StoragePath  string
	MaxFileSize  int64
	DatabasePath string
	BaseURL      string
	TokenExpiry  int // hours
	FrontendURL  string

	// Google OAuth
	GoogleClientID     string
	GoogleClientSecret string

	// Google Drive
	GoogleDriveFolder string

	// Google Ads API
	GoogleAdsCustomerID     string
	GoogleAdsDeveloperToken string
}

func Load() *Config {
	return &Config{
		Port:                    getEnv("PORT", "8005"),
		StoragePath:             getEnv("STORAGE_PATH", "./storage"),
		MaxFileSize:             getEnvAsInt64("MAX_FILE_SIZE", 100<<20),                                // 100MB default
		DatabasePath:            getEnv("DATABASE_URL", getEnv("DATABASE_PATH", "./data/gomanager.db")), // Support both DATABASE_URL (PostgreSQL) and DATABASE_PATH (SQLite)
		BaseURL:                 getEnv("BASE_URL", "http://localhost:8005"),
		TokenExpiry:             int(getEnvAsInt64("TOKEN_EXPIRY_HOURS", 24)),
		FrontendURL:             getEnv("FRONTEND_URL", "http://localhost:5173"),
		GoogleClientID:          getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret:      getEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleDriveFolder:       getEnv("GOOGLE_DRIVE_FOLDER", "GoManager"),
		GoogleAdsCustomerID:     getEnv("GOOGLE_ADS_CUSTOMER_ID", ""),
		GoogleAdsDeveloperToken: getEnv("GOOGLE_ADS_DEVELOPER_TOKEN", ""),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intVal
		}
	}
	return defaultValue
}
