package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/danpi/marca_ai_backend/internal/config"
	"github.com/danpi/marca_ai_backend/internal/middleware"
	"github.com/danpi/marca_ai_backend/internal/models"
	"github.com/danpi/marca_ai_backend/internal/utils"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UsuarioID int    `json:"usuario_id"`
	Email     string `json:"email"`
	jwt.RegisteredClaims
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

	log.Printf("Login attempt for email: %s", creds.Email)

	var userID int
	var userEmail, hashedPassword string
	err = config.DB.QueryRow(
		"SELECT usuario_id, email, senha FROM usuario WHERE email = $1",
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

	expirationTime := time.Now().Add(1 * time.Hour)
	claims := &Claims{
		Email:     creds.Email,
		UsuarioID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(config.JWTKey())
	if err != nil {
		http.Error(w, "Erro ao gerar token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":    "Logado com sucesso!!",
		"token":      tokenString,
		"usuario_id": userID,
		"id_usuario": userID,
	})
}

func GetUserHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "ID do usuario nao encontrado no token", http.StatusInternalServerError)
		return
	}

	var user models.Usuario
	err := config.DB.QueryRow("SELECT usua, nome, email, telefone FROM usuario WHERE id_cadastro=$1", userID).Scan(&user.ID, &user.Username, &user.Email, &user.Telefone)
	if err != nil {
		http.Error(w, "Usuario nao encontrado", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func UpdateUsuarioHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Nao foi possivel obter o ID do usuario do token", http.StatusInternalServerError)
		return
	}

	var user models.Usuario
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, "Corpo da requisicao invalido", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(user.Username) == "" {
		http.Error(w, "Requer Nome", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(user.Email) == "" {
		http.Error(w, "Requer Email", http.StatusBadRequest)
		return
	}
	if err := validarEmail(user.Email); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if user.Senha != "" {
		if err := validarSenha(user.Senha); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		hashedPassword, err := utils.HashSenha(user.Senha)
		if err != nil {
			http.Error(w, "Erro ao processar senha", http.StatusInternalServerError)
			return
		}
		_, err = config.DB.Exec(
			"UPDATE usuario SET nome=$1, email=$2, telefone=$3, senha=$4 WHERE id_usuario=$5",
			user.Username, user.Email, user.Telefone, hashedPassword, userID,
		)
	} else {
		_, err = config.DB.Exec(
			"UPDATE usuario SET nome=$1, email=$2, telefone=$3 WHERE id_cadastro=$4",
			user.Username, user.Email, user.Telefone, userID,
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
	json.NewEncoder(w).Encode(map[string]string{"message": "Usuario atualizado com sucesso"})
}

func DeleteUsuarioHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Nao foi possivel obter o ID do usuario do token", http.StatusInternalServerError)
		return
	}

	_, err := config.DB.Exec("DELETE FROM usuario WHERE id_usuario=$1", userID)
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
