package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/danpi/marca_ai_backend/internal/middleware"
	"github.com/danpi/marca_ai_backend/internal/models"
	"github.com/gorilla/mux"
)

type agendamentoCreateRequest struct {
	CampoID           int    `json:"campo_id"`
	IDCampo           int    `json:"id_campo"`
	Horario           string `json:"horario"`
	Jogadores         int    `json:"jogadores"`
	Pagamento         string `json:"pagamento"`
	Pago              bool   `json:"pago"`
	OrigemAgendamento string `json:"origem_agendamento"`
	Origem            string `json:"origem"`
	Time1             string `json:"time1"`
	Time2             string `json:"time2"`
	ModoDeJogo        string `json:"modo_de_jogo"`
}

type agendamentoStatusRequest struct {
	Status string `json:"status"`
}

type agendamentoResponse struct {
	ID                int     `json:"id"`
	IDUsuario         int     `json:"id_usuario,omitempty"`
	IDCampo           int     `json:"id_campo"`
	CampoID           int     `json:"campo_id"`
	IDArena           int     `json:"id_arena,omitempty"`
	Horario           string  `json:"horario"`
	Jogadores         int     `json:"jogadores"`
	Pagamento         string  `json:"pagamento"`
	Pago              bool    `json:"pago"`
	Status            string  `json:"status"`
	CriadoEm          string  `json:"criado_em,omitempty"`
	NomeCampo         string  `json:"nome_campo,omitempty"`
	NomeArena         string  `json:"nome_arena,omitempty"`
	OrigemAgendamento string  `json:"origem_agendamento"`
	ValorTotal        float64 `json:"valor_total"`
	ValorRestante     float64 `json:"valor_restante"`
	Time1             string  `json:"time1,omitempty"`
	Time2             string  `json:"time2,omitempty"`
	ModoDeJogo        string  `json:"modo_de_jogo,omitempty"`
}

func AgendarCampo(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Usuario nao autenticado", http.StatusUnauthorized)
		return
	}

	input, err := parseAgendamentoCreateRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	service := newAgendamentoService()
	agendamento, err := service.CreateManual(r.Context(), userID, input)
	if err != nil {
		writeAgendamentoServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"message":     "Agendamento realizado com sucesso",
		"agendamento": newAgendamentoResponse(agendamento),
	})
}

func CriarPedidoAgendamentoJogador(w http.ResponseWriter, r *http.Request) {
	if err := validateJogadorIntegrationRequest(r); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	input, err := parseAgendamentoCreateRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if input.OrigemAgendamento == "" {
		input.OrigemAgendamento = models.AgendamentoOrigemJogador
	}

	service := newAgendamentoService()
	agendamento, err := service.CreatePedidoExterno(r.Context(), input)
	if err != nil {
		writeAgendamentoServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"message":     "Pedido de agendamento recebido com sucesso",
		"agendamento": newAgendamentoResponse(agendamento),
	})
}

func GetAgendamentos(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Usuario nao autenticado", http.StatusUnauthorized)
		return
	}

	service := newAgendamentoService()
	agendamentos, err := service.ListByOwner(r.Context(), userID)
	if err != nil {
		http.Error(w, "Erro ao buscar agendamentos", http.StatusInternalServerError)
		return
	}

	response := make([]agendamentoResponse, 0, len(agendamentos))
	for _, agendamento := range agendamentos {
		response = append(response, newAgendamentoResponse(agendamento))
	}

	writeJSON(w, http.StatusOK, response)
}

func GetPedidos(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Usuario nao autenticado", http.StatusUnauthorized)
		return
	}

	service := newAgendamentoService()
	pedidos, err := service.ListPedidosByOwner(r.Context(), userID)
	if err != nil {
		http.Error(w, "Erro ao buscar pedidos", http.StatusInternalServerError)
		return
	}

	response := make([]agendamentoResponse, 0, len(pedidos))
	for _, pedido := range pedidos {
		response = append(response, newAgendamentoResponse(pedido))
	}

	writeJSON(w, http.StatusOK, response)
}

