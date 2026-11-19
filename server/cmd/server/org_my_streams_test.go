package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CLOSERPROJECT/attesta/server/internal/org"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestOrgCatalogKeyHelpers(t *testing.T) {
	if got := orgCatalogKey("recycling"); got != "org-recycling" {
		t.Fatalf("orgCatalogKey = %q", got)
	}
	key, ok := parseOrgCatalogKey("org-recycling")
	if !ok || key != "recycling" {
		t.Fatalf("parseOrgCatalogKey = %q %v", key, ok)
	}
	if _, ok := parseOrgCatalogKey("abcdef0123456789abcdef01"); ok {
		t.Fatal("hex Formata key must not parse as Org")
	}
}

func TestOrgDiscoveryVisibleOnlyToOwningOrg(t *testing.T) {
	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	store := NewMemoryStore()
	server := &Server{
		authorizer:  fakeAuthorizer{},
		store:       store,
		tmpl:        parseTestTemplates(t),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}
	seedOrgDemo(t, server)

	ownerSession := "org-discovery-owner"
	otherSession := "org-discovery-other"
	server.identity = testIdentityForSessions(now, map[string]AccountUser{
		ownerSession: {
			ID: primitive.NewObjectID(), Email: "owner@example.com", OrgSlug: "feragame",
			RoleSlugs: []string{"administration"}, Status: "active", CreatedAt: now,
		},
		otherSession: {
			ID: primitive.NewObjectID(), Email: "other@example.com", OrgSlug: "other-org",
			RoleSlugs: []string{"operator"}, Status: "active", CreatedAt: now,
		},
	})

	ownerReq := httptest.NewRequest(http.MethodGet, "/my", nil)
	ownerReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: ownerSession})
	ownerRec := httptest.NewRecorder()
	server.handleHome(ownerRec, ownerReq)
	if ownerRec.Code != http.StatusOK {
		t.Fatalf("owner status = %d", ownerRec.Code)
	}
	ownerBody := ownerRec.Body.String()
	wantHref := `href="/my/streams/org-recycling/"`
	if !strings.Contains(ownerBody, "Lavorazione rifiuti") || !strings.Contains(ownerBody, wantHref) {
		t.Fatalf("owner must see Org discovery card, got:\n%s", ownerBody)
	}
	if !strings.Contains(ownerBody, "Ingresso camion e lavorazione lotti") {
		t.Fatalf("owner card must use stream description, got:\n%s", ownerBody)
	}
	if strings.Contains(ownerBody, `id="delete-workflow-org-recycling"`) ||
		strings.Contains(ownerBody, "formata-builder?stream=org-recycling") {
		t.Fatalf("Org discovery must hide Formata delete/edit actions, got:\n%s", ownerBody)
	}

	otherReq := httptest.NewRequest(http.MethodGet, "/my", nil)
	otherReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: otherSession})
	otherRec := httptest.NewRecorder()
	server.handleHome(otherRec, otherReq)
	if otherRec.Code != http.StatusOK {
		t.Fatalf("other status = %d", otherRec.Code)
	}
	otherBody := otherRec.Body.String()
	if strings.Contains(otherBody, "Lavorazione rifiuti") || strings.Contains(otherBody, "org-recycling") {
		t.Fatalf("other org must not see feragame Org stream, got:\n%s", otherBody)
	}
}

