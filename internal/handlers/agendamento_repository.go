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
	IDCampo         int
	Horario         time.Time
	Jogadores       int
	Pagamento       string
	Pago            bool
	NomeSolicitante string
	ValorTotal      float64
	ValorRestante   float64
}

type agendamentoFinancialUpdate struct {
	ValorRestante     float64
	Pago              bool
	StatusDePagamento bool
	Status            *models.AgendamentoStatus
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
	createdAt := agendamentoNow()
	query := fmt.Sprintf(`
		INSERT INTO %s (
			id_usuario,
			id_campo,
			horario,
			jogadores,
			pagamento,
			pago,
			criado_em,
			nome_solicitante,
			status,
			status_de_pagamento,
			origem_agendamento,
			valor_total,
			valor_restante,
			time1,
			time2,
			modo_de_jogo
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NULLIF($8, ''), $9, $10, $11, $12, $13, NULLIF($14, ''), NULLIF($15, ''), NULLIF($16, ''))
		RETURNING id_agendamento, criado_em
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
		createdAt,
		input.NomeSolicitante,
		string(status),
		input.Pago,
		string(input.OrigemAgendamento),
		valorTotal,
		valorRestante,
		input.Time1,
		input.Time2,
		input.ModoDeJogo,
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
	agendamento.NomeSolicitante = input.NomeSolicitante
	agendamento.StatusDePagamento = input.Pago
	agendamento.Status = status
	agendamento.OrigemAgendamento = input.OrigemAgendamento
	agendamento.ValorTotal = valorTotal
	agendamento.ValorRestante = valorRestante
	agendamento.Time1 = input.Time1
	agendamento.Time2 = input.Time2
	agendamento.ModoDeJogo = input.ModoDeJogo
	return agendamento, nil
}

func (agendamentoRepository) loadJogadorNomeByID(ctx context.Context, jogadorID int) (string, error) {
	query := fmt.Sprintf(`
		SELECT COALESCE(nome, ''), COALESCE(sobrenome, '')
		FROM %s
		WHERE id = $1
	`, usuarioJogadorTableName())

	var nome string
	var sobrenome string
	if err := config.DB.QueryRowContext(ctx, query, jogadorID).Scan(&nome, &sobrenome); err != nil {
		return "", err
	}

	return buildNomeCompleto(nome, sobrenome), nil
}

func buildNomeCompleto(nome string, sobrenome string) string {
	parts := make([]string, 0, 2)

	if nome = strings.TrimSpace(nome); nome != "" {
		parts = append(parts, nome)
	}
	if sobrenome = strings.TrimSpace(sobrenome); sobrenome != "" {
		parts = append(parts, sobrenome)
	}

	return strings.Join(parts, " ")
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
				WHEN 'em_andamento' THEN 2
				WHEN 'aguardando_pagamento' THEN 3
				WHEN 'concluido' THEN 4
				WHEN 'cancelado' THEN 5
				ELSE 6
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
			status_de_pagamento = $6,
			nome_solicitante = NULLIF($7, ''),
			valor_total = $8,
			valor_restante = $9
		WHERE id_agendamento = $10
	`, agendamentosTableName()),
		input.IDCampo,
		input.Horario,
		input.Jogadores,
		input.Pagamento,
		input.Pago,
		input.Pago,
		input.NomeSolicitante,
		input.ValorTotal,
		input.ValorRestante,
		agendamentoID,
	)
	return err
}

func (agendamentoRepository) startCronometro(ctx context.Context, agendamentoID int, inicioUnix int64) error {
	_, err := config.DB.ExecContext(ctx, fmt.Sprintf(`
		UPDATE %s
		SET
			inicio_cronometro = $1,
			fim_cronometro = NULL,
			status = $2
		WHERE id_agendamento = $3
	`, agendamentosTableName()), inicioUnix, string(models.AgendamentoStatusEmAndamento), agendamentoID)
	return err
}

func (agendamentoRepository) finishCronometro(ctx context.Context, agendamentoID int, fim time.Time, status models.AgendamentoStatus) error {
	_, err := config.DB.ExecContext(ctx, fmt.Sprintf(`
		UPDATE %s
		SET
			fim_cronometro = $1,
			status = $2
		WHERE id_agendamento = $3
	`, agendamentosTableName()), fim, string(status), agendamentoID)
	return err
}

func (agendamentoRepository) updateFinancialState(ctx context.Context, agendamentoID int, input agendamentoFinancialUpdate) error {
	query := fmt.Sprintf(`
		UPDATE %s
		SET
			valor_restante = $1,
			pago = $2,
			status_de_pagamento = $3
	`, agendamentosTableName())

	args := []any{input.ValorRestante, input.Pago, input.StatusDePagamento}
	if input.Status != nil {
		query += ", status = $4"
		args = append(args, string(*input.Status))
	}

	query += fmt.Sprintf(" WHERE id_agendamento = $%d", len(args)+1)
	args = append(args, agendamentoID)

	_, err := config.DB.ExecContext(ctx, query, args...)
	return err
}

