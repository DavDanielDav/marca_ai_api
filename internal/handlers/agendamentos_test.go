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
