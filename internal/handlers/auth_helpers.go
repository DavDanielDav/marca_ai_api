package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/danpi/marca_ai_backend/internal/config"
	"github.com/danpi/marca_ai_backend/internal/middleware"
	"github.com/golang-jwt/jwt/v5"
)

const (
	accessTokenTTL  = 1 * time.Hour
	refreshTokenTTL = 72 * time.Hour
)

var errInvalidToken = errors.New("token_invalido")

func issueAuthToken(email string, userID int) (string, error) {
	return issueSignedToken(email, userID, middleware.AccessTokenType, accessTokenTTL)
}

func issueRefreshToken(email string, userID int) (string, error) {
	return issueSignedToken(email, userID, middleware.RefreshTokenType, refreshTokenTTL)
}

func issueTokenPair(email string, userID int) (string, string, error) {
	accessToken, err := issueAuthToken(email, userID)
	if err != nil {
		return "", "", err
	}

	refreshToken, err := issueRefreshToken(email, userID)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func issueSignedToken(email string, userID int, tokenType string, ttl time.Duration) (string, error) {
	jwtKey, err := config.JWTKey()
	if err != nil {
		return "", err
	}

	now := time.Now()
	claims := &middleware.Claims{
		Email:     email,
		IDUsuario: userID,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}

func parseSignedToken(tokenString string) (*middleware.Claims, error) {
	jwtKey, err := config.JWTKey()
	if err != nil {
		return nil, err
	}

	claims := &middleware.Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok || token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, jwt.ErrTokenSignatureInvalid
		}

		return jwtKey, nil
	})
	if err != nil || !token.Valid {
		return nil, errInvalidToken
	}

	return claims, nil
}

func writeAuthSuccess(w http.ResponseWriter, message string, userID int, token string, refreshToken string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"message":            message,
		"token":              token,
		"refresh_token":      refreshToken,
		"id_usuario":         userID,
		"token_type":         "Bearer",
		"expires_in":         int(accessTokenTTL.Seconds()),
		"refresh_expires_in": int(refreshTokenTTL.Seconds()),
	})
}
