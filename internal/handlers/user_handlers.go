package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/danpi/marca_ai_backend/internal/config"
	"github.com/danpi/marca_ai_backend/internal/models"
	"github.com/danpi/marca_ai_backend/internal/utils"
)

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
	if strings.TrimSpace(newUser.Telefone) == "" {
		http.Error(w, "Telefone is required", http.StatusBadRequest)
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
		"INSERT INTO usuario (nome, email, telefone, senha) VALUES ($1, $2, $3, $4)",
		newUser.Username, newUser.Email, newUser.Telefone, hashedPassword,
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

func RegisterDonodeArenaHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var newDonodeArena models.DonoDeArena
	err := json.NewDecoder(r.Body).Decode(&newDonodeArena)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("Dados de cadastro recebidos: %+v", newDonodeArena)
	//lógica de inserção no banco de dados aqui

	if strings.TrimSpace(newDonodeArena.NomeDonoArena) == "" {
		http.Error(w, "Nome is required", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(newDonodeArena.Cnpj) == "" {
		http.Error(w, "CNPJ is required", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(newDonodeArena.Arena) == "" {
		http.Error(w, "Nome da arena is required", http.StatusBadRequest)
		return
	}
	_, err = config.DB.Exec(
		"INSERT INTO dono_de_arena (nome_dono_arena, cnpj, arena) VALUES ($1, $2, $3)",
		newDonodeArena.NomeDonoArena, newDonodeArena.Cnpj, newDonodeArena.Arena, // TODO: Hash da senha antes de salvar!
	)

	if err != nil {
		log.Printf("Erro ao inserir usuário Dono de arena no banco de dados: %v", err)
		http.Error(w, "Erro ao registrar Dono de arena", http.StatusInternalServerError)
		return
	}

	log.Println("Dono de arena inserido no banco de dados com sucesso!")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Dono de Arena registrado com sucesso"})
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

	var userEmail, hashedPassword string
	err = config.DB.QueryRow(
		"SELECT email, senha FROM usuario WHERE email = $1",
		creds.Email,
	).Scan(&userEmail, &hashedPassword)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "User not found", http.StatusNotFound)
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

	// Login OK (aqui poderia gerar JWT, por exemplo)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Login successful",
		"email":   userEmail,
	})
}
