package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestStreamCardTemplateRendersCoreFields(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	card := StreamCardView{
		Key:         "demo",
		Name:        "Demo Stream",
		Description: "Track a pilot batch end to end",
		Counts: WorkflowProcessCounts{
			NotStarted: 2,
			Started:    1,
			Terminated: 3,
		},
		HasUserTurn: true,
		CanClone:    true,
		CanEdit:     true,
		CanDelete:   true,
		EditAction:  "/org-admin/formata-builder?stream=demo",
		DeleteAction: "/my/streams/demo/delete",
	}
	if err := tmpl.ExecuteTemplate(&out, "stream_card", card); err != nil {
		t.Fatalf("render stream_card template: %v", err)
	}
	body := out.String()

	for _, want := range []string{
		`class="stream-card-shell"`,
		`class="stream-card"`,
		`class="stream-card-title"`,
		`class="stream-card-body"`,
		`class="stream-card-footer"`,
		`class="stream-card-turn-indicator"`,
		`class="btn btn-ghost btn-icon stream-card-menu-trigger"`,
		`href="/my/streams/demo/"`,
		"Demo Stream",
		"Track a pilot batch end to end",
		"<td>2</td>",
		"<td>1</td>",
		"<td>3</td>",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in rendered stream card, got: %s", want, body)
		}
	}
}

func TestStreamCardCreateTemplateRendersCTA(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "stream_card_create", nil); err != nil {
		t.Fatalf("render stream_card_create template: %v", err)
	}
	body := out.String()

	for _, want := range []string{
		`class="stream-card stream-card-create"`,
		`class="stream-card-cta"`,
		`href="/org-admin/formata-builder"`,
		"Create new stream",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in rendered create card, got: %s", want, body)
		}
	}
}
