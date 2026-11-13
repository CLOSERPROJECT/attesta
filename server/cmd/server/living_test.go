package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CLOSERPROJECT/attesta/server/internal/living"
)

// seedLivingDemo puts Feragame Active Config for handler tests. Product boot
// (ensureLivingStores) no longer seeds — only task seed / tests call SeedDemo.
// SkipEmptyDomainStaging keeps EmptyDomain (Active Config without staged matrix).
func seedLivingDemo(t *testing.T, server *Server) {
	t.Helper()
	server.ensureLivingStores()
	if err := server.livingDomains.Put(t.Context(), "feragame", living.EmptyDomain()); err != nil {
		t.Fatal(err)
	}
	if _, err := living.SeedDemo(t.Context(), server.livingDomains, server.livingConfigs, living.SeedOptions{
		SkipEmptyDomainStaging: true,
	}); err != nil {
		t.Fatal(err)
	}
}

func livingCommit(t *testing.T, server *Server, sessionID, body string) living.CommitResult {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/living/commit", strings.NewReader(body))
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.livingHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("commit status = %d body=%s", rec.Code, rec.Body.String())
	}
	var result living.CommitResult
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func openWorkingDayThenArrive(t *testing.T, server *Server, sessionID string) string {
	t.Helper()
	livingCommit(t, server, sessionID,
		`{"taskPath":"workingDay/open","jobInstanceId":"","mint":{"kind":"startingNew"},"value":{"date":"2026-10-19","mode":"multi"}}`)
	arrived := livingCommit(t, server, sessionID,
		`{"taskPath":"intake/truckArrive","jobInstanceId":"","mint":{"kind":"startingNew"},"value":{"truckWeight":1000}}`)
	if arrived.JobInstanceID != "261019-1" {
		t.Fatalf("truck jobInstanceId = %q, want 261019-1", arrived.JobInstanceID)
	}
	return arrived.JobInstanceID
}

func TestEnsureLivingStoresUsesMemoryWithoutMongo(t *testing.T) {
	server := &Server{store: NewMemoryStore()}
	server.ensureLivingStores()
	if _, ok := server.livingDomains.(*living.MemoryDomainStore); !ok {
		t.Fatalf("domains = %T, want *living.MemoryDomainStore", server.livingDomains)
	}
	if _, ok := server.livingConfigs.(*living.MemoryConfigStore); !ok {
		t.Fatalf("configs = %T, want *living.MemoryConfigStore", server.livingConfigs)
	}
	if _, ok := server.livingLinks.(*living.MemoryLinkStore); !ok {
		t.Fatalf("links = %T, want *living.MemoryLinkStore", server.livingLinks)
	}
	if _, ok := server.livingPassports.(*living.MemoryPassportStore); !ok {
		t.Fatalf("passports = %T, want *living.MemoryPassportStore", server.livingPassports)
	}
}

