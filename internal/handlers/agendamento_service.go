package handlers

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/danpi/marca_ai_backend/internal/models"
)

var (
	errAgendamentoCampoNaoEncontrado  = errors.New("campo nao encontrado")
	errAgendamentoCampoSemPermissao   = errors.New("campo nao pertence ao usuario")
	errAgendamentoNaoEncontrado       = errors.New("agendamento nao encontrado")
	errAgendamentoOrigemInvalida      = errors.New("origem do agendamento invalida")
	errAgendamentoStatusInvalido      = errors.New("status do agendamento invalido")
	errAgendamentoHorarioIndisponivel = errors.New("horario indisponivel")
	errAgendamentoCampoIndisponivel   = errors.New("campo indisponivel para agendamento")
	errAgendamentoJogadoresInvalidos  = errors.New("quantidade de jogadores invalida")
	errAgendamentoPedidoNaoPendente   = errors.New("pedido nao esta pendente")
)

type agendamentoMutationResult struct {
	Agendamento models.Agendamento        `json:"agendamento"`
	Notificacao jogadorNotificationResult `json:"notificacao"`
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
	valorRestante := calcularValorRestante(valorTotal, totalPago)

	err = service.repository.update(ctx, agendamentoID, agendamentoUpdateInput{
		IDCampo:       input.IDCampo,
		Horario:       input.Horario,
		Jogadores:     input.Jogadores,
		Pagamento:     input.Pagamento,
		Pago:          input.Pago,
		ValorTotal:    valorTotal,
		ValorRestante: valorRestante,
	})
	if err != nil {
		return models.Agendamento{}, err
	}

	agendamentoAtual.IDCampo = input.IDCampo
	agendamentoAtual.IDArena = campo.IDArena
	agendamentoAtual.Horario = input.Horario
	agendamentoAtual.Jogadores = input.Jogadores
	agendamentoAtual.Pagamento = input.Pagamento
	agendamentoAtual.Pago = input.Pago
	agendamentoAtual.NomeCampo = campo.NomeCampo
	agendamentoAtual.NomeArena = campo.NomeArena
	agendamentoAtual.ValorTotal = valorTotal
	agendamentoAtual.ValorRestante = valorRestante

	return agendamentoAtual, nil
}

func (service agendamentoService) UpdateStatus(ctx context.Context, ownerUserID int, agendamentoID int, status models.AgendamentoStatus) (agendamentoMutationResult, error) {
	switch status {
	case models.AgendamentoStatusAgendado, models.AgendamentoStatusCancelado, models.AgendamentoStatusConcluido, models.AgendamentoStatusPedido:
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

func (service agendamentoService) create(ctx context.Context, input models.CreateAgendamentoInput, ownerUserID int, status models.AgendamentoStatus) (models.Agendamento, error) {
	campo, err := service.validateCampoAndSchedule(ctx, input, ownerUserID, nil)
	if err != nil {
		return models.Agendamento{}, err
	}

	valorTotal := campo.ValorHora
	valorRestante := valorTotal

	agendamento, err := service.repository.create(ctx, input, status, valorTotal, valorRestante)
	if err != nil {
		return models.Agendamento{}, err
	}

	agendamento.IDArena = campo.IDArena
	agendamento.NomeCampo = campo.NomeCampo
	agendamento.NomeArena = campo.NomeArena
	return agendamento, nil
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
