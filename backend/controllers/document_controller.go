package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"ocr-backend/models"
	"ocr-backend/services"
)

type DocumentController struct {
	DB           *gorm.DB
	MinioService *services.MinioService
	AIService    *services.AIService
}

func NewDocumentController(db *gorm.DB, minioService *services.MinioService, aiService *services.AIService) *DocumentController {
	return &DocumentController{DB: db, MinioService: minioService, AIService: aiService}
}

// UploadDocument
// @Summary Upload a new document
// @Description Uploads a file (PDF or image) to MinIO and records its metadata in the database. Triggers AI processing.
// @Tags Documents
// @Accept mpfd
// @Produce json
// @Param document formData file true "Document file to upload"
// @Param doc_type formData string true "Type of document (e.g., KTP, KK, Ijazah, Transkrip)"
// @Security ApiKeyAuth
// @Success 201 {object} models.Document
// @Failure 400 {object} map[string]string "error: Invalid input"
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Failure 500 {object} map[string]string "error: Failed to upload document"
// @Router /documents/upload [post]
func (ctrl *DocumentController) UploadDocument(c *gin.Context) {
	userID := c.MustGet("userID").(uint) // Get userID from JWT middleware

	fileHeader, err := c.FormFile("document")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get document file: " + err.Error()})
		return
	}

	docType := c.PostForm("doc_type")
	if docType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Document type is required"})
		return
	}

	file, err := fileHeader.Open() // file is io.ReaderCloser
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open file: " + err.Error()})
		return
	}
	defer file.Close()

	// Upload file to MinIO - Sesuaikan pemanggilan ini
	// minioPath, err := ctrl.MinioService.UploadFile(c.Request.Context(), file, fileHeader.Filename, fileHeader.Size, fileHeader.Header.Get("Content-Type"))
	minioPath, err := ctrl.MinioService.UploadFile(c.Request.Context(), file, fileHeader.Filename, fileHeader.Size, fileHeader.Header.Get("Content-Type"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload file to storage: " + err.Error()})
		return
	}

	// Create document record in database
	document := models.Document{
		UserID:          userID,
		Filename:        fileHeader.Filename,
		MimeType:        fileHeader.Header.Get("Content-Type"),
		Size:            fileHeader.Size,
		MinioPath:       minioPath,
		DocType:         docType,
		IsProcessed:     false,
		IsValid:         false,
		ExtractedData:   "{}", // <-- UBAH KE INI! Ini adalah objek JSON kosong yang valid.
		ValidationNotes: "",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := ctrl.DB.Create(&document).Error; err != nil {
		// If DB fails, try to remove from MinIO (best effort)
		_ = ctrl.MinioService.DeleteFile(c.Request.Context(), minioPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save document record: " + err.Error()})
		return
	}

	// Asynchronously trigger AI processing
	go func(docID uint, minioPath string) {
		// Gunakan context.Background() atau context yang relevan jika goroutine ini hidup lebih lama dari context request
		// Untuk AI processing asinkron, context.Background() adalah pilihan yang aman.
		ctx := context.Background()

		fmt.Printf("Triggering AI processing for document ID: %d, MinIO path: %s\n", docID, minioPath)

		// Get the file from MinIO again to pass to AI service - Sesuaikan pemanggilan ini
		fileReader, err := ctrl.MinioService.GetFile(ctx, minioPath) // Menggunakan ctx yang baru
		if err != nil {
			fmt.Printf("Error getting file from MinIO for AI processing: %v\n", err)
			return
		}
		defer func() {
			if closer, ok := fileReader.(io.Closer); ok {
				closer.Close()
			}
		}()

		// Fetch the document again to get its current state, including Filename and MimeType
		var currentDocument models.Document
		if err := ctrl.DB.First(&currentDocument, docID).Error; err != nil {
			fmt.Printf("Error fetching document %d for AI processing: %v\n", docID, err)
			return
		}

		extractedData, err := ctrl.AIService.ProcessDocument(fileReader, currentDocument.Filename, currentDocument.MimeType)
		if err != nil {
			fmt.Printf("AI processing failed for document %d: %v\n", docID, err)
			// Update document status to indicate processing failure
			ctrl.DB.Model(&models.Document{}).Where("id = ?", docID).Updates(map[string]interface{}{
				"is_processed":     true,
				"is_valid":         false, // Mark as invalid on processing failure
				"validation_notes": "AI processing failed: " + err.Error(),
				"updated_at":       time.Now(),
			})
			return
		}

		// Update document with extracted data and status
		jsonExtractedData, _ := json.Marshal(extractedData) // Handle error in production
		ctrl.DB.Model(&models.Document{}).Where("id = ?", docID).Updates(map[string]interface{}{
			"is_processed":   true,
			"extracted_data": string(jsonExtractedData),
			"updated_at":     time.Now(),
		})
		fmt.Printf("AI processing completed for document ID: %d. Data: %v\n", docID, extractedData)

	}(document.ID, minioPath) // Pass docID and minioPath to goroutine

	c.JSON(http.StatusCreated, document)
}

// GetDocumentByID (tidak ada perubahan)
func (ctrl *DocumentController) GetDocumentByID(c *gin.Context) {
	docID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document ID"})
		return
	}

	var document models.Document
	if err := ctrl.DB.Preload("User").First(&document, docID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve document: " + err.Error()})
		return
	}

	userID := c.MustGet("userID").(uint)
	userRole := c.MustGet("role").(string)
	if document.UserID != userID && userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	c.JSON(http.StatusOK, document)
}

// GetDocumentsByUserID (tidak ada perubahan)
func (ctrl *DocumentController) GetDocumentsByUserID(c *gin.Context) {
	targetUserID, err := strconv.ParseUint(c.Param("userID"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid User ID"})
		return
	}

	currentUserID := c.MustGet("userID").(uint)
	userRole := c.MustGet("role").(string)

	if uint(targetUserID) != currentUserID && userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied. You can only view your own documents."})
		return
	}

	var documents []models.Document
	if err := ctrl.DB.Where("user_id = ?", targetUserID).Find(&documents).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve documents: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, documents)
}

