package config

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var DB *sql.DB

func envOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	return value
}

func buildPostgresDSN() (string, error) {
	if databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL")); databaseURL != "" {
		parsedURL, err := url.Parse(databaseURL)
		if err != nil {
			return "", fmt.Errorf("DATABASE_URL invalida: %w", err)
		}

		query := parsedURL.Query()
		if query.Get("sslmode") == "" {
			query.Set("sslmode", envOrDefault("DB_SSLMODE", "disable"))
			parsedURL.RawQuery = query.Encode()
		}

		return parsedURL.String(), nil
	}

	host := strings.TrimSpace(os.Getenv("DB_HOST"))
	port := strings.TrimSpace(os.Getenv("DB_PORT"))
	user := strings.TrimSpace(os.Getenv("DB_USER"))
	password := os.Getenv("DB_PASSWORD")
	name := strings.TrimSpace(os.Getenv("DB_NAME"))

	missing := make([]string, 0, 4)
	if host == "" {
		missing = append(missing, "DB_HOST")
	}
	if port == "" {
		missing = append(missing, "DB_PORT")
	}
	if user == "" {
		missing = append(missing, "DB_USER")
	}
	if name == "" {
		missing = append(missing, "DB_NAME")
	}

	if len(missing) > 0 {
		return "", fmt.Errorf("variaveis do banco ausentes: %s", strings.Join(missing, ", "))
	}

	connectionURL := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(user, password),
		Host:   net.JoinHostPort(host, port),
		Path:   name,
	}

	query := connectionURL.Query()
	query.Set("sslmode", envOrDefault("DB_SSLMODE", "disable"))
	connectionURL.RawQuery = query.Encode()

	return connectionURL.String(), nil
}

func ConnectDB() {
	dsn, err := buildPostgresDSN()
	if err != nil {
		log.Fatal(err)
	}

	DB, err = sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("erro ao abrir conexao com o banco: %v", err)
	}

	err = DB.Ping()
	if err != nil {
		log.Fatalf("banco de dados nao respondeu: %v", err)
	}

	log.Println("conectado ao PostgreSQL com sucesso")
}

func EnsureEmailCodesTable() {
	if DB == nil {
		log.Fatal("database connection not initialized")
	}

	statements := []string{
		`
		CREATE TABLE IF NOT EXISTS email_codes (
			id BIGSERIAL PRIMARY KEY,
			email TEXT NOT NULL,
			purpose TEXT NOT NULL,
			code_hash TEXT NOT NULL,
			payload BYTEA,
			expires_at TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			CONSTRAINT email_codes_email_purpose_key UNIQUE (email, purpose)
		)
		`,
		`CREATE INDEX IF NOT EXISTS idx_email_codes_expires_at ON email_codes (expires_at)`,
	}

	for _, statement := range statements {
		if _, err := DB.Exec(statement); err != nil {
			log.Fatalf("failed to ensure email_codes table: %v", err)
		}
	}

	log.Println("email_codes table is ready")
}
