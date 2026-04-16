package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danpi/marca_ai_backend/internal/middleware"
)

func TestRefreshTokenHandlerReturnsNewTokenPair(t *testing.T) {
	t.Setenv("jwtKey", "test-secret")

	refreshToken, err := issueRefreshToken("refresh@test.com", 21)
	if err != nil {
		t.Fatalf("failed to issue refresh token: %v", err)
	}

	body, err := json.Marshal(map[string]string{
		"refresh_token": refreshToken,
	})
	if err != nil {
		t.Fatalf("failed to marshal request body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	RefreshTokenHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response struct {
		Token        string `json:"token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		IDUsuario    int    `json:"id_usuario"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Token == "" || response.RefreshToken == "" {
		t.Fatal("expected both access and refresh tokens in response")
	}
	if response.TokenType != "Bearer" {
		t.Fatalf("expected token_type Bearer, got %q", response.TokenType)
	}
	if response.IDUsuario != 21 {
		t.Fatalf("expected id_usuario 21, got %d", response.IDUsuario)
	}

	accessClaims, err := parseSignedToken(response.Token)
	if err != nil {
		t.Fatalf("failed to parse access token: %v", err)
	}
	if accessClaims.TokenType != middleware.AccessTokenType {
		t.Fatalf("expected access token type %q, got %q", middleware.AccessTokenType, accessClaims.TokenType)
	}

	refreshClaims, err := parseSignedToken(response.RefreshToken)
	if err != nil {
		t.Fatalf("failed to parse refresh token: %v", err)
	}
	if refreshClaims.TokenType != middleware.RefreshTokenType {
		t.Fatalf("expected refresh token type %q, got %q", middleware.RefreshTokenType, refreshClaims.TokenType)
	}
}
