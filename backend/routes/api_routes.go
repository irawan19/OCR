package routes

import (
	"ocr-backend/controllers"
	"ocr-backend/middlewares"
	"ocr-backend/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetupAPIRoutes(router *gin.Engine, db *gorm.DB, minioService *services.MinioService) {
	// Initialize services
	authService := services.NewAuthService(db)
	aiService := services.NewAIService("http://127.0.0.1:5000") // This will be the AI service URL in Docker

	// Initialize controllers
	authController := controllers.NewAuthController(authService)
	userController := controllers.NewUserController(db)
	documentController := controllers.NewDocumentController(db, minioService, aiService)
	scholarshipController := controllers.NewScholarshipController(db, aiService)

	v1 := router.Group("/api/v1")
	{
		// Public routes
		v1.POST("/register", authController.Register)
		v1.POST("/login", authController.Login)

		// Authenticated routes
		authenticated := v1.Group("/")
		authenticated.Use(middlewares.AuthMiddleware())
		{
			// User routes
			authenticated.GET("/users/:id", userController.GetUserByID)
			authenticated.PUT("/users/:id", userController.UpdateUser)
			authenticated.DELETE("/users/:id", userController.DeleteUser)

			// Document routes
			authenticated.POST("/documents/upload", documentController.UploadDocument)
			authenticated.GET("/documents/:id", documentController.GetDocumentByID)
			authenticated.GET("/documents/user/:userID", documentController.GetDocumentsByUserID)
			authenticated.POST("/documents/:id/process", documentController.ProcessDocumentWithAI)
			authenticated.DELETE("/documents/:id", documentController.DeleteDocument) // <--- Pindahkan ke sini!

			// Scholarship application routes
			authenticated.POST("/scholarships", scholarshipController.CreateScholarshipApplication)
			authenticated.GET("/scholarships/:id", scholarshipController.GetScholarshipApplicationByID)
			authenticated.GET("/scholarships", scholarshipController.GetAllScholarshipApplications)
			authenticated.PUT("/scholarships/:id", scholarshipController.UpdateScholarshipApplication)
		}

		// Admin routes (requires both authentication and admin role)
		admin := v1.Group("/admin")
		admin.Use(middlewares.AuthMiddleware(), middlewares.AdminAuthMiddleware())
		{
			admin.GET("/users", userController.GetAllUsers)
			admin.GET("/documents", documentController.GetAllDocuments)
			// Hapus baris DELETE dokumen dari sini jika dipindahkan ke atas
			// admin.DELETE("/documents/:id", documentController.DeleteDocument) // <--- Hapus baris ini
			admin.PUT("/scholarships/:id/status", scholarshipController.UpdateScholarshipApplicationStatus)
			admin.POST("/scholarships/:id/validate", scholarshipController.ValidateApplicationWithAI)
		}
	}
}
