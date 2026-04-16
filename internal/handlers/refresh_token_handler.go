package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/danpi/marca_ai_backend/internal/middleware"
)

type refreshTokenRequest struct {
	RefreshToken    string `json:"refresh_token"`
	RefreshTokenAlt string `json:"refreshToken"`
}

func RefreshTokenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req refreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	tokenString := strings.TrimSpace(firstNonEmpty(req.RefreshToken, req.RefreshTokenAlt))
	if tokenString == "" {
		http.Error(w, "Refresh token is required", http.StatusBadRequest)
		return
	}

	claims, err := parseSignedToken(tokenString)
	if err != nil {
		if errors.Is(err, errInvalidToken) {
			http.Error(w, "Refresh token invalido", http.StatusUnauthorized)
			return
		}

		http.Error(w, "Erro ao validar refresh token", http.StatusInternalServerError)
		return
	}
	if claims.TokenType != middleware.RefreshTokenType {
		http.Error(w, "Refresh token invalido", http.StatusUnauthorized)
		return
	}

	release, err := middleware.AcquireEmailRequestSlot(r.Context(), claims.Email)
	if err != nil {
		return
	}
	defer release()

	accessToken, refreshToken, err := issueTokenPair(claims.Email, claims.IDUsuario)
	if err != nil {
		http.Error(w, "Erro ao renovar token", http.StatusInternalServerError)
		return
	}

	writeAuthSuccess(w, "Token renovado com sucesso", claims.IDUsuario, accessToken, refreshToken)
}
