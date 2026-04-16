package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/danpi/marca_ai_backend/internal/config"
	"github.com/danpi/marca_ai_backend/internal/middleware"
	"github.com/danpi/marca_ai_backend/internal/models"
	"github.com/danpi/marca_ai_backend/internal/utils"
)

func CadastrodeArena(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		log.Printf("Metodo nao permitido")
		return
	}

	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Nao foi possivel obter o ID do usuario do token", http.StatusInternalServerError)
		log.Printf("Nao foi possivel obter o ID do usuario")
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Erro ao processar form data", http.StatusBadRequest)
		log.Printf("Erro ao processar form data")
		return
	}

	nome := r.FormValue("nome")
	cnpj := r.FormValue("cnpj")
	qtdCampos := r.FormValue("qtdCampos")
	tipo := r.FormValue("tipo")
	endereco := r.FormValue("endereco")
	observacoes := firstNonEmptyArenaValue(r.FormValue("observacoes"), r.FormValue("observações"))
	esportesOferecidos := firstNonEmptyArenaValue(
		r.FormValue("esportes_oferecidos"),
		r.FormValue("esportesOferecidos"),
		r.FormValue("esportes disponíveis"),
	)
	informacoesArena := firstNonEmptyArenaValue(
		r.FormValue("informacoes_arena"),
		r.FormValue("informacoesArena"),
	)

	cnpjInfo := utils.CNPJInfo{}
	cnpjValidado := false
	cnpjArmazenado := utils.NormalizeCNPJ(cnpj)
	var err error
	optionalColumns, err := loadArenaOptionalColumns(r.Context())
	if err != nil {
		http.Error(w, "Erro ao carregar estrutura de arenas", http.StatusInternalServerError)
		log.Printf("Erro ao carregar colunas opcionais de arenas: %v", err)
		return
	}

	if utils.IsCNPJValidationEnabled() {
		cnpjInfo, err = utils.ValidateExistingCNPJ(r.Context(), cnpj)
		if err != nil {
			writeCNPJValidationError(w, err)
			return
		}

		cnpjArmazenado = cnpjInfo.CNPJ
		cnpjValidado = true
	}

	file, fileHeader, err := r.FormFile("imagem")
	if err != nil {
		http.Error(w, "Erro ao ler a imagem", http.StatusBadRequest)
		log.Printf("Erro ao ler a imagem")
		return
	}
	defer file.Close()

	urlImagem, err := utils.UploadCloudinary(file, fileHeader.Filename)
	if err != nil {
		http.Error(w, "Erro ao enviar imagem: "+err.Error(), http.StatusInternalServerError)
		log.Printf("Erro ao enviar imagem")
		return
	}

	newArena := models.Arenas{
		Nome:               nome,
		Cnpj:               cnpjArmazenado,
		QtdCampos:          atoi(qtdCampos),
		Tipo:               tipo,
		Imagem:             urlImagem,
		Endereco:           endereco,
		Observacoes:        observacoes,
		EsportesOferecidos: esportesOferecidos,
		InformacoesArena:   informacoesArena,
	}

	_, err = config.DB.Exec(
		buildArenaInsertQuery(optionalColumns),
		buildArenaInsertArgs(
			newArena.Nome,
			newArena.Cnpj,
			newArena.QtdCampos,
			newArena.Tipo,
			newArena.Imagem,
			newArena.Endereco,
			newArena.Observacoes,
			newArena.EsportesOferecidos,
			newArena.InformacoesArena,
			userID,
			optionalColumns,
		)...,
	)
	if err != nil {
		log.Printf("Erro ao inserir arena no banco: %v", err)
		http.Error(w, "Erro ao registrar arena", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"message":             "Arena registrada com sucesso",
		"imagem":              newArena.Imagem,
		"cnpj":                newArena.Cnpj,
		"cnpj_validado":       cnpjValidado,
		"observacoes":         newArena.Observacoes,
		"esportes_oferecidos": newArena.EsportesOferecidos,
		"informacoes_arena":   newArena.InformacoesArena,
		"razao_social":        cnpjInfo.RazaoSocial,
		"nome_fantasia":       cnpjInfo.NomeFantasia,
		"situacao_cadastral":  cnpjInfo.DescricaoSituacao,
	})
}

func GetArenas(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		log.Printf("Metodo nao permitido")
		return
	}

	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Nao foi possivel obter o ID do usuario do token", http.StatusInternalServerError)
		return
	}

	optionalColumns, err := loadArenaOptionalColumns(r.Context())
	if err != nil {
		http.Error(w, "Erro ao carregar estrutura de arenas", http.StatusInternalServerError)
		log.Printf("Erro ao carregar colunas opcionais de arenas: %v", err)
		return
	}

	query := fmt.Sprintf(`
		SELECT
			id,
			nome,
			cnpj,
			qtd_campos,
			tipo,
			imagem,
			endereco,
			%s,
			%s,
			%s
		FROM arenas
		WHERE id_usuario = $1
	`,
		optionalArenaSelectExpression("", "observacoes", optionalColumns.Observacoes),
		optionalArenaSelectExpression("", "esportes_oferecidos", optionalColumns.EsportesOferecidos),
		optionalArenaSelectExpression("", "informacoes_arena", optionalColumns.InformacoesArena),
	)

	rows, err := config.DB.Query(query, userID)
	if err != nil {
		http.Error(w, "Erro ao buscar arenas", http.StatusInternalServerError)
		log.Printf("Erro ao buscar arenas: %v", err)
		return
	}
	defer rows.Close()

	var arenas []models.Arenas

	for rows.Next() {
		var arena models.Arenas
		err := rows.Scan(
			&arena.ID,
			&arena.Nome,
			&arena.Cnpj,
			&arena.QtdCampos,
			&arena.Tipo,
			&arena.Imagem,
			&arena.Endereco,
			&arena.Observacoes,
			&arena.EsportesOferecidos,
			&arena.InformacoesArena,
		)
		if err != nil {
			http.Error(w, "Erro ao ler dados", http.StatusInternalServerError)
			log.Printf("Erro ao escanear arenas: %v", err)
			return
		}
		arenas = append(arenas, arena)
	}

	if len(arenas) == 0 {
		http.Error(w, "Nenhuma arena encontrada", http.StatusNotFound)
		return
	}

	log.Printf("Arenas encontradas: %d", len(arenas))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(arenas)
}

