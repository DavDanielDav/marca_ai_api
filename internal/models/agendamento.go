package models

import (
	"strings"
	"time"
)

type AgendamentoOrigem string

const (
	AgendamentoOrigemManual    AgendamentoOrigem = "manual"
	AgendamentoOrigemJogador   AgendamentoOrigem = "jogador"
	AgendamentoOrigemTimeXTime AgendamentoOrigem = "time_x_time"
)

type AgendamentoStatus string

const (
	AgendamentoStatusPedido              AgendamentoStatus = "pedido"
	AgendamentoStatusAgendado            AgendamentoStatus = "agendado"
	AgendamentoStatusEmAndamento         AgendamentoStatus = "em_andamento"
	AgendamentoStatusAguardandoPagamento AgendamentoStatus = "aguardando_pagamento"
	AgendamentoStatusCancelado           AgendamentoStatus = "cancelado"
	AgendamentoStatusConcluido           AgendamentoStatus = "concluido"
)

type Agendamento struct {
	ID                 int               `json:"id"`
	IDUsuario          int               `json:"id_usuario,omitempty"`
	IDCampo            int               `json:"id_campo"`
	IDArena            int               `json:"id_arena,omitempty"`
	Horario            time.Time         `json:"horario"`
	Jogadores          int               `json:"jogadores"`
	Pagamento          string            `json:"pagamento"`
	Pago               bool              `json:"pago"`
	Status             AgendamentoStatus `json:"status"`
	CriadoEm           time.Time         `json:"criado_em,omitempty"`
	NomeCampo          string            `json:"nome_campo,omitempty"`
	NomeArena          string            `json:"nome_arena,omitempty"`
	OrigemAgendamento  AgendamentoOrigem `json:"origem_agendamento"`
	ValorTotal         float64           `json:"valor_total"`
	ValorRestante      float64           `json:"valor_restante"`
	StatusDePagamento  bool              `json:"status_de_pagamento"`
	InicioCronometro   *int64            `json:"inicio_cronometro,omitempty"`
	FimCronometro      *time.Time        `json:"fim_cronometro,omitempty"`
	Time1              string            `json:"time1,omitempty"`
	Time2              string            `json:"time2,omitempty"`
	ModoDeJogo         string            `json:"modo_de_jogo,omitempty"`
	OrigemStatusEvento string            `json:"origem_status_evento,omitempty"`
}

type CreateAgendamentoInput struct {
	IDCampo           int
	Horario           time.Time
	Jogadores         int
	Pagamento         string
	Pago              bool
	OrigemAgendamento AgendamentoOrigem
	IDUsuario         *int
	Time1             string
	Time2             string
	ModoDeJogo        string
}

func NormalizeAgendamentoOrigem(raw string) (AgendamentoOrigem, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(AgendamentoOrigemManual):
		return AgendamentoOrigemManual, true
	case string(AgendamentoOrigemJogador):
		return AgendamentoOrigemJogador, true
	case string(AgendamentoOrigemTimeXTime):
		return AgendamentoOrigemTimeXTime, true
	default:
		return "", false
	}
}

func NormalizeAgendamentoStatus(raw string) (AgendamentoStatus, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(AgendamentoStatusPedido):
		return AgendamentoStatusPedido, true
	case string(AgendamentoStatusAgendado):
		return AgendamentoStatusAgendado, true
	case string(AgendamentoStatusEmAndamento):
		return AgendamentoStatusEmAndamento, true
	case string(AgendamentoStatusAguardandoPagamento):
		return AgendamentoStatusAguardandoPagamento, true
	case string(AgendamentoStatusCancelado):
		return AgendamentoStatusCancelado, true
	case "concluido", "concluído":
		return AgendamentoStatusConcluido, true
	default:
		return "", false
	}
}
