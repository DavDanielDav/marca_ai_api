package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"

	"github.com/danpi/marca_ai_backend/internal/config"
	"github.com/danpi/marca_ai_backend/internal/models"
	"github.com/gorilla/mux"
)

func GetArenasJogador(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		log.Printf("Metodo nao permitido")
		return
	}

	arenas, err := fetchArenasJogador(r, nil)
	if err != nil {
		http.Error(w, "Erro ao buscar arenas", http.StatusInternalServerError)
		log.Printf("Erro ao buscar arenas: %v", err)
		return
	}

	if len(arenas) == 0 {
		http.Error(w, "Nenhuma arena encontrada", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(arenas)
}

func GetArenaJogadorPorID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		log.Printf("Metodo nao permitido")
		return
	}

	rawArenaID := mux.Vars(r)["id"]
	arenaID, err := strconv.Atoi(rawArenaID)
	if err != nil || arenaID <= 0 {
		http.Error(w, "ID da arena invalido", http.StatusBadRequest)
		return
	}

	arenas, err := fetchArenasJogador(r, &arenaID)
	if err != nil {
		http.Error(w, "Erro ao buscar arena", http.StatusInternalServerError)
		log.Printf("Erro ao buscar arena %d: %v", arenaID, err)
		return
	}

	if len(arenas) == 0 {
		http.Error(w, "Arena nao encontrada", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(arenas[0])
}

func fetchArenasJogador(r *http.Request, arenaID *int) ([]models.Arenas, error) {
	optionalColumns, err := loadArenaOptionalColumns(r.Context())
	if err != nil {
		return nil, err
	}

	campoOptionalColumns, err := loadCampoOptionalColumns(r.Context())
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`
		SELECT
			a.id,
			a.nome AS nome_arena,
			a.endereco,
			a.cnpj,
			a.qtd_campos,
			a.tipo,
			a.imagem,
			%s,
			%s,
			%s,
			c.id_campo,
			c.nome_campo,
			c.max_jogadores,
			c.modalidade,
			c.tipo_campo,
			c.imagem AS imagem_campo,
			%s,
			%s,
			%s
		FROM %s a
		LEFT JOIN %s c ON c.id_arena = a.id
	`,
		optionalArenaSelectExpression("a", "observacoes", optionalColumns.Observacoes),
		optionalArenaSelectExpression("a", "esportes_oferecidos", optionalColumns.EsportesOferecidos),
		optionalArenaSelectExpression("a", "informacoes_arena", optionalColumns.InformacoesArena),
		optionalCampoSelectExpression("c", "valor_hora", campoOptionalColumns.ValorHora),
		optionalCampoSelectExpression("c", "ativo", campoOptionalColumns.Ativo),
		optionalCampoSelectExpression("c", "horarios_disponiveis", campoOptionalColumns.HorariosDisponiveis),
		arenasTableName(),
		campoTableName(),
	)

	args := make([]any, 0, 1)
	if arenaID != nil {
		query += "\n\t\tWHERE a.id = $1"
		args = append(args, *arenaID)
	}

	query += "\n\t\tORDER BY a.id, c.id_campo;"

	rows, err := config.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	arenasMap := make(map[int]*models.Arenas)
	arenaHasCampo := make(map[int]bool)
	arenaHasCampoAtivo := make(map[int]bool)

	for rows.Next() {
		var (
			idArena      int
			nomeArena    string
			endereco     string
			cnpj         string
			qtdCampos    int
			tipoArena    string
			imagemArena  string
			observacoes  sql.NullString
			esportes     sql.NullString
			informacoes  sql.NullString
			idCampo      sql.NullInt64
			nomeCampo    sql.NullString
			maxJogadores sql.NullInt64
			modalidade   sql.NullString
			tipoCampo    sql.NullString
			imagemCampo  sql.NullString
			valorHora    sql.NullFloat64
			ativoCampo   sql.NullBool
			horariosRaw  sql.NullString
		)

		if err := rows.Scan(
			&idArena,
			&nomeArena,
			&endereco,
			&cnpj,
			&qtdCampos,
			&tipoArena,
			&imagemArena,
			&observacoes,
			&esportes,
			&informacoes,
			&idCampo,
			&nomeCampo,
			&maxJogadores,
			&modalidade,
			&tipoCampo,
			&imagemCampo,
			&valorHora,
			&ativoCampo,
			&horariosRaw,
		); err != nil {
			return nil, err
		}

		if _, exists := arenasMap[idArena]; !exists {
			arenasMap[idArena] = &models.Arenas{
				ID:                 idArena,
				Nome:               nomeArena,
				Cnpj:               cnpj,
				QtdCampos:          qtdCampos,
				Tipo:               tipoArena,
				Imagem:             imagemArena,
				Endereco:           endereco,
				Observacoes:        observacoes.String,
				EsportesOferecidos: esportes.String,
				InformacoesArena:   informacoes.String,
			}
		}

		if !idCampo.Valid {
			continue
		}

		arenaHasCampo[idArena] = true
		campoAtivo := !ativoCampo.Valid || ativoCampo.Bool
		if campoAtivo {
			arenaHasCampoAtivo[idArena] = true
		}

		if !campoAtivo {
			continue
		}

		campo := models.Campo{
			IDCampo:      int(idCampo.Int64),
			Nome:         nomeCampo.String,
			MaxJogadores: int(maxJogadores.Int64),
			Modalidade:   modalidade.String,
			TipoCampo:    tipoCampo.String,
			Imagem:       imagemCampo.String,
			ValorHora:    valorHora.Float64,
			Ativo:        campoAtivo,
			EmManutencao: !campoAtivo,
			IdArena:      idArena,
			NomeArena:    nomeArena,
		}
		applyCampoHorarioAliases(&campo, decodeCampoHorarios(horariosRaw.String))
		arenasMap[idArena].Campos = append(arenasMap[idArena].Campos, campo)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(arenasMap) == 0 {
		return []models.Arenas{}, nil
	}

	orderedIDs := make([]int, 0, len(arenasMap))
	for id := range arenasMap {
		orderedIDs = append(orderedIDs, id)
	}
	sort.Ints(orderedIDs)

	arenas := make([]models.Arenas, 0, len(orderedIDs))
	for _, id := range orderedIDs {
		arena := arenasMap[id]
		if arenaHasCampo[arena.ID] && !arenaHasCampoAtivo[arena.ID] {
			arena.EmManutencao = true
		}
		arena.QtdCampos = len(arena.Campos)
		arenas = append(arenas, *arena)
	}

	return arenas, nil
}
