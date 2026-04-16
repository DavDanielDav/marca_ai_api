package utils

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestNormalizeAndValidateCNPJ(t *testing.T) {
	if got := NormalizeCNPJ("12.345.678/0001-95"); got != "12345678000195" {
		t.Fatalf("expected normalized CNPJ, got %q", got)
	}

	if !IsValidCNPJ("11222333000181") {
		t.Fatal("expected known valid CNPJ to pass checksum validation")
	}

	if IsValidCNPJ("11111111111111") {
		t.Fatal("expected repeated digits CNPJ to be invalid")
	}

	if IsValidCNPJ("11222333000182") {
		t.Fatal("expected invalid checksum CNPJ to fail")
	}
}

func TestValidateExistingCNPJSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/11222333000181" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"cnpj":"11222333000181",
			"razao_social":"Arena Central LTDA",
			"nome_fantasia":"Arena Central",
			"descricao_situacao_cadastral":"ATIVA",
			"uf":"SP",
			"municipio":"Sao Paulo"
		}`))
	}))
	defer server.Close()

	previousBaseURL := os.Getenv("CNPJ_API_BASE_URL")
	t.Cleanup(func() {
		if previousBaseURL == "" {
			os.Unsetenv("CNPJ_API_BASE_URL")
			return
		}
		os.Setenv("CNPJ_API_BASE_URL", previousBaseURL)
	})
	os.Setenv("CNPJ_API_BASE_URL", server.URL)

	info, err := ValidateExistingCNPJ(context.Background(), "11.222.333/0001-81")
	if err != nil {
		t.Fatalf("ValidateExistingCNPJ returned error: %v", err)
	}

	if info.CNPJ != "11222333000181" {
		t.Fatalf("expected normalized CNPJ, got %q", info.CNPJ)
	}
	if info.RazaoSocial != "Arena Central LTDA" {
		t.Fatalf("expected razao social from API, got %q", info.RazaoSocial)
	}
	if info.DescricaoSituacao != "ATIVA" {
		t.Fatalf("expected situacao cadastral from API, got %q", info.DescricaoSituacao)
	}
}

func TestValidateExistingCNPJNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"CNPJ not found"}`))
	}))
	defer server.Close()

	previousBaseURL := os.Getenv("CNPJ_API_BASE_URL")
	t.Cleanup(func() {
		if previousBaseURL == "" {
			os.Unsetenv("CNPJ_API_BASE_URL")
			return
		}
		os.Setenv("CNPJ_API_BASE_URL", previousBaseURL)
	})
	os.Setenv("CNPJ_API_BASE_URL", server.URL)

	_, err := ValidateExistingCNPJ(context.Background(), "11.222.333/0001-81")
	if err == nil {
		t.Fatal("expected not found error")
	}
	if err != ErrCNPJNotFound {
		t.Fatalf("expected ErrCNPJNotFound, got %v", err)
	}
}

func TestIsCNPJValidationEnabled(t *testing.T) {
	previousFlag := os.Getenv("ENABLE_CNPJ_VALIDATION")
	t.Cleanup(func() {
		if previousFlag == "" {
			os.Unsetenv("ENABLE_CNPJ_VALIDATION")
			return
		}
		os.Setenv("ENABLE_CNPJ_VALIDATION", previousFlag)
	})

	os.Unsetenv("ENABLE_CNPJ_VALIDATION")
	if IsCNPJValidationEnabled() {
		t.Fatal("expected CNPJ validation to be disabled by default")
	}

	os.Setenv("ENABLE_CNPJ_VALIDATION", "true")
	if !IsCNPJValidationEnabled() {
		t.Fatal("expected CNPJ validation to be enabled when flag is true")
	}
}
