package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CLOSERPROJECT/attesta/server/internal/org"
)

// seedOrgDemo puts Feragame Active Config for handler tests. Product boot
// (ensureOrgStores) no longer seeds — only task seed / tests call SeedDemo.
// SkipEmptyDomainStaging keeps EmptyDomain (Active Config without staged matrix).
func seedOrgDemo(t *testing.T, server *Server) {
	t.Helper()
	server.ensureOrgStores()
	if err := server.orgDomains.Put(t.Context(), org.FeragameScope(), org.EmptyDomain()); err != nil {
		t.Fatal(err)
	}
	if _, err := org.SeedDemo(t.Context(), server.orgDomains, server.orgConfigs, org.SeedOptions{
		SkipEmptyDomainStaging: true,
	}); err != nil {
		t.Fatal(err)
	}
}

func livingCommit(t *testing.T, server *Server, sessionID, query, body string) org.CommitResult {
	t.Helper()
	path := "/api/org/commit"
	if query != "" {
		path += "?" + query
	}
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.orgAPI().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("commit status = %d body=%s", rec.Code, rec.Body.String())
	}
	var result org.CommitResult
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func arriveIntakeTruck(t *testing.T, server *Server, sessionID string) string {
	t.Helper()
	arrived := livingCommit(t, server, sessionID, "stream=recycling",
		`{"taskPath":"intake/truckArrive","jobInstanceId":"","mint":{"kind":"startingNew"},"value":{"truckWeight":1000}}`)
	if arrived.JobInstanceID == "" {
		t.Fatal("truck jobInstanceId empty")
	}
	return arrived.JobInstanceID
}

func TestEnsureLivingStoresUsesMemoryWithoutMongo(t *testing.T) {
	server := &Server{store: NewMemoryStore()}
	server.ensureOrgStores()
	if _, ok := server.orgDomains.(*org.MemoryDomainStore); !ok {
		t.Fatalf("domains = %T, want *org.MemoryDomainStore", server.orgDomains)
	}
	if _, ok := server.orgConfigs.(*org.MemoryConfigStore); !ok {
		t.Fatalf("configs = %T, want *org.MemoryConfigStore", server.orgConfigs)
	}
	if _, ok := server.orgLinks.(*org.MemoryLinkStore); !ok {
		t.Fatalf("links = %T, want *org.MemoryLinkStore", server.orgLinks)
	}
	if _, ok := server.orgDPPs.(*org.MemoryDPPStore); !ok {
		t.Fatalf("passports = %T, want *org.MemoryDPPStore", server.orgDPPs)
	}
}

func TestEnsureLivingStoresDoesNotSeed(t *testing.T) {
	domains := org.NewMemoryDomainStore()
	seeded := org.EmptyDomain()
	seeded.Entities["lot"] = org.EntityMap{"L1": {"client": "NO"}}
	if err := domains.Put(t.Context(), org.FeragameScope(), seeded); err != nil {
		t.Fatal(err)
	}
	server := &Server{orgDomains: domains}
	server.ensureOrgStores()
	got, err := domains.Get(t.Context(), org.FeragameScope())
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Kind("lot")) != 1 {
		t.Fatalf("domain mutated by ensureOrgStores: %#v", got)
	}
	_, err = server.orgConfigs.ActiveVersion(t.Context(), org.FeragameScope())
	if !errors.Is(err, org.ErrNotFound) {
		t.Fatalf("expected no Active Config (SeedDemo is task seed only), got %v", err)
	}
}

