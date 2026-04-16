package handlers

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

func TestParseCampoUpdatePayloadMultipart(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	fields := map[string]string{
		"idCampo":      "17",
		"nome_campo":   "Campo reformado",
		"maxJogadores": "14",
		"modalidade":   "Society",
		"tipoCampo":    "Grama sintetica",
		"idArena":      "9",
	}

	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("failed to write multipart field %s: %v", key, err)
		}
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPut, "/editar-campo", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	payload, err := parseCampoUpdatePayload(req)
	if err != nil {
		t.Fatalf("parseCampoUpdatePayload returned error: %v", err)
	}

	if payload.IDCampo != 17 {
		t.Fatalf("expected IDCampo 17, got %d", payload.IDCampo)
	}
	if payload.Nome != "Campo reformado" {
		t.Fatalf("expected Nome to be parsed from multipart form, got %q", payload.Nome)
	}
	if payload.MaxJogadores != 14 {
		t.Fatalf("expected MaxJogadores 14, got %d", payload.MaxJogadores)
	}
	if payload.Modalidade != "Society" {
		t.Fatalf("expected Modalidade Society, got %q", payload.Modalidade)
	}
	if payload.TipoCampo != "Grama sintetica" {
		t.Fatalf("expected TipoCampo to be parsed from multipart form, got %q", payload.TipoCampo)
	}
	if payload.IdArena != 9 {
		t.Fatalf("expected IdArena 9, got %d", payload.IdArena)
	}
}

func TestParseCampoUpdatePayloadJSON(t *testing.T) {
	body, err := json.Marshal(map[string]any{
		"id_campo":      21,
		"nome_campo":    "Campo central",
		"max_jogadores": 10,
		"modalidade":    "Futsal",
		"tipo_campo":    "Concreto",
		"id_arena":      4,
		"imagem":        "https://example.com/campo.png",
	})
	if err != nil {
		t.Fatalf("failed to marshal json body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPut, "/editar-campo", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	payload, parseErr := parseCampoUpdatePayload(req)
	if parseErr != nil {
		t.Fatalf("parseCampoUpdatePayload returned error: %v", parseErr)
	}

	if payload.IDCampo != 21 || payload.IdArena != 4 {
		t.Fatalf("expected IDs from json body, got campo=%d arena=%d", payload.IDCampo, payload.IdArena)
	}
	if payload.Nome != "Campo central" || payload.TipoCampo != "Concreto" {
		t.Fatalf("expected string fields from json body, got nome=%q tipo=%q", payload.Nome, payload.TipoCampo)
	}
}

func TestResolveCampoIDSupportsRouteQueryAndBody(t *testing.T) {
	reqWithRouteID := httptest.NewRequest(http.MethodPut, "/editar-campo/8", nil)
	reqWithRouteID = mux.SetURLVars(reqWithRouteID, map[string]string{"id": "8"})

	id, err := resolveCampoID(reqWithRouteID, 99)
	if err != nil {
		t.Fatalf("resolveCampoID returned error for route id: %v", err)
	}
	if id != 8 {
		t.Fatalf("expected route ID 8, got %d", id)
	}

	reqWithQueryID := httptest.NewRequest(http.MethodPut, "/editar-campo?id=11", nil)
	id, err = resolveCampoID(reqWithQueryID, 99)
	if err != nil {
		t.Fatalf("resolveCampoID returned error for query id: %v", err)
	}
	if id != 11 {
		t.Fatalf("expected query ID 11, got %d", id)
	}

	reqWithBodyID := httptest.NewRequest(http.MethodPut, "/editar-campo", nil)
	id, err = resolveCampoID(reqWithBodyID, 13)
	if err != nil {
		t.Fatalf("resolveCampoID returned error for body fallback: %v", err)
	}
	if id != 13 {
		t.Fatalf("expected fallback ID 13, got %d", id)
	}
}
