package handlers

import (
	"encoding/json"
	"errors"
	"io"
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
	IDUsuarioJogador  *int   `json:"id_usuario_jogador"`
	IDJogador         *int   `json:"id_jogador"`
	IDUsuario         *int   `json:"id_usuario"`
	NomeSolicitante   string `json:"nome_solicitante"`
	OrigemAgendamento string `json:"origem_agendamento"`
	Origem            string `json:"origem"`
	Time1             string `json:"time1"`
	Time2             string `json:"time2"`
	ModoDeJogo        string `json:"modo_de_jogo"`
}

type agendamentoStatusRequest struct {
	Status string `json:"status"`
}

type agendamentoPagamentoRequest struct {
	IDUsuario      *int    `json:"id_usuario"`
	ValorPago      float64 `json:"valor_pago"`
	FormaPagamento string  `json:"forma_pagamento"`
}

type agendamentoResponse struct {
	ID                int     `json:"id"`
	IDUsuario         int     `json:"id_usuario,omitempty"`
	IDCampo           int     `json:"id_campo"`
	CampoID           int     `json:"campo_id"`
	IDArena           int     `json:"id_arena,omitempty"`
	NomeSolicitante   string  `json:"nome_solicitante,omitempty"`
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
	StatusDePagamento bool    `json:"status_de_pagamento"`
	InicioCronometro  *int64  `json:"inicio_cronometro,omitempty"`
	FimCronometro     string  `json:"fim_cronometro,omitempty"`
	Time1             string  `json:"time1,omitempty"`
	Time2             string  `json:"time2,omitempty"`
	ModoDeJogo        string  `json:"modo_de_jogo,omitempty"`
}

type agendamentoPagamentoResponse struct {
	ID               int     `json:"id"`
	IDAgendamento    int     `json:"id_agendamento"`
	IDUsuario        *int    `json:"id_usuario,omitempty"`
	ValorPago        float64 `json:"valor_pago"`
	FormaPagamento   string  `json:"forma_pagamento"`
	DataPagamento    string  `json:"data_pagamento"`
	NomeUsuario      string  `json:"nome_usuario,omitempty"`
	SobrenomeUsuario string  `json:"sobrenome_usuario,omitempty"`
	EmailUsuario     string  `json:"email_usuario,omitempty"`
}

type agendamentoPagamentosResumoResponse struct {
	Agendamento agendamentoResponse            `json:"agendamento"`
	Pagamentos  []agendamentoPagamentoResponse `json:"pagamentos"`
	TotalPago   float64                        `json:"total_pago"`
}

func formatAgendamentoDateTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}

	return value.Format("2006-01-02T15:04:05")
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

func IniciarCronometroAgendamento(w http.ResponseWriter, r *http.Request) {
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
	agendamento, err := service.IniciarCronometro(r.Context(), userID, agendamentoID)
	if err != nil {
		writeAgendamentoServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message":     "Cronometro iniciado com sucesso",
		"agendamento": newAgendamentoResponse(agendamento),
	})
}

func EncerrarCronometroAgendamento(w http.ResponseWriter, r *http.Request) {
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
	agendamento, err := service.EncerrarCronometro(r.Context(), userID, agendamentoID)
	if err != nil {
		writeAgendamentoServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message":     "Cronometro encerrado com sucesso",
		"agendamento": newAgendamentoResponse(agendamento),
	})
}

func GetPagamentosAgendamento(w http.ResponseWriter, r *http.Request) {
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
	resumo, err := service.GetPagamentosResumo(r.Context(), userID, agendamentoID)
	if err != nil {
		writeAgendamentoServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newAgendamentoPagamentosResumoResponse(resumo))
}

func RegistrarPagamentoParcialAgendamento(w http.ResponseWriter, r *http.Request) {
	handlePagamentoAgendamento(w, r, false)
}

func RegistrarPagamentoTotalAgendamento(w http.ResponseWriter, r *http.Request) {
	handlePagamentoAgendamento(w, r, true)
}

func ConcluirAgendamento(w http.ResponseWriter, r *http.Request) {
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
	result, err := service.Concluir(r.Context(), userID, agendamentoID)
	if err != nil {
		writeAgendamentoServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message":     "Agendamento concluido com sucesso",
		"agendamento": newAgendamentoResponse(result.Agendamento),
		"notificacao": result.Notificacao,
	})
}

func handlePagamentoAgendamento(w http.ResponseWriter, r *http.Request, pagamentoTotal bool) {
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

	input, err := parsePagamentoAgendamentoRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	service := newAgendamentoService()
	var result agendamentoPagamentoMutationResult
	if pagamentoTotal {
		result, err = service.RegistrarPagamentoTotal(r.Context(), userID, agendamentoID, input)
	} else {
		result, err = service.RegistrarPagamentoParcial(r.Context(), userID, agendamentoID, input)
	}
	if err != nil {
		writeAgendamentoServiceError(w, err)
		return
	}

	message := "Pagamento parcial registrado com sucesso"
	if pagamentoTotal {
		message = "Pagamento total registrado com sucesso"
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"message":     message,
		"agendamento": newAgendamentoResponse(result.Agendamento),
		"pagamento":   newAgendamentoPagamentoResponse(result.Pagamento),
		"total_pago":  result.TotalPago,
	})
}

