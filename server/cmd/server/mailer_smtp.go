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

// smtpSecureMode selects how the SMTP connection is secured.
type smtpSecureMode int

const (
	smtpSecurePlain smtpSecureMode = iota
	smtpSecureStartTLS
	smtpSecureImplicitTLS
)

// parseSMTPSecure maps SMTP_SECURE env values to a typed mode.
// Empty/unknown → plain; "true"|"1"|"yes"|"on"|"starttls" → STARTTLS;
// "tls"|"ssl" → implicit TLS (typically port 465).
func parseSMTPSecure(v string) smtpSecureMode {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "tls", "ssl":
		return smtpSecureImplicitTLS
	case "true", "1", "yes", "on", "starttls":
		return smtpSecureStartTLS
	default:
		return smtpSecurePlain
	}
}

// smtpMailer sends plaintext MailMessage over SMTP (stdlib net/smtp).
type smtpMailer struct {
	host          string
	port          string
	username      string
	password      string
	from          string
	secure        smtpSecureMode
	publicBaseURL string
}

// newMailerFromEnv returns an SMTP mailer when SMTP_HOST is set; otherwise noopMailer.
//
// Env:
//
//	SMTP_HOST        — required to enable (empty → noop)
//	SMTP_PORT        — default 1025 (Mailpit)
//	SMTP_USER        — optional (empty OK for Mailpit)
//	SMTP_PASSWORD    — optional
//	SMTP_FROM        — default attesta@localhost
//	SMTP_SECURE      — empty/plain for Mailpit; "true"|"starttls" for STARTTLS; "tls" for implicit TLS (465)
//	PUBLIC_BASE_URL  — origin for AbsoluteURL (e.g. http://localhost:3000); empty → path-only links
func newMailerFromEnv() Mailer {
	publicBaseURL := strings.TrimSpace(os.Getenv("PUBLIC_BASE_URL"))
	host := strings.TrimSpace(os.Getenv("SMTP_HOST"))
	if host == "" {
		return noopMailer{publicBaseURL: publicBaseURL}
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
		host:          host,
		port:          port,
		username:      strings.TrimSpace(os.Getenv("SMTP_USER")),
		password:      os.Getenv("SMTP_PASSWORD"),
		from:          from,
		secure:        parseSMTPSecure(os.Getenv("SMTP_SECURE")),
		publicBaseURL: publicBaseURL,
	}
}

func (m *smtpMailer) AbsoluteURL(path string) string {
	return mailAbsoluteURL(m.publicBaseURL, path)
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
	case smtpSecureImplicitTLS:
		return sendSMTPImplicitTLS(ctx, addr, m.host, auth, m.from, msg.To, raw)
	case smtpSecureStartTLS:
		return sendSMTPStartTLS(ctx, addr, m.host, auth, m.from, msg.To, raw)
	default:
		// Plain SMTP (Mailpit local). Prefer dial+send so ctx cancel is honored.
		return sendSMTPPlain(ctx, addr, m.host, auth, m.from, msg.To, raw)
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

func sendSMTPPlain(ctx context.Context, addr, serverName string, auth smtp.Auth, from string, to []string, raw []byte) error {
	var d net.Dialer
	return sendSMTPDialed(ctx, addr, serverName, d.DialContext, nil, auth, from, to, raw)
}

func sendSMTPStartTLS(ctx context.Context, addr, serverName string, auth smtp.Auth, from string, to []string, raw []byte) error {
	var d net.Dialer
	upgrade := func(c *smtp.Client) error {
		if ok, _ := c.Extension("STARTTLS"); ok {
			cfg := &tls.Config{ServerName: serverName, MinVersion: tls.VersionTLS12}
			return c.StartTLS(cfg)
		}
		return nil
	}
	return sendSMTPDialed(ctx, addr, serverName, d.DialContext, upgrade, auth, from, to, raw)
}

func sendSMTPImplicitTLS(ctx context.Context, addr, serverName string, auth smtp.Auth, from string, to []string, raw []byte) error {
	var d tls.Dialer
	d.Config = &tls.Config{ServerName: serverName, MinVersion: tls.VersionTLS12}
	return sendSMTPDialed(ctx, addr, serverName, d.DialContext, nil, auth, from, to, raw)
}

// sendSMTPDialed dials, builds an SMTP client, optionally upgrades TLS, then sends.
// dial must honor ctx cancel (e.g. Dialer.DialContext / tls.Dialer.DialContext).
func sendSMTPDialed(
	ctx context.Context,
	addr, serverName string,
	dial func(ctx context.Context, network, addr string) (net.Conn, error),
	upgrade func(*smtp.Client) error,
	auth smtp.Auth,
	from string,
	to []string,
	raw []byte,
) error {
	conn, err := dial(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	c, err := smtp.NewClient(conn, serverName)
	if err != nil {
		_ = conn.Close()
		return err
	}
	defer func() { _ = c.Close() }()
	if upgrade != nil {
		if err := upgrade(c); err != nil {
			return err
		}
	}
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
