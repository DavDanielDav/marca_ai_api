package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/danpi/marca_ai_backend/internal/config"
)

type arenaOptionalColumns struct {
	Observacoes        bool
	EsportesOferecidos bool
	InformacoesArena   bool
}

func loadArenaOptionalColumns(ctx context.Context) (arenaOptionalColumns, error) {
	rows, err := config.DB.QueryContext(ctx, `
		SELECT column_name
		FROM information_schema.columns
		WHERE table_schema = $1
		  AND table_name = 'arenas'
		  AND column_name IN ('observacoes', 'esportes_oferecidos', 'informacoes_arena')
	`, config.DBSchemaName())
	if err != nil {
		return arenaOptionalColumns{}, err
	}
	defer rows.Close()

	var columns arenaOptionalColumns

	for rows.Next() {
		var columnName string
		if err := rows.Scan(&columnName); err != nil {
			return arenaOptionalColumns{}, err
		}

		switch strings.TrimSpace(columnName) {
		case "observacoes":
			columns.Observacoes = true
		case "esportes_oferecidos":
			columns.EsportesOferecidos = true
		case "informacoes_arena":
			columns.InformacoesArena = true
		}
	}

	if err := rows.Err(); err != nil {
		return arenaOptionalColumns{}, err
	}

	return columns, nil
}

func optionalArenaSelectExpression(tableAlias, columnName string, exists bool) string {
	if exists {
		if strings.TrimSpace(tableAlias) == "" {
			return fmt.Sprintf("COALESCE(%s, '') AS %s", columnName, columnName)
		}

		return fmt.Sprintf("COALESCE(%s.%s, '') AS %s", tableAlias, columnName, columnName)
	}

	return fmt.Sprintf("CAST('' AS TEXT) AS %s", columnName)
}

func buildArenaInsertQuery(columns arenaOptionalColumns) string {
	columnNames := []string{
		"nome",
		"cnpj",
		"qtd_campos",
		"tipo",
		"imagem",
		"endereco",
	}

	if columns.Observacoes {
		columnNames = append(columnNames, "observacoes")
	}
	if columns.EsportesOferecidos {
		columnNames = append(columnNames, "esportes_oferecidos")
	}
	if columns.InformacoesArena {
		columnNames = append(columnNames, "informacoes_arena")
	}

	columnNames = append(columnNames, "id_usuario")

	placeholders := make([]string, 0, len(columnNames))
	for index := range columnNames {
		placeholders = append(placeholders, fmt.Sprintf("$%d", index+1))
	}

	return fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		arenasTableName(),
		strings.Join(columnNames, ", "),
		strings.Join(placeholders, ", "),
	)
}

func buildArenaInsertArgs(arenaName string, arenaCNPJ string, arenaQtdCampos int, arenaTipo string, arenaImagem string, arenaEndereco string, arenaObservacoes string, arenaEsportesOferecidos string, arenaInformacoes string, userID int, columns arenaOptionalColumns) []any {
	args := []any{
		arenaName,
		arenaCNPJ,
		arenaQtdCampos,
		arenaTipo,
		arenaImagem,
		arenaEndereco,
	}

	if columns.Observacoes {
		args = append(args, arenaObservacoes)
	}
	if columns.EsportesOferecidos {
		args = append(args, arenaEsportesOferecidos)
	}
	if columns.InformacoesArena {
		args = append(args, arenaInformacoes)
	}

	args = append(args, userID)
	return args
}

func buildArenaUpdateQuery(columns arenaOptionalColumns) string {
	assignments := []string{
		"nome = COALESCE(NULLIF($1, ''), nome)",
		"cnpj = COALESCE(NULLIF($2, ''), cnpj)",
		"qtd_campos = COALESCE(NULLIF($3::text, '')::int, qtd_campos)",
		"tipo = COALESCE(NULLIF($4, ''), tipo)",
		"endereco = COALESCE(NULLIF($5, ''), endereco)",
		"imagem = COALESCE(NULLIF($6, ''), imagem)",
	}

	nextPlaceholder := 7

	if columns.Observacoes {
		assignments = append(assignments, fmt.Sprintf("observacoes = COALESCE(NULLIF($%d, ''), observacoes)", nextPlaceholder))
		nextPlaceholder++
	}
	if columns.EsportesOferecidos {
		assignments = append(assignments, fmt.Sprintf("esportes_oferecidos = COALESCE(NULLIF($%d, ''), esportes_oferecidos)", nextPlaceholder))
		nextPlaceholder++
	}
	if columns.InformacoesArena {
		assignments = append(assignments, fmt.Sprintf("informacoes_arena = COALESCE(NULLIF($%d, ''), informacoes_arena)", nextPlaceholder))
		nextPlaceholder++
	}

	return fmt.Sprintf(
		"UPDATE %s SET %s WHERE id_usuario = $%d",
		arenasTableName(),
		strings.Join(assignments, ", "),
		nextPlaceholder,
	)
}

func buildArenaUpdateArgs(arenaName string, arenaCNPJ string, arenaQtdCampos int, arenaTipo string, arenaEndereco string, arenaImagem string, arenaObservacoes string, arenaEsportesOferecidos string, arenaInformacoes string, userID int, columns arenaOptionalColumns) []any {
	args := []any{
		arenaName,
		arenaCNPJ,
		fmt.Sprintf("%d", arenaQtdCampos),
		arenaTipo,
		arenaEndereco,
		arenaImagem,
	}

	if columns.Observacoes {
		args = append(args, arenaObservacoes)
	}
	if columns.EsportesOferecidos {
		args = append(args, arenaEsportesOferecidos)
	}
	if columns.InformacoesArena {
		args = append(args, arenaInformacoes)
	}

	args = append(args, userID)
	return args
}
