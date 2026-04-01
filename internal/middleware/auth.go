package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/danpi/marca_ai_backend/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

// Context key segura
type contextKey string

const UserIDKey contextKey = "userID"

// Claims personalizados
type Claims struct {
	UsuarioID int    `json:"usuario_id"`
	Email     string `json:"email,omitempty"`

	jwt.RegisteredClaims
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Token de autorização não fornecido", http.StatusUnauthorized)
			return
		}

		// Garante formato correto: "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			http.Error(w, "Formato do token inválido", http.StatusUnauthorized)
			return
		}
		tokenString := parts[1]

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return config.JWTKey(), nil
		})
		if err != nil || !token.Valid {
			http.Error(w, "Token inválido", http.StatusUnauthorized)
			return
		}

		// Adiciona o ID ao contexto
		ctx := context.WithValue(r.Context(), UserIDKey, claims.UsuarioID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