func (agendamentoRepository) insertPayment(ctx context.Context, agendamentoID int, input models.RegistrarPagamentoInput) (models.AgendamentoPagamento, error) {
	dataPagamento := agendamentoNow()
	query := fmt.Sprintf(`
		INSERT INTO %s (
			id_agendamento,
			id_usuario,
			valor_pago,
			forma_pagamento,
			data_pagamento
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, data_pagamento
	`, pagamentosPorAgendamentoTableName())

	var pagamento models.AgendamentoPagamento
	err := config.DB.QueryRowContext(
		ctx,
		query,
		agendamentoID,
		nullableIntValue(input.IDUsuario),
		input.ValorPago,
		input.FormaPagamento,
		dataPagamento,
	).Scan(&pagamento.ID, &pagamento.DataPagamento)
	if err != nil {
		return models.AgendamentoPagamento{}, err
	}

	pagamento.IDAgendamento = agendamentoID
	pagamento.IDUsuario = input.IDUsuario
	pagamento.ValorPago = input.ValorPago
	pagamento.FormaPagamento = input.FormaPagamento
	return pagamento, nil
}

func (agendamentoRepository) listPayments(ctx context.Context, agendamentoID int) ([]models.AgendamentoPagamento, error) {
	query := fmt.Sprintf(`
		SELECT
			p.id,
			p.id_agendamento,
			p.id_usuario,
			p.valor_pago,
			COALESCE(p.forma_pagamento, ''),
			p.data_pagamento,
			COALESCE(u.nome, ''),
			COALESCE(u.sobrenome, ''),
			COALESCE(u.email, '')
		FROM %s p
		LEFT JOIN %s u ON u.id = p.id_usuario
		WHERE p.id_agendamento = $1
		ORDER BY p.data_pagamento ASC, p.id ASC
	`, pagamentosPorAgendamentoTableName(), usuarioJogadorTableName())

	rows, err := config.DB.QueryContext(ctx, query, agendamentoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pagamentos := make([]models.AgendamentoPagamento, 0)
	for rows.Next() {
		var (
			pagamento models.AgendamentoPagamento
			idUsuario sql.NullInt64
		)

		if err := rows.Scan(
			&pagamento.ID,
			&pagamento.IDAgendamento,
			&idUsuario,
			&pagamento.ValorPago,
			&pagamento.FormaPagamento,
			&pagamento.DataPagamento,
			&pagamento.NomeUsuario,
			&pagamento.SobrenomeUsuario,
			&pagamento.EmailUsuario,
		); err != nil {
			return nil, err
		}

		if idUsuario.Valid {
			value := int(idUsuario.Int64)
			pagamento.IDUsuario = &value
		}

		pagamentos = append(pagamentos, pagamento)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return pagamentos, nil
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
			COALESCE(a.nome_solicitante, ''),
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
			COALESCE(a.valor_restante, 0),
			COALESCE(a.status_de_pagamento, FALSE),
			a.inicio_cronometro,
			a.fim_cronometro,
			COALESCE(a.time1, ''),
			COALESCE(a.time2, ''),
			COALESCE(a.modo_de_jogo, '')
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
		agendamento       models.Agendamento
		idUsuario         sql.NullInt64
		statusRaw         string
		origemRaw         string
		criadoEm          sql.NullTime
		nomeCampo         sql.NullString
		nomeArena         sql.NullString
		pagamento         sql.NullString
		valorTotal        sql.NullFloat64
		valorRestante     sql.NullFloat64
		statusDePagamento sql.NullBool
		inicioCronometro  sql.NullInt64
		fimCronometro     sql.NullTime
		time1             sql.NullString
		time2             sql.NullString
		modoDeJogo        sql.NullString
	)

	err := scanner.Scan(
		&agendamento.ID,
		&idUsuario,
		&agendamento.IDCampo,
		&agendamento.IDArena,
		&agendamento.NomeSolicitante,
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
		&statusDePagamento,
		&inicioCronometro,
		&fimCronometro,
		&time1,
		&time2,
		&modoDeJogo,
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
	if statusDePagamento.Valid {
		agendamento.StatusDePagamento = statusDePagamento.Bool
	}
	if inicioCronometro.Valid {
		value := inicioCronometro.Int64
		agendamento.InicioCronometro = &value
	}
	if fimCronometro.Valid {
		value := fimCronometro.Time
		agendamento.FimCronometro = &value
	}
	if time1.Valid {
		agendamento.Time1 = time1.String
	}
	if time2.Valid {
		agendamento.Time2 = time2.String
	}
	if modoDeJogo.Valid {
		agendamento.ModoDeJogo = modoDeJogo.String
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