func parseAgendamentoCreateRequest(r *http.Request) (models.CreateAgendamentoInput, error) {
	var request agendamentoCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		if errors.Is(err, io.EOF) {
			request = agendamentoCreateRequestFromQuery(r)
		} else {
			return models.CreateAgendamentoInput{}, errors.New("Erro ao decodificar JSON")
		}
	}

	if request.CampoID <= 0 && request.IDCampo <= 0 {
		queryRequest := agendamentoCreateRequestFromQuery(r)
		request.CampoID = queryRequest.CampoID
		request.IDCampo = queryRequest.IDCampo
	}

	if request.Horario == "" {
		request.Horario = strings.TrimSpace(r.URL.Query().Get("horario"))
	}

	if request.Jogadores <= 0 {
		request.Jogadores, _ = strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("jogadores")))
	}

	if request.NomeSolicitante == "" {
		request.NomeSolicitante = strings.TrimSpace(r.URL.Query().Get("nome_solicitante"))
	}

	if request.Pagamento == "" {
		request.Pagamento = strings.TrimSpace(r.URL.Query().Get("pagamento"))
	}

	if !request.Pago {
		request.Pago, _ = strconv.ParseBool(strings.TrimSpace(r.URL.Query().Get("pago")))
	}

	if request.IDUsuarioJogador == nil && request.IDJogador == nil && request.IDUsuario == nil {
		request.IDUsuarioJogador = optionalPositiveIntFromQuery(r, "id_usuario_jogador")
		request.IDJogador = optionalPositiveIntFromQuery(r, "id_jogador")
		request.IDUsuario = optionalPositiveIntFromQuery(r, "id_usuario")
	}

	if request.OrigemAgendamento == "" {
		request.OrigemAgendamento = strings.TrimSpace(r.URL.Query().Get("origem_agendamento"))
	}

	if request.Origem == "" {
		request.Origem = strings.TrimSpace(r.URL.Query().Get("origem"))
	}

	if request.Time1 == "" {
		request.Time1 = strings.TrimSpace(r.URL.Query().Get("time1"))
	}

	if request.Time2 == "" {
		request.Time2 = strings.TrimSpace(r.URL.Query().Get("time2"))
	}

	if request.ModoDeJogo == "" {
		request.ModoDeJogo = strings.TrimSpace(r.URL.Query().Get("modo_de_jogo"))
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

	jogadorID, err := resolveAgendamentoJogadorID(request)
	if err != nil {
		return models.CreateAgendamentoInput{}, err
	}

	origemRaw := strings.TrimSpace(request.OrigemAgendamento)
	if origemRaw == "" {
		origemRaw = strings.TrimSpace(request.Origem)
	}

	var origem models.AgendamentoOrigem
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
		IDUsuarioJogador:  jogadorID,
		NomeSolicitante:   strings.TrimSpace(request.NomeSolicitante),
		OrigemAgendamento: origem,
		Time1:             strings.TrimSpace(request.Time1),
		Time2:             strings.TrimSpace(request.Time2),
		ModoDeJogo:        strings.TrimSpace(request.ModoDeJogo),
	}, nil
}

func agendamentoCreateRequestFromQuery(r *http.Request) agendamentoCreateRequest {
	query := r.URL.Query()
	campoID, _ := strconv.Atoi(strings.TrimSpace(query.Get("campo_id")))
	idCampo, _ := strconv.Atoi(strings.TrimSpace(query.Get("id_campo")))
	jogadores, _ := strconv.Atoi(strings.TrimSpace(query.Get("jogadores")))
	pago, _ := strconv.ParseBool(strings.TrimSpace(query.Get("pago")))

	return agendamentoCreateRequest{
		CampoID:           campoID,
		IDCampo:           idCampo,
		Horario:           strings.TrimSpace(query.Get("horario")),
		Jogadores:         jogadores,
		Pagamento:         strings.TrimSpace(query.Get("pagamento")),
		Pago:              pago,
		IDUsuarioJogador:  optionalPositiveIntFromQuery(r, "id_usuario_jogador"),
		IDJogador:         optionalPositiveIntFromQuery(r, "id_jogador"),
		IDUsuario:         optionalPositiveIntFromQuery(r, "id_usuario"),
		NomeSolicitante:   strings.TrimSpace(query.Get("nome_solicitante")),
		OrigemAgendamento: strings.TrimSpace(query.Get("origem_agendamento")),
		Origem:            strings.TrimSpace(query.Get("origem")),
		Time1:             strings.TrimSpace(query.Get("time1")),
		Time2:             strings.TrimSpace(query.Get("time2")),
		ModoDeJogo:        strings.TrimSpace(query.Get("modo_de_jogo")),
	}
}

func optionalPositiveIntFromQuery(r *http.Request, key string) *int {
	value, err := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get(key)))
	if err != nil || value <= 0 {
		return nil
	}

	return &value
}

