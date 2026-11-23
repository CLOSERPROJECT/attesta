package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CLOSERPROJECT/attesta/server/internal/org"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestOrgInstanceTimelineRedirectsToSPAHome(t *testing.T) {
	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	server := &Server{
		authorizer:    fakeAuthorizer{},
		orgAuthorizer: org.AllowAllAuthorizer{},
		store:         NewMemoryStore(),
		tmpl:          parseTestTemplates(t),
		enforceAuth:   true,
		now:           func() time.Time { return now },
	}
	seedOrgDemo(t, server)

	sessionID := "org-instance-timeline"
	server.identity = testIdentityForSessions(now, map[string]AccountUser{
		sessionID: {
			ID: primitive.NewObjectID(), Email: "ops@feragame.test", OrgSlug: "feragame",
			RoleSlugs: []string{"administration", "logistic", "line"}, Status: "active", CreatedAt: now,
		},
	})

	truckID := arriveIntakeTruck(t, server, sessionID)

	req := httptest.NewRequest(http.MethodGet, "/my/streams/org-recycling/instance/"+truckID, nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleMyRoutes(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	wantLoc := "/app/recycling/i/" + truckID
	if loc := rec.Header().Get("Location"); loc != wantLoc {
		t.Fatalf("Location = %q, want %q", loc, wantLoc)
	}
}

func TestOrgInstanceTimelineMutationsAndEventsNotFound(t *testing.T) {
	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	server := &Server{
		authorizer:    fakeAuthorizer{},
		orgAuthorizer: org.AllowAllAuthorizer{},
		store:         NewMemoryStore(),
		tmpl:          parseTestTemplates(t),
		enforceAuth:   true,
		now:           func() time.Time { return now },
	}
	seedOrgDemo(t, server)
	sessionID := "org-instance-mutations"
	server.identity = testIdentityForSessions(now, map[string]AccountUser{
		sessionID: {
			ID: primitive.NewObjectID(), Email: "ops@feragame.test", OrgSlug: "feragame",
			RoleSlugs: []string{"administration", "logistic"}, Status: "active", CreatedAt: now,
		},
	})
	truckID := arriveIntakeTruck(t, server, sessionID)

	for _, tc := range []struct {
		method, path string
	}{
		{http.MethodPost, "/my/streams/org-recycling/instance/" + truckID + "/terminate"},
		{http.MethodGet, "/my/streams/org-recycling/instance/" + truckID + "/substep/1.1/override"},
		{http.MethodPost, "/my/streams/org-recycling/instance/" + truckID + "/substep/1.1/complete"},
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
