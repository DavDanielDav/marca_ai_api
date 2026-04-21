package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/danpi/marca_ai_backend/internal/config"
	"github.com/danpi/marca_ai_backend/internal/models"
)

type agendamentoRepository struct{}

type campoAgendamentoSnapshot struct {
	IDCampo           int
	IDArena           int
	OwnerUserID       int
	NomeCampo         string
	NomeArena         string
	ValorHora         float64
	MaxJogadores      int
	Ativo             bool
	CampoEmManutencao bool
	ArenaEmManutencao bool
}

type agendamentoUpdateInput struct {
	IDCampo       int
	Horario       time.Time
	Jogadores     int
	Pagamento     string
	Pago          bool
	ValorTotal    float64
	ValorRestante float64
}

func newAgendamentoRepository() agendamentoRepository {
	return agendamentoRepository{}
}

func (agendamentoRepository) loadCampoSnapshot(ctx context.Context, campoID int) (campoAgendamentoSnapshot, error) {
	optionalColumns, err := loadCampoOptionalColumns(ctx)
	if err != nil {
		return campoAgendamentoSnapshot{}, err
	}

	campoTable := campoTableName()
	arenasTable := arenasTableName()
	arenaMaintenanceExpr := "CAST(FALSE AS BOOLEAN) AS arena_em_manutencao"
	if optionalColumns.Ativo {
		arenaMaintenanceExpr = fmt.Sprintf(`
			NOT EXISTS (
				SELECT 1
				FROM %s c2
				WHERE c2.id_arena = c.id_arena
				  AND COALESCE(c2.ativo, TRUE)
			) AS arena_em_manutencao
		`, campoTable)
	}

	query := fmt.Sprintf(`
		SELECT
			c.id_campo,
			c.id_arena,
			a.id_usuario,
			c.nome_campo,
			a.nome,
			%s,
			c.max_jogadores,
			%s,
			%s
		FROM %s c
		JOIN %s a ON a.id = c.id_arena
		WHERE c.id_campo = $1
	`,
		optionalCampoSelectExpression("c", "valor_hora", optionalColumns.ValorHora),
		optionalCampoSelectExpression("c", "ativo", optionalColumns.Ativo),
		arenaMaintenanceExpr,
		campoTable,
		arenasTable,
	)

	var snapshot campoAgendamentoSnapshot
	var maxJogadores sql.NullInt64
	err = config.DB.QueryRowContext(ctx, query, campoID).Scan(
		&snapshot.IDCampo,
		&snapshot.IDArena,
		&snapshot.OwnerUserID,
		&snapshot.NomeCampo,
		&snapshot.NomeArena,
		&snapshot.ValorHora,
		&maxJogadores,
		&snapshot.Ativo,
		&snapshot.ArenaEmManutencao,
	)
	if err != nil {
		return campoAgendamentoSnapshot{}, err
	}

	if maxJogadores.Valid {
		snapshot.MaxJogadores = int(maxJogadores.Int64)
	}
	snapshot.CampoEmManutencao = !snapshot.Ativo
	return snapshot, nil
}

