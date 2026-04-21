package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/danpi/marca_ai_backend/internal/config"
	"github.com/danpi/marca_ai_backend/internal/middleware"
	"github.com/danpi/marca_ai_backend/internal/models"
	"github.com/danpi/marca_ai_backend/internal/utils"
	"github.com/gorilla/mux"
)

type campoUpdatePayload struct {
	IDCampo      int
	Nome         string
	MaxJogadores int
	Modalidade   string
	TipoCampo    string
	Imagem       string
	IdArena      int
	ValorHora    *float64
	Ativo        *bool
	HorariosJSON *string
}

type campoUpdateJSONInput struct {
	IDCampo         int      `json:"id_campo"`
	IDCampoAlt      int      `json:"idCampo"`
	ID              int      `json:"id"`
	Nome            string   `json:"nome_campo"`
	NomeAlt         string   `json:"nome"`
	MaxJogadores    int      `json:"max_jogadores"`
	MaxJogadoresAlt int      `json:"maxJogadores"`
	Modalidade      string   `json:"modalidade"`
	TipoCampo       string   `json:"tipo_campo"`
	TipoCampoAlt    string   `json:"tipoCampo"`
	Imagem          string   `json:"imagem"`
	IdArena         int      `json:"id_arena"`
	IdArenaAlt      int      `json:"idArena"`
	ValorHora       *float64 `json:"valor_hora"`
	ValorHoraAlt    *float64 `json:"valorHora"`
	Ativo           *bool    `json:"ativo"`
	EmManutencao    *bool    `json:"em_manutencao"`
}

type campoMaintenancePayload struct {
	IDCampo    int   `json:"id_campo"`
	IDCampoAlt int   `json:"idCampo"`
	ID         int   `json:"id"`
	Ativo      *bool `json:"ativo"`
}

func CadastrodeCampo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		log.Printf("Metodo nao autorizado")
		return
	}

	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Nao foi possivel obter o ID do usuario", http.StatusUnauthorized)
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Erro ao processar form data", http.StatusBadRequest)
		return
	}

	nome := r.FormValue("nome_campo")
	maxJogadores := r.FormValue("maxJogadores")
	modalidade := r.FormValue("modalidade")
	tipoCampo := r.FormValue("tipoCampo")
	idArenaStr := r.FormValue("idArena")
	valorHoraStr := firstNonEmptyCampoValue(r.FormValue("valor_hora"), r.FormValue("valorHora"))
	ativoStr := r.FormValue("ativo")
	horariosRaw, horariosProvided := extractCampoHorariosRawFromForm(r.FormValue)

	idArena, err := strconv.Atoi(idArenaStr)
	if err != nil {
		http.Error(w, "ID da arena invalido", http.StatusBadRequest)
		return
	}

	valorHora, err := parseOptionalFloat64(valorHoraStr)
	if err != nil {
		http.Error(w, "Valor da hora invalido", http.StatusBadRequest)
		return
	}

	ativo := true
	if strings.TrimSpace(ativoStr) != "" {
		ativo, err = strconv.ParseBool(strings.TrimSpace(ativoStr))
		if err != nil {
			http.Error(w, "Status ativo invalido", http.StatusBadRequest)
			return
		}
	}

	var pertence bool
	err = config.DB.QueryRow(fmt.Sprintf(`
		SELECT EXISTS (
			SELECT 1 FROM %s WHERE id = $1 AND id_usuario = $2
		)
	`, arenasTableName()), idArena, userID).Scan(&pertence)
	if err != nil {
		http.Error(w, "Erro ao verificar arena", http.StatusInternalServerError)
		return
	}
	if !pertence {
		http.Error(w, "Arena nao pertence ao usuario logado", http.StatusForbidden)
		return
	}

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
	horarios := defaultCampoHorarios()
	if horariosProvided {
		horarios, err = parseCampoHorariosRaw(horariosRaw)
		if err != nil {
			http.Error(w, "Horarios invalidos", http.StatusBadRequest)
			return
		}
		if len(horarios) == 0 {
			horarios = defaultCampoHorarios()
		}
	}

	newCampo := models.Campo{
		Nome:         nome,
		MaxJogadores: maxJogadoresInt,
		Modalidade:   modalidade,
		TipoCampo:    tipoCampo,
		Imagem:       urlImagem,
		ValorHora:    valorHora,
		Ativo:        ativo,
		IdArena:      idArena,
	}
	applyCampoHorarioAliases(&newCampo, horarios)

	optionalColumns, err := loadCampoOptionalColumns(r.Context())
	if err != nil {
		http.Error(w, "Erro ao carregar estrutura de campos", http.StatusInternalServerError)
		log.Printf("Erro ao carregar colunas opcionais de campos: %v", err)
		return
	}

	_, err = config.DB.Exec(
		buildCampoInsertQuery(optionalColumns),
		buildCampoInsertArgs(
			newCampo.Nome,
			newCampo.Modalidade,
			newCampo.TipoCampo,
			newCampo.Imagem,
			newCampo.MaxJogadores,
			newCampo.IdArena,
			newCampo.ValorHora,
			newCampo.Ativo,
			serializeCampoHorarios(newCampo.HorariosDisponiveis),
			optionalColumns,
		)...,
	)
	if err != nil {
		log.Printf("Erro ao inserir campo no banco: %v", err)
		http.Error(w, "Erro ao registrar campo", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"message":              "Campo cadastrado com sucesso",
		"imagem":               newCampo.Imagem,
		"valor_hora":           newCampo.ValorHora,
		"ativo":                newCampo.Ativo,
		"horarios":             newCampo.Horarios,
		"horarios_disponiveis": newCampo.HorariosDisponiveis,
		"horariosDisponiveis":  newCampo.HorariosDisponiveisCamel,
		"horarios_campo":       newCampo.HorariosCampo,
	})
}