func TestEnsureLivingStoresDoesNotSeed(t *testing.T) {
	domains := living.NewMemoryDomainStore()
	seeded := living.EmptyDomain()
	seeded.Entities["lot"] = living.EntityMap{"L1": {"client": "NO"}}
	if err := domains.Put(t.Context(), "feragame", seeded); err != nil {
		t.Fatal(err)
	}
	server := &Server{livingDomains: domains}
	server.ensureLivingStores()
	got, err := domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Kind("lot")) != 1 {
		t.Fatalf("domain mutated by ensureLivingStores: %#v", got)
	}
	_, err = server.livingConfigs.ActiveVersion(t.Context(), "feragame")
	if !errors.Is(err, living.ErrNotFound) {
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
	seedLivingDemo(t, server)

	req := httptest.NewRequest(http.MethodGet, "/api/living/domain", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.livingDomainHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var wire map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &wire); err != nil {
		t.Fatal(err)
	}
	if wire["org"] != "feragame" || wire["activeConfigVersion"] != living.DefaultFixtureVersion {
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
	if len(jobs) != 4 {
		t.Fatalf("worklist jobs = %#v", wl)
	}
	var passportJob map[string]any
	for _, raw := range jobs {
		job, _ := raw.(map[string]any)
		if job["jobId"] == "passport" {
			passportJob = job
			break
		}
	}
	if passportJob == nil || passportJob["startLabel"] != "Emetti DPP" {
		t.Fatalf("passport job = %#v", passportJob)
	}
}

func TestLivingDomainLoadUnauthenticated(t *testing.T) {
	server := &Server{
		identity:    testIdentityForSessions(time.Now().UTC(), map[string]AccountUser{}),
		store:       NewMemoryStore(),
		enforceAuth: true,
		now:         time.Now,
	}
	req := httptest.NewRequest(http.MethodGet, "/api/living/domain", nil)
	rec := httptest.NewRecorder()
	server.livingDomainHandler().ServeHTTP(rec, req)
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
	req := httptest.NewRequest(http.MethodGet, "/api/living/domain", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.livingDomainHandler().ServeHTTP(rec, req)
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
	seedLivingDemo(t, server)
	mux := server.newMux()
	req := httptest.NewRequest(http.MethodGet, "/api/living/domain", nil)
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
		store:            NewMemoryStore(),
		enforceAuth:      true,
		now:              func() time.Time { return now },
		livingAuthorizer: living.AllowAllAuthorizer{},
		// Test double: product no longer plants FakeHttpPort; exercise Commit+links seam.
		livingHttpPort: living.NewFakeHttpPort(),
	}
	seedLivingDemo(t, server)
	truckID := openWorkingDayThenArrive(t, server, sessionID)

	body := `{"taskPath":"intake/register","jobInstanceId":"` + truckID + `","value":{"client":"CO","wasteCode":"160214","numeroFir":"FIR-1","palletCount":2,"truckWeightAfter":700,"notes":""}}`
	req := httptest.NewRequest(http.MethodPost, "/api/living/commit", strings.NewReader(body))
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.livingHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("commit status = %d body=%s", rec.Code, rec.Body.String())
	}
	var committed living.CommitResult
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
	domain, err := server.livingDomains.Get(t.Context(), "feragame")
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
	if committed.Links[0].Label != "Stampa etichette" || committed.Links[0].Action != living.HttpActionLink {
		t.Fatalf("link0 = %#v", committed.Links[0])
	}
	if !strings.HasPrefix(committed.Links[0].URL, "data:application/pdf;base64,") {
		t.Fatalf("labels url = %q", committed.Links[0].URL)
	}
	if committed.Links[1].Label != "Scarica CSV registrazione" || committed.Links[1].Action != living.HttpActionDownload {
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
		if j.Instances[0].Tasks["register"] == living.TaskStatusActionable {
			registerOpen = true
		}
	}
	if !registerOpen {
		t.Fatalf("Commit worklist intake register not actionable: %#v", committed.Worklist)
	}

	loadReq := httptest.NewRequest(http.MethodGet, "/api/living/domain", nil)
	loadReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	loadRec := httptest.NewRecorder()
	server.livingHandler().ServeHTTP(loadRec, loadReq)
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
	var loaded living.LoadResult
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
			if len(j.Instances) != 1 || j.Instances[0].Tasks["truckArrive"] != living.TaskStatusDone ||
				j.Instances[0].Tasks["register"] != living.TaskStatusActionable {
				t.Fatalf("intake worklist = %#v", j)
			}
		}
	}
	if intakeCanStart {
		t.Fatal("intake canStart want false while prior truck incomplete")
	}
}

