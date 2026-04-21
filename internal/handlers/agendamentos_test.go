package handlers

import (
	"strings"
	"testing"
	"time"
)

func TestFormatAgendamentoDateTimeDoesNotIncludeTimezone(t *testing.T) {
	formatted := formatAgendamentoDateTime(time.Date(2026, 4, 22, 17, 0, 0, 0, time.UTC))

	if formatted != "2026-04-22T17:00:00" {
		t.Fatalf("expected local wall-clock format, got %q", formatted)
	}

	if strings.Contains(formatted, "Z") || strings.Contains(formatted, "+") {
		t.Fatalf("expected formatted time without timezone suffix, got %q", formatted)
	}
}