func GetCampos(w http.ResponseWriter, r *http.Request) {
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

	optionalColumns, err := loadCampoOptionalColumns(r.Context())
	if err != nil {
		http.Error(w, "Erro ao carregar estrutura de campos", http.StatusInternalServerError)
		log.Printf("Erro ao carregar colunas opcionais de campos: %v", err)
		return
	}

	rowsArenas, err := config.DB.Query(fmt.Sprintf(`SELECT id FROM %s WHERE id_usuario = $1`, arenasTableName()), userID)
	if err != nil {
		http.Error(w, "Erro ao buscar arenas do usuario", http.StatusInternalServerError)
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

	if len(arenaIDs) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]models.Campo{})
		return
	}

	query := fmt.Sprintf(`
		SELECT
			c.id_campo,
			c.nome_campo,
			c.max_jogadores,
			c.modalidade,
			c.tipo_campo,
			c.imagem,
			`+optionalCampoSelectExpression("c", "valor_hora", optionalColumns.ValorHora)+`,
			`+optionalCampoSelectExpression("c", "ativo", optionalColumns.Ativo)+`,
			`+optionalCampoSelectExpression("c", "horarios_disponiveis", optionalColumns.HorariosDisponiveis)+`,
			c.id_arena,
			a.nome AS nome_arena
		FROM %s c
		JOIN %s a ON c.id_arena = a.id
		WHERE a.id_usuario = $1;
	`, campoTableName(), arenasTableName())

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
		var horariosRaw string
		err := rowsCampos.Scan(
			&campo.IDCampo,
			&campo.Nome,
			&campo.MaxJogadores,
			&campo.Modalidade,
			&campo.TipoCampo,
			&campo.Imagem,
			&campo.ValorHora,
			&campo.Ativo,
			&horariosRaw,
			&campo.IdArena,
			&campo.NomeArena,
		)
		if err != nil {
			http.Error(w, "Erro ao ler campos", http.StatusInternalServerError)
			log.Printf("Erro ao escanear campos: %v", err)
			return
		}
		campo.EmManutencao = !campo.Ativo
		applyCampoHorarioAliases(&campo, decodeCampoHorarios(horariosRaw))
		campos = append(campos, campo)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(campos)
}