func TestLivingCommitUnauthenticated(t *testing.T) {
	server := &Server{
		identity:         testIdentityForSessions(time.Now().UTC(), map[string]AccountUser{}),
		store:            NewMemoryStore(),
		enforceAuth:      true,
		now:              time.Now,
		livingAuthorizer: living.AllowAllAuthorizer{},
	}
	req := httptest.NewRequest(http.MethodPost, "/api/living/commit", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	server.livingHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestLivingProductAuthorizerDefaultsToCerbos(t *testing.T) {
	server := &Server{now: time.Now}
	auth := server.livingProductAuthorizer()
	if _, ok := auth.(*living.CerbosAuthorizer); !ok {
		t.Fatalf("product Authorizer type = %T, want *living.CerbosAuthorizer", auth)
	}
	stub := living.DenyAllAuthorizer{}
	server.livingAuthorizer = stub
	if got := server.livingProductAuthorizer(); got != stub {
		t.Fatalf("override Authorizer = %#v", got)
	}
}

func TestLivingProductHttpPortFromEnvOrSoftFail(t *testing.T) {
	t.Setenv("LIVING_HTTP_URL", "")
	t.Setenv("LIVING_HTTP_TOKEN", "")
	t.Setenv("PUBLIC_BASE_URL", "https://plant.example")
	server := &Server{}
	port := server.livingProductHttpPort()
	if _, ok := port.(living.UnconfiguredHttpPort); !ok {
		t.Fatalf("product HttpPort type = %T, want living.UnconfiguredHttpPort", port)
	}
	if arts := server.livingProductArtifacts(); arts != nil {
		t.Fatalf("artifacts without URL = %#v", arts)
	}

	t.Setenv("LIVING_HTTP_URL", "http://living-http:8090")
	t.Setenv("LIVING_HTTP_TOKEN", "tok")
	port = server.livingProductHttpPort()
	sidecar, ok := port.(*living.SidecarHttpPort)
	if !ok || sidecar.BaseURL != "http://living-http:8090" || sidecar.Token != "tok" {
		t.Fatalf("env HttpPort = %#v", port)
	}
	if arts := server.livingProductArtifacts(); arts == nil || arts.BaseURL != "http://living-http:8090" {
		t.Fatalf("artifacts = %#v", arts)
	}

	stub := &living.FakeHttpPort{}
	server.livingHttpPort = stub
	if got := server.livingProductHttpPort(); got != stub {
		t.Fatalf("override HttpPort = %#v", got)
	}
	h := server.livingHandler()
	handler, ok := h.(*living.Handler)
	if !ok || handler.Committer == nil || handler.Committer.HttpPort == nil {
		t.Fatalf("livingHandler Committer.HttpPort = %#v", h)
	}
	if handler.Committer.HttpPort != stub {
		t.Fatalf("handler HttpPort = %T, want override", handler.Committer.HttpPort)
	}
	if handler.Committer.SubmitBindings["qrEndpoint"] != "https://plant.example/app" {
		t.Fatalf("SubmitBindings = %#v", handler.Committer.SubmitBindings)
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
		store:            NewMemoryStore(),
		enforceAuth:      true,
		now:              func() time.Time { return now },
		livingAuthorizer: living.DenyAllAuthorizer{},
	}
	seedLivingDemo(t, server)
	body := `{"taskPath":"intake/truckArrive","jobInstanceId":"","mint":{"kind":"startingNew"},"value":{"truckWeight":1000}}`
	req := httptest.NewRequest(http.MethodPost, "/api/living/commit", strings.NewReader(body))
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.livingHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}

	loadReq := httptest.NewRequest(http.MethodGet, "/api/living/domain", nil)
	loadReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	loadRec := httptest.NewRecorder()
	server.livingHandler().ServeHTTP(loadRec, loadReq)
	if loadRec.Code != http.StatusOK {
		t.Fatalf("load status = %d", loadRec.Code)
	}
	var loadWire map[string]any
	if err := json.Unmarshal(loadRec.Body.Bytes(), &loadWire); err != nil {
		t.Fatal(err)
	}
	if _, has := loadWire["domain"]; has {
		t.Fatalf("Load JSON must omit domain")
	}
	domain, err := server.livingDomains.Get(t.Context(), "feragame")
	if err != nil && !errors.Is(err, living.ErrNotFound) {
		t.Fatal(err)
	}
	if err == nil && (len(domain.Submissions) != 0 || len(domain.Kind("lot")) != 0) {
		t.Fatalf("domain should be unchanged after deny: %#v", domain)
	}
	var loaded living.LoadResult
	if err := json.Unmarshal(loadRec.Body.Bytes(), &loaded); err != nil {
		t.Fatal(err)
	}
	for _, j := range loaded.Worklist.Jobs {
		if len(j.Instances) != 0 {
			t.Fatalf("worklist should stay empty after deny: %#v", loaded.Worklist)
		}
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
		store:            NewMemoryStore(),
		enforceAuth:      true,
		now:              func() time.Time { return now },
		tmpl:             testTemplates(),
		livingAuthorizer: living.AllowAllAuthorizer{},
	}
	seedLivingDemo(t, server)
	mux := server.newMux()
	openBody := `{"taskPath":"workingDay/open","jobInstanceId":"","mint":{"kind":"startingNew"},"value":{"date":"2026-10-19","mode":"multi"}}`
	openReq := httptest.NewRequest(http.MethodPost, "/api/living/commit", strings.NewReader(openBody))
	openReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	openRec := httptest.NewRecorder()
	mux.ServeHTTP(openRec, openReq)
	if openRec.Code != http.StatusOK {
		t.Fatalf("open status = %d body=%s", openRec.Code, openRec.Body.String())
	}
	body := `{"taskPath":"intake/truckArrive","jobInstanceId":"","mint":{"kind":"startingNew"},"value":{"truckWeight":1000}}`
	req := httptest.NewRequest(http.MethodPost, "/api/living/commit", strings.NewReader(body))
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
		store:            NewMemoryStore(),
		enforceAuth:      true,
		now:              func() time.Time { return now },
		tmpl:             testTemplates(),
		livingAuthorizer: living.AllowAllAuthorizer{},
	}
	seedLivingDemo(t, server)
	mux := server.newMux()
	req := httptest.NewRequest(http.MethodPost, "/api/living/qr-resolve",
		strings.NewReader(`{"raw":"missing-pallet"}`))
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	// 422 from living.Handler proves mux registration; Go default would be 404.
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
}
