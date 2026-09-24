package main

import (
	"context"
	"strings"
	"sync"
)

// Mailer sends affiliation (and later other) notification messages.
// When notify is wired after a durable status transition, prefer log-and-continue
// on Send errors so a mail failure does not roll back an already-persisted decision.
type Mailer interface {
	Send(ctx context.Context, msg MailMessage) error
	// AbsoluteURL joins PUBLIC_BASE_URL with path (leading slash normalized).
	// When the base is empty, returns the path only (still slash-prefixed when path is non-empty).
	AbsoluteURL(path string) string
}

type MailMessage struct {
	Kind    string
	To      []string
	Subject string
	Body    string
}

func mailAbsoluteURL(base, path string) string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	path = strings.TrimSpace(path)
	if path == "" {
		return base
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if base == "" {
		return path
	}
	return base + path
}

type noopMailer struct {
	publicBaseURL string
}

func (m noopMailer) Send(context.Context, MailMessage) error { return nil }

func (m noopMailer) AbsoluteURL(path string) string {
	return mailAbsoluteURL(m.publicBaseURL, path)
}

type recordingMailer struct {
	mu            sync.Mutex
	messages      []MailMessage
	err           error
	publicBaseURL string
}

func (m *recordingMailer) Send(_ context.Context, msg MailMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cloned := msg
	if msg.To != nil {
		cloned.To = append([]string(nil), msg.To...)
	}
	m.messages = append(m.messages, cloned)
	return m.err
}

func (m *recordingMailer) AbsoluteURL(path string) string {
	return mailAbsoluteURL(m.publicBaseURL, path)
}

func (m *recordingMailer) Messages() []MailMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]MailMessage, len(m.messages))
	for i, msg := range m.messages {
		out[i] = msg
		if msg.To != nil {
			out[i].To = append([]string(nil), msg.To...)
		}
	}
	return out
}
