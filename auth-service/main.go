package main

import (
	"database/sql"
	"fmt"
	"log"

	"auth-service/config"
	"auth-service/internal/handler"
	"auth-service/internal/repository"
	"auth-service/internal/service"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

func main() {
	// Load config
	cfg, err := config.LoadConfig("config/config.json")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to database
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.DBName,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Connected to database")

	// Initialize layers
	userRepo := repository.NewUserRepository(db)
	authService := service.NewAuthService(userRepo, cfg.JWT)
	authHandler := handler.NewAuthHandler(authService)

	// Setup Gin router
	r := gin.Default()

	// Health check
	r.GET("/health", authHandler.Health)

	// Auth routes
	r.POST("/register", authHandler.Register)
	r.POST("/login", authHandler.Login)
	r.GET("/validate", authHandler.Validate)
	r.GET("/me", authHandler.Me)

	// Start server
	addr := fmt.Sprintf(":%s", cfg.Server.Port)
	log.Printf("Auth service starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