func AceitarPedido(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Usuario nao autenticado", http.StatusUnauthorized)
		return
	}

	agendamentoID, err := resolveAgendamentoID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	service := newAgendamentoService()
	result, err := service.AcceptPedido(r.Context(), userID, agendamentoID)
	if err != nil {
		writeAgendamentoServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message":     "Pedido aceito com sucesso",
		"agendamento": newAgendamentoResponse(result.Agendamento),
		"notificacao": result.Notificacao,
	})
}

func CancelarPedido(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Usuario nao autenticado", http.StatusUnauthorized)
		return
	}

	agendamentoID, err := resolveAgendamentoID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	service := newAgendamentoService()
	result, err := service.CancelPedido(r.Context(), userID, agendamentoID)
	if err != nil {
		writeAgendamentoServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message":     "Pedido cancelado com sucesso",
		"agendamento": newAgendamentoResponse(result.Agendamento),
		"notificacao": result.Notificacao,
	})
}

func AtualizarStatusAgendamento(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Usuario nao autenticado", http.StatusUnauthorized)
		return
	}

	agendamentoID, err := resolveAgendamentoID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var body agendamentoStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Erro ao decodificar JSON", http.StatusBadRequest)
		return
	}

	status, ok := models.NormalizeAgendamentoStatus(body.Status)
	if !ok {
		http.Error(w, "Status invalido", http.StatusBadRequest)
		return
	}

	service := newAgendamentoService()
	result, err := service.UpdateStatus(r.Context(), userID, agendamentoID, status)
	if err != nil {
		writeAgendamentoServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message":     "Status atualizado com sucesso",
		"agendamento": newAgendamentoResponse(result.Agendamento),
		"notificacao": result.Notificacao,
	})
}

func EditarAgendamento(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Usuario nao autenticado", http.StatusUnauthorized)
		return
	}

	agendamentoID, err := resolveAgendamentoID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	input, err := parseAgendamentoCreateRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	service := newAgendamentoService()
	agendamento, err := service.Edit(r.Context(), userID, agendamentoID, input)
	if err != nil {
		writeAgendamentoServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message":     "Agendamento atualizado com sucesso",
		"agendamento": newAgendamentoResponse(agendamento),
	})
}

func parseAgendamentoCreateRequest(r *http.Request) (models.CreateAgendamentoInput, error) {
	var request agendamentoCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return models.CreateAgendamentoInput{}, errors.New("Erro ao decodificar JSON")
	}

	campoID := request.CampoID
	if campoID <= 0 {
		campoID = request.IDCampo
	}
	if campoID <= 0 {
		return models.CreateAgendamentoInput{}, errors.New("Campo e obrigatorio")
	}

	horario, err := parseAgendamentoHorario(request.Horario)
	if err != nil {
		return models.CreateAgendamentoInput{}, errors.New("Formato de horario invalido")
	}

	origemRaw := strings.TrimSpace(request.OrigemAgendamento)
	if origemRaw == "" {
		origemRaw = strings.TrimSpace(request.Origem)
	}

	origem := models.AgendamentoOrigemManual
	if origemRaw != "" {
		normalizedOrigem, ok := models.NormalizeAgendamentoOrigem(origemRaw)
		if !ok {
			return models.CreateAgendamentoInput{}, errors.New("Origem do agendamento invalida")
		}
		origem = normalizedOrigem
	}

	return models.CreateAgendamentoInput{
		IDCampo:           campoID,
		Horario:           horario,
		Jogadores:         request.Jogadores,
		Pagamento:         strings.TrimSpace(request.Pagamento),
		Pago:              request.Pago,
		OrigemAgendamento: origem,
		Time1:             strings.TrimSpace(request.Time1),
		Time2:             strings.TrimSpace(request.Time2),
		ModoDeJogo:        strings.TrimSpace(request.ModoDeJogo),
	}, nil
}

func parseAgendamentoHorario(raw string) (time.Time, error) {
	location, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		return time.Time{}, err
	}

	raw = strings.TrimSpace(raw)
	layouts := []string{
		"2006-01-02T15:04",
		"2006-01-02 15:04",
		"2006-01-02T15:04:05",
		time.RFC3339,
		time.RFC3339Nano,
	}

	for _, layout := range layouts {
		switch layout {
		case time.RFC3339, time.RFC3339Nano:
			parsed, parseErr := time.Parse(layout, raw)
			if parseErr == nil {
				return parsed.In(location), nil
			}
		default:
			parsed, parseErr := time.ParseInLocation(layout, raw, location)
			if parseErr == nil {
				return parsed, nil
			}
		}
	}

	return time.Time{}, errors.New("horario invalido")
}

