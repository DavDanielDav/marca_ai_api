package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFormatAgendamentoDateTimeDoesNotIncludeTimezone(t *testing.T) {
	formatted := formatAgendamentoDateTime(time.Date(2026, 4, 22, 17, 0, 0, 0, time.UTC))

	if formatted != "2026-04-22T17:00:00" {
		t.Fatalf("expected local wall-clock format, got %q", formatted)
	}

	if strings.Contains(formatted, "Z") || strings.Contains(formatted, "+") {
		t.Fatalf("expected formatted time without timezone suffix, got %q", formatted)
	}
}

func TestParseAgendamentoCreateRequestSupportsJogadorIDAliases(t *testing.T) {
	testCases := []struct {
		name       string
		body       string
		expectedID int
	}{
		{
			name:       "id_usuario_jogador",
			body:       `{"campo_id":1,"horario":"2026-04-22T17:00","jogadores":10,"id_usuario_jogador":17}`,
			expectedID: 17,
		},
		{
			name:       "id_jogador",
			body:       `{"campo_id":1,"horario":"2026-04-22T17:00","jogadores":10,"id_jogador":21}`,
			expectedID: 21,
		},
		{
			name:       "id_usuario",
			body:       `{"campo_id":1,"horario":"2026-04-22T17:00","jogadores":10,"id_usuario":34}`,
			expectedID: 34,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/integracao/agendamentos", strings.NewReader(testCase.body))

			input, err := parseAgendamentoCreateRequest(request)
			if err != nil {
				t.Fatalf("parseAgendamentoCreateRequest returned error: %v", err)
			}
			if input.IDUsuarioJogador == nil {
				t.Fatal("expected IDUsuarioJogador to be filled")
			}
			if *input.IDUsuarioJogador != testCase.expectedID {
				t.Fatalf("expected IDUsuarioJogador %d, got %d", testCase.expectedID, *input.IDUsuarioJogador)
			}
		})
	}
}

func TestParseAgendamentoCreateRequestAcceptsStringCampoID(t *testing.T) {
	request := httptest.NewRequest(
		http.MethodPost,
		"/integracao/agendamentos",
		strings.NewReader(`{"id_campo":"3","horario":"2026-04-28T18:00","jogadores":"10"}`),
	)

	input, err := parseAgendamentoCreateRequest(request)
	if err != nil {
		t.Fatalf("parseAgendamentoCreateRequest returned error: %v", err)
	}
	if input.IDCampo != 3 {
		t.Fatalf("expected IDCampo 3, got %d", input.IDCampo)
	}
	if input.Jogadores != 10 {
		t.Fatalf("expected Jogadores 10, got %d", input.Jogadores)
	}
}

func TestParseAgendamentoCreateRequestSupportsCampoIDAliases(t *testing.T) {
	testCases := []struct {
		name string
		url  string
		body string
	}{
		{
			name: "id_campo",
			url:  "/integracao/agendamentos",
			body: `{"id_campo":3,"horario":"2026-04-28T18:00","jogadores":10}`,
		},
		{
			name: "idCampo",
			url:  "/integracao/agendamentos",
			body: `{"idCampo":4,"horario":"2026-04-28T18:00","jogadores":10}`,
		},
		{
			name: "campoId query",
			url:  "/integracao/agendamentos?campoId=5&horario=2026-04-28T18:00&jogadores=10",
			body: ``,
		},
		{
			name: "idCampo query",
			url:  "/integracao/agendamentos?idCampo=6&horario=2026-04-28T18:00&jogadores=10",
			body: ``,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, testCase.url, strings.NewReader(testCase.body))

			input, err := parseAgendamentoCreateRequest(request)
			if err != nil {
				t.Fatalf("parseAgendamentoCreateRequest returned error: %v", err)
			}
			if input.IDCampo <= 0 {
				t.Fatalf("expected IDCampo to be filled, got %d", input.IDCampo)
			}
		})
	}
}

func TestParseAgendamentoCreateRequestRejectsInvalidJogadorID(t *testing.T) {
	request := httptest.NewRequest(
		http.MethodPost,
		"/integracao/agendamentos",
		strings.NewReader(`{"campo_id":1,"horario":"2026-04-22T17:00","jogadores":10,"id_usuario_jogador":0}`),
	)

	_, err := parseAgendamentoCreateRequest(request)
	if err == nil {
		t.Fatal("expected invalid player id to return an error")
	}
	if err.Error() != "ID do jogador invalido" {
		t.Fatalf("expected invalid player id error, got %q", err.Error())
	}
}
