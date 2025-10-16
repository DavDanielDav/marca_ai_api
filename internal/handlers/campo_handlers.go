package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/danpi/marca_ai_backend/internal/config"
	"github.com/danpi/marca_ai_backend/internal/models"
	"github.com/danpi/marca_ai_backend/internal/utils"
)

func CadastrodeCampo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		log.Printf("Metodo nao autorizado")
		return
	}
	// Parse multipart form
	err := r.ParseMultipartForm(10 << 20) // 10MB máximo
	if err != nil {
		http.Error(w, "Erro ao processar form data", http.StatusBadRequest)
		log.Printf("Erro ao processar form data")
		return
	}
	//pegar campos
	nome := r.FormValue("nome")
	maxJogadores := r.FormValue("maxjogadores")
	modalidade := r.FormValue("modalidade")
	tipoCampo := r.FormValue("tipocampo")

	// Processar arquivo
	file, fileHeader, err := r.FormFile("imagem")
	if err != nil {
		http.Error(w, "Erro ao ler a imagem", http.StatusBadRequest)
		log.Printf("Erro ao ler  a imagem")
		return
	}
	defer file.Close()

	//upload imagem para cloudinary
	urlImagem, err := utils.UploadCloudinary(file, fileHeader.Filename)
	if err != nil {
		http.Error(w, "Erro ao enviar imagem: "+err.Error(), http.StatusInternalServerError)
		log.Printf("Erro ao enviar imagem")
		return
	}

	//criaçao e inserçao struct
	newCampo := models.Campo{
		Nome:         nome,
		MaxJogadores: atoi(maxJogadores),
		Modalidade:   modalidade,
		TipoCampo:    tipoCampo,
		Imagem:       urlImagem,
	}

	// Salvar no DB
	_, err = config.DB.Exec(
		"INSERT INTO campo (nome_campo, modalidade, tipo_campo, imagem, max_jogadores) VALUES ($1, $2, $3, $4, $5)",
		newCampo.Nome, newCampo.Modalidade, newCampo.TipoCampo, newCampo.Imagem, newCampo.MaxJogadores,
	)
	if err != nil {
		log.Printf("Erro ao inserir Campo no banco: %v", err)
		http.Error(w, "Erro ao registrar Campo", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Arena registrada com sucesso",
		"imagem":  newCampo.Imagem,
	})

}
