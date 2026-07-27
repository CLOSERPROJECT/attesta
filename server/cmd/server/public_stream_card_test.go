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
