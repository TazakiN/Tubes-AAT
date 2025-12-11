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

	// Initialize layers
	reportRepo := repository.NewReportRepository(db)
	reportService := service.NewReportService(reportRepo, cfg.Anonymous)
	reportHandler := handler.NewReportHandler(reportService)

	// Setup Gin router
	r := gin.Default()

	// Health check
	r.GET("/health", reportHandler.Health)

	// Report routes
	r.POST("/", reportHandler.CreateReport)
	r.GET("/", reportHandler.GetReports)
	r.GET("/my", reportHandler.GetMyReports)
	r.GET("/:id", reportHandler.GetReportByID)
	r.PATCH("/:id/status", reportHandler.UpdateStatus)

	// Start server
	addr := fmt.Sprintf(":%s", cfg.Server.Port)
	log.Printf("Report service starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
