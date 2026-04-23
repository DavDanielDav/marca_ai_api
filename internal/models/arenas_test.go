package models

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestArenasJSONIncludesOptionalMetadataFieldsEvenWhenEmpty(t *testing.T) {
	arena := Arenas{
		ID:           3,
		Nome:         "Arena Marca AI",
		Cnpj:         "52592620000160",
		QtdCampos:    2,
		Tipo:         "",
		Imagem:       "https://example.com/arena.jpg",
		Endereco:     "{}",
		Observacoes:  "",
		EmManutencao: false,
	}

	payload, err := json.Marshal(arena)
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}

	body := string(payload)
	for _, field := range []string{
		`"observacoes":""`,
		`"esportes_oferecidos":""`,
		`"informacoes_arena":""`,
	} {
		if !strings.Contains(body, field) {
			t.Fatalf("expected marshaled arena to include %s, got %s", field, body)
		}
	}

	if strings.Contains(body, `"campos"`) {
		t.Fatalf("expected empty campos to remain omitted, got %s", body)
	}
}
