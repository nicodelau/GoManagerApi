package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	authService "gomanager/internal/application/auth"
	fileService "gomanager/internal/application/file"
	"gomanager/internal/delivery/http/handler"
	"gomanager/internal/delivery/http/router"
	"gomanager/internal/infrastructure/config"
	"gomanager/internal/infrastructure/database"
	"gomanager/internal/infrastructure/repository"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize database
	db, err := database.New(cfg.DatabasePath)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Run migrations
	if err := db.Migrate(); err != nil {
		log.Fatal("Failed to run migrations:", err)
	}

	// Initialize repositories
	fileRepo := repository.NewFilesystemRepository(cfg.StoragePath)
	userRepo := repository.NewUserRepository(db)
	sessionRepo := repository.NewSessionRepository(db)
	shareRepo := repository.NewShareRepository(db)

	// Initialize services
	fileSvc := fileService.NewService(fileRepo)
	authSvc := authService.NewService(userRepo, sessionRepo, time.Duration(cfg.TokenExpiry)*time.Hour)

	// Initialize handlers
	fileHandler := handler.NewFileHandler(fileSvc, cfg.MaxFileSize)
	authHandler := handler.NewAuthHandler(authSvc)
	shareHandler := handler.NewShareHandler(shareRepo, fileSvc, cfg.BaseURL)
	oauthHandler := handler.NewOAuthHandler(cfg, authSvc, userRepo)
	userHandler := handler.NewUserHandler(authSvc, userRepo, cfg.StoragePath)
	googleServicesHandler := handler.NewGoogleServicesHandler(cfg, userRepo)

	// Setup routes
	handlers := router.Handlers{
		File:           fileHandler,
		Auth:           authHandler,
		Share:          shareHandler,
		OAuth:          oauthHandler,
		User:           userHandler,
		GoogleServices: googleServicesHandler,
	}
	mux := router.Setup(handlers, authSvc)

	// Start server
	addr := fmt.Sprintf(":%s", cfg.Port)
	fmt.Println("=================================")
	fmt.Println("       GoManager Server")
	fmt.Println("=================================")
	fmt.Printf("Server:    http://localhost%s\n", addr)
	fmt.Printf("Storage:   %s\n", cfg.StoragePath)
	fmt.Printf("Database:  %s\n", cfg.DatabasePath)
	if cfg.GoogleClientID != "" {
		fmt.Println("Google:    Enabled")
	}
	fmt.Println("=================================")
	log.Fatal(http.ListenAndServe(addr, mux))
}