func (agendamentoRepository) hasScheduleConflict(ctx context.Context, campoID int, horario time.Time, excludeID *int) (bool, error) {
	query := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM %s
		WHERE id_campo = $1
		  AND horario = $2
		  AND status != $3
	`, agendamentosTableName())

	args := []any{campoID, horario, string(models.AgendamentoStatusCancelado)}
	if excludeID != nil {
		query += " AND id_agendamento != $4"
		args = append(args, *excludeID)
	}

	var count int
	if err := config.DB.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return false, err
	}

	return count > 0, nil
}

func (agendamentoRepository) create(ctx context.Context, input models.CreateAgendamentoInput, status models.AgendamentoStatus, valorTotal float64, valorRestante float64) (models.Agendamento, error) {
	query := fmt.Sprintf(`
		INSERT INTO %s (
			id_usuario,
			id_campo,
			horario,
			jogadores,
			pagamento,
			pago,
			status,
			origem_agendamento,
			valor_total,
			valor_restante
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id_agendamento, COALESCE(criado_em, NOW())
	`, agendamentosTableName())

	var agendamento models.Agendamento
	err := config.DB.QueryRowContext(
		ctx,
		query,
		nullableIntValue(input.IDUsuario),
		input.IDCampo,
		input.Horario,
		input.Jogadores,
		input.Pagamento,
		input.Pago,
		string(status),
		string(input.OrigemAgendamento),
		valorTotal,
		valorRestante,
	).Scan(&agendamento.ID, &agendamento.CriadoEm)
	if err != nil {
		return models.Agendamento{}, err
	}

	if input.IDUsuario != nil {
		agendamento.IDUsuario = *input.IDUsuario
	}
	agendamento.IDCampo = input.IDCampo
	agendamento.Horario = input.Horario
	agendamento.Jogadores = input.Jogadores
	agendamento.Pagamento = input.Pagamento
	agendamento.Pago = input.Pago
	agendamento.Status = status
	agendamento.OrigemAgendamento = input.OrigemAgendamento
	agendamento.ValorTotal = valorTotal
	agendamento.ValorRestante = valorRestante
	return agendamento, nil
}

func (repository agendamentoRepository) listByOwner(ctx context.Context, ownerUserID int, status *models.AgendamentoStatus) ([]models.Agendamento, error) {
	where := []string{"ar.id_usuario = $1"}
	args := []any{ownerUserID}

	if status != nil {
		where = append(where, fmt.Sprintf("a.status = $%d", len(args)+1))
		args = append(args, string(*status))
	}

	query := fmt.Sprintf(`
		%s
		WHERE %s
		ORDER BY
			CASE a.status
				WHEN 'pedido' THEN 0
				WHEN 'agendado' THEN 1
				WHEN 'concluido' THEN 2
				WHEN 'cancelado' THEN 3
				ELSE 4
			END,
			a.horario DESC
	`, agendamentoBaseSelectQuery(), strings.Join(where, " AND "))

	rows, err := config.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	agendamentos := make([]models.Agendamento, 0)
	for rows.Next() {
		agendamento, scanErr := scanAgendamento(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		agendamentos = append(agendamentos, agendamento)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return agendamentos, nil
}

func (repository agendamentoRepository) getByIDForOwner(ctx context.Context, agendamentoID int, ownerUserID int) (models.Agendamento, error) {
	query := fmt.Sprintf(`
		%s
		WHERE a.id_agendamento = $1
		  AND ar.id_usuario = $2
	`, agendamentoBaseSelectQuery())

	return scanAgendamento(
		config.DB.QueryRowContext(ctx, query, agendamentoID, ownerUserID),
	)
}

func (agendamentoRepository) updateStatus(ctx context.Context, agendamentoID int, status models.AgendamentoStatus) error {
	_, err := config.DB.ExecContext(
		ctx,
		fmt.Sprintf(`UPDATE %s SET status = $1 WHERE id_agendamento = $2`, agendamentosTableName()),
		string(status),
		agendamentoID,
	)
	return err
}

func (agendamentoRepository) update(ctx context.Context, agendamentoID int, input agendamentoUpdateInput) error {
	_, err := config.DB.ExecContext(ctx, fmt.Sprintf(`
		UPDATE %s
		SET
			id_campo = $1,
			horario = $2,
			jogadores = $3,
			pagamento = $4,
			pago = $5,
			valor_total = $6,
			valor_restante = $7
		WHERE id_agendamento = $8
	`, agendamentosTableName()),
		input.IDCampo,
		input.Horario,
		input.Jogadores,
		input.Pagamento,
		input.Pago,
		input.ValorTotal,
		input.ValorRestante,
		agendamentoID,
	)
	return err
}

func (agendamentoRepository) sumPayments(ctx context.Context, agendamentoID int) (float64, error) {
	var total sql.NullFloat64
	err := config.DB.QueryRowContext(
		ctx,
		fmt.Sprintf(`SELECT COALESCE(SUM(valor_pago), 0) FROM %s WHERE id_agendamento = $1`, pagamentosPorAgendamentoTableName()),
		agendamentoID,
	).Scan(&total)
	if err != nil {
		return 0, err
	}

	if !total.Valid {
		return 0, nil
	}

	return total.Float64, nil
}

func agendamentoBaseSelectQuery() string {
	return fmt.Sprintf(`
		SELECT
			a.id_agendamento,
			a.id_usuario,
			a.id_campo,
			c.id_arena,
			a.horario,
			a.jogadores,
			a.pagamento,
			a.pago,
			a.status,
			COALESCE(a.criado_em, a.horario),
			c.nome_campo,
			ar.nome AS nome_arena,
			COALESCE(a.origem_agendamento, 'manual') AS origem_agendamento,
			COALESCE(a.valor_total, 0),
			COALESCE(a.valor_restante, 0)
		FROM %s a
		JOIN %s c ON a.id_campo = c.id_campo
		JOIN %s ar ON c.id_arena = ar.id
	`, agendamentosTableName(), campoTableName(), arenasTableName())
}

type agendamentoScanner interface {
	Scan(dest ...any) error
}

func scanAgendamento(scanner agendamentoScanner) (models.Agendamento, error) {
	var (
		agendamento   models.Agendamento
		idUsuario     sql.NullInt64
		statusRaw     string
		origemRaw     string
		criadoEm      sql.NullTime
		nomeCampo     sql.NullString
		nomeArena     sql.NullString
		pagamento     sql.NullString
		valorTotal    sql.NullFloat64
		valorRestante sql.NullFloat64
	)

	err := scanner.Scan(
		&agendamento.ID,
		&idUsuario,
		&agendamento.IDCampo,
		&agendamento.IDArena,
		&agendamento.Horario,
		&agendamento.Jogadores,
		&pagamento,
		&agendamento.Pago,
		&statusRaw,
		&criadoEm,
		&nomeCampo,
		&nomeArena,
		&origemRaw,
		&valorTotal,
		&valorRestante,
	)
	if err != nil {
		return models.Agendamento{}, err
	}

	if idUsuario.Valid {
		agendamento.IDUsuario = int(idUsuario.Int64)
	}
	if pagamento.Valid {
		agendamento.Pagamento = pagamento.String
	}
	if criadoEm.Valid {
		agendamento.CriadoEm = criadoEm.Time
	}
	if nomeCampo.Valid {
		agendamento.NomeCampo = nomeCampo.String
	}
	if nomeArena.Valid {
		agendamento.NomeArena = nomeArena.String
	}
	if valorTotal.Valid {
		agendamento.ValorTotal = valorTotal.Float64
	}
	if valorRestante.Valid {
		agendamento.ValorRestante = valorRestante.Float64
	}

	if normalizedStatus, ok := models.NormalizeAgendamentoStatus(statusRaw); ok {
		agendamento.Status = normalizedStatus
	}
	if normalizedOrigem, ok := models.NormalizeAgendamentoOrigem(origemRaw); ok {
		agendamento.OrigemAgendamento = normalizedOrigem
	}
	if agendamento.OrigemAgendamento == "" {
		agendamento.OrigemAgendamento = models.AgendamentoOrigemManual
	}

	return agendamento, nil
}

func nullableIntValue(value *int) any {
	if value == nil {
		return nil
	}

	return *value
}
