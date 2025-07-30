package main

import (
	"context" // Pastikan import context ada
	"fmt"
	"log"
	"os"
	"time"

	"ocr-backend/config"
	"ocr-backend/routes"
	"ocr-backend/services"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Jakarta",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
	)
	if os.Getenv("DATABASE_URL") != "" {
		dsn = os.Getenv("DATABASE_URL")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to initialize database, got error %v", err)
	}

	err = config.MigrateModels(db)
	if err != nil {
		log.Fatalf("failed to migrate database models, got error %v", err)
	}

	// Initialize MinIO service
	// --- PERBAIKAN DI SINI: Teruskan cfg.MinioBucketName ke NewMinioService ---
	minioService := services.NewMinioService(
		cfg.MinioEndpoint,
		cfg.MinioAccessKeyID,
		cfg.MinioSecretAccessKey,
		cfg.MinioUseSSL,
		cfg.MinioBucketName, // Parameter baru: bucketName
	)
	// --- PERBAIKAN DI SINI: Panggilan CreateBucket disederhanakan ---
	err = minioService.CreateBucket(context.Background()) // Tidak perlu parameter bucketName lagi
	if err != nil {
		log.Fatalf("Failed to create MinIO bucket '%s': %v", cfg.MinioBucketName, err)
	}

	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://127.0.0.1:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	routes.SetupAPIRoutes(router, db, minioService)

	log.Printf("Server starting on port %s", cfg.ServerPort)
	if err := router.Run(fmt.Sprintf(":%s", cfg.ServerPort)); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
