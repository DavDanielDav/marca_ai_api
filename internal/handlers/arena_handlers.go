package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/danpi/marca_ai_backend/internal/config"
	"github.com/danpi/marca_ai_backend/internal/models"
	"github.com/danpi/marca_ai_backend/internal/utils"
)

func CadastrodeArena(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		log.Printf("Methodo nao permitido")
		return
	}

	/*userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Não foi possível obter o ID do usuário do token", http.StatusInternalServerError)
		log.Printf("Nao foi possivel obter o ID do usuario")
		return
	}*/

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
		"INSERT INTO arenas (nome, cnpj, qtd_campos, tipo, imagem, endereco) VALUES ($1, $2, $3, $4, $5, $6)",
		newArena.Nome, newArena.Cnpj, newArena.QtdCampos, newArena.Tipo, newArena.Imagem, newArena.Endereco,
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

// Helper para converter string para int
func atoi(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}
