package handlers

import (
	"encoding/json"
	"testing"
)

func TestParseCampoHorariosRawSupportsJSONAndMap(t *testing.T) {
	horarios, err := parseCampoHorariosRaw(`["07:00","8:00","07:00","00:00"]`)
	if err != nil {
		t.Fatalf("parseCampoHorariosRaw returned error for array input: %v", err)
	}
	if len(horarios) != 3 {
		t.Fatalf("expected deduplicated horarios, got %v", horarios)
	}

	horarios, err = parseCampoHorariosRaw(`{"07:00":true,"08:00":1,"09:00":false}`)
	if err != nil {
		t.Fatalf("parseCampoHorariosRaw returned error for object input: %v", err)
	}
	if len(horarios) != 2 {
		t.Fatalf("expected only enabled horarios from object map, got %v", horarios)
	}
}

func TestDecodeCampoHorariosFallsBackToDefault(t *testing.T) {
	horarios := decodeCampoHorarios("")
	if len(horarios) == 0 {
		t.Fatal("expected default horarios when raw value is empty")
	}
	if horarios[0] != "07:00" {
		t.Fatalf("expected default horarios to start at 07:00, got %v", horarios)
	}
}

func TestExtractCampoHorariosRawFromJSONSupportsString(t *testing.T) {
	rawMessages := map[string]json.RawMessage{
		"horarios": json.RawMessage(`"07:00, 08:00"`),
	}

	raw, ok, err := extractCampoHorariosRawFromJSON(rawMessages)
	if err != nil {
		t.Fatalf("extractCampoHorariosRawFromJSON returned error: %v", err)
	}
	if !ok {
		t.Fatal("expected horarios to be detected")
	}
	if raw != "07:00, 08:00" {
		t.Fatalf("unexpected raw horarios value: %q", raw)
	}
}
