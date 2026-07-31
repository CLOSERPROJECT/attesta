package main

import (
	"bytes"
	"context"
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
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in rendered public stream card, got: %s", want, body)
		}
	}
	for _, mustNot := range []string{
		"public-stream-card-badge",
		">Stream<",
	} {
		if strings.Contains(body, mustNot) {
			t.Fatalf("public stream card must not contain %q, got: %s", mustNot, body)
		}
	}
}

func TestPublicStreamCardTemplateLinksWhenHrefSet(t *testing.T) {
	tmpl := parseTestTemplates(t)
	var out bytes.Buffer
	card := PublicStreamCardView{
		Name: "Linked Stream",
		Href: "/streams/linked",
	}
	if err := tmpl.ExecuteTemplate(&out, "public_stream_card", card); err != nil {
		t.Fatalf("render: %v", err)
	}
	body := out.String()
	if !strings.Contains(body, `href="/streams/linked"`) {
		t.Fatalf("expected href, got: %s", body)
	}
	if !strings.Contains(body, `class="public-stream-card"`) {
		t.Fatalf("expected public-stream-card class, got: %s", body)
	}
	if strings.Contains(body, "<article") {
		t.Fatalf("linked card must not use article shell, got: %s", body)
	}
}

func TestPublicStreamCardTemplateStaysArticleWithoutHref(t *testing.T) {
	tmpl := parseTestTemplates(t)
	var out bytes.Buffer
	card := PublicStreamCardView{Name: "Static"}
	if err := tmpl.ExecuteTemplate(&out, "public_stream_card", card); err != nil {
		t.Fatalf("render: %v", err)
	}
	body := out.String()
	if !strings.Contains(body, `<article class="public-stream-card"`) {
		t.Fatalf("expected article shell, got: %s", body)
	}
	if strings.Contains(body, `href=`) {
		t.Fatalf("empty Href must not link, got: %s", body)
	}
}

func TestPublicStreamCardTemplateRendersStepsAndRolesMetrics(t *testing.T) {
	tmpl := parseTestTemplates(t)
	var out bytes.Buffer
	card := PublicStreamCardView{Name: "PV", StepCount: 3, RoleCount: 4}
	if err := tmpl.ExecuteTemplate(&out, "public_stream_card", card); err != nil {
		t.Fatalf("render: %v", err)
	}
	body := out.String()
	for _, want := range []string{
		`class="public-stream-card-metrics-row"`,
		"3 steps",
		"4 roles",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q, got: %s", want, body)
		}
	}
	if strings.Contains(body, "public-stream-card-steps") {
		t.Fatalf("must not render steps list, got: %s", body)
	}
}

func TestPublicStreamCardTemplateOmitsPassportBadgeWhenDisabled(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	card := PublicStreamCardView{
		Name:            "Indium recycling recovery",
		PassportEnabled: false,
	}
	if err := tmpl.ExecuteTemplate(&out, "public_stream_card", card); err != nil {
		t.Fatalf("render public_stream_card template: %v", err)
	}
	body := out.String()

	if strings.Contains(body, "public-stream-card-dpp") {
		t.Fatalf("did not expect DPP chip when disabled, got: %s", body)
	}
	if strings.Contains(body, ">DPP<") {
		t.Fatalf("did not expect DPP label when disabled, got: %s", body)
	}
}

