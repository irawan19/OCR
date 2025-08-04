package vision

import (
	"context"
	"fmt"
	"os"

	vision "cloud.google.com/go/vision/apiv1"
	"google.golang.org/api/option"
)

func ExtractText(imagePath string) (string, error) {
	ctx := context.Background()

	client, err := vision.NewImageAnnotatorClient(ctx, option.WithCredentialsFile(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")))
	if err != nil {
		return "", fmt.Errorf("Gagal membuat client: %w", err)
	}
	defer client.Close()

	file, err := os.Open(imagePath)
	if err != nil {
		return "", fmt.Errorf("Gagal buka file: %w", err)
	}
	defer file.Close()

	image, err := vision.NewImageFromReader(file)
	if err != nil {
		return "", fmt.Errorf("Gagal buat image: %w", err)
	}

	annotations, err := client.DetectTexts(ctx, image, nil, 1)
	if err != nil {
		return "", fmt.Errorf("Gagal mendeteksi teks: %w", err)
	}

	if len(annotations) == 0 {
		return "", fmt.Errorf("Teks tidak ditemukan")
	}

	return annotations[0].Description, nil
}
