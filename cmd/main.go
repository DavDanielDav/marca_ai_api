package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/danpi/marca_ai_backend/internal/config"
	"github.com/joho/godotenv"
)

type UsuarioRegistration struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	//Password string `json:"password"`
	Telefone string `json:"telefone"`
}

func registerUsuarioHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var newUser UsuarioRegistration
	err := json.NewDecoder(r.Body).Decode(&newUser)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("Dados de cadastro recebidos: %+v", newUser)
	//lógica de inserção no banco de dados aqui

	if strings.TrimSpace(newUser.Username) == "" {
		http.Error(w, "Nome is required", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(newUser.Email) == "" {
		http.Error(w, "Email is required", http.StatusBadRequest)
		return
	}
	//if strings.TrimSpace(newUser.password) == "" {
	//http.Error(w, "passaword is required", http.StatusBadRequest)
	//return
	//}
	if strings.TrimSpace(newUser.Telefone) == "" {
		http.Error(w, "passaword is required", http.StatusBadRequest)
		return
	}
	_, err = config.DB.Exec(
		"INSERT INTO cadastro (nome, email, telefone) VALUES ($1, $2, $3)",
		newUser.Username, newUser.Email, newUser.Telefone, // TODO: Hash da senha antes de salvar!
	)
	//apos adicionar coluna passaword(senha) no banco, adicionar no insert dado password
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

type UserDonodeArenaRegistration struct {
	Nome_dono_arena string `json:"nome_dono_arena"`
	Cnpj            string `json:"cnpj"`
	Arena           string `json:"arena"`
}

func registerDonodeArenaHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var newDonodeArena UserDonodeArenaRegistration
	err := json.NewDecoder(r.Body).Decode(&newDonodeArena)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("Dados de cadastro recebidos: %+v", newDonodeArena)
	//lógica de inserção no banco de dados aqui

	if strings.TrimSpace(newDonodeArena.Nome_dono_arena) == "" {
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
		newDonodeArena.Nome_dono_arena, newDonodeArena.Cnpj, newDonodeArena.Arena, // TODO: Hash da senha antes de salvar!
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

func main() {
	// Carregar variáveis do .env
	err := godotenv.Load()
	if err != nil {
		log.Println("⚠️ Aviso: arquivo .env não encontrado, usando variáveis do sistema")
	}

	// Conectar ao banco PostgreSQL
	config.ConnectDB()

	// Porta do servidor (default: 8080)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Rota raiz
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "🚀 API Marca-Ai rodando com Go e PostgreSQL!")
	})

	// Rota de healthcheck
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK")
	})

	// Adicione esta linha para a rota de cadastro
	http.HandleFunc("/register-dono-de-arena", registerDonodeArenaHandler)
	http.HandleFunc("/register-usuario", registerUsuarioHandler)

	log.Printf("Servidor rodando em http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
