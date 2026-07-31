package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestBuildPublicStreamOrganizationsOrdersByFirstStepAppearance(t *testing.T) {
	cfg := RuntimeConfig{
		Workflow: WorkflowDef{
			Steps: []WorkflowStep{
				{StepID: "2", Order: 2, OrganizationSlug: "org-b"},
				{StepID: "1", Order: 1, OrganizationSlug: "org-a"},
				{StepID: "3", Order: 3, OrganizationSlug: "org-a"},
				{StepID: "4", Order: 4, OrganizationSlug: "org-c"},
			},
		},
		Organizations: []WorkflowOrganization{
			{Slug: "org-c", Name: "Charlie Org"},
			{Slug: "org-a", Name: "Alpha Org"},
			{Slug: "org-b", Name: "Bravo Org"},
		},
	}
	got := buildPublicStreamOrganizations(cfg, map[string]string{
		"org-b": "/organization/logo/org-b",
	})
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3 unique orgs", len(got))
	}
	want := []PublicStreamCardOrgView{
		{Name: "Alpha Org", Initials: "AL"},
		{Name: "Bravo Org", LogoURL: "/organization/logo/org-b", Initials: "BR"},
		{Name: "Charlie Org", Initials: "CH"},
	}
	for i := range want {
		if got[i].Name != want[i].Name || got[i].LogoURL != want[i].LogoURL || got[i].Initials != want[i].Initials {
			t.Fatalf("org[%d] = %#v, want %#v", i, got[i], want[i])
		}
	}
}

