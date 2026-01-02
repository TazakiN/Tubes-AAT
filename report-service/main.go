package main

import (
	"database/sql"
	"fmt"
	"log"

	"report-service/config"
	"report-service/internal/handler"
	"report-service/internal/messaging"
	"report-service/internal/repository"
	"report-service/internal/service"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

func main() {
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

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Connected to database")

	// Connect to RabbitMQ
	rmq, err := messaging.NewRabbitMQ(
		cfg.RabbitMQ.Host,
		cfg.RabbitMQ.Port,
		cfg.RabbitMQ.User,
		cfg.RabbitMQ.Password,
	)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer rmq.Close()
	log.Println("Connected to RabbitMQ")

	// Initialize SSE Hub
	sseHub := messaging.NewSSEHub()
	go sseHub.Run()

	// Initialize repositories
	reportRepo := repository.NewReportRepository(db)
	voteRepo := repository.NewVoteRepository(db)
	notificationRepo := repository.NewNotificationRepository(db)

	// Start notification consumer
	consumer := messaging.NewNotificationConsumer(rmq, notificationRepo, sseHub)
	consumer.Start()
	defer consumer.Stop()
	log.Println("Notification consumer started")

	// Initialize services
	reportService := service.NewReportService(reportRepo, cfg.Anonymous, rmq)
	voteService := service.NewVoteService(voteRepo, reportRepo, rmq)
	notificationService := service.NewNotificationService(notificationRepo, sseHub)

	// Initialize handlers
	reportHandler := handler.NewReportHandler(reportService)
	voteHandler := handler.NewVoteHandler(voteService)
	notificationHandler := handler.NewNotificationHandler(notificationService, cfg.JWT.Secret)

	// Setup Gin
	r := gin.Default()

	// Health check
	r.GET("/health", reportHandler.Health)

	// Public endpoints
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

	// Notification routes (auth required)
	notifications := r.Group("/notifications")
	{
		notifications.GET("", notificationHandler.GetNotifications)
		notifications.GET("/stream", notificationHandler.StreamNotifications)
		notifications.PATCH("/:id/read", notificationHandler.MarkAsRead)
		notifications.PATCH("/read-all", notificationHandler.MarkAllAsRead)
	}

	addr := fmt.Sprintf(":%s", cfg.Server.Port)
	log.Printf("Report service starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
