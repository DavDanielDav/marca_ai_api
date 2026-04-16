package handlers

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/danpi/marca_ai_backend/internal/config"
	"github.com/danpi/marca_ai_backend/internal/middleware"
	"github.com/danpi/marca_ai_backend/internal/utils"
	"github.com/golang-jwt/jwt/v5"
)

const googleCertsURL = "https://www.googleapis.com/oauth2/v1/certs"

type googleAuthRequest struct {
	Credential string `json:"credential"`
}

type googleProfile struct {
	Email string
	Name  string
}

type googleCertsResponse map[string]string

func GoogleAuthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req googleAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Credential) == "" {
		http.Error(w, "Credential is required", http.StatusBadRequest)
		return
	}

	googleClientID, err := config.GoogleClientID()
	if err != nil {
		log.Printf("google auth unavailable: %v", err)
		http.Error(w, "Login com Google nao configurado", http.StatusServiceUnavailable)
		return
	}

	profile, err := verifyGoogleCredential(r.Context(), req.Credential, googleClientID)
	if err != nil {
		log.Printf("google auth verification failed: %v", err)
		http.Error(w, "Google credential invalida", http.StatusUnauthorized)
		return
	}

	release, err := middleware.AcquireEmailRequestSlot(r.Context(), profile.Email)
	if err != nil {
		return
	}
	defer release()

	userID, err := findUserIDByEmail(profile.Email)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("google auth database lookup failed: %v", err)
		http.Error(w, "Erro ao autenticar com Google", http.StatusInternalServerError)
		return
	}

	if err == sql.ErrNoRows {
		userID, err = createGoogleUser(profile)
		if err != nil {
			log.Printf("google auth user creation failed: %v", err)
			http.Error(w, "Erro ao criar usuario com Google", http.StatusInternalServerError)
			return
		}

		_ = deleteEmailCode(profile.Email, codePurposeSignup)
	}

	token, refreshToken, err := issueTokenPair(profile.Email, userID)
	if err != nil {
		http.Error(w, "Erro ao gerar token", http.StatusInternalServerError)
		return
	}

	writeAuthSuccess(w, "Autenticado com Google com sucesso!", userID, token, refreshToken)
}

func verifyGoogleCredential(parent context.Context, credential string, googleClientID string) (googleProfile, error) {
	kid, err := extractGoogleKeyID(credential)
	if err != nil {
		return googleProfile{}, err
	}

	publicKey, err := fetchGooglePublicKey(parent, kid)
	if err != nil {
		return googleProfile{}, err
	}

	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(credential, claims, func(token *jwt.Token) (any, error) {
		signingMethod, ok := token.Method.(*jwt.SigningMethodRSA)
		if !ok || signingMethod.Alg() != jwt.SigningMethodRS256.Alg() {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return publicKey, nil
	})
	if err != nil || !token.Valid {
		return googleProfile{}, fmt.Errorf("invalid google token: %w", err)
	}

	if !isAllowedGoogleIssuer(fmt.Sprint(claims["iss"])) {
		return googleProfile{}, errors.New("invalid google issuer")
	}

	if fmt.Sprint(claims["aud"]) != googleClientID {
		return googleProfile{}, errors.New("invalid google audience")
	}

	if !googleEmailVerified(claims["email_verified"]) {
		return googleProfile{}, errors.New("google email not verified")
	}

	email := strings.ToLower(strings.TrimSpace(fmt.Sprint(claims["email"])))
	if email == "" {
		return googleProfile{}, errors.New("google email missing")
	}

	name := strings.TrimSpace(fmt.Sprint(claims["name"]))
	if name == "" {
		name = strings.TrimSpace(strings.Split(email, "@")[0])
	}

	return googleProfile{
		Email: email,
		Name:  name,
	}, nil
}

func extractGoogleKeyID(credential string) (string, error) {
	parser := jwt.Parser{}
	token, _, err := parser.ParseUnverified(credential, jwt.MapClaims{})
	if err != nil {
		return "", err
	}

	kid := strings.TrimSpace(fmt.Sprint(token.Header["kid"]))
	if kid == "" {
		return "", errors.New("google key id missing")
	}

	return kid, nil
}

func fetchGooglePublicKey(parent context.Context, kid string) (*rsa.PublicKey, error) {
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, googleCertsURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google certs returned status %d", resp.StatusCode)
	}

	var certs googleCertsResponse
	if err := json.NewDecoder(resp.Body).Decode(&certs); err != nil {
		return nil, err
	}

	certPEM, ok := certs[kid]
	if !ok {
		return nil, errors.New("google signing certificate not found")
	}

	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		return nil, errors.New("failed to decode google certificate")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}

	publicKey, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("google certificate does not contain rsa public key")
	}

	return publicKey, nil
}

func isAllowedGoogleIssuer(issuer string) bool {
	switch strings.TrimSpace(issuer) {
	case "accounts.google.com", "https://accounts.google.com":
		return true
	default:
		return false
	}
}

func googleEmailVerified(value any) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return strings.EqualFold(strings.TrimSpace(v), "true")
	default:
		return false
	}
}

func findUserIDByEmail(email string) (int, error) {
	var userID int
	err := config.DB.QueryRow(
		"SELECT id_usuario FROM usuario WHERE LOWER(email) = LOWER($1)",
		email,
	).Scan(&userID)

	return userID, err
}

func createGoogleUser(profile googleProfile) (int, error) {
	passwordHash, err := utils.HashSenha(randomGooglePassword())
	if err != nil {
		return 0, err
	}

	var userID int
	err = config.DB.QueryRow(
		"INSERT INTO usuario (nome, email, senha) VALUES ($1, $2, $3) RETURNING id_usuario",
		profile.Name,
		profile.Email,
		passwordHash,
	).Scan(&userID)

	return userID, err
}

func randomGooglePassword() string {
	buffer := make([]byte, 24)
	if _, err := rand.Read(buffer); err != nil {
		return fmt.Sprintf("GoogleTemp1_%d", time.Now().UnixNano())
	}

	return "GoogleTemp1_" + hex.EncodeToString(buffer)
}
