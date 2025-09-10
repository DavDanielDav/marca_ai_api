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
	//lógica de inserção no banco de dados aqui

	if strings.TrimSpace(newUser.Username) == "" {
		http.Error(w, "Requer Nome", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(newUser.Email) == "" {
		http.Error(w, "Requer Email", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(newUser.Senha) == "" {
		http.Error(w, "passaword is required", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(newUser.Telefone) == "" {
		http.Error(w, "passaword is required", http.StatusBadRequest)
		return
	}

	_, err = utils.HashSenha(newUser.Senha)
	if err != nil {
		http.Error(w, "Erro ao processar senha", http.StatusInternalServerError)
		return
	}

	_, err = config.DB.Exec(
		"INSERT INTO cadastro (nome, email, telefone, senha) VALUES ($1, $2, $3, $4)",
		newUser.Username, newUser.Email, newUser.Telefone, newUser.Senha,
	)

	if err != nil {
		log.Printf("Erro ao inserir usuario no banco: %v", err)
		http.Error(w, "Erro ao registrar usuario", http.StatusInternalServerError)
		return
	}

	log.Println("Usuario inserido no banco de dados com sucesso!")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
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
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	email := r.URL.Query().Get("email")
	senha := r.URL.Query().Get("senha")

	if strings.TrimSpace(email) == "" {
		http.Error(w, "Email is required", http.StatusBadRequest)
		return
	}

	log.Printf("Login attempt for email: %s", email)

	var userEmail string
	err := config.DB.QueryRow("SELECT email FROM cadastro WHERE email = $1", email).Scan(&userEmail)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		log.Printf("Error querying database: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	hashedSenha := ""
	if !utils.CheckSenhaHash(senha, hashedSenha) {
		http.Error(w, "Senha incorreta", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Login successful"})
}
