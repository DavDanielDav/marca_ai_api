package config

import (
	"log"
	"os"
	"strconv"
)

func JWTKey() []byte {
	secret := os.Getenv("jwtKey")
	if secret == "" {
		log.Fatal("JWT_SECRET nao configurado")
	}

	return []byte(secret)
}

func GoogleClientID() string {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	if clientID == "" {
		log.Fatal("GOOGLE_CLIENT_ID nao configurado")
	}

	return clientID
}

func ResendKey() string {
	key := os.Getenv("RESEND_API_KEY")
	if key == "" {
		log.Fatal("RESEND_API_KEY not configured")
	}
	return key
}

func ResendFromEmail() string {
	from := os.Getenv("RESEND_FROM_EMAIL")
	if from == "" {
		return "onboarding@resend.dev"
	}
	return from
}

func SMTPHost() string {
	return os.Getenv("SMTP_HOST")
}

func SMTPPort() int {
	port := os.Getenv("SMTP_PORT")
	if port == "" {
		return 587
	}

	parsed, err := strconv.Atoi(port)
	if err != nil {
		return 587
	}

	return parsed
}

func SMTPUser() string {
	return os.Getenv("SMTP_USER")
}

func SMTPPass() string {
	return os.Getenv("SMTP_PASS")
}

func SMTPFrom() string {
	return os.Getenv("SMTP_FROM")
}
