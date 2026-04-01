package utils

import (
	"fmt"
	"net/smtp"

	"github.com/danpi/marca_ai_backend/internal/config"
)

func SendEmail(to, subject, body string) error {
	return SendSMTPMail(to, subject, body)
}

func SendSMTPMail(to, subject, body string) error {
	host := config.SMTPHost()
	port := config.SMTPPort()
	user := config.SMTPUser()
	pass := config.SMTPPass()
	from := config.SMTPFrom()

	if host == "" || user == "" || pass == "" || from == "" {
		return fmt.Errorf("smtp nao configurado: defina SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASS e SMTP_FROM")
	}

	auth := smtp.PlainAuth("", user, pass, host)
	message := []byte(
		"From: " + from + "\r\n" +
			"To: " + to + "\r\n" +
			"Subject: " + subject + "\r\n" +
			"MIME-Version: 1.0\r\n" +
			"Content-Type: text/plain; charset=\"UTF-8\"\r\n\r\n" +
			body,
	)

	return smtp.SendMail(
		fmt.Sprintf("%s:%d", host, port),
		auth,
		from,
		[]string{to},
		message,
	)
}
