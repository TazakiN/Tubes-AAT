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
	log.Println("notification-service starting...")

	cfg, err := config.LoadConfig("config/config.json")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

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
		log.Fatalf("db ping: %v", err)
	}
	log.Println("db connected")

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
	log.Println("rabbitmq connected")

	sseHub := messaging.NewSSEHub()
	go sseHub.Run()

	notificationRepo := repository.NewNotificationRepository(db)

	consumer := messaging.NewNotificationConsumer(rmq, notificationRepo, sseHub)
	consumer.Start()

	notificationService := service.NewNotificationService(notificationRepo, sseHub)

	notificationHandler := handler.NewNotificationHandler(notificationService, cfg.JWT.Secret)

	r := gin.Default()

	r.GET("/health", notificationHandler.Health)
	r.GET("/health/detailed", notificationHandler.HealthCheck)

	notifications := r.Group("/notifications")
	{
		notifications.GET("", notificationHandler.GetNotifications)
		notifications.GET("/stream", notificationHandler.StreamNotifications)
		notifications.PATCH("/:id/read", notificationHandler.MarkAsRead)
		notifications.PATCH("/read-all", notificationHandler.MarkAllAsRead)
	}

	admin := r.Group("/admin")
	{
		admin.GET("/dlq/stats", notificationHandler.GetDLQStats)
	}

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
