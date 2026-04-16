package utils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode"
)

const defaultCNPJAPIBaseURL = "https://brasilapi.com.br/api/cnpj/v1"
const cnpjValidationFlagEnv = "ENABLE_CNPJ_VALIDATION"

var (
	ErrCNPJEmpty                 = errors.New("cnpj nao informado")
	ErrCNPJInvalid               = errors.New("cnpj invalido")
	ErrCNPJNotFound              = errors.New("cnpj nao encontrado")
	ErrCNPJValidationUnavailable = errors.New("nao foi possivel validar o cnpj no momento")
	cnpjValidationHTTPClient     = &http.Client{Timeout: 10 * time.Second}
)

type CNPJInfo struct {
	CNPJ              string
	RazaoSocial       string
	NomeFantasia      string
	DescricaoSituacao string
	UF                string
	Municipio         string
}

type brasilAPICNPJResponse struct {
	CNPJ                       string `json:"cnpj"`
	RazaoSocial                string `json:"razao_social"`
	NomeFantasia               string `json:"nome_fantasia"`
	DescricaoSituacaoCadastral string `json:"descricao_situacao_cadastral"`
	UF                         string `json:"uf"`
	Municipio                  string `json:"municipio"`
	Message                    string `json:"message"`
	Type                       string `json:"type"`
	Name                       string `json:"name"`
}

func ValidateExistingCNPJ(ctx context.Context, rawCNPJ string) (CNPJInfo, error) {
	normalized := NormalizeCNPJ(rawCNPJ)
	if normalized == "" {
		return CNPJInfo{}, ErrCNPJEmpty
	}

	if !IsValidCNPJ(normalized) {
		return CNPJInfo{}, ErrCNPJInvalid
	}

	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("CNPJ_API_BASE_URL")), "/")
	if baseURL == "" {
		baseURL = defaultCNPJAPIBaseURL
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/"+normalized, nil)
	if err != nil {
		return CNPJInfo{}, fmt.Errorf("%w: %v", ErrCNPJValidationUnavailable, err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "marca-ai-backend/1.0")

	resp, err := cnpjValidationHTTPClient.Do(req)
	if err != nil {
		return CNPJInfo{}, fmt.Errorf("%w: %v", ErrCNPJValidationUnavailable, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return CNPJInfo{}, fmt.Errorf("%w: %v", ErrCNPJValidationUnavailable, err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return CNPJInfo{}, ErrCNPJNotFound
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		apiErrMessage := parseCNPJAPIErrorMessage(body)
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			if apiErrMessage != "" {
				return CNPJInfo{}, fmt.Errorf("%w: %s", ErrCNPJInvalid, apiErrMessage)
			}
			return CNPJInfo{}, ErrCNPJInvalid
		}

		if apiErrMessage != "" {
			return CNPJInfo{}, fmt.Errorf("%w: %s", ErrCNPJValidationUnavailable, apiErrMessage)
		}
		return CNPJInfo{}, fmt.Errorf("%w: status %d", ErrCNPJValidationUnavailable, resp.StatusCode)
	}

	var apiResponse brasilAPICNPJResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return CNPJInfo{}, fmt.Errorf("%w: %v", ErrCNPJValidationUnavailable, err)
	}

	return CNPJInfo{
		CNPJ:              firstNonEmpty(apiResponse.CNPJ, normalized),
		RazaoSocial:       strings.TrimSpace(apiResponse.RazaoSocial),
		NomeFantasia:      strings.TrimSpace(apiResponse.NomeFantasia),
		DescricaoSituacao: strings.TrimSpace(apiResponse.DescricaoSituacaoCadastral),
		UF:                strings.TrimSpace(apiResponse.UF),
		Municipio:         strings.TrimSpace(apiResponse.Municipio),
	}, nil
}

func IsCNPJValidationEnabled() bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(cnpjValidationFlagEnv)))

	switch value {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func NormalizeCNPJ(value string) string {
	var builder strings.Builder
	builder.Grow(len(value))

	for _, r := range value {
		if unicode.IsDigit(r) {
			builder.WriteRune(r)
		}
	}

	return builder.String()
}

func IsValidCNPJ(cnpj string) bool {
	cnpj = NormalizeCNPJ(cnpj)
	if len(cnpj) != 14 {
		return false
	}

	if allDigitsEqual(cnpj) {
		return false
	}

	firstDigit := calculateCNPJCheckDigit(cnpj[:12], []int{5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2})
	secondDigit := calculateCNPJCheckDigit(cnpj[:12]+string(rune('0'+firstDigit)), []int{6, 5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2})

	return int(cnpj[12]-'0') == firstDigit && int(cnpj[13]-'0') == secondDigit
}

func calculateCNPJCheckDigit(base string, weights []int) int {
	sum := 0
	for i := 0; i < len(base); i++ {
		sum += int(base[i]-'0') * weights[i]
	}

	remainder := sum % 11
	if remainder < 2 {
		return 0
	}

	return 11 - remainder
}

func allDigitsEqual(value string) bool {
	for i := 1; i < len(value); i++ {
		if value[i] != value[0] {
			return false
		}
	}

	return true
}

func parseCNPJAPIErrorMessage(body []byte) string {
	var apiErr brasilAPICNPJResponse
	if err := json.Unmarshal(body, &apiErr); err == nil {
		if msg := firstNonEmpty(apiErr.Message, apiErr.Type, apiErr.Name); msg != "" {
			return msg
		}
	}

	return strings.TrimSpace(string(body))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}

	return ""
}
