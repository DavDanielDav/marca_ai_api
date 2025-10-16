package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/danpi/marca_ai_backend/internal/config"
	"github.com/danpi/marca_ai_backend/internal/middleware"
	"github.com/danpi/marca_ai_backend/internal/models"
	"github.com/danpi/marca_ai_backend/internal/utils"
)

func CadastrodeArena(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		log.Printf("Methodo nao permitido")
		return
	}

	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Não foi possível obter o ID do usuário do token", http.StatusInternalServerError)
		log.Printf("Nao foi possivel obter o ID do usuario")
		return
	}

	// Parse multipart form
	err := r.ParseMultipartForm(10 << 20) // 10MB máximo
	if err != nil {
		http.Error(w, "Erro ao processar form data", http.StatusBadRequest)
		log.Printf("Erro ao processar form data")
		return
	}

	// Pegar campos
	nome := r.FormValue("nome")
	cnpj := r.FormValue("cnpj")
	qtdCampos := r.FormValue("qtdCampos")
	tipo := r.FormValue("tipo")
	endereco := r.FormValue("endereco") // já vem como JSON string

	// Processar arquivo
	file, fileHeader, err := r.FormFile("imagem")
	if err != nil {
		http.Error(w, "Erro ao ler a imagem", http.StatusBadRequest)
		log.Printf("Erro ao ler  a imagem")
		return
	}
	defer file.Close()

	// Upload para Cloudinary
	urlImagem, err := utils.UploadCloudinary(file, fileHeader.Filename)
	if err != nil {
		http.Error(w, "Erro ao enviar imagem: "+err.Error(), http.StatusInternalServerError)
		log.Printf("Erro ao enviar imagem")
		return
	}

	// Montar struct
	newArena := models.Arenas{
		Nome:      nome,
		Cnpj:      cnpj,
		QtdCampos: atoi(qtdCampos), // converte string para int
		Tipo:      tipo,
		Imagem:    urlImagem,
		Endereco:  endereco,
	}

	// Salvar no DB
	_, err = config.DB.Exec(
		"INSERT INTO arenas (nome, cnpj, qtd_campos, tipo, imagem, endereco, usuario_id) VALUES ($1, $2, $3, $4, $5, $6, $7)",
		newArena.Nome, newArena.Cnpj, newArena.QtdCampos, newArena.Tipo, newArena.Imagem, newArena.Endereco, userID,
	)
	if err != nil {
		log.Printf("Erro ao inserir Arena no banco: %v", err)
		http.Error(w, "Erro ao registrar Arena", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Arena registrada com sucesso",
		"imagem":  newArena.Imagem,
	})
}

// API para requisiçao de listagem de arenas
func GetArenas(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		log.Printf("Methodo nao permitido")
		return
	}

	// Pega o ID do usuário do contexto, injetado pelo middleware
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Não foi possível obter o ID do usuário do token", http.StatusInternalServerError)
		return
	}

	// Faz a consulta de todas as arenas do usuário logado
	rows, err := config.DB.Query(`
		SELECT id, nome, qtd_campos, tipo, imagem, endereco
		FROM arenas
		WHERE usuario_id = $1
	`, userID)
	if err != nil {
		http.Error(w, "Erro ao buscar arenas", http.StatusInternalServerError)
		log.Printf("Erro ao buscar arenas: %v", err)
		return
	}
	defer rows.Close()

	var arenas []models.Arenas

	// Itera sobre o resultado
	for rows.Next() {
		var arena models.Arenas
		err := rows.Scan(&arena.ID, &arena.Nome, &arena.QtdCampos, &arena.Tipo, &arena.Imagem, &arena.Endereco)
		if err != nil {
			http.Error(w, "Erro ao ler dados", http.StatusInternalServerError)
			log.Printf("Erro ao escanear arenas: %v", err)
			return
		}
		arenas = append(arenas, arena)
	}

	// Caso não tenha arenas
	if len(arenas) == 0 {
		http.Error(w, "Nenhuma arena encontrada", http.StatusNotFound)
		return
	}

	log.Printf("Arenas encontradas: %d", len(arenas))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(arenas)
}

func DeleteArena(w http.ResponseWriter, r *http.Request) {
	// Pega o ID do usuário do contexto.
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Não foi possível obter o ID do usuário do token", http.StatusInternalServerError)
		log.Printf("Nao foi possivel pegar ID do usuário do token")
		return
	}

	_, err := config.DB.Exec("DELETE FROM arenas WHERE id_usuario=$1", userID)
	if err != nil {
		log.Printf("Erro ao deletar arena do banco: %v", err)
		http.Error(w, "Erro ao deletar arena do banco", http.StatusInternalServerError)
		return
	}

	log.Printf("Arena de Usuario com ID %d deletado com sucesso!", userID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Arena deletado com sucesso"})
}

func UpdateArena(w http.ResponseWriter, r *http.Request) {

	// Pega o ID do usuário do token
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Não foi possível obter o ID do usuário do token", http.StatusInternalServerError)
		log.Printf("Não foi possível pegar ID do usuário do token")
		return
	}

	// Faz o parse do form (pra aceitar campos + imagem)
	err := r.ParseMultipartForm(10 << 20) // 10 MB
	if err != nil {
		http.Error(w, "Erro ao processar formulário", http.StatusBadRequest)
		return
	}

	// Cria struct e preenche os campos
	var arena models.Arenas
	arena.Nome = r.FormValue("nome")
	arena.Cnpj = r.FormValue("cnpj")
	arena.Tipo = r.FormValue("tipo")
	arena.Endereco = r.FormValue("endereco")
	fmt.Sscan(r.FormValue("qtdCampos"), &arena.QtdCampos)

	// --- Upload da nova imagem (opcional) ---
	var urlImagem string
	file, header, err := r.FormFile("imagem")
	if err == nil {
		defer file.Close()
		urlImagem, err = utils.UploadCloudinary(file, header.Filename)
		if err != nil {
			http.Error(w, "Erro ao enviar imagem: "+err.Error(), http.StatusInternalServerError)
			return
		}
		log.Printf("Nova imagem enviada: %s", urlImagem)
	}

	// --- Atualiza no banco ---
	query := `
		UPDATE arenas
		SET
			nome = COALESCE(NULLIF($1, ''), nome),
			cnpj = COALESCE(NULLIF($2, ''), cnpj),
			qtd_campos = COALESCE(NULLIF($3::text, '')::int, qtd_campos),
			tipo = COALESCE(NULLIF($4, ''), tipo),
			endereco = COALESCE(NULLIF($5, ''), endereco),
			imagem = COALESCE(NULLIF($6, ''), imagem)
		WHERE usuario_id = $7
	`

	_, err = config.DB.Exec(query,
		arena.Nome,
		arena.Cnpj,
		fmt.Sprintf("%d", arena.QtdCampos), // força string pra NULLIF
		arena.Tipo,
		arena.Endereco,
		urlImagem, // se vazio, mantém a antiga
		userID,
	)

	if err != nil {
		log.Printf("Erro ao atualizar arena no banco: %v", err)
		http.Error(w, "Erro ao atualizar arena", http.StatusInternalServerError)
		return
	}

	log.Println("✅ Arena atualizada com sucesso!")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Arena atualizada com sucesso",
		"imagem":  urlImagem,
	})
}

// Helper para converter string para int
func atoi(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}
