package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestBuildPublicStreamRunCardsOnlyCompletedCappedWithDPP(t *testing.T) {
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
	// Newest-first input: builder must preserve ListRecent order among completed only.
	ordered := []Process{processes[2], processes[3], processes[0], processes[1]}
	cards := buildPublicStreamRunCards(def, ordered)
	if len(cards) != 2 {
		t.Fatalf("len = %d, want 2 completed", len(cards))
	}
	if cards[0].DigitalLink != digitalLinkURL("09506000134352", "LOT-1", "SER-1") {
		t.Fatalf("first DigitalLink = %q", cards[0].DigitalLink)
	}
	if !cards[0].PassportChip || cards[0].StatusLabel != "Completed" {
		t.Fatalf("first card = %#v", cards[0])
	}
	if cards[1].DigitalLink != "" || cards[1].PassportChip {
		t.Fatalf("second card must be non-link, got %#v", cards[1])
	}
}

func TestBuildPublicStreamRunCardsRespectsCap(t *testing.T) {
	def := WorkflowDef{}
	doneAt := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	var processes []Process
	for i := 0; i < publicStreamRecentRunLimit+3; i++ {
		at := doneAt.Add(-time.Duration(i) * time.Hour)
		processes = append(processes, Process{
			ID:       primitive.NewObjectID(),
			Status:   processStatusDone,
			Progress: map[string]ProcessStep{"1_1": {State: "done", DoneAt: &at}},
		})
	}
	cards := buildPublicStreamRunCards(def, processes)
	if len(cards) != publicStreamRecentRunLimit {
		t.Fatalf("len = %d, want %d", len(cards), publicStreamRecentRunLimit)
	}
}

func TestPublicStreamRunCardTemplateLinkVsStatic(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var linked bytes.Buffer
	if err := tmpl.ExecuteTemplate(&linked, "public_stream_run_card", PublicStreamRunCardView{
		StatusLabel:  "Completed",
		CompletedAt:  "1 Jul 2026 at 12:00 UTC",
		DigitalLink:  "/01/09506000134352/10/LOT-1/21/SER-1",
		PassportChip: true,
	}); err != nil {
		t.Fatalf("linked render: %v", err)
	}
	lb := linked.String()
	if !strings.Contains(lb, `href="/01/09506000134352/10/LOT-1/21/SER-1"`) {
		t.Fatalf("expected DPP href, got: %s", lb)
	}
	if !strings.Contains(lb, "DPP") {
		t.Fatalf("expected DPP cue, got: %s", lb)
	}

	var static bytes.Buffer
	if err := tmpl.ExecuteTemplate(&static, "public_stream_run_card", PublicStreamRunCardView{
		StatusLabel: "Completed",
		CompletedAt: "1 Jul 2026 at 11:00 UTC",
	}); err != nil {
		t.Fatalf("static render: %v", err)
	}
	sb := static.String()
	if strings.Contains(sb, "href=") {
		t.Fatalf("static card must not link, got: %s", sb)
	}
	if !strings.Contains(sb, `<article class="public-stream-run-card"`) {
		t.Fatalf("expected article, got: %s", sb)
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
		Header: PublicStreamCardView{
			Name:        "Pilot Workflow",
			Description: "Gallium batches",
			StepCount:   3,
			RoleCount:   5,
		},
		RecentRuns: []PublicStreamRunCardView{{
			StatusLabel: "Completed",
			CompletedAt: "1 Jul 2026 at 12:00 UTC",
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
		`class="public-stream u-max-w-7xl`,
		"Pilot Workflow",
		"Gallium batches",
		"Recent completed runs",
		"Workflow",
		`class="public-stream-run-card"`,
		`class="stream-timeline-list"`,
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
	} {
		if strings.Contains(body, mustNot) {
			t.Fatalf("must not contain %q, got: %s", mustNot, body)
		}
	}
}

func TestPublicStreamBodyTemplateEmptyRuns(t *testing.T) {
	tmpl := parseTestTemplates(t)
	var out bytes.Buffer
	view := PublicStreamPageView{
		Header: PublicStreamCardView{Name: "Empty Runs"},
		Blueprint: StreamInstanceDetailView{HideStatus: true},
	}
	if err := tmpl.ExecuteTemplate(&out, "public_stream_body", view); err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(out.String(), "No completed runs yet.") {
		t.Fatalf("expected empty copy, got: %s", out.String())
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
	if err := os.WriteFile(filepath.Join(tempDir, "pilot.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	store := NewMemoryStore()
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
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q, got: %s", want, body)
		}
	}
	if strings.Contains(body, "/instance/start") || strings.Contains(body, "Sign in to run") {
		t.Fatalf("unexpected CTA/actions in body")
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
