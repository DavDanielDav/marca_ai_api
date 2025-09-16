package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/danpi/marca_ai_backend/internal/config"
	"github.com/danpi/marca_ai_backend/internal/handlers"
	"github.com/joho/godotenv"
)

func main() {
	// Carregar variáveis do .env
	err := godotenv.Load()
	if err != nil {
		log.Println("⚠️ Aviso: arquivo .env não encontrado, usando variáveis do sistema")
	}

	mux := http.NewServeMux()

	// Conectar ao banco PostgreSQL
	config.ConnectDB()

	// Porta do servidor (default: 8080)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Rota raiz
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "🚀 API Marca-Ai rodando com Go e PostgreSQL!")
	})

	// Rota de healthcheck
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK")
	})

	// Adicione esta linha para a rota de cadastro

	mux.HandleFunc("/Cadastro", handlers.RegisterUsuarioHandler)
	mux.HandleFunc("/register-dono-de-arena", handlers.RegisterDonodeArenaHandler)
	mux.HandleFunc("/login", handlers.LoginHandler)

	log.Printf("Servidor rodando em http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
