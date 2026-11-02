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

func TestLivingSeedWipeEnabled(t *testing.T) {
	if livingSeedWipeEnabled("") || livingSeedWipeEnabled("0") || livingSeedWipeEnabled("false") {
		t.Fatal("expected disabled")
	}
	for _, v := range []string{"1", "true", "YES", "on"} {
		if !livingSeedWipeEnabled(v) {
			t.Fatalf("livingSeedWipeEnabled(%q) = false", v)
		}
	}
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
}

func TestEnsureLivingStoresSeedsFixtureAndOptionalWipe(t *testing.T) {
	t.Setenv("ATTESTA_LIVING_SEED", "true")
	domains := living.NewMemoryDomainStore()
	seeded := living.EmptyDomain()
	seeded.Entities["load"] = living.EntityMap{"L1": {"client": "NO"}}
	if err := domains.Put(t.Context(), "feragame", seeded); err != nil {
		t.Fatal(err)
	}
	server := &Server{livingDomains: domains}
	server.ensureLivingStores()
	got, err := domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Kind("load")) != 0 {
		t.Fatalf("expected wipe when ATTESTA_LIVING_SEED set: %#v", got)
	}
	version, err := server.livingConfigs.ActiveVersion(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	if version != living.DefaultFixtureVersion {
		t.Fatalf("active = %q", version)
	}
}

func TestEnsureLivingStoresSeedsFixtureWithoutWipeByDefault(t *testing.T) {
	t.Setenv("ATTESTA_LIVING_SEED", "")
	domains := living.NewMemoryDomainStore()
	seeded := living.EmptyDomain()
	seeded.Entities["load"] = living.EntityMap{"L1": {"client": "NO"}}
	if err := domains.Put(t.Context(), "feragame", seeded); err != nil {
		t.Fatal(err)
	}
	server := &Server{livingDomains: domains}
	server.ensureLivingStores()
	got, err := domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Kind("load")) != 1 {
		t.Fatalf("domain wiped without ATTESTA_LIVING_SEED: %#v", got)
	}
	version, err := server.livingConfigs.ActiveVersion(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	if version != living.DefaultFixtureVersion {
		t.Fatalf("active = %q", version)
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
	if len(jobs) != 2 {
		t.Fatalf("worklist jobs = %#v", wl)
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
	}

	body := `{"taskPath":"intake/register","jobInstanceId":"","mint":{"kind":"startingNew"},"value":{"client":"NO","date":"2026-10-19","wasteCode":"191212","idoneitaRiutilizzo":"NIR","truckWeight":100,"truckTare":200,"palletCount":1,"notes":""}}`
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
	if domain.Kind("load")["NIR-261019-NO-1"]["client"] != "NO" {
		t.Fatalf("stored domain = %#v", domain)
	}
	if len(domain.Submissions) != 1 || domain.Submissions[0].TaskID != "register" {
		t.Fatalf("submissions = %#v", domain.Submissions)
	}
	if len(domain.JobInstances) != 1 || domain.JobInstances[0].ID != "NIR-261019-NO-1" {
		t.Fatalf("jobInstances = %#v", domain.JobInstances)
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
	if committed.Links[1].Filename != "NIR-261019-NO-1.csv" {
		t.Fatalf("csv filename = %q", committed.Links[1].Filename)
	}
	if len(committed.HttpErrors) != 0 {
		t.Fatalf("httpErrors = %#v", committed.HttpErrors)
	}
	if len(committed.Worklist.Jobs) == 0 {
		t.Fatal("Commit JSON missing worklist jobs")
	}
	var commitIntakeDone bool
	for _, j := range committed.Worklist.Jobs {
		if j.JobID == "intake" && len(j.Instances) == 1 &&
			j.Instances[0].Tasks["register"] == living.TaskStatusDone {
			commitIntakeDone = true
		}
	}
	if !commitIntakeDone {
		t.Fatalf("Commit worklist intake register not done: %#v", committed.Worklist)
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
			if len(j.Instances) != 1 || j.Instances[0].Tasks["register"] != living.TaskStatusDone {
				t.Fatalf("intake worklist = %#v", j)
			}
		}
	}
	if !intakeCanStart {
		t.Fatal("intake worklist canStart")
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

func TestLivingProductHttpPortDefaultsToFake(t *testing.T) {
	server := &Server{}
	port := server.livingProductHttpPort()
	if _, ok := port.(*living.FakeHttpPort); !ok {
		t.Fatalf("product HttpPort type = %T, want *living.FakeHttpPort", port)
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
	body := `{"taskPath":"intake/register","jobInstanceId":"NIR-261019-NO-1","value":{"client":"NO","date":"2026-10-19","wasteCode":"191212","idoneitaRiutilizzo":"NIR","truckWeight":100,"truckTare":200,"palletCount":1}}`
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
	if err == nil && (len(domain.Submissions) != 0 || len(domain.Kind("load")) != 0) {
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
	mux := server.newMux()
	body := `{"taskPath":"intake/register","jobInstanceId":"NIR-261019-NO-1","value":{"client":"NO","date":"2026-10-19","wasteCode":"191212","idoneitaRiutilizzo":"NIR","truckWeight":100,"truckTare":200,"palletCount":1}}`
	req := httptest.NewRequest(http.MethodPost, "/api/living/commit", strings.NewReader(body))
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
}
