package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/danpi/marca_ai_backend/internal/config"
	"github.com/danpi/marca_ai_backend/internal/middleware"
)

type AgendamentoRequest struct {
	IDCampo   int       `json:"id_campo"`
	Horario   time.Time `json:"horario"`
	Jogadores int       `json:"jogadores"`
	Pagamento string    `json:"pagamento"`
	Pago      bool      `json:"pago"`
	Status    string    `json:"status"`
	CriadoEm  time.Time `json:"criado_em"`
	ID        int       `json:"id"`
	IDUsuario int       `json:"id_usuario"`
	NomeCampo string    `json:"nome_campo"`
	NomeArena string    `json:"nome_arena"`
}

func AgendarCampo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	var agendamento struct {
		CampoID   int    `json:"campo_id"`
		Horario   string `json:"horario"`
		Jogadores int    `json:"jogadores"`
		Pagamento string `json:"pagamento"`
		Pago      bool   `json:"pago"`
	}

	if err := json.NewDecoder(r.Body).Decode(&agendamento); err != nil {
		http.Error(w, "Erro ao decodificar JSON", http.StatusBadRequest)
		log.Printf("Erro no decode do agendamento: %v", err)
		return
	}
	log.Printf("DEBUG - JSON recebido: %+v\n", agendamento)

	// ✅ Corrigir formato para datetime-local
	horarioParsed, err := time.Parse("2006-01-02T15:04", agendamento.Horario)
	if err != nil {
		http.Error(w, "Formato de horário inválido", http.StatusBadRequest)
		log.Printf("Erro ao converter horário: %v", err)
		return
	}

	// ID do usuário logado (token)
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Usuário não autenticado", http.StatusUnauthorized)
		return
	}

	//teste para verificar dados recebidos:
	log.Printf("Inserindo agendamento: user_id=%d, campo_id=%d, horario=%s, jogadores=%d, pagamento=%s, pago=%t",
		userID, agendamento.CampoID, horarioParsed, agendamento.Jogadores, agendamento.Pagamento, agendamento.Pago)

	// Inserção no banco
	_, err = config.DB.Exec(`
    INSERT INTO agendamentos (id_usuario, id_campo, horario, jogadores, pagamento, pago, status)
    VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, userID, agendamento.CampoID, horarioParsed, agendamento.Jogadores, agendamento.Pagamento, agendamento.Pago, "agendado")

	if err != nil {
		http.Error(w, "Erro ao registrar agendamento", http.StatusInternalServerError)
		log.Printf("Erro ao inserir agendamento: %v", err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	fmt.Fprint(w, "Agendamento realizado com sucesso")
}
func GetAgendamentos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Usuário não autenticado", http.StatusUnauthorized)
		return
	}

	query := `
		SELECT 
			a.id,
			a.id_usuario,
			a.id_campo,
			a.horario,
			a.jogadores,
			a.pagamento,
			a.pago,
			a.status,
			a.criado_em,
			c.nome_campo,
			ar.nome AS nome_arena
		FROM agendamentos a
		JOIN campo c ON a.id_campo = c.id_campo
		JOIN arenas ar ON c.id_arena = ar.id
		WHERE a.id_usuario = $1
		ORDER BY a.horario DESC;
	`

	rows, err := config.DB.Query(query, userID)
	if err != nil {
		http.Error(w, "Erro ao buscar agendamentos", http.StatusInternalServerError)
		log.Printf("Erro ao buscar agendamentos: %v", err)
		return
	}
	defer rows.Close()

	var agendamentos []AgendamentoRequest
	for rows.Next() {
		var ag AgendamentoRequest
		if err := rows.Scan(
			&ag.ID,
			&ag.IDUsuario,
			&ag.IDCampo,
			&ag.Horario,
			&ag.Jogadores,
			&ag.Pagamento,
			&ag.Pago,
			&ag.Status,
			&ag.CriadoEm,
			&ag.NomeCampo,
			&ag.NomeArena,
		); err != nil {
			http.Error(w, "Erro ao ler agendamentos", http.StatusInternalServerError)
			log.Printf("Erro ao escanear agendamento: %v", err)
			return
		}
		agendamentos = append(agendamentos, ag)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agendamentos)
}
func AtualizarStatusAgendamento(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	// Ler parâmetros JSON
	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Erro ao decodificar JSON", http.StatusBadRequest)
		return
	}

	// Pegar o ID do agendamento da URL
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "ID do agendamento é obrigatório", http.StatusBadRequest)
		return
	}

	// Atualizar no banco
	_, err := config.DB.Exec(`UPDATE agendamentos SET status = $1 WHERE id = $2`, body.Status, id)
	if err != nil {
		http.Error(w, "Erro ao atualizar status", http.StatusInternalServerError)
		log.Printf("Erro ao atualizar status: %v", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Status atualizado para %s", body.Status)
}
