package controllers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"ocr-backend/models"
	"ocr-backend/services"
)

// AuthController handles authentication related requests
type AuthController struct {
	AuthService *services.AuthService
}

// NewAuthController creates a new AuthController
func NewAuthController(authService *services.AuthService) *AuthController {
	return &AuthController{AuthService: authService}
}

// RegisterRequest defines the request body for user registration
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3"`
	Email    string `json:"email" binding:"required,email"` // <-- Tambahkan baris ini
	Password string `json:"password" binding:"required,min=6"`
}

// LoginRequest defines the request body for user login
type LoginRequest struct {
	Identifier string `json:"identifier" binding:"required"` // Could be username or email
	Password   string `json:"password" binding:"required"`
}

// Register
// @Summary Register a new user
// @Description Register a new user with username, email, and password
// @Tags Auth
// @Accept json
// @Produce json
// @Param user body RegisterRequest true "Register user"
// @Success 201 {object} map[string]interface{} "message: User registered successfully, user: {models.UserWithoutPassword}"
// @Failure 400 {object} map[string]string "error: Invalid input"
// @Failure 409 {object} map[string]string "error: Username or Email already taken"
// @Failure 500 {object} map[string]string "error: Failed to register user"
// @Router /register [post]
func (ctrl *AuthController) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	// Cek apakah username atau email sudah ada
	if ctrl.AuthService.IsUsernameOrEmailTaken(req.Username, req.Email) {
		c.JSON(http.StatusConflict, gin.H{"error": "Username or Email is already taken"})
		return
	}

	user := models.User{
		Username:  req.Username,
		Email:     req.Email, // <-- Pastikan ini dipetakan
		Role:      "user",    // Default role
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := user.HashPassword(req.Password); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	if err := ctrl.AuthService.RegisterUser(&user); err != nil {
		// Periksa jika errornya karena constraint unique
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			c.JSON(http.StatusConflict, gin.H{"error": "Username or Email is already taken"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register user: " + err.Error()})
		return
	}

	// Remove password for response
	user.Password = "" // Jangan kirim password kembali ke frontend
	c.JSON(http.StatusCreated, gin.H{"message": "User registered successfully", "user": user})
}

// Login
// @Summary Log in a user
// @Description Authenticate user by username/email and password
// @Tags Auth
// @Accept json
// @Produce json
// @Param credentials body LoginRequest true "User credentials"
// @Success 200 {object} map[string]string "token: JWT Token"
// @Failure 400 {object} map[string]string "error: Invalid input"
// @Failure 401 {object} map[string]string "error: Invalid credentials"
// @Failure 500 {object} map[string]string "error: Failed to login"
// @Router /login [post]
func (ctrl *AuthController) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	token, err := ctrl.AuthService.LoginUser(req.Identifier, req.Password)
	if err != nil {
		if strings.Contains(err.Error(), "invalid credentials") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to login: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}
