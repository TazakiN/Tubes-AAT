package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"notification-service/config"
	"notification-service/internal/handler"
	"notification-service/internal/messaging"
	"notification-service/internal/repository"
	"notification-service/internal/service"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

func main() {
	log.Println("==============================================")
	log.Println("  NOTIFICATION SERVICE - Starting Up")
	log.Println("  Features: DLQ, Retry, Idempotency, SSE")
	log.Println("==============================================")

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
	log.Println("✓ Connected to database")

	// Connect to RabbitMQ with DLQ configuration
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
	log.Println("✓ Connected to RabbitMQ (with DLQ configuration)")

	// Initialize SSE Hub
	sseHub := messaging.NewSSEHub()
	go sseHub.Run()
	log.Println("✓ SSE Hub started")

	// Initialize repository
	notificationRepo := repository.NewNotificationRepository(db)

	// Start notification consumer with retry logic
	consumer := messaging.NewNotificationConsumer(rmq, notificationRepo, sseHub)
	consumer.Start()
	log.Println("✓ Message consumers started (3 queues)")
	log.Println("  - queue.status_updates")
	log.Println("  - queue.report_created")
	log.Println("  - queue.vote_received")

	// Initialize service
	notificationService := service.NewNotificationService(notificationRepo, sseHub)

	// Initialize handler
	notificationHandler := handler.NewNotificationHandler(notificationService)

	// Setup Gin router
	r := gin.Default()

	// Health check
	r.GET("/health", notificationHandler.Health)
	r.GET("/health/detailed", notificationHandler.HealthCheck)

	// Notification routes
	notifications := r.Group("/notifications")
	{
		notifications.GET("", notificationHandler.GetNotifications)
		notifications.GET("/stream", notificationHandler.StreamNotifications)
		notifications.PATCH("/:id/read", notificationHandler.MarkAsRead)
		notifications.PATCH("/read-all", notificationHandler.MarkAllAsRead)
	}

	// Admin/monitoring routes
	admin := r.Group("/admin")
	{
		admin.GET("/dlq/stats", notificationHandler.GetDLQStats)
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("\nShutdown signal received...")
		consumer.Stop()
		log.Println("Notification service stopped gracefully")
		os.Exit(0)
	}()

	addr := fmt.Sprintf(":%s", cfg.Server.Port)
	log.Println("==============================================")
	log.Printf("  Notification service listening on %s", addr)
	log.Println("==============================================")

	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
