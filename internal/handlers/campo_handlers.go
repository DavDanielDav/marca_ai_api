package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/danpi/marca_ai_backend/internal/config"
	"github.com/danpi/marca_ai_backend/internal/middleware"
	"github.com/danpi/marca_ai_backend/internal/models"
	"github.com/danpi/marca_ai_backend/internal/utils"
)

func CadastrodeCampo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		log.Printf("Método não autorizado")
		return
	}

	// ✅ Recuperar userID do contexto (middleware)
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Não foi possível obter o ID do usuário", http.StatusUnauthorized)
		return
	}

	// ✅ Processar multipart form
	err := r.ParseMultipartForm(10 << 20) // até 10MB
	if err != nil {
		http.Error(w, "Erro ao processar form data", http.StatusBadRequest)
		return
	}

	// ✅ Coletar campos do form
	nome := r.FormValue("nome_campo")
	maxJogadores := r.FormValue("maxJogadores")
	modalidade := r.FormValue("modalidade")
	tipoCampo := r.FormValue("tipoCampo")
	idArenaStr := r.FormValue("idArena")

	// ✅ Converter ID da arena
	idArena, err := strconv.Atoi(idArenaStr)
	if err != nil {
		http.Error(w, "ID da arena inválido", http.StatusBadRequest)
		return
	}

	// ✅ Verificar se a arena pertence ao usuário
	var pertence bool
	err = config.DB.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM arenas WHERE id = $1 AND usuario_id = $2
		)
	`, idArena, userID).Scan(&pertence)
	if err != nil {
		http.Error(w, "Erro ao verificar arena", http.StatusInternalServerError)
		return
	}
	if !pertence {
		http.Error(w, "Arena não pertence ao usuário logado", http.StatusForbidden)
		return
	}

	// ✅ Processar imagem
	file, fileHeader, err := r.FormFile("imagem")
	if err != nil {
		http.Error(w, "Erro ao ler imagem", http.StatusBadRequest)
		return
	}
	defer file.Close()

	urlImagem, err := utils.UploadCloudinary(file, fileHeader.Filename)
	if err != nil {
		http.Error(w, "Erro ao enviar imagem: "+err.Error(), http.StatusInternalServerError)
		return
	}

	maxJogadoresInt, _ := strconv.Atoi(maxJogadores)
	newCampo := models.Campo{
		Nome:         nome,
		MaxJogadores: maxJogadoresInt,
		Modalidade:   modalidade,
		TipoCampo:    tipoCampo,
		Imagem:       urlImagem,
		IdArena:      idArena, // ✅ vínculo com a arena
	}

	// ✅ Inserir no DB
	_, err = config.DB.Exec(`
		INSERT INTO campo (nome_campo, modalidade, tipo_campo, imagem, max_jogadores, id_arena)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, newCampo.Nome, newCampo.Modalidade, newCampo.TipoCampo, newCampo.Imagem, newCampo.MaxJogadores, newCampo.IdArena)
	if err != nil {
		log.Printf("Erro ao inserir Campo no banco: %v", err)
		http.Error(w, "Erro ao registrar Campo", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Campo cadastrado com sucesso",
		"imagem":  newCampo.Imagem,
	})

}

func GetCampos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		log.Printf("Método não permitido")
		return
	}

	// ✅ Pega o ID do usuário logado pelo token
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Não foi possível obter o ID do usuário do token", http.StatusInternalServerError)
		return
	}

	// ✅ Verifica se o usuário tem arenas
	rowsArenas, err := config.DB.Query(`SELECT id FROM arenas WHERE usuario_id = $1`, userID)
	if err != nil {
		http.Error(w, "Erro ao buscar arenas do usuário", http.StatusInternalServerError)
		log.Printf("Erro ao buscar arenas: %v", err)
		return
	}
	defer rowsArenas.Close()

	var arenaIDs []int
	for rowsArenas.Next() {
		var id int
		if err := rowsArenas.Scan(&id); err != nil {
			http.Error(w, "Erro ao ler arenas", http.StatusInternalServerError)
			return
		}
		arenaIDs = append(arenaIDs, id)
	}

	// ✅ Se o usuário não tiver arenas, retorna JSON vazio
	if len(arenaIDs) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]models.Campo{})
		return
	}

	// ✅ Busca todos os campos dessas arenas
	query := `
		SELECT 
			c.id_campo, 
			c.nome_campo, 
			c.max_jogadores, 
			c.modalidade, 
			c.tipo_campo, 
			c.imagem,
			c.id_arena,
			a.nome AS nome_arena
		FROM campo c
		JOIN arenas a ON c.id_arena = a.id
		WHERE a.usuario_id = $1;
	`

	rowsCampos, err := config.DB.Query(query, userID)
	if err != nil {
		http.Error(w, "Erro ao buscar campos", http.StatusInternalServerError)
		log.Printf("Erro ao buscar campos: %v", err)
		return
	}
	defer rowsCampos.Close()

	var campos []models.Campo

	for rowsCampos.Next() {
		var campo models.Campo
		err := rowsCampos.Scan(
			&campo.IDCampo,
			&campo.Nome,
			&campo.MaxJogadores,
			&campo.Modalidade,
			&campo.TipoCampo,
			&campo.Imagem,
			&campo.IdArena,
			&campo.NomeArena,
		)
		if err != nil {
			http.Error(w, "Erro ao ler campos", http.StatusInternalServerError)
			log.Printf("Erro ao escanear campos: %v", err)
			return
		}
		campos = append(campos, campo)
	}

	// ✅ Mesmo se não tiver campos, retorna JSON vazio
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(campos)
}
