package handlers

import (
	"strings"
	"testing"
)

func TestOptionalCampoSelectExpression(t *testing.T) {
	got := optionalCampoSelectExpression("c", "valor_hora", true)
	if got != "c.valor_hora AS valor_hora" {
		t.Fatalf("unexpected valor_hora expression: %q", got)
	}

	got = optionalCampoSelectExpression("c", "ativo", false)
	if got != "CAST(TRUE AS BOOLEAN) AS ativo" {
		t.Fatalf("unexpected ativo fallback expression: %q", got)
	}

	got = optionalCampoSelectExpression("c", "horarios_disponiveis", false)
	if got != "CAST('[]' AS TEXT) AS horarios_disponiveis" {
		t.Fatalf("unexpected horarios_disponiveis fallback expression: %q", got)
	}
}

func TestBuildCampoInsertQueryAndArgs(t *testing.T) {
	columns := campoOptionalColumns{
		ValorHora:           true,
		Ativo:               true,
		HorariosDisponiveis: true,
	}

	query := buildCampoInsertQuery(columns)
	if !strings.Contains(query, "valor_hora") || !strings.Contains(query, "ativo") || !strings.Contains(query, "horarios_disponiveis") {
		t.Fatalf("expected optional campo columns in insert query: %s", query)
	}

	args := buildCampoInsertArgs("Campo 1", "Society", "Grama", "img", 14, 3, 99.90, false, `["07:00","08:00"]`, columns)
	if len(args) != 9 {
		t.Fatalf("expected 9 args with all optional campo columns enabled, got %d", len(args))
	}
}

func TestFirstNonZero(t *testing.T) {
	if got := firstNonZero(0, 0, 9, 4); got != 9 {
		t.Fatalf("expected first non-zero value 9, got %d", got)
	}
}
