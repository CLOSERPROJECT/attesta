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

func TestPublicStreamCardTemplateRendersOrgLogoAvatarWithAccessibleName(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	card := PublicStreamCardView{
		Name: "PV Module Tracing",
		Organizations: []PublicStreamCardOrgView{
			{Name: "Acme Corp", LogoURL: "/organization/logo/acme"},
		},
	}
	if err := tmpl.ExecuteTemplate(&out, "public_stream_card", card); err != nil {
		t.Fatalf("render public_stream_card template: %v", err)
	}
	body := out.String()

	for _, want := range []string{
		`class="public-stream-card-avatars"`,
		`src="/organization/logo/acme"`,
		`alt="Acme Corp"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in rendered public stream card, got: %s", want, body)
		}
	}
}

func TestPublicStreamCardTemplateFallsBackToOrgInitialsWhenNoLogo(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	card := PublicStreamCardView{
		Name: "PV Module Tracing",
		Organizations: []PublicStreamCardOrgView{
			{Name: "Acme Corp", Initials: "AC"},
		},
	}
	if err := tmpl.ExecuteTemplate(&out, "public_stream_card", card); err != nil {
		t.Fatalf("render public_stream_card template: %v", err)
	}
	body := out.String()

	if strings.Contains(body, "<img ") {
		t.Fatalf("did not expect logo img when LogoURL empty, got: %s", body)
	}
	for _, want := range []string{
		`class="public-stream-card-avatar"`,
		`aria-label="Acme Corp"`,
		">AC<",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in rendered public stream card, got: %s", want, body)
		}
	}
}

func TestPublicStreamCardTemplateRendersOrgOverflowCount(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	card := PublicStreamCardView{
		Name: "PV Module Tracing",
		Organizations: []PublicStreamCardOrgView{
			{Name: "One", Initials: "ON"},
			{Name: "Two", Initials: "TW"},
			{Name: "Three", Initials: "TH"},
			{Name: "Four", Initials: "FO"},
		},
		OrganizationsOverflow: 3,
	}
	if err := tmpl.ExecuteTemplate(&out, "public_stream_card", card); err != nil {
		t.Fatalf("render public_stream_card template: %v", err)
	}
	body := out.String()

	for _, want := range []string{
		`class="public-stream-card-avatar-more"`,
		`aria-label="3 more organizations"`,
		">+3<",
		">ON<",
		">FO<",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in rendered public stream card, got: %s", want, body)
		}
	}
}

func TestPublicStreamCardTemplateRendersOrganizationsFooterAndMetrics(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	card := PublicStreamCardView{
		Name:          "PV Module Tracing",
		InstanceCount: 28,
		ActivityLabel: "all completed",
		Organizations: []PublicStreamCardOrgView{
			{Name: "Acme Corp", Initials: "AC"},
		},
	}
	if err := tmpl.ExecuteTemplate(&out, "public_stream_card", card); err != nil {
		t.Fatalf("render public_stream_card template: %v", err)
	}
	body := out.String()

	for _, want := range []string{
		`class="public-stream-card-metrics"`,
		">28<",
		"all completed",
		`<strong>Organizations</strong>`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in rendered public stream card, got: %s", want, body)
		}
	}
	if strings.Contains(body, "Stream Instances") {
		t.Fatalf("footer must not say Stream Instances, got: %s", body)
	}
}

func TestOrganizationInitialsAreUppercase(t *testing.T) {
	if got := organizationInitials("Acme Corp"); got != "AC" {
		t.Fatalf("organizationInitials(Acme Corp) = %q, want AC", got)
	}
	if got := organizationInitials("b"); got != "B" {
		t.Fatalf("organizationInitials(b) = %q, want B", got)
	}
}

func TestPublicStreamCardOrganizationsLimitsToFourWithOverflow(t *testing.T) {
	orgs := []WorkflowOrganization{
		{Slug: "a", Name: "Alpha"},
		{Slug: "b", Name: "Beta"},
		{Slug: "c", Name: "Gamma"},
		{Slug: "d", Name: "Delta"},
		{Slug: "e", Name: "Epsilon"},
		{Slug: "f", Name: "Zeta"},
	}
	got, overflow := publicStreamCardOrganizations(orgs, nil)
	if len(got) != 4 {
		t.Fatalf("len(orgs) = %d, want 4", len(got))
	}
	if overflow != 2 {
		t.Fatalf("overflow = %d, want 2", overflow)
	}
	if got[0].Initials != "AL" || got[3].Initials != "DE" {
		t.Fatalf("initials = %#v", got)
	}
}