func TestLivingDomainLoadVerifiedSession(t *testing.T) {
	now := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	sessionID := "living-load-ok"
	server := &Server{
		identity: testIdentityForSessions(now, map[string]AccountUser{
			sessionID: {
				IdentityUserID: "user-1",
				Email:          "op@example.com",
				OrgSlug:        "feragame",
				RoleSlugs:      []string{"administration"},
				Status:         "active",
				EmailVerified:  true,
			},
		}),
		store:       NewMemoryStore(),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}
	seedOrgDemo(t, server)
	server.orgAuthorizer = org.AllowAllAuthorizer{}
	truckID := arriveIntakeTruck(t, server, sessionID)

	req := httptest.NewRequest(http.MethodGet, "/api/org/domain?stream=recycling&instance="+truckID, nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.orgDomainHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var wire map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &wire); err != nil {
		t.Fatal(err)
	}
	if wire["org"] != "feragame" || wire["activeConfigVersion"] != org.DefaultFixtureVersion {
		t.Fatalf("got = %#v", wire)
	}
	if wire["configId"] != "feragame-config" {
		t.Fatalf("configId = %#v", wire["configId"])
	}
	if _, has := wire["domain"]; has {
		t.Fatalf("Load JSON must omit domain: %#v", wire["domain"])
	}
	if _, has := wire["config"]; has {
		t.Fatalf("Load JSON must omit config: %#v", wire["config"])
	}
	wl, _ := wire["worklist"].(map[string]any)
	jobs, _ := wl["jobs"].([]any)
	if len(jobs) != 2 {
		t.Fatalf("worklist jobs = %#v, want 2 (intake lot_processing)", wl)
	}
	wantIDs := []string{"intake", "lot_processing"}
	for i, raw := range jobs {
		job, _ := raw.(map[string]any)
		if job["jobId"] != wantIDs[i] {
			t.Fatalf("jobs[%d].jobId = %#v, want %q; worklist=%#v", i, job["jobId"], wantIDs[i], wl)
		}
	}
	intakeJob, _ := jobs[0].(map[string]any)
	instances, _ := intakeJob["instances"].([]any)
	if len(instances) != 1 {
		t.Fatalf("scoped intake instances = %#v", instances)
	}
	lpJob, _ := jobs[1].(map[string]any)
	taskIDs, _ := lpJob["taskIds"].([]any)
	var hasLock bool
	for _, id := range taskIDs {
		if id == "lock" {
			hasLock = true
			break
		}
	}
	if !hasLock {
		t.Fatalf("lot_processing taskIds = %#v, want lock", taskIDs)
	}
}