func TestOrgDiscoverySingleStreamLinksToDashboard(t *testing.T) {
	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	server := &Server{
		authorizer:  fakeAuthorizer{},
		store:       NewMemoryStore(),
		tmpl:        parseTestTemplates(t),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}
	seedOrgDemo(t, server)
	sessionID := "org-single-stream"
	server.identity = testIdentityForSessions(now, map[string]AccountUser{
		sessionID: {
			ID: primitive.NewObjectID(), Email: "admin@feragame.test", OrgSlug: "feragame",
			RoleSlugs: []string{"administration"}, Status: "active", CreatedAt: now,
		},
	})

	items, err := server.listDiscoveryStreams(t.Context(), &AccountUser{
		OrgSlug: "feragame", RoleSlugs: []string{"administration"}, Status: "active",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("discovery len = %d, want 1 Org stream shortcut, got %#v", len(items), items)
	}
	if items[0].Href != "/my/streams/org-recycling/" {
		t.Fatalf("single-stream shortcut Href = %q", items[0].Href)
	}
	if items[0].WorkflowKey != "org-recycling" {
		t.Fatalf("WorkflowKey = %q", items[0].WorkflowKey)
	}
}

func TestOrgDashboardListsInstancesWithActiveDoneFilter(t *testing.T) {
	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	server := &Server{
		authorizer:  fakeAuthorizer{},
		store:       NewMemoryStore(),
		tmpl:        parseTestTemplates(t),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}
	server.ensureOrgStores()
	cfg, err := org.SeedDemo(t.Context(), server.orgDomains, server.orgConfigs, org.SeedOptions{
		WipeDomain: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Stream == nil || cfg.Stream.Key != "recycling" {
		t.Fatalf("seed stream = %#v", cfg.Stream)
	}

	sessionID := "org-dashboard"
	server.identity = testIdentityForSessions(now, map[string]AccountUser{
		sessionID: {
			ID: primitive.NewObjectID(), Email: "admin@feragame.test", OrgSlug: "feragame",
			RoleSlugs: []string{"administration", "logistic", "line"}, Status: "active", CreatedAt: now,
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/my/streams/org-recycling/", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleMyRoutes(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, org.SeedTruckID) {
		t.Fatalf("dashboard must list staged truck %s, got:\n%s", org.SeedTruckID, body)
	}
	if !strings.Contains(body, `name="filter"`) || !strings.Contains(body, `value="done"`) {
		t.Fatalf("dashboard must expose active/done filter, got:\n%s", body)
	}
	if strings.Contains(body, "View preview") || strings.Contains(body, "stream-preview-dialog") {
		t.Fatalf("Org dashboard must hide Formata preview, got:\n%s", body)
	}
	if strings.Contains(body, "New instance") || strings.Contains(body, "data-start-stream-name") {
		t.Fatalf("Org Play must not prompt for instance name, got:\n%s", body)
	}

	doneReq := httptest.NewRequest(http.MethodGet, "/my/streams/org-recycling/?filter=done", nil)
	doneReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	doneRec := httptest.NewRecorder()
	server.handleMyRoutes(doneRec, doneReq)
	if doneRec.Code != http.StatusOK {
		t.Fatalf("done filter status = %d", doneRec.Code)
	}
}

func TestOrgPlayRedirectsAndRoleGates(t *testing.T) {
	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	server := &Server{
		authorizer:  fakeAuthorizer{},
		store:       NewMemoryStore(),
		tmpl:        parseTestTemplates(t),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}
	seedOrgDemo(t, server)

	adminSession := "org-play-admin"
	lineSession := "org-play-line"
	server.identity = testIdentityForSessions(now, map[string]AccountUser{
		adminSession: {
			ID: primitive.NewObjectID(), Email: "admin@feragame.test", OrgSlug: "feragame",
			RoleSlugs: []string{"administration"}, Status: "active", CreatedAt: now,
		},
		lineSession: {
			ID: primitive.NewObjectID(), Email: "line@feragame.test", OrgSlug: "feragame",
			RoleSlugs: []string{"line"}, Status: "active", CreatedAt: now,
		},
	})

	adminHome := httptest.NewRequest(http.MethodGet, "/my", nil)
	adminHome.AddCookie(&http.Cookie{Name: "attesta_session", Value: adminSession})
	adminHomeRec := httptest.NewRecorder()
	server.handleHome(adminHomeRec, adminHome)
	adminHomeBody := adminHomeRec.Body.String()
	if !strings.Contains(adminHomeBody, `action="/my/streams/org-recycling/instance/start"`) {
		t.Fatalf("administration must see Play, got:\n%s", adminHomeBody)
	}
	if strings.Contains(adminHomeBody, "data-start-stream-name") {
		t.Fatalf("Org Play must be direct (no name field), got:\n%s", adminHomeBody)
	}

	lineHome := httptest.NewRequest(http.MethodGet, "/my", nil)
	lineHome.AddCookie(&http.Cookie{Name: "attesta_session", Value: lineSession})
	lineHomeRec := httptest.NewRecorder()
	server.handleHome(lineHomeRec, lineHome)
	if strings.Contains(lineHomeRec.Body.String(), `action="/my/streams/org-recycling/instance/start"`) {
		t.Fatalf("line role must not see Play for intake start Task")
	}

	startReq := httptest.NewRequest(http.MethodPost, "/my/streams/org-recycling/instance/start", strings.NewReader(""))
	startReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	startReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: adminSession})
	startRec := httptest.NewRecorder()
	server.handleMyRoutes(startRec, startReq)
	if startRec.Code != http.StatusSeeOther {
		t.Fatalf("Play status = %d body=%s", startRec.Code, startRec.Body.String())
	}
	if loc := startRec.Header().Get("Location"); loc != "/app/recycling/new" {
		t.Fatalf("Play Location = %q, want /app/recycling/new", loc)
	}

	denied := httptest.NewRequest(http.MethodPost, "/my/streams/org-recycling/instance/start", strings.NewReader(""))
	denied.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	denied.AddCookie(&http.Cookie{Name: "attesta_session", Value: lineSession})
	deniedRec := httptest.NewRecorder()
	server.handleMyRoutes(deniedRec, denied)
	if deniedRec.Code != http.StatusForbidden {
		t.Fatalf("line Play status = %d, want 403", deniedRec.Code)
	}
}

func TestOrgStreamForeignOrgGetsNotFound(t *testing.T) {
	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	server := &Server{
		authorizer:  fakeAuthorizer{},
		store:       NewMemoryStore(),
		tmpl:        parseTestTemplates(t),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}
	seedOrgDemo(t, server)
	sessionID := "org-foreign"
	server.identity = testIdentityForSessions(now, map[string]AccountUser{
		sessionID: {
			ID: primitive.NewObjectID(), Email: "x@other.test", OrgSlug: "other-org",
			RoleSlugs: []string{"operator"}, Status: "active", CreatedAt: now,
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/my/streams/org-recycling/", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleMyRoutes(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestOrgStreamHidesStreamOnlyActions(t *testing.T) {
	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	server := &Server{
		authorizer:  fakeAuthorizer{},
		store:       NewMemoryStore(),
		tmpl:        parseTestTemplates(t),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}
	seedOrgDemo(t, server)
	sessionID := "org-hidden-actions"
	server.identity = testIdentityForSessions(now, map[string]AccountUser{
		sessionID: {
			ID: primitive.NewObjectID(), Email: "admin@feragame.test", OrgSlug: "feragame",
			RoleSlugs: []string{"administration"}, Status: "active", CreatedAt: now,
		},
	})

	for _, tc := range []struct {
		method, path string
	}{
		{http.MethodPost, "/my/streams/org-recycling/delete"},
		{http.MethodGet, "/my/streams/org-recycling/events"},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()
		server.handleMyRoutes(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("%s %s status = %d, want 404", tc.method, tc.path, rec.Code)
		}
	}
}
