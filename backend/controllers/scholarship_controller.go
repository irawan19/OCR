package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"ocr-backend/models"
	"ocr-backend/services"
)

type ScholarshipController struct {
	DB        *gorm.DB
	AIService *services.AIService
}

func NewScholarshipController(db *gorm.DB, aiService *services.AIService) *ScholarshipController {
	return &ScholarshipController{DB: db, AIService: aiService}
}

// CreateScholarshipApplication
// @Summary Create a new scholarship application
// @Description Allows a user to submit a new scholarship application with references to uploaded documents.
// @Tags Scholarship Applications
// @Accept json
// @Produce json
// @Param application body models.ScholarshipApplication true "Scholarship application data"
// @Security ApiKeyAuth
// @Success 201 {object} models.ScholarshipApplication
// @Failure 400 {object} map[string]string "error: Invalid input"
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Failure 404 {object} map[string]string "error: Referenced document not found"
// @Failure 500 {object} map[string]string "error: Failed to create application"
// @Router /scholarships [post]
func (ctrl *ScholarshipController) CreateScholarshipApplication(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var application models.ScholarshipApplication
	if err := c.ShouldBindJSON(&application); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	application.UserID = userID
	application.Status = "pending" // Default status
	application.CreatedAt = time.Now()
	application.UpdatedAt = time.Now()

	// Validate if referenced documents belong to the user and exist
	docIDs := []uint{application.KTPDocumentID, application.KKDocumentID, application.IjazahDocumentID, application.TranskripDocumentID}
	for _, docID := range docIDs {
		if docID == 0 { // Skip if document ID is not provided (optional documents)
			continue
		}
		var doc models.Document
		if err := ctrl.DB.Where("id = ? AND user_id = ?", docID, userID).First(&doc).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Referenced document (ID: %d) not found or does not belong to you.", docID)})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error checking document: " + err.Error()})
			return
		}
	}

	if err := ctrl.DB.Create(&application).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create scholarship application: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, application)
}

// GetScholarshipApplicationByID
// @Summary Get scholarship application by ID
// @Description Retrieve a specific scholarship application by its ID.
// @Tags Scholarship Applications
// @Produce json
// @Param id path int true "Application ID"
// @Security ApiKeyAuth
// @Success 200 {object} models.ScholarshipApplication
// @Failure 400 {object} map[string]string "error: Invalid ID"
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Failure 403 {object} map[string]string "error: Access denied"
// @Failure 404 {object} map[string]string "error: Application not found"
// @Router /scholarships/{id} [get]
func (ctrl *ScholarshipController) GetScholarshipApplicationByID(c *gin.Context) {
	appID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid application ID"})
		return
	}

	userID := c.MustGet("userID").(uint)
	userRole := c.MustGet("role").(string)

	var application models.ScholarshipApplication
	// Preload related documents and user for richer response
	query := ctrl.DB.Preload("User").
		Preload("KTPDocument").
		Preload("KKDocument").
		Preload("IjazahDocument").
		Preload("TranskripDocument")

	if userRole != "admin" {
		query = query.Where("user_id = ?", userID)
	}

	if err := query.First(&application, appID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Scholarship application not found or you don't have access"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve application: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, application)
}

// GetAllScholarshipApplications
// @Summary Get all scholarship applications
// @Description Retrieve all scholarship applications (admin can see all, users see their own).
// @Tags Scholarship Applications
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {array} models.ScholarshipApplication
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Failure 500 {object} map[string]string "error: Failed to retrieve applications"
// @Router /scholarships [get]
func (ctrl *ScholarshipController) GetAllScholarshipApplications(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	userRole := c.MustGet("role").(string)

	var applications []models.ScholarshipApplication
	query := ctrl.DB.Preload("User").
		Preload("KTPDocument").
		Preload("KKDocument").
		Preload("IjazahDocument").
		Preload("TranskripDocument")

	if userRole != "admin" {
		query = query.Where("user_id = ?", userID)
	}

	if err := query.Find(&applications).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve scholarship applications: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, applications)
}

