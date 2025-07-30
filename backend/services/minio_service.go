package services

import (
	"context"
	"fmt"
	"io" // Pastikan ini diimport
	"log"
	"path/filepath" // Untuk mendapatkan ekstensi file
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinioService provides methods for interacting with MinIO object storage.
type MinioService struct {
	Client     *minio.Client
	BucketName string
}

// NewMinioService creates a new MinioService instance and initializes its bucket name.
func NewMinioService(endpoint, accessKeyID, secretAccessKey string, useSSL bool, bucketName string) *MinioService {
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Fatalf("Failed to initialize MinIO client: %v", err)
	}
	return &MinioService{
		Client:     minioClient,
		BucketName: bucketName,
	}
}

// CreateBucket ensures that the specified MinIO bucket exists.
func (s *MinioService) CreateBucket(ctx context.Context) error {
	exists, err := s.Client.BucketExists(ctx, s.BucketName)
	if err != nil {
		return fmt.Errorf("failed to check if bucket %s exists: %w", s.BucketName, err)
	}
	if !exists {
		log.Printf("Bucket %s does not exist, creating...", s.BucketName)
		err = s.Client.MakeBucket(ctx, s.BucketName, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("failed to create bucket %s: %w", s.BucketName, err)
		}
		log.Printf("Bucket %s created successfully.", s.BucketName)
	} else {
		log.Printf("Bucket %s already exists.", s.BucketName)
	}
	return nil
}

// UploadFile uploads a file to the MinIO bucket.
// It takes an io.Reader (the file content), original filename, file size, and content type.
// It returns the object name (path in MinIO) or an error.
func (s *MinioService) UploadFile(ctx context.Context, fileReader io.Reader, originalFilename string, fileSize int64, contentType string) (string, error) {
	// Generate a unique object name to prevent collisions
	// Contoh: userID/timestamp_originalfilename.ext
	// Untuk saat ini, kita akan membuat nama unik sederhana.
	fileExtension := filepath.Ext(originalFilename)
	objectName := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), originalFilename[:len(originalFilename)-len(fileExtension)], fileExtension)

	info, err := s.Client.PutObject(ctx, s.BucketName, objectName, fileReader, fileSize, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload object %s to bucket %s: %w", objectName, s.BucketName, err)
	}
	log.Printf("Successfully uploaded %s of size %d to bucket %s", objectName, info.Size, s.BucketName)
	return objectName, nil // Mengembalikan nama objek (minioPath)
}

// GetFile retrieves a file from the MinIO bucket.
// Returns an io.ReadCloser (the object content) and an error.
func (s *MinioService) GetFile(ctx context.Context, objectName string) (io.ReadCloser, error) {
	object, err := s.Client.GetObject(ctx, s.BucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object %s from bucket %s: %w", objectName, s.BucketName, err)
	}
	// GetObject returns *minio.Object which implements io.ReadCloser
	return object, nil
}

// DeleteFile deletes a file from the MinIO bucket.
func (s *MinioService) DeleteFile(ctx context.Context, objectName string) error {
	err := s.Client.RemoveObject(ctx, s.BucketName, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete object %s from bucket %s: %w", objectName, s.BucketName, err)
	}
	log.Printf("Successfully deleted object %s from bucket %s", objectName, s.BucketName)
	return nil
}

// GetObjectPresignedURL generates a pre-signed URL for an object.
func (s *MinioService) GetObjectPresignedURL(ctx context.Context, objectName string, expiry time.Duration) (string, error) {
	url, err := s.Client.PresignedGetObject(ctx, s.BucketName, objectName, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL for object %s: %w", objectName, err)
	}
	return url.String(), nil
}
