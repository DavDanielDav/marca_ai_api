package handlers

import (
	"database/sql"
	"encoding/json"
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

	rows, err := config.DB.Query(`
		SELECT 
			a.id,
			a.nome AS nome_arena,
			a.endereco,
			a.cnpj,
			a.qtd_campos,
			a.tipo,
			a.imagem,
			c.id_campo,
			c.nome_campo,
			c.max_jogadores,
			c.modalidade,
			c.tipo_campo,
			c.imagem AS imagem_campo
		FROM arenas a
		LEFT JOIN campo c ON c.id_arena = a.id
		ORDER BY a.id;
	`)
	if err != nil {
		http.Error(w, "Erro ao buscar arenas", http.StatusInternalServerError)
		log.Printf("Erro ao buscar arenas: %v", err)
		return
	}
	defer rows.Close()

	arenasMap := make(map[int]*models.Arenas)

	for rows.Next() {
		var (
			idArena      int
			nomeArena    string
			endereco     string
			cnpj         string
			qtdCampos    int
			tipoArena    string
			imagemArena  string
			idCampo      sql.NullInt64
			nomeCampo    sql.NullString
			maxJogadores sql.NullInt64
			modalidade   sql.NullString
			tipoCampo    sql.NullString
			imagemCampo  sql.NullString
		)

		err := rows.Scan(
			&idArena,
			&nomeArena,
			&endereco,
			&cnpj,
			&qtdCampos,
			&tipoArena,
			&imagemArena,
			&idCampo,
			&nomeCampo,
			&maxJogadores,
			&modalidade,
			&tipoCampo,
			&imagemCampo,
		)
		if err != nil {
			http.Error(w, "Erro ao ler dados", http.StatusInternalServerError)
			log.Printf("Erro ao escanear arenas: %v", err)
			return
		}

		// Se a arena ainda não estiver no mapa, cria
		if _, exists := arenasMap[idArena]; !exists {
			arenasMap[idArena] = &models.Arenas{
				ID:        idArena,
				Nome:      nomeArena,
				Cnpj:      cnpj,
				QtdCampos: qtdCampos,
				Tipo:      tipoArena,
				Imagem:    imagemArena,
				Endereco:  endereco,
			}
		}

		// Adiciona o campo (se existir)
		if idCampo.Valid {
			campo := models.Campo{
				IDCampo:      int(idCampo.Int64),
				Nome:         nomeCampo.String,
				MaxJogadores: int(maxJogadores.Int64),
				Modalidade:   modalidade.String,
				TipoCampo:    tipoCampo.String,
				Imagem:       imagemCampo.String,
				IdArena:      idArena,
				NomeArena:    nomeArena,
			}
			// Adiciona o campo à arena correspondente
			arenasMap[idArena].QtdCampos++
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
		arenas = append(arenas, *a)
	}

	if len(arenas) == 0 {
		http.Error(w, "Nenhuma arena encontrada", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(arenas)
}
