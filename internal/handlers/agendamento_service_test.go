package handlers

import (
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
