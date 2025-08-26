package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/danpi/marca_ai_backend/internal/config"
)

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

	log.Printf("Servidor rodando em http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
