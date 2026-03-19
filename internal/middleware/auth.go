package middleware

import (
	"context"
	"log"
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
	ID    int    `json:"id_usuario"`
	Email string `json:"email"`

	jwt.RegisteredClaims
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Token de autorização não fornecido", http.StatusUnauthorized)
			log.Printf("Token nao fornecido")
			return
		}

		// Garante formato correto: "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			http.Error(w, "Formato do token inválido", http.StatusUnauthorized)
			log.Printf("Formato de Token Invalido")
			return
		}
		tokenString := parts[1]

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return config.JWTKey(), nil
		})
		if err != nil || !token.Valid {
			http.Error(w, "Token inválido", http.StatusUnauthorized)
			log.Printf("Token invalido")
			return
		}

		// Adiciona o ID ao contexto
		ctx := context.WithValue(r.Context(), UserIDKey, claims.ID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
