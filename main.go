package main

import (
	"go-ocr-gvision/parser"
	"go-ocr-gvision/vision"
	"log"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Set Google Credential
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"))
	ocrEngine := os.Getenv("OCR_ENGINE") // "googlevision" atau "tesseract"

	// Setup Gin
	r := gin.Default()
	r.MaxMultipartMemory = 8 << 20 // 8 MiB

	// Endpoint
	r.POST("/upload/:type", func(c *gin.Context) {
		docType := c.Param("type")

		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(400, gin.H{"error": "file tidak ditemukan"})
			return
		}

		// Simpan file
		dst := filepath.Join("uploads", file.Filename)
		if err := c.SaveUploadedFile(file, dst); err != nil {
			c.JSON(500, gin.H{"error": "gagal menyimpan file", "detail": err.Error()})
			return
		}

		// Jalankan OCR berdasarkan engine
		var text string
		if ocrEngine == "tesseract" {
			text, err = vision.ExtractTextWithPython(dst) // ← Panggil Python
		} else {
			text, err = vision.ExtractText(dst) // ← Google Vision
		}

		if err != nil {
			c.JSON(500, gin.H{"error": "OCR gagal", "detail": err.Error()})
			return
		}

		// Parsing berdasarkan tipe dokumen
		switch docType {
		case "ktp":
			c.JSON(200, parser.ParseKTP(text))
		case "kk":
			c.JSON(200, parser.ParseKK(text))
		case "ijazah":
			c.JSON(200, parser.ParseIjazah(text))
		case "transkrip":
			c.JSON(200, parser.ParseTranskrip(text))
		default:
			c.JSON(400, gin.H{"error": "Tipe dokumen tidak dikenali"})
		}
	})

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	r.Run(":" + port)
}
