package handlers

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/danpi/marca_ai_backend/internal/models"
)

var (
	errAgendamentoCampoNaoEncontrado     = errors.New("campo nao encontrado")
	errAgendamentoCampoSemPermissao      = errors.New("campo nao pertence ao usuario")
	errAgendamentoNaoEncontrado          = errors.New("agendamento nao encontrado")
	errAgendamentoJogadorNaoEncontrado   = errors.New("jogador nao encontrado")
	errAgendamentoOrigemInvalida         = errors.New("origem do agendamento invalida")
	errAgendamentoStatusInvalido         = errors.New("status do agendamento invalido")
	errAgendamentoHorarioIndisponivel    = errors.New("horario indisponivel")
	errAgendamentoCampoIndisponivel      = errors.New("campo indisponivel para agendamento")
	errAgendamentoJogadoresInvalidos     = errors.New("quantidade de jogadores invalida")
	errAgendamentoPedidoNaoPendente      = errors.New("pedido nao esta pendente")
	errAgendamentoCronometroNaoIniciado  = errors.New("cronometro nao iniciado")
	errAgendamentoCronometroNaoEncerrado = errors.New("cronometro nao encerrado")
	errAgendamentoConclusaoBloqueada     = errors.New("agendamento ainda possui saldo pendente")
	errAgendamentoPagamentoInvalido      = errors.New("pagamento invalido")
	errAgendamentoSemSaldoPendente       = errors.New("agendamento nao possui saldo pendente")
	errAgendamentoEstadoOperacaoInvalido = errors.New("estado atual do agendamento nao permite esta operacao")
)

type agendamentoMutationResult struct {
	Agendamento models.Agendamento        `json:"agendamento"`
	Notificacao jogadorNotificationResult `json:"notificacao"`
}

type agendamentoPagamentoMutationResult struct {
	Agendamento models.Agendamento          `json:"agendamento"`
	Pagamento   models.AgendamentoPagamento `json:"pagamento"`
	TotalPago   float64                     `json:"total_pago"`
}

type agendamentoService struct {
	repository agendamentoRepository
	notifier   jogadorStatusNotifier
}

func newAgendamentoService() agendamentoService {
	return agendamentoService{
		repository: newAgendamentoRepository(),
		notifier:   newJogadorStatusNotifier(),
	}
}

func (service agendamentoService) CreateManual(ctx context.Context, ownerUserID int, input models.CreateAgendamentoInput) (models.Agendamento, error) {
	input.OrigemAgendamento = models.AgendamentoOrigemManual
	input.IDUsuario = &ownerUserID

	return service.create(ctx, input, ownerUserID, models.AgendamentoStatusAgendado)
}

func (service agendamentoService) CreatePedidoExterno(ctx context.Context, input models.CreateAgendamentoInput) (models.Agendamento, error) {
	switch input.OrigemAgendamento {
	case models.AgendamentoOrigemJogador, models.AgendamentoOrigemTimeXTime:
	default:
		return models.Agendamento{}, errAgendamentoOrigemInvalida
	}

	input.IDUsuario = nil
	input, err := resolveNomeSolicitanteFromJogador(ctx, input, service.repository.loadJogadorNomeByID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Agendamento{}, errAgendamentoJogadorNaoEncontrado
		}
		return models.Agendamento{}, err
	}

	return service.create(ctx, input, 0, models.AgendamentoStatusPedido)
}

func (service agendamentoService) ListByOwner(ctx context.Context, ownerUserID int) ([]models.Agendamento, error) {
	return service.repository.listByOwner(ctx, ownerUserID, nil)
}

func (service agendamentoService) ListPedidosByOwner(ctx context.Context, ownerUserID int) ([]models.Agendamento, error) {
	status := models.AgendamentoStatusPedido
	return service.repository.listByOwner(ctx, ownerUserID, &status)
}

