package models

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// User represents a user in the system
type User struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Username  string         `gorm:"unique;not null" json:"username"`
	Email     string         `gorm:"unique;not null" json:"email"` // <-- Tambahkan baris ini
	Password  string         `gorm:"not null" json:"-"`            // JSON tag "-" untuk menyembunyikan password
	Role      string         `gorm:"default:'user'" json:"role"`   // 'user' or 'admin'
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"` // Soft delete

	Documents               []Document               `gorm:"foreignKey:UserID" json:"documents"`
	ScholarshipApplications []ScholarshipApplication `gorm:"foreignKey:UserID" json:"scholarship_applications"`
}

// HashPassword hashes the user's password using bcrypt
func (u *User) HashPassword(password string) error {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(bytes)
	return nil
}

// CheckPasswordHash compares a hashed password with a plain-text password
func (u *User) CheckPasswordHash(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}
