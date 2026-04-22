package handlers

import (
	"context"
	"testing"
	"time"

	"github.com/danpi/marca_ai_backend/internal/models"
)

func TestCalcularValorRestanteNeverNegative(t *testing.T) {
	if got := calcularValorRestante(120, 150); got != 0 {
		t.Fatalf("expected remaining value to floor at zero, got %v", got)
	}
}

func TestShouldNotifyJogadorOnlyForExternalAcceptedOrCanceled(t *testing.T) {
	manual := models.Agendamento{
		OrigemAgendamento: models.AgendamentoOrigemManual,
		Status:            models.AgendamentoStatusAgendado,
	}
	if shouldNotifyJogador(manual) {
		t.Fatal("manual agendamento should not trigger notification")
	}

	pedidoAceito := models.Agendamento{
		OrigemAgendamento: models.AgendamentoOrigemJogador,
		Status:            models.AgendamentoStatusAgendado,
	}
	if !shouldNotifyJogador(pedidoAceito) {
		t.Fatal("external accepted agendamento should trigger notification")
	}

	pedidoPendente := models.Agendamento{
		OrigemAgendamento: models.AgendamentoOrigemTimeXTime,
		Status:            models.AgendamentoStatusPedido,
	}
	if shouldNotifyJogador(pedidoPendente) {
		t.Fatal("pending request should not trigger notification")
	}
}

func TestResolveFinancialStateSupportsLegacyPaidFlag(t *testing.T) {
	valorRestante, pago, statusDePagamento := resolveFinancialState(180, 0, true, true)
	if valorRestante != 0 {
		t.Fatalf("expected remaining value 0 for legacy paid flag, got %v", valorRestante)
	}
	if !pago || !statusDePagamento {
		t.Fatal("expected financial flags to remain paid when legacy flag is set")
	}
}

func TestStatusAfterCronometroEncerradoDependsOnRemainingValue(t *testing.T) {
	if got := statusAfterCronometroEncerrado(50); got != models.AgendamentoStatusAguardandoPagamento {
		t.Fatalf("expected awaiting payment status, got %q", got)
	}

	if got := statusAfterCronometroEncerrado(0); got != models.AgendamentoStatusAgendado {
		t.Fatalf("expected agendado status when remaining is zero, got %q", got)
	}
}

func TestAgendamentoLocationUsesSaoPauloOffset(t *testing.T) {
	location := agendamentoLocation()
	_, offset := time.Date(2026, time.April, 21, 12, 0, 0, 0, location).Zone()
	if offset != -3*60*60 {
		t.Fatalf("expected Sao Paulo offset -03:00, got %d", offset)
	}
}

func TestResolveNomeSolicitanteFromJogadorUsesNomeDoBanco(t *testing.T) {
	jogadorID := 42
	input := models.CreateAgendamentoInput{
		OrigemAgendamento: models.AgendamentoOrigemJogador,
		IDUsuarioJogador:  &jogadorID,
		NomeSolicitante:   "Nome enviado pelo payload",
	}

	resolved, err := resolveNomeSolicitanteFromJogador(
		context.Background(),
		input,
		func(ctx context.Context, id int) (string, error) {
			if id != jogadorID {
				t.Fatalf("expected jogador id %d, got %d", jogadorID, id)
			}

			return "Maria Souza", nil
		},
	)
	if err != nil {
		t.Fatalf("resolveNomeSolicitanteFromJogador returned error: %v", err)
	}
	if resolved.NomeSolicitante != "Maria Souza" {
		t.Fatalf("expected nome_solicitante from database, got %q", resolved.NomeSolicitante)
	}
}

func TestResolveNomeSolicitanteFromJogadorSkipsLookupWithoutJogadorID(t *testing.T) {
	lookupCalled := false
	input := models.CreateAgendamentoInput{
		OrigemAgendamento: models.AgendamentoOrigemJogador,
		NomeSolicitante:   "Nome enviado pelo payload",
	}

	resolved, err := resolveNomeSolicitanteFromJogador(
		context.Background(),
		input,
		func(context.Context, int) (string, error) {
			lookupCalled = true
			return "", nil
		},
	)
	if err != nil {
		t.Fatalf("resolveNomeSolicitanteFromJogador returned error: %v", err)
	}
	if lookupCalled {
		t.Fatal("expected lookup to be skipped when player id is absent")
	}
	if resolved.NomeSolicitante != input.NomeSolicitante {
		t.Fatalf("expected nome_solicitante to remain unchanged, got %q", resolved.NomeSolicitante)
	}
}

func TestBuildNomeCompleto(t *testing.T) {
	if got := buildNomeCompleto(" Maria ", " Souza "); got != "Maria Souza" {
		t.Fatalf("expected full name with trimmed spaces, got %q", got)
	}

	if got := buildNomeCompleto("Maria", ""); got != "Maria" {
		t.Fatalf("expected single-name result, got %q", got)
	}
}
