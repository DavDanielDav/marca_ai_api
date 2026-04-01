package utils

import (
	"context"
	"fmt"
	"log"
	"mime/multipart"
	"os"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

func UploadCloudinary(file multipart.File, filename string) (string, error) {
	cloudName := os.Getenv("CLOUDINARY_CLOUD_NAME")
	apiKey := os.Getenv("CLOUDINARY_API_KEY")
	apiSecret := os.Getenv("CLOUDINARY_API_SECRET")

	if cloudName == "" || apiKey == "" || apiSecret == "" {
		err := fmt.Errorf("cloudinary nao configurado: defina CLOUDINARY_CLOUD_NAME, CLOUDINARY_API_KEY e CLOUDINARY_API_SECRET")
		log.Print(err)
		return "", err
	}

	cld, err := cloudinary.NewFromParams(
		cloudName,
		apiKey,
		apiSecret,
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
