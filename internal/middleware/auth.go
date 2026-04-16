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

const (
	AccessTokenType  = "access"
	RefreshTokenType = "refresh"
)

// Claims personalizados
type Claims struct {
	IDUsuario int    `json:"id_usuario"`
	Email     string `json:"email,omitempty"`
	TokenType string `json:"token_type,omitempty"`

	jwt.RegisteredClaims
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Token de autorizacao nao fornecido", http.StatusUnauthorized)
			return
		}

		// Garante formato correto: "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			http.Error(w, "Formato do token invalido", http.StatusUnauthorized)
			return
		}
		tokenString := parts[1]

		jwtKey, err := config.JWTKey()
		if err != nil {
			log.Printf("erro de configuracao de autenticacao: %v", err)
			http.Error(w, "Configuracao de autenticacao ausente", http.StatusInternalServerError)
			return
		}

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok || token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, jwt.ErrTokenSignatureInvalid
			}

			return jwtKey, nil
		})
		if err != nil || !token.Valid {
			http.Error(w, "Token invalido", http.StatusUnauthorized)
			return
		}
		if claims.TokenType != "" && claims.TokenType != AccessTokenType {
			http.Error(w, "Token invalido", http.StatusUnauthorized)
			return
		}

		// Adiciona o ID e email ao contexto
		ctx := context.WithValue(r.Context(), UserIDKey, claims.IDUsuario)
		ctx = context.WithValue(ctx, UserEmailKey, strings.ToLower(strings.TrimSpace(claims.Email)))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