// GetAllDocuments (Admin Only) (tidak ada perubahan)
func (ctrl *DocumentController) GetAllDocuments(c *gin.Context) {
	var documents []models.Document
	if err := ctrl.DB.Preload("User").Find(&documents).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve documents: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, documents)
}

// DeleteDocument (Admin Only) (tidak ada perubahan)
func (ctrl *DocumentController) DeleteDocument(c *gin.Context) {
	docID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document ID"})
		return
	}

	var document models.Document
	if err := ctrl.DB.First(&document, docID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve document: " + err.Error()})
		return
	}

	if err := ctrl.MinioService.DeleteFile(c.Request.Context(), document.MinioPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete file from storage: " + err.Error()})
		return
	}

	if err := ctrl.DB.Delete(&document).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete document record: " + err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// ProcessDocumentWithAI
// @Summary Manually trigger AI processing for a document
// @Description Allows triggering AI processing for an already uploaded document.
// @Tags Documents
// @Produce json
// @Param id path int true "Document ID"
// @Security ApiKeyAuth
// @Success 200 {object} map[string]string "message: AI processing triggered successfully"
// @Failure 400 {object} map[string]string "error: Invalid ID"
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Failure 403 {object} map[string]string "error: Access denied"
// @Failure 404 {object} map[string]string "error: Document not found"
// @Failure 500 {object} map[string]string "error: Failed to trigger AI processing"
// @Router /documents/{id}/process [post]
func (ctrl *DocumentController) ProcessDocumentWithAI(c *gin.Context) {
	docID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document ID"})
		return
	}

	var document models.Document
	if err := ctrl.DB.First(&document, docID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve document: " + err.Error()})
		return
	}

	userID := c.MustGet("userID").(uint)
	userRole := c.MustGet("role").(string)

	if document.UserID != userID && userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied. You can only trigger processing for your own documents."})
		return
	}

	// Asynchronously trigger AI processing
	go func(doc models.Document) { // Hanya passing document object
		ctx := context.Background() // Gunakan context.Background() untuk goroutine asinkron

		fmt.Printf("Manually triggering AI processing for document ID: %d, MinIO path: %s\n", doc.ID, doc.MinioPath)

		// Get the file from MinIO to pass to AI service
		fileReader, err := ctrl.MinioService.GetFile(ctx, doc.MinioPath) // Menggunakan MinioPath dari doc
		if err != nil {
			fmt.Printf("Failed to retrieve file from storage for AI processing for document %d: %v\n", doc.ID, err)
			ctrl.DB.Model(&models.Document{}).Where("id = ?", doc.ID).Updates(map[string]interface{}{
				"is_processed":     true,
				"is_valid":         false,
				"validation_notes": "Failed to retrieve file for AI processing: " + err.Error(),
				"updated_at":       time.Now(),
			})
			return
		}
		defer func() {
			if closer, ok := fileReader.(io.Closer); ok {
				closer.Close()
			}
		}()

		extractedData, err := ctrl.AIService.ProcessDocument(fileReader, doc.Filename, doc.MimeType)
		if err != nil {
			fmt.Printf("AI processing failed for document %d: %v\n", doc.ID, err)
			ctrl.DB.Model(&models.Document{}).Where("id = ?", doc.ID).Updates(map[string]interface{}{
				"is_processed":     true,
				"is_valid":         false,
				"validation_notes": "AI reprocessing failed: " + err.Error(),
				"updated_at":       time.Now(),
			})
			return
		}

		jsonExtractedData, _ := json.Marshal(extractedData)
		ctrl.DB.Model(&models.Document{}).Where("id = ?", doc.ID).Updates(map[string]interface{}{
			"is_processed":   true,
			"extracted_data": string(jsonExtractedData),
			"updated_at":     time.Now(),
		})
		fmt.Printf("AI reprocessing completed for document ID: %d. Data: %v\n", doc.ID, extractedData)

	}(document) // Hanya passing document object

	c.JSON(http.StatusOK, gin.H{"message": "AI processing triggered successfully. Check document status later."})
}
