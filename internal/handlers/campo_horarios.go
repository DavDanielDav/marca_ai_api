package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/danpi/marca_ai_backend/internal/models"
)

var campoHorarioInputKeys = []string{
	"horarios",
	"horariosDisponiveis",
	"horarios_disponiveis",
	"horarios_disponivel",
	"horarios_campo",
	"horario_disponivel",
	"disponibilidade",
	"agenda",
	"horario",
	"hora",
}

func extractCampoHorariosRawFromForm(getValue func(string) string) (string, bool) {
	for _, key := range campoHorarioInputKeys {
		value := strings.TrimSpace(getValue(key))
		if value != "" {
			return value, true
		}
	}

	return "", false
}

func extractCampoHorariosRawFromJSON(rawMessages map[string]json.RawMessage) (string, bool, error) {
	for _, key := range campoHorarioInputKeys {
		message, exists := rawMessages[key]
		if !exists {
			continue
		}

		raw, ok, err := normalizeCampoHorarioRawJSON(message)
		if err != nil {
			return "", false, err
		}
		if ok {
			return raw, true, nil
		}
	}

	return "", false, nil
}

func normalizeCampoHorarioRawJSON(message json.RawMessage) (string, bool, error) {
	trimmed := bytes.TrimSpace(message)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return "", false, nil
	}

	if trimmed[0] == '"' {
		var value string
		if err := json.Unmarshal(trimmed, &value); err != nil {
			return "", false, err
		}

		value = strings.TrimSpace(value)
		if value == "" {
			return "", false, nil
		}

		return value, true, nil
	}

	return string(trimmed), true, nil
}

func parseCampoHorariosRaw(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	horarios, err := parseCampoHorariosJSON(raw)
	if err == nil && len(horarios) > 0 {
		return normalizeCampoHorarios(horarios)
	}

	parts := splitCampoHorarioText(raw)
	if len(parts) == 0 {
		return nil, nil
	}

	return normalizeCampoHorarios(parts)
}

func parseCampoHorariosJSON(raw string) ([]string, error) {
	var stringSlice []string
	if err := json.Unmarshal([]byte(raw), &stringSlice); err == nil {
		return stringSlice, nil
	}

	var anySlice []any
	if err := json.Unmarshal([]byte(raw), &anySlice); err == nil {
		horarios := make([]string, 0, len(anySlice))
		for _, item := range anySlice {
			horarios = append(horarios, stringifyHorarioCandidate(item)...)
		}
		return horarios, nil
	}

	var anyMap map[string]any
	if err := json.Unmarshal([]byte(raw), &anyMap); err == nil {
		horarios := make([]string, 0, len(anyMap))
		keys := make([]string, 0, len(anyMap))
		for key := range anyMap {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		for _, key := range keys {
			value := anyMap[key]
			if horario := strings.TrimSpace(key); horario != "" && horarioEntryEnabled(value) {
				horarios = append(horarios, horario)
				continue
			}
			horarios = append(horarios, stringifyHorarioCandidate(value)...)
		}

		return horarios, nil
	}

	return nil, fmt.Errorf("horarios is not valid JSON")
}

func stringifyHorarioCandidate(value any) []string {
	switch typed := value.(type) {
	case string:
		if strings.TrimSpace(typed) == "" {
			return nil
		}
		return []string{typed}
	case float64:
		return []string{strconv.FormatFloat(typed, 'f', -1, 64)}
	case int:
		return []string{strconv.Itoa(typed)}
	case []any:
		horarios := make([]string, 0, len(typed))
		for _, item := range typed {
			horarios = append(horarios, stringifyHorarioCandidate(item)...)
		}
		return horarios
	default:
		return nil
	}
}

func horarioEntryEnabled(value any) bool {
	switch typed := value.(type) {
	case nil:
		return true
	case bool:
		return typed
	case string:
		trimmed := strings.TrimSpace(strings.ToLower(typed))
		return trimmed == "" || trimmed == "true" || trimmed == "1" || trimmed == "ativo" || trimmed == "disponivel"
	case float64:
		return typed != 0
	default:
		return false
	}
}

func splitCampoHorarioText(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	replacer := strings.NewReplacer(";", ",", "|", ",", "\n", ",", "\r", ",")
	raw = replacer.Replace(raw)
	parts := strings.Split(raw, ",")

	horarios := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		horarios = append(horarios, part)
	}

	if len(horarios) == 0 && raw != "" {
		return []string{raw}
	}

	return horarios
}

