package models

import "testing"

func TestNormalizeAgendamentoOrigem(t *testing.T) {
	tests := map[string]AgendamentoOrigem{
		"manual":      AgendamentoOrigemManual,
		"jogador":     AgendamentoOrigemJogador,
		"time_x_time": AgendamentoOrigemTimeXTime,
	}

	for raw, expected := range tests {
		got, ok := NormalizeAgendamentoOrigem(raw)
		if !ok {
			t.Fatalf("expected origem %q to be valid", raw)
		}
		if got != expected {
			t.Fatalf("expected origem %q, got %q", expected, got)
		}
	}
}

func TestNormalizeAgendamentoStatusSupportsLegacyAccent(t *testing.T) {
	got, ok := NormalizeAgendamentoStatus("concluído")
	if !ok {
		t.Fatal("expected accented status to be valid")
	}
	if got != AgendamentoStatusConcluido {
		t.Fatalf("expected status %q, got %q", AgendamentoStatusConcluido, got)
	}
}
