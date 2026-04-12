package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"strings"

	"github.com/danpi/marca_ai_backend/internal/config"
)

const resendAPIURL = "https://api.resend.com/emails"

type resendSendEmailRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Text    string   `json:"text,omitempty"`
}

type resendSendEmailResponse struct {
	ID      string `json:"id"`
	Message string `json:"message"`
	Name    string `json:"name"`
}

func SendEmail(to, subject, body string) error {
	if config.IsRenderEnvironment() {
		return SendResendMail(to, subject, body)
	}

	return SendSMTPMail(to, subject, body)
}

func SendSMTPMail(to, subject, body string) error {
	host := strings.TrimSpace(config.SMTPHost())
	port := config.SMTPPort()
	user := strings.TrimSpace(config.SMTPUser())
	pass := config.SMTPPass()
	from := strings.TrimSpace(config.SMTPFrom())

	if host == "" || user == "" || pass == "" || from == "" {
		return fmt.Errorf("smtp nao configurado: defina SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASS e SMTP_FROM")
	}

	auth := smtp.PlainAuth("", user, pass, host)
	message := []byte(
		"From: " + from + "\r\n" +
			"To: " + strings.TrimSpace(to) + "\r\n" +
			"Subject: " + subject + "\r\n" +
			"MIME-Version: 1.0\r\n" +
			"Content-Type: text/plain; charset=\"UTF-8\"\r\n\r\n" +
			body,
	)

	if err := smtp.SendMail(
		fmt.Sprintf("%s:%d", host, port),
		auth,
		from,
		[]string{strings.TrimSpace(to)},
		message,
	); err != nil {
		return fmt.Errorf("erro ao enviar email com smtp: %w", err)
	}

	return nil
}

func SendResendMail(to, subject, body string) error {
	apiKey, err := config.ResendKey()
	if err != nil {
		return err
	}

	from := strings.TrimSpace(config.ResendFromEmail())
	if from == "" {
		return fmt.Errorf("resend nao configurado: defina RESEND_FROM_EMAIL")
	}

	payload, err := json.Marshal(resendSendEmailRequest{
		From:    from,
		To:      []string{strings.TrimSpace(to)},
		Subject: subject,
		Text:    body,
	})
	if err != nil {
		return fmt.Errorf("erro ao serializar payload do resend: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, resendAPIURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("erro ao criar requisicao para o resend: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "marca-ai-backend/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("erro ao enviar email com resend: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("erro ao ler resposta do resend: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var resendErr resendSendEmailResponse
		if err := json.Unmarshal(responseBody, &resendErr); err == nil {
			if resendErr.Message != "" {
				return fmt.Errorf("resend respondeu com status %d: %s", resp.StatusCode, resendErr.Message)
			}
			if resendErr.Name != "" {
				return fmt.Errorf("resend respondeu com status %d: %s", resp.StatusCode, resendErr.Name)
			}
		}

		return fmt.Errorf("resend respondeu com status %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	return nil
}
