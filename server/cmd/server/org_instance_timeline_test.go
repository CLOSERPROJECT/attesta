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

func TestOrgInstanceTimelineRendersEachGroupSeedAndBlockedReason(t *testing.T) {
	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	server := &Server{
		authorizer:  fakeAuthorizer{},
		store:       NewMemoryStore(),
		tmpl:        parseTestTemplates(t),
		enforceAuth: true,
		now:         func() time.Time { return now },
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
	livingCommit(t, server, sessionID,
		`{"taskPath":"intake/register","jobInstanceId":"`+truckID+`","value":{"client":"CO","wasteCode":"160214","numeroFir":"FIR-CO","palletCount":1,"truckWeightAfter":900,"notes":""}}`)
	livingCommit(t, server, sessionID,
		`{"taskPath":"intake/register","jobInstanceId":"`+truckID+`","value":{"client":"DA","wasteCode":"160214","numeroFir":"FIR-DA","palletCount":1,"truckWeightAfter":800,"notes":""}}`)

	domain, err := server.orgDomains.Get(t.Context(), org.FeragameScope())
	if err != nil {
		t.Fatal(err)
	}
	var lotIDs []string
	for id, facts := range domain.Kind("lot") {
		if truck, _ := facts["truckId"].(string); truck == truckID {
			lotIDs = append(lotIDs, id)
		}
	}
	if len(lotIDs) != 2 {
		t.Fatalf("want 2 lots on truck, got %#v", lotIDs)
	}
	openLot, waitLot := lotIDs[0], lotIDs[1]
	livingCommit(t, server, sessionID,
		`{"taskPath":"lot_processing/startLot","jobInstanceId":"","mint":{"kind":"startingNew"},"value":{"lotId":"`+openLot+`"}}`)

	req := httptest.NewRequest(http.MethodGet, "/my/streams/org-recycling/instance/"+truckID, nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleMyRoutes(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()

	if !strings.Contains(body, "Lavorazione ×2") {
		t.Fatalf("expected each group title Lavorazione ×2, got:\n%s", body)
	}
	if !strings.Contains(body, "Ingresso") {
		t.Fatalf("expected once step title Ingresso, got:\n%s", body)
	}
	if !strings.Contains(body, "Lotto: "+waitLot) {
		t.Fatalf("expected waiting row label, got:\n%s", body)
	}

	seedHref := `/app/recycling/i/` + truckID + `/lot_processing/new/startLot?seed.lotId=` + waitLot
	if !strings.Contains(body, `href="`+seedHref+`"`) {
		t.Fatalf("expected seeded start link %q, got:\n%s", seedHref, body)
	}
	openTaskHref := `/app/recycling/i/` + truckID + `/lot_processing/` + openLot + `/`
	if !strings.Contains(body, openTaskHref) {
		t.Fatalf("expected open-lot task link containing %q, got:\n%s", openTaskHref, body)
	}

	wantBlocked := "Linea occupata: lotto " + openLot
	if !strings.Contains(body, wantBlocked) {
		t.Fatalf("expected blockedReason %q, got:\n%s", wantBlocked, body)
	}
	if !strings.Contains(body, `class="substep substep-locked"`) {
		t.Fatalf("expected locked substep class, got:\n%s", body)
	}
	if !strings.Contains(body, `class="substep substep-done"`) && !strings.Contains(body, `class="substep substep-available"`) {
		t.Fatalf("expected done/available substep states, got:\n%s", body)
	}

	if strings.Contains(body, "End stream") || strings.Contains(body, "terminate-process-dialog") {
		t.Fatalf("Org instance must hide terminate, got:\n%s", body)
	}
	if strings.Contains(body, `data-live-updates="1"`) || !strings.Contains(body, `data-live-updates="0"`) {
		t.Fatalf("Org instance must disable live updates for EventSource gate, got:\n%s", body)
	}
	if strings.Contains(body, "js-mine-filter") || strings.Contains(body, "My tasks") {
		t.Fatalf("Org instance must hide mine filter, got:\n%s", body)
	}
	if strings.Contains(body, "js-open-substep-override") || strings.Contains(body, "/override") {
		t.Fatalf("Org instance must hide Formata override affordances, got:\n%s", body)
	}
}

func TestOrgInstanceTimelineMutationsAndEventsNotFound(t *testing.T) {
	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	server := &Server{
		authorizer:  fakeAuthorizer{},
		store:       NewMemoryStore(),
		tmpl:        parseTestTemplates(t),
		enforceAuth: true,
		now:         func() time.Time { return now },
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
