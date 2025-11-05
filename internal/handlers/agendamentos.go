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
	IDCampo   int       `json:"campo"`
	Horario   time.Time `json:"horario"`
	Jogadores int       `json:"jogadores"`
	Pagamento string    `json:"pagamento"`
	Pago      bool      `json:"pago"`
	Status    string    `json:"status"`
	CriadoEm  time.Time `json:"criadoEm"`
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
