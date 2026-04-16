package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/danpi/marca_ai_backend/internal/config"
	"github.com/danpi/marca_ai_backend/internal/middleware"
	"github.com/danpi/marca_ai_backend/internal/models"
	"github.com/danpi/marca_ai_backend/internal/utils"
)

const (
	codePurposeSignup        = "signup"
	codePurposePasswordReset = "password_reset"
	codeTTL                  = 10 * time.Minute
)

type startSignupRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Senha    string `json:"senha"`
}

type emailCodeRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

type resetPasswordRequest struct {
	Email     string `json:"email"`
	Code      string `json:"code"`
	NovaSenha string `json:"novaSenha"`
}

type signupPayload struct {
	Username     string `json:"username"`
	PasswordHash string `json:"password_hash"`
}

func StartSignupVerification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req startSignupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(req.Email)

	if req.Username == "" {
		http.Error(w, "Requer Nome", http.StatusBadRequest)
		return
	}
	if err := validarEmail(req.Email); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := validarSenha(req.Senha); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	release, err := middleware.AcquireEmailRequestSlot(r.Context(), req.Email)
	if err != nil {
		return
	}
	defer release()

	exists, err := userEmailExists(req.Email)
	if err != nil {
		http.Error(w, "Erro ao verificar email", http.StatusInternalServerError)
		return
	}
	if exists {
		http.Error(w, "Ja existe uma conta cadastrada com este e-mail", http.StatusConflict)
		return
	}

	passwordHash, err := utils.HashSenha(req.Senha)
	if err != nil {
		http.Error(w, "Erro ao processar senha", http.StatusInternalServerError)
		return
	}

	payload, err := json.Marshal(signupPayload{
		Username:     req.Username,
		PasswordHash: passwordHash,
	})
	if err != nil {
		http.Error(w, "Erro ao preparar cadastro", http.StatusInternalServerError)
		return
	}

	code, err := generateNumericCode(6)
	if err != nil {
		http.Error(w, "Erro ao gerar codigo", http.StatusInternalServerError)
		return
	}

	if err := upsertEmailCode(req.Email, codePurposeSignup, code, payload); err != nil {
		http.Error(w, "Erro ao salvar codigo", http.StatusInternalServerError)
		return
	}

	if err := utils.SendEmail(req.Email, "Codigo de confirmacao do cadastro", buildSignupEmailBody(code)); err != nil {
		log.Printf("erro ao enviar email de cadastro: %v", err)
		http.Error(w, "Nao foi possivel enviar o codigo por email", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Codigo de confirmacao enviado para o seu e-mail",
		"email":   req.Email,
	})
}

func ConfirmSignupCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req emailCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	req.Code = strings.TrimSpace(req.Code)

	release, err := middleware.AcquireEmailRequestSlot(r.Context(), req.Email)
	if err != nil {
		return
	}
	defer release()

	record, err := validateEmailCode(req.Email, codePurposeSignup, req.Code)
	if err != nil {
		writeCodeError(w, err)
		return
	}

	var payload signupPayload
	if err := json.Unmarshal(record.Payload, &payload); err != nil {
		http.Error(w, "Dados de cadastro invalidos", http.StatusInternalServerError)
		return
	}

	exists, err := userEmailExists(req.Email)
	if err != nil {
		http.Error(w, "Erro ao verificar email", http.StatusInternalServerError)
		return
	}
	if exists {
		_ = deleteEmailCode(req.Email, codePurposeSignup)
		http.Error(w, "Ja existe uma conta cadastrada com este e-mail", http.StatusConflict)
		return
	}

	_, err = config.DB.Exec(
		"INSERT INTO usuario (nome, email, senha) VALUES ($1, $2, $3)",
		payload.Username, req.Email, payload.PasswordHash,
	)
	if err != nil {
		log.Printf("erro ao efetivar cadastro: %v", err)
		http.Error(w, "Erro ao concluir cadastro", http.StatusInternalServerError)
		return
	}

	_ = deleteEmailCode(req.Email, codePurposeSignup)

	writeJSON(w, http.StatusCreated, map[string]string{
		"message": "Cadastro confirmado com sucesso",
	})
}

func ResendSignupCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req emailCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	if err := validarEmail(req.Email); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	release, err := middleware.AcquireEmailRequestSlot(r.Context(), req.Email)
	if err != nil {
		return
	}
	defer release()

	record, err := getEmailCode(req.Email, codePurposeSignup)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Nenhum cadastro pendente encontrado para este e-mail", http.StatusNotFound)
			return
		}
		http.Error(w, "Erro ao buscar codigo", http.StatusInternalServerError)
		return
	}

	code, err := generateNumericCode(6)
	if err != nil {
		http.Error(w, "Erro ao gerar codigo", http.StatusInternalServerError)
		return
	}

	if err := upsertEmailCode(req.Email, codePurposeSignup, code, record.Payload); err != nil {
		http.Error(w, "Erro ao salvar codigo", http.StatusInternalServerError)
		return
	}

	if err := utils.SendEmail(req.Email, "Novo codigo de confirmacao do cadastro", buildSignupEmailBody(code)); err != nil {
		log.Printf("erro ao reenviar email de cadastro: %v", err)
		http.Error(w, "Nao foi possivel reenviar o codigo", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Novo codigo enviado para o seu e-mail",
	})
}

func SendForgotPasswordCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req emailCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	if err := validarEmail(req.Email); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	release, err := middleware.AcquireEmailRequestSlot(r.Context(), req.Email)
	if err != nil {
		return
	}
	defer release()

	exists, err := userEmailExists(req.Email)
	if err != nil {
		http.Error(w, "Erro ao verificar email", http.StatusInternalServerError)
		return
	}
	if !exists {
		writeJSON(w, http.StatusOK, map[string]string{
			"message": "Se o e-mail existir, enviaremos um codigo de recuperacao",
		})
		return
	}

	code, err := generateNumericCode(6)
	if err != nil {
		http.Error(w, "Erro ao gerar codigo", http.StatusInternalServerError)
		return
	}

	if err := upsertEmailCode(req.Email, codePurposePasswordReset, code, nil); err != nil {
		http.Error(w, "Erro ao salvar codigo", http.StatusInternalServerError)
		return
	}

	if err := utils.SendEmail(req.Email, "Codigo para redefinir sua senha", buildResetEmailBody(code)); err != nil {
		log.Printf("erro ao enviar email de recuperacao: %v", err)
		http.Error(w, "Nao foi possivel enviar o codigo por email", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Se o e-mail existir, enviaremos um codigo de recuperacao",
	})
}

func VerifyForgotPasswordCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req emailCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	req.Code = strings.TrimSpace(req.Code)

	release, err := middleware.AcquireEmailRequestSlot(r.Context(), req.Email)
	if err != nil {
		return
	}
	defer release()

	if _, err := validateEmailCode(req.Email, codePurposePasswordReset, req.Code); err != nil {
		writeCodeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Codigo validado com sucesso",
	})
}

func ResetForgotPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req resetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	req.Code = strings.TrimSpace(req.Code)

	if err := validarEmail(req.Email); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := validarSenha(req.NovaSenha); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	release, err := middleware.AcquireEmailRequestSlot(r.Context(), req.Email)
	if err != nil {
		return
	}
	defer release()

	if _, err := validateEmailCode(req.Email, codePurposePasswordReset, req.Code); err != nil {
		writeCodeError(w, err)
		return
	}

	passwordHash, err := utils.HashSenha(req.NovaSenha)
	if err != nil {
		http.Error(w, "Erro ao processar senha", http.StatusInternalServerError)
		return
	}

	result, err := config.DB.Exec(
		"UPDATE usuario SET senha = $1 WHERE email = $2",
		passwordHash, req.Email,
	)
	if err != nil {
		http.Error(w, "Erro ao redefinir senha", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "Erro ao redefinir senha", http.StatusInternalServerError)
		return
	}
	if rowsAffected == 0 {
		http.Error(w, "Nao foi possivel redefinir a senha", http.StatusBadRequest)
		return
	}

	_ = deleteEmailCode(req.Email, codePurposePasswordReset)

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Senha redefinida com sucesso",
	})
}

func getEmailCode(email, purpose string) (models.EmailCode, error) {
	var record models.EmailCode
	err := config.DB.QueryRow(`
		SELECT id, email, purpose, code_hash, payload, expires_at, created_at
		FROM email_codes
		WHERE email = $1 AND purpose = $2
	`, email, purpose).Scan(
		&record.ID,
		&record.Email,
		&record.Purpose,
		&record.CodeHash,
		&record.Payload,
		&record.ExpiresAt,
		&record.CreatedAt,
	)

	return record, err
}

func validateEmailCode(email, purpose, code string) (models.EmailCode, error) {
	record, err := getEmailCode(email, purpose)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.EmailCode{}, fmt.Errorf("codigo_nao_encontrado")
		}
		return models.EmailCode{}, err
	}

	if time.Now().After(record.ExpiresAt) {
		_ = deleteEmailCode(email, purpose)
		return models.EmailCode{}, fmt.Errorf("codigo_expirado")
	}

	if record.CodeHash != hashVerificationCode(email, purpose, code) {
		return models.EmailCode{}, fmt.Errorf("codigo_invalido")
	}

	return record, nil
}

func upsertEmailCode(email, purpose, code string, payload []byte) error {
	_, err := config.DB.Exec(`
		INSERT INTO email_codes (email, purpose, code_hash, payload, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (email, purpose)
		DO UPDATE SET
			code_hash = EXCLUDED.code_hash,
			payload = EXCLUDED.payload,
			expires_at = EXCLUDED.expires_at,
			created_at = NOW()
	`,
		email,
		purpose,
		hashVerificationCode(email, purpose, code),
		payload,
		time.Now().Add(codeTTL),
	)

	return err
}

func deleteEmailCode(email, purpose string) error {
	_, err := config.DB.Exec(`
		DELETE FROM email_codes
		WHERE email = $1 AND purpose = $2
	`, email, purpose)
	return err
}

func userEmailExists(email string) (bool, error) {
	var exists bool
	err := config.DB.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM usuario WHERE email = $1)
	`, email).Scan(&exists)

	return exists, err
}

func generateNumericCode(length int) (string, error) {
	const digits = "0123456789"

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	code := make([]byte, length)
	for i, b := range bytes {
		code[i] = digits[int(b)%len(digits)]
	}

	return string(code), nil
}

func hashVerificationCode(email, purpose, code string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(email)) + "|" + purpose + "|" + code))
	return hex.EncodeToString(sum[:])
}

func buildSignupEmailBody(code string) string {
	return fmt.Sprintf(
		"Seu codigo de confirmacao de cadastro e: %s\n\nEsse codigo expira em 10 minutos.",
		code,
	)
}

func buildResetEmailBody(code string) string {
	return fmt.Sprintf(
		"Seu codigo para redefinir a senha e: %s\n\nEsse codigo expira em 10 minutos.",
		code,
	)
}

func writeCodeError(w http.ResponseWriter, err error) {
	switch err.Error() {
	case "codigo_nao_encontrado", "codigo_invalido":
		http.Error(w, "Codigo invalido", http.StatusBadRequest)
	case "codigo_expirado":
		http.Error(w, "Codigo expirado. Solicite um novo codigo", http.StatusBadRequest)
	default:
		http.Error(w, "Erro ao validar codigo", http.StatusInternalServerError)
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
