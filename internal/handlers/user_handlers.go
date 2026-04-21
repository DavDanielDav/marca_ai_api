package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/danpi/marca_ai_backend/internal/config"
	"github.com/danpi/marca_ai_backend/internal/middleware"
	"github.com/danpi/marca_ai_backend/internal/models"
	"github.com/danpi/marca_ai_backend/internal/utils"
)

type updateUsuarioRequest struct {
	Username string
	Email    string
	Telefone string
	Senha    string
}

var (
	uppercaseRegex = regexp.MustCompile(`[A-Z]`)
	numberRegex    = regexp.MustCompile(`[0-9]`)
	emailRegex     = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
)

func validarSenha(senha string) error {
	switch {
	case len(senha) < 8:
		return fmt.Errorf("a senha deve ter pelo menos 8 caracteres")
	case !uppercaseRegex.MatchString(senha):
		return fmt.Errorf("a senha deve conter pelo menos uma letra maiuscula")
	case !numberRegex.MatchString(senha):
		return fmt.Errorf("a senha deve conter pelo menos um numero")
	default:
		return nil
	}
}

func validarEmail(email string) error {
	if !emailRegex.MatchString(strings.TrimSpace(email)) {
		return fmt.Errorf("email invalido")
	}

	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}

	return ""
}

func stringFromJSONValue(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(v)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func parseUpdateUsuarioRequest(r *http.Request) (updateUsuarioRequest, error) {
	var payload map[string]any
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return updateUsuarioRequest{}, err
	}

	return updateUsuarioRequest{
		Username: firstNonEmpty(
			stringFromJSONValue(payload["username"]),
			stringFromJSONValue(payload["nome"]),
			stringFromJSONValue(payload["name"]),
		),
		Email: firstNonEmpty(
			stringFromJSONValue(payload["email"]),
		),
		Telefone: firstNonEmpty(
			stringFromJSONValue(payload["telefone"]),
			stringFromJSONValue(payload["phone"]),
		),
		Senha: firstNonEmpty(
			stringFromJSONValue(payload["senha"]),
			stringFromJSONValue(payload["novaSenha"]),
			stringFromJSONValue(payload["password"]),
			stringFromJSONValue(payload["newPassword"]),
		),
	}, nil
}

func RegisterUsuarioHandler(w http.ResponseWriter, r *http.Request) {
	StartSignupVerification(w, r)
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var creds models.Usuario
	err := json.NewDecoder(r.Body).Decode(&creds)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(creds.Email) == "" || strings.TrimSpace(creds.Senha) == "" {
		http.Error(w, "Email and password are required", http.StatusBadRequest)
		return
	}

	creds.Email = strings.TrimSpace(creds.Email)

	release, err := middleware.AcquireEmailRequestSlot(r.Context(), creds.Email)
	if err != nil {
		return
	}
	defer release()

	log.Printf("Login attempt for email: %s", creds.Email)

	var userID int
	var userEmail, hashedPassword string
	err = config.DB.QueryRow(
		fmt.Sprintf("SELECT id_usuario, email, senha FROM %s WHERE email = $1", usuarioTableName()),
		creds.Email,
	).Scan(&userID, &userEmail, &hashedPassword)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		log.Printf("Error querying database: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if !utils.CheckSenhaHash(creds.Senha, hashedPassword) {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	tokenString, refreshToken, err := issueTokenPair(userEmail, userID)
	if err != nil {
		http.Error(w, "Erro ao gerar token", http.StatusInternalServerError)
		return
	}

	writeAuthSuccess(w, "Logado com sucesso!!", userID, tokenString, refreshToken)
}

func GetUserHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "ID do usuario nao encontrado no token", http.StatusInternalServerError)
		return
	}

	var user models.Usuario
	var username, email, telefone sql.NullString
	err := config.DB.QueryRow(fmt.Sprintf(
		`SELECT id_usuario::text, nome, email, telefone
		FROM %s
		WHERE id_usuario = $1`,
		usuarioTableName(),
	), userID).Scan(&user.ID, &username, &email, &telefone)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Usuario nao encontrado", http.StatusNotFound)
			return
		}

		log.Printf("Erro ao buscar usuario %d: %v", userID, err)
		http.Error(w, "Erro ao buscar usuario", http.StatusInternalServerError)
		return
	}

	user.Username = username.String
	user.Email = email.String
	user.Telefone = telefone.String

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func UpdateUsuarioHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Nao foi possivel obter o ID do usuario do token", http.StatusInternalServerError)
		return
	}

	req, err := parseUpdateUsuarioRequest(r)
	if err != nil {
		http.Error(w, "Corpo da requisicao invalido", http.StatusBadRequest)
		return
	}

	if req.Username == "" {
		http.Error(w, "Requer Nome", http.StatusBadRequest)
		return
	}
	if req.Email == "" {
		http.Error(w, "Requer Email", http.StatusBadRequest)
		return
	}
	if err := validarEmail(req.Email); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Senha != "" {
		if err := validarSenha(req.Senha); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		hashedPassword, err := utils.HashSenha(req.Senha)
		if err != nil {
			http.Error(w, "Erro ao processar senha", http.StatusInternalServerError)
			return
		}
		_, err = config.DB.Exec(
			fmt.Sprintf("UPDATE %s SET nome=$1, email=$2, telefone=$3, senha=$4 WHERE id_usuario=$5", usuarioTableName()),
			req.Username, req.Email, req.Telefone, hashedPassword, userID,
		)
	} else {
		_, err = config.DB.Exec(
			fmt.Sprintf("UPDATE %s SET nome=$1, email=$2, telefone=$3 WHERE id_usuario=$4", usuarioTableName()),
			req.Username, req.Email, req.Telefone, userID,
		)
	}

	if err != nil {
		log.Printf("Erro ao atualizar usuario no banco: %v", err)
		http.Error(w, "Erro ao atualizar usuario", http.StatusInternalServerError)
		return
	}

	log.Printf("Usuario com ID %d atualizado com sucesso!", userID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message":    "Usuario atualizado com sucesso",
		"id_usuario": strconv.Itoa(userID),
		"username":   req.Username,
		"nome":       req.Username,
		"email":      req.Email,
		"telefone":   req.Telefone,
	})
}

func DeleteUsuarioHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Nao foi possivel obter o ID do usuario do token", http.StatusInternalServerError)
		return
	}

	_, err := config.DB.Exec(fmt.Sprintf("DELETE FROM %s WHERE id_usuario=$1", usuarioTableName()), userID)
	if err != nil {
		log.Printf("Erro ao deletar usuario do banco: %v", err)
		http.Error(w, "Erro ao deletar usuario", http.StatusInternalServerError)
		return
	}

	log.Printf("Usuario com ID %d deletado com sucesso!", userID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Usuario deletado com sucesso"})
}
