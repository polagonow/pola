package mailer

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"mime"
	"mime/multipart"
	"net"
	"net/smtp"
	"net/textproto"
	"strings"

	"github.com/polagonow/pola/core"
)

// SMTPConfig holds SMTP connection settings.
type SMTPConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	TLS      bool
}

// SMTPTransport delivers emails via SMTP.
type SMTPTransport struct {
	cfg SMTPConfig
}

// NewSMTPTransport creates a new SMTP transport.
func NewSMTPTransport(cfg SMTPConfig) *SMTPTransport {
	return &SMTPTransport{cfg: cfg}
}

func (t *SMTPTransport) Name() string { return "smtp" }

func (t *SMTPTransport) Send(_ context.Context, msg *core.MailMessage) error {
	addr := net.JoinHostPort(t.cfg.Host, t.cfg.Port)

	body, err := buildMIME(msg)
	if err != nil {
		return fmt.Errorf("smtp: build mime: %w", err)
	}

	var auth smtp.Auth
	if t.cfg.Username != "" {
		auth = smtp.PlainAuth("", t.cfg.Username, t.cfg.Password, t.cfg.Host)
	}

	recipients := make([]string, 0, len(msg.To)+len(msg.CC)+len(msg.BCC))
	recipients = append(recipients, msg.To...)
	recipients = append(recipients, msg.CC...)
	recipients = append(recipients, msg.BCC...)

	if t.cfg.TLS {
		return sendTLS(addr, auth, msg.From, recipients, body, t.cfg.Host)
	}
	return smtp.SendMail(addr, auth, msg.From, recipients, body)
}

func sendTLS(addr string, auth smtp.Auth, from string, to []string, body []byte, host string) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: host})
	if err != nil {
		return fmt.Errorf("smtp: tls dial: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("smtp: new client: %w", err)
	}
	defer client.Close()

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp: auth: %w", err)
		}
	}
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("smtp: mail from: %w", err)
	}
	for _, addr := range to {
		if err := client.Rcpt(addr); err != nil {
			return fmt.Errorf("smtp: rcpt %s: %w", addr, err)
		}
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp: data: %w", err)
	}
	if _, err := w.Write(body); err != nil {
		return fmt.Errorf("smtp: write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp: close data: %w", err)
	}
	return client.Quit()
}

func buildMIME(msg *core.MailMessage) ([]byte, error) {
	var buf bytes.Buffer

	// Headers.
	buf.WriteString("From: " + msg.From + "\r\n")
	if len(msg.To) > 0 {
		buf.WriteString("To: " + strings.Join(msg.To, ", ") + "\r\n")
	}
	if len(msg.CC) > 0 {
		buf.WriteString("Cc: " + strings.Join(msg.CC, ", ") + "\r\n")
	}
	if msg.ReplyTo != "" {
		buf.WriteString("Reply-To: " + msg.ReplyTo + "\r\n")
	}
	buf.WriteString("Subject: " + mime.QEncoding.Encode("utf-8", msg.Subject) + "\r\n")
	buf.WriteString("MIME-Version: 1.0\r\n")
	for k, v := range msg.Headers {
		buf.WriteString(textproto.CanonicalMIMEHeaderKey(k) + ": " + v + "\r\n")
	}

	hasHTML := msg.HTML != ""
	hasText := msg.Text != ""

	if hasHTML && hasText {
		// Multipart alternative.
		w := multipart.NewWriter(&buf)
		buf.WriteString("Content-Type: multipart/alternative; boundary=" + w.Boundary() + "\r\n\r\n")

		textPart, err := w.CreatePart(textproto.MIMEHeader{
			"Content-Type": {"text/plain; charset=utf-8"},
		})
		if err != nil {
			return nil, err
		}
		textPart.Write([]byte(msg.Text))

		htmlPart, err := w.CreatePart(textproto.MIMEHeader{
			"Content-Type": {"text/html; charset=utf-8"},
		})
		if err != nil {
			return nil, err
		}
		htmlPart.Write([]byte(msg.HTML))

		w.Close()
	} else if hasHTML {
		buf.WriteString("Content-Type: text/html; charset=utf-8\r\n\r\n")
		buf.WriteString(msg.HTML)
	} else {
		buf.WriteString("Content-Type: text/plain; charset=utf-8\r\n\r\n")
		buf.WriteString(msg.Text)
	}

	return buf.Bytes(), nil
}

// LogTransport logs the email to the framework logger instead of sending.
// Useful for development and testing.
type LogTransport struct {
	logger core.Logger
}

// NewLogTransport creates a new log transport.
func NewLogTransport(logger core.Logger) *LogTransport {
	return &LogTransport{logger: logger}
}

func (t *LogTransport) Name() string { return "log" }

func (t *LogTransport) Send(_ context.Context, msg *core.MailMessage) error {
	t.logger.Info("mailer: email delivered",
		"from", msg.From,
		"to", msg.To,
		"subject", msg.Subject,
		"html_length", len(msg.HTML),
		"text_length", len(msg.Text),
	)
	return nil
}