func normalizeCampoHorarios(rawHorarios []string) ([]string, error) {
	unique := make(map[string]struct{}, len(rawHorarios))
	normalized := make([]string, 0, len(rawHorarios))

	for _, raw := range rawHorarios {
		horario, err := normalizeCampoHorario(raw)
		if err != nil {
			return nil, err
		}
		if _, exists := unique[horario]; exists {
			continue
		}
		unique[horario] = struct{}{}
		normalized = append(normalized, horario)
	}

	sort.Slice(normalized, func(i, j int) bool {
		return campoHorarioSortValue(normalized[i]) < campoHorarioSortValue(normalized[j])
	})

	return normalized, nil
}

func normalizeCampoHorario(raw string) (string, error) {
	raw = strings.TrimSpace(strings.ToLower(raw))
	raw = strings.ReplaceAll(raw, "h", ":")
	raw = strings.ReplaceAll(raw, ".", ":")

	if raw == "" {
		return "", fmt.Errorf("horario vazio")
	}

	if !strings.Contains(raw, ":") {
		switch len(raw) {
		case 1, 2:
			raw += ":00"
		case 3:
			raw = "0" + raw[:1] + ":" + raw[1:]
		case 4:
			raw = raw[:2] + ":" + raw[2:]
		default:
			return "", fmt.Errorf("horario invalido: %s", raw)
		}
	}

	parts := strings.Split(raw, ":")
	if len(parts) != 2 {
		return "", fmt.Errorf("horario invalido: %s", raw)
	}

	hour, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return "", fmt.Errorf("horario invalido: %s", raw)
	}

	minute, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return "", fmt.Errorf("horario invalido: %s", raw)
	}

	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return "", fmt.Errorf("horario invalido: %s", raw)
	}

	return fmt.Sprintf("%02d:%02d", hour, minute), nil
}

func campoHorarioSortValue(horario string) int {
	parts := strings.Split(horario, ":")
	if len(parts) != 2 {
		return 0
	}

	hour, _ := strconv.Atoi(parts[0])
	minute, _ := strconv.Atoi(parts[1])

	if hour < 6 {
		hour += 24
	}

	return hour*60 + minute
}

func defaultCampoHorarios() []string {
	horarios := make([]string, 0, 18)
	for hour := 7; hour <= 23; hour++ {
		horarios = append(horarios, fmt.Sprintf("%02d:00", hour))
	}
	horarios = append(horarios, "00:00")
	return horarios
}

func serializeCampoHorarios(horarios []string) string {
	if len(horarios) == 0 {
		return "[]"
	}

	encoded, err := json.Marshal(horarios)
	if err != nil {
		return "[]"
	}

	return string(encoded)
}

func decodeCampoHorarios(raw string) []string {
	horarios, err := parseCampoHorariosRaw(raw)
	if err != nil || len(horarios) == 0 {
		return defaultCampoHorarios()
	}

	return horarios
}

func applyCampoHorarioAliases(campo *models.Campo, horarios []string) {
	if campo == nil {
		return
	}

	if len(horarios) == 0 {
		horarios = defaultCampoHorarios()
	}

	aliases := append([]string(nil), horarios...)
	campo.Horarios = aliases
	campo.HorariosDisponiveis = append([]string(nil), aliases...)
	campo.HorariosDisponiveisCamel = append([]string(nil), aliases...)
	campo.HorariosCampo = append([]string(nil), aliases...)
}