func (service agendamentoService) Edit(ctx context.Context, ownerUserID int, agendamentoID int, input models.CreateAgendamentoInput) (models.Agendamento, error) {
	agendamentoAtual, err := service.repository.getByIDForOwner(ctx, agendamentoID, ownerUserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Agendamento{}, errAgendamentoNaoEncontrado
		}
		return models.Agendamento{}, err
	}

	campo, err := service.validateCampoAndSchedule(ctx, input, ownerUserID, &agendamentoID)
	if err != nil {
		return models.Agendamento{}, err
	}

	totalPago, err := service.repository.sumPayments(ctx, agendamentoID)
	if err != nil {
		return models.Agendamento{}, err
	}

	valorTotal := campo.ValorHora
	valorRestante, pago, statusDePagamento := resolveFinancialState(
		valorTotal,
		totalPago,
		agendamentoAtual.Pago || input.Pago,
		agendamentoAtual.StatusDePagamento || input.Pago,
	)

	err = service.repository.update(ctx, agendamentoID, agendamentoUpdateInput{
		IDCampo:         input.IDCampo,
		Horario:         input.Horario,
		Jogadores:       input.Jogadores,
		Pagamento:       input.Pagamento,
		Pago:            pago,
		NomeSolicitante: input.NomeSolicitante,
		ValorTotal:      valorTotal,
		ValorRestante:   valorRestante,
	})
	if err != nil {
		return models.Agendamento{}, err
	}

	agendamentoAtual.IDCampo = input.IDCampo
	agendamentoAtual.IDArena = campo.IDArena
	agendamentoAtual.Horario = input.Horario
	agendamentoAtual.Jogadores = input.Jogadores
	agendamentoAtual.Pagamento = input.Pagamento
	agendamentoAtual.Pago = pago
	agendamentoAtual.NomeSolicitante = input.NomeSolicitante
	agendamentoAtual.StatusDePagamento = statusDePagamento
	agendamentoAtual.NomeCampo = campo.NomeCampo
	agendamentoAtual.NomeArena = campo.NomeArena
	agendamentoAtual.ValorTotal = valorTotal
	agendamentoAtual.ValorRestante = valorRestante

	return agendamentoAtual, nil
}

func (service agendamentoService) UpdateStatus(ctx context.Context, ownerUserID int, agendamentoID int, status models.AgendamentoStatus) (agendamentoMutationResult, error) {
	if status == models.AgendamentoStatusConcluido {
		return service.Concluir(ctx, ownerUserID, agendamentoID)
	}

	switch status {
	case models.AgendamentoStatusAgendado, models.AgendamentoStatusCancelado, models.AgendamentoStatusPedido:
	default:
		return agendamentoMutationResult{}, errAgendamentoStatusInvalido
	}

	agendamento, err := service.repository.getByIDForOwner(ctx, agendamentoID, ownerUserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return agendamentoMutationResult{}, errAgendamentoNaoEncontrado
		}
		return agendamentoMutationResult{}, err
	}

	return service.transitionStatus(ctx, agendamento, status)
}

func (service agendamentoService) AcceptPedido(ctx context.Context, ownerUserID int, agendamentoID int) (agendamentoMutationResult, error) {
	agendamento, err := service.repository.getByIDForOwner(ctx, agendamentoID, ownerUserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return agendamentoMutationResult{}, errAgendamentoNaoEncontrado
		}
		return agendamentoMutationResult{}, err
	}

	if agendamento.Status != models.AgendamentoStatusPedido {
		return agendamentoMutationResult{}, errAgendamentoPedidoNaoPendente
	}

	return service.transitionStatus(ctx, agendamento, models.AgendamentoStatusAgendado)
}

func (service agendamentoService) CancelPedido(ctx context.Context, ownerUserID int, agendamentoID int) (agendamentoMutationResult, error) {
	agendamento, err := service.repository.getByIDForOwner(ctx, agendamentoID, ownerUserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return agendamentoMutationResult{}, errAgendamentoNaoEncontrado
		}
		return agendamentoMutationResult{}, err
	}

	if agendamento.Status != models.AgendamentoStatusPedido {
		return agendamentoMutationResult{}, errAgendamentoPedidoNaoPendente
	}

	return service.transitionStatus(ctx, agendamento, models.AgendamentoStatusCancelado)
}

func (service agendamentoService) IniciarCronometro(ctx context.Context, ownerUserID int, agendamentoID int) (models.Agendamento, error) {
	agendamento, err := service.repository.getByIDForOwner(ctx, agendamentoID, ownerUserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Agendamento{}, errAgendamentoNaoEncontrado
		}
		return models.Agendamento{}, err
	}

	if !canManageExecutionState(agendamento.Status) {
		return models.Agendamento{}, errAgendamentoEstadoOperacaoInvalido
	}

	inicioUnix := agendamentoNow().Unix()
	if err := service.repository.startCronometro(ctx, agendamentoID, inicioUnix); err != nil {
		return models.Agendamento{}, err
	}

	agendamento.Status = models.AgendamentoStatusEmAndamento
	agendamento.InicioCronometro = &inicioUnix
	agendamento.FimCronometro = nil
	return agendamento, nil
}