func resolveAgendamentoID(r *http.Request) (int, error) {
	vars := mux.Vars(r)
	if rawID, ok := vars["id"]; ok && strings.TrimSpace(rawID) != "" {
		id, err := strconv.Atoi(strings.TrimSpace(rawID))
		if err != nil || id <= 0 {
			return 0, errors.New("ID do agendamento invalido")
		}
		return id, nil
	}

	rawID := strings.TrimSpace(r.URL.Query().Get("id"))
	if rawID == "" {
		return 0, errors.New("ID do agendamento e obrigatorio")
	}

	id, err := strconv.Atoi(rawID)
	if err != nil || id <= 0 {
		return 0, errors.New("ID do agendamento invalido")
	}

	return id, nil
}

func validateJogadorIntegrationRequest(r *http.Request) error {
	expectedToken := strings.TrimSpace(os.Getenv("JOGADOR_INTEGRATION_TOKEN"))
	if expectedToken == "" {
		return nil
	}

	receivedToken := strings.TrimSpace(r.Header.Get("X-Integration-Token"))
	if receivedToken == "" {
		authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
		if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
			receivedToken = strings.TrimSpace(authHeader[7:])
		}
	}

	if receivedToken == "" || receivedToken != expectedToken {
		return errors.New("Token de integracao invalido")
	}

	return nil
}

func newAgendamentoResponse(agendamento models.Agendamento) agendamentoResponse {
	response := agendamentoResponse{
		ID:                agendamento.ID,
		IDUsuario:         agendamento.IDUsuario,
		IDCampo:           agendamento.IDCampo,
		CampoID:           agendamento.IDCampo,
		IDArena:           agendamento.IDArena,
		Horario:           agendamento.Horario.Format(time.RFC3339),
		Jogadores:         agendamento.Jogadores,
		Pagamento:         agendamento.Pagamento,
		Pago:              agendamento.Pago,
		Status:            string(agendamento.Status),
		NomeCampo:         agendamento.NomeCampo,
		NomeArena:         agendamento.NomeArena,
		OrigemAgendamento: string(agendamento.OrigemAgendamento),
		ValorTotal:        agendamento.ValorTotal,
		ValorRestante:     agendamento.ValorRestante,
		Time1:             agendamento.Time1,
		Time2:             agendamento.Time2,
		ModoDeJogo:        agendamento.ModoDeJogo,
	}

	if !agendamento.CriadoEm.IsZero() {
		response.CriadoEm = agendamento.CriadoEm.Format(time.RFC3339)
	}

	return response
}

func writeAgendamentoServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errAgendamentoCampoNaoEncontrado):
		http.Error(w, "Campo nao encontrado", http.StatusNotFound)
	case errors.Is(err, errAgendamentoNaoEncontrado):
		http.Error(w, "Agendamento nao encontrado", http.StatusNotFound)
	case errors.Is(err, errAgendamentoCampoSemPermissao):
		http.Error(w, "Campo nao pertence ao usuario logado", http.StatusForbidden)
	case errors.Is(err, errAgendamentoHorarioIndisponivel):
		http.Error(w, "Este horario ja esta reservado para o campo selecionado.", http.StatusConflict)
	case errors.Is(err, errAgendamentoCampoIndisponivel):
		http.Error(w, "O campo selecionado esta indisponivel para agendamento", http.StatusConflict)
	case errors.Is(err, errAgendamentoJogadoresInvalidos):
		http.Error(w, "Quantidade de jogadores invalida para o campo selecionado", http.StatusBadRequest)
	case errors.Is(err, errAgendamentoOrigemInvalida):
		http.Error(w, "Origem do agendamento invalida", http.StatusBadRequest)
	case errors.Is(err, errAgendamentoStatusInvalido):
		http.Error(w, "Status do agendamento invalido", http.StatusBadRequest)
	case errors.Is(err, errAgendamentoPedidoNaoPendente):
		http.Error(w, "O pedido informado nao esta com status pendente", http.StatusBadRequest)
	default:
		http.Error(w, "Erro interno ao processar agendamento", http.StatusInternalServerError)
	}
}
