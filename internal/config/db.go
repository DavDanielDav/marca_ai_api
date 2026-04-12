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

func IsRenderEnvironment() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("RENDER")), "true") ||
		strings.TrimSpace(os.Getenv("RENDER_EXTERNAL_HOSTNAME")) != ""
}

func isLocalDatabaseHost(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return false
	}

	switch host {
	case "localhost", "127.0.0.1", "::1", "0.0.0.0", "host.docker.internal":
		return true
	}

	return strings.HasSuffix(host, ".local")
}

func inferSSLMode(host string) string {
	if explicit := strings.TrimSpace(os.Getenv("DB_SSLMODE")); explicit != "" {
		return explicit
	}

	if IsRenderEnvironment() {
		return "require"
	}

	if isLocalDatabaseHost(host) {
		return "disable"
	}

	if ip := net.ParseIP(host); ip != nil {
		if ip.IsLoopback() {
			return "disable"
		}

		return "disable"
	}

	if strings.Contains(host, ".") {
		return "require"
	}

	return "disable"
}

func buildPostgresDSN() (string, string, error) {
	if databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL")); databaseURL != "" {
		parsedURL, err := url.Parse(databaseURL)
		if err != nil {
			return "", "", fmt.Errorf("DATABASE_URL invalida: %w", err)
		}

		query := parsedURL.Query()
		if query.Get("sslmode") == "" {
			query.Set("sslmode", inferSSLMode(parsedURL.Hostname()))
			parsedURL.RawQuery = query.Encode()
		}

		return parsedURL.String(), "DATABASE_URL", nil
	}

	host := strings.TrimSpace(os.Getenv("DB_HOST"))
	port := strings.TrimSpace(os.Getenv("DB_PORT"))
	user := strings.TrimSpace(os.Getenv("DB_USER"))
	password := os.Getenv("DB_PASSWORD")
	name := strings.TrimSpace(os.Getenv("DB_NAME"))

	missing := make([]string, 0, 5)
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
		return "", "", fmt.Errorf(
			"configuracao do banco ausente: defina DATABASE_URL ou as variaveis DB_HOST, DB_PORT, DB_USER e DB_NAME; use DB_PASSWORD quando o seu banco exigir. Faltando: %s",
			strings.Join(missing, ", "),
		)
	}

	connectionURL := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(user, password),
		Host:   net.JoinHostPort(host, port),
		Path:   name,
	}

	query := connectionURL.Query()
	query.Set("sslmode", inferSSLMode(host))
	connectionURL.RawQuery = query.Encode()

	return connectionURL.String(), "DB_HOST/DB_PORT/DB_USER/DB_PASSWORD/DB_NAME", nil
}

func ConnectDB() {
	dsn, source, err := buildPostgresDSN()
	if err != nil {
		log.Fatalf("erro de configuracao do banco: %v", err)
	}

	DB, err = sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("erro ao abrir conexao com o banco: %v", err)
	}

	err = DB.Ping()
	if err != nil {
		log.Fatalf("banco de dados nao respondeu: %v", err)
	}

	log.Printf("conectado ao PostgreSQL com sucesso usando %s", source)
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
