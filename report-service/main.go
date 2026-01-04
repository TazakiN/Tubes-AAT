package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"report-service/config"
	"report-service/internal/handler"
	"report-service/internal/messaging"
	"report-service/internal/repository"
	"report-service/internal/service"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

func main() {
	log.Println("report-service starting...")

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
		log.Fatalf("db ping: %v", err)
	}
	log.Println("db connected")

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
	log.Println("rabbitmq connected")

	// Initialize repositories
	reportRepo := repository.NewReportRepository(db)
	voteRepo := repository.NewVoteRepository(db)
	outboxRepo := repository.NewOutboxRepository(db)

	// Start outbox worker (replaces direct async publishing)
	outboxWorker := messaging.NewOutboxWorker(outboxRepo, rmq)
	outboxWorker.Start()

	// Initialize services with outbox pattern
	reportService := service.NewReportService(reportRepo, outboxRepo, cfg.Anonymous, rmq, db)
	voteService := service.NewVoteService(voteRepo, reportRepo, outboxRepo, rmq)

	// Initialize handlers
	reportHandler := handler.NewReportHandler(reportService)
	voteHandler := handler.NewVoteHandler(voteService)

	// Setup Gin router
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

	// Admin/monitoring routes
	r.GET("/admin/outbox/stats", func(c *gin.Context) {
		stats, err := outboxWorker.GetStats()
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"outbox_stats": stats})
	})

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("\nShutdown signal received...")
		outboxWorker.Stop()
		log.Println("Report service stopped gracefully")
		os.Exit(0)
	}()

	addr := fmt.Sprintf(":%s", cfg.Server.Port)
	log.Printf("listening on %s", addr)

	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
