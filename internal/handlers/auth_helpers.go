package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/danpi/marca_ai_backend/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

func issueAuthToken(email string, userID int) (string, error) {
	jwtKey, err := config.JWTKey()
	if err != nil {
		return "", err
	}

	expirationTime := time.Now().Add(1 * time.Hour)
	claims := &Claims{
		Email:     email,
		IDUsuario: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}

func writeAuthSuccess(w http.ResponseWriter, message string, userID int, token string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"message":    message,
		"token":      token,
		"id_usuario": userID,
	})
}
