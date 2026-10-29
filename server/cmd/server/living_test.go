package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CLOSERPROJECT/attesta/server/internal/living"
)

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
	var got living.LoadResult
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Org != "feragame" || got.ActiveConfigVersion != living.DefaultFixtureVersion {
		t.Fatalf("got = %#v", got)
	}
	if len(got.Domain.Load) != 0 || len(got.Domain.Submissions) != 0 {
		t.Fatalf("domain should be empty: %#v", got.Domain)
	}
	if _, ok := got.Config.Jobs.Get("intake"); !ok {
		t.Fatalf("config missing intake job: keys=%v", got.Config.Jobs.Keys)
	}
	if _, ok := got.Config.Jobs.Get("line"); !ok {
		t.Fatalf("config missing line job: keys=%v", got.Config.Jobs.Keys)
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

	body := `{"taskPath":"intake/register","jobInstanceId":"NIR-261019-NO-1","value":{"client":"NO","date":"2026-10-19","wasteCode":"191212","idoneitaRiutilizzo":"NIR","truckWeight":100,"truckTare":200,"palletCount":1,"notes":""}}`
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
	if committed.Domain.Load["NIR-261019-NO-1"]["client"] != "NO" {
		t.Fatalf("commit result = %#v", committed)
	}
	if len(committed.Domain.Submissions) != 1 || committed.Domain.Submissions[0].TaskID != "register" {
		t.Fatalf("submissions = %#v", committed.Domain.Submissions)
	}

	loadReq := httptest.NewRequest(http.MethodGet, "/api/living/domain", nil)
	loadReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	loadRec := httptest.NewRecorder()
	server.livingHandler().ServeHTTP(loadRec, loadReq)
	if loadRec.Code != http.StatusOK {
		t.Fatalf("load status = %d body=%s", loadRec.Code, loadRec.Body.String())
	}
	var loaded living.LoadResult
	if err := json.Unmarshal(loadRec.Body.Bytes(), &loaded); err != nil {
		t.Fatal(err)
	}
	if loaded.Domain.Load["NIR-261019-NO-1"]["client"] != "NO" {
		t.Fatalf("load after commit = %#v", loaded.Domain)
	}
	if len(loaded.Domain.JobInstances) != 1 {
		t.Fatalf("jobInstances = %#v", loaded.Domain.JobInstances)
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
	var loaded living.LoadResult
	if err := json.Unmarshal(loadRec.Body.Bytes(), &loaded); err != nil {
		t.Fatal(err)
	}
	if len(loaded.Domain.Submissions) != 0 || len(loaded.Domain.Load) != 0 {
		t.Fatalf("domain should be unchanged after deny: %#v", loaded.Domain)
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
	body := `{"taskPath":"intake/register","jobInstanceId":"NIR-261019-NO-1","value":{"client":"NO","date":"2026-10-19","wasteCode":"191212","idoneitaRiutilizzo":"NIR","truckWeight":100,"truckTare":200,"palletCount":1}}`
	req := httptest.NewRequest(http.MethodPost, "/api/living/commit", strings.NewReader(body))
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
}