func TestPublicStreamBodyTemplateRendersOrganizationsAboveRuns(t *testing.T) {
	tmpl := parseTestTemplates(t)
	var out bytes.Buffer
	view := PublicStreamPageView{
		Header: PublicStreamCardView{Name: "Pilot", PassportEnabled: true},
		Organizations: []PublicStreamCardOrgView{
			{Name: "Alpha Org", LogoURL: "/organization/logo/alpha", Initials: "AO"},
			{Name: "Bravo Org", Initials: "BO"},
		},
		RecentRuns: []PublicStreamRunView{{
			CompletedAt: "1 Jul 2026 at 12:00 UTC",
			DigitalLink: "/01/x/10/y/21/z",
		}},
		Blueprint: StreamInstanceDetailView{HideStatus: true},
	}
	if err := tmpl.ExecuteTemplate(&out, "public_stream_body", view); err != nil {
		t.Fatalf("render: %v", err)
	}
	body := out.String()
	orgIdx := strings.Index(body, `id="public-stream-orgs-heading"`)
	runsIdx := strings.Index(body, `id="public-stream-runs-heading"`)
	if orgIdx < 0 || runsIdx < 0 {
		t.Fatalf("expected orgs and runs headings, got: %s", body)
	}
	if orgIdx > runsIdx {
		t.Fatalf("organizations section must appear above runs")
	}
	for _, want := range []string{
		`class="public-stream-aside"`,
		`class="public-stream-orgs"`,
		`class="public-stream-orgs-list"`,
		`class="public-stream-org"`,
		`src="/organization/logo/alpha"`,
		"Alpha Org",
		"Bravo Org",
		`class="public-stream-org-mark"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in body, got: %s", want, body)
		}
	}
}

func TestPublicStreamBodyTemplateHidesOrganizationsWhenEmpty(t *testing.T) {
	tmpl := parseTestTemplates(t)
	var out bytes.Buffer
	view := PublicStreamPageView{
		Header:    PublicStreamCardView{Name: "No Orgs", PassportEnabled: true},
		Blueprint: StreamInstanceDetailView{HideStatus: true},
	}
	if err := tmpl.ExecuteTemplate(&out, "public_stream_body", view); err != nil {
		t.Fatalf("render: %v", err)
	}
	body := out.String()
	if strings.Contains(body, `class="public-stream-orgs"`) || strings.Contains(body, "Organizations") {
		t.Fatalf("orgs section must be hidden when empty, got: %s", body)
	}
}

func TestBuildPublicStreamRunsOnlyCompletedWithDPP(t *testing.T) {
	doneAt := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	def := WorkflowDef{Steps: []WorkflowStep{{Substep: []WorkflowSub{{SubstepID: "1.1"}}}}}

	processes := []Process{
		{ID: primitive.NewObjectID(), Status: processStatusActive, Progress: map[string]ProcessStep{"1_1": {State: "pending"}}},
		{ID: primitive.NewObjectID(), Status: processStatusTerminated, Termination: &ProcessTermination{EndedAt: doneAt}},
		{
			ID:     primitive.NewObjectID(),
			Status: processStatusDone,
			Progress: map[string]ProcessStep{
				"1_1": {State: "done", DoneAt: &doneAt},
			},
			DPP: &ProcessDPP{GTIN: "09506000134352", Lot: "LOT-1", Serial: "SER-1", GeneratedAt: doneAt},
		},
		{
			ID:     primitive.NewObjectID(),
			Status: processStatusDone,
			Progress: map[string]ProcessStep{
				"1_1": {State: "done", DoneAt: ptrTime(doneAt.Add(-time.Hour))},
			},
		},
	}
	// Newest-first input: builder must preserve ListRecent order among DPP completed only.
	ordered := []Process{processes[2], processes[3], processes[0], processes[1]}
	runs := buildPublicStreamRuns(def, ordered)
	if len(runs) != 1 {
		t.Fatalf("len = %d, want 1 DPP completed", len(runs))
	}
	if runs[0].DigitalLink != digitalLinkURL("09506000134352", "LOT-1", "SER-1") {
		t.Fatalf("DigitalLink = %q", runs[0].DigitalLink)
	}
	if runs[0].CompletedAt == "" {
		t.Fatal("expected CompletedAt")
	}
}

func TestBuildPublicStreamRunsRespectsCap(t *testing.T) {
	def := WorkflowDef{}
	doneAt := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	var processes []Process
	for i := 0; i < publicStreamRecentRunLimit+3; i++ {
		at := doneAt.Add(-time.Duration(i) * time.Hour)
		processes = append(processes, Process{
			ID:       primitive.NewObjectID(),
			Status:   processStatusDone,
			Progress: map[string]ProcessStep{"1_1": {State: "done", DoneAt: &at}},
			DPP: &ProcessDPP{
				GTIN: "09506000134352", Lot: "LOT-1",
				Serial: fmt.Sprintf("SER-%d", i), GeneratedAt: at,
			},
		})
	}
	runs := buildPublicStreamRuns(def, processes)
	if len(runs) != publicStreamRecentRunLimit {
		t.Fatalf("len = %d, want %d", len(runs), publicStreamRecentRunLimit)
	}
}

func TestBuildPublicStreamBlueprintIsReadOnly(t *testing.T) {
	server := &Server{tmpl: parseTestTemplates(t), store: NewMemoryStore()}
	cfg, err := parseRuntimeConfigData("wf.yaml", []byte(minimalCategorizedWorkflowYAML("")))
	if err != nil {
		t.Fatal(err)
	}
	view := server.buildPublicStreamBlueprint(t.Context(), cfg, "wf")
	if !view.HideStatus {
		t.Fatal("expected HideStatus")
	}
	if view.CanTerminate || view.TerminateAction != "" {
		t.Fatalf("must not expose terminate, got %#v", view)
	}
	if len(view.Timeline) == 0 {
		t.Fatal("expected timeline steps")
	}
	for _, step := range view.Timeline {
		if !step.Expanded {
			t.Fatalf("public blueprint steps should be expanded, got step %#v", step.Summary)
		}
		for _, sub := range step.Substeps {
			if sub.Body == nil || !sub.Body.ReadOnly {
				t.Fatalf("substep body must be read-only, got %#v", sub.Body)
			}
		}
	}
}

func TestPublicStreamBodyTemplateRendersSections(t *testing.T) {
	tmpl := parseTestTemplates(t)
	var out bytes.Buffer
	view := PublicStreamPageView{
		PageBase: PageBase{Body: "public_stream_body"},
		HomeHref: "/?category=supply-chain&subCategory=procurement",
		Header: PublicStreamCardView{
			Name:            "Pilot Workflow",
			Description:     "Gallium batches",
			PassportEnabled: true,
			StepCount:       3,
			RoleCount:       5,
		},
		RecentRuns: []PublicStreamRunView{{
			CompletedAt: "1 Jul 2026 at 12:00 UTC",
			DigitalLink: "/01/09506000134352/10/LOT-1/21/SER-1",
		}},
		Blueprint: StreamInstanceDetailView{
			HideStatus: true,
			Timeline: []TimelineStep{{
				Summary: StepSummaryView{StepID: "1", Title: "Step 1"},
				Substeps: []TimelineSubstep{{
					SubstepID: "1.1", Title: "Input",
					Body: &SubstepBodyView{Mode: "preview", Title: "Input", ReadOnly: true, Reason: "Public preview."},
				}},
			}},
		},
	}
	if err := tmpl.ExecuteTemplate(&out, "public_stream_body", view); err != nil {
		t.Fatalf("render: %v", err)
	}
	body := out.String()
	for _, want := range []string{
		`class="public-stream"`,
		`class="public-stream-back"`,
		`href="/?category=supply-chain&amp;subCategory=procurement"`,
		"Pilot Workflow",
		"Gallium batches",
		"Recent completed runs",
		"Workflow",
		`class="public-stream-runs-list"`,
		`href="/01/09506000134352/10/LOT-1/21/SER-1"`,
		"1 Jul 2026 at 12:00 UTC",
		`class="stream-timeline-list"`,
		`class="public-stream-metrics"`,
		"3 steps",
		"5 roles",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in body, got: %s", want, body)
		}
	}
	for _, mustNot := range []string{
		`action=`,
		"/instance/start",
		"/terminate",
		"Sign in to run",
		"Open in workspace",
		`public-stream-run-card`,
	} {
		if strings.Contains(body, mustNot) {
			t.Fatalf("must not contain %q, got: %s", mustNot, body)
		}
	}
}

func TestPublicStreamBodyTemplateHidesRunsWithoutDPP(t *testing.T) {
	tmpl := parseTestTemplates(t)
	var out bytes.Buffer
	view := PublicStreamPageView{
		Header: PublicStreamCardView{Name: "No DPP", PassportEnabled: false},
		RecentRuns: []PublicStreamRunView{{
			CompletedAt: "1 Jul 2026 at 12:00 UTC",
			DigitalLink: "/01/x/10/y/21/z",
		}},
		Blueprint: StreamInstanceDetailView{HideStatus: true},
	}
	if err := tmpl.ExecuteTemplate(&out, "public_stream_body", view); err != nil {
		t.Fatalf("render: %v", err)
	}
	body := out.String()
	if strings.Contains(body, "Recent completed runs") || strings.Contains(body, `class="public-stream-runs"`) {
		t.Fatalf("runs section must be hidden without DPP, got: %s", body)
	}
}

func TestPublicStreamBodyTemplateEmptyRuns(t *testing.T) {
	tmpl := parseTestTemplates(t)
	var out bytes.Buffer
	view := PublicStreamPageView{
		Header:    PublicStreamCardView{Name: "Empty Runs", PassportEnabled: true},
		Blueprint: StreamInstanceDetailView{HideStatus: true},
	}
	if err := tmpl.ExecuteTemplate(&out, "public_stream_body", view); err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(out.String(), "No completed runs yet") {
		t.Fatalf("expected empty copy, got: %s", out.String())
	}
	if !strings.Contains(out.String(), `class="public-stream-runs-empty"`) {
		t.Fatalf("expected empty state, got: %s", out.String())
	}
}

func TestHandlePublicStreamOK(t *testing.T) {
	tempDir := t.TempDir()
	yaml := strings.Replace(
		minimalCategorizedWorkflowYAML("  categorySlug: supply-chain\n  subCategorySlug: procurement\n"),
		`name: "Workflow"`,
		`name: "Pilot Workflow"`,
		1,
	)
	yaml += "dpp:\n  enabled: true\n  gtin: \"09506000134352\"\n"
	if err := os.WriteFile(filepath.Join(tempDir, "pilot.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	store := NewMemoryStore()
	seedPlatformAdminTaxonomy(t, store)
	doneAt := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	store.SeedProcess(Process{
		ID:          primitive.NewObjectID(),
		WorkflowKey: "pilot",
		Status:      processStatusDone,
		CreatedAt:   doneAt,
		Progress:    map[string]ProcessStep{"1_1": {State: "done", DoneAt: &doneAt}},
		DPP:         &ProcessDPP{GTIN: "09506000134352", Lot: "LOT-1", Serial: "SER-1", GeneratedAt: doneAt},
	})
	store.SeedProcess(Process{
		ID:          primitive.NewObjectID(),
		WorkflowKey: "pilot",
		Status:      processStatusActive,
		CreatedAt:   doneAt,
		Progress:    map[string]ProcessStep{"1_1": {State: "pending"}},
	})
	server := &Server{store: store, configDir: tempDir, tmpl: parseTestTemplates(t)}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/streams/pilot", nil)
	server.handlePublicStream(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{
		"Pilot Workflow",
		"Recent completed runs",
		`href="/01/09506000134352/10/LOT-1/21/SER-1"`,
		"Workflow",
		`href="/?category=supply-chain&amp;subCategory=procurement"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q, got: %s", want, body)
		}
	}
	if strings.Contains(body, "/instance/start") || strings.Contains(body, "Sign in to run") {
		t.Fatalf("unexpected CTA/actions in body")
	}
}

