package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/danpi/marca_ai_backend/internal/config"
	"github.com/danpi/marca_ai_backend/internal/models"
	"github.com/danpi/marca_ai_backend/internal/utils"
	"github.com/gorilla/mux"
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

func UpdateUsuarioHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var user models.Usuario
	err = json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validações
	if strings.TrimSpace(user.Username) == "" {
		http.Error(w, "Requer Nome", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(user.Email) == "" {
		http.Error(w, "Requer Email", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(user.Telefone) == "" {
		http.Error(w, "Telefone is required", http.StatusBadRequest)
		return
	}

	// Se a senha for fornecida, atualize o hash
	if user.Senha != "" {
		hashedPassword, err := utils.HashSenha(user.Senha)
		if err != nil {
			http.Error(w, "Erro ao processar senha", http.StatusInternalServerError)
			return
		}
		_, err = config.DB.Exec(
			"UPDATE usuario SET nome=$1, email=$2, telefone=$3, senha=$4 WHERE id=$5",
			user.Username, user.Email, user.Telefone, hashedPassword, id,
		)
		if err != nil {
			log.Printf("Erro ao atualizar usuario no banco: %v", err)
			http.Error(w, "Erro ao atualizar usuario", http.StatusInternalServerError)
			return
		}
	} else {
		// Se a senha não for fornecida, não a atualize
		_, err = config.DB.Exec(
			"UPDATE usuario SET nome=$1, email=$2, telefone=$3 WHERE id=$4",
			user.Username, user.Email, user.Telefone, id,
		)
		if err != nil {
			log.Printf("Erro ao atualizar usuario no banco: %v", err)
			http.Error(w, "Erro ao atualizar usuario", http.StatusInternalServerError)
			return
		}
	}

	log.Printf("Usuario com ID %d atualizado com sucesso!", id)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Usuario atualizado com sucesso"})
}

func DeleteUsuarioHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id_cadastro"])
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	_, err = config.DB.Exec("DELETE FROM usuario WHERE id_cadastro=$1", id)
	if err != nil {
		log.Printf("Erro ao deletar usuario do banco: %v", err)
		http.Error(w, "Erro ao deletar usuario", http.StatusInternalServerError)
		return
	}

	log.Printf("Usuario com ID %d deletado com sucesso!", id)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Usuario deletado com sucesso"})
}
