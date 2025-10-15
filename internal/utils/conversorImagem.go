package utils

import (
	"context"
	"fmt"
	"mime/multipart"
	"os"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

func UploadCloudinary(file multipart.File, filename string) (string, error) {
	cld, err := cloudinary.NewFromParams(
		os.Getenv("CLOUD_NAME"),
		os.Getenv("API_KEY"),
		os.Getenv("API_SECRET"),
	)
	if err != nil {
		return "", fmt.Errorf("erro ao conectar ao Cloudinary: %v", err)
	}

	uploadResult, err := cld.Upload.Upload(context.Background(), file, uploader.UploadParams{
		PublicID: filename,
		Folder:   "arenas",
	})
	if err != nil {
		return "", fmt.Errorf("erro ao enviar imagem: %v", err)
	}

	return uploadResult.SecureURL, nil
}
