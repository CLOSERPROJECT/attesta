package main

import (
	"context"
	"sync"
)

// Mailer sends affiliation (and later other) notification messages.
// When notify is wired after a durable status transition, prefer log-and-continue
// on Send errors so a mail failure does not roll back an already-persisted decision.
type Mailer interface {
	Send(ctx context.Context, msg MailMessage) error
}

type MailMessage struct {
	Kind    string
	To      []string
	Subject string
	Body    string
}

type noopMailer struct{}

func (noopMailer) Send(context.Context, MailMessage) error { return nil }

type recordingMailer struct {
	mu       sync.Mutex
	messages []MailMessage
}

func (m *recordingMailer) Send(_ context.Context, msg MailMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cloned := msg
	if msg.To != nil {
		cloned.To = append([]string(nil), msg.To...)
	}
	m.messages = append(m.messages, cloned)
	return nil
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