// UpdateScholarshipApplication
// @Summary Update an existing scholarship application
// @Description Allows a user to update their own scholarship application.
// @Tags Scholarship Applications
// @Accept json
// @Produce json
// @Param id path int true "Application ID"
// @Param application body models.ScholarshipApplication true "Updated scholarship application data"
// @Security ApiKeyAuth
// @Success 200 {object} models.ScholarshipApplication
// @Failure 400 {object} map[string]string "error: Invalid input"
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Failure 403 {object} map[string]string "error: Access denied"
// @Failure 404 {object} map[string]string "error: Application not found"
// @Failure 500 {object} map[string]string "error: Failed to update application"
// @Router /scholarships/{id} [put]
func (ctrl *ScholarshipController) UpdateScholarshipApplication(c *gin.Context) {
	appID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid application ID"})
		return
	}

	userID := c.MustGet("userID").(uint)
	userRole := c.MustGet("role").(string)

	var existingApplication models.ScholarshipApplication
	if err := ctrl.DB.First(&existingApplication, appID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Scholarship application not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve existing application: " + err.Error()})
		return
	}

	// Only allow user to update their own application unless they are admin
	if existingApplication.UserID != userID && userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied. You can only update your own applications."})
		return
	}

	var updatedApplication models.ScholarshipApplication
	if err := c.ShouldBindJSON(&updatedApplication); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	// Prevent direct update of UserID, Status, CreatedAt
	updatedApplication.ID = existingApplication.ID
	updatedApplication.UserID = existingApplication.UserID
	updatedApplication.Status = existingApplication.Status // Status can only be updated by admin via separate endpoint
	updatedApplication.CreatedAt = existingApplication.CreatedAt
	updatedApplication.UpdatedAt = time.Now()

	// Update only allowed fields.
	// For simplicity, we directly assign here. For more control, use `Select` or `Omit`.
	existingApplication.ApplicantName = updatedApplication.ApplicantName
	existingApplication.IPK = updatedApplication.IPK
	existingApplication.KTPDocumentID = updatedApplication.KTPDocumentID
	existingApplication.KKDocumentID = updatedApplication.KKDocumentID
	existingApplication.IjazahDocumentID = updatedApplication.IjazahDocumentID
	existingApplication.TranskripDocumentID = updatedApplication.TranskripDocumentID

	// Validate if new referenced documents belong to the user
	docIDs := []uint{existingApplication.KTPDocumentID, existingApplication.KKDocumentID, existingApplication.IjazahDocumentID, existingApplication.TranskripDocumentID}
	for _, docID := range docIDs {
		if docID == 0 {
			continue
		}
		var doc models.Document
		if err := ctrl.DB.Where("id = ? AND user_id = ?", docID, userID).First(&doc).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Referenced document (ID: %d) not found or does not belong to you.", docID)})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error checking document: " + err.Error()})
			return
		}
	}

	if err := ctrl.DB.Save(&existingApplication).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update scholarship application: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, existingApplication)
}

// UpdateScholarshipApplicationStatus (Admin Only)
// @Summary Update scholarship application status
// @Description Allows an admin to update the status of a scholarship application.
// @Tags Scholarship Applications
// @Accept json
// @Produce json
// @Param id path int true "Application ID"
// @Param status body object{status=string,admin_notes=string} true "New status (e.g., reviewed, accepted, rejected) and admin notes"
// @Security ApiKeyAuth
// @Success 200 {object} models.ScholarshipApplication
// @Failure 400 {object} map[string]string "error: Invalid input"
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Failure 403 {object} map[string]string "error: Forbidden"
// @Failure 404 {object} map[string]string "error: Application not found"
// @Failure 500 {object} map[string]string "error: Failed to update status"
// @Router /admin/scholarships/{id}/status [put]
func (ctrl *ScholarshipController) UpdateScholarshipApplicationStatus(c *gin.Context) {
	appID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid application ID"})
		return
	}

	var statusUpdate struct {
		Status     string `json:"status" binding:"required"`
		AdminNotes string `json:"admin_notes"`
	}
	if err := c.ShouldBindJSON(&statusUpdate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	var application models.ScholarshipApplication
	if err := ctrl.DB.First(&application, appID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Scholarship application not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve application: " + err.Error()})
		return
	}

	// Basic validation for status values
	validStatuses := map[string]bool{
		"pending": true, "reviewed": true, "accepted": true, "rejected": true,
	}
	if _, ok := validStatuses[statusUpdate.Status]; !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status value. Allowed: pending, reviewed, accepted, rejected."})
		return
	}

	application.Status = statusUpdate.Status
	application.AdminNotes = statusUpdate.AdminNotes
	application.UpdatedAt = time.Now()

	if err := ctrl.DB.Save(&application).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update scholarship status: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, application)
}