func UpdateCampo(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Nao foi possivel obter o ID do usuario", http.StatusUnauthorized)
		log.Printf("Nao foi possivel obter o ID do usuario")
		return
	}

	payload, err := parseCampoUpdatePayload(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Printf("Erro ao ler corpo da requisicao de campo: %v", err)
		return
	}

	idCampo, err := resolveCampoID(r, payload.IDCampo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Printf("Erro ao resolver ID do campo: %v", err)
		return
	}

	var pertence bool
	err = config.DB.QueryRow(fmt.Sprintf(`
		SELECT EXISTS (
			SELECT 1
			FROM %s c
			JOIN %s a ON c.id_arena = a.id
			WHERE c.id_campo = $1 AND a.id_usuario = $2
		)
	`, campoTableName(), arenasTableName()), idCampo, userID).Scan(&pertence)
	if err != nil {
		log.Printf("Erro ao verificar propriedade do campo: %v", err)
		http.Error(w, "Erro ao verificar propriedade do campo", http.StatusInternalServerError)
		return
	}
	if !pertence {
		http.Error(w, "O campo nao pertence ao usuario", http.StatusForbidden)
		log.Printf("O campo %d nao pertence ao usuario %d", idCampo, userID)
		return
	}

	if payload.IdArena > 0 {
		var arenaPertence bool
		err = config.DB.QueryRow(fmt.Sprintf(`
			SELECT EXISTS (
				SELECT 1
				FROM %s
				WHERE id = $1 AND id_usuario = $2
			)
		`, arenasTableName()), payload.IdArena, userID).Scan(&arenaPertence)
		if err != nil {
			log.Printf("Erro ao verificar arena de destino do campo: %v", err)
			http.Error(w, "Erro ao verificar arena do campo", http.StatusInternalServerError)
			return
		}
		if !arenaPertence {
			http.Error(w, "A arena selecionada nao pertence ao usuario", http.StatusForbidden)
			log.Printf("A arena %d nao pertence ao usuario %d", payload.IdArena, userID)
			return
		}
	}

	optionalColumns, err := loadCampoOptionalColumns(r.Context())
	if err != nil {
		http.Error(w, "Erro ao carregar estrutura de campos", http.StatusInternalServerError)
		log.Printf("Erro ao carregar colunas opcionais de campos: %v", err)
		return
	}

	_, err = config.DB.Exec(fmt.Sprintf(`
		UPDATE %s
		SET
			nome_campo = COALESCE(NULLIF($1, ''), nome_campo),
			max_jogadores = CASE WHEN $2 > 0 THEN $2 ELSE max_jogadores END,
			modalidade = COALESCE(NULLIF($3, ''), modalidade),
			tipo_campo = COALESCE(NULLIF($4, ''), tipo_campo),
			imagem = COALESCE(NULLIF($5, ''), imagem),
			id_arena = CASE WHEN $6 > 0 THEN $6 ELSE id_arena END
		WHERE id_campo = $7
	`, campoTableName()), payload.Nome, payload.MaxJogadores, payload.Modalidade, payload.TipoCampo, payload.Imagem, payload.IdArena, idCampo)
	if err != nil {
		log.Printf("Erro ao atualizar campo: %v", err)
		http.Error(w, "Erro ao atualizar campo", http.StatusInternalServerError)
		return
	}

	if err := updateCampoOptionalFields(idCampo, payload, optionalColumns); err != nil {
		log.Printf("Erro ao atualizar dados adicionais do campo %d: %v", idCampo, err)
		http.Error(w, "Erro ao atualizar dados adicionais do campo", http.StatusInternalServerError)
		return
	}

	response := map[string]any{
		"message": "Campo atualizado com sucesso",
		"imagem":  payload.Imagem,
	}
	if payload.ValorHora != nil {
		response["valor_hora"] = *payload.ValorHora
	}
	if payload.Ativo != nil {
		response["ativo"] = *payload.Ativo
		response["em_manutencao"] = !*payload.Ativo
	}
	if payload.HorariosJSON != nil {
		horarios := decodeCampoHorarios(*payload.HorariosJSON)
		response["horarios"] = horarios
		response["horarios_disponiveis"] = horarios
		response["horariosDisponiveis"] = horarios
		response["horarios_campo"] = horarios
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func AtualizarManutencaoCampo(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Nao foi possivel obter o ID do usuario", http.StatusUnauthorized)
		return
	}

	optionalColumns, err := loadCampoOptionalColumns(r.Context())
	if err != nil {
		http.Error(w, "Erro ao carregar estrutura de campos", http.StatusInternalServerError)
		log.Printf("Erro ao carregar colunas opcionais de campos: %v", err)
		return
	}

	if !optionalColumns.Ativo {
		http.Error(w, "Coluna ativo nao disponivel. Aplique a migration antes de usar manutencao.", http.StatusInternalServerError)
		log.Printf("Tentativa de usar manutencao de campo sem a coluna ativo")
		return
	}

	payload, err := parseCampoMaintenancePayload(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Printf("Erro ao ler payload de manutencao do campo: %v", err)
		return
	}

	idCampo, err := resolveCampoID(r, firstNonZero(payload.IDCampo, payload.IDCampoAlt, payload.ID))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := ensureCampoBelongsToUser(idCampo, userID); err != nil {
		switch {
		case errors.Is(err, errCampoOwnershipForbidden):
			http.Error(w, "O campo nao pertence ao usuario", http.StatusForbidden)
		default:
			http.Error(w, "Erro ao verificar propriedade do campo", http.StatusInternalServerError)
		}
		log.Printf("Erro ao validar manutencao do campo %d para usuario %d: %v", idCampo, userID, err)
		return
	}

	currentStatus, err := getCampoAtivoStatus(idCampo)
	if err != nil {
		http.Error(w, "Erro ao obter status atual do campo", http.StatusInternalServerError)
		log.Printf("Erro ao obter status atual do campo %d: %v", idCampo, err)
		return
	}

	desiredStatus := !currentStatus
	if payload.Ativo != nil {
		desiredStatus = *payload.Ativo
	}

	_, err = config.DB.Exec(fmt.Sprintf(`UPDATE %s SET ativo = $1 WHERE id_campo = $2`, campoTableName()), desiredStatus, idCampo)
	if err != nil {
		http.Error(w, "Erro ao atualizar manutencao do campo", http.StatusInternalServerError)
		log.Printf("Erro ao atualizar coluna ativo do campo %d: %v", idCampo, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"message":       "Status de manutencao do campo atualizado com sucesso",
		"id_campo":      idCampo,
		"ativo":         desiredStatus,
		"status":        campoStatusLabel(desiredStatus),
		"em_manutencao": !desiredStatus,
	})
}

