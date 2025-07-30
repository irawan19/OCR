package models

import (
	"time"
)

type Document struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	UserID          uint      `json:"user_id"` // User who uploaded the document
	Filename        string    `gorm:"not null" json:"filename"`
	MimeType        string    `json:"mime_type"`
	Size            int64     `json:"size"`
	MinioPath       string    `gorm:"not null" json:"minio_path"` // Path in MinIO bucket
	DocType         string    `json:"doc_type"`                   // e.g., KTP, KK, Ijazah, Transkrip
	IsProcessed     bool      `gorm:"default:false" json:"is_processed"`
	IsValid         bool      `gorm:"default:false" json:"is_valid"`    // Based on AI validation
	ExtractedData   string    `gorm:"type:jsonb" json:"extracted_data"` // JSONB for storing OCR extracted data
	ValidationNotes string    `json:"validation_notes"`                 // Notes from AI validation
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	User            User      `gorm:"foreignKey:UserID"`
}
