package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/danpi/marca_ai_backend/internal/models"
)

type jogadorStatusNotifier struct {
	httpClient    *http.Client
	callbackURL   string
	callbackToken string
}

type jogadorStatusCallbackPayload struct {
	IDAgendamento     int     `json:"id_agendamento"`
	IDCampo           int     `json:"id_campo"`
	IDArena           int     `json:"id_arena,omitempty"`
	Status            string  `json:"status"`
	OrigemAgendamento string  `json:"origem_agendamento"`
	Horario           string  `json:"horario"`
	NomeCampo         string  `json:"nome_campo,omitempty"`
	NomeArena         string  `json:"nome_arena,omitempty"`
	ValorTotal        float64 `json:"valor_total"`
	ValorRestante     float64 `json:"valor_restante"`
}

type jogadorNotificationResult struct {
	Attempted   bool                         `json:"attempted"`
	Delivered   bool                         `json:"delivered"`
	CallbackURL string                       `json:"callback_url,omitempty"`
	Error       string                       `json:"error,omitempty"`
	Payload     jogadorStatusCallbackPayload `json:"payload"`
}

func newJogadorStatusNotifier() jogadorStatusNotifier {
	return jogadorStatusNotifier{
		httpClient:    &http.Client{Timeout: 5 * time.Second},
		callbackURL:   strings.TrimSpace(os.Getenv("JOGADOR_STATUS_CALLBACK_URL")),
		callbackToken: strings.TrimSpace(os.Getenv("JOGADOR_STATUS_CALLBACK_TOKEN")),
	}
}

func (notifier jogadorStatusNotifier) NotifyStatusChange(ctx context.Context, agendamento models.Agendamento) jogadorNotificationResult {
	payload := jogadorStatusCallbackPayload{
		IDAgendamento:     agendamento.ID,
		IDCampo:           agendamento.IDCampo,
		IDArena:           agendamento.IDArena,
		Status:            string(agendamento.Status),
		OrigemAgendamento: string(agendamento.OrigemAgendamento),
		Horario:           agendamento.Horario.Format(time.RFC3339),
		NomeCampo:         agendamento.NomeCampo,
		NomeArena:         agendamento.NomeArena,
		ValorTotal:        agendamento.ValorTotal,
		ValorRestante:     agendamento.ValorRestante,
	}

	result := jogadorNotificationResult{
		CallbackURL: notifier.callbackURL,
		Payload:     payload,
	}

	if notifier.callbackURL == "" {
		return result
	}

	body, err := json.Marshal(payload)
	if err != nil {
		result.Attempted = true
		result.Error = fmt.Sprintf("erro ao serializar payload de notificacao: %v", err)
		return result
	}

	callbackCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(callbackCtx, http.MethodPost, notifier.callbackURL, bytes.NewReader(body))
	if err != nil {
		result.Attempted = true
		result.Error = fmt.Sprintf("erro ao criar requisicao de notificacao: %v", err)
		return result
	}

	req.Header.Set("Content-Type", "application/json")
	if notifier.callbackToken != "" {
		req.Header.Set("Authorization", "Bearer "+notifier.callbackToken)
	}

	resp, err := notifier.httpClient.Do(req)
	if err != nil {
		result.Attempted = true
		result.Error = fmt.Sprintf("erro ao enviar notificacao: %v", err)
		return result
	}
	defer resp.Body.Close()

	result.Attempted = true
	result.Delivered = resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices
	if !result.Delivered {
		result.Error = fmt.Sprintf("callback respondeu com status %d", resp.StatusCode)
	}

	return result
}