func (service agendamentoService) EncerrarCronometro(ctx context.Context, ownerUserID int, agendamentoID int) (models.Agendamento, error) {
	agendamento, err := service.repository.getByIDForOwner(ctx, agendamentoID, ownerUserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Agendamento{}, errAgendamentoNaoEncontrado
		}
		return models.Agendamento{}, err
	}

	if !canManageExecutionState(agendamento.Status) {
		return models.Agendamento{}, errAgendamentoEstadoOperacaoInvalido
	}
	if agendamento.InicioCronometro == nil {
		return models.Agendamento{}, errAgendamentoCronometroNaoIniciado
	}

	agendamento, _, err = service.refreshFinancialState(ctx, agendamento)
	if err != nil {
		return models.Agendamento{}, err
	}

	nextStatus := statusAfterCronometroEncerrado(agendamento.ValorRestante)
	fim := agendamentoNow()
	if err := service.repository.finishCronometro(ctx, agendamentoID, fim, nextStatus); err != nil {
		return models.Agendamento{}, err
	}

	agendamento.Status = nextStatus
	agendamento.FimCronometro = &fim
	return agendamento, nil
}

func (service agendamentoService) GetPagamentosResumo(ctx context.Context, ownerUserID int, agendamentoID int) (models.AgendamentoPagamentosResumo, error) {
	agendamento, err := service.repository.getByIDForOwner(ctx, agendamentoID, ownerUserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.AgendamentoPagamentosResumo{}, errAgendamentoNaoEncontrado
		}
		return models.AgendamentoPagamentosResumo{}, err
	}

	agendamento, totalPago, err := service.refreshFinancialState(ctx, agendamento)
	if err != nil {
		return models.AgendamentoPagamentosResumo{}, err
	}

	pagamentos, err := service.repository.listPayments(ctx, agendamentoID)
	if err != nil {
		return models.AgendamentoPagamentosResumo{}, err
	}

	return models.AgendamentoPagamentosResumo{
		Agendamento: agendamento,
		Pagamentos:  pagamentos,
		TotalPago:   totalPago,
	}, nil
}

func (service agendamentoService) RegistrarPagamentoParcial(ctx context.Context, ownerUserID int, agendamentoID int, input models.RegistrarPagamentoInput) (agendamentoPagamentoMutationResult, error) {
	if input.ValorPago <= 0 {
		return agendamentoPagamentoMutationResult{}, errAgendamentoPagamentoInvalido
	}

	agendamento, err := service.repository.getByIDForOwner(ctx, agendamentoID, ownerUserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return agendamentoPagamentoMutationResult{}, errAgendamentoNaoEncontrado
		}
		return agendamentoPagamentoMutationResult{}, err
	}

	if !canRegisterPayment(agendamento.Status) {
		return agendamentoPagamentoMutationResult{}, errAgendamentoEstadoOperacaoInvalido
	}

	agendamento, totalPagoAtual, err := service.refreshFinancialState(ctx, agendamento)
	if err != nil {
		return agendamentoPagamentoMutationResult{}, err
	}

	if agendamento.ValorRestante <= 0 {
		return agendamentoPagamentoMutationResult{}, errAgendamentoSemSaldoPendente
	}
	if input.ValorPago > agendamento.ValorRestante {
		return agendamentoPagamentoMutationResult{}, errAgendamentoPagamentoInvalido
	}

	input.FormaPagamento = sanitizePagamento(input.FormaPagamento)
	if input.FormaPagamento == "" {
		input.FormaPagamento = sanitizePagamento(agendamento.Pagamento)
	}
	pagamento, err := service.repository.insertPayment(ctx, agendamentoID, input)
	if err != nil {
		return agendamentoPagamentoMutationResult{}, err
	}

	totalPago := totalPagoAtual + input.ValorPago
	valorRestante, pago, statusDePagamento := resolveFinancialState(
		agendamento.ValorTotal,
		totalPago,
		agendamento.Pago,
		agendamento.StatusDePagamento,
	)

	statusUpdate := statusAfterPayment(agendamento, valorRestante)
	if err := service.repository.updateFinancialState(ctx, agendamentoID, agendamentoFinancialUpdate{
		ValorRestante:     valorRestante,
		Pago:              pago,
		StatusDePagamento: statusDePagamento,
		Status:            statusUpdate,
	}); err != nil {
		return agendamentoPagamentoMutationResult{}, err
	}

	agendamento.ValorRestante = valorRestante
	agendamento.Pago = pago
	agendamento.StatusDePagamento = statusDePagamento
	if statusUpdate != nil {
		agendamento.Status = *statusUpdate
	}

	return agendamentoPagamentoMutationResult{
		Agendamento: agendamento,
		Pagamento:   pagamento,
		TotalPago:   totalPago,
	}, nil
}

