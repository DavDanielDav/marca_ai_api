package config

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq" // Driver PostgreSQL
)

var DB *sql.DB

// ConnectDB abre a conexão com o banco PostgreSQL
func ConnectDB() {
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName,
	)

	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("❌ Erro ao abrir conexão com o banco: %v", err)
	}

	err = DB.Ping()
	if err != nil {
		log.Fatalf("❌ Banco de dados não respondeu: %v", err)
	}

	log.Println("✅ Conectado ao PostgreSQL com sucesso!")
}
