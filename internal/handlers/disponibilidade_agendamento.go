package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/danpi/marca_ai_backend/internal/config"
	"github.com/gorilla/mux"
)

type campoDisponibilidadeInfo struct {
	IDCampo             int
	NomeCampo           string
	IDArena             int
	NomeArena           string
	ValorHora           float64
	Ativo               bool
	CampoEmManutencao   bool
	ArenaEmManutencao   bool
	HorariosDisponiveis []string
}

type horarioDisponivelResponse struct {
	IDCampo                  int      `json:"id_campo"`
	NomeCampo                string   `json:"nome_campo"`
	IDArena                  int      `json:"id_arena"`
	NomeArena                string   `json:"nome_arena"`
	Data                     string   `json:"data"`
	ValorHora                float64  `json:"valor_hora"`
	Ativo                    bool     `json:"ativo"`
	EmManutencao             bool     `json:"em_manutencao"`
	ArenaEmManutencao        bool     `json:"arena_em_manutencao"`
	Horarios                 []string `json:"horarios,omitempty"`
	HorariosDisponiveis      []string `json:"horarios_disponiveis"`
	HorariosDisponiveisCamel []string `json:"horariosDisponiveis,omitempty"`
	HorariosCampo            []string `json:"horarios_campo,omitempty"`
	HorariosOcupados         []string `json:"horarios_ocupados"`
}