func (service agendamentoService) RegistrarPagamentoTotal(ctx context.Context, ownerUserID int, agendamentoID int, input models.RegistrarPagamentoInput) (agendamentoPagamentoMutationResult, error) {
	agendamento, err := service.repository.getByIDForOwner(ctx, agendamentoID, ownerUserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return agendamentoPagamentoMutationResult{}, errAgendamentoNaoEncontrado
		}
		return agendamentoPagamentoMutationResult{}, err
	}

	if !canRegisterPayment(agendamento.Status) {
		return agendamentoPagamentoMutationResult{}, errAgendamentoEstadoOperacaoInvalido
	}

	agendamento, _, err = service.refreshFinancialState(ctx, agendamento)
	if err != nil {
		return agendamentoPagamentoMutationResult{}, err
	}

	if agendamento.ValorRestante <= 0 {
		return agendamentoPagamentoMutationResult{}, errAgendamentoSemSaldoPendente
	}

	input.ValorPago = agendamento.ValorRestante
	return service.RegistrarPagamentoParcial(ctx, ownerUserID, agendamentoID, input)
}

func (service agendamentoService) Concluir(ctx context.Context, ownerUserID int, agendamentoID int) (agendamentoMutationResult, error) {
	agendamento, err := service.repository.getByIDForOwner(ctx, agendamentoID, ownerUserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return agendamentoMutationResult{}, errAgendamentoNaoEncontrado
		}
		return agendamentoMutationResult{}, err
	}

	if agendamento.FimCronometro == nil {
		return agendamentoMutationResult{}, errAgendamentoCronometroNaoEncerrado
	}

	agendamento, _, err = service.refreshFinancialState(ctx, agendamento)
	if err != nil {
		return agendamentoMutationResult{}, err
	}

	if agendamento.ValorRestante > 0 {
		return agendamentoMutationResult{}, errAgendamentoConclusaoBloqueada
	}

	return service.transitionStatus(ctx, agendamento, models.AgendamentoStatusConcluido)
}

func (service agendamentoService) create(ctx context.Context, input models.CreateAgendamentoInput, ownerUserID int, status models.AgendamentoStatus) (models.Agendamento, error) {
	campo, err := service.validateCampoAndSchedule(ctx, input, ownerUserID, nil)
	if err != nil {
		return models.Agendamento{}, err
	}

	valorTotal := campo.ValorHora
	valorRestante, pago, statusDePagamento := resolveFinancialState(valorTotal, 0, input.Pago, input.Pago)

	agendamento, err := service.repository.create(ctx, input, status, valorTotal, valorRestante)
	if err != nil {
		return models.Agendamento{}, err
	}

	agendamento.IDArena = campo.IDArena
	agendamento.NomeCampo = campo.NomeCampo
	agendamento.NomeArena = campo.NomeArena
	agendamento.Pago = pago
	agendamento.StatusDePagamento = statusDePagamento
	return agendamento, nil
}

func resolveNomeSolicitanteFromJogador(
	ctx context.Context,
	input models.CreateAgendamentoInput,
	loadJogadorNome func(context.Context, int) (string, error),
) (models.CreateAgendamentoInput, error) {
	if input.OrigemAgendamento != models.AgendamentoOrigemJogador || input.IDUsuarioJogador == nil {
		return input, nil
	}

	nomeSolicitante, err := loadJogadorNome(ctx, *input.IDUsuarioJogador)
	if err != nil {
		return models.CreateAgendamentoInput{}, err
	}
	if nomeSolicitante != "" {
		input.NomeSolicitante = nomeSolicitante
	}

	return input, nil
}

