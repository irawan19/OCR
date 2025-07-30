package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt" // <-- Pastikan ini di-import
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"time"

	"ocr-backend/models"
)

type AIService struct {
	AIBackendURL string
	Client       *http.Client
}

func NewAIService(aiBackendURL string) *AIService {
	return &AIService{
		AIBackendURL: aiBackendURL,
		Client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// ProcessDocument mengirim file dokumen ke layanan AI untuk OCR dan ekstraksi data.
func (s *AIService) ProcessDocument(file io.Reader, filename, mimeType string) (map[string]interface{}, error) {
	fmt.Printf("DEBUG: ProcessDocument received filename: %s, mimeType: %s\n", filename, mimeType) // <-- TAMBAHKAN BARIS INI

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition",
		fmt.Sprintf(`form-data; name="%s"; filename="%s"`,
			"document",
			filename,
		))
	header.Set("Content-Type", mimeType) // <-- Ini yang akan kita verifikasi

	part, err := writer.CreatePart(header)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file part: %w", err)
	}

	_, err = io.Copy(part, file)
	if err != nil {
		return nil, fmt.Errorf("failed to copy file to form: %w", err)
	}

	writer.Close()

	req, err := http.NewRequestWithContext(context.Background(), "POST", s.AIBackendURL+"/process-document", body)
	if err != nil {
		return nil, fmt.Errorf("failed to create AI request: %w", err)
	}

	contentTypeForRequest := writer.FormDataContentType() // Dapatkan Content-Type untuk seluruh request
	req.Header.Set("Content-Type", contentTypeForRequest)
	fmt.Printf("DEBUG: Sending request with Content-Type header: %s\n", contentTypeForRequest) // <-- TAMBAHKAN BARIS INI

	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to AI service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("AI service returned non-OK status: %d, body: %s", resp.StatusCode, string(respBodyBytes))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode AI service response: %w", err)
	}

	return result, nil
}

// ValidateScholarshipData (kode ini tidak berubah)
func (s *AIService) ValidateScholarshipData(application *models.ScholarshipApplication, extractedData map[string]interface{}) (bool, string, error) {
	requestBody := map[string]interface{}{
		"application_data": application,
		"extracted_data":   extractedData,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return false, "", fmt.Errorf("failed to marshal validation request body: %w", err)
	}

	req, err := http.NewRequestWithContext(context.Background(), "POST", s.AIBackendURL+"/validate-scholarship", bytes.NewBuffer(jsonBody))
	if err != nil {
		return false, "", fmt.Errorf("failed to create AI validation request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.Client.Do(req)
	if err != nil {
		return false, "", fmt.Errorf("failed to send validation request to AI service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBodyBytes, _ := io.ReadAll(resp.Body)
		return false, "", fmt.Errorf("AI validation service returned non-OK status: %d, body: %s", resp.StatusCode, string(respBodyBytes))
	}

	var result struct {
		IsValid bool   `json:"is_valid"`
		Notes   string `json:"notes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, "", fmt.Errorf("failed to decode AI validation response: %w", err)
	}

	return result.IsValid, result.Notes, nil
}
