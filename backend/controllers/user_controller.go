package controllers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"ocr-backend/models"
)

type UserController struct {
	DB *gorm.DB
}

func NewUserController(db *gorm.DB) *UserController {
	return &UserController{DB: db}
}

// GetUserByID
// @Summary Get user by ID
// @Description Retrieve user details by ID
// @Tags Users
// @Produce json
// @Param id path int true "User ID"
// @Security ApiKeyAuth
// @Success 200 {object} models.User
// @Failure 400 {object} map[string]string "error: Invalid ID"
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Failure 403 {object} map[string]string "error: Access denied"
// @Failure 404 {object} map[string]string "error: User not found"
// @Router /users/{id} [get]
func (ctrl *UserController) GetUserByID(c *gin.Context) {
	userID := c.MustGet("userID").(uint) // User ID from authenticated JWT
	userRole := c.MustGet("role").(string)

	idParam, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var user models.User
	if err := ctrl.DB.First(&user, idParam).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user: " + err.Error()})
		return
	}

	// Only allow user to get their own profile, or if they are admin
	if user.ID != userID && userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// GetAllUsers (Admin Only)
// @Summary Get all users
// @Description Retrieve a list of all users in the system (Admin only)
// @Tags Users
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {array} models.User
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Failure 403 {object} map[string]string "error: Forbidden"
// @Failure 500 {object} map[string]string "error: Failed to retrieve users"
// @Router /admin/users [get]
func (ctrl *UserController) GetAllUsers(c *gin.Context) {
	// This route is protected by AdminAuthMiddleware in api_routes.go
	var users []models.User
	if err := ctrl.DB.Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve users: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, users)
}

// UpdateUser
// @Summary Update user details
// @Description Update details of an existing user. Users can update their own profile, admin can update any user.
// @Tags Users
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param user body models.User true "Updated user details (username, role - admin only for role)"
// @Security ApiKeyAuth
// @Success 200 {object} models.User
// @Failure 400 {object} map[string]string "error: Invalid input"
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Failure 403 {object} map[string]string "error: Access denied"
// @Failure 404 {object} map[string]string "error: User not found"
// @Failure 500 {object} map[string]string "error: Failed to update user"
// @Router /users/{id} [put]
func (ctrl *UserController) UpdateUser(c *gin.Context) {
	userID := c.MustGet("userID").(uint) // User ID from authenticated JWT
	userRole := c.MustGet("role").(string)

	idParam, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var existingUser models.User
	if err := ctrl.DB.First(&existingUser, idParam).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user: " + err.Error()})
		return
	}

	// Check authorization: User can only update their own profile, or if admin
	if existingUser.ID != userID && userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	var updatedUser models.User
	if err := c.ShouldBindJSON(&updatedUser); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	// Update only allowed fields.
	// Users are not allowed to change their own role unless they are admin.
	// Users cannot change their password via this endpoint (should be separate "change password" flow)
	if updatedUser.Username != "" {
		existingUser.Username = updatedUser.Username
	}

	// Only admin can change role
	if userRole == "admin" && updatedUser.Role != "" {
		if updatedUser.Role != "user" && updatedUser.Role != "admin" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role. Only 'user' or 'admin' allowed."})
			return
		}
		existingUser.Role = updatedUser.Role
	} else if updatedUser.Role != "" && userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not authorized to change user role."})
		return
	}

	existingUser.UpdatedAt = time.Now()

	if err := ctrl.DB.Save(&existingUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, existingUser)
}

// DeleteUser
// @Summary Delete a user
// @Description Delete a user by ID. Users can delete their own account, admin can delete any user.
// @Tags Users
// @Produce json
// @Param id path int true "User ID"
// @Security ApiKeyAuth
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "error: Invalid ID"
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Failure 403 {object} map[string]string "error: Access denied"
// @Failure 404 {object} map[string]string "error: User not found"
// @Failure 500 {object} map[string]string "error: Failed to delete user"
// @Router /users/{id} [delete]
func (ctrl *UserController) DeleteUser(c *gin.Context) {
	userID := c.MustGet("userID").(uint) // User ID from authenticated JWT
	userRole := c.MustGet("role").(string)

	idParam, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var user models.User
	if err := ctrl.DB.First(&user, idParam).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user: " + err.Error()})
		return
	}

	// Check authorization: User can only delete their own account, or if admin
	if user.ID != userID && userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	if err := ctrl.DB.Delete(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user: " + err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}
