package models

import "time"

type AgendamentoPagamento struct {
	ID               int       `json:"id"`
	IDAgendamento    int       `json:"id_agendamento"`
	IDUsuario        *int      `json:"id_usuario,omitempty"`
	ValorPago        float64   `json:"valor_pago"`
	FormaPagamento   string    `json:"forma_pagamento"`
	DataPagamento    time.Time `json:"data_pagamento"`
	NomeUsuario      string    `json:"nome_usuario,omitempty"`
	SobrenomeUsuario string    `json:"sobrenome_usuario,omitempty"`
	EmailUsuario     string    `json:"email_usuario,omitempty"`
}

type AgendamentoPagamentosResumo struct {
	Agendamento Agendamento            `json:"agendamento"`
	Pagamentos  []AgendamentoPagamento `json:"pagamentos"`
	TotalPago   float64                `json:"total_pago"`
}

type RegistrarPagamentoInput struct {
	IDUsuario      *int
	ValorPago      float64
	FormaPagamento string
}