func DeleteArena(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Nao foi possivel obter o ID do usuario do token", http.StatusInternalServerError)
		log.Printf("Nao foi possivel pegar ID do usuario do token")
		return
	}

	_, err := config.DB.Exec("DELETE FROM arenas WHERE id_usuario=$1", userID)
	if err != nil {
		log.Printf("Erro ao deletar arena do banco: %v", err)
		http.Error(w, "Erro ao deletar arena do banco", http.StatusInternalServerError)
		return
	}

	log.Printf("Arena de usuario com ID %d deletado com sucesso!", userID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Arena deletado com sucesso"})
}

func UpdateArena(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Nao foi possivel obter o ID do usuario do token", http.StatusInternalServerError)
		log.Printf("Nao foi possivel pegar ID do usuario do token")
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Erro ao processar formulario", http.StatusBadRequest)
		return
	}

	var arena models.Arenas
	arena.Nome = r.FormValue("nome")
	arena.Cnpj = r.FormValue("cnpj")
	arena.Tipo = r.FormValue("tipo")
	arena.Endereco = r.FormValue("endereco")
	arena.Observacoes = firstNonEmptyArenaValue(r.FormValue("observacoes"), r.FormValue("observações"))
	arena.EsportesOferecidos = firstNonEmptyArenaValue(
		r.FormValue("esportes_oferecidos"),
		r.FormValue("esportesOferecidos"),
		r.FormValue("esportes disponíveis"),
	)
	arena.InformacoesArena = firstNonEmptyArenaValue(
		r.FormValue("informacoes_arena"),
		r.FormValue("informacoesArena"),
	)
	fmt.Sscan(r.FormValue("qtdCampos"), &arena.QtdCampos)

	var cnpjInfo utils.CNPJInfo
	cnpjValidado := false
	var err error
	optionalColumns, err := loadArenaOptionalColumns(r.Context())
	if err != nil {
		http.Error(w, "Erro ao carregar estrutura de arenas", http.StatusInternalServerError)
		log.Printf("Erro ao carregar colunas opcionais de arenas: %v", err)
		return
	}

	arena.Cnpj = utils.NormalizeCNPJ(arena.Cnpj)
	if arena.Cnpj != "" && utils.IsCNPJValidationEnabled() {
		cnpjInfo, err = utils.ValidateExistingCNPJ(r.Context(), arena.Cnpj)
		if err != nil {
			writeCNPJValidationError(w, err)
			return
		}
		arena.Cnpj = cnpjInfo.CNPJ
		cnpjValidado = true
	}

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

	_, err = config.DB.Exec(
		buildArenaUpdateQuery(optionalColumns),
		buildArenaUpdateArgs(
			arena.Nome,
			arena.Cnpj,
			arena.QtdCampos,
			arena.Tipo,
			arena.Endereco,
			urlImagem,
			arena.Observacoes,
			arena.EsportesOferecidos,
			arena.InformacoesArena,
			userID,
			optionalColumns,
		)...,
	)
	if err != nil {
		log.Printf("Erro ao atualizar arena no banco: %v", err)
		http.Error(w, "Erro ao atualizar arena", http.StatusInternalServerError)
		return
	}

	log.Println("Arena atualizada com sucesso!")

	response := map[string]any{
		"message":             "Arena atualizada com sucesso",
		"imagem":              urlImagem,
		"observacoes":         arena.Observacoes,
		"esportes_oferecidos": arena.EsportesOferecidos,
		"informacoes_arena":   arena.InformacoesArena,
	}

	if strings.TrimSpace(arena.Cnpj) != "" {
		response["cnpj"] = arena.Cnpj
		response["cnpj_validado"] = cnpjValidado
		response["razao_social"] = cnpjInfo.RazaoSocial
		response["nome_fantasia"] = cnpjInfo.NomeFantasia
		response["situacao_cadastral"] = cnpjInfo.DescricaoSituacao
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func writeCNPJValidationError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	code := "cnpj_invalid"
	message := "CNPJ invalido"

	switch {
	case errors.Is(err, utils.ErrCNPJEmpty):
		status = http.StatusBadRequest
		code = "cnpj_required"
		message = "CNPJ nao informado"
	case errors.Is(err, utils.ErrCNPJInvalid):
		status = http.StatusBadRequest
		code = "cnpj_invalid"
		message = "CNPJ invalido"
	case errors.Is(err, utils.ErrCNPJNotFound):
		status = http.StatusUnprocessableEntity
		code = "cnpj_not_found"
		message = "CNPJ nao encontrado na consulta externa"
	case errors.Is(err, utils.ErrCNPJValidationUnavailable):
		status = http.StatusBadGateway
		code = "cnpj_validation_unavailable"
		message = "Nao foi possivel validar o CNPJ no momento"
	default:
		status = http.StatusBadGateway
		code = "cnpj_validation_error"
		message = "Erro ao validar CNPJ"
	}

	log.Printf("Falha na validacao de CNPJ: %v", err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"message": message,
		"code":    code,
		"field":   "cnpj",
	})
}

func atoi(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

func firstNonEmptyArenaValue(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}

	return ""
}