func parseCampoUpdatePayload(r *http.Request) (campoUpdatePayload, error) {
	if isMultipartCampoRequest(r) {
		return parseCampoUpdateMultipart(r)
	}

	return parseCampoUpdateJSON(r)
}

func parseCampoUpdateMultipart(r *http.Request) (campoUpdatePayload, error) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		return campoUpdatePayload{}, errors.New("Erro ao processar form data")
	}

	payload := campoUpdatePayload{
		Nome:       firstNonEmptyCampoValue(r.FormValue("nome_campo"), r.FormValue("nome")),
		Modalidade: strings.TrimSpace(r.FormValue("modalidade")),
		TipoCampo:  firstNonEmptyCampoValue(r.FormValue("tipoCampo"), r.FormValue("tipo_campo")),
	}

	idCampo, err := parseOptionalInt(firstNonEmptyCampoValue(r.FormValue("idCampo"), r.FormValue("id_campo"), r.FormValue("id")))
	if err != nil {
		return campoUpdatePayload{}, errors.New("ID do campo invalido")
	}
	payload.IDCampo = idCampo

	maxJogadores, err := parseOptionalInt(firstNonEmptyCampoValue(r.FormValue("maxJogadores"), r.FormValue("max_jogadores")))
	if err != nil {
		return campoUpdatePayload{}, errors.New("Maximo de jogadores invalido")
	}
	payload.MaxJogadores = maxJogadores

	idArena, err := parseOptionalInt(firstNonEmptyCampoValue(r.FormValue("idArena"), r.FormValue("id_arena")))
	if err != nil {
		return campoUpdatePayload{}, errors.New("ID da arena invalido")
	}
	payload.IdArena = idArena

	if valorHoraStr := firstNonEmptyCampoValue(r.FormValue("valor_hora"), r.FormValue("valorHora")); valorHoraStr != "" {
		valorHora, err := parseOptionalFloat64(valorHoraStr)
		if err != nil {
			return campoUpdatePayload{}, errors.New("Valor da hora invalido")
		}
		payload.ValorHora = &valorHora
	}

	if ativoStr := strings.TrimSpace(r.FormValue("ativo")); ativoStr != "" {
		ativo, err := strconv.ParseBool(ativoStr)
		if err != nil {
			return campoUpdatePayload{}, errors.New("Status ativo invalido")
		}
		payload.Ativo = &ativo
	}

	horariosRaw, horariosProvided := extractCampoHorariosRawFromForm(r.FormValue)
	if horariosProvided {
		horarios, err := parseCampoHorariosRaw(horariosRaw)
		if err != nil {
			return campoUpdatePayload{}, errors.New("Horarios invalidos")
		}
		if len(horarios) == 0 {
			horarios = defaultCampoHorarios()
		}
		serialized := serializeCampoHorarios(horarios)
		payload.HorariosJSON = &serialized
	}

	file, fileHeader, err := r.FormFile("imagem")
	switch {
	case err == nil:
		defer file.Close()

		urlImagem, uploadErr := utils.UploadCloudinary(file, fileHeader.Filename)
		if uploadErr != nil {
			return campoUpdatePayload{}, errors.New("Erro ao enviar imagem: " + uploadErr.Error())
		}

		payload.Imagem = urlImagem
	case errors.Is(err, http.ErrMissingFile):
	default:
		return campoUpdatePayload{}, errors.New("Erro ao ler imagem")
	}

	return payload, nil
}