func TestPublicStreamCardTemplateRendersPassportBadgeWhenEnabled(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	card := PublicStreamCardView{
		Name:            "Battery Passport",
		PassportEnabled: true,
	}
	if err := tmpl.ExecuteTemplate(&out, "public_stream_card", card); err != nil {
		t.Fatalf("render public_stream_card template: %v", err)
	}
	body := out.String()

	if !strings.Contains(body, "public-stream-card-dpp") {
		t.Fatalf("expected DPP chip class, got: %s", body)
	}
	if !strings.Contains(body, "DPP") {
		t.Fatalf("expected DPP label, got: %s", body)
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
		OrganizationCount: 1,
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
		OrganizationCount: 1,
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
		OrganizationCount:     7,
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

func TestPublicStreamCardTemplateRendersEmptyMetricsAsSingleLayersChip(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	card := PublicStreamCardView{
		Name:          "Empty Stream",
		InstanceCount: 0,
	}
	if err := tmpl.ExecuteTemplate(&out, "public_stream_card", card); err != nil {
		t.Fatalf("render public_stream_card template: %v", err)
	}
	body := out.String()

	for _, want := range []string{
		`class="public-stream-card-metrics"`,
		`class="public-stream-card-metric"`,
		"no runs yet",
		// icon-layers-2 path
		`M13 13.74a2 2 0 0 1-2 0L2.5 8.87`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in empty metrics layout, got: %s", want, body)
		}
	}
	if strings.Contains(body, "all completed") {
		t.Fatalf("empty metrics must not show all-completed chip, got: %s", body)
	}
	if strings.Contains(body, "active now") {
		t.Fatalf("empty metrics must not show active chip, got: %s", body)
	}
	if strings.Contains(body, `d="m9 12 2 2 4-4"`) {
		t.Fatalf("empty metrics must not render check-circle icon, got: %s", body)
	}
}

func TestPublicStreamCardTemplateRendersOneInstanceAllCompletedMetrics(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	card := PublicStreamCardView{
		Name:          "Settled Stream",
		InstanceCount: 1,
		AllCompleted:  true,
	}
	if err := tmpl.ExecuteTemplate(&out, "public_stream_card", card); err != nil {
		t.Fatalf("render public_stream_card template: %v", err)
	}
	body := out.String()

	for _, want := range []string{
		`class="public-stream-card-metric"`,
		"1 run",
		"all completed",
		`M13 13.74a2 2 0 0 1-2 0L2.5 8.87`,
		`d="m9 12 2 2 4-4"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in all-completed metrics layout, got: %s", want, body)
		}
	}
	if strings.Contains(body, "1 runs") {
		t.Fatalf("singular count must not use plural label, got: %s", body)
	}
	if strings.Contains(body, "no runs yet") {
		t.Fatalf("settled metrics must not show empty label, got: %s", body)
	}
	if strings.Contains(body, "active now") {
		t.Fatalf("all-completed metrics must not show active chip, got: %s", body)
	}
}

func TestPublicStreamCardTemplateRendersPluralInstancesAllCompletedMetrics(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	card := PublicStreamCardView{
		Name:          "Settled Stream",
		InstanceCount: 2,
		AllCompleted:  true,
	}
	if err := tmpl.ExecuteTemplate(&out, "public_stream_card", card); err != nil {
		t.Fatalf("render public_stream_card template: %v", err)
	}
	body := out.String()

	for _, want := range []string{
		"2 runs",
		"all completed",
		`M13 13.74a2 2 0 0 1-2 0L2.5 8.87`,
		`d="m9 12 2 2 4-4"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in plural all-completed metrics, got: %s", want, body)
		}
	}
	if strings.Contains(body, "2 run<") || strings.Contains(body, "2 run\n") || strings.Contains(body, ">2 run<") {
		t.Fatalf("plural count must use runs label, got: %s", body)
	}
}

func TestPublicStreamCardTemplateRendersOneActiveNowMetrics(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	card := PublicStreamCardView{
		Name:          "Active Stream",
		InstanceCount: 1,
		ActiveCount:   1,
	}
	if err := tmpl.ExecuteTemplate(&out, "public_stream_card", card); err != nil {
		t.Fatalf("render public_stream_card template: %v", err)
	}
	body := out.String()

	for _, want := range []string{
		`class="public-stream-card-metric"`,
		`class="public-stream-card-metric public-stream-card-metric-active"`,
		"1 run",
		"1 active now",
		`M13 13.74a2 2 0 0 1-2 0L2.5 8.87`,
		`M22 12h-2.48a2 2 0 0 0-1.93 1.46l-2.35 8.36`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in active-now metrics layout, got: %s", want, body)
		}
	}
	if strings.Contains(body, "all completed") {
		t.Fatalf("active-now metrics must not show all-completed chip, got: %s", body)
	}
	if strings.Contains(body, "1 runs") {
		t.Fatalf("singular run count must not use plural label, got: %s", body)
	}
	if strings.Contains(body, "1 actives now") || strings.Contains(body, "1 active nows") {
		t.Fatalf("singular active count must use '1 active now', got: %s", body)
	}
}

