package services

import (
	"errors"

	"gorm.io/gorm"

	"ocr-backend/models"
	"ocr-backend/utils" // Untuk JWT
)

// AuthService handles user authentication logic
type AuthService struct {
	DB *gorm.DB
}

// NewAuthService creates a new AuthService
func NewAuthService(db *gorm.DB) *AuthService {
	return &AuthService{DB: db}
}

// IsUsernameOrEmailTaken checks if a username or email already exists
func (s *AuthService) IsUsernameOrEmailTaken(username, email string) bool {
	var user models.User
	// Cari berdasarkan username ATAU email
	result := s.DB.Where("username = ? OR email = ?", username, email).First(&user)
	return result.Error == nil // If no error, user exists
}

// RegisterUser registers a new user in the database
func (s *AuthService) RegisterUser(user *models.User) error {
	// Password should already be hashed by controller/model method
	if err := s.DB.Create(user).Error; err != nil {
		return err
	}
	return nil
}

// LoginUser authenticates a user and returns a JWT token
func (s *AuthService) LoginUser(identifier, password string) (string, error) {
	var user models.User
	// Try to find user by username or email
	result := s.DB.Where("username = ? OR email = ?", identifier, identifier).First(&user)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return "", errors.New("invalid credentials")
		}
		return "", result.Error
	}

	// Check password hash
	if !user.CheckPasswordHash(password) {
		return "", errors.New("invalid credentials")
	}

	// Generate JWT token
	token, err := utils.GenerateJWT(user.ID, user.Role) // Assuming GenerateJWT takes userID and role
	if err != nil {
		return "", errors.New("failed to generate token")
	}

	return token, nil
}
