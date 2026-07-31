package main

import (
	"context"
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestHandlePublicHomeRendersTaxonomySidebar(t *testing.T) {
	store := NewMemoryStore()
	seedPlatformAdminTaxonomy(t, store)
	server := &Server{store: store, configDir: t.TempDir(), tmpl: parseTestTemplates(t)}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	server.handlePublicHome(rec, req)
	body := rec.Body.String()
	for _, want := range []string{
		`class="category-sidebar"`,
		"Supply Chain",
		"Procurement",
		"Order Fulfillment",
		`hx-get="/streams/public?category=supply-chain&amp;subCategory=procurement"`,
		`hx-target="#public-home-stream-results"`,
		`hx-push-url="/?category=supply-chain&amp;subCategory=procurement"`,
		`id="public-home-stream-results"`,
		`class="public-home-results-category-header"`,
		`class="public-home-results-category-name">Supply Chain<`,
		`/static/taxonomy/batch-traceability.svg`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %s", want, body)
		}
	}
	for _, gone := range []string{
		`data-landing-tabs`,
		`public-home-tabs`,
		"See all streams",
		`icon-cat-compliance.svg`,
	} {
		if strings.Contains(body, gone) {
			t.Fatalf("did not expect %q", gone)
		}
	}
}

func TestHandlePublicHomeQuerySelectsSubcategory(t *testing.T) {
	store := NewMemoryStore()
	seedPlatformAdminTaxonomy(t, store)
	server := &Server{store: store, configDir: t.TempDir(), tmpl: parseTestTemplates(t)}
	req := httptest.NewRequest(http.MethodGet, "/?category=supply-chain&subCategory=order-fulfillment", nil)
	rec := httptest.NewRecorder()
	server.handlePublicHome(rec, req)
	body := rec.Body.String()
	if !strings.Contains(body, `subCategory=order-fulfillment`) || !strings.Contains(body, "is-active") {
		t.Fatalf("expected active subcategory markup, got %s", body)
	}
}

