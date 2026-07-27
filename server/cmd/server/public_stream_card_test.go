package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestPublicStreamCardTemplateRendersCoreFields(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	card := PublicStreamCardView{
		Name:        "PV Module Tracing",
		Description: "End-to-end tracing of photovoltaic modules",
	}
	if err := tmpl.ExecuteTemplate(&out, "public_stream_card", card); err != nil {
		t.Fatalf("render public_stream_card template: %v", err)
	}
	body := out.String()

	for _, want := range []string{
		`class="public-stream-card"`,
		"PV Module Tracing",
		"End-to-end tracing of photovoltaic modules",
		"Stream",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in rendered public stream card, got: %s", want, body)
		}
	}
	if strings.Contains(body, "<a ") || strings.Contains(body, "<a>") {
		t.Fatalf("public stream card must not be a link, got: %s", body)
	}
	if strings.Contains(body, `href=`) {
		t.Fatalf("public stream card must not navigate, got: %s", body)
	}
}

func TestPublicStreamCardTemplateRendersStepPreview(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	card := PublicStreamCardView{
		Name: "PV Module Tracing",
		Steps: []PublicStreamCardStepView{
			{Title: "Incoming intake", SubstepCount: 3},
			{Title: "Quality check", SubstepCount: 1},
		},
	}
	if err := tmpl.ExecuteTemplate(&out, "public_stream_card", card); err != nil {
		t.Fatalf("render public_stream_card template: %v", err)
	}
	body := out.String()

	for _, want := range []string{
		`class="public-stream-card-steps-head"><strong>Steps</strong> <span>2</span>`,
		`class="public-stream-card-steps-list"`,
		"Incoming intake",
		"<span>3</span>",
		"Quality check",
		"<span>1</span>",
		`data-tooltip="Actions"`,
		`class="tip public-stream-card-step-icon"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in rendered public stream card, got: %s", want, body)
		}
	}
	if strings.Contains(body, " actions") {
		t.Fatalf("did not expect literal actions label, got: %s", body)
	}
}

func TestPublicStreamCardTemplateOmitsPassportBadgeWhenDisabled(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	card := PublicStreamCardView{
		Name:            "Indium recycling recovery",
		PassportEnabled: false,
		Steps: []PublicStreamCardStepView{
			{Title: "Incoming intake", SubstepCount: 2},
		},
	}
	if err := tmpl.ExecuteTemplate(&out, "public_stream_card", card); err != nil {
		t.Fatalf("render public_stream_card template: %v", err)
	}
	body := out.String()

	if !strings.Contains(body, `class="public-stream-card-badge">Stream</span>`) {
		t.Fatalf("expected static Stream badge, got: %s", body)
	}
	if strings.Contains(body, "Product Passport") {
		t.Fatalf("did not expect Product Passport badge when DPP disabled, got: %s", body)
	}
	if strings.Contains(body, `data-category="passport"`) {
		t.Fatalf("did not expect passport category hook when DPP disabled, got: %s", body)
	}
}

func TestPublicStreamCardTemplateRendersPassportBadgeWhenEnabled(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	card := PublicStreamCardView{
		Name:            "Battery Passport",
		PassportEnabled: true,
		Steps: []PublicStreamCardStepView{
			{Title: "Cell assembly", SubstepCount: 4},
		},
	}
	if err := tmpl.ExecuteTemplate(&out, "public_stream_card", card); err != nil {
		t.Fatalf("render public_stream_card template: %v", err)
	}
	body := out.String()

	if !strings.Contains(body, `class="public-stream-card-badge">Stream</span>`) {
		t.Fatalf("expected static Stream badge to remain, got: %s", body)
	}
	if !strings.Contains(body, `data-category="passport"`) {
		t.Fatalf("expected passport category hook, got: %s", body)
	}
	if !strings.Contains(body, "Product Passport") {
		t.Fatalf("expected Product Passport badge label, got: %s", body)
	}
}