func TestLivingDomainLoadUnauthenticated(t *testing.T) {
	server := &Server{
		identity:    testIdentityForSessions(time.Now().UTC(), map[string]AccountUser{}),
		store:       NewMemoryStore(),
		enforceAuth: true,
		now:         time.Now,
	}
	req := httptest.NewRequest(http.MethodGet, "/api/org/domain", nil)
	rec := httptest.NewRecorder()
	server.orgDomainHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestLivingDomainLoadUnverified(t *testing.T) {
	now := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	sessionID := "living-load-unverified"
	server := &Server{
		identity: testIdentityForSessionsRespectingEmailVerified(now, map[string]AccountUser{
			sessionID: {
				IdentityUserID: "user-2",
				Email:          "waiting@example.com",
				OrgSlug:        "feragame",
				Status:         "active",
				EmailVerified:  false,
			},
		}),
		store:       NewMemoryStore(),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}
	req := httptest.NewRequest(http.MethodGet, "/api/org/domain", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.orgDomainHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

func TestLivingDomainMuxRegistration(t *testing.T) {
	now := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	sessionID := "living-mux"
	server := &Server{
		identity: testIdentityForSessions(now, map[string]AccountUser{
			sessionID: {
				IdentityUserID: "user-1",
				Email:          "op@example.com",
				OrgSlug:        "feragame",
				Status:         "active",
				EmailVerified:  true,
			},
		}),
		store:       NewMemoryStore(),
		enforceAuth: true,
		now:         func() time.Time { return now },
		tmpl:        testTemplates(),
	}
	seedOrgDemo(t, server)
	server.orgAuthorizer = org.AllowAllAuthorizer{}
	truckID := arriveIntakeTruck(t, server, sessionID)
	mux := server.newMux()
	req := httptest.NewRequest(http.MethodGet, "/api/org/domain?stream=recycling&instance="+truckID, nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestLivingCommitRegisterVerifiedSession(t *testing.T) {
	t.Setenv("PUBLIC_BASE_URL", "https://plant.example")
	now := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	sessionID := "living-commit-ok"
	server := &Server{
		identity: testIdentityForSessions(now, map[string]AccountUser{
			sessionID: {
				IdentityUserID: "user-1",
				Email:          "op@example.com",
				OrgSlug:        "feragame",
				RoleSlugs:      []string{"administration"},
				Status:         "active",
				EmailVerified:  true,
			},
		}),
		store:         NewMemoryStore(),
		enforceAuth:   true,
		now:           func() time.Time { return now },
		orgAuthorizer: org.AllowAllAuthorizer{},
		// Test double: product no longer plants FakeHttpPort; exercise Commit+links seam.
		orgHttpPort: org.NewFakeHttpPort(),
	}
	seedOrgDemo(t, server)
	truckID := arriveIntakeTruck(t, server, sessionID)

	body := `{"taskPath":"intake/register","jobInstanceId":"` + truckID + `","value":{"client":"CO","wasteCode":"160214","numeroFir":"FIR-1","palletCount":2,"truckWeightAfter":700,"notes":""}}`
	req := httptest.NewRequest(http.MethodPost, "/api/org/commit?stream=recycling&instance="+truckID, strings.NewReader(body))
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.orgAPI().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("commit status = %d body=%s", rec.Code, rec.Body.String())
	}
	var committed org.CommitResult
	if err := json.Unmarshal(rec.Body.Bytes(), &committed); err != nil {
		t.Fatal(err)
	}
	var commitWire map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &commitWire); err != nil {
		t.Fatal(err)
	}
	if _, has := commitWire["domain"]; has {
		t.Fatalf("Commit JSON must omit domain: %#v", commitWire["domain"])
	}
	domain, err := server.orgDomains.Get(t.Context(), org.FeragameScope())
	if err != nil {
		t.Fatal(err)
	}
	if domain.Kind("truck")[truckID]["lastScaleReading"] != 700.0 {
		t.Fatalf("stored truck = %#v", domain.Kind("truck"))
	}
	if len(domain.Kind("lot")) != 1 {
		t.Fatalf("loads = %#v", domain.Kind("lot"))
	}
	if len(domain.Submissions) < 2 || domain.Submissions[len(domain.Submissions)-1].TaskID != "register" {
		t.Fatalf("submissions = %#v", domain.Submissions)
	}
	if len(committed.Links) != 2 {
		t.Fatalf("wired HttpPort links = %#v", committed.Links)
	}
	if committed.Links[0].Label != "Stampa etichette" || committed.Links[0].Action != org.HttpActionLink {
		t.Fatalf("link0 = %#v", committed.Links[0])
	}
	if !strings.HasPrefix(committed.Links[0].URL, "data:application/pdf;base64,") {
		t.Fatalf("labels url = %q", committed.Links[0].URL)
	}
	if committed.Links[1].Label != "Scarica CSV registrazione" || committed.Links[1].Action != org.HttpActionDownload {
		t.Fatalf("link1 = %#v", committed.Links[1])
	}
	if committed.Links[1].Filename == "" {
		t.Fatalf("csv filename empty: %#v", committed.Links[1])
	}
	if len(committed.HttpErrors) != 0 {
		t.Fatalf("httpErrors = %#v", committed.HttpErrors)
	}
	if len(committed.Worklist.Jobs) == 0 {
		t.Fatal("Commit JSON missing worklist jobs")
	}
	var registerOpen bool
	for _, j := range committed.Worklist.Jobs {
		if j.JobID != "intake" || len(j.Instances) != 1 {
			continue
		}
		if j.Instances[0].Tasks["register"] == org.TaskStatusActionable {
			registerOpen = true
		}
	}
	if !registerOpen {
		t.Fatalf("Commit worklist intake register not actionable: %#v", committed.Worklist)
	}

	loadReq := httptest.NewRequest(http.MethodGet, "/api/org/domain?stream=recycling&instance="+truckID, nil)
	loadReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	loadRec := httptest.NewRecorder()
	server.orgAPI().ServeHTTP(loadRec, loadReq)
	if loadRec.Code != http.StatusOK {
		t.Fatalf("load status = %d body=%s", loadRec.Code, loadRec.Body.String())
	}
	var loadWire map[string]any
	if err := json.Unmarshal(loadRec.Body.Bytes(), &loadWire); err != nil {
		t.Fatal(err)
	}
	if _, has := loadWire["domain"]; has {
		t.Fatalf("Load JSON must omit domain after Commit")
	}
	if _, has := loadWire["config"]; has {
		t.Fatalf("Load JSON must omit config after Commit")
	}
	var loaded org.LoadResult
	if err := json.Unmarshal(loadRec.Body.Bytes(), &loaded); err != nil {
		t.Fatal(err)
	}
	if len(loaded.Worklist.Jobs) == 0 {
		t.Fatal("Load JSON missing worklist jobs")
	}
	var intakeCanStart bool
	for _, j := range loaded.Worklist.Jobs {
		if j.JobID == "intake" {
			intakeCanStart = j.CanStart
			if len(j.Instances) != 1 || j.Instances[0].Tasks["truckArrive"] != org.TaskStatusDone ||
				j.Instances[0].Tasks["register"] != org.TaskStatusActionable {
				t.Fatalf("intake worklist = %#v", j)
			}
		}
	}
	if !intakeCanStart {
		t.Fatal("intake canStart want true while prior truck incomplete (parallel trucks)")
	}
}

func TestLivingCommitUnauthenticated(t *testing.T) {
	server := &Server{
		identity:      testIdentityForSessions(time.Now().UTC(), map[string]AccountUser{}),
		store:         NewMemoryStore(),
		enforceAuth:   true,
		now:           time.Now,
		orgAuthorizer: org.AllowAllAuthorizer{},
	}
	req := httptest.NewRequest(http.MethodPost, "/api/org/commit", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	server.orgAPI().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestLivingProductAuthorizerDefaultsToCerbos(t *testing.T) {
	server := &Server{now: time.Now}
	auth := server.orgProductAuthorizer()
	if _, ok := auth.(*org.CerbosAuthorizer); !ok {
		t.Fatalf("product Authorizer type = %T, want *org.CerbosAuthorizer", auth)
	}
	stub := org.DenyAllAuthorizer{}
	server.orgAuthorizer = stub
	if got := server.orgProductAuthorizer(); got != stub {
		t.Fatalf("override Authorizer = %#v", got)
	}
}

func TestLivingProductHttpPortFromEnvOrSoftFail(t *testing.T) {
	t.Setenv("ORG_HTTP_URL", "")
	t.Setenv("ORG_HTTP_TOKEN", "")
	t.Setenv("PUBLIC_BASE_URL", "https://plant.example")
	server := &Server{}
	port := server.orgProductHttpPort()
	if _, ok := port.(org.UnconfiguredHttpPort); !ok {
		t.Fatalf("product HttpPort type = %T, want org.UnconfiguredHttpPort", port)
	}
	if arts := server.orgProductArtifacts(); arts != nil {
		t.Fatalf("artifacts without URL = %#v", arts)
	}

	t.Setenv("ORG_HTTP_URL", "http://org-http:8090")
	t.Setenv("ORG_HTTP_TOKEN", "tok")
	port = server.orgProductHttpPort()
	sidecar, ok := port.(*org.SidecarHttpPort)
	if !ok || sidecar.BaseURL != "http://org-http:8090" || sidecar.Token != "tok" {
		t.Fatalf("env HttpPort = %#v", port)
	}
	if arts := server.orgProductArtifacts(); arts == nil || arts.BaseURL != "http://org-http:8090" {
		t.Fatalf("artifacts = %#v", arts)
	}

	stub := &org.FakeHttpPort{}
	server.orgHttpPort = stub
	if got := server.orgProductHttpPort(); got != stub {
		t.Fatalf("override HttpPort = %#v", got)
	}
	h := server.orgAPI()
	handler, ok := h.(*org.API)
	if !ok || handler.TaskCommitter == nil || handler.TaskCommitter.HttpPort == nil {
		t.Fatalf("orgAPI TaskCommitter.HttpPort = %#v", h)
	}
	if handler.TaskCommitter.HttpPort != stub {
		t.Fatalf("handler HttpPort = %T, want override", handler.TaskCommitter.HttpPort)
	}
	if got := org.OrgAppBaseURL("recycling"); got != "https://plant.example/app/recycling" {
		t.Fatalf("OrgAppBaseURL(recycling) = %q", got)
	}
	if handler.Artifacts == nil {
		t.Fatal("handler Artifacts should still come from env when HttpPort is overridden")
	}
}

func TestLivingCommitAuthorizerDeny(t *testing.T) {
	now := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	sessionID := "living-commit-deny"
	server := &Server{
		identity: testIdentityForSessions(now, map[string]AccountUser{
			sessionID: {
				IdentityUserID: "user-1",
				Email:          "op@example.com",
				OrgSlug:        "feragame",
				RoleSlugs:      []string{"administration"},
				Status:         "active",
				EmailVerified:  true,
			},
		}),
		store:         NewMemoryStore(),
		enforceAuth:   true,
		now:           func() time.Time { return now },
		orgAuthorizer: org.DenyAllAuthorizer{},
	}
	seedOrgDemo(t, server)
	body := `{"taskPath":"intake/truckArrive","jobInstanceId":"","mint":{"kind":"startingNew"},"value":{"truckWeight":1000}}`
	req := httptest.NewRequest(http.MethodPost, "/api/org/commit?stream=recycling", strings.NewReader(body))
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.orgAPI().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}

	domain, err := server.orgDomains.Get(t.Context(), org.FeragameScope())
	if err != nil && !errors.Is(err, org.ErrNotFound) {
		t.Fatal(err)
	}
	if err == nil && (len(domain.Submissions) != 0 || len(domain.Kind("lot")) != 0 || len(domain.JobInstances) != 0) {
		t.Fatalf("domain should be unchanged after deny: %#v", domain)
	}
}

func TestLivingCommitMuxRegistration(t *testing.T) {
	now := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	sessionID := "living-commit-mux"
	server := &Server{
		identity: testIdentityForSessions(now, map[string]AccountUser{
			sessionID: {
				IdentityUserID: "user-1",
				Email:          "op@example.com",
				OrgSlug:        "feragame",
				RoleSlugs:      []string{"administration"},
				Status:         "active",
				EmailVerified:  true,
			},
		}),
		store:         NewMemoryStore(),
		enforceAuth:   true,
		now:           func() time.Time { return now },
		tmpl:          testTemplates(),
		orgAuthorizer: org.AllowAllAuthorizer{},
	}
	seedOrgDemo(t, server)
	mux := server.newMux()
	body := `{"taskPath":"intake/truckArrive","jobInstanceId":"","mint":{"kind":"startingNew"},"value":{"truckWeight":1000}}`
	req := httptest.NewRequest(http.MethodPost, "/api/org/commit?stream=recycling", strings.NewReader(body))
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestLivingQRResolveMuxRegistration(t *testing.T) {
	now := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	sessionID := "living-qr-resolve-mux"
	server := &Server{
		identity: testIdentityForSessions(now, map[string]AccountUser{
			sessionID: {
				IdentityUserID: "user-1",
				Email:          "op@example.com",
				OrgSlug:        "feragame",
				RoleSlugs:      []string{"administration"},
				Status:         "active",
				EmailVerified:  true,
			},
		}),
		store:         NewMemoryStore(),
		enforceAuth:   true,
		now:           func() time.Time { return now },
		tmpl:          testTemplates(),
		orgAuthorizer: org.AllowAllAuthorizer{},
	}
	seedOrgDemo(t, server)
	mux := server.newMux()
	req := httptest.NewRequest(http.MethodPost, "/api/org/qr-resolve?stream=recycling",
		strings.NewReader(`{"raw":"missing-pallet"}`))
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	// 422 from org.API proves mux registration; Go default would be 404.
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
}