func TestHandlePublicHomeIsBlankAndPublic(t *testing.T) {
	server := &Server{
		store:     NewMemoryStore(),
		tmpl:      parseTestTemplates(t),
		configDir: t.TempDir(),
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	server.handlePublicHome(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "" {
		t.Fatalf("unexpected redirect to %q", loc)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `class="topbar`) {
		t.Fatalf("expected shared topbar on public home, got %q", body)
	}
	if !strings.Contains(body, `id="theme-toggle"`) {
		t.Fatalf("expected theme toggle on public home, got %q", body)
	}
	if !strings.Contains(body, `href="/login"`) {
		t.Fatalf("expected Login link on public home, got %q", body)
	}
	if !strings.Contains(body, `btn btn-ghost btn-lg nav-action`) {
		t.Fatalf("expected shared topbar Login control on public home, got %q", body)
	}
	if !strings.Contains(body, `d="M15 12H3"`) {
		t.Fatalf("expected login icon in topbar on public home, got %q", body)
	}
	if !strings.Contains(body, "Login") {
		t.Fatalf("expected Login label on public home, got %q", body)
	}
	if strings.Contains(body, `class="public-home-header"`) {
		t.Fatalf("expected no marketing header on public home, got %q", body)
	}
	if strings.Contains(body, `class="public-home-signin`) {
		t.Fatalf("expected no landing Sign In chrome on public home, got %q", body)
	}
	if !strings.Contains(body, `class="site-footer"`) {
		t.Fatalf("expected shared site-footer on public home, got %q", body)
	}
	if !strings.Contains(body, `class="public-home"`) {
		t.Fatalf("expected public landing markup, got %q", body)
	}
	if !strings.Contains(body, "Verified traceability for supply chains") {
		t.Fatalf("expected landing hero copy, got %q", body)
	}
	if strings.Contains(body, `account-menu`) {
		t.Fatalf("expected no account menu on public home, got %q", body)
	}
	if strings.Contains(body, `>Dashboard</a>`) {
		t.Fatalf("expected no Dashboard link when logged out, got %q", body)
	}
	if !strings.Contains(body, `class="site-footer-legal"`) {
		t.Fatalf("expected legal prose slot in site footer, got %q", body)
	}
	if !strings.Contains(body, "GNU AGPLv3") {
		t.Fatalf("expected AGPL legal copy in site footer, got %q", body)
	}
	if !strings.Contains(body, "Project No. 101161109") {
		t.Fatalf("expected CLOSER funding copy in site footer, got %q", body)
	}
	if !strings.Contains(body, "Project No. 101228240") {
		t.Fatalf("expected Even Closer funding copy in site footer, got %q", body)
	}
	if strings.Contains(body, `class="site-footer-heading"`) {
		t.Fatalf("expected parked footer nav columns not rendered, got %q", body)
	}
	if strings.Contains(body, `>Platform</p>`) {
		t.Fatalf("expected Platform footer heading not rendered, got %q", body)
	}
	if server.tmpl.Lookup("public_home_footer_nav") == nil {
		t.Fatal("expected parked template define public_home_footer_nav")
	}
}

func TestHandlePublicHomeEmptyCatalogRendersNoStreamCards(t *testing.T) {
	server := &Server{
		store:     NewMemoryStore(),
		tmpl:      parseTestTemplates(t),
		configDir: t.TempDir(),
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	server.handlePublicHome(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if strings.Contains(body, `class="public-stream-card"`) {
		t.Fatalf("expected no public stream cards for empty catalog, got %q", body)
	}
	for _, fake := range []string{
		"PV Module Tracing",
		"Indium recycling recovery",
		"Battery Passport",
		"Critical Raw Materials",
	} {
		if strings.Contains(body, fake) {
			t.Fatalf("expected no hard-coded stream tile %q on empty catalog, got %q", fake, body)
		}
	}
}

func TestHandlePublicHomeRendersCatalogStreamCards(t *testing.T) {
	tempDir := t.TempDir()
	writePublicHomeWorkflowConfig(t, filepath.Join(tempDir, "alpha.yaml"), "Alpha Stream", "string", "Alpha description")
	writePublicHomeWorkflowConfig(t, filepath.Join(tempDir, "beta.yaml"), "Beta Stream", "number", "Beta description")

	store := NewMemoryStore()
	seedPlatformAdminTaxonomy(t, store)
	server := &Server{
		store:     store,
		tmpl:      parseTestTemplates(t),
		configDir: tempDir,
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	server.handlePublicHome(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if got := strings.Count(body, `class="public-stream-card"`); got != 2 {
		t.Fatalf("public stream card count = %d, want 2", got)
	}
	for _, want := range []string{
		"Alpha Stream",
		"Alpha description",
		"Beta Stream",
		"Beta description",
		`href="/streams/alpha"`,
		`href="/streams/beta"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in public home body, got %q", want, body)
		}
	}
	if strings.Contains(body, `public-stream-card-badge`) {
		t.Fatalf("public home cards must not render legacy badges, got %q", body)
	}
	alphaIdx := strings.Index(body, "Alpha Stream")
	betaIdx := strings.Index(body, "Beta Stream")
	if alphaIdx < 0 || betaIdx < 0 || alphaIdx > betaIdx {
		t.Fatalf("expected stable key order alpha before beta, alpha=%d beta=%d", alphaIdx, betaIdx)
	}
	if strings.Contains(body, "PV Module Tracing") {
		t.Fatalf("expected no hard-coded Figma stream tiles, got %q", body)
	}
}

func TestHandlePublicHomeRendersStreamStepPreviewFromCatalog(t *testing.T) {
	tempDir := t.TempDir()
	content := `workflow:
  name: "Catalog Trace Stream"
  description: "Catalog step preview fixture"
  categorySlug: "supply-chain"
  subCategorySlug: "procurement"
  steps:
    - id: "1"
      title: "Incoming intake"
      order: 1
      organization: "org1"
      substeps:
        - id: "1.1"
          title: "Record lot"
          order: 1
          roles: ["dep1"]
          inputKey: "lot"
          inputType: "formata"
          schema:
            type: object
        - id: "1.2"
          title: "Attach photo"
          order: 2
          roles: ["dep1"]
          inputKey: "photo"
          inputType: "formata"
          schema:
            type: object
    - id: "2"
      title: "Quality check"
      order: 2
      organization: "org1"
      substeps:
        - id: "2.1"
          title: "Approve"
          order: 1
          roles: ["dep1"]
          inputKey: "ok"
          inputType: "formata"
          schema:
            type: object
organizations:
  - slug: "org1"
    name: "Organization 1"
roles:
  - orgSlug: "org1"
    slug: "dep1"
    name: "Department 1"
users:
  - id: "u1"
    name: "User 1"
    departmentId: "dep1"
`
	if err := os.WriteFile(filepath.Join(tempDir, "trace.yaml"), []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	store := NewMemoryStore()
	seedPlatformAdminTaxonomy(t, store)
	server := &Server{
		store:     store,
		tmpl:      parseTestTemplates(t),
		configDir: tempDir,
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	server.handlePublicHome(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"Catalog Trace Stream",
		`class="public-stream-card-metrics-row"`,
		"2 steps",
		"1 role",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in public home body, got %q", want, body)
		}
	}
	for _, gone := range []string{
		"public-stream-card-steps",
		"Incoming intake",
		"Quality check",
	} {
		if strings.Contains(body, gone) {
			t.Fatalf("did not expect %q in public home body, got %q", gone, body)
		}
	}
}

func TestHandlePublicHomeRendersPassportBadgeOnlyWhenDPPEnabled(t *testing.T) {
	tempDir := t.TempDir()
	writePublicHomeWorkflowConfig(t, filepath.Join(tempDir, "alpha.yaml"), "Alpha Plain Stream", "string", "No passport")
	writePublicHomeWorkflowConfigWithDPP(t, filepath.Join(tempDir, "beta.yaml"), "  enabled: true\n  gtin: \"9506000134352\"\n")

	store := NewMemoryStore()
	seedPlatformAdminTaxonomy(t, store)
	server := &Server{
		store:     store,
		tmpl:      parseTestTemplates(t),
		configDir: tempDir,
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	server.handlePublicHome(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()

	alphaIdx := strings.Index(body, "Alpha Plain Stream")
	betaIdx := strings.Index(body, ">Workflow</h3>")
	if alphaIdx < 0 || betaIdx < 0 {
		t.Fatalf("expected both catalog streams in body, alpha=%d beta=%d body=%q", alphaIdx, betaIdx, body)
	}

	alphaCardEnd := betaIdx
	if alphaIdx > betaIdx {
		alphaCardEnd = len(body)
	}
	alphaCard := body[alphaIdx:alphaCardEnd]
	if strings.Contains(alphaCard, "public-stream-card-dpp") || strings.Contains(alphaCard, ">DPP<") {
		t.Fatalf("plain stream must not show DPP chip, got %q", alphaCard)
	}

	betaCard := body[betaIdx:]
	if !strings.Contains(betaCard, `class="public-stream-card-dpp"`) {
		t.Fatalf("DPP-enabled stream must show DPP chip, got %q", betaCard)
	}
	if !strings.Contains(betaCard, "DPP") {
		t.Fatalf("DPP-enabled stream must show DPP label, got %q", betaCard)
	}
	if strings.Contains(body, `public-stream-card-badge`) {
		t.Fatalf("public home cards must not render legacy badges, got %q", body)
	}
}

func TestHandlePublicHomeRendersOrgAvatarsAndMetricsFromCatalog(t *testing.T) {
	tempDir := t.TempDir()
	content := `workflow:
  name: "Org Avatar Stream"
  description: "Org footer and metrics fixture"
  categorySlug: "supply-chain"
  subCategorySlug: "procurement"
  steps:
    - id: "1"
      title: "Incoming intake"
      order: 1
      organization: "acme"
      substeps:
        - id: "1.1"
          title: "Record lot"
          order: 1
          roles: ["dep1"]
          inputKey: "lot"
          inputType: "formata"
          schema:
            type: object
organizations:
  - slug: "acme"
    name: "Acme Corp"
  - slug: "beta"
    name: "Beta Org"
roles:
  - orgSlug: "acme"
    slug: "dep1"
    name: "Department 1"
users:
  - id: "u1"
    name: "User 1"
    departmentId: "dep1"
`
	if err := os.WriteFile(filepath.Join(tempDir, "orgs.yaml"), []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	store := NewMemoryStore()
	seedPlatformAdminTaxonomy(t, store)
	server := &Server{
		store: store,
		tmpl:  parseTestTemplates(t),
		identity: &fakeIdentityStore{
			listOrganizationsFunc: func(ctx context.Context) ([]IdentityOrg, error) {
				return []IdentityOrg{
					{Slug: "acme", Name: "Acme Corp", LogoFileID: "logo-1"},
					{Slug: "beta", Name: "Beta Org"},
				}, nil
			},
		},
		configDir: tempDir,
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	server.handlePublicHome(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()

	for _, want := range []string{
		"Org Avatar Stream",
		`src="/organization/logo/acme"`,
		`alt="Acme Corp"`,
		`aria-label="Beta Org"`,
		">BE<",
		`<strong>Organizations</strong>`,
		`class="public-stream-card-metrics"`,
		`class="public-stream-card-orgs-count"`,
		"no runs yet",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in public home body, got %q", want, body)
		}
	}
	if strings.Contains(body, "Stream Instances") {
		t.Fatalf("footer must not say Stream Instances, got %q", body)
	}
	if strings.Contains(body, ">28<") {
		t.Fatalf("stub instance count must not appear, got %q", body)
	}
	if strings.Contains(body, "all completed") {
		t.Fatalf("empty catalog stream must not show all-completed chip, got %q", body)
	}
	if strings.Contains(body, `src="/organization/logo/beta"`) {
		t.Fatalf("beta without logo must use initials, not logo url, got %q", body)
	}
}

func TestHandlePublicHomeRendersNoInstancesYetWhenStreamHasNoProcesses(t *testing.T) {
	tempDir := t.TempDir()
	writePublicHomeWorkflowConfig(t, filepath.Join(tempDir, "empty.yaml"), "Empty Metrics Stream", "string", "No processes yet")

	store := NewMemoryStore()
	seedPlatformAdminTaxonomy(t, store)
	server := &Server{
		store:     store,
		tmpl:      parseTestTemplates(t),
		configDir: tempDir,
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	server.handlePublicHome(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()

	for _, want := range []string{
		"Empty Metrics Stream",
		"no runs yet",
		`M13 13.74a2 2 0 0 1-2 0L2.5 8.87`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in public home body, got %q", want, body)
		}
	}
	if strings.Contains(body, ">28<") {
		t.Fatalf("stub instance count must not appear, got %q", body)
	}
	if strings.Contains(body, "all completed") {
		t.Fatalf("empty stream must not show all-completed chip, got %q", body)
	}
}

func TestHandlePublicHomeRendersAllCompletedMetricsFromSettledInstances(t *testing.T) {
	tempDir := t.TempDir()
	writePublicHomeWorkflowConfig(t, filepath.Join(tempDir, "settled.yaml"), "Settled Metrics Stream", "string", "Done and terminated only")

	store := NewMemoryStore()
	seedPlatformAdminTaxonomy(t, store)
	now := time.Now().UTC()
	store.SeedProcess(Process{
		WorkflowKey: "settled",
		CreatedAt:   now.Add(-2 * time.Minute),
		Status:      processStatusDone,
	})
	store.SeedProcess(Process{
		WorkflowKey: "settled",
		CreatedAt:   now.Add(-time.Minute),
		Status:      processStatusTerminated,
	})

	server := &Server{
		store:     store,
		tmpl:      parseTestTemplates(t),
		configDir: tempDir,
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	server.handlePublicHome(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()

	for _, want := range []string{
		"Settled Metrics Stream",
		"2 runs",
		"all completed",
		`M13 13.74a2 2 0 0 1-2 0L2.5 8.87`,
		`d="m9 12 2 2 4-4"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in public home body, got %q", want, body)
		}
	}
	if strings.Contains(body, ">28<") {
		t.Fatalf("stub instance count must not appear, got %q", body)
	}
	if strings.Contains(body, "no runs yet") {
		t.Fatalf("settled stream must not show empty label, got %q", body)
	}
	if strings.Count(body, `class="public-stream-card-metrics-row"`) != 2 {
		t.Fatalf("settled metrics must render run and step/role rows, got %q", body)
	}
}

func TestHandlePublicHomeRendersActiveNowMetricsScopedPerStream(t *testing.T) {
	tempDir := t.TempDir()
	writePublicHomeWorkflowConfig(t, filepath.Join(tempDir, "live.yaml"), "Live Metrics Stream", "string", "Has active instances")
	writePublicHomeWorkflowConfig(t, filepath.Join(tempDir, "settled.yaml"), "Settled Metrics Stream", "string", "Done only")

	store := NewMemoryStore()
	seedPlatformAdminTaxonomy(t, store)
	now := time.Now().UTC()
	store.SeedProcess(Process{
		WorkflowKey: "live",
		CreatedAt:   now.Add(-3 * time.Minute),
		Status:      processStatusActive,
	})
	store.SeedProcess(Process{
		WorkflowKey: "live",
		CreatedAt:   now.Add(-2 * time.Minute),
		Status:      processStatusTerminated,
	})
	store.SeedProcess(Process{
		WorkflowKey: "live",
		CreatedAt:   now.Add(-time.Minute),
		Status:      processStatusActive,
	})
	store.SeedProcess(Process{
		WorkflowKey: "settled",
		CreatedAt:   now.Add(-4 * time.Minute),
		Status:      processStatusDone,
	})
	store.SeedProcess(Process{
		WorkflowKey: "settled",
		CreatedAt:   now.Add(-5 * time.Minute),
		Status:      processStatusDone,
	})

	server := &Server{
		store:     store,
		tmpl:      parseTestTemplates(t),
		configDir: tempDir,
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	server.handlePublicHome(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()

	liveIdx := strings.Index(body, "Live Metrics Stream")
	settledIdx := strings.Index(body, "Settled Metrics Stream")
	if liveIdx < 0 || settledIdx < 0 {
		t.Fatalf("expected both stream cards in body, got %q", body)
	}

	// Slice each card roughly by the next article boundary so assertions stay per-stream.
	liveEnd := settledIdx
	settledEnd := len(body)
	if liveIdx > settledIdx {
		liveEnd = len(body)
		settledEnd = liveIdx
	}
	liveCard := body[liveIdx:liveEnd]
	settledCard := body[settledIdx:settledEnd]

	for _, want := range []string{
		"3 runs",
		"2 active now",
		`M22 12h-2.48a2 2 0 0 0-1.93 1.46l-2.35 8.36`,
	} {
		if !strings.Contains(liveCard, want) {
			t.Fatalf("live stream card missing %q, got %q", want, liveCard)
		}
	}
	if strings.Contains(liveCard, "all completed") {
		t.Fatalf("live stream with active instances must not show all-completed, got %q", liveCard)
	}

	for _, want := range []string{
		"2 runs",
		"all completed",
	} {
		if !strings.Contains(settledCard, want) {
			t.Fatalf("settled stream card missing %q, got %q", want, settledCard)
		}
	}
	if strings.Contains(settledCard, "active now") {
		t.Fatalf("settled stream must not inherit live active-now metrics, got %q", settledCard)
	}
}

func TestHandlePublicHomeLimitsToFirstSixCatalogStreams(t *testing.T) {
	tempDir := t.TempDir()
	names := []string{
		"Catalog One",
		"Catalog Two",
		"Catalog Three",
		"Catalog Four",
		"Catalog Five",
		"Catalog Six",
		"Catalog Seven",
	}
	files := []string{"a.yaml", "b.yaml", "c.yaml", "d.yaml", "e.yaml", "f.yaml", "g.yaml"}
	for i, name := range names {
		writePublicHomeWorkflowConfig(t, filepath.Join(tempDir, files[i]), name, "string", name+" description")
	}

	store := NewMemoryStore()
	seedPlatformAdminTaxonomy(t, store)
	server := &Server{
		store:     store,
		tmpl:      parseTestTemplates(t),
		configDir: tempDir,
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	server.handlePublicHome(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if got := strings.Count(body, `class="public-stream-card"`); got != 6 {
		t.Fatalf("public stream card count = %d, want 6", got)
	}
	for _, want := range names[:6] {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q among first six cards, got %q", want, body)
		}
	}
	if strings.Contains(body, "Catalog Seven") {
		t.Fatalf("did not expect seventh catalog stream on public home, got %q", body)
	}
	prev := -1
	for _, want := range names[:6] {
		idx := strings.Index(body, want)
		if idx < 0 || idx < prev {
			t.Fatalf("expected stable order for %q (prev=%d idx=%d)", want, prev, idx)
		}
		prev = idx
	}
}

func TestHandlePublicHomeShowsDashboardWhenLoggedIn(t *testing.T) {
	now := time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC)
	server := &Server{
		store:     NewMemoryStore(),
		tmpl:      parseTestTemplates(t),
		configDir: t.TempDir(),
		identity: &fakeIdentityStore{
			getSessionFunc: func(ctx context.Context, sessionSecret string) (IdentitySession, error) {
				if sessionSecret != "session-public-home" {
					return IdentitySession{}, ErrIdentityUnauthorized
				}
				return fakeIdentitySession(sessionSecret, "user-1", now.Add(24*time.Hour)), nil
			},
			getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
				return IdentityUser{ID: "user-1", Email: "user@example.com", Status: "active"}, nil
			},
		},
		enforceAuth: true,
		now:         func() time.Time { return now },
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-public-home"})
	rec := httptest.NewRecorder()
	server.handlePublicHome(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `account-menu`) {
		t.Fatalf("expected account menu when logged in, got %q", body)
	}
	if !strings.Contains(body, `href="/my"`) || !strings.Contains(body, "Dashboard") {
		t.Fatalf("expected Dashboard when logged in, got %q", body)
	}
	if strings.Contains(body, `class="public-home-header"`) {
		t.Fatalf("expected no marketing header when logged in, got %q", body)
	}
	if strings.Contains(body, `class="public-home-signin`) {
		t.Fatalf("expected no landing Sign In chrome when logged in, got %q", body)
	}
	if !strings.Contains(body, `class="topbar`) {
		t.Fatalf("expected shared topbar when logged in, got %q", body)
	}
	if strings.Contains(body, `href="/login"`) {
		t.Fatalf("expected no topbar Login link when logged in, got %q", body)
	}
}

func TestHandleHomeRequiresAuthAtMy(t *testing.T) {
	server := &Server{
		store:       NewMemoryStore(),
		tmpl:        testTemplates(),
		enforceAuth: true,
	}
	req := httptest.NewRequest(http.MethodGet, "/my", nil)
	rec := httptest.NewRecorder()
	server.handleHome(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want redirect", rec.Code)
	}
	if got := rec.Header().Get("Location"); !strings.HasPrefix(got, "/login?next=") {
		t.Fatalf("location = %q", got)
	}
}

func writeMyHomeCatalogOtherOrgWorkflow(t *testing.T, path string) {
	t.Helper()
	content := "workflow:\n" +
		"  name: \"Other Org Stream\"\n" +
		"  categorySlug: \"" + publicHomeTestCategorySlug + "\"\n" +
		"  subCategorySlug: \"" + publicHomeTestSubCategorySlug + "\"\n" +
		"  steps:\n" +
		"    - id: \"1\"\n" +
		"      title: \"Step 1\"\n" +
		"      order: 1\n" +
		"      organization: \"org2\"\n" +
		"      substeps:\n" +
		"        - id: \"1.1\"\n" +
		"          title: \"Input\"\n" +
		"          order: 1\n" +
		"          roles: [\"dep2\"]\n" +
		"          inputKey: \"value\"\n" +
		"          inputType: \"formata\"\n" +
		"          schema:\n" +
		"            type: object\n" +
		"organizations:\n" +
		"  - slug: \"org2\"\n" +
		"    name: \"Organization 2\"\n" +
		"roles:\n" +
		"  - orgSlug: \"org2\"\n" +
		"    slug: \"dep2\"\n" +
		"    name: \"Department 2\"\n" +
		"users:\n" +
		"  - id: \"u2\"\n" +
		"    name: \"User 2\"\n" +
		"    departmentId: \"dep2\"\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write other org workflow %s: %v", path, err)
	}
}

func TestHandleHomeCatalogWiring(t *testing.T) {
	store := NewMemoryStore()
	seedPlatformAdminTaxonomy(t, store)

	tempDir := t.TempDir()
	writePublicHomeWorkflowConfig(t, filepath.Join(tempDir, "accessible.yaml"), "Accessible Stream", "string", "Visible to org1")
	writeMyHomeCatalogOtherOrgWorkflow(t, filepath.Join(tempDir, "other-org.yaml"))

	now := time.Now().UTC()
	sessionID := "session-my-home-catalog"
	user := AccountUser{
		ID:        primitive.NewObjectID(),
		Email:     "member@example.com",
		OrgSlug:   "org1",
		RoleSlugs: []string{"dep1"},
		Status:    "active",
		CreatedAt: now,
	}

	server := &Server{
		authorizer:  fakeAuthorizer{},
		store:       store,
		identity:    testIdentityForSessions(now, map[string]AccountUser{sessionID: user}),
		tmpl:        parseTestTemplates(t),
		configDir:   tempDir,
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodGet, "/my", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleHome(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`nav-drawer-trigger`,
		`id="my-home-category-sidebar"`,
		`class="category-sidebar"`,
		`class="my-home-catalog"`,
		"Accessible Stream",
		`href="/my/streams/accessible/"`,
		"Supply Chain",
		"Procurement",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in home catalog, got: %s", want, body)
		}
	}
	for _, gone := range []string{
		"Other Org Stream",
		`href="/my/streams/other-org/"`,
		`href="/streams/accessible"`,
	} {
		if strings.Contains(body, gone) {
			t.Fatalf("did not expect %q in home catalog, got: %s", gone, body)
		}
	}

	t.Run("org admin sees create stream in header", func(t *testing.T) {
		adminSession := "session-my-home-org-admin"
		admin := AccountUser{
			ID:        primitive.NewObjectID(),
			Email:     "admin@example.com",
			OrgSlug:   "org1",
			RoleSlugs: []string{"org-admin"},
			Status:    "active",
			CreatedAt: now,
		}
		adminServer := &Server{
			authorizer:  fakeAuthorizer{},
			store:       store,
			identity:    testIdentityForSessions(now, map[string]AccountUser{adminSession: admin}),
			tmpl:        parseTestTemplates(t),
			configDir:   tempDir,
			enforceAuth: true,
			now:         func() time.Time { return now },
		}
		adminReq := httptest.NewRequest(http.MethodGet, "/my", nil)
		adminReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: adminSession})
		adminRec := httptest.NewRecorder()
		adminServer.handleHome(adminRec, adminReq)
		if adminRec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", adminRec.Code, http.StatusOK)
		}
		adminBody := adminRec.Body.String()
		for _, want := range []string{
			`class="page-header-actions"`,
			`href="/my/organization/formata-builder"`,
			"Create a stream",
		} {
			if !strings.Contains(adminBody, want) {
				t.Fatalf("expected %q in org admin home, got: %s", want, adminBody)
			}
		}
	})
}

func TestHandleHomeListsProcesses(t *testing.T) {
	store := NewMemoryStore()
	now := time.Date(2026, 2, 3, 12, 0, 0, 0, time.UTC)

	activeID := primitive.NewObjectID()
	active := Process{
		ID:          activeID,
		WorkflowKey: "workflow",
		Name:        "Pilot batch",
		CreatedAt:   now.Add(-2 * time.Hour),
		Status:      "",
		Progress: map[string]ProcessStep{
			"1_1": {State: "done", DoneAt: ptrTime(now.Add(-110 * time.Minute))},
			"1_2": {State: "done", DoneAt: ptrTime(now.Add(-100 * time.Minute)), Data: map[string]interface{}{"note": "alpha"}},
			"1_3": {State: "pending"},
			"2_1": {State: "pending"},
			"2_2": {State: "pending"},
			"3_1": {State: "pending"},
			"3_2": {State: "pending"},
		},
	}

	doneID := primitive.NewObjectID()
	done := Process{
		ID:          doneID,
		WorkflowKey: "workflow",
		CreatedAt:   now.Add(-1 * time.Hour),
		Status:      "active",
		Progress: map[string]ProcessStep{
			"1_1": {State: "done", DoneAt: ptrTime(now.Add(-70 * time.Minute))},
			"1_2": {State: "done", DoneAt: ptrTime(now.Add(-60 * time.Minute))},
			"1_3": {State: "done", DoneAt: ptrTime(now.Add(-50 * time.Minute))},
			"2_1": {State: "done", DoneAt: ptrTime(now.Add(-40 * time.Minute))},
			"2_2": {State: "done", DoneAt: ptrTime(now.Add(-30 * time.Minute))},
			"3_1": {State: "done", DoneAt: ptrTime(now.Add(-20 * time.Minute))},
			"3_2": {State: "done", DoneAt: ptrTime(now.Add(-10 * time.Minute))},
		},
	}

	store.SeedProcess(active)
	store.SeedProcess(done)

	server := &Server{
		authorizer: fakeAuthorizer{},
		store:      store,
		tmpl:       homeTestTemplates(),
		configProvider: func() (RuntimeConfig, error) {
			cfg := testRuntimeConfig()
			cfg.Workflow.Description = "Demo workflow description"
			return cfg, nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/my/streams/workflow/", nil)
	cfg := testRuntimeConfig()
	cfg.Workflow.Description = "Demo workflow description"
	req = req.WithContext(context.WithValue(req.Context(), workflowContextKey{}, workflowContextValue{
		Key: "workflow",
		Cfg: cfg,
	}))
	rec := httptest.NewRecorder()
	server.handleWorkflowHome(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "PROC 2 SORT time_desc FILTER all") {
		t.Fatalf("expected processes count and default controls, got %q", body)
	}
	if !strings.Contains(body, activeID.Hex()+":Pilot batch:available:28") {
		t.Fatalf("expected process name in process list item, got %q", body)
	}
	if !strings.Contains(body, doneID.Hex()+"::done:100") {
		t.Fatalf("expected done process stats, got %q", body)
	}
	if !strings.Contains(body, "SORT time_desc") {
		t.Fatalf("expected default sort, got %q", body)
	}
	if !strings.Contains(body, "DESC Demo workflow description") {
		t.Fatalf("expected workflow description, got %q", body)
	}
}

func TestHandleHomeFiltersProcessesByStatus(t *testing.T) {
	store := NewMemoryStore()
	now := time.Date(2026, 2, 3, 12, 0, 0, 0, time.UTC)

	activeID := primitive.NewObjectID()
	store.SeedProcess(Process{
		ID:          activeID,
		WorkflowKey: "workflow",
		CreatedAt:   now.Add(-2 * time.Hour),
		Status:      "active",
		Progress: map[string]ProcessStep{
			"1_1": {State: "done", DoneAt: ptrTime(now.Add(-110 * time.Minute))},
		},
	})

	doneID := primitive.NewObjectID()
	store.SeedProcess(Process{
		ID:          doneID,
		WorkflowKey: "workflow",
		CreatedAt:   now.Add(-1 * time.Hour),
		Status:      "done",
		Progress: map[string]ProcessStep{
			"1_1": {State: "done", DoneAt: ptrTime(now.Add(-70 * time.Minute))},
			"1_2": {State: "done", DoneAt: ptrTime(now.Add(-60 * time.Minute))},
			"1_3": {State: "done", DoneAt: ptrTime(now.Add(-50 * time.Minute))},
			"2_1": {State: "done", DoneAt: ptrTime(now.Add(-40 * time.Minute))},
			"2_2": {State: "done", DoneAt: ptrTime(now.Add(-30 * time.Minute))},
			"3_1": {State: "done", DoneAt: ptrTime(now.Add(-20 * time.Minute))},
			"3_2": {State: "done", DoneAt: ptrTime(now.Add(-10 * time.Minute))},
		},
	})

	server := &Server{
		authorizer: fakeAuthorizer{},
		store:      store,
		tmpl:       homeTestTemplates(),
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/my/streams/workflow/?filter=done", nil)
	req = req.WithContext(context.WithValue(req.Context(), workflowContextKey{}, workflowContextValue{
		Key: "workflow",
		Cfg: testRuntimeConfig(),
	}))
	rec := httptest.NewRecorder()
	server.handleWorkflowHome(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "PROC 1 SORT time_desc FILTER done") {
		t.Fatalf("expected done filter selection, got %q", body)
	}
	if !strings.Contains(body, doneID.Hex()+"::done:100") {
		t.Fatalf("expected done process in filtered list, got %q", body)
	}
	if strings.Contains(body, activeID.Hex()+"::active") {
		t.Fatalf("did not expect active process in done filter, got %q", body)
	}
}

func TestHandleHomePaginatesProcesses(t *testing.T) {
	store := NewMemoryStore()
	base := time.Date(2026, 2, 3, 12, 0, 0, 0, time.UTC)
	expectedPageTwoID := ""

	for i := 0; i < homeProcessesPerPage+1; i++ {
		processID := primitive.NewObjectID()
		if i == homeProcessesPerPage {
			expectedPageTwoID = processID.Hex()
		}
		store.SeedProcess(Process{
			ID:          processID,
			WorkflowKey: "workflow",
			CreatedAt:   base.Add(-time.Duration(i) * time.Minute),
			Status:      "active",
			Progress: map[string]ProcessStep{
				"1_1": {State: "done", DoneAt: ptrTime(base.Add(-time.Duration(i) * time.Minute))},
			},
		})
	}

	server := &Server{
		authorizer: fakeAuthorizer{},
		store:      store,
		tmpl:       homeTestTemplates(),
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/my/streams/workflow/?page=2", nil)
	req = req.WithContext(context.WithValue(req.Context(), workflowContextKey{}, workflowContextValue{
		Key: "workflow",
		Cfg: testRuntimeConfig(),
	}))
	rec := httptest.NewRecorder()
	server.handleWorkflowHome(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "PROC 1 SORT time_desc FILTER all PAGE 2/2") {
		t.Fatalf("expected second page with one process, got %q", body)
	}
	if !strings.Contains(body, expectedPageTwoID+"::available") {
		t.Fatalf("expected last process on page 2, got %q", body)
	}
}

func TestHandleHomeRendersWorkflowPicker(t *testing.T) {
	tempDir := t.TempDir()
	writeWorkflowConfig(t, tempDir+"/workflow.yaml", "Main workflow", "string", "Main workflow description")
	writeWorkflowConfig(t, tempDir+"/secondary.yaml", "Secondary workflow", "number")

	now := time.Now().UTC()
	sessionID := "session-home-picker"
	user := AccountUser{
		ID:        primitive.NewObjectID(),
		Email:     "picker@example.com",
		OrgSlug:   "org1",
		RoleSlugs: []string{"dep1"},
		Status:    "active",
		CreatedAt: now,
	}

	server := &Server{
		authorizer:  fakeAuthorizer{},
		store:       NewMemoryStore(),
		identity:    testIdentityForSessions(now, map[string]AccountUser{sessionID: user}),
		tmpl:        homePickerTemplates(),
		configDir:   tempDir,
		enforceAuth: true,
	}

	req := httptest.NewRequest(http.MethodGet, "/my", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleHome(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "PICK GROUPS 1") {
		t.Fatalf("expected uncategorized group marker, got %q", body)
	}
	if !strings.Contains(body, "workflow:Main workflow:Main workflow description:instances=") || !strings.Contains(body, "secondary:Secondary workflow:instances=") {
		t.Fatalf("expected accessible workflow cards in picker, got %q", body)
	}
	if strings.Contains(body, "secondary:Secondary workflow:Secondary workflow description|") {
		t.Fatalf("expected optional description to be omitted when empty, got %q", body)
	}
}

func TestNextAvailableAuthorizedActionFiltersByAvailableRoleAndOrganization(t *testing.T) {
	cfg := RuntimeConfig{
		Workflow: WorkflowDef{
			Name: "Org workflow",
			Steps: []WorkflowStep{
				{
					StepID:           "1",
					Title:            "Org 1 step",
					Order:            1,
					OrganizationSlug: "org1",
					Substep:          []WorkflowSub{{SubstepID: "1.1", Title: "Org 1 input", Order: 1, Roles: []string{"dep1"}, InputKey: "value", InputType: "formata"}},
				},
				{
					StepID:           "2",
					Title:            "Org 2 step",
					Order:            2,
					OrganizationSlug: "org2",
					Substep:          []WorkflowSub{{SubstepID: "2.1", Title: "Org 2 input", Order: 1, Roles: []string{"dep2"}, InputKey: "value", InputType: "formata"}},
				},
			},
		},
		Organizations: []WorkflowOrganization{
			{Slug: "org1", Name: "Organization 1"},
			{Slug: "org2", Name: "Organization 2"},
		},
		Roles: []WorkflowRole{
			{OrgSlug: "org1", Slug: "dep1", Name: "Department 1"},
			{OrgSlug: "org2", Slug: "dep2", Name: "Department 2"},
		},
	}
	now := time.Date(2026, 2, 4, 12, 0, 0, 0, time.UTC)
	matching := Process{
		ID:          primitive.NewObjectID(),
		WorkflowKey: "workflow",
		CreatedAt:   now,
		Status:      "active",
		Progress: map[string]ProcessStep{
			"1.1": {State: "pending"},
			"2.1": {State: "pending"},
		},
	}
	otherOrg := Process{
		ID:          primitive.NewObjectID(),
		WorkflowKey: "workflow",
		CreatedAt:   now.Add(-1 * time.Hour),
		Status:      "active",
		Progress: map[string]ProcessStep{
			"1.1": {State: "done", DoneAt: ptrTime(now.Add(-30 * time.Minute))},
			"2.1": {State: "pending"},
		},
	}
	done := Process{
		ID:          primitive.NewObjectID(),
		WorkflowKey: "workflow",
		CreatedAt:   now.Add(-2 * time.Hour),
		Status:      "done",
		Progress: map[string]ProcessStep{
			"1.1": {State: "done", DoneAt: ptrTime(now.Add(-90 * time.Minute))},
			"2.1": {State: "done", DoneAt: ptrTime(now.Add(-60 * time.Minute))},
		},
	}

	roleIndex := testRoleIndexForOrg("org1", map[string]RoleMeta{
		"dep1": {ID: "dep1", Label: "Department 1", Palette: "blue"},
	})
	roleIndex[roleMetaKey{OrgSlug: "org2", RoleSlug: "dep2"}] = RoleMeta{ID: "dep2", Label: "Department 2", Palette: "emerald"}

	action, ok := nextAuthorizedSubstepBody(cfg.Workflow, &matching, "workflow", Actor{
		OrgSlug:   "org1",
		RoleSlugs: []string{"dep1"},
	}, roleIndex, cfg.Roles)
	if !ok {
		t.Fatalf("expected available authorized action")
	}
	if action.SubstepID != "1.1" {
		t.Fatalf("substep id = %q, want 1.1", action.SubstepID)
	}
	if len(action.MatchingRoles) != 1 || action.MatchingRoles[0].Slug != "dep1" || action.MatchingRoles[0].Label != "Department 1" {
		t.Fatalf("matching roles = %#v, want dep1/Department 1", action.MatchingRoles)
	}

	if _, ok := nextAuthorizedSubstepBody(cfg.Workflow, &otherOrg, "workflow", Actor{
		OrgSlug:   "org1",
		RoleSlugs: []string{"dep1"},
	}, roleIndex, cfg.Roles); ok {
		t.Fatalf("did not expect authorized action for step in another organization")
	}

	if _, ok := nextAuthorizedSubstepBody(cfg.Workflow, &done, "workflow", Actor{
		OrgSlug:   "org1",
		RoleSlugs: []string{"dep1"},
	}, roleIndex, cfg.Roles); ok {
		t.Fatalf("did not expect authorized action for done process")
	}
}

func TestHandleHomePickerRendersWorkflowCardsAndScopedLinks(t *testing.T) {
	tempDir := t.TempDir()
	writeWorkflowConfig(t, filepath.Join(tempDir, "workflow.yaml"), "Main workflow", "string", "Main workflow description")
	writeWorkflowConfig(t, filepath.Join(tempDir, "secondary.yaml"), "Secondary workflow", "number")

	now := time.Now().UTC()
	sessionID := "session-home-scoped-links"
	user := AccountUser{
		ID:        primitive.NewObjectID(),
		Email:     "scoped-links@example.com",
		OrgSlug:   "org1",
		RoleSlugs: []string{"dep1"},
		Status:    "active",
		CreatedAt: now,
	}

	tmpl := parseTestTemplates(t)
	server := &Server{
		authorizer:  fakeAuthorizer{},
		store:       NewMemoryStore(),
		identity:    testIdentityForSessions(now, map[string]AccountUser{sessionID: user}),
		tmpl:        tmpl,
		configDir:   tempDir,
		enforceAuth: true,
	}

	req := httptest.NewRequest(http.MethodGet, "/my", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleHome(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `class="stack u-max-w-7xl u-mx-auto my-home"`) ||
		!strings.Contains(body, `class="page-header"`) ||
		!strings.Contains(body, "Choose a stream") {
		t.Fatalf("expected home picker wrapper structure, got %q", body)
	}
	if !strings.Contains(body, `class="my-home-catalog"`) ||
		!strings.Contains(body, `class="public-home-stream-grid"`) ||
		!strings.Contains(body, `class="public-stream-card"`) {
		t.Fatalf("expected public stream card grid markup, got %q", body)
	}
	if !strings.Contains(body, `href="/my/streams/workflow/"`) {
		t.Fatalf("expected scoped workflow href for workflow key, got %q", body)
	}
	if !strings.Contains(body, `href="/my/streams/secondary/"`) {
		t.Fatalf("expected scoped workflow href for secondary key, got %q", body)
	}
	if !strings.Contains(body, "Main workflow description") {
		t.Fatalf("expected description content in cards, got %q", body)
	}
	if !strings.Contains(body, "1 step") || !strings.Contains(body, "1 role") {
		t.Fatalf("expected public card metrics in cards, got %q", body)
	}
}

func TestHandleWorkflowHomeMarksProcessesWithMyTurn(t *testing.T) {
	store := NewMemoryStore()
	now := time.Date(2026, 2, 3, 12, 0, 0, 0, time.UTC)

	processID := primitive.NewObjectID()
	store.SeedProcess(Process{
		ID:          processID,
		WorkflowKey: "workflow",
		CreatedAt:   now.Add(-1 * time.Hour),
		Status:      "active",
		Progress: map[string]ProcessStep{
			"1_1": {State: "pending"},
		},
	})

	server := &Server{
		authorizer: fakeAuthorizer{},
		store:      store,
		tmpl:       homeTestTemplates(),
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/my/streams/workflow/", nil)
	req = req.WithContext(context.WithValue(req.Context(), workflowContextKey{}, workflowContextValue{
		Key: "workflow",
		Cfg: testRuntimeConfig(),
	}))
	rec := httptest.NewRecorder()
	server.handleWorkflowHome(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, processID.Hex()+"::available:0|") {
		t.Fatalf("expected process to surface available status, got %q", body)
	}
}

func TestHandleHomePickerCreateStreamCardVisibility(t *testing.T) {
	tempDir := t.TempDir()
	writeWorkflowConfig(t, filepath.Join(tempDir, "workflow.yaml"), "Main workflow", "string", "Main workflow description")

	tmpl := parseTestTemplates(t)

	t.Run("visible for org admin", func(t *testing.T) {
		store := NewMemoryStore()
		user := AccountUser{
			ID:        primitive.NewObjectID(),
			Email:     "org-admin-picker@example.com",
			OrgSlug:   "org1",
			RoleSlugs: []string{"org-admin"},
			Status:    "active",
			CreatedAt: time.Now().UTC(),
		}
		sessionID := "session-org-admin"

		server := &Server{
			authorizer:  fakeAuthorizer{},
			store:       store,
			identity:    testIdentityForSessions(time.Now().UTC(), map[string]AccountUser{sessionID: user}),
			tmpl:        tmpl,
			configDir:   tempDir,
			enforceAuth: true,
		}

		req := httptest.NewRequest(http.MethodGet, "/my", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()
		server.handleHome(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		body := rec.Body.String()
		for _, want := range []string{
			`class="page-header-actions"`,
			`href="/my/organization/formata-builder"`,
			"Create a stream",
		} {
			if !strings.Contains(body, want) {
				t.Fatalf("expected %q in org admin home header, got %q", want, body)
			}
		}
		if strings.Contains(body, "stream-card-cta") || strings.Contains(body, "Create new stream") {
			t.Fatalf("did not expect legacy create stream card on /my, got %q", body)
		}
	})

	t.Run("hidden for non org admin", func(t *testing.T) {
		store := NewMemoryStore()
		user := AccountUser{
			ID:        primitive.NewObjectID(),
			Email:     "member-picker@example.com",
			OrgSlug:   "org1",
			RoleSlugs: []string{"operator"},
			Status:    "active",
			CreatedAt: time.Now().UTC(),
		}
		sessionID := "session-member"

		server := &Server{
			authorizer:  fakeAuthorizer{},
			store:       store,
			identity:    testIdentityForSessions(time.Now().UTC(), map[string]AccountUser{sessionID: user}),
			tmpl:        tmpl,
			configDir:   tempDir,
			enforceAuth: true,
		}

		req := httptest.NewRequest(http.MethodGet, "/my", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()
		server.handleHome(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if strings.Contains(rec.Body.String(), `href="/my/organization/formata-builder"`) {
			t.Fatalf("did not expect create stream card for non org admin, got %q", rec.Body.String())
		}
	})

}

func TestHandleWorkflowHomeRendersValidationState(t *testing.T) {
	tmpl := parseTestTemplates(t)
	cfg := RuntimeConfig{
		Organizations: []WorkflowOrganization{
			{Slug: "org1", Name: "Organization 1"},
		},
		Roles: []WorkflowRole{
			{OrgSlug: "org1", Slug: "dep1", Name: "Department 1"},
		},
		Workflow: WorkflowDef{
			Name: "Workflow with missing refs",
			Steps: []WorkflowStep{
				{
					StepID:           "1",
					Title:            "Step 1",
					Order:            1,
					OrganizationSlug: "org1",
					Substep: []WorkflowSub{
						{SubstepID: "1.1", Title: "Sub 1", Order: 1, Roles: []string{"dep1"}, InputKey: "value", InputType: "formata"},
					},
				},
			},
		},
	}

	server := &Server{
		authorizer: fakeAuthorizer{},
		store:      NewMemoryStore(),
		tmpl:       tmpl,
		identity: &fakeIdentityStore{
			getSessionFunc: func(ctx context.Context, sessionSecret string) (IdentitySession, error) {
				return IdentitySession{Secret: sessionSecret, ExpiresAt: time.Now().UTC().Add(time.Hour)}, nil
			},
			getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
				return IdentityUser{ID: "user-1", Email: "user@example.com"}, nil
			},
			listOrganizationsFunc: func(ctx context.Context) ([]IdentityOrg, error) {
				return nil, nil
			},
		},
		enforceAuth: true,
	}

	req := httptest.NewRequest(http.MethodGet, "/my/streams/workflow/", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
	req = req.WithContext(context.WithValue(req.Context(), workflowContextKey{}, workflowContextValue{
		Key: "workflow",
		Cfg: cfg,
	}))
	rec := httptest.NewRecorder()
	server.handleWorkflowHome(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `class="error`) {
		t.Fatalf("expected error banner, got %q", body)
	}
	if !strings.Contains(body, "Stream configuration issue") {
		t.Fatalf("expected validation heading inside error, got %q", body)
	}
	if !strings.Contains(body, "workflow references are invalid") {
		t.Fatalf("expected validation error details, got %q", body)
	}
	if strings.Contains(body, `class="rail-layout`) || strings.Contains(body, `class="home-workflow-grid"`) {
		t.Fatalf("did not expect stream dashboard grid when config is invalid, got %q", body)
	}
	if strings.Contains(body, `action="/my/streams/workflow/instance/start"`) || strings.Contains(body, "New instance") {
		t.Fatalf("did not expect new instance controls when config is invalid, got %q", body)
	}
}

func TestHandleHomePickerDeleteButtonVisibility(t *testing.T) {
	tmpl := parseTestTemplates(t)
	now := time.Date(2026, 3, 7, 11, 0, 0, 0, time.UTC)

	t.Run("visible for creator before any process starts", func(t *testing.T) {
		store := NewMemoryStore()
		user := AccountUser{
			ID:             primitive.NewObjectID(),
			IdentityUserID: "creator-home-user",
			Email:          "creator-home@example.com",
			OrgSlug:        "org1",
			RoleSlugs:      []string{"org-admin"},
			Status:         "active",
		}
		sessionID := "session-home-creator"
		stream, err := store.SaveFormataBuilderStream(t.Context(), FormataBuilderStream{
			Stream:          workflowStreamYAML("Delete from home"),
			CreatedByUserID: user.IdentityUserID,
			UpdatedByUserID: user.IdentityUserID,
			UpdatedAt:       now,
		})
		if err != nil {
			t.Fatalf("SaveFormataBuilderStream: %v", err)
		}

		server := &Server{
			store:       store,
			identity:    testIdentityForSessions(now, map[string]AccountUser{sessionID: user}),
			authorizer:  fakeAuthorizer{deleteDecide: workflowDeleteDecision},
			tmpl:        tmpl,
			enforceAuth: true,
			now:         func() time.Time { return now },
		}

		req := httptest.NewRequest(http.MethodGet, "/my", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()
		server.handleHome(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		body := rec.Body.String()
		if !strings.Contains(body, `class="btn btn-ghost btn-icon public-stream-card-menu-trigger"`) {
			t.Fatalf("expected workflow actions menu trigger for creator, got %q", body)
		}
		if !strings.Contains(body, `href="/my/organization/formata-builder?stream=`+stream.ID.Hex()+`&new=true"`) {
			t.Fatalf("expected clone action for creator, got %q", body)
		}
		if !strings.Contains(body, `href="/my/organization/formata-builder?stream=`+stream.ID.Hex()+`"`) {
			t.Fatalf("expected edit action for creator, got %q", body)
		}
		if strings.Contains(body, `id="edit-workflow-`+stream.ID.Hex()+`"`) {
			t.Fatalf("did not expect edit warning dialog for creator, got %q", body)
		}
		if !strings.Contains(body, `id="delete-workflow-`+stream.ID.Hex()+`"`) {
			t.Fatalf("expected delete dialog for creator, got %q", body)
		}
		if !strings.Contains(body, `onclick="document.getElementById('delete-workflow-`+stream.ID.Hex()+`').showModal()"`) {
			t.Fatalf("expected delete dialog trigger for creator, got %q", body)
		}
		if !strings.Contains(body, `action="/my/streams/`+stream.ID.Hex()+`/delete"`) {
			t.Fatalf("expected delete form action for creator, got %q", body)
		}
		if !strings.Contains(rec.Body.String(), `href="/my/organization/formata-builder?stream=`+stream.ID.Hex()+`"`) {
			t.Fatalf("expected edit button for creator, got %q", rec.Body.String())
		}
	})

	t.Run("hidden for creator after process start", func(t *testing.T) {
		store := NewMemoryStore()
		user := AccountUser{
			ID:             primitive.NewObjectID(),
			IdentityUserID: "creator-home-started-user",
			Email:          "creator-home-started@example.com",
			OrgSlug:        "org1",
			RoleSlugs:      []string{"org-admin"},
			Status:         "active",
		}
		sessionID := "session-home-started"
		stream, err := store.SaveFormataBuilderStream(t.Context(), FormataBuilderStream{
			Stream:          workflowStreamYAML("Started stream"),
			CreatedByUserID: user.IdentityUserID,
			UpdatedByUserID: user.IdentityUserID,
			UpdatedAt:       now,
		})
		if err != nil {
			t.Fatalf("SaveFormataBuilderStream: %v", err)
		}
		store.SeedProcess(Process{
			ID:          primitive.NewObjectID(),
			WorkflowKey: stream.ID.Hex(),
			CreatedAt:   now,
			Status:      "active",
			Progress:    map[string]ProcessStep{},
		})

		server := &Server{
			store:       store,
			identity:    testIdentityForSessions(now, map[string]AccountUser{sessionID: user}),
			authorizer:  fakeAuthorizer{deleteDecide: workflowDeleteDecision},
			tmpl:        tmpl,
			enforceAuth: true,
			now:         func() time.Time { return now },
		}

		req := httptest.NewRequest(http.MethodGet, "/my", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()
		server.handleHome(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		body := rec.Body.String()
		if !strings.Contains(body, `class="btn btn-ghost btn-icon public-stream-card-menu-trigger"`) {
			t.Fatalf("expected workflow actions menu trigger for started stream, got %q", body)
		}
		if strings.Contains(body, `id="delete-workflow-`+stream.ID.Hex()+`"`) {
			t.Fatalf("did not expect delete dialog for started stream, got %q", body)
		}
		if strings.Contains(body, `action="/my/streams/`+stream.ID.Hex()+`/delete"`) {
			t.Fatalf("did not expect delete button for started stream, got %q", body)
		}
		if !strings.Contains(body, `href="/my/organization/formata-builder?stream=`+stream.ID.Hex()+`&new=true"`) {
			t.Fatalf("expected clone action for started stream, got %q", body)
		}
		if strings.Contains(body, `href="/my/organization/formata-builder?stream=`+stream.ID.Hex()+`"`) {
			t.Fatalf("did not expect direct edit action for started stream, got %q", body)
		}
		if !strings.Contains(body, `class="public-stream-card-menu-item public-stream-card-menu-item-disabled"`) {
			t.Fatalf("expected disabled edit action for started stream, got %q", body)
		}
		if !strings.Contains(body, `class="public-stream-card-menu-item public-stream-card-menu-item-danger public-stream-card-menu-item-disabled"`) {
			t.Fatalf("expected disabled delete action for started stream, got %q", body)
		}
	})

	t.Run("visible for platform admin even with processes", func(t *testing.T) {
		t.Setenv("ADMIN_EMAIL", "admin@example.com")
		t.Setenv("ADMIN_PASSWORD", "secret")

		store := NewMemoryStore()
		stream, err := store.SaveFormataBuilderStream(t.Context(), FormataBuilderStream{
			Stream:          workflowStreamYAML("Platform delete"),
			CreatedByUserID: "someone-else",
			UpdatedByUserID: "someone-else",
			UpdatedAt:       now,
		})
		if err != nil {
			t.Fatalf("SaveFormataBuilderStream: %v", err)
		}
		store.SeedProcess(Process{
			ID:          primitive.NewObjectID(),
			WorkflowKey: stream.ID.Hex(),
			CreatedAt:   now,
			Status:      "done",
			Progress: map[string]ProcessStep{
				"1_1": {State: "done"},
			},
		})

		server := &Server{
			store:       store,
			authorizer:  fakeAuthorizer{deleteDecide: workflowDeleteDecision},
			tmpl:        tmpl,
			enforceAuth: true,
			now:         func() time.Time { return now },
		}

		req := httptest.NewRequest(http.MethodGet, "/my", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
		rec := httptest.NewRecorder()
		server.handleHome(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		body := rec.Body.String()
		if !strings.Contains(body, `class="btn btn-ghost btn-icon public-stream-card-menu-trigger"`) {
			t.Fatalf("expected workflow actions menu trigger for platform admin, got %q", body)
		}
		if !strings.Contains(body, `href="/my/organization/formata-builder?stream=`+stream.ID.Hex()+`&new=true"`) {
			t.Fatalf("expected clone action for platform admin, got %q", body)
		}
		if !strings.Contains(body, `id="edit-workflow-`+stream.ID.Hex()+`"`) {
			t.Fatalf("expected edit warning dialog for platform admin, got %q", body)
		}
		if !strings.Contains(body, `onclick="document.getElementById('edit-workflow-`+stream.ID.Hex()+`').showModal()"`) {
			t.Fatalf("expected edit warning dialog trigger for platform admin, got %q", body)
		}
		if !strings.Contains(body, `href="/my/organization/formata-builder?stream=`+stream.ID.Hex()+`"`) {
			t.Fatalf("expected edit dialog continue action for platform admin, got %q", body)
		}
		if !strings.Contains(body, `id="delete-workflow-`+stream.ID.Hex()+`"`) {
			t.Fatalf("expected delete dialog for platform admin, got %q", body)
		}
		if !strings.Contains(body, `onclick="document.getElementById('delete-workflow-`+stream.ID.Hex()+`').showModal()"`) {
			t.Fatalf("expected delete dialog trigger for platform admin, got %q", body)
		}
		if !strings.Contains(body, `action="/my/streams/`+stream.ID.Hex()+`/delete"`) {
			t.Fatalf("expected delete form action for platform admin, got %q", body)
		}
	})

	t.Run("hidden for user without builder access", func(t *testing.T) {
		store := NewMemoryStore()
		user := AccountUser{
			ID:             primitive.NewObjectID(),
			IdentityUserID: "member-home-user",
			Email:          "member-home@example.com",
			OrgSlug:        "org1",
			RoleSlugs:      []string{"inspector"},
			Status:         "active",
		}
		sessionID := "session-home-member"
		stream, err := store.SaveFormataBuilderStream(t.Context(), FormataBuilderStream{
			Stream:          workflowStreamYAML("Member hidden"),
			CreatedByUserID: "creator-user",
			UpdatedByUserID: "creator-user",
			UpdatedAt:       now,
		})
		if err != nil {
			t.Fatalf("SaveFormataBuilderStream: %v", err)
		}

		server := &Server{
			store:       store,
			identity:    testIdentityForSessions(now, map[string]AccountUser{sessionID: user}),
			authorizer:  fakeAuthorizer{deleteDecide: workflowDeleteDecision},
			tmpl:        tmpl,
			enforceAuth: true,
			now:         func() time.Time { return now },
		}

		req := httptest.NewRequest(http.MethodGet, "/my", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()
		server.handleHome(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		body := rec.Body.String()
		if strings.Contains(body, `public-stream-card-menu-trigger`) {
			t.Fatalf("did not expect workflow actions menu trigger without builder access, got %q", body)
		}
		if strings.Contains(body, `href="/my/organization/formata-builder?stream=`+stream.ID.Hex()+`&new=true"`) {
			t.Fatalf("did not expect clone action without builder access, got %q", body)
		}
		if strings.Contains(body, `href="/my/organization/formata-builder?stream=`+stream.ID.Hex()+`"`) {
			t.Fatalf("did not expect edit action without builder access, got %q", body)
		}
		if strings.Contains(body, `public-stream-card-menu-item-danger public-stream-card-menu-item-disabled`) {
			t.Fatalf("did not expect delete action without builder access, got %q", body)
		}
	})
}

func TestHandleHomeRendersWorkflowPickerCountsByWorkflow(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_PASSWORD", "secret")

	tempDir := t.TempDir()
	writeTwoSubstepWorkflowConfig(t, tempDir+"/workflow.yaml", "Main workflow")
	writeTwoSubstepWorkflowConfig(t, tempDir+"/secondary.yaml", "Secondary workflow")

	now := time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)
	store := NewMemoryStore()
	store.SeedProcess(Process{
		ID:          primitive.NewObjectID(),
		WorkflowKey: "workflow",
		CreatedAt:   now.Add(-6 * time.Hour),
		Status:      "active",
		Progress: map[string]ProcessStep{
			"1_1": {State: "pending"},
			"1_2": {State: "pending"},
		},
	})
	store.SeedProcess(Process{
		ID:          primitive.NewObjectID(),
		WorkflowKey: "workflow",
		CreatedAt:   now.Add(-5 * time.Hour),
		Status:      "active",
		Progress: map[string]ProcessStep{
			"1_1": {State: "done", DoneAt: ptrTime(now.Add(-4 * time.Hour))},
			"1_2": {State: "pending"},
		},
	})
	store.SeedProcess(Process{
		ID:          primitive.NewObjectID(),
		WorkflowKey: "workflow",
		CreatedAt:   now.Add(-4 * time.Hour),
		Progress: map[string]ProcessStep{
			"1_1": {State: "done", DoneAt: ptrTime(now.Add(-3 * time.Hour))},
			"1_2": {State: "done", DoneAt: ptrTime(now.Add(-2 * time.Hour))},
		},
	})
	store.SeedProcess(Process{
		ID:        primitive.NewObjectID(),
		CreatedAt: now.Add(-3 * time.Hour),
		Status:    "active",
		Progress: map[string]ProcessStep{
			"1_1": {State: "pending"},
			"1_2": {State: "pending"},
		},
	})
	store.SeedProcess(Process{
		ID:          primitive.NewObjectID(),
		WorkflowKey: "secondary",
		CreatedAt:   now.Add(-2 * time.Hour),
		Status:      "active",
		Progress: map[string]ProcessStep{
			"1_1": {State: "done", DoneAt: ptrTime(now.Add(-90 * time.Minute))},
			"1_2": {State: "pending"},
		},
	})
	store.SeedProcess(Process{
		ID:          primitive.NewObjectID(),
		WorkflowKey: "secondary",
		CreatedAt:   now.Add(-1 * time.Hour),
		Progress: map[string]ProcessStep{
			"1_1": {State: "done", DoneAt: ptrTime(now.Add(-50 * time.Minute))},
			"1_2": {State: "done", DoneAt: ptrTime(now.Add(-40 * time.Minute))},
		},
	})

	server := &Server{
		authorizer:  fakeAuthorizer{},
		tmpl:        homePickerTemplates(),
		configDir:   tempDir,
		store:       store,
		enforceAuth: true,
	}

	req := httptest.NewRequest(http.MethodGet, "/my", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
	rec := httptest.NewRecorder()
	server.handleHome(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "workflow:Main workflow:instances=4:active=3|") {
		t.Fatalf("expected workflow instance metrics, got %q", body)
	}
	if !strings.Contains(body, "secondary:Secondary workflow:instances=2:active=1|") {
		t.Fatalf("expected secondary instance metrics, got %q", body)
	}
}

func TestNormalizeHomeSortKey(t *testing.T) {
	if got := normalizeHomeSortKey("status"); got != "status" {
		t.Fatalf("expected status, got %q", got)
	}
	if got := normalizeHomeSortKey("unknown"); got != "time_desc" {
		t.Fatalf("expected time_desc for unknown, got %q", got)
	}
}

func TestNormalizeHomeStatusFilter(t *testing.T) {
	if got := normalizeHomeStatusFilter("done"); got != "done" {
		t.Fatalf("expected done, got %q", got)
	}
	if got := normalizeHomeStatusFilter("ACTIVE"); got != "active" {
		t.Fatalf("expected active, got %q", got)
	}
	if got := normalizeHomeStatusFilter("all"); got != "all" {
		t.Fatalf("expected all, got %q", got)
	}
	if got := normalizeHomeStatusFilter("unknown"); got != "all" {
		t.Fatalf("expected all for unknown, got %q", got)
	}
}

func TestHomeProcessStatusCopy(t *testing.T) {
	navAria, navTitle, heading, empty, pagination := homeProcessStatusCopy("available")
	if navAria != "Available streams" || navTitle != "Streams waiting for your input" {
		t.Fatalf("unexpected available nav copy: %q / %q", navAria, navTitle)
	}
	if heading != "Available stream instances" || empty != "No available instances" {
		t.Fatalf("unexpected available section copy: %q / %q", heading, empty)
	}
	if pagination != "Available stream instances pagination" {
		t.Fatalf("unexpected available pagination label: %q", pagination)
	}

	_, _, heading, empty, _ = homeProcessStatusCopy("all")
	if heading != "All stream instances" || empty != "No instances" {
		t.Fatalf("unexpected all section copy: %q / %q", heading, empty)
	}
}

func TestHomePaginationURLUsesGlobalSortAndPage(t *testing.T) {
	got := homePaginationURL("/my/streams/workflow", "active", "status", 3)
	want := "/my/streams/workflow/?filter=active&page=3&sort=status"
	if got != want {
		t.Fatalf("pagination url = %q, want %q", got, want)
	}
	if strings.Contains(got, "#") {
		t.Fatalf("did not expect hash in pagination url, got %q", got)
	}
	if strings.Contains(got, "_sort=") || strings.Contains(got, "_page=") {
		t.Fatalf("did not expect per-status sort/page params, got %q", got)
	}

	allURL := homePaginationURL("/my/streams/workflow", "all", "time_desc", 1)
	if allURL != "/my/streams/workflow/" {
		t.Fatalf("defaults should omit query params, got %q", allURL)
	}
}

func TestBuildHomeProcessGroupsUsesGlobalSortAndFilterFields(t *testing.T) {
	groups := buildHomeProcessGroups("/my/streams/workflow", []StreamInstanceCard{
		{ID: "1", Status: "active"},
		{ID: "2", Status: "done"},
	}, "progress_desc", 1)

	var done *ProcessStatusGroup
	for i := range groups {
		if groups[i].Status == "done" {
			done = &groups[i]
			break
		}
	}
	if done == nil {
		t.Fatal("expected done process group")
	}
	if done.Sort != "progress_desc" {
		t.Fatalf("expected shared sort progress_desc, got %q", done.Sort)
	}
	foundFilter := false
	for _, field := range done.SortFields {
		if field.Name == "filter" && field.Value == "done" {
			foundFilter = true
		}
		if strings.HasSuffix(field.Name, "_sort") || strings.HasSuffix(field.Name, "_page") {
			t.Fatalf("did not expect legacy sort/page field %#v", field)
		}
	}
	if !foundFilter {
		t.Fatalf("expected filter=done in sort fields, got %#v", done.SortFields)
	}
	if !strings.Contains(done.NextURL, "filter=done") {
		t.Fatalf("expected filter in pagination url, got %q", done.NextURL)
	}
	if !strings.Contains(done.NextURL, "sort=progress_desc") {
		t.Fatalf("expected sort=progress_desc in pagination url, got %q", done.NextURL)
	}
	if done.Heading != "Done stream instances" || done.EmptyMessage != "No completed instances" {
		t.Fatalf("expected done status copy on group, got heading=%q empty=%q", done.Heading, done.EmptyMessage)
	}
	if done.NavAriaLabel != "Completed streams" || done.PaginationAriaLabel != "Done stream instances pagination" {
		t.Fatalf("expected done nav/pagination copy, got aria=%q pagination=%q", done.NavAriaLabel, done.PaginationAriaLabel)
	}
}

func TestNormalizeHomePage(t *testing.T) {
	if got := normalizeHomePage(0, 0); got != 1 {
		t.Fatalf("expected page 1 for empty result set, got %d", got)
	}
	if got := normalizeHomePage(99, 13); got != 2 {
		t.Fatalf("expected last page for overflow, got %d", got)
	}
	if got := normalizeHomePage(2, 24); got != 2 {
		t.Fatalf("expected page 2, got %d", got)
	}
}

func TestSortHomeProcessListByStatus(t *testing.T) {
	items := []StreamInstanceCard{
		{ID: "a", Status: "done", Percent: 100, CreatedAtTime: time.Date(2026, 2, 3, 10, 0, 0, 0, time.UTC)},
		{ID: "b", Status: "active", Percent: 10, CreatedAtTime: time.Date(2026, 2, 3, 11, 0, 0, 0, time.UTC)},
	}
	sortHomeProcessList(items, "status")
	if items[0].Status != "active" {
		t.Fatalf("expected active first, got %q", items[0].Status)
	}
}

func TestHandleHomeErrorPaths(t *testing.T) {
	t.Run("workflow options error", func(t *testing.T) {
		server := &Server{
			authorizer: fakeAuthorizer{},
			tmpl:       homePickerTemplates(),
			configDir:  t.TempDir(),
		}

		req := httptest.NewRequest(http.MethodGet, "/my", nil)
		rec := httptest.NewRecorder()
		server.handleHome(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
	})

	t.Run("template error", func(t *testing.T) {
		tempDir := t.TempDir()
		writeWorkflowConfig(t, filepath.Join(tempDir, "workflow.yaml"), "Main workflow", "string")

		server := &Server{
			authorizer: fakeAuthorizer{},
			tmpl:       template.Must(template.New("broken").Parse(`{{define "other"}}x{{end}}`)),
			configDir:  tempDir,
			configProvider: func() (RuntimeConfig, error) {
				return RuntimeConfig{}, errors.New("not used")
			},
		}

		req := httptest.NewRequest(http.MethodGet, "/my", nil)
		rec := httptest.NewRecorder()
		server.handleHome(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
	})
}

func TestHandleWorkflowHomeErrorPaths(t *testing.T) {
	t.Run("selected workflow error", func(t *testing.T) {
		server := &Server{
			authorizer: fakeAuthorizer{},
			tmpl:       testTemplates(),
			configProvider: func() (RuntimeConfig, error) {
				return RuntimeConfig{}, errors.New("config down")
			},
		}

		req := httptest.NewRequest(http.MethodGet, "/my/streams/workflow/", nil)
		rec := httptest.NewRecorder()
		server.handleWorkflowHome(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
	})

	t.Run("template error", func(t *testing.T) {
		server := &Server{
			authorizer: fakeAuthorizer{},
			store:      NewMemoryStore(),
			tmpl:       template.Must(template.New("broken").Parse(`{{define "other"}}x{{end}}`)),
			configProvider: func() (RuntimeConfig, error) {
				return testRuntimeConfig(), nil
			},
		}
		req := httptest.NewRequest(http.MethodGet, "/my/streams/workflow/", nil)
		req = req.WithContext(context.WithValue(req.Context(), workflowContextKey{}, workflowContextValue{
			Key: "workflow",
			Cfg: testRuntimeConfig(),
		}))

		rec := httptest.NewRecorder()
		server.handleWorkflowHome(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
	})
}

func TestHandleWorkflowHomeUsesHumanReadableProcessDates(t *testing.T) {
	tmpl := parseTestTemplates(t)
	store := NewMemoryStore()
	createdAt := time.Date(2026, 2, 3, 10, 0, 0, 0, time.UTC)
	doneAt := time.Date(2026, 2, 3, 11, 30, 0, 0, time.UTC)

	store.SeedProcess(Process{
		ID:          primitive.NewObjectID(),
		WorkflowKey: "workflow",
		CreatedAt:   createdAt,
		Status:      "active",
		Progress: map[string]ProcessStep{
			"1_1": {State: "done", DoneAt: ptrTime(doneAt), Data: map[string]interface{}{"note": "alpha"}},
		},
	})

	server := &Server{
		authorizer: fakeAuthorizer{},
		store:      store,
		tmpl:       tmpl,
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/my/streams/workflow/", nil)
	req = req.WithContext(context.WithValue(req.Context(), workflowContextKey{}, workflowContextValue{
		Key: "workflow",
		Cfg: testRuntimeConfig(),
	}))
	rec := httptest.NewRecorder()
	server.handleWorkflowHome(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Created:") || !strings.Contains(body, "3 Feb 2026 at 10:00 UTC") {
		t.Fatalf("expected human readable created date, got %q", body)
	}
	if !strings.Contains(body, `datetime="2026-02-03T10:00:00Z"`) {
		t.Fatalf("expected created datetime ISO, got %q", body)
	}
	if !strings.Contains(body, "Last notarized:") || !strings.Contains(body, "3 Feb 2026 at 11:30 UTC") {
		t.Fatalf("expected human readable last notarized date, got %q", body)
	}
	if !strings.Contains(body, `datetime="2026-02-03T11:30:00Z"`) {
		t.Fatalf("expected last notarized datetime ISO, got %q", body)
	}
}

func TestHandleWorkflowHomeHTMXReturnsPartial(t *testing.T) {
	store := NewMemoryStore()
	now := time.Date(2026, 2, 3, 12, 0, 0, 0, time.UTC)

	activeID := primitive.NewObjectID()
	active := Process{
		ID:          activeID,
		WorkflowKey: "workflow",
		Name:        "Pilot batch",
		CreatedAt:   now.Add(-2 * time.Hour),
		Status:      "",
		Progress: map[string]ProcessStep{
			"1_1": {State: "done", DoneAt: ptrTime(now.Add(-110 * time.Minute))},
			"1_2": {State: "done", DoneAt: ptrTime(now.Add(-100 * time.Minute)), Data: map[string]interface{}{"note": "alpha"}},
			"1_3": {State: "pending"},
			"2_1": {State: "pending"},
			"2_2": {State: "pending"},
			"3_1": {State: "pending"},
			"3_2": {State: "pending"},
		},
	}

	doneID := primitive.NewObjectID()
	done := Process{
		ID:          doneID,
		WorkflowKey: "workflow",
		CreatedAt:   now.Add(-1 * time.Hour),
		Status:      "active",
		Progress: map[string]ProcessStep{
			"1_1": {State: "done", DoneAt: ptrTime(now.Add(-70 * time.Minute))},
			"1_2": {State: "done", DoneAt: ptrTime(now.Add(-60 * time.Minute))},
			"1_3": {State: "done", DoneAt: ptrTime(now.Add(-50 * time.Minute))},
			"2_1": {State: "done", DoneAt: ptrTime(now.Add(-40 * time.Minute))},
			"2_2": {State: "done", DoneAt: ptrTime(now.Add(-30 * time.Minute))},
			"3_1": {State: "done", DoneAt: ptrTime(now.Add(-20 * time.Minute))},
			"3_2": {State: "done", DoneAt: ptrTime(now.Add(-10 * time.Minute))},
		},
	}

	store.SeedProcess(active)
	store.SeedProcess(done)

	server := &Server{
		authorizer: fakeAuthorizer{},
		store:      store,
		tmpl:       homeTestTemplates(),
		configProvider: func() (RuntimeConfig, error) {
			cfg := testRuntimeConfig()
			cfg.Workflow.Description = "Demo workflow description"
			return cfg, nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/my/streams/workflow/", nil)
	req.Header.Set("HX-Request", "true")
	req = req.WithContext(context.WithValue(req.Context(), workflowContextKey{}, workflowContextValue{
		Key: "workflow",
		Cfg: testRuntimeConfig(),
	}))
	rec := httptest.NewRecorder()
	server.handleWorkflowHome(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `id="stream-dashboard-results"`) {
		t.Fatalf("expected HTMX results root, got %q", body)
	}
	if strings.Contains(body, "data-home-root") {
		t.Fatalf("did not expect full page chrome in HTMX partial, got %q", body)
	}
	if !strings.Contains(body, "PROC 2 SORT time_desc FILTER all") {
		t.Fatalf("expected active process list in partial, got %q", body)
	}
	if !strings.Contains(body, activeID.Hex()+":Pilot batch:available:28") {
		t.Fatalf("expected process in HTMX partial, got %q", body)
	}
}

func TestHandleWorkflowHomeHTMXFiltersAndPaginates(t *testing.T) {
	store := NewMemoryStore()
	base := time.Date(2026, 2, 3, 12, 0, 0, 0, time.UTC)

	activeID := primitive.NewObjectID()
	store.SeedProcess(Process{
		ID:          activeID,
		WorkflowKey: "workflow",
		CreatedAt:   base.Add(-2 * time.Hour),
		Status:      "active",
		Progress: map[string]ProcessStep{
			"1_1": {State: "done", DoneAt: ptrTime(base.Add(-110 * time.Minute))},
		},
	})

	doneID := primitive.NewObjectID()
	store.SeedProcess(Process{
		ID:          doneID,
		WorkflowKey: "workflow",
		CreatedAt:   base.Add(-1 * time.Hour),
		Status:      "done",
		Progress: map[string]ProcessStep{
			"1_1": {State: "done", DoneAt: ptrTime(base.Add(-70 * time.Minute))},
			"1_2": {State: "done", DoneAt: ptrTime(base.Add(-60 * time.Minute))},
			"1_3": {State: "done", DoneAt: ptrTime(base.Add(-50 * time.Minute))},
			"2_1": {State: "done", DoneAt: ptrTime(base.Add(-40 * time.Minute))},
			"2_2": {State: "done", DoneAt: ptrTime(base.Add(-30 * time.Minute))},
			"3_1": {State: "done", DoneAt: ptrTime(base.Add(-20 * time.Minute))},
			"3_2": {State: "done", DoneAt: ptrTime(base.Add(-10 * time.Minute))},
		},
	})

	server := &Server{
		authorizer: fakeAuthorizer{},
		store:      store,
		tmpl:       homeTestTemplates(),
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}

	t.Run("filter", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/my/streams/workflow/?filter=done", nil)
		req.Header.Set("HX-Request", "true")
		req = req.WithContext(context.WithValue(req.Context(), workflowContextKey{}, workflowContextValue{
			Key: "workflow",
			Cfg: testRuntimeConfig(),
		}))
		rec := httptest.NewRecorder()
		server.handleWorkflowHome(rec, req)

		body := rec.Body.String()
		if !strings.Contains(body, "PROC 1 SORT time_desc FILTER done") {
			t.Fatalf("expected done filter in HTMX partial, got %q", body)
		}
		if !strings.Contains(body, doneID.Hex()+"::done:100") {
			t.Fatalf("expected done process in HTMX partial, got %q", body)
		}
		if strings.Contains(body, activeID.Hex()+"::active") {
			t.Fatalf("did not expect active process in done HTMX partial, got %q", body)
		}
	})

	t.Run("page", func(t *testing.T) {
		pageStore := NewMemoryStore()
		pageBase := time.Date(2026, 2, 3, 12, 0, 0, 0, time.UTC)
		expectedPageTwoID := ""
		for i := 0; i < homeProcessesPerPage+1; i++ {
			processID := primitive.NewObjectID()
			if i == homeProcessesPerPage {
				expectedPageTwoID = processID.Hex()
			}
			pageStore.SeedProcess(Process{
				ID:          processID,
				WorkflowKey: "workflow",
				CreatedAt:   pageBase.Add(-time.Duration(i) * time.Minute),
				Status:      "active",
				Progress: map[string]ProcessStep{
					"1_1": {State: "done", DoneAt: ptrTime(pageBase.Add(-time.Duration(i) * time.Minute))},
				},
			})
		}
		pageServer := &Server{
			authorizer: fakeAuthorizer{},
			store:      pageStore,
			tmpl:       homeTestTemplates(),
			configProvider: func() (RuntimeConfig, error) {
				return testRuntimeConfig(), nil
			},
		}

		req := httptest.NewRequest(http.MethodGet, "/my/streams/workflow/?page=2", nil)
		req.Header.Set("HX-Request", "true")
		req = req.WithContext(context.WithValue(req.Context(), workflowContextKey{}, workflowContextValue{
			Key: "workflow",
			Cfg: testRuntimeConfig(),
		}))
		rec := httptest.NewRecorder()
		pageServer.handleWorkflowHome(rec, req)

		body := rec.Body.String()
		if !strings.Contains(body, "PROC 1 SORT time_desc FILTER all PAGE 2/2") {
			t.Fatalf("expected paginated all filter in HTMX partial, got %q", body)
		}
		if !strings.Contains(body, expectedPageTwoID+"::available") {
			t.Fatalf("expected last process on page 2, got %q", body)
		}
	})
}

func TestBuildHomeFilterOptionsIncludesAllStatuses(t *testing.T) {
	options := buildHomeFilterOptions([]StreamInstanceCard{
		{ID: "1", Status: "active"},
		{ID: "2", Status: "done"},
		{ID: "3", Status: "available"},
	})
	if len(options) != len(homeProcessStatuses()) {
		t.Fatalf("expected %d filter options, got %d", len(homeProcessStatuses()), len(options))
	}
	for _, option := range options {
		if len(option.Processes) != 0 {
			t.Fatalf("expected empty processes on filter option %q, got %d", option.Status, len(option.Processes))
		}
		if option.PanelID == "" || option.NavAriaLabel == "" || option.NavTitle == "" {
			t.Fatalf("expected nav metadata on filter option %q, got %#v", option.Status, option)
		}
	}
	var allOption *ProcessStatusGroup
	for i := range options {
		if options[i].Status == "all" {
			allOption = &options[i]
			break
		}
	}
	if allOption == nil || allOption.TotalCount != 3 {
		t.Fatalf("expected all filter count 3, got %#v", allOption)
	}
}

func TestBuildHomeActiveProcessGroupBuildsSingleStatus(t *testing.T) {
	group := buildHomeActiveProcessGroup("/my/streams/workflow", []StreamInstanceCard{
		{ID: "1", Status: "active"},
		{ID: "2", Status: "done"},
	}, "done", "time_desc", 1)
	if group.Status != "done" {
		t.Fatalf("expected done group, got %q", group.Status)
	}
	if len(group.Processes) != 1 || group.Processes[0].ID != "2" {
		t.Fatalf("expected one done process, got %#v", group.Processes)
	}
}

func homeTestTemplates() *template.Template {
	return template.Must(template.New("test").Parse(`
{{define "layout.html"}}{{template "home_body" .}}{{end}}
{{define "home_body"}}
{{ $filter := .StatusFilter }}{{ range .ProcessGroups }}{{ if eq .Status $filter }}PROC {{len .Processes}} SORT {{.Sort}} FILTER {{$filter}} PAGE {{.CurrentPage}}/{{.TotalPages}} DESC {{$.WorkflowDescription}}
PROCESSES {{range .Processes}}{{.ID}}:{{.Name}}:{{.Status}}:{{.Percent}}|{{end}}{{ end }}{{ end }}
{{end}}
{{define "stream.html"}}{{template "layout.html" .}}{{end}}
{{define "stream_dashboard_results"}}
<div id="stream-dashboard-results">
{{ $filter := .StatusFilter }}{{ range .ProcessGroups }}{{ if eq .Status $filter }}PROC {{len .Processes}} SORT {{.Sort}} FILTER {{$filter}} PAGE {{.CurrentPage}}/{{.TotalPages}}
PROCESSES {{range .Processes}}{{.ID}}:{{.Name}}:{{.Status}}:{{.Percent}}|{{end}}{{ end }}{{ end }}
</div>
{{end}}
`))
}

func homePickerTemplates() *template.Template {
	return template.Must(template.New("test").Parse(`
{{define "layout.html"}}{{template "home_picker_body" .}}{{end}}
{{define "home_picker_body"}}PICK GROUPS {{len .Groups}}{{if .ShowCreateStream}} CREATE{{end}} {{range .Groups}}{{range .Streams}}{{.Key}}:{{.Card.Name}}{{if .Card.Description}}:{{.Card.Description}}{{end}}:instances={{.Card.InstanceCount}}:active={{.Card.ActiveCount}}|{{end}}{{end}}{{end}}
{{define "home.html"}}{{template "layout.html" .}}{{end}}
`))
}

func ptrTime(t time.Time) *time.Time {
	return &t
}

func workflowDeleteDecision(user *AccountUser, workflowKey string, createdByUserID string, hasProcesses bool) (bool, error) {
	if user == nil {
		return false, nil
	}
	if user.IsPlatformAdmin {
		return true, nil
	}
	return !hasProcesses && formataStreamUserID(user) == createdByUserID, nil
}
