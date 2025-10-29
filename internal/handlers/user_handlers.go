package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/danpi/marca_ai_backend/internal/config"
	"github.com/danpi/marca_ai_backend/internal/middleware"
	"github.com/danpi/marca_ai_backend/internal/models"
	"github.com/danpi/marca_ai_backend/internal/utils"
	"github.com/golang-jwt/jwt/v5"
)

var jwtKey = []byte("Tn9Jb2lfVGhpc19pc19hX3N0cm9uZ19qd3Rfa2V5X2ZvciB5b3Uh")

func RegisterUsuarioHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var newUser models.Usuario
	err := json.NewDecoder(r.Body).Decode(&newUser)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("Dados de cadastro recebidos: %+v", newUser)

	// Validações
	if strings.TrimSpace(newUser.Username) == "" {
		http.Error(w, "Requer Nome", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(newUser.Email) == "" {
		http.Error(w, "Requer Email", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(newUser.Senha) == "" {
		http.Error(w, "Password is required", http.StatusBadRequest)
		return
	}

	// Gera o hash da senha
	hashedPassword, err := utils.HashSenha(newUser.Senha)
	if err != nil {
		http.Error(w, "Erro ao processar senha", http.StatusInternalServerError)
		return
	}

	// Salva no banco já com hash
	_, err = config.DB.Exec(
		"INSERT INTO usuario (nome, email, senha) VALUES ($1, $2, $3 )",
		newUser.Username, newUser.Email, hashedPassword,
	)
	if err != nil {
		log.Printf("Erro ao inserir usuario no banco: %v", err)
		http.Error(w, "Erro ao registrar usuario", http.StatusInternalServerError)
		return
	}

	log.Println("Usuario inserido no banco de dados com sucesso!")

	// Limpa senha antes de retornar a resposta
	newUser.Senha = ""

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "Usuario registrado com sucesso"})
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
		"SELECT id_usuario, email, senha FROM usuario WHERE email = $1",
		creds.Email,
	).Scan(&userID, &userEmail, &hashedPassword)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Usuario nao encontrado!!", http.StatusNotFound)
			return
		}
		log.Printf("Error querying database: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Verifica senha
	if !utils.CheckSenhaHash(creds.Senha, hashedPassword) {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Login OK gerar JWT
	experationTime := time.Now().Add(1 * time.Hour)

	claims := &middleware.Claims{
		ID:    userID,
		Email: userEmail,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(experationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		http.Error(w, "Erro ao gerar token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"messagem": "Logado com sucesso!!",
		"token":    tokenString,
	})
}

func GetUserHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Não foi possível obter o ID do usuário do token", http.StatusInternalServerError)
		return
	}

	var user models.Usuario
	err := config.DB.QueryRow("SELECT id_cadastro, nome, email, telefone FROM usuario WHERE id_cadastro=$1", userID).Scan(&user.ID, &user.Username, &user.Email, &user.Telefone)
	if err != nil {
		http.Error(w, "Usuário não encontrado", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func UpdateUsuarioHandler(w http.ResponseWriter, r *http.Request) {
	// Pega o ID do usuário do contexto, injetado pelo middleware.
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Não foi possível obter o ID do usuário do token", http.StatusInternalServerError)
		return
	}

	var user models.Usuario
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, "Corpo da requisição inválido", http.StatusBadRequest)
		return
	}

	// Se a senha for fornecida, atualize o hash
	if user.Senha != "" {
		hashedPassword, err := utils.HashSenha(user.Senha)
		if err != nil {
			http.Error(w, "Erro ao processar senha", http.StatusInternalServerError)
			return
		}
		_, err = config.DB.Exec(`
			UPDATE usuario 
			SET 
			nome = COALESCE(NULLIF($1, ''), nome),
			email = COALESCE(NULLIF($2, ''), email),
			telefone = COALESCE(NULLIF($3, ''), telefone),
			senha = COALESCE(NULLIF($4, ''), senha)
			WHERE id_usuario = $5`,
			user.Username, user.Email, user.Telefone, hashedPassword, userID,
		)
		if err != nil {
			log.Printf("Erro ao atualizar usuario no banco: %v", err)
			http.Error(w, "Erro ao atualizar usuario", http.StatusInternalServerError)
			return
		}

	} else {
		// Se a senha não for fornecida, não a atualize
		_, err = config.DB.Exec(`
			UPDATE usuario 
			SET 
			nome = COALESCE(NULLIF($1, ''), nome),
			email = COALESCE(NULLIF($2, ''), email),
			telefone = COALESCE(NULLIF($3, ''), telefone)
			WHERE id_usuario = $4`,
			user.Username, user.Email, user.Telefone, userID,
		)
		if err != nil {
			log.Printf("Erro ao atualizar usuario no banco: %v", err)
			http.Error(w, "Erro ao atualizar usuario", http.StatusInternalServerError)
			return
		}
	}

	log.Printf("Usuario com ID %d atualizado com sucesso!", userID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Usuario atualizado com sucesso"})
}

// DeleteUsuarioHandler deleta o usuário logado.
func DeleteUsuarioHandler(w http.ResponseWriter, r *http.Request) {
	// Pega o ID do usuário do contexto.
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Não foi possível obter o ID do usuário do token", http.StatusInternalServerError)
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
