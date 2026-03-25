package config

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

var DB *sql.DB

// ConnectDB opens the PostgreSQL connection.
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
		log.Fatalf("Error opening database connection: %v", err)
	}

	err = DB.Ping()
	if err != nil {
		log.Fatalf("Database did not respond: %v", err)
	}

	if err := ensureForgotPasswordColumns(); err != nil {
		log.Fatalf("Failed to ensure forgot-password columns: %v", err)
	}

	log.Println("Connected to PostgreSQL successfully")
}

func ensureForgotPasswordColumns() error {
	_, err := DB.Exec(`
		ALTER TABLE usuario
		ADD COLUMN IF NOT EXISTS reset_code VARCHAR(6),
		ADD COLUMN IF NOT EXISTS reset_expiry TIMESTAMPTZ
	`)
	return err
}
