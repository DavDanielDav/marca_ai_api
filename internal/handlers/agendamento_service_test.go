package handlers

import (
	"testing"

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
