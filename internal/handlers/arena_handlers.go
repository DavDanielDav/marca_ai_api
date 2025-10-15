package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/danpi/marca_ai_backend/internal/config"
	"github.com/danpi/marca_ai_backend/internal/middleware"
	"github.com/danpi/marca_ai_backend/internal/models"
	"github.com/danpi/marca_ai_backend/internal/utils"
)

func CadastrodeArena(w http.ResponseWriter, r *http.Request) {

	//verificaçao de metodo
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	//Pegar ID a partir do token
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Não foi possível obter o ID do usuário do token", http.StatusInternalServerError)
		return
	}

	//cria corpo json a partir de variaveis
	var newArena models.Arenas
	err := json.NewDecoder(r.Body).Decode(&newArena)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	//retorno de dados da aquisiçao
	log.Printf("Dados recebidos: %+v", newArena)

	file, fileHeader, err := r.FormFile("imagem")
	if err != nil {
		http.Error(w, "Erro ao ler a imagem", http.StatusBadRequest)
		return
	}
	defer file.Close()
	// Aqui você pode processar o arquivo de imagem recebido.
	// Por exemplo, salvar o arquivo em disco ou armazenar em algum serviço externo.
	// No momento, apenas comentamos que o arquivo foi recebido com sucesso.
	log.Printf("Arquivo de imagem recebido: %s", fileHeader.Filename)

	//  Faz o upload pro Cloudinary
	urlImagem, err := utils.UploadCloudinary(file, fileHeader.Filename)
	if err != nil {
		http.Error(w, "Erro ao enviar imagem: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Salva a URL na struct
	newArena.Imagem = urlImagem

	//logica do DB
	_, err = config.DB.Exec(
		"INSERT INTO arenas (nome, cnpj, qtd_campos, tipo, imagem, endereco, usuario_id) VALUES ($1, $2, $3, $4, $5, $6, $7)",
		newArena.Nome, newArena.Cnpj, newArena.QtdCampos, newArena.Tipo, newArena.Imagem, newArena.Endereco, userID,
	)
	if err != nil {
		log.Printf("Erro ao inserir Arena no banco: %v", err)
		http.Error(w, "Erro ao registrar Arena", http.StatusInternalServerError)
		return
	}

	log.Printf("Arena inserido no banco de dados com sucesso!")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Arena registrado com sucesso",
		"imagem":  newArena.Imagem})
}