// ValidateApplicationWithAI (Admin Only)
// @Summary Validate scholarship application data using AI
// @Description Compares applicant's input data with data extracted from uploaded documents via AI.
// @Tags Scholarship Applications
// @Produce json
// @Param id path int true "Application ID"
// @Security ApiKeyAuth
// @Success 200 {object} map[string]interface{} "message: AI validation completed, is_valid: true/false, notes: validation_notes"
// @Failure 400 {object} map[string]string "error: Invalid ID"
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Failure 403 {object} map[string]string "error: Forbidden"
// @Failure 404 {object} map[string]string "error: Application or documents not found"
// @Failure 500 {object} map[string]string "error: Failed to perform AI validation"
// @Router /admin/scholarships/{id}/validate [post]
func (ctrl *ScholarshipController) ValidateApplicationWithAI(c *gin.Context) {
	appID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid application ID"})
		return
	}

	var application models.ScholarshipApplication
	// Eager load all related documents
	if err := ctrl.DB.Preload("KTPDocument").
		Preload("KKDocument").
		Preload("IjazahDocument").
		Preload("TranskripDocument").
		First(&application, appID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Scholarship application not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve application: " + err.Error()})
		return
	}

	// Aggregate all extracted data from linked documents
	extractedData := make(map[string]interface{})

	// Helper function to process document's extracted data
	// Perbaikan di sini: Parameter doc sekarang adalah pointer (*models.Document)
	processDocExtractedData := func(doc *models.Document, keyPrefix string) error {
		// Cek jika doc adalah nil, ini bisa terjadi jika relasi tidak ditemukan atau tidak dimuat
		if doc == nil || doc.ID == 0 || doc.ExtractedData == "" {
			return nil // No document or no extracted data
		}
		var docExtracted map[string]interface{}
		if err := json.Unmarshal([]byte(doc.ExtractedData), &docExtracted); err != nil {
			return fmt.Errorf("failed to unmarshal extracted data for %s: %w", doc.Filename, err)
		}
		// Prefix keys to avoid conflicts (e.g., ktp_nama, kk_alamat)
		for k, v := range docExtracted {
			extractedData[keyPrefix+k] = v
		}
		return nil
	}

	// Perbaikan di sini: Ambil alamat (&) dari setiap struct dokumen
	if err := processDocExtractedData(&application.KTPDocument, "ktp_"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := processDocExtractedData(&application.KKDocument, "kk_"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := processDocExtractedData(&application.IjazahDocument, "ijazah_"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := processDocExtractedData(&application.TranskripDocument, "transkrip_"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Call AI Service for validation
	isValid, notes, err := ctrl.AIService.ValidateScholarshipData(&application, extractedData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI validation failed: " + err.Error()})
		return
	}

	// Update document validity and notes in the database
	application.AdminNotes = notes // Store AI notes in admin notes for now
	// Anda mungkin ingin menambahkan logika untuk memperbarui `IsValid` atau kolom lain di `ScholarshipApplication`
	// berdasarkan hasil AI secara keseluruhan, tidak hanya notes.
	if err := ctrl.DB.Save(&application).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save AI validation result to application: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "AI validation completed",
		"is_valid": isValid,
		"notes":    notes,
	})
}
