package config

import (
	"log"
	"os"
)

func JWTKey() []byte {
	secret := os.Getenv("jwtKey")
	if secret == "" {
		log.Fatal("JWT_SECRET nao configurado")
	}

	return []byte(secret)
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