func resolveAgendamentoJogadorID(request agendamentoCreateRequest) (*int, error) {
	candidates := []*int{request.IDUsuarioJogador, request.IDJogador, request.IDUsuario}

	for _, candidate := range candidates {
		if candidate == nil {
			continue
		}
		if *candidate <= 0 {
			return nil, errors.New("ID do jogador invalido")
		}

		jogadorID := *candidate
		return &jogadorID, nil
	}

	return nil, nil
}

func parsePagamentoAgendamentoRequest(r *http.Request) (models.RegistrarPagamentoInput, error) {
	var request agendamentoPagamentoRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		if errors.Is(err, io.EOF) {
			return models.RegistrarPagamentoInput{}, nil
		}
		return models.RegistrarPagamentoInput{}, errors.New("Erro ao decodificar JSON")
	}

	return models.RegistrarPagamentoInput{
		IDUsuario:      request.IDUsuario,
		ValorPago:      request.ValorPago,
		FormaPagamento: strings.TrimSpace(request.FormaPagamento),
	}, nil
}

func parseAgendamentoHorario(raw string) (time.Time, error) {
	location := agendamentoLocation()

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
		NomeSolicitante:   agendamento.NomeSolicitante,
		Horario:           formatAgendamentoDateTime(agendamento.Horario),
		Jogadores:         agendamento.Jogadores,
		Pagamento:         agendamento.Pagamento,
		Pago:              agendamento.Pago,
		Status:            string(agendamento.Status),
		NomeCampo:         agendamento.NomeCampo,
		NomeArena:         agendamento.NomeArena,
		OrigemAgendamento: string(agendamento.OrigemAgendamento),
		ValorTotal:        agendamento.ValorTotal,
		ValorRestante:     agendamento.ValorRestante,
		StatusDePagamento: agendamento.StatusDePagamento,
		InicioCronometro:  agendamento.InicioCronometro,
		Time1:             agendamento.Time1,
		Time2:             agendamento.Time2,
		ModoDeJogo:        agendamento.ModoDeJogo,
	}

	if !agendamento.CriadoEm.IsZero() {
		response.CriadoEm = formatAgendamentoDateTime(agendamento.CriadoEm)
	}
	if agendamento.FimCronometro != nil && !agendamento.FimCronometro.IsZero() {
		response.FimCronometro = formatAgendamentoDateTime(*agendamento.FimCronometro)
	}

	return response
}

func newAgendamentoPagamentoResponse(pagamento models.AgendamentoPagamento) agendamentoPagamentoResponse {
	return agendamentoPagamentoResponse{
		ID:               pagamento.ID,
		IDAgendamento:    pagamento.IDAgendamento,
		IDUsuario:        pagamento.IDUsuario,
		ValorPago:        pagamento.ValorPago,
		FormaPagamento:   pagamento.FormaPagamento,
		DataPagamento:    formatAgendamentoDateTime(pagamento.DataPagamento),
		NomeUsuario:      pagamento.NomeUsuario,
		SobrenomeUsuario: pagamento.SobrenomeUsuario,
		EmailUsuario:     pagamento.EmailUsuario,
	}
}

func newAgendamentoPagamentosResumoResponse(resumo models.AgendamentoPagamentosResumo) agendamentoPagamentosResumoResponse {
	pagamentos := make([]agendamentoPagamentoResponse, 0, len(resumo.Pagamentos))
	for _, pagamento := range resumo.Pagamentos {
		pagamentos = append(pagamentos, newAgendamentoPagamentoResponse(pagamento))
	}

	return agendamentoPagamentosResumoResponse{
		Agendamento: newAgendamentoResponse(resumo.Agendamento),
		Pagamentos:  pagamentos,
		TotalPago:   resumo.TotalPago,
	}
}

func writeAgendamentoServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errAgendamentoCampoNaoEncontrado):
		http.Error(w, "Campo nao encontrado", http.StatusNotFound)
	case errors.Is(err, errAgendamentoJogadorNaoEncontrado):
		http.Error(w, "Jogador nao encontrado", http.StatusNotFound)
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
	case errors.Is(err, errAgendamentoCronometroNaoIniciado):
		http.Error(w, "O cronometro ainda nao foi iniciado", http.StatusBadRequest)
	case errors.Is(err, errAgendamentoCronometroNaoEncerrado):
		http.Error(w, "O cronometro precisa ser encerrado antes da conclusao", http.StatusBadRequest)
	case errors.Is(err, errAgendamentoConclusaoBloqueada):
		http.Error(w, "O agendamento ainda possui saldo pendente", http.StatusBadRequest)
	case errors.Is(err, errAgendamentoPagamentoInvalido):
		http.Error(w, "Pagamento invalido", http.StatusBadRequest)
	case errors.Is(err, errAgendamentoSemSaldoPendente):
		http.Error(w, "O agendamento nao possui saldo pendente", http.StatusBadRequest)
	case errors.Is(err, errAgendamentoEstadoOperacaoInvalido):
		http.Error(w, "O estado atual do agendamento nao permite esta operacao", http.StatusBadRequest)
	default:
		http.Error(w, "Erro interno ao processar agendamento", http.StatusInternalServerError)
	}
}