func TestHandlePublicStreamHomeHrefUncategorized(t *testing.T) {
	tempDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(tempDir, "pilot.yaml"), []byte(minimalCategorizedWorkflowYAML("")), 0o644)
	server := &Server{store: NewMemoryStore(), configDir: tempDir, tmpl: parseTestTemplates(t)}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/streams/pilot", nil)
	server.handlePublicStream(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `class="public-stream-back" href="/"`) {
		t.Fatalf("expected Home href=/, got: %s", body)
	}
	if strings.Contains(body, "category=") {
		t.Fatalf("uncategorized stream must not filter Home href, got: %s", body)
	}
}

func TestHandlePublicStreamUnknownKey404(t *testing.T) {
	server := &Server{store: NewMemoryStore(), configDir: t.TempDir(), tmpl: parseTestTemplates(t)}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/streams/missing", nil)
	server.handlePublicStream(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestHandlePublicStreamRejectsNestedPath(t *testing.T) {
	tempDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(tempDir, "pilot.yaml"), []byte(minimalCategorizedWorkflowYAML("")), 0o644)
	server := &Server{store: NewMemoryStore(), configDir: tempDir, tmpl: parseTestTemplates(t)}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/streams/pilot/instance/abc", nil)
	server.handlePublicStream(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestHandlePublicStreamTrailingSlashOK(t *testing.T) {
	tempDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(tempDir, "pilot.yaml"), []byte(minimalCategorizedWorkflowYAML("")), 0o644)
	server := &Server{store: NewMemoryStore(), configDir: tempDir, tmpl: parseTestTemplates(t)}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/streams/pilot/", nil)
	server.handlePublicStream(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestNewMuxPublicStreamAndPublicPartialCoexist(t *testing.T) {
	tempDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(tempDir, "pilot.yaml"), []byte(minimalCategorizedWorkflowYAML(
		"  categorySlug: supply-chain\n  subCategorySlug: procurement\n",
	)), 0o644)
	store := NewMemoryStore()
	seedPlatformAdminTaxonomy(t, store)
	server := &Server{store: store, configDir: tempDir, tmpl: parseTestTemplates(t)}
	mux := server.newMux()

	recPage := httptest.NewRecorder()
	mux.ServeHTTP(recPage, httptest.NewRequest(http.MethodGet, "/streams/pilot", nil))
	if recPage.Code != http.StatusOK {
		t.Fatalf("page status = %d", recPage.Code)
	}

	recPartial := httptest.NewRecorder()
	mux.ServeHTTP(recPartial, httptest.NewRequest(http.MethodGet, "/streams/public?category=supply-chain&subCategory=procurement", nil))
	if recPartial.Code != http.StatusOK {
		t.Fatalf("partial status = %d", recPartial.Code)
	}
	if !strings.Contains(recPartial.Body.String(), `id="public-home-stream-results"`) {
		t.Fatalf("partial body missing results root: %s", recPartial.Body.String())
	}
}
