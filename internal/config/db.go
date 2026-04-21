package config

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"strings"
	"unicode"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var DB *sql.DB

func sanitizeIdentifier(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	for _, char := range value {
		if char != '_' && !unicode.IsLetter(char) && !unicode.IsDigit(char) {
			return ""
		}
	}

	return value
}

func quoteIdentifier(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

func DBSchemaName() string {
	if schema := sanitizeIdentifier(os.Getenv("DB_SCHEMA")); schema != "" {
		return schema
	}

	return "arena"
}

func QualifiedName(name string) string {
	return fmt.Sprintf("%s.%s", quoteIdentifier(DBSchemaName()), quoteIdentifier(strings.TrimSpace(name)))
}

func normalizeSearchPath(value string) string {
	parts := strings.Split(value, ",")
	normalized := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))

	for _, part := range parts {
		schema := strings.TrimSpace(part)
		if schema == "" {
			continue
		}

		key := strings.ToLower(schema)
		if _, exists := seen[key]; exists {
			continue
		}

		seen[key] = struct{}{}
		normalized = append(normalized, schema)
	}

	return strings.Join(normalized, ",")
}

func resolveDBSearchPath() string {
	if explicit := normalizeSearchPath(os.Getenv("DB_SEARCH_PATH")); explicit != "" {
		return explicit
	}

	if schema := DBSchemaName(); schema != "" {
		return normalizeSearchPath(schema + ",public")
	}

	return "arena,public"
}

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
	searchPath := resolveDBSearchPath()

	if databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL")); databaseURL != "" {
		parsedURL, err := url.Parse(databaseURL)
		if err != nil {
			return "", "", fmt.Errorf("DATABASE_URL invalida: %w", err)
		}

		query := parsedURL.Query()
		if query.Get("sslmode") == "" {
			query.Set("sslmode", inferSSLMode(parsedURL.Hostname()))
		}
		if query.Get("search_path") == "" && searchPath != "" {
			query.Set("search_path", searchPath)
		}
		parsedURL.RawQuery = query.Encode()

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
	if searchPath != "" {
		query.Set("search_path", searchPath)
	}
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

	log.Printf("conectado ao PostgreSQL com sucesso usando %s (search_path=%s)", source, resolveDBSearchPath())
}

func EnsureEmailCodesTable() {
	if DB == nil {
		log.Fatal("database connection not initialized")
	}

	emailCodesTable := QualifiedName("email_codes")
	var exists bool
	if err := DB.QueryRow(
		`SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = $1 AND table_name = 'email_codes'
		)`,
		DBSchemaName(),
	).Scan(&exists); err != nil {
		log.Fatalf("failed to validate email_codes table: %v", err)
	}

	if !exists {
		log.Fatalf("required table %s was not found", emailCodesTable)
	}

	log.Println("email_codes table is ready")
}
