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

// AgendamentoResponse é a estrutura para a resposta JSON com horários formatados
type AgendamentoResponse struct {
	ID        int    `json:"id"`
	IDUsuario int    `json:"id_usuario"`
	IDCampo   int    `json:"id_campo"`
	Horario   string `json:"horario"`
	Jogadores int    `json:"jogadores"`
	Pagamento string `json:"pagamento"`
	Pago      bool   `json:"pago"`
	Status    string `json:"status"`
	CriadoEm  string `json:"criado_em"`
	NomeCampo string `json:"nome_campo"`
	NomeArena string `json:"nome_arena"`
}

func AgendarCampo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	var agendamento struct {
		CampoID   int    `json:"campo_id"`
		Horario   string `json:"horario"` // Ex: 2025-11-26T20:00
		Jogadores int    `json:"jogadores"`
		Pagamento string `json:"pagamento"`
		Pago      bool   `json:"pago"`
	}

	if err := json.NewDecoder(r.Body).Decode(&agendamento); err != nil {
		http.Error(w, "Erro ao decodificar JSON", http.StatusBadRequest)
		return
	}

	// --------------------------
	// 1. Fuso horário correto (SP)
	// --------------------------
	location, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		http.Error(w, "Erro ao carregar fuso horário", http.StatusInternalServerError)
		return
	}

	// ---------------------------------------------------
	// 2. Converte EXATAMENTE o horário vindo do front-end
	//     Sem UTC, sem perda de horas
	// ---------------------------------------------------
	horarioParsed, err := time.ParseInLocation("2006-01-02T15:04", agendamento.Horario, location)
	if err != nil {
		log.Printf("❌ Erro ao converter horário: %v", err)
		http.Error(w, "Formato de horário inválido", http.StatusBadRequest)
		return
	}

	// -------------------------
	// 3. Recupera o ID do usuário
	// -------------------------
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Usuário não autenticado", http.StatusUnauthorized)
		return
	}

	// --------------------------------------
	// 4. Verifica se o horário já está ocupado
	// --------------------------------------
	var count int
	err = config.DB.QueryRow(`
        SELECT COUNT(*)
        FROM agendamentos
        WHERE id_campo = $1
          AND horario = $2
          AND status != 'cancelado'
    `,
		agendamento.CampoID,
		horarioParsed,
	).Scan(&count)

	if err != nil {
		http.Error(w, "Erro ao verificar disponibilidade do horário", http.StatusInternalServerError)
		return
	}

	if count > 0 {
		http.Error(w, "Este horário já está reservado para o campo selecionado.", http.StatusConflict)
		return
	}

	// -----------------------
	// 5. Inserir no banco
	// -----------------------
	_, err = config.DB.Exec(`
        INSERT INTO agendamentos
            (id_usuario, id_campo, horario, jogadores, pagamento, pago, status)
        VALUES
            ($1, $2, $3, $4, $5, $6, $7)
    `,
		userID,
		agendamento.CampoID,
		horarioParsed, // SALVA exatamente como 20:00 -03
		agendamento.Jogadores,
		agendamento.Pagamento,
		agendamento.Pago,
		"agendado",
	)

	if err != nil {
		http.Error(w, "Erro ao registrar agendamento", http.StatusInternalServerError)
		return
	}

	// -----------------------
	// 6. Log completo
	// -----------------------
	log.Printf(
		"📥 FRONT → BACK | Dados recebidos:\n"+
			"CampoID: %d\n"+
			"Horario (string): %s\n"+
			"Horario (Go): %s\n"+
			"Jogadores: %d\n"+
			"Pagamento: %s\n"+
			"Pago: %t\n"+
			"UserID: %d\n",
		agendamento.CampoID,
		agendamento.Horario,
		horarioParsed.Format("2006-01-02 15:04:05 -07:00"),
		agendamento.Jogadores,
		agendamento.Pagamento,
		agendamento.Pago,
		userID,
	)

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
			a.id, a.id_usuario, a.id_campo, a.horario, a.jogadores, a.pagamento, 
			a.pago, a.status, a.criado_em, c.nome_campo, ar.nome AS nome_arena
		FROM agendamentos a
		JOIN campo c ON a.id_campo = c.id_campo
		JOIN arenas ar ON c.id_arena = ar.id
		WHERE a.id_usuario = $1
		ORDER BY a.horario DESC;
	`

	rows, err := config.DB.Query(query, userID)
	if err != nil {
		http.Error(w, "Erro ao buscar agendamentos", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var response []AgendamentoResponse

	for rows.Next() {
		var ag AgendamentoRequest

		if err := rows.Scan(
			&ag.ID, &ag.IDUsuario, &ag.IDCampo, &ag.Horario,
			&ag.Jogadores, &ag.Pagamento, &ag.Pago,
			&ag.Status, &ag.CriadoEm, &ag.NomeCampo, &ag.NomeArena,
		); err != nil {
			http.Error(w, "Erro ao ler agendamentos", http.StatusInternalServerError)
			return
		}

		// 🚫 Não alterar o horário
		// ❗ Apenas enviar como RFC3339 para o front

		// LOG COMPLETO CORRETO
		/*log.Printf(
			"📤 BACK → FRONT | Enviando agendamento:\n"+
				"ID: %d\n"+
				"Horario (Go): %s\n"+
				"Horario (RFC3339): %s\n"+
				"Status: %s\n"+
				"NomeCampo: %s\n"+
				"NomeArena: %s\n",
			ag.ID,
			ag.Horario.String(),
			ag.Horario.Format(time.RFC3339),
			ag.Status,
			ag.NomeCampo,
			ag.NomeArena,
		)*/

		response = append(response, AgendamentoResponse{
			ID:        ag.ID,
			IDUsuario: ag.IDUsuario,
			IDCampo:   ag.IDCampo,
			Horario:   ag.Horario.Format(time.RFC3339),
			Jogadores: ag.Jogadores,
			Pagamento: ag.Pagamento,
			Pago:      ag.Pago,
			Status:    ag.Status,
			CriadoEm:  ag.CriadoEm.Format(time.RFC3339),
			NomeCampo: ag.NomeCampo,
			NomeArena: ag.NomeArena,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func AtualizarStatusAgendamento(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Erro ao decodificar JSON", http.StatusBadRequest)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "ID do agendamento é obrigatório", http.StatusBadRequest)
		return
	}

	_, err := config.DB.Exec(`UPDATE agendamentos SET status = $1 WHERE id = $2`, body.Status, id)
	if err != nil {
		http.Error(w, "Erro ao atualizar status", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Status atualizado")
}
