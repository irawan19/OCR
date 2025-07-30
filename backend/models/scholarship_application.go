package models

import (
	"time"
)

type ScholarshipApplication struct {
	ID                  uint      `gorm:"primaryKey" json:"id"`
	UserID              uint      `json:"user_id"`
	ApplicantName       string    `gorm:"not null" json:"applicant_name"`
	IPK                 float64   `json:"ipk"`
	KTPDocumentID       uint      `json:"ktp_document_id"`
	KKDocumentID        uint      `json:"kk_document_id"`
	IjazahDocumentID    uint      `json:"ijazah_document_id"`
	TranskripDocumentID uint      `json:"transkrip_document_id"`
	Status              string    `gorm:"default:'pending'" json:"status"` // e.g., pending, reviewed, accepted, rejected
	AdminNotes          string    `json:"admin_notes"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
	User                User      `gorm:"foreignKey:UserID"`
	KTPDocument         Document  `gorm:"foreignKey:KTPDocumentID"`
	KKDocument          Document  `gorm:"foreignKey:KKDocumentID"`
	IjazahDocument      Document  `gorm:"foreignKey:IjazahDocumentID"`
	TranskripDocument   Document  `gorm:"foreignKey:TranskripDocumentID"`
}
