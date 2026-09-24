package main

import (
	"bufio"
	"context"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

func TestNewMailerFromEnvNoopWhenHostEmpty(t *testing.T) {
	t.Setenv("SMTP_HOST", "")
	t.Setenv("SMTP_PORT", "1025")
	t.Setenv("PUBLIC_BASE_URL", "https://app.example")
	mailer := newMailerFromEnv()
	nm, ok := mailer.(noopMailer)
	if !ok {
		t.Fatalf("expected noopMailer when SMTP_HOST empty, got %T", mailer)
	}
	if got := nm.AbsoluteURL("/my/organization"); got != "https://app.example/my/organization" {
		t.Fatalf("AbsoluteURL = %q", got)
	}
}

func TestMailAbsoluteURL(t *testing.T) {
	cases := []struct {
		base, path, want string
	}{
		{"https://app.example", "/my", "https://app.example/my"},
		{"https://app.example/", "my", "https://app.example/my"},
		{"https://app.example/", "/my/organization", "https://app.example/my/organization"},
		{"", "/my/onboarding", "/my/onboarding"},
		{"", "my/onboarding", "/my/onboarding"},
		{"https://app.example", "", "https://app.example"},
	}
	for _, tc := range cases {
		if got := mailAbsoluteURL(tc.base, tc.path); got != tc.want {
			t.Fatalf("mailAbsoluteURL(%q, %q)=%q, want %q", tc.base, tc.path, got, tc.want)
		}
	}
}

func TestRecordingMailerAbsoluteURL(t *testing.T) {
	m := &recordingMailer{publicBaseURL: "http://localhost:3000"}
	if got := m.AbsoluteURL("my/organization/members"); got != "http://localhost:3000/my/organization/members" {
		t.Fatalf("AbsoluteURL = %q", got)
	}
}

func TestNewMailerFromEnvSMTPWhenHostSet(t *testing.T) {
	t.Setenv("SMTP_HOST", "mailpit")
	t.Setenv("SMTP_PORT", "1025")
	t.Setenv("SMTP_FROM", "attesta@localhost")
	t.Setenv("SMTP_USER", "")
	t.Setenv("SMTP_PASSWORD", "")
	t.Setenv("SMTP_SECURE", "")
	t.Setenv("PUBLIC_BASE_URL", "http://localhost:3000")
	mailer := newMailerFromEnv()
	sm, ok := mailer.(*smtpMailer)
	if !ok {
		t.Fatalf("expected *smtpMailer, got %T", mailer)
	}
	if sm.host != "mailpit" || sm.port != "1025" || sm.from != "attesta@localhost" {
		t.Fatalf("unexpected config: %+v", sm)
	}
	if sm.secure != smtpSecurePlain {
		t.Fatalf("expected smtpSecurePlain, got %v", sm.secure)
	}
	if sm.publicBaseURL != "http://localhost:3000" {
		t.Fatalf("publicBaseURL = %q", sm.publicBaseURL)
	}
	if got := sm.AbsoluteURL("/admin/organizations"); got != "http://localhost:3000/admin/organizations" {
		t.Fatalf("AbsoluteURL = %q", got)
	}
}

func TestParseSMTPSecure(t *testing.T) {
	cases := []struct {
		in   string
		want smtpSecureMode
	}{
		{"", smtpSecurePlain},
		{"plain", smtpSecurePlain},
		{"garbage", smtpSecurePlain},
		{"true", smtpSecureStartTLS},
		{"1", smtpSecureStartTLS},
		{"yes", smtpSecureStartTLS},
		{"on", smtpSecureStartTLS},
		{"starttls", smtpSecureStartTLS},
		{"STARTTLS", smtpSecureStartTLS},
		{"tls", smtpSecureImplicitTLS},
		{"ssl", smtpSecureImplicitTLS},
		{" TLS ", smtpSecureImplicitTLS},
	}
	for _, tc := range cases {
		if got := parseSMTPSecure(tc.in); got != tc.want {
			t.Fatalf("parseSMTPSecure(%q)=%v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestNewMailerFromEnvSMTPSecure(t *testing.T) {
	t.Setenv("SMTP_HOST", "smtp.example")
	t.Setenv("SMTP_SECURE", "tls")
	mailer := newMailerFromEnv()
	sm, ok := mailer.(*smtpMailer)
	if !ok {
		t.Fatalf("expected *smtpMailer, got %T", mailer)
	}
	if sm.secure != smtpSecureImplicitTLS {
		t.Fatalf("expected smtpSecureImplicitTLS, got %v", sm.secure)
	}
}

func TestNewMailerFromEnvDefaults(t *testing.T) {
	t.Setenv("SMTP_HOST", "smtp.example")
	t.Setenv("SMTP_PORT", "")
	t.Setenv("SMTP_FROM", "")
	mailer := newMailerFromEnv()
	sm, ok := mailer.(*smtpMailer)
	if !ok {
		t.Fatalf("expected *smtpMailer, got %T", mailer)
	}
	if sm.port != "1025" {
		t.Fatalf("default port: got %q", sm.port)
	}
	if sm.from != "attesta@localhost" {
		t.Fatalf("default from: got %q", sm.from)
	}
}

func TestSMTPMailerSendPlaintext(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	got := make(chan string, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
		data, err := runFakeSMTP(conn)
		if err != nil {
			got <- "ERR:" + err.Error()
			return
		}
		got <- data
	}()

	host, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	mailer := &smtpMailer{
		host:   host,
		port:   port,
		from:   "attesta@localhost",
		secure: smtpSecurePlain,
	}
	msg := MailMessage{
		Kind:    "affiliation_invite",
		To:      []string{"user@example.com"},
		Subject: "Join Acme on Attesta",
		Body:    "You were invited to join Acme.",
	}
	if err := mailer.Send(context.Background(), msg); err != nil {
		t.Fatalf("Send: %v", err)
	}

	select {
	case raw := <-got:
		if strings.HasPrefix(raw, "ERR:") {
			t.Fatal(raw)
		}
		for _, want := range []string{
			"Subject: Join Acme on Attesta",
			"You were invited to join Acme.",
			"To: user@example.com",
			"From: attesta@localhost",
		} {
			if !strings.Contains(raw, want) {
				t.Fatalf("message missing %q\n---\n%s", want, raw)
			}
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for fake SMTP DATA")
	}
}

func TestSMTPMailerSendNoRecipients(t *testing.T) {
	mailer := &smtpMailer{host: "127.0.0.1", port: "9", from: "a@b.c"}
	err := mailer.Send(context.Background(), MailMessage{Subject: "x", Body: "y"})
	if err == nil {
		t.Fatal("expected error for empty To")
	}
}

// runFakeSMTP speaks a minimal SMTP dialogue and returns the DATA payload.
func runFakeSMTP(conn net.Conn) (string, error) {
	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
	writeLine := func(s string) error {
		if _, err := rw.WriteString(s + "\r\n"); err != nil {
			return err
		}
		return rw.Flush()
	}
	if err := writeLine("220 localhost ESMTP fake"); err != nil {
		return "", err
	}
	var data string
	for {
		line, err := rw.ReadString('\n')
		if err != nil {
			if data != "" && err == io.EOF {
				return data, nil
			}
			return "", err
		}
		cmd := strings.ToUpper(strings.TrimSpace(line))
		switch {
		case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
			_ = writeLine("250-localhost")
			_ = writeLine("250 OK")
		case strings.HasPrefix(cmd, "MAIL FROM:"):
			_ = writeLine("250 OK")
		case strings.HasPrefix(cmd, "RCPT TO:"):
			_ = writeLine("250 OK")
		case strings.HasPrefix(cmd, "DATA"):
			if err := writeLine("354 End data with <CR><LF>.<CR><LF>"); err != nil {
				return "", err
			}
			var b strings.Builder
			for {
				l, err := rw.ReadString('\n')
				if err != nil {
					return "", err
				}
				if strings.TrimRight(l, "\r\n") == "." {
					break
				}
				b.WriteString(l)
			}
			data = b.String()
			if err := writeLine("250 OK"); err != nil {
				return "", err
			}
		case strings.HasPrefix(cmd, "QUIT"):
			_ = writeLine("221 Bye")
			return data, nil
		default:
			_ = writeLine("250 OK")
		}
	}
}