func TestPublicStreamCardTemplateRendersPluralActiveNowMetrics(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	card := PublicStreamCardView{
		Name:          "Active Stream",
		InstanceCount: 3,
		ActiveCount:   2,
	}
	if err := tmpl.ExecuteTemplate(&out, "public_stream_card", card); err != nil {
		t.Fatalf("render public_stream_card template: %v", err)
	}
	body := out.String()

	for _, want := range []string{
		"3 runs",
		"2 active now",
		`M13 13.74a2 2 0 0 1-2 0L2.5 8.87`,
		`M22 12h-2.48a2 2 0 0 0-1.93 1.46l-2.35 8.36`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in plural active-now metrics, got: %s", want, body)
		}
	}
	if strings.Contains(body, "all completed") {
		t.Fatalf("active-now metrics must not show all-completed chip, got: %s", body)
	}
	if strings.Contains(body, "1 active now") {
		t.Fatalf("plural active count must not use singular label, got: %s", body)
	}
	if !strings.Contains(body, `class="public-stream-card-metric public-stream-card-metric-active"`) {
		t.Fatalf("expected active metric chip class, got: %s", body)
	}
}

func TestPublicStreamCardTemplateRendersOrganizationsFooterAndMetrics(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	card := PublicStreamCardView{
		Name:              "PV Module Tracing",
		InstanceCount:     28,
		AllCompleted:      true,
		OrganizationCount: 5,
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
		"28 runs",
		"all completed",
		`M13 13.74a2 2 0 0 1-2 0L2.5 8.87`,
		`d="m9 12 2 2 4-4"`,
		`<strong>Organizations</strong>`,
		`class="public-stream-card-orgs-count"`,
		">5<",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in rendered public stream card, got: %s", want, body)
		}
	}
	if strings.Contains(body, "Stream Instances") {
		t.Fatalf("footer must not say Stream Instances, got: %s", body)
	}
}

func TestBuildPublicStreamCardViewFillsStepRoleOrgCounts(t *testing.T) {
	s := &Server{store: NewMemoryStore()}
	cfg := RuntimeConfig{
		Workflow: WorkflowDef{
			Name:        "Counted Stream",
			Description: "counts",
			Steps: []WorkflowStep{
				{
					Title: "One",
					Substep: []WorkflowSub{
						{Roles: []string{"qa"}},
						{Roles: []string{"ops"}},
					},
				},
				{
					Title:   "Two",
					Substep: []WorkflowSub{{Roles: []string{"qa"}}},
				},
			},
		},
		Organizations: []WorkflowOrganization{
			{Slug: "a", Name: "Alpha"},
			{Slug: "b", Name: "Beta"},
			{Slug: "c", Name: "Gamma"},
			{Slug: "d", Name: "Delta"},
			{Slug: "e", Name: "Epsilon"},
		},
		DPP: DPPConfig{Enabled: true},
	}
	card, err := s.buildPublicStreamCardView(context.Background(), "counted", cfg, nil)
	if err != nil {
		t.Fatalf("buildPublicStreamCardView: %v", err)
	}
	if card.StepCount != 2 {
		t.Fatalf("StepCount = %d, want 2", card.StepCount)
	}
	if card.RoleCount != 2 {
		t.Fatalf("RoleCount = %d, want 2", card.RoleCount)
	}
	if card.OrganizationCount != 5 {
		t.Fatalf("OrganizationCount = %d, want 5", card.OrganizationCount)
	}
	if len(card.Organizations) != 4 || card.OrganizationsOverflow != 1 {
		t.Fatalf("avatars=%d overflow=%d, want 4/1", len(card.Organizations), card.OrganizationsOverflow)
	}
	if !card.PassportEnabled {
		t.Fatal("PassportEnabled want true")
	}
	if card.Href != "/streams/counted" {
		t.Fatalf("Href = %q, want /streams/counted", card.Href)
	}
}

func TestPublicStreamCardRoleCountDistinctAcrossSubsteps(t *testing.T) {
	def := WorkflowDef{
		Steps: []WorkflowStep{
			{
				Substep: []WorkflowSub{
					{Roles: []string{"qa", "ops"}},
					{Roles: []string{" qa "}}, // duplicate after trim
				},
			},
			{
				Substep: []WorkflowSub{
					{Role: "reviewer"},
					{Roles: []string{"ops", ""}},
				},
			},
		},
	}
	if got := publicStreamCardRoleCount(def); got != 3 {
		t.Fatalf("publicStreamCardRoleCount = %d, want 3 (qa, ops, reviewer)", got)
	}
	if got := publicStreamCardRoleCount(WorkflowDef{}); got != 0 {
		t.Fatalf("empty def role count = %d, want 0", got)
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
