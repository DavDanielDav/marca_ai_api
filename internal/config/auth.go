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