func parseCampoUpdateJSON(r *http.Request) (campoUpdatePayload, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return campoUpdatePayload{}, errors.New("Erro ao ler corpo da requisicao")
	}

	var input campoUpdateJSONInput
	if err := json.Unmarshal(body, &input); err != nil {
		return campoUpdatePayload{}, errors.New("Erro ao decodificar corpo da requisicao")
	}

	payload := campoUpdatePayload{
		IDCampo:      firstNonZero(input.IDCampo, input.IDCampoAlt, input.ID),
		Nome:         firstNonEmptyCampoValue(input.Nome, input.NomeAlt),
		MaxJogadores: firstNonZero(input.MaxJogadores, input.MaxJogadoresAlt),
		Modalidade:   strings.TrimSpace(input.Modalidade),
		TipoCampo:    firstNonEmptyCampoValue(input.TipoCampo, input.TipoCampoAlt),
		Imagem:       strings.TrimSpace(input.Imagem),
		IdArena:      firstNonZero(input.IdArena, input.IdArenaAlt),
	}

	switch {
	case input.ValorHora != nil:
		payload.ValorHora = input.ValorHora
	case input.ValorHoraAlt != nil:
		payload.ValorHora = input.ValorHoraAlt
	}

	switch {
	case input.Ativo != nil:
		payload.Ativo = input.Ativo
	case input.EmManutencao != nil:
		ativo := !*input.EmManutencao
		payload.Ativo = &ativo
	}

	var rawMessages map[string]json.RawMessage
	if err := json.Unmarshal(body, &rawMessages); err != nil {
		return campoUpdatePayload{}, errors.New("Erro ao decodificar corpo da requisicao")
	}

	horariosRaw, horariosProvided, err := extractCampoHorariosRawFromJSON(rawMessages)
	if err != nil {
		return campoUpdatePayload{}, errors.New("Horarios invalidos")
	}
	if horariosProvided {
		horarios, err := parseCampoHorariosRaw(horariosRaw)
		if err != nil {
			return campoUpdatePayload{}, errors.New("Horarios invalidos")
		}
		if len(horarios) == 0 {
			horarios = defaultCampoHorarios()
		}
		serialized := serializeCampoHorarios(horarios)
		payload.HorariosJSON = &serialized
	}

	return payload, nil
}