func (service agendamentoService) validateCampoAndSchedule(ctx context.Context, input models.CreateAgendamentoInput, ownerUserID int, excludeAgendamentoID *int) (campoAgendamentoSnapshot, error) {
	if input.IDCampo <= 0 {
		return campoAgendamentoSnapshot{}, errAgendamentoCampoNaoEncontrado
	}
	if input.Jogadores <= 0 {
		return campoAgendamentoSnapshot{}, errAgendamentoJogadoresInvalidos
	}

	campo, err := service.repository.loadCampoSnapshot(ctx, input.IDCampo)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return campoAgendamentoSnapshot{}, errAgendamentoCampoNaoEncontrado
		}
		return campoAgendamentoSnapshot{}, err
	}

	if ownerUserID > 0 && campo.OwnerUserID != ownerUserID {
		return campoAgendamentoSnapshot{}, errAgendamentoCampoSemPermissao
	}
	if campo.CampoEmManutencao || campo.ArenaEmManutencao {
		return campoAgendamentoSnapshot{}, errAgendamentoCampoIndisponivel
	}
	if campo.MaxJogadores > 0 && input.Jogadores > campo.MaxJogadores {
		return campoAgendamentoSnapshot{}, errAgendamentoJogadoresInvalidos
	}

	conflict, err := service.repository.hasScheduleConflict(ctx, input.IDCampo, input.Horario, excludeAgendamentoID)
	if err != nil {
		return campoAgendamentoSnapshot{}, err
	}
	if conflict {
		return campoAgendamentoSnapshot{}, errAgendamentoHorarioIndisponivel
	}

	return campo, nil
}

func (service agendamentoService) transitionStatus(ctx context.Context, agendamento models.Agendamento, status models.AgendamentoStatus) (agendamentoMutationResult, error) {
	if agendamento.Status == status {
		return agendamentoMutationResult{Agendamento: agendamento}, nil
	}

	if err := service.repository.updateStatus(ctx, agendamento.ID, status); err != nil {
		return agendamentoMutationResult{}, err
	}

	agendamento.Status = status
	notificacao := jogadorNotificationResult{}
	if shouldNotifyJogador(agendamento) {
		notificacao = service.notifier.NotifyStatusChange(ctx, agendamento)
	}

	return agendamentoMutationResult{
		Agendamento: agendamento,
		Notificacao: notificacao,
	}, nil
}

func (service agendamentoService) refreshFinancialState(ctx context.Context, agendamento models.Agendamento) (models.Agendamento, float64, error) {
	totalPagoRegistrado, err := service.repository.sumPayments(ctx, agendamento.ID)
	if err != nil {
		return models.Agendamento{}, 0, err
	}

	valorRestante, pago, statusDePagamento := resolveFinancialState(
		agendamento.ValorTotal,
		totalPagoRegistrado,
		agendamento.Pago,
		agendamento.StatusDePagamento,
	)

	totalPago := agendamento.ValorTotal - valorRestante
	if totalPago < 0 {
		totalPago = 0
	}

	agendamento.ValorRestante = valorRestante
	agendamento.Pago = pago
	agendamento.StatusDePagamento = statusDePagamento
	return agendamento, totalPago, nil
}

func canManageExecutionState(status models.AgendamentoStatus) bool {
	switch status {
	case models.AgendamentoStatusCancelado, models.AgendamentoStatusConcluido:
		return false
	default:
		return true
	}
}

func canRegisterPayment(status models.AgendamentoStatus) bool {
	switch status {
	case models.AgendamentoStatusCancelado, models.AgendamentoStatusConcluido:
		return false
	default:
		return true
	}
}

func statusAfterCronometroEncerrado(valorRestante float64) models.AgendamentoStatus {
	if valorRestante > 0 {
		return models.AgendamentoStatusAguardandoPagamento
	}

	return models.AgendamentoStatusAgendado
}

func statusAfterPayment(agendamento models.Agendamento, valorRestante float64) *models.AgendamentoStatus {
	if agendamento.FimCronometro == nil {
		return nil
	}

	status := statusAfterCronometroEncerrado(valorRestante)
	return &status
}

func resolveFinancialState(valorTotal float64, totalPago float64, pagoFlag bool, statusDePagamento bool) (float64, bool, bool) {
	valorRestante := calcularValorRestante(valorTotal, totalPago)
	if totalPago == 0 && (pagoFlag || statusDePagamento) {
		valorRestante = 0
	}

	quitado := valorRestante <= 0
	return valorRestante, quitado, quitado
}

func shouldNotifyJogador(agendamento models.Agendamento) bool {
	if agendamento.OrigemAgendamento == models.AgendamentoOrigemManual {
		return false
	}

	switch agendamento.Status {
	case models.AgendamentoStatusAgendado, models.AgendamentoStatusCancelado:
		return true
	default:
		return false
	}
}

func calcularValorRestante(valorTotal float64, valorPago float64) float64 {
	valorRestante := valorTotal - valorPago
	if valorRestante < 0 {
		return 0
	}

	return valorRestante
}

func sanitizePagamento(value string) string {
	return strings.TrimSpace(value)
}
