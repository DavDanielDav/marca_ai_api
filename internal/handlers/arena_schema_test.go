package handlers

import (
	"strings"
	"testing"
)

func TestOptionalArenaSelectExpression(t *testing.T) {
	got := optionalArenaSelectExpression("", "observacoes", true)
	if got != "COALESCE(observacoes, '') AS observacoes" {
		t.Fatalf("unexpected select expression without alias: %q", got)
	}

	got = optionalArenaSelectExpression("a", "observacoes", true)
	if got != "COALESCE(a.observacoes, '') AS observacoes" {
		t.Fatalf("unexpected select expression with alias: %q", got)
	}

	got = optionalArenaSelectExpression("a", "observacoes", false)
	if got != "CAST('' AS TEXT) AS observacoes" {
		t.Fatalf("unexpected fallback select expression: %q", got)
	}
}

func TestBuildArenaInsertAndUpdateQueriesRespectOptionalColumns(t *testing.T) {
	columns := arenaOptionalColumns{
		Observacoes:        true,
		EsportesOferecidos: false,
		InformacoesArena:   true,
	}

	insertQuery := buildArenaInsertQuery(columns)
	if !strings.Contains(insertQuery, "observacoes") {
		t.Fatalf("expected insert query to include observacoes: %s", insertQuery)
	}
	if strings.Contains(insertQuery, "esportes_oferecidos") {
		t.Fatalf("expected insert query to omit esportes_oferecidos when column is absent: %s", insertQuery)
	}
	if !strings.Contains(insertQuery, "informacoes_arena") {
		t.Fatalf("expected insert query to include informacoes_arena: %s", insertQuery)
	}

	updateQuery := buildArenaUpdateQuery(columns)
	if !strings.Contains(updateQuery, "observacoes = COALESCE") {
		t.Fatalf("expected update query to include observacoes assignment: %s", updateQuery)
	}
	if strings.Contains(updateQuery, "esportes_oferecidos = COALESCE") {
		t.Fatalf("expected update query to omit esportes_oferecidos assignment when column is absent: %s", updateQuery)
	}
	if !strings.Contains(updateQuery, "informacoes_arena = COALESCE") {
		t.Fatalf("expected update query to include informacoes_arena assignment: %s", updateQuery)
	}

	insertArgs := buildArenaInsertArgs("Arena X", "123", 2, "Society", "img", "Rua A", "Obs", "Futsal", "Info", 99, columns)
	if len(insertArgs) != 9 {
		t.Fatalf("expected 9 insert args with two optional columns enabled, got %d", len(insertArgs))
	}

	updateArgs := buildArenaUpdateArgs("Arena X", "123", 2, "Society", "Rua A", "img", "Obs", "Futsal", "Info", 99, columns)
	if len(updateArgs) != 9 {
		t.Fatalf("expected 9 update args with two optional columns enabled, got %d", len(updateArgs))
	}
}
