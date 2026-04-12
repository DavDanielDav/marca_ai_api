package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

func requiredEnv(key string) (string, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return "", fmt.Errorf("%s nao configurado", key)
	}

	return value, nil
}

func JWTKey() ([]byte, error) {
	secret, err := requiredEnv("jwtKey")
	if err != nil {
		return nil, err
	}

	return []byte(secret), nil
}

func GoogleClientID() (string, error) {
	return requiredEnv("GOOGLE_CLIENT_ID")
}

func ResendKey() (string, error) {
	return requiredEnv("RESEND_API_KEY")
}

func ResendFromEmail() string {
	from := os.Getenv("RESEND_FROM_EMAIL")
	if from == "" {
		return "marcaaisend@marcaai.tec.br"
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