func parseCampoMaintenancePayload(r *http.Request) (campoMaintenancePayload, error) {
	payload := campoMaintenancePayload{}

	if strings.TrimSpace(r.URL.Query().Get("ativo")) != "" {
		ativo, err := strconv.ParseBool(strings.TrimSpace(r.URL.Query().Get("ativo")))
		if err != nil {
			return campoMaintenancePayload{}, errors.New("Status ativo invalido")
		}
		payload.Ativo = &ativo
	}

	if r.Body == nil || r.Body == http.NoBody {
		return payload, nil
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		if errors.Is(err, http.ErrBodyNotAllowed) || strings.Contains(strings.ToLower(err.Error()), "eof") {
			return payload, nil
		}
		return campoMaintenancePayload{}, errors.New("Erro ao decodificar corpo da requisicao")
	}

	return payload, nil
}

func resolveCampoID(r *http.Request, fallbackID int) (int, error) {
	vars := mux.Vars(r)
	if idCampoStr, ok := vars["id"]; ok && strings.TrimSpace(idCampoStr) != "" {
		idCampo, err := strconv.Atoi(idCampoStr)
		if err != nil {
			return 0, errors.New("ID do campo invalido")
		}
		return idCampo, nil
	}

	if idCampoStr := strings.TrimSpace(r.URL.Query().Get("id")); idCampoStr != "" {
		idCampo, err := strconv.Atoi(idCampoStr)
		if err != nil {
			return 0, errors.New("ID do campo invalido")
		}
		return idCampo, nil
	}

	if fallbackID <= 0 {
		return 0, errors.New("ID do campo nao fornecido")
	}

	return fallbackID, nil
}

func parseOptionalInt(value string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}

	number, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}

	return number, nil
}

func firstNonEmptyCampoValue(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}

	return ""
}

func isMultipartCampoRequest(r *http.Request) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type"))), "multipart/form-data")
}

var errCampoOwnershipForbidden = errors.New("campo nao pertence ao usuario")

func ensureCampoBelongsToUser(idCampo, userID int) error {
	var pertence bool
	err := config.DB.QueryRow(fmt.Sprintf(`
		SELECT EXISTS (
			SELECT 1
			FROM %s c
			JOIN %s a ON c.id_arena = a.id
			WHERE c.id_campo = $1 AND a.id_usuario = $2
		)
	`, campoTableName(), arenasTableName()), idCampo, userID).Scan(&pertence)
	if err != nil {
		return err
	}
	if !pertence {
		return errCampoOwnershipForbidden
	}

	return nil
}

func getCampoAtivoStatus(idCampo int) (bool, error) {
	var ativo bool
	err := config.DB.QueryRow(fmt.Sprintf(`SELECT ativo FROM %s WHERE id_campo = $1`, campoTableName()), idCampo).Scan(&ativo)
	if err != nil {
		return false, err
	}

	return ativo, nil
}

