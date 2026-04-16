package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestAuthMiddlewareRejectsRefreshToken(t *testing.T) {
	t.Setenv("jwtKey", "test-secret")

	tokenString := signTestToken(t, &Claims{
		IDUsuario: 7,
		Email:     "user@test.com",
		TokenType: RefreshTokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/Usuario", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	rec := httptest.NewRecorder()
	called := false

	AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rec, req)

	if called {
		t.Fatal("refresh token should not pass through auth middleware")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestAuthMiddlewareAcceptsLegacyAccessTokenWithoutType(t *testing.T) {
	t.Setenv("jwtKey", "test-secret")

	tokenString := signTestToken(t, &Claims{
		IDUsuario: 11,
		Email:     "legacy@test.com",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/Usuario", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	rec := httptest.NewRecorder()
	called := false

	AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rec, req)

	if !called {
		t.Fatal("legacy access token should remain valid")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func signTestToken(t *testing.T, claims *Claims) string {
	t.Helper()

	jwtKey := os.Getenv("jwtKey")
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(jwtKey))
	if err != nil {
		t.Fatalf("failed to sign test token: %v", err)
	}

	return tokenString
}