func GetHorariosDisponiveisCampo(w http.ResponseWriter, r *http.Request) {
	campoIDStr := strings.TrimSpace(r.URL.Query().Get("campo_id"))
	if campoIDStr == "" {
		campoIDStr = strings.TrimSpace(r.URL.Query().Get("id_campo"))
	}
	if campoIDStr == "" {
		campoIDStr = strings.TrimSpace(mux.Vars(r)["campo_id"])
	}
	if campoIDStr == "" {
		campoIDStr = strings.TrimSpace(mux.Vars(r)["id_campo"])
	}

	if campoIDStr == "" {
		http.Error(w, "campo_id e obrigatorio", http.StatusBadRequest)
		return
	}

	campoID, err := strconv.Atoi(campoIDStr)
	if err != nil || campoID <= 0 {
		http.Error(w, "campo_id invalido", http.StatusBadRequest)
		return
	}

	location, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		http.Error(w, "Erro ao carregar fuso horario", http.StatusInternalServerError)
		return
	}

	requestedDate, err := parseAvailabilityDate(r.URL.Query().Get("data"), location)
	if err != nil {
		http.Error(w, "Data invalida. Use o formato YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	info, err := loadCampoDisponibilidadeInfo(campoID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Campo nao encontrado", http.StatusNotFound)
			return
		}
		http.Error(w, "Erro ao buscar campo para disponibilidade", http.StatusInternalServerError)
		log.Printf("Erro ao carregar disponibilidade do campo %d: %v", campoID, err)
		return
	}

	response := horarioDisponivelResponse{
		IDCampo:                  info.IDCampo,
		NomeCampo:                info.NomeCampo,
		IDArena:                  info.IDArena,
		NomeArena:                info.NomeArena,
		Data:                     requestedDate.Format("2006-01-02"),
		ValorHora:                info.ValorHora,
		Ativo:                    info.Ativo,
		EmManutencao:             info.CampoEmManutencao,
		ArenaEmManutencao:        info.ArenaEmManutencao,
		Horarios:                 append([]string(nil), info.HorariosDisponiveis...),
		HorariosDisponiveis:      []string{},
		HorariosDisponiveisCamel: append([]string(nil), info.HorariosDisponiveis...),
		HorariosCampo:            append([]string(nil), info.HorariosDisponiveis...),
		HorariosOcupados:         []string{},
	}

	if info.CampoEmManutencao || info.ArenaEmManutencao {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	allSlots := generateBookingSlots(requestedDate, location, info.HorariosDisponiveis)
	occupied, err := loadOccupiedSlotsByCampo(campoID, requestedDate, location)
	if err != nil {
		http.Error(w, "Erro ao buscar horarios ocupados", http.StatusInternalServerError)
		log.Printf("Erro ao buscar horarios ocupados do campo %d: %v", campoID, err)
		return
	}

	occupiedSet := make(map[string]struct{}, len(occupied))
	for _, slot := range occupied {
		label := slot.In(location).Format("15:04")
		occupiedSet[label] = struct{}{}
		response.HorariosOcupados = append(response.HorariosOcupados, label)
	}
	sort.Strings(response.HorariosOcupados)

	for _, slot := range allSlots {
		label := slot.In(location).Format("15:04")
		if _, exists := occupiedSet[label]; exists {
			continue
		}
		response.HorariosDisponiveis = append(response.HorariosDisponiveis, label)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func loadCampoDisponibilidadeInfo(campoID int) (campoDisponibilidadeInfo, error) {
	optionalColumns, err := loadCampoOptionalColumns(context.Background())
	if err != nil {
		return campoDisponibilidadeInfo{}, err
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
			c.nome_campo,
			c.id_arena,
			a.nome,
			%s,
			%s,
			%s,
			%s
		FROM %s c
		JOIN %s a ON a.id = c.id_arena
		WHERE c.id_campo = $1
	`,
		optionalCampoSelectExpression("c", "valor_hora", optionalColumns.ValorHora),
		optionalCampoSelectExpression("c", "ativo", optionalColumns.Ativo),
		optionalCampoSelectExpression("c", "horarios_disponiveis", optionalColumns.HorariosDisponiveis),
		arenaMaintenanceExpr,
		campoTable,
		arenasTable,
	)

	var info campoDisponibilidadeInfo
	var horariosRaw string
	err = config.DB.QueryRow(query, campoID).Scan(
		&info.IDCampo,
		&info.NomeCampo,
		&info.IDArena,
		&info.NomeArena,
		&info.ValorHora,
		&info.Ativo,
		&horariosRaw,
		&info.ArenaEmManutencao,
	)
	if err != nil {
		return campoDisponibilidadeInfo{}, err
	}

	info.CampoEmManutencao = !info.Ativo
	info.HorariosDisponiveis = decodeCampoHorarios(horariosRaw)
	return info, nil
}

func loadOccupiedSlotsByCampo(campoID int, requestedDate time.Time, location *time.Location) ([]time.Time, error) {
	startOfDay := time.Date(requestedDate.Year(), requestedDate.Month(), requestedDate.Day(), 0, 0, 0, 0, location)
	endOfDay := startOfDay.Add(30 * time.Hour)

	rows, err := config.DB.Query(fmt.Sprintf(`
		SELECT horario
		FROM %s
		WHERE id_campo = $1
		  AND horario >= $2
		  AND horario < $3
		  AND status != 'cancelado'
		ORDER BY horario ASC
	`, agendamentosTableName()), campoID, startOfDay, endOfDay)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var occupied []time.Time
	for rows.Next() {
		var horario time.Time
		if err := rows.Scan(&horario); err != nil {
			return nil, err
		}
		occupied = append(occupied, horario)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return occupied, nil
}

func parseAvailabilityDate(rawDate string, location *time.Location) (time.Time, error) {
	rawDate = strings.TrimSpace(rawDate)
	if rawDate == "" {
		now := time.Now().In(location)
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location), nil
	}

	parsed, err := time.ParseInLocation("2006-01-02", rawDate, location)
	if err != nil {
		return time.Time{}, err
	}

	return parsed, nil
}

func generateBookingSlots(requestedDate time.Time, location *time.Location, configuredHorarios []string) []time.Time {
	horarios := configuredHorarios
	if len(horarios) == 0 {
		horarios = defaultCampoHorarios()
	}

	slots := make([]time.Time, 0, len(horarios))
	for _, horario := range horarios {
		parts := strings.Split(horario, ":")
		if len(parts) != 2 {
			continue
		}

		hour, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		minute, err := strconv.Atoi(parts[1])
		if err != nil {
			continue
		}

		slotDate := requestedDate
		if hour < 6 {
			slotDate = requestedDate.Add(24 * time.Hour)
		}

		slots = append(slots, time.Date(slotDate.Year(), slotDate.Month(), slotDate.Day(), hour, minute, 0, 0, location))
	}

	return slots
}
