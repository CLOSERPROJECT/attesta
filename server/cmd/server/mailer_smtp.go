package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"os"
	"strings"
)

// smtpMailer sends plaintext MailMessage over SMTP (stdlib net/smtp).
type smtpMailer struct {
	host     string
	port     string
	username string
	password string
	from     string
	secure   string // "", "true"|"starttls", or "tls"
}

// newMailerFromEnv returns an SMTP mailer when SMTP_HOST is set; otherwise noopMailer.
//
// Env:
//
//	SMTP_HOST     — required to enable (empty → noop)
//	SMTP_PORT     — default 1025 (Mailpit)
//	SMTP_USER     — optional (empty OK for Mailpit)
//	SMTP_PASSWORD — optional
//	SMTP_FROM     — default attesta@localhost
//	SMTP_SECURE   — empty/plain for Mailpit; "true"|"starttls" for STARTTLS; "tls" for implicit TLS (465)
func newMailerFromEnv() Mailer {
	host := strings.TrimSpace(os.Getenv("SMTP_HOST"))
	if host == "" {
		return noopMailer{}
	}
	port := strings.TrimSpace(os.Getenv("SMTP_PORT"))
	if port == "" {
		port = "1025"
	}
	from := strings.TrimSpace(os.Getenv("SMTP_FROM"))
	if from == "" {
		from = "attesta@localhost"
	}
	return &smtpMailer{
		host:     host,
		port:     port,
		username: strings.TrimSpace(os.Getenv("SMTP_USER")),
		password: os.Getenv("SMTP_PASSWORD"),
		from:     from,
		secure:   strings.ToLower(strings.TrimSpace(os.Getenv("SMTP_SECURE"))),
	}
}

func (m *smtpMailer) Send(ctx context.Context, msg MailMessage) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(msg.To) == 0 {
		return fmt.Errorf("smtp mailer: no recipients")
	}
	addr := net.JoinHostPort(m.host, m.port)
	raw := buildSMTPMessage(m.from, msg)

	var auth smtp.Auth
	if m.username != "" {
		auth = smtp.PlainAuth("", m.username, m.password, m.host)
	}

	switch m.secure {
	case "tls", "ssl":
		return sendSMTPImplicitTLS(ctx, addr, m.host, auth, m.from, msg.To, raw)
	case "true", "1", "yes", "on", "starttls":
		return sendSMTPStartTLS(ctx, addr, m.host, auth, m.from, msg.To, raw)
	default:
		// Plain SMTP (Mailpit local). Prefer dial+send so ctx cancel is honored.
		return sendSMTPPlain(ctx, addr, auth, m.from, msg.To, raw)
	}
}

func buildSMTPMessage(from string, msg MailMessage) []byte {
	var b strings.Builder
	b.WriteString("From: ")
	b.WriteString(from)
	b.WriteString("\r\n")
	b.WriteString("To: ")
	b.WriteString(strings.Join(msg.To, ", "))
	b.WriteString("\r\n")
	b.WriteString("Subject: ")
	b.WriteString(msg.Subject)
	b.WriteString("\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("\r\n")
	b.WriteString(msg.Body)
	return []byte(b.String())
}

func sendSMTPPlain(ctx context.Context, addr string, auth smtp.Auth, from string, to []string, raw []byte) error {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	host, _, _ := net.SplitHostPort(addr)
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		_ = conn.Close()
		return err
	}
	defer func() { _ = c.Close() }()
	return smtpClientSend(c, auth, from, to, raw)
}

func sendSMTPStartTLS(ctx context.Context, addr, serverName string, auth smtp.Auth, from string, to []string, raw []byte) error {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	c, err := smtp.NewClient(conn, serverName)
	if err != nil {
		_ = conn.Close()
		return err
	}
	defer func() { _ = c.Close() }()
	if ok, _ := c.Extension("STARTTLS"); ok {
		cfg := &tls.Config{ServerName: serverName, MinVersion: tls.VersionTLS12}
		if err := c.StartTLS(cfg); err != nil {
			return err
		}
	}
	return smtpClientSend(c, auth, from, to, raw)
}

func sendSMTPImplicitTLS(ctx context.Context, addr, serverName string, auth smtp.Auth, from string, to []string, raw []byte) error {
	var d tls.Dialer
	d.Config = &tls.Config{ServerName: serverName, MinVersion: tls.VersionTLS12}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	c, err := smtp.NewClient(conn, serverName)
	if err != nil {
		_ = conn.Close()
		return err
	}
	defer func() { _ = c.Close() }()
	return smtpClientSend(c, auth, from, to, raw)
}

func smtpClientSend(c *smtp.Client, auth smtp.Auth, from string, to []string, raw []byte) error {
	if auth != nil {
		if ok, _ := c.Extension("AUTH"); ok {
			if err := c.Auth(auth); err != nil {
				return err
			}
		}
	}
	if err := c.Mail(from); err != nil {
		return err
	}
	for _, rcpt := range to {
		if err := c.Rcpt(rcpt); err != nil {
			return err
		}
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(raw); err != nil {
		_ = w.Close()
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}