func parseOptionalFloat64(value string) (float64, error) {
	value = strings.TrimSpace(strings.ReplaceAll(value, ",", "."))
	if value == "" {
		return 0, nil
	}

	number, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, err
	}

	return number, nil
}

func campoStatusLabel(ativo bool) string {
	if ativo {
		return "ativo"
	}

	return "inativo"
}

func firstNonZero(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}

	return 0
}

func updateCampoOptionalFields(idCampo int, payload campoUpdatePayload, columns campoOptionalColumns) error {
	assignments := make([]string, 0, 3)
	args := []any{idCampo}
	placeholder := 2

	if columns.ValorHora && payload.ValorHora != nil {
		assignments = append(assignments, fmt.Sprintf("valor_hora = $%d", placeholder))
		args = append(args, *payload.ValorHora)
		placeholder++
	}

	if columns.Ativo && payload.Ativo != nil {
		assignments = append(assignments, fmt.Sprintf("ativo = $%d", placeholder))
		args = append(args, *payload.Ativo)
		placeholder++
	}

	if columns.HorariosDisponiveis && payload.HorariosJSON != nil {
		assignments = append(assignments, fmt.Sprintf("horarios_disponiveis = $%d::jsonb", placeholder))
		args = append(args, *payload.HorariosJSON)
		placeholder++
	}

	if len(assignments) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE %s SET %s WHERE id_campo = $1", campoTableName(), strings.Join(assignments, ", "))
	_, err := config.DB.Exec(query, args...)
	return err
}

func DeleteCampo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idCampoStr, ok := vars["id"]
	if !ok {
		http.Error(w, "ID do campo nao fornecido", http.StatusBadRequest)
		return
	}

	idCampo, err := strconv.Atoi(idCampoStr)
	if err != nil {
		http.Error(w, "ID do campo invalido", http.StatusBadRequest)
		return
	}

	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Nao foi possivel obter o ID do usuario", http.StatusUnauthorized)
		return
	}

	var pertence bool
	err = config.DB.QueryRow(fmt.Sprintf(`
		SELECT EXISTS (
			SELECT 1
			FROM %s c
			JOIN %s a ON c.id_arena = a.id
			WHERE c.id_campo = $1 AND a.id_usuario = $2
		)
	`, campoTableName(), arenasTableName()), idCampo, userID).Scan(&pertence)
	if err != nil {
		http.Error(w, "Erro ao verificar propriedade do campo", http.StatusInternalServerError)
		return
	}
	if !pertence {
		http.Error(w, "O campo nao pertence ao usuario", http.StatusForbidden)
		return
	}

	var countNaoPermitidos int
	err = config.DB.QueryRow(fmt.Sprintf(`
		SELECT COUNT(*)
		FROM %s
		WHERE id_campo = $1
		  AND status NOT IN ('concluido', 'cancelado')
	`, agendamentosTableName()), idCampo).Scan(&countNaoPermitidos)
	if err != nil {
		http.Error(w, "Erro ao verificar agendamentos do campo", http.StatusInternalServerError)
		return
	}
	if countNaoPermitidos > 0 {
		http.Error(w, "Nao e possivel excluir o campo: existem agendamentos ativos vinculados", http.StatusBadRequest)
		return
	}

	_, err = config.DB.Exec(fmt.Sprintf(`
		DELETE FROM %s
		WHERE id_campo = $1
		  AND status IN ('cancelado', 'concluido')
	`, agendamentosTableName()), idCampo)
	if err != nil {
		http.Error(w, "Erro ao excluir agendamentos cancelados", http.StatusInternalServerError)
		return
	}

	_, err = config.DB.Exec(fmt.Sprintf("DELETE FROM %s WHERE id_campo = $1", campoTableName()), idCampo)
	if err != nil {
		http.Error(w, "Erro ao deletar campo", http.StatusInternalServerError)
		log.Println("ERRO AO DELETAR:", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Campo deletado com sucesso"})
}
