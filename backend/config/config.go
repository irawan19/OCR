package config

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"ocr-backend/models" // Import model Anda di sini

	"gorm.io/gorm"
)

// Config holds the application's configuration
type Config struct {
	ServerPort           string
	DatabaseURL          string
	JWTSecret            string
	MinioEndpoint        string
	MinioAccessKeyID     string
	MinioSecretAccessKey string
	MinioUseSSL          bool
	MinioBucketName      string // Tambahkan bucket name
	AIBackendURL         string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	minioUseSSL, err := strconv.ParseBool(os.Getenv("MINIO_USE_SSL"))
	if err != nil {
		minioUseSSL = false // Default to false if not set or invalid
	}

	return &Config{
		ServerPort:           getEnv("SERVER_PORT", "8080"),
		DatabaseURL:          getEnv("DATABASE_URL", ""),
		JWTSecret:            getEnv("JWT_SECRET", "supersecretjwtkey_ubah_ini_untuk_produksi_lebih_kuat"),
		MinioEndpoint:        getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinioAccessKeyID:     getEnv("MINIO_ACCESS_KEY_ID", "minioadmin"),
		MinioSecretAccessKey: getEnv("MINIO_SECRET_ACCESS_KEY", "minioadmin"),
		MinioUseSSL:          minioUseSSL,
		MinioBucketName:      getEnv("MINIO_BUCKET_NAME", "ocr-documents"), // Default bucket name
		AIBackendURL:         getEnv("AI_BACKEND_URL", "http://localhost:5000"),
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// MigrateModels automatically migrates the database schema for the given models.
// This function should be called during application startup.
func MigrateModels(db *gorm.DB) error {
	log.Println("Starting database migrations...")
	err := db.AutoMigrate(
		&models.User{},
		&models.Document{},
		&models.ScholarshipApplication{},
		// Tambahkan semua model GORM Anda di sini agar di-migrate
	)
	if err != nil {
		return fmt.Errorf("failed to auto migrate database models: %w", err)
	}
	log.Println("Database migrations completed successfully.")
	return nil
}
