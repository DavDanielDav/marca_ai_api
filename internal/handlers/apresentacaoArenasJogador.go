package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/danpi/marca_ai_backend/internal/config"
	"github.com/danpi/marca_ai_backend/internal/models"
)

func GetArenasJogador(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		log.Printf("Método não permitido")
		return
	}

	optionalColumns, err := loadArenaOptionalColumns(r.Context())
	if err != nil {
		http.Error(w, "Erro ao carregar estrutura de arenas", http.StatusInternalServerError)
		log.Printf("Erro ao carregar colunas opcionais de arenas: %v", err)
		return
	}

	campoOptionalColumns, err := loadCampoOptionalColumns(r.Context())
	if err != nil {
		http.Error(w, "Erro ao carregar estrutura de campos", http.StatusInternalServerError)
		log.Printf("Erro ao carregar colunas opcionais de campos: %v", err)
		return
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
		FROM arenas a
		LEFT JOIN campo c ON c.id_arena = a.id
		ORDER BY a.id;
	`,
		optionalArenaSelectExpression("a", "observacoes", optionalColumns.Observacoes),
		optionalArenaSelectExpression("a", "esportes_oferecidos", optionalColumns.EsportesOferecidos),
		optionalArenaSelectExpression("a", "informacoes_arena", optionalColumns.InformacoesArena),
		optionalCampoSelectExpression("c", "valor_hora", campoOptionalColumns.ValorHora),
		optionalCampoSelectExpression("c", "ativo", campoOptionalColumns.Ativo),
		optionalCampoSelectExpression("c", "horarios_disponiveis", campoOptionalColumns.HorariosDisponiveis),
	)

	rows, err := config.DB.Query(query)
	if err != nil {
		http.Error(w, "Erro ao buscar arenas", http.StatusInternalServerError)
		log.Printf("Erro ao buscar arenas: %v", err)
		return
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

		err := rows.Scan(
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
		)
		if err != nil {
			http.Error(w, "Erro ao ler dados", http.StatusInternalServerError)
			log.Printf("Erro ao escanear arenas: %v", err)
			return
		}

		// Se a arena ainda não estiver no mapa, cria
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

		// Adiciona o campo (se existir)
		if idCampo.Valid {
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
			// Adiciona o campo à arena correspondente
			// Cria o slice de campos dentro da arena se ainda não existir
			// (se quiser armazenar dentro do JSON)
			arenasMap[idArena].Campos = append(arenasMap[idArena].Campos, campo)

			// Caso queira retornar arenas e campos separados:
			// você pode criar um slice separado fora do loop
		}
	}

	// Converte o mapa em slice
	var arenas []models.Arenas
	for _, a := range arenasMap {
		if arenaHasCampo[a.ID] && !arenaHasCampoAtivo[a.ID] {
			a.EmManutencao = true
		}
		a.QtdCampos = len(a.Campos)
		arenas = append(arenas, *a)
	}

	if len(arenas) == 0 {
		http.Error(w, "Nenhuma arena encontrada", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(arenas)
}
