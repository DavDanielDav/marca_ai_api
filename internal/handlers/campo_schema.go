package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/danpi/marca_ai_backend/internal/config"
)

type campoOptionalColumns struct {
	ValorHora           bool
	Ativo               bool
	HorariosDisponiveis bool
}

func loadCampoOptionalColumns(ctx context.Context) (campoOptionalColumns, error) {
	rows, err := config.DB.QueryContext(ctx, `
		SELECT column_name
		FROM information_schema.columns
		WHERE table_schema = 'public'
		  AND table_name = 'campo'
		  AND column_name IN ('valor_hora', 'ativo', 'horarios_disponiveis')
	`)
	if err != nil {
		return campoOptionalColumns{}, err
	}
	defer rows.Close()

	var columns campoOptionalColumns

	for rows.Next() {
		var columnName string
		if err := rows.Scan(&columnName); err != nil {
			return campoOptionalColumns{}, err
		}

		switch strings.TrimSpace(columnName) {
		case "valor_hora":
			columns.ValorHora = true
		case "ativo":
			columns.Ativo = true
		case "horarios_disponiveis":
			columns.HorariosDisponiveis = true
		}
	}

	if err := rows.Err(); err != nil {
		return campoOptionalColumns{}, err
	}

	return columns, nil
}

func optionalCampoSelectExpression(tableAlias, columnName string, exists bool) string {
	if exists {
		if columnName == "horarios_disponiveis" {
			if strings.TrimSpace(tableAlias) == "" {
				return "COALESCE(horarios_disponiveis, '[]'::jsonb)::text AS horarios_disponiveis"
			}

			return fmt.Sprintf("COALESCE(%s.%s, '[]'::jsonb)::text AS %s", tableAlias, columnName, columnName)
		}

		if strings.TrimSpace(tableAlias) == "" {
			return fmt.Sprintf("%s AS %s", columnName, columnName)
		}

		return fmt.Sprintf("%s.%s AS %s", tableAlias, columnName, columnName)
	}

	switch columnName {
	case "valor_hora":
		return "CAST(0 AS NUMERIC(10,2)) AS valor_hora"
	case "ativo":
		return "CAST(TRUE AS BOOLEAN) AS ativo"
	case "horarios_disponiveis":
		return "CAST('[]' AS TEXT) AS horarios_disponiveis"
	default:
		return fmt.Sprintf("NULL AS %s", columnName)
	}
}

func buildCampoInsertQuery(columns campoOptionalColumns) string {
	columnNames := []string{
		"nome_campo",
		"modalidade",
		"tipo_campo",
		"imagem",
		"max_jogadores",
		"id_arena",
	}

	if columns.ValorHora {
		columnNames = append(columnNames, "valor_hora")
	}
	if columns.Ativo {
		columnNames = append(columnNames, "ativo")
	}
	if columns.HorariosDisponiveis {
		columnNames = append(columnNames, "horarios_disponiveis")
	}

	placeholders := make([]string, 0, len(columnNames))
	for index, columnName := range columnNames {
		placeholder := fmt.Sprintf("$%d", index+1)
		if columnName == "horarios_disponiveis" {
			placeholder += "::jsonb"
		}
		placeholders = append(placeholders, placeholder)
	}

	return fmt.Sprintf(
		"INSERT INTO campo (%s) VALUES (%s)",
		strings.Join(columnNames, ", "),
		strings.Join(placeholders, ", "),
	)
}

func buildCampoInsertArgs(campoNome string, campoModalidade string, campoTipo string, campoImagem string, campoMaxJogadores int, campoIDArena int, campoValorHora float64, campoAtivo bool, campoHorariosJSON string, columns campoOptionalColumns) []any {
	args := []any{
		campoNome,
		campoModalidade,
		campoTipo,
		campoImagem,
		campoMaxJogadores,
		campoIDArena,
	}

	if columns.ValorHora {
		args = append(args, campoValorHora)
	}
	if columns.Ativo {
		args = append(args, campoAtivo)
	}
	if columns.HorariosDisponiveis {
		args = append(args, campoHorariosJSON)
	}

	return args
}
