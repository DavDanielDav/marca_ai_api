package utils

import (
	"context"
	"log"
	"mime/multipart"
	"os"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

func UploadCloudinary(file multipart.File, filename string) (string, error) {
	cld, err := cloudinary.NewFromParams(
		os.Getenv("CLOUDINARY_CLOUD_NAME"),
		os.Getenv("CLOUDINARY_API_KEY"),
		os.Getenv("CLOUDINARY_API_SECRET"),
	)
	if err != nil {
		log.Printf("erro ao conectar ao Cloudinary: %v", err)
		return "", err
	}

	uploadResult, err := cld.Upload.Upload(context.Background(), file, uploader.UploadParams{
		PublicID: filename,
		Folder:   "arenas",
	})
	if err != nil {
		log.Printf("erro ao enviar imagem cloudinary: %v", err)
		return "", err
	}

	return uploadResult.SecureURL, nil
}
