package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/danpi/marca_ai_backend/internal/config"
	"github.com/danpi/marca_ai_backend/internal/middleware"
)

type DashboardDados struct {
	CamposAgendados   int `json:"camposAgendados"`
	CamposCadastrados int `json:"camposCadastrados"`
	Cancelados        int `json:"cancelados"`
	Concluidos        int `json:"concluidos"`
	OcupacaoHoje      int `json:"ocupacaoHoje"`
}

type ProximoJogo struct {
	Campo      string `json:"campo"`
	Horario    string `json:"horario"`
	Modalidade string `json:"modalidade"`
}

type RankingCampo struct {
	Nome     string `json:"nome"`
	Reservas int    `json:"reservas"`
}

type DashboardResponse struct {
	Dados         DashboardDados `json:"dados"`
	ProximosJogos []ProximoJogo  `json:"proximosJogos"`
	RankingCampos []RankingCampo `json:"rankingCampos"`
}

func GetDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "M\u00e9todo n\u00e3o permitido", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "N\u00e3o foi poss\u00edvel obter o ID do usu\u00e1rio do token", http.StatusInternalServerError)
		return
	}

	var (
		camposCadastrados  int
		camposAgendados    int
		cancelados         int
		concluidos         int
		camposOcupadosHoje int
	)

	metricasQuery := `
		WITH campos_usuario AS (
			SELECT c.id_campo
			FROM campo c
			JOIN arenas a ON a.id = c.id_arena
			WHERE a.id_usuario = $1
		)
		SELECT
			(SELECT COUNT(*) FROM campos_usuario) AS campos_cadastrados,
			(SELECT COUNT(*)
			 FROM agendamentos ag
			 JOIN campos_usuario cu ON cu.id_campo = ag.id_campo
			 WHERE ag.status = 'agendado') AS campos_agendados,
			(SELECT COUNT(*)
			 FROM agendamentos ag
			 JOIN campos_usuario cu ON cu.id_campo = ag.id_campo
			 WHERE ag.status = 'cancelado') AS cancelados,
			(SELECT COUNT(*)
			 FROM agendamentos ag
			 JOIN campos_usuario cu ON cu.id_campo = ag.id_campo
			 WHERE ag.status = 'concluido') AS concluidos,
			(SELECT COUNT(DISTINCT ag.id_campo)
			 FROM agendamentos ag
			 JOIN campos_usuario cu ON cu.id_campo = ag.id_campo
			 WHERE ag.status <> 'cancelado'
			   AND DATE(ag.horario AT TIME ZONE 'America/Sao_Paulo') = DATE(NOW() AT TIME ZONE 'America/Sao_Paulo')
			) AS campos_ocupados_hoje;
	`

	err := config.DB.QueryRow(metricasQuery, userID).Scan(
		&camposCadastrados,
		&camposAgendados,
		&cancelados,
		&concluidos,
		&camposOcupadosHoje,
	)
	if err != nil {
		log.Printf("Erro ao buscar m\u00e9tricas do dashboard: %v", err)
		http.Error(w, "Erro ao carregar dashboard", http.StatusInternalServerError)
		return
	}

	ocupacaoHoje := 0
	if camposCadastrados > 0 {
		ocupacaoHoje = (camposOcupadosHoje * 100) / camposCadastrados
	}

	proximosJogos := make([]ProximoJogo, 0)
	proximosJogosRows, err := config.DB.Query(`
		SELECT
			c.nome_campo,
			TO_CHAR(ag.horario AT TIME ZONE 'America/Sao_Paulo', 'HH24:MI') AS horario,
			c.modalidade
		FROM agendamentos ag
		JOIN campo c ON c.id_campo = ag.id_campo
		JOIN arenas a ON a.id = c.id_arena
		WHERE a.id_usuario = $1
		  AND ag.status = 'agendado'
		  AND ag.horario >= NOW()
		ORDER BY ag.horario ASC
		LIMIT 3;
	`, userID)
	if err != nil {
		log.Printf("Erro ao buscar pr\u00f3ximos jogos do dashboard: %v", err)
		http.Error(w, "Erro ao carregar dashboard", http.StatusInternalServerError)
		return
	}
	defer proximosJogosRows.Close()

	for proximosJogosRows.Next() {
		var item ProximoJogo
		if err := proximosJogosRows.Scan(&item.Campo, &item.Horario, &item.Modalidade); err != nil {
			log.Printf("Erro ao ler pr\u00f3ximos jogos do dashboard: %v", err)
			http.Error(w, "Erro ao carregar dashboard", http.StatusInternalServerError)
			return
		}
		proximosJogos = append(proximosJogos, item)
	}

	if err := proximosJogosRows.Err(); err != nil {
		log.Printf("Erro na itera\u00e7\u00e3o de pr\u00f3ximos jogos do dashboard: %v", err)
		http.Error(w, "Erro ao carregar dashboard", http.StatusInternalServerError)
		return
	}

	rankingCampos := make([]RankingCampo, 0)
	rankingRows, err := config.DB.Query(`
		SELECT
			c.nome_campo,
			COUNT(*) AS reservas
		FROM agendamentos ag
		JOIN campo c ON c.id_campo = ag.id_campo
		JOIN arenas a ON a.id = c.id_arena
		WHERE a.id_usuario = $1
		  AND ag.status <> 'cancelado'
		GROUP BY c.id_campo, c.nome_campo
		ORDER BY reservas DESC, c.nome_campo ASC
		LIMIT 3;
	`, userID)
	if err != nil {
		log.Printf("Erro ao buscar ranking de campos do dashboard: %v", err)
		http.Error(w, "Erro ao carregar dashboard", http.StatusInternalServerError)
		return
	}
	defer rankingRows.Close()

	for rankingRows.Next() {
		var item RankingCampo
		if err := rankingRows.Scan(&item.Nome, &item.Reservas); err != nil {
			log.Printf("Erro ao ler ranking de campos do dashboard: %v", err)
			http.Error(w, "Erro ao carregar dashboard", http.StatusInternalServerError)
			return
		}
		rankingCampos = append(rankingCampos, item)
	}

	if err := rankingRows.Err(); err != nil {
		log.Printf("Erro na itera\u00e7\u00e3o de ranking de campos do dashboard: %v", err)
		http.Error(w, "Erro ao carregar dashboard", http.StatusInternalServerError)
		return
	}

	response := DashboardResponse{
		Dados: DashboardDados{
			CamposAgendados:   camposAgendados,
			CamposCadastrados: camposCadastrados,
			Cancelados:        cancelados,
			Concluidos:        concluidos,
			OcupacaoHoje:      ocupacaoHoje,
		},
		ProximosJogos: proximosJogos,
		RankingCampos: rankingCampos,
	}
	log.Printf("DashboardDados")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
