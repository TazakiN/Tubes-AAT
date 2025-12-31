package main

import (
	"database/sql"
	"fmt"
	"log"

	"report-service/config"
	"report-service/internal/handler"
	"report-service/internal/repository"
	"report-service/internal/service"

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

	// Initialize repositories
	reportRepo := repository.NewReportRepository(db)
	voteRepo := repository.NewVoteRepository(db)
	notificationRepo := repository.NewNotificationRepository(db)

	// Initialize services
	reportService := service.NewReportService(reportRepo, cfg.Anonymous)
	voteService := service.NewVoteService(voteRepo, reportRepo)
	notificationService := service.NewNotificationService(notificationRepo)

	// Wire notification service to report service for status update notifications
	reportService.SetNotificationService(notificationService)

	// Initialize handlers
	reportHandler := handler.NewReportHandler(reportService)
	voteHandler := handler.NewVoteHandler(voteService)
	notificationHandler := handler.NewNotificationHandler(notificationService)

	// Setup Gin router
	r := gin.Default()

	// Health check
	r.GET("/health", reportHandler.Health)

	// Public endpoints (no auth required - these will bypass auth in nginx)
	r.GET("/public", reportHandler.GetPublicReports)
	r.GET("/categories", reportHandler.GetCategories)
	r.POST("/categories", reportHandler.CreateCategory)

	// Report routes (auth required)
	r.POST("/", reportHandler.CreateReport)
	r.GET("/", reportHandler.GetReports)
	r.GET("/my", reportHandler.GetMyReports)
	r.GET("/:id", reportHandler.GetReportByID)
	r.PUT("/:id", reportHandler.UpdateReport)
	r.PATCH("/:id/status", reportHandler.UpdateStatus)

	// Vote routes (auth required)
	r.POST("/:id/vote", voteHandler.CastVote)
	r.DELETE("/:id/vote", voteHandler.RemoveVote)
	r.GET("/:id/vote", voteHandler.GetVote)

	// Notification routes (auth required) - these will be accessed via /api/v1/notifications
	notifications := r.Group("/notifications")
	{
		notifications.GET("", notificationHandler.GetNotifications)
		notifications.GET("/stream", notificationHandler.StreamNotifications)
		notifications.PATCH("/:id/read", notificationHandler.MarkAsRead)
		notifications.PATCH("/read-all", notificationHandler.MarkAllAsRead)
	}

	// Start server
	addr := fmt.Sprintf(":%s", cfg.Server.Port)
	log.Printf("Report service starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

